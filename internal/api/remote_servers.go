package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pulak-ranjan/kumoops/internal/models"
)

// GET /api/servers
func (s *Server) handleListRemoteServers(w http.ResponseWriter, r *http.Request) {
	servers, err := s.Store.ListRemoteServers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db error"})
		return
	}
	// Redact tokens from list response
	for i := range servers {
		if len(servers[i].APIToken) > 8 {
			servers[i].APIToken = servers[i].APIToken[:4] + "****" + servers[i].APIToken[len(servers[i].APIToken)-4:]
		}
	}
	writeJSON(w, http.StatusOK, servers)
}

// POST /api/servers
func (s *Server) handleCreateRemoteServer(w http.ResponseWriter, r *http.Request) {
	var srv models.RemoteServer
	if err := json.NewDecoder(r.Body).Decode(&srv); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	srv.URL = strings.TrimRight(srv.URL, "/")
	if srv.Name == "" || srv.URL == "" || srv.APIToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, url, and api_token are required"})
		return
	}
	if err := s.Store.CreateRemoteServer(&srv); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save: " + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, srv)
}

// DELETE /api/servers/{id}
func (s *Server) handleDeleteRemoteServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := s.Store.DeleteRemoteServer(uint(id)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "delete failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/servers/{id}/test  — pings the remote instance
func (s *Server) handleTestRemoteServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	srv, err := s.Store.GetRemoteServer(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}

	client := &http.Client{Timeout: 8 * time.Second}
	req, _ := http.NewRequest("GET", srv.URL+"/api/stats/summary", nil)
	req.Header.Set("Authorization", "Bearer "+srv.APIToken)

	resp, err := client.Do(req)
	if err != nil {
		srv.Status = "offline"
		_ = s.Store.UpdateRemoteServer(srv)
		writeJSON(w, http.StatusOK, map[string]string{"status": "offline", "error": err.Error()})
		return
	}
	resp.Body.Close()

	srv.Status = "online"
	srv.LastSeen = time.Now()
	_ = s.Store.UpdateRemoteServer(srv)
	writeJSON(w, http.StatusOK, map[string]string{"status": "online"})
}

// GET|POST /api/servers/{id}/proxy  — transparent proxy to remote instance
// The caller passes the target path as a query param: ?path=/api/stats/summary
func (s *Server) handleProxyRemoteServer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid server id"})
		return
	}
	srv, err := s.Store.GetRemoteServer(uint(id))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "server not found"})
		return
	}

	targetPath := r.URL.Query().Get("path")
	if targetPath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path query param required"})
		return
	}

	// Build query string from original request, minus the "path" param
	origQuery := r.URL.Query()
	origQuery.Del("path")
	qs := origQuery.Encode()

	remoteURL := srv.URL + targetPath
	if qs != "" {
		sep := "?"
		if strings.Contains(targetPath, "?") {
			sep = "&"
		}
		remoteURL += sep + qs
	}

	// Forward body for POST/PUT
	var bodyReader io.Reader
	if r.Body != nil {
		bodyReader = r.Body
	}

	proxyReq, err := http.NewRequest(r.Method, remoteURL, bodyReader)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build request"})
		return
	}
	proxyReq.Header.Set("Authorization", "Bearer "+srv.APIToken)
	if r.Header.Get("Content-Type") != "" {
		proxyReq.Header.Set("Content-Type", r.Header.Get("Content-Type"))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": fmt.Sprintf("remote unreachable: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Update last_seen on successful proxy
	srv.LastSeen = time.Now()
	srv.Status = "online"
	_ = s.Store.UpdateRemoteServer(srv)

	// Stream response back
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body) //nolint:errcheck
}
