package adapthttp

import (
	"context"
	"net/http"

	"biometrics/internal/app"
)

type contextKey string

const userContextKey contextKey = "user"

// authMiddleware validates session tokens and forward auth headers.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth if disabled (for tests)
		if s.disableAuth {
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

		// Fall back to cookie-based session
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		user, err := s.authSvc.ValidateSession(r.Context(), cookie.Value)
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
