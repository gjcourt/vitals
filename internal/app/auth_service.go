// Package app holds the application services and business logic.
package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

	"vitals/internal/domain"
	"vitals/internal/ports/inbound"
	"vitals/internal/ports/outbound"

	"golang.org/x/crypto/bcrypt"
)

// authService implements inbound.AuthService.
type authService struct {
	users    outbound.UserRepository
	sessions outbound.SessionRepository
}

// NewAuthService creates a new authentication service.
func NewAuthService(users outbound.UserRepository, sessions outbound.SessionRepository) inbound.AuthService {
	return &authService{
		users:    users,
		sessions: sessions,
	}
}

// Login authenticates a user and creates a session.
func (s *authService) Login(ctx context.Context, username, password, userAgent, ip string) (string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil || user == nil {
		return "", domain.ErrInvalidCredentials
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", domain.ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.sessions.Create(ctx, user.ID, token, userAgent, ip, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

// Logout invalidates a session.
func (s *authService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// ValidateSession checks if a session token is valid and matches the user agent.
func (s *authService) ValidateSession(ctx context.Context, token, userAgent string) (*domain.User, error) {
	session, err := s.sessions.GetByToken(ctx, token)
	if err != nil || session == nil {
		return nil, domain.ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, token)
		return nil, domain.ErrSessionExpired
	}

	if session.UserAgent != userAgent {
		_ = s.sessions.Delete(ctx, token)
		return nil, domain.ErrSessionExpired
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	return user, nil
}

// CreateInitialUser creates the first user if no users exist.
func (s *authService) CreateInitialUser(ctx context.Context, username, password string) error {
	count, err := s.users.Count(ctx)
	if err != nil {
		return err
	}

	if count > 0 {
		return errors.New("users already exist")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = s.users.Create(ctx, username, string(hash))
	return err
}

// ValidateForwardAuth validates a request from Authelia forward auth.
func (s *authService) ValidateForwardAuth(ctx context.Context, remoteUser string) (*domain.User, error) {
	if remoteUser == "" {
		return nil, errors.New("no remote user header")
	}

	user, err := s.users.GetByUsername(ctx, remoteUser)
	if err != nil {
		user, err = s.users.Create(ctx, remoteUser, "")
		if err != nil {
			return nil, err
		}
	}

	return user, nil
}

// LoginWithUser creates a session for an already authenticated user (e.g. via SSO).
func (s *authService) LoginWithUser(ctx context.Context, username, userAgent, ip string) (string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		user, err = s.users.Create(ctx, username, "")
		if err != nil {
			user, err = s.users.GetByUsername(ctx, username)
			if err != nil {
				return "", err
			}
		}
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.sessions.Create(ctx, user.ID, token, userAgent, ip, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ConstantTimeCompare performs a constant-time comparison of two strings.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
