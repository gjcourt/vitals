// Package domain contains the core business entities.
package domain

import (
	"errors"
	"time"
)

// User represents an authenticated user in the system.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	CreatedAt    time.Time
}

// Session represents an active user session.
type Session struct {
	Token     string
	UserID    int64
	UserAgent string
	IP        string
	ExpiresAt time.Time
	CreatedAt time.Time
}

var (
	// ErrInvalidCredentials indicates that the provided username or password was incorrect.
	ErrInvalidCredentials = errors.New("invalid username or password")
	// ErrSessionNotFound indicates that the requested session does not exist.
	ErrSessionNotFound = errors.New("session not found")
	// ErrSessionExpired indicates that the session has expired.
	ErrSessionExpired = errors.New("session expired")
	// ErrUserNotFound indicates that the user does not exist.
	ErrUserNotFound = errors.New("user not found")
)
