package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// GET /api/logs/stream?service=kumomta
// Streams log output as Server-Sent Events. Falls back through:
//  1. /opt/kumomta/sbin/tailer --tail /var/log/kumomta
//  2. tail -f <latest log file in /var/log/kumomta/>
//  3. journalctl -fu <service>
func (s *Server) handleLiveLogStream(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	if service == "" {
		service = "kumomta"
	}
	// Allowlist
	allowed := map[string]bool{"kumomta": true, "dovecot": true, "fail2ban": true, "postfix": true}
	if !allowed[service] {
		http.Error(w, "unknown service", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	ctx := r.Context()

	// Decide command
	var cmd *exec.Cmd
	if service == "kumomta" {
		tailer := "/opt/kumomta/sbin/tailer"
		logDir := "/var/log/kumomta"

		if _, err := os.Stat(tailer); err == nil {
			cmd = exec.CommandContext(ctx, tailer, "--tail", logDir)
		} else if files := latestLogFiles(logDir); len(files) > 0 {
			args := append([]string{"-F"}, files...)
			cmd = exec.CommandContext(ctx, "tail", args...)
		}
	}

	// Fallback: journalctl
	if cmd == nil {
		cmd = exec.CommandContext(ctx, "journalctl", "-fu", service, "--no-pager", "--output=short-iso")
	}

	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "data: [error] could not start log stream: %v\n\n", err)
		flusher.Flush()
		return
	}

	// Send a heartbeat comment every 15s to keep the connection alive
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := pr.Read(buf)
			if n > 0 {
				line := strings.TrimRight(string(buf[:n]), "\n")
				for _, l := range strings.Split(line, "\n") {
					if l == "" {
						continue
					}
					fmt.Fprintf(w, "data: %s\n\n", l)
					flusher.Flush()
				}
			}
			if err != nil {
				return
			}
		}
	}()

	<-ctx.Done()
	pw.Close()
	cmd.Process.Kill() //nolint:errcheck
	<-done
}

func latestLogFiles(dir string) []string {
	files, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil || len(files) == 0 {
		return nil
	}
	type fi struct {
		path    string
		modTime int64
	}
	var infos []fi
	for _, f := range files {
		if info, err := os.Stat(f); err == nil && !info.IsDir() {
			infos = append(infos, fi{f, info.ModTime().Unix()})
		}
	}
	sort.Slice(infos, func(i, j int) bool { return infos[i].modTime > infos[j].modTime })
	// Return up to 3 most recent NON-compressed files
	result := make([]string, 0, 3)
	for _, f := range infos {
		if strings.HasSuffix(f.path, ".gz") || strings.HasSuffix(f.path, ".bz2") || strings.HasSuffix(f.path, ".xz") || strings.HasSuffix(f.path, ".zst") {
			continue // skip compressed rotated logs
		}
		result = append(result, f.path)
		if len(result) >= 3 {
			break
		}
	}
	return result
}

// safeCommands maps a short command key to the actual exec args.
// All commands are read-only and non-privileged.
var safeCommands = map[string][]string{
	"ps":          {"ps", "aux", "--sort=-%cpu"},
	"df":          {"df", "-h"},
	"free":        {"free", "-h"},
	"uptime":      {"uptime"},
	"who":         {"who"},
	"ss":          {"ss", "-tlnp"},
	"netstat":     {"netstat", "-tlnp"},
	"last":        {"last", "-n", "20"},
	"top1":        {"top", "-b", "-n", "1", "-o", "%CPU"},
	"iostat":      {"iostat", "-x", "1", "1"},
	"vmstat":      {"vmstat", "-s"},
	"lsof-ports":  {"lsof", "-i", "-P", "-n"},
	"svc-kumomta": {"systemctl", "status", "kumomta", "--no-pager", "-l"},
	"svc-dovecot": {"systemctl", "status", "dovecot", "--no-pager", "-l"},
	"svc-postfix": {"systemctl", "status", "postfix", "--no-pager", "-l"},
	"svc-fail2ban":{"systemctl", "status", "fail2ban", "--no-pager", "-l"},
	"journal-kumo":{"journalctl", "-u", "kumomta", "-n", "100", "--no-pager", "--output=short-iso"},
	"journal-dove":{"journalctl", "-u", "dovecot", "-n", "100", "--no-pager", "--output=short-iso"},
	"journal-f2b": {"journalctl", "-u", "fail2ban", "-n", "100", "--no-pager", "--output=short-iso"},
	"dmesg":       {"dmesg", "-T", "--level=err,warn", "-n", "50"},
}

// POST /api/system/run-command
// Runs a whitelisted read-only system command and returns output.
// Body: {"cmd": "ps"}
func (s *Server) handleRunCommand(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Cmd string `json:"cmd"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	args, ok := safeCommands[req.Cmd]
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "command not in allowlist"})
		return
	}

	ctx := r.Context()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.CombinedOutput()

	resp := map[string]interface{}{
		"cmd":    req.Cmd,
		"output": string(out),
		"ran_at": time.Now().Format(time.RFC3339),
	}
	if err != nil {
		resp["warning"] = err.Error() // non-zero exit is common for some commands, still return output
	}
	writeJSON(w, http.StatusOK, resp)
}
