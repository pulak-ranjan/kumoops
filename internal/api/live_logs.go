package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
	// Return up to 3 most recent files
	result := make([]string, 0, 3)
	for i, f := range infos {
		if i >= 3 {
			break
		}
		result = append(result, f.path)
	}
	return result
}
