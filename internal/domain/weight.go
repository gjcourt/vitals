package domain

import (
	"context"
	"time"
)

// WeightEntry represents a single weight measurement.
type WeightEntry struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}

// WeightRepository is the port for weight persistence.
type WeightRepository interface {
	AddWeightEvent(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error)
	DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error)
	LatestWeightForLocalDay(ctx context.Context, userID int64, localDay string) (*WeightEntry, error)
	ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]WeightEntry, error)
}
