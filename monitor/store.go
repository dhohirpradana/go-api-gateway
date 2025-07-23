package monitor

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

type Stats struct {
	Success int
	Fail    int
}

var (
	db    *sql.DB
	stats = make(map[string]*Stats)
)

// Init monitoring DB
func Init(filename string) {
	var err error
	db, err = sql.Open("sqlite", filename)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS stats (
		path TEXT PRIMARY KEY,
		success_count INTEGER DEFAULT 0,
		fail_count INTEGER DEFAULT 0
	)`)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	loadFromDB()
}

// Load all stats from DB to memory
func loadFromDB() {
	rows, err := db.Query("SELECT path, success_count, fail_count FROM stats")
	if err != nil {
		log.Printf("failed to load stats: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var path string
		var succ, fail int
		if err := rows.Scan(&path, &succ, &fail); err == nil {
			stats[path] = &Stats{Success: succ, Fail: fail}
		}
	}
}

// Record monitoring event
func Record(path string, success bool) {
	s, ok := stats[path]
	if !ok {
		s = &Stats{}
		stats[path] = s
	}

	if success {
		s.Success++
		_, _ = db.Exec(`INSERT INTO stats(path, success_count, fail_count)
			VALUES(?, 1, 0)
			ON CONFLICT(path) DO UPDATE SET success_count = success_count + 1`, path)
	} else {
		s.Fail++
		_, _ = db.Exec(`INSERT INTO stats(path, success_count, fail_count)
			VALUES(?, 0, 1)
			ON CONFLICT(path) DO UPDATE SET fail_count = fail_count + 1`, path)
	}
}

// Delete a target from monitoring stats
func DeleteTarget(path string) {
	delete(stats, path)
	_, err := db.Exec("DELETE FROM stats WHERE path = ?", path)
	if err != nil {
		log.Printf("failed to delete target %s: %v", path, err)
	}
}

func DashboardHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT path, success_count, fail_count FROM stats")
		if err != nil {
			http.Error(w, "Failed to read stats", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		html := `<html><head><title>API Stats</title></head><body>`
		html += `<h1>API Monitoring Dashboard</h1><table border="1"><tr><th>Path</th><th>Success</th><th>Fail</th></tr>`

		for rows.Next() {
			var path string
			var success, fail int
			rows.Scan(&path, &success, &fail)
			html += fmt.Sprintf("<tr><td>%s</td><td>%d</td><td>%d</td></tr>", path, success, fail)
		}

		html += `</table></body></html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	}
}

// Get all stats for dashboard
func All() map[string]*Stats {
	return stats
}
