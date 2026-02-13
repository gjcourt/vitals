package postgres

import (
	"context"
	"time"

	"biometrics/internal/domain"
)

// AddWaterEvent inserts a new water intake event.
func (d *DB) AddWaterEvent(ctx context.Context, deltaLiters float64, createdAt time.Time) (int64, error) {
	var id int64
	err := d.sql.QueryRowContext(ctx,
		"INSERT INTO water_events(delta_liters, created_at) VALUES($1, $2) RETURNING id;",
		deltaLiters, createdAt.UTC(),
	).Scan(&id)
	return id, err
}

// DeleteWaterEvent removes a water event by ID.
func (d *DB) DeleteWaterEvent(ctx context.Context, id int64) error {
	_, err := d.sql.ExecContext(ctx, "DELETE FROM water_events WHERE id=$1;", id)
	return err
}

// ListRecentWaterEvents returns the most recent water events up to limit.
func (d *DB) ListRecentWaterEvents(ctx context.Context, limit int) ([]domain.WaterEvent, error) {
	rows, err := d.sql.QueryContext(ctx,
		"SELECT id, delta_liters, created_at FROM water_events ORDER BY created_at DESC LIMIT $1;", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	out := make([]domain.WaterEvent, 0, limit)
	for rows.Next() {
		var e domain.WaterEvent
		if err := rows.Scan(&e.ID, &e.DeltaLiters, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// WaterTotalForLocalDay returns the total water intake for a local calendar day.
func (d *DB) WaterTotalForLocalDay(ctx context.Context, localDay string) (float64, error) {
	dayStart, err := time.ParseInLocation("2006-01-02", localDay, time.Local)
	if err != nil {
		return 0, err
	}
	dayEnd := dayStart.Add(24 * time.Hour)

	var total float64
	err = d.sql.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(delta_liters), 0) FROM water_events WHERE created_at >= $1 AND created_at < $2;",
		dayStart.UTC(), dayEnd.UTC(),
	).Scan(&total)
	return total, err
}
