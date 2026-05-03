package adapthttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func parseJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func intQuery(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func localDayString(t time.Time) string {
	return t.In(time.Local).Format("2006-01-02")
}

func withNoCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func spaFromDisk(dir string) http.Handler {
	fileServer := http.FileServer(http.Dir(dir))
	indexPath := path.Join(dir, "index.html")
	chartsPath := path.Join(dir, "charts.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPath := path.Clean(r.URL.Path)
		if reqPath == "/" {
			http.ServeFile(w, r, indexPath)
			return
		}
		if reqPath == "/charts" {
			http.ServeFile(w, r, chartsPath)
			return
		}

		staticPath := path.Join(dir, reqPath)
		if _, err := os.Stat(staticPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}
