package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps a *sql.DB and implements domain repository interfaces.
type DB struct {
	sql *sql.DB
}

// Open opens the SQLite database at path, pings, and runs migrations.
//
// SQLite allows exactly one writer. The pool is capped at a single connection
// so concurrent requests queue in database/sql rather than colliding at the
// driver and surfacing as SQLITE_BUSY. This mirrors what this app did before it
// moved to PostgreSQL (commit c988243).
func Open(path string) (*DB, error) {
	// _txlock=immediate takes the write lock at BEGIN instead of on first write,
	// which turns lock contention into a clean wait rather than a mid-transaction
	// SQLITE_BUSY that database/sql cannot retry.
	s, err := sql.Open("sqlite", path+"?_txlock=immediate")
	if err != nil {
		return nil, err
	}
	s.SetMaxOpenConns(1)
	s.SetMaxIdleConns(1)
	s.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.PingContext(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}

	// WAL lets reads proceed during a write; foreign_keys is OFF by default in
	// SQLite and the schema relies on ON DELETE CASCADE for sessions.
	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	} {
		if _, err := s.ExecContext(ctx, pragma); err != nil {
			_ = s.Close()
			return nil, fmt.Errorf("open: %s: %w", pragma, err)
		}
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
		// TIMESTAMPTZ has no SQLite equivalent. These are declared DATETIME, NOT
		// TEXT, and that distinction is load-bearing: modernc.org/sqlite decides
		// whether to hand back a time.Time or a raw string based on the DECLARED
		// column type. With TEXT every scan into a time.Time fails at runtime with
		// "unsupported Scan, storing driver.Value type string into type *time.Time".
		// Verified 2026-09-13: TEXT fails, DATETIME and TIMESTAMP both round-trip.
		// SQLite stores the value as text either way -- only the affinity differs.
		"CREATE TABLE IF NOT EXISTS weights (day TEXT PRIMARY KEY, value REAL NOT NULL, unit TEXT NOT NULL CHECK(unit IN ('kg','lb')), created_at DATETIME NOT NULL);",
		"CREATE TABLE IF NOT EXISTS weight_events (id INTEGER PRIMARY KEY AUTOINCREMENT, value REAL NOT NULL, unit TEXT NOT NULL CHECK(unit IN ('kg','lb')), created_at DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_weight_events_created_at ON weight_events(created_at);",
		"CREATE TABLE IF NOT EXISTS water_events (id INTEGER PRIMARY KEY AUTOINCREMENT, delta_liters REAL NOT NULL, created_at DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_water_events_created_at ON water_events(created_at);",
		"CREATE TABLE IF NOT EXISTS users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT UNIQUE NOT NULL, password_hash TEXT NOT NULL, created_at DATETIME NOT NULL);",
		"CREATE TABLE IF NOT EXISTS sessions (token TEXT PRIMARY KEY, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, expires_at DATETIME NOT NULL, created_at DATETIME NOT NULL);",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);",
	}

	for _, stmt := range stmts {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// SQLite has no ALTER TABLE ... ADD COLUMN IF NOT EXISTS, so each add is
	// guarded by inspecting PRAGMA table_info first. Re-running an unguarded add
	// is an error, not a no-op, which would make migrate() fail on every restart
	// after the first.
	type addCol struct{ table, column, ddl string }
	adds := []addCol{
		{"weight_events", "user_id", "ALTER TABLE weight_events ADD COLUMN user_id INTEGER REFERENCES users(id);"},
		{"water_events", "user_id", "ALTER TABLE water_events ADD COLUMN user_id INTEGER REFERENCES users(id);"},
		{"sessions", "user_agent", "ALTER TABLE sessions ADD COLUMN user_agent TEXT;"},
		{"sessions", "ip", "ALTER TABLE sessions ADD COLUMN ip TEXT;"},
	}
	for _, a := range adds {
		has, err := d.hasColumn(ctx, a.table, a.column)
		if err != nil {
			return fmt.Errorf("migrate: check %s.%s: %w", a.table, a.column, err)
		}
		if has {
			continue
		}
		if _, err := d.sql.ExecContext(ctx, a.ddl); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	idx := []string{
		"CREATE INDEX IF NOT EXISTS idx_weight_events_user_id ON weight_events(user_id);",
		"CREATE INDEX IF NOT EXISTS idx_water_events_user_id ON water_events(user_id);",
	}
	for _, stmt := range idx {
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

// hasColumn reports whether table already has the named column.
func (d *DB) hasColumn(ctx context.Context, table, column string) (bool, error) {
	// PRAGMA does not accept bound parameters for the table name. table is only
	// ever a package-internal constant from the migration list, never user input.
	rows, err := d.sql.QueryContext(ctx, "PRAGMA table_info("+table+");")
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			dfltValue  sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
