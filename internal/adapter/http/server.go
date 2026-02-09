package adapthttp

import (
	"net/http"

	"biometrics/internal/app"
)

// Server is the driving HTTP adapter that routes requests to application
// services.
type Server struct {
	weight      *app.WeightService
	water       *app.WaterService
	charts      *app.ChartsService
	authSvc     *app.AuthService
	webDir      string
	disableAuth bool
}

// New creates a Server wired to the given application services.
func New(ws *app.WeightService, wa *app.WaterService, cs *app.ChartsService, as *app.AuthService, webDir string) *Server {
	return &Server{weight: ws, water: wa, charts: cs, authSvc: as, webDir: webDir, disableAuth: false}
}

// WithoutAuth disables authentication (for testing).
func (s *Server) WithoutAuth() *Server {
	s.disableAuth = true
	return s
}

// Handler returns the root http.Handler for the application.
func (s *Server) Handler() http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	// Auth endpoints (public)
	api.HandleFunc("/auth/login", s.handleLogin)
	api.HandleFunc("/auth/logout", s.handleLogout)
	api.HandleFunc("/auth/setup", s.handleSetupUser)

	// Protected API endpoints - wrap each handler with auth middleware
	api.Handle("/weight/today", s.authMiddleware(http.HandlerFunc(s.handleWeightToday)))
	api.Handle("/weight/recent", s.authMiddleware(http.HandlerFunc(s.handleWeightRecent)))
	api.Handle("/weight/undo-last", s.authMiddleware(http.HandlerFunc(s.handleWeightUndoLast)))

	api.Handle("/water/today", s.authMiddleware(http.HandlerFunc(s.handleWaterToday)))
	api.Handle("/water/event", s.authMiddleware(http.HandlerFunc(s.handleWaterEvent)))
	api.Handle("/water/recent", s.authMiddleware(http.HandlerFunc(s.handleWaterRecent)))
	api.Handle("/water/undo-last", s.authMiddleware(http.HandlerFunc(s.handleWaterUndoLast)))

	api.Handle("/charts/daily", s.authMiddleware(http.HandlerFunc(s.handleChartsDaily)))

	root := http.NewServeMux()
	root.Handle("/api/", http.StripPrefix("/api", api))
	root.Handle("/", spaFromDisk(s.webDir))

	return withNoCache(root)
}
