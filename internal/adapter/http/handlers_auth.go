// Package adapthttp implements the HTTP adapter for the application.
package adapthttp

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"biometrics/internal/app"

	"github.com/coreos/go-oidc/v3/oidc"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token, err := s.authSvc.Login(r.Context(), req.Username, req.Password, r.UserAgent(), r.RemoteAddr)
	if err == app.ErrInvalidCredentials {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cookie, err := r.Cookie("session")
	if err == nil {
		_ = s.authSvc.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleSetupUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if err := s.authSvc.CreateInitialUser(r.Context(), req.Username, req.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"sso_enabled": s.oidcConfig.Enabled,
	})
}

func (s *Server) handleSSOLogin(w http.ResponseWriter, r *http.Request) {
	if !s.oidcConfig.Enabled {
		http.Error(w, "sso disabled", http.StatusNotFound)
		return
	}
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode, // Lax required for cross-site redirect returns
		MaxAge:   300,
	})
	http.Redirect(w, r, s.oidcConfig.OAuth2Config.AuthCodeURL(state), http.StatusFound)
}

func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	if !s.oidcConfig.Enabled {
		http.Error(w, "sso disabled", http.StatusNotFound)
		return
	}

	state, err := r.Cookie("oauth_state")
	if err != nil || r.URL.Query().Get("state") != state.Value {
		http.Error(w, "invalid state", http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

	token, err := s.oidcConfig.OAuth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "failed to exchange token", http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "no id_token", http.StatusInternalServerError)
		return
	}

	idToken, err := s.oidcConfig.Provider.Verifier(&oidc.Config{ClientID: s.oidcConfig.OAuth2Config.ClientID}).Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "failed to verify token", http.StatusInternalServerError)
		return
	}

	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}
	if err = idToken.Claims(&claims); err != nil {
		http.Error(w, "failed to parse claims", http.StatusInternalServerError)
		return
	}

	username := claims.Email
	if username == "" {
		username = claims.Sub
	}

	sessionToken, err := s.authSvc.LoginWithUser(r.Context(), username, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		http.Error(w, "login failed", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionToken,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func generateState() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
