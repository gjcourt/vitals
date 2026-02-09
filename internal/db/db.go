package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	sql *sql.DB
}

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

func (d *DB) Close() error {
	return d.sql.Close()
}

func (d *DB) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS weights (
			day TEXT PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL,
			unit TEXT NOT NULL CHECK(unit IN ('kg','lb')),
			created_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS weight_events (
			id BIGSERIAL PRIMARY KEY,
			value DOUBLE PRECISION NOT NULL,
			unit TEXT NOT NULL CHECK(unit IN ('kg','lb')),
			created_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_weight_events_created_at ON weight_events(created_at);`,
		`CREATE TABLE IF NOT EXISTS water_events (
			id BIGSERIAL PRIMARY KEY,
			delta_liters DOUBLE PRECISION NOT NULL,
			created_at TIMESTAMPTZ NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_water_events_created_at ON water_events(created_at);`,
	}

	for _, stmt := range stmts {
		if _, err := d.sql.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	// One-time migration: if we have legacy per-day weights but no events yet,
	// copy those rows into the event table.
	var eventCount int
	if err := d.sql.QueryRowContext(ctx, `SELECT COUNT(1) FROM weight_events;`).Scan(&eventCount); err != nil {
		return fmt.Errorf("migrate: count weight_events: %w", err)
	}
	if eventCount == 0 {
		if _, err := d.sql.ExecContext(ctx, `INSERT INTO weight_events(value, unit, created_at) SELECT value, unit, created_at FROM weights;`); err != nil {
			return fmt.Errorf("migrate: migrate weights->weight_events: %w", err)
		}
	}
	return nil
}

type WeightEntry struct {
	ID        int64     `json:"id"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}

type WaterEvent struct {
	ID          int64     `json:"id"`
	DeltaLiters float64   `json:"deltaLiters"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (d *DB) AddWeightEvent(ctx context.Context, value float64, unit string, createdAt time.Time) (int64, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx,
		`INSERT INTO weight_events(value, unit, created_at) VALUES($1, $2, $3) RETURNING id;`,
		value, unit, createdAt.UTC(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d *DB) DeleteLatestWeightEvent(ctx context.Context) (bool, error) {
	row := d.sql.QueryRowContext(ctx, `SELECT id FROM weight_events ORDER BY created_at DESC LIMIT 1;`)
	var id int64
	if err := row.Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	_, err := d.sql.ExecContext(ctx, `DELETE FROM weight_events WHERE id=$1;`, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (d *DB) LatestWeightForLocalDay(ctx context.Context, localDay string) (*WeightEntry, error) {
	dayStartLocal, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return nil, err
	}
	dayEndLocal := dayStartLocal.Add(24 * time.Hour)

	row := d.sql.QueryRowContext(ctx,
		`SELECT id, value, unit, created_at FROM weight_events WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC LIMIT 1;`,
		dayStartLocal.UTC(), dayEndLocal.UTC(),
	)

	var e WeightEntry
	if err := row.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e.Day = localDay
	return &e, nil
}

func (d *DB) ListRecentWeightEvents(ctx context.Context, limit int) ([]WeightEntry, error) {
	rows, err := d.sql.QueryContext(ctx,
		`SELECT id, value, unit, created_at FROM weight_events ORDER BY created_at DESC LIMIT $1;`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WeightEntry, 0, limit)
	for rows.Next() {
		var e WeightEntry
		if err := rows.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Day = e.CreatedAt.In(time.Local).Format("2006-01-02")
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) AddWaterEvent(ctx context.Context, deltaLiters float64, createdAt time.Time) (int64, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx,
		`INSERT INTO water_events(delta_liters, created_at) VALUES($1, $2) RETURNING id;`,
		deltaLiters, createdAt.UTC(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (d *DB) DeleteWaterEvent(ctx context.Context, id int64) error {
	_, err := d.sql.ExecContext(ctx, `DELETE FROM water_events WHERE id=$1;`, id)
	return err
}

func (d *DB) ListRecentWaterEvents(ctx context.Context, limit int) ([]WaterEvent, error) {
	rows, err := d.sql.QueryContext(ctx,
		`SELECT id, delta_liters, created_at FROM water_events ORDER BY created_at DESC LIMIT $1;`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]WaterEvent, 0, limit)
	for rows.Next() {
		var e WaterEvent
		if err := rows.Scan(&e.ID, &e.DeltaLiters, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (d *DB) WaterTotalForLocalDay(ctx context.Context, localDay string) (float64, error) {
	dayStartLocal, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return 0, err
	}
	dayEndLocal := dayStartLocal.Add(24 * time.Hour)

	row := d.sql.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(delta_liters), 0) FROM water_events WHERE created_at >= $1 AND created_at < $2;`,
		dayStartLocal.UTC(), dayEndLocal.UTC(),
	)
	var total float64
	if err := row.Scan(&total); err != nil {
		return 0, err
	}
	return total, nil
}
