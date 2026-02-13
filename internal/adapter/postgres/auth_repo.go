// Package postgres implements the domain repositories using PostgreSQL.
package postgres

import (
	"context"
	"database/sql"
	"time"

	"biometrics/internal/domain"
)

// GetByUsername retrieves a user by username.
func (d *DB) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var u domain.User
	err := d.sql.QueryRowContext(ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByID retrieves a user by ID.
func (d *DB) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := d.sql.QueryRowContext(ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE id = $1",
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Create creates a new user.
func (d *DB) Create(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	var u domain.User
	err := d.sql.QueryRowContext(ctx,
		"INSERT INTO users (username, password_hash, created_at) VALUES ($1, $2, $3) RETURNING id, username, password_hash, created_at",
		username, passwordHash, time.Now(),
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// Count returns the total number of users.
func (d *DB) Count(ctx context.Context) (int, error) {
	var count int
	err := d.sql.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	return count, err
}

// SessionRepo implements session repository operations on DB.
type SessionRepo struct {
	db *DB
}

// NewSessionRepo wraps a DB as a SessionRepository.
func NewSessionRepo(db *DB) *SessionRepo {
	return &SessionRepo{db: db}
}

// Create creates a new session.
func (r *SessionRepo) Create(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	_, err := r.db.sql.ExecContext(ctx,
		"INSERT INTO sessions (user_id, token, expires_at, created_at) VALUES ($1, $2, $3, $4)",
		userID, token, expiresAt, time.Now(),
	)
	return err
}

// GetByToken retrieves a session by token.
func (r *SessionRepo) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	var s domain.Session
	err := r.db.sql.QueryRowContext(ctx,
		"SELECT token, user_id, expires_at, created_at FROM sessions WHERE token = $1",
		token,
	).Scan(&s.Token, &s.UserID, &s.ExpiresAt, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// Delete deletes a session by token.
func (r *SessionRepo) Delete(ctx context.Context, token string) error {
	_, err := r.db.sql.ExecContext(ctx, "DELETE FROM sessions WHERE token = $1", token)
	return err
}

// DeleteExpired deletes all expired sessions.
func (r *SessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.sql.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < $1", time.Now())
	return err
}
