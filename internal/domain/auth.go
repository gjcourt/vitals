// Package domain contains the core business entities and interfaces.
package domain

import (
	"context"
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
	ExpiresAt time.Time
	CreatedAt time.Time
}

// UserRepository defines the port for user persistence operations.
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, username, passwordHash string) (*User, error)
	Count(ctx context.Context) (int, error)
}

// SessionRepository defines the port for session persistence operations.
type SessionRepository interface {
	Create(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}
