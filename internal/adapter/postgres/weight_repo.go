package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"biometrics/internal/domain"
)

// AddWeightEvent inserts a new weight event.
func (d *DB) AddWeightEvent(ctx context.Context, value float64, unit string, createdAt time.Time) (int64, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx,
		"INSERT INTO weight_events(value, unit, created_at) VALUES($1, $2, $3) RETURNING id;",
		value, unit, createdAt.UTC(),
	).Scan(&id)
	return id, err
}

// DeleteLatestWeightEvent removes the most recent weight event.
func (d *DB) DeleteLatestWeightEvent(ctx context.Context) (bool, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx, "SELECT id FROM weight_events ORDER BY created_at DESC LIMIT 1;").Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	_, err = d.sql.ExecContext(ctx, "DELETE FROM weight_events WHERE id=$1;", id)
	return err == nil, err
}

// LatestWeightForLocalDay returns the most recent weight entry for a local calendar day.
func (d *DB) LatestWeightForLocalDay(ctx context.Context, localDay string) (*domain.WeightEntry, error) {
	dayStart, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	row := d.sql.QueryRowContext(ctx,
		"SELECT id, value, unit, created_at FROM weight_events WHERE created_at >= $1 AND created_at < $2 ORDER BY created_at DESC LIMIT 1;",
		dayStart.UTC(), dayEnd.UTC(),
	)

	var e domain.WeightEntry
	if err := row.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e.Day = localDay
	return &e, nil
}

// ListRecentWeightEvents returns the most recent weight events up to limit.
func (d *DB) ListRecentWeightEvents(ctx context.Context, limit int) ([]domain.WeightEntry, error) {
	rows, err := d.sql.QueryContext(ctx,
		"SELECT id, value, unit, created_at FROM weight_events ORDER BY created_at DESC LIMIT $1;", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.WeightEntry, 0, limit)
	for rows.Next() {
		var e domain.WeightEntry
		if err := rows.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.Day = e.CreatedAt.In(time.Local).Format("2006-01-02")
		out = append(out, e)
	}
	return out, rows.Err()
}
