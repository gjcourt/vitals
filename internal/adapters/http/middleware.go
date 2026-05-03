package adapthttp

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"vitals/internal/domain"
)

type contextKey string

const userContextKey contextKey = "user"

// userFromContext returns the authenticated user from the request context.
// Returns nil when no user is associated with the request or when the value
// stored under userContextKey is itself nil.
func userFromContext(r *http.Request) *domain.User {
	if u, ok := r.Context().Value(userContextKey).(*domain.User); ok && u != nil {
		return u
	}
	return nil
}

// requireUser fetches the authenticated user from context or writes a 401
// response and returns nil. Handlers should bail out when nil is returned.
func requireUser(w http.ResponseWriter, r *http.Request) *domain.User {
	u := userFromContext(r)
	if u == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
	return u
}

// authMiddleware validates session tokens and forward auth headers.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		if errors.Is(err, domain.ErrSessionNotFound) || errors.Is(err, domain.ErrSessionExpired) {
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

		s.log().LogAttrs(r.Context(), s.requestLogLevel(rw.code), "http request",
			slog.String("remote", r.RemoteAddr),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rw.code),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// log returns the configured slog logger, falling back to slog.Default
// when the server was constructed without one (e.g. test zero values).
func (s *Server) log() *slog.Logger {
	if s.logger != nil {
		return s.logger
	}
	return slog.Default()
}

func (s *Server) requestLogLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
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
	if path == "/login" || path == "/signup" || path == "/health" {
		return true
	}
	if len(path) >= 6 && path[:6] == "/auth/" {
		return true
	}
	ext := ""
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			ext = path[i:]
			break
		}
	}
	return ext == ".css" || ext == ".js" || ext == ".ico" || ext == ".png" || ext == ".jpg" || ext == ".svg"
}
