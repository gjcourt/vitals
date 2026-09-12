package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

// openTemp returns a DB backed by a throwaway file. A file (not :memory:) is
// deliberate: :memory: would give each pooled connection its own private
// database, and it would not exercise WAL, which is the mode production runs in.
func openTemp(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestOpenRunsMigrations(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	for _, table := range []string{"weights", "weight_events", "water_events", "users", "sessions"} {
		var n int
		err := db.sql.QueryRowContext(ctx,
			"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?;", table).Scan(&n)
		if err != nil {
			t.Fatalf("query %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("table %q: got %d, want 1", table, n)
		}
	}
}

// migrate() runs on every Open. SQLite has no ADD COLUMN IF NOT EXISTS, so an
// unguarded add would fail on the second start -- which would only show up in
// production on the first pod restart. This is the regression that guard exists
// to prevent.
func TestMigrateIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")

	for i := range 3 {
		db, err := Open(path)
		if err != nil {
			t.Fatalf("Open #%d: %v", i+1, err)
		}
		if err := db.Close(); err != nil {
			t.Fatalf("Close #%d: %v", i+1, err)
		}
	}
}

func TestPragmasApplied(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	var journal string
	if err := db.sql.QueryRowContext(ctx, "PRAGMA journal_mode;").Scan(&journal); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want %q", journal, "wal")
	}

	// foreign_keys defaults OFF in SQLite and the sessions -> users cascade
	// depends on it.
	var fk int
	if err := db.sql.QueryRowContext(ctx, "PRAGMA foreign_keys;").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestHasColumn(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	// Added by the guarded ALTER path, so this also proves that path ran.
	has, err := db.hasColumn(ctx, "sessions", "user_agent")
	if err != nil {
		t.Fatalf("hasColumn: %v", err)
	}
	if !has {
		t.Error("sessions.user_agent: got false, want true")
	}

	has, err = db.hasColumn(ctx, "sessions", "definitely_not_a_column")
	if err != nil {
		t.Fatalf("hasColumn: %v", err)
	}
	if has {
		t.Error("sessions.definitely_not_a_column: got true, want false")
	}
}

// Timestamps are TEXT in SQLite where they were TIMESTAMPTZ in PostgreSQL. This
// asserts a time.Time survives the round trip, which is the single most likely
// place for the port to have gone quietly wrong.
func TestTimestampRoundTrip(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	want := time.Date(2026, 9, 13, 14, 30, 45, 0, time.UTC)
	if _, err := db.sql.ExecContext(ctx,
		"INSERT INTO weight_events (value, unit, created_at) VALUES (?, ?, ?);",
		82.5, "kg", want); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var (
		got   time.Time
		value float64
		unit  string
	)
	if err := db.sql.QueryRowContext(ctx,
		"SELECT value, unit, created_at FROM weight_events ORDER BY id DESC LIMIT 1;").
		Scan(&value, &unit, &got); err != nil {
		t.Fatalf("select: %v", err)
	}

	if value != 82.5 || unit != "kg" {
		t.Errorf("got (%v, %q), want (82.5, \"kg\")", value, unit)
	}
	if !got.UTC().Equal(want) {
		t.Errorf("created_at round trip: got %v, want %v", got.UTC(), want)
	}
}

// AUTOINCREMENT replaces BIGSERIAL; RETURNING is used by the repos and is only
// available in SQLite 3.35+, so this asserts the bundled driver is new enough.
func TestAutoincrementAndReturning(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()

	var first, second int64
	if err := db.sql.QueryRowContext(ctx,
		"INSERT INTO water_events (delta_liters, created_at) VALUES (?, ?) RETURNING id;",
		0.5, time.Now().UTC()).Scan(&first); err != nil {
		t.Fatalf("insert 1: %v", err)
	}
	if err := db.sql.QueryRowContext(ctx,
		"INSERT INTO water_events (delta_liters, created_at) VALUES (?, ?) RETURNING id;",
		0.75, time.Now().UTC()).Scan(&second); err != nil {
		t.Fatalf("insert 2: %v", err)
	}

	if first <= 0 {
		t.Errorf("first id = %d, want > 0", first)
	}
	if second != first+1 {
		t.Errorf("second id = %d, want %d", second, first+1)
	}
}

// The sessions -> users cascade is the only referential-integrity rule in the
// schema and it is inert unless PRAGMA foreign_keys is ON.
func TestForeignKeyCascade(t *testing.T) {
	db := openTemp(t)
	ctx := context.Background()
	now := time.Now().UTC()

	var userID int64
	if err := db.sql.QueryRowContext(ctx,
		"INSERT INTO users (username, password_hash, created_at) VALUES (?, ?, ?) RETURNING id;",
		"tester", "hash", now).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := db.sql.ExecContext(ctx,
		"INSERT INTO sessions (token, user_id, expires_at, created_at) VALUES (?, ?, ?, ?);",
		"tok", userID, now.Add(time.Hour), now); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	if _, err := db.sql.ExecContext(ctx, "DELETE FROM users WHERE id = ?;", userID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	var n int
	if err := db.sql.QueryRowContext(ctx, "SELECT count(*) FROM sessions WHERE token = ?;", "tok").Scan(&n); err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if n != 0 {
		t.Errorf("sessions after user delete: got %d, want 0 (cascade did not fire)", n)
	}
}
