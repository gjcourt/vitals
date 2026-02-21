package adapthttp

import (
	"context"
	"log"
	"net/http"
	"os"
	"path"

	"biometrics/internal/app"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCConfig holds OIDC configuration.
type OIDCConfig struct {
	Provider     *oidc.Provider
	OAuth2Config oauth2.Config
	Enabled      bool
}

// Server is the driving HTTP adapter that routes requests to application
// services.
type Server struct {
	weight      *app.WeightService
	water       *app.WaterService
	charts      *app.ChartsService
	authSvc     *app.AuthService
	webDir      string
	disableAuth bool
	oidcConfig  OIDCConfig
}

// New creates a Server wired to the given application services.
func New(ws *app.WeightService, wa *app.WaterService, cs *app.ChartsService, as *app.AuthService, webDir string) *Server {
	s := &Server{weight: ws, water: wa, charts: cs, authSvc: as, webDir: webDir, disableAuth: false}

	// Initialize OIDC (SSO) if configured
	if issuer := os.Getenv("SSO_ISSUER_URL"); issuer != "" {
		ctx := backgroundContext() // Use a detached context or background
		provider, err := oidc.NewProvider(ctx, issuer)
		if err != nil {
			log.Printf("Failed to initialize OIDC provider: %v", err)
		} else {
			s.oidcConfig = OIDCConfig{
				Provider: provider,
				OAuth2Config: oauth2.Config{
					ClientID:     os.Getenv("SSO_CLIENT_ID"),
					ClientSecret: os.Getenv("SSO_CLIENT_SECRET"),
					RedirectURL:  os.Getenv("SSO_REDIRECT_URL"),
					Endpoint:     provider.Endpoint(),
					Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
				},
				Enabled: true,
			}
			log.Println("SSO (OIDC) enabled")
		}
	}

	return s
}

// backgroundContext returns a context for initialization.
func backgroundContext() context.Context {
	return context.Background()
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

	// Server HTML files for login/signup directly to ensure they are found and public
	root.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(s.webDir, "login.html"))
	})
	root.HandleFunc("/signup", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join(s.webDir, "signup.html"))
	})

	// Apply HTML auth middleware to SPA catch-all
	root.Handle("/", s.requireAuthHTML(spaFromDisk(s.webDir)))

	return s.loggingMiddleware(withNoCache(root))
}
