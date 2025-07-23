package proxy

import (
	"dhohirpradana/api-gateway/monitor"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Config map[string]string

var (
	mutex      sync.RWMutex
	configFile = "config/targets.json"
)

func LoadConfig() (Config, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return Config{}, nil
	}
	file, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SaveConfig(cfg Config) error {
	mutex.Lock()
	defer mutex.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFile, data, 0644)
}

func NewProxyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		path := r.URL.Path

		// 🔄 Selalu baca config terbaru
		cfg, err := LoadConfig()
		if err != nil {
			http.Error(w, "Failed to load config", http.StatusInternalServerError)
			monitor.Record(path, false)
			return
		}

		targetStr, ok := cfg[path]
		if !ok {
			http.Error(w, "API not configured", http.StatusNotFound)
			monitor.Record(path, false)
			return
		}

		remote, err := url.Parse(targetStr)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusInternalServerError)
			monitor.Record(path, false)
			return
		}

		r.URL.Path = strings.TrimPrefix(r.URL.Path, path)
		r.Host = remote.Host

		proxy := httputil.NewSingleHostReverseProxy(remote)

		// Logging dan monitoring
		proxy.ModifyResponse = func(resp *http.Response) error {
			duration := time.Since(start)
			log.Printf("[SUCCESS] %s %s => %d in %s", r.Method, path, resp.StatusCode, duration)
			monitor.Record(path, true)
			return nil
		}
		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, e error) {
			duration := time.Since(start)
			log.Printf("[FAIL] %s %s => error: %v in %s", r.Method, path, e, duration)
			monitor.Record(path, false)
			http.Error(w, "Upstream error", http.StatusBadGateway)
		}

		proxy.ServeHTTP(w, r)
	}
}
