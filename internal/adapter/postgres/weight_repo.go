package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"biometrics/internal/domain"
)

// AddWeightEvent inserts a new weight event.
func (d *DB) AddWeightEvent(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx,
		"INSERT INTO weight_events(user_id, value, unit, created_at) VALUES($1, $2, $3, $4) RETURNING id;",
		userID, value, unit, createdAt.UTC(),
	).Scan(&id)
	return id, err
}

// DeleteLatestWeightEvent removes the most recent weight event for a user.
func (d *DB) DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx, "SELECT id FROM weight_events WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1;", userID).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	_, err = d.sql.ExecContext(ctx, "DELETE FROM weight_events WHERE id=$1 AND user_id=$2;", id, userID)
	return err == nil, err
}

// LatestWeightForLocalDay returns the most recent weight entry for a local calendar day for a user.
func (d *DB) LatestWeightForLocalDay(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error) {
	dayStart, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return nil, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	row := d.sql.QueryRowContext(ctx,
		"SELECT id, value, unit, created_at FROM weight_events WHERE user_id=$1 AND created_at >= $2 AND created_at < $3 ORDER BY created_at DESC LIMIT 1;",
		userID, dayStart.UTC(), dayEnd.UTC(),
	)

	var e domain.WeightEntry
	if err := row.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	e.UserID = userID
	e.Day = localDay
	return &e, nil
}

// ListRecentWeightEvents returns the most recent weight events up to limit for a user.
func (d *DB) ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	rows, err := d.sql.QueryContext(ctx,
		"SELECT id, value, unit, created_at FROM weight_events WHERE user_id=$1 ORDER BY created_at DESC LIMIT $2;", userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	out := make([]domain.WeightEntry, 0, limit)
	for rows.Next() {
		var e domain.WeightEntry
		if err := rows.Scan(&e.ID, &e.Value, &e.Unit, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.UserID = userID
		e.Day = e.CreatedAt.In(time.Local).Format("2006-01-02")
		out = append(out, e)
	}
	return out, rows.Err()
}
