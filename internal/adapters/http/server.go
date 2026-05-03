package adapthttp

import (
	"log/slog"
	"net/http"
	"path"
	"strings"

	"vitals/internal/ports/inbound"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig holds OIDC configuration.
type OIDCConfig struct {
	Provider     *oidc.Provider
	OAuth2Config oauth2.Config
	Enabled      bool
}

// Options configures optional Server dependencies.
type Options struct {
	// Logger used for structured logging. Defaults to slog.Default() when nil.
	Logger *slog.Logger
	// OIDC, when Enabled, wires SSO endpoints with the supplied provider/config.
	OIDC OIDCConfig
	// CookieSecure forces the Secure flag on auth cookies even when the
	// request is reaching the server over plain HTTP (e.g. behind a
	// TLS-terminating reverse proxy).
	CookieSecure bool
}

// Server is the driving HTTP adapter that routes requests to application
// services via inbound port interfaces.
type Server struct {
	weight       inbound.WeightService
	water        inbound.WaterService
	charts       inbound.ChartsService
	authSvc      inbound.AuthService
	webDir       string
	disableAuth  bool
	oidcConfig   OIDCConfig
	logger       *slog.Logger
	cookieSecure bool
}

// requestIsSecure reports whether the request reached the server over a
// connection considered TLS-protected. It honours the cookieSecure flag
// (typical when behind a TLS-terminating reverse proxy) and falls back
// to the X-Forwarded-Proto header / r.TLS.
func (s *Server) requestIsSecure(r *http.Request) bool {
	if s.cookieSecure {
		return true
	}
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		// May be a comma-separated list; the first entry is the original.
		first := proto
		if i := strings.IndexByte(proto, ','); i >= 0 {
			first = proto[:i]
		}
		return strings.EqualFold(strings.TrimSpace(first), "https")
	}
	return false
}

// New creates a Server wired to the given application services.
func New(ws inbound.WeightService, wa inbound.WaterService, cs inbound.ChartsService, as inbound.AuthService, webDir string, opts ...Options) *Server {
	var opt Options
	if len(opts) > 0 {
		opt = opts[0]
	}
	logger := opt.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{
		weight:       ws,
		water:        wa,
		charts:       cs,
		authSvc:      as,
		webDir:       webDir,
		disableAuth:  false,
		oidcConfig:   opt.OIDC,
		logger:       logger,
		cookieSecure: opt.CookieSecure,
	}
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
	api.HandleFunc("/auth/config", s.handleConfig)
	api.HandleFunc("/auth/oidc/login", s.handleSSOLogin)
	api.HandleFunc("/auth/oidc/callback", s.handleSSOCallback)

	// Protected API endpoints
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

	root.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(s.webDir, "login.html"))
	})
	root.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(s.webDir, "signup.html"))
	})

	root.Handle("/", s.requireAuthHTML(spaFromDisk(s.webDir)))

	return s.loggingMiddleware(withNoCache(root))
}
