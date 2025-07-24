package main

import (
	"fmt"
	"log"
	"net/http"

	"dhohirpradana/api-gateway/config"
	"dhohirpradana/api-gateway/monitor"
	"dhohirpradana/api-gateway/proxy"
	"encoding/json"
)

func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	stats := monitor.All()

	var out []map[string]interface{}
	for path, s := range stats {
		out = append(out, map[string]interface{}{
			"path":    path,
			"success": s.Success,
			"fail":    s.Fail,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func main() {
	monitor.Init("stats.db")

	http.HandleFunc("/", proxy.NewProxyHandler())
	http.HandleFunc("/metrics", dashboardHandler)

	fs := http.FileServer(http.Dir("static"))
	http.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/", http.StatusMovedPermanently)
	})
	http.Handle("/dashboard/", http.StripPrefix("/dashboard/", fs))

	http.HandleFunc("/targets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			config.GetTargetsHandler(w, r)
		case http.MethodPost:
			config.AddTargetHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	http.HandleFunc("/targets/", config.DeleteTargetHandler)

	fmt.Println("Gateway running at http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
