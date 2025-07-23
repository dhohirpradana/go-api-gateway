package config

import (
	"dhohirpradana/api-gateway/proxy"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// GET /api/targets
func GetTargetsHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := proxy.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	if err != nil {
		http.Error(w, "failed to load config", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(cfg)
}

// POST /api/targets { "path": "/new", "target": "https://api.com/real" }
func AddTargetHandler(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path   string `json:"path"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Validasi path: harus diawali dengan "/"
	if !strings.HasPrefix(body.Path, "/") {
		http.Error(w, "Path must start with '/'", http.StatusBadRequest)
		return
	}

	// Validasi target: harus URL valid
	parsedURL, err := url.Parse(body.Target)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		http.Error(w, "Invalid target URL", http.StatusBadRequest)
		return
	}

	// Cek path yang dikecualikan
	disallowedPaths := map[string]bool{
		"/":          true,
		"/dashboard": true,
		"/metrics":   true,
		"/targets":   true,
	}
	if disallowedPaths[body.Path] {
		http.Error(w, "This path is reserved and cannot be overridden", http.StatusForbidden)
		return
	}

	// Load config
	cfg, err := proxy.LoadConfig()
	if err != nil {
		http.Error(w, "Failed to load config", http.StatusInternalServerError)
		return
	}

	// Cek jika path sudah ada
	if _, exists := cfg[body.Path]; exists {
		http.Error(w, "Path already exists", http.StatusConflict)
		return
	}

	// Simpan target baru
	cfg[body.Path] = body.Target
	if err := proxy.SaveConfig(cfg); err != nil {
		http.Error(w, "Failed to save config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Added route: %s → %s", body.Path, body.Target)
}

// DELETE /api/targets/posts
func DeleteTargetHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/targets/")
	if path == "" {
		http.Error(w, "missing path", http.StatusBadRequest)
		return
	}
	cfg, _ := proxy.LoadConfig()
	delete(cfg, "/"+path)
	if err := proxy.SaveConfig(cfg); err != nil {
		http.Error(w, "failed to save config", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
