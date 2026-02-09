package adapthttp

import (
	"net/http"

	"biometrics/internal/app"
)

// Server is the driving HTTP adapter that routes requests to application
// services.
type Server struct {
	weight *app.WeightService
	water  *app.WaterService
	charts *app.ChartsService
	webDir string
}

// New creates a Server wired to the given application services.
func New(ws *app.WeightService, wa *app.WaterService, cs *app.ChartsService, webDir string) *Server {
	return &Server{weight: ws, water: wa, charts: cs, webDir: webDir}
}

// Handler returns the root http.Handler for the application.
func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	api.HandleFunc("/weight/today", s.handleWeightToday)
	api.HandleFunc("/weight/recent", s.handleWeightRecent)
	api.HandleFunc("/weight/undo-last", s.handleWeightUndoLast)

	api.HandleFunc("/water/today", s.handleWaterToday)
	api.HandleFunc("/water/event", s.handleWaterEvent)
	api.HandleFunc("/water/recent", s.handleWaterRecent)
	api.HandleFunc("/water/undo-last", s.handleWaterUndoLast)

	api.HandleFunc("/charts/daily", s.handleChartsDaily)

	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", api))
	root.Handle("/", spaFromDisk(s.webDir))

	return withNoCache(root)
}
