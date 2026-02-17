package domain

import (
	"context"
	"time"
)

// WaterEvent represents a single water intake/decrement event.
type WaterEvent struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	DeltaLiters float64   `json:"deltaLiters"`
	CreatedAt   time.Time `json:"createdAt"`
}

// WaterRepository is the port for water persistence.
type WaterRepository interface {
	AddWaterEvent(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error)
	DeleteWaterEvent(ctx context.Context, userID int64, id int64) error
	ListRecentWaterEvents(ctx context.Context, userID int64, limit int) ([]WaterEvent, error)
	WaterTotalForLocalDay(ctx context.Context, userID int64, localDay string) (float64, error)
}
