package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps a *sql.DB and implements domain repository interfaces.
type DB struct {
	sql *sql.DB
}

// Open connects to PostgreSQL, pings, and runs migrations.
func Open(connStr string) (*DB, error) {
	s, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	s.SetMaxOpenConns(10)
	s.SetMaxIdleConns(5)
	s.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.PingContext(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}

	d := &DB{sql: s}
	if err := d.migrate(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error {
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	stmts := []string{
		"CREATE TABLE IF NOT EXISTS weights (day TEXT PRIMARY KEY, value DOUBLE PRECISION NOT NULL, unit TEXT NOT NULL CHECK(unit IN ('kg','lb')), created_at TIMESTAMPTZ NOT NULL);",
		"CREATE TABLE IF NOT EXISTS weight_events (id BIGSERIAL PRIMARY KEY, value DOUBLE PRECISION NOT NULL, unit TEXT NOT NULL CHECK(unit IN ('kg','lb')), created_at TIMESTAMPTZ NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_weight_events_created_at ON weight_events(created_at);",
		"CREATE TABLE IF NOT EXISTS water_events (id BIGSERIAL PRIMARY KEY, delta_liters DOUBLE PRECISION NOT NULL, created_at TIMESTAMPTZ NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_water_events_created_at ON water_events(created_at);",
		"CREATE TABLE IF NOT EXISTS users (id BIGSERIAL PRIMARY KEY, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, created_at TIMESTAMPTZ NOT NULL);",
		"CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE, expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);",
	}

	for _, stmt := range stmts {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// Add user_id columns to weight_events and water_events if they don't exist.
	alterStmts := []string{
		"ALTER TABLE weight_events ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);",
		"ALTER TABLE water_events ADD COLUMN IF NOT EXISTS user_id BIGINT REFERENCES users(id);",
		"CREATE INDEX IF NOT EXISTS idx_weight_events_user_id ON weight_events(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_water_events_user_id ON water_events(user_id);",
	}
	for _, stmt := range alterStmts {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// Assign orphaned events to the first user if one exists.
	_, _ = d.sql.ExecContext(ctx, "UPDATE weight_events SET user_id = (SELECT id FROM users ORDER BY id LIMIT 1) WHERE user_id IS NULL AND EXISTS (SELECT 1 FROM users);")
	_, _ = d.sql.ExecContext(ctx, "UPDATE water_events SET user_id = (SELECT id FROM users ORDER BY id LIMIT 1) WHERE user_id IS NULL AND EXISTS (SELECT 1 FROM users);")

	var eventCount int
	if err := d.sql.QueryRowContext(ctx, "SELECT COUNT(1) FROM weight_events;").Scan(&eventCount); err != nil {
		return fmt.Errorf("migrate: count weight_events: %w", err)
	}
	if eventCount == 0 {
		if _, err := d.sql.ExecContext(ctx, "INSERT INTO weight_events(value, unit, created_at) SELECT value, unit, created_at FROM weights;"); err != nil {
			return fmt.Errorf("migrate: migrate weights->weight_events: %w", err)
		}
	}
	return nil
}
