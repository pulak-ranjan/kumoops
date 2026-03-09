package api

// Multi-Node Cluster Management API
//
// Extends the existing RemoteServer system with:
// - Config push across all registered nodes
// - Aggregate metrics from all nodes
// - Per-node health summary
//
// Endpoints (all authenticated):
//   GET  /api/cluster/nodes          — list nodes with health
//   POST /api/cluster/push-config    — push generated config to all active nodes
//   GET  /api/cluster/metrics        — aggregate delivery metrics across all nodes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type nodeHealth struct {
	ID      uint   `json:"id"`
	Label   string `json:"label"`
	BaseURL string `json:"base_url"`
	Status   string `json:"status"` // "online", "offline", "unknown"
	PingMs   int64  `json:"ping_ms"`
	LastSeen string `json:"last_seen"`
}

// GET /api/cluster/nodes
func (s *Server) handleListClusterNodes(w http.ResponseWriter, r *http.Request) {
	servers, err := s.Store.ListRemoteServers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var nodes []nodeHealth
	for _, srv := range servers {
		nh := nodeHealth{
			ID:      srv.ID,
			Label:   srv.Name,
			BaseURL: srv.URL,
			Status:   "unknown",
			LastSeen: "never",
		}
		// Quick health ping
		start := time.Now()
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get(srv.URL + "/api/auth/ping")
		elapsed := time.Since(start).Milliseconds()
		if err == nil && resp.StatusCode < 500 {
			nh.Status = "online"
			nh.PingMs = elapsed
			nh.LastSeen = "just now"
			resp.Body.Close()
		} else {
			nh.Status = "offline"
		}
		nodes = append(nodes, nh)
	}
	if nodes == nil {
		nodes = []nodeHealth{}
	}
	writeJSON(w, http.StatusOK, nodes)
}

// POST /api/cluster/push-config — push KumoMTA config to all active remote nodes
func (s *Server) handleClusterPushConfig(w http.ResponseWriter, r *http.Request) {
	servers, err := s.Store.ListRemoteServers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type result struct {
		NodeID  uint   `json:"node_id"`
		Label   string `json:"label"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	var results []result

	for _, srv := range servers {
		// Call the remote node's apply-config endpoint using its API key
		url := srv.URL + "/api/config/apply"
		payload, _ := json.Marshal(map[string]string{"source": "cluster_push"})
		req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			results = append(results, result{srv.ID, srv.Name, "error", err.Error()})
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if srv.APIToken != "" {
			req.Header.Set("Authorization", "Bearer "+srv.APIToken)
		}
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			results = append(results, result{srv.ID, srv.Name, "error", err.Error()})
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			results = append(results, result{srv.ID, srv.Name, "ok", string(body)})
		} else {
			results = append(results, result{srv.ID, srv.Name, "error",
				fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))})
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pushed":  len(results),
		"results": results,
	})
}

// GET /api/cluster/metrics — aggregate basic stats across all nodes
func (s *Server) handleClusterMetrics(w http.ResponseWriter, r *http.Request) {
	servers, err := s.Store.ListRemoteServers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	type nodeMetrics struct {
		NodeID    uint        `json:"node_id"`
		Label     string      `json:"label"`
		Status    string      `json:"status"`
		QueueSize interface{} `json:"queue_size"`
	}
	var all []nodeMetrics

	for _, srv := range servers {
		nm := nodeMetrics{NodeID: srv.ID, Label: srv.Name, Status: "offline"}
		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("GET", srv.URL+"/api/queue/summary", nil)
		if srv.APIToken != "" {
			req.Header.Set("Authorization", "Bearer "+srv.APIToken)
		}
		if resp, err := client.Do(req); err == nil {
			var data interface{}
			if json.NewDecoder(resp.Body).Decode(&data) == nil {
				nm.QueueSize = data
				nm.Status = "online"
			}
			resp.Body.Close()
		}
		all = append(all, nm)
	}
	if all == nil {
		all = []nodeMetrics{}
	}
	writeJSON(w, http.StatusOK, all)
}
