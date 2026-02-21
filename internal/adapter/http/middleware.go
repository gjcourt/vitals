package adapthttp

import (
	"context"
	"log"
	"net/http"
	"time"

	"biometrics/internal/app"
	"biometrics/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

// userFromContext returns the authenticated user from the request context.
func userFromContext(r *http.Request) *domain.User {
	if u, ok := r.Context().Value(userContextKey).(*domain.User); ok {
		return u
	}
	return nil
}

// authMiddleware validates session tokens and forward auth headers.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled (for tests / dev) — inject a default user
		if s.disableAuth {
			ctx := context.WithValue(r.Context(), userContextKey, &domain.User{ID: 0, Username: "dev"})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Check for Authelia forward auth header first
		if remoteUser := r.Header.Get("Remote-User"); remoteUser != "" {
			user, err := s.authSvc.ValidateForwardAuth(r.Context(), remoteUser)
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Fall back to cookie-based session
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.authSvc.ValidateSession(r.Context(), cookie.Value, r.UserAgent())
		if err == app.ErrSessionNotFound || err == app.ErrSessionExpired {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loggingMiddleware logs the details of each request
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &loggingResponseWriter{ResponseWriter: w, code: http.StatusOK}
		next.ServeHTTP(rw, r)

		log.Printf("[HTTP] %s %s %s %d %v", r.RemoteAddr, r.Method, r.URL.Path, rw.code, time.Since(start))
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *loggingResponseWriter) WriteHeader(code int) {
	rw.code = code
	rw.ResponseWriter.WriteHeader(code)
}

// requireAuthHTML enforces authentication for HTML pages, redirecting to login if needed.
func (s *Server) requireAuthHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.disableAuth || isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check for Authelia forward auth header first
		if remoteUser := r.Header.Get("Remote-User"); remoteUser != "" {
			user, err := s.authSvc.ValidateForwardAuth(r.Context(), remoteUser)
			if err == nil && user != nil {
				ctx := context.WithValue(r.Context(), userContextKey, user)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Check session cookie
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		user, err := s.authSvc.ValidateSession(r.Context(), cookie.Value, r.UserAgent())
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isPublicPath(path string) bool {
	// Public paths
	if path == "/login" || path == "/signup" || path == "/health" {
		return true
	}

	// Public prefixes
	if len(path) >= 6 && path[:6] == "/auth/" {
		return true
	}

	// Static assets
	ext := ""
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			ext = path[i:]
			break
		}
	}
	// Allow standard assets
	if ext == ".css" || ext == ".js" || ext == ".ico" || ext == ".png" || ext == ".jpg" || ext == ".svg" {
		return true
	}

	return false
}
