package app

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"time"

	"biometrics/internal/domain"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionExpired     = errors.New("session expired")
	ErrUserNotFound       = errors.New("user not found")
)

// AuthService handles authentication and session management.
type AuthService struct {
	users    domain.UserRepository
	sessions domain.SessionRepository
}

// NewAuthService creates a new authentication service.
func NewAuthService(users domain.UserRepository, sessions domain.SessionRepository) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
	}
}

// Login authenticates a user and creates a session.
func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if err != nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.sessions.Create(ctx, user.ID, token, expiresAt); err != nil {
		return "", err
	}

	return token, nil
}

// Logout invalidates a session.
func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, token)
}

// ValidateSession checks if a session token is valid.
func (s *AuthService) ValidateSession(ctx context.Context, token string) (*domain.User, error) {
	session, err := s.sessions.GetByToken(ctx, token)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		_ = s.sessions.Delete(ctx, token)
		return nil, ErrSessionExpired
	}

	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

// CreateInitialUser creates the first user if no users exist.
func (s *AuthService) CreateInitialUser(ctx context.Context, username, password string) error {
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
// It checks for the Remote-User header set by Authelia.
func (s *AuthService) ValidateForwardAuth(ctx context.Context, remoteUser string) (*domain.User, error) {
	if remoteUser == "" {
		return nil, errors.New("no remote user header")
	}

	user, err := s.users.GetByUsername(ctx, remoteUser)
	if err != nil {
		// Auto-create user from SSO if they don't exist
		user, err = s.users.Create(ctx, remoteUser, "")
		if err != nil {
			return nil, err
		}
	}

	return user, nil
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
