package domain

import (
	"context"
	"time"
)

// WeightEntry represents a single weight measurement.
type WeightEntry struct {
	ID        int64     `json:"id"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}

// WeightRepository is the port for weight persistence.
type WeightRepository interface {
	AddWeightEvent(ctx context.Context, value float64, unit string, createdAt time.Time) (int64, error)
	DeleteLatestWeightEvent(ctx context.Context) (bool, error)
	LatestWeightForLocalDay(ctx context.Context, localDay string) (*WeightEntry, error)
	ListRecentWeightEvents(ctx context.Context, limit int) ([]WeightEntry, error)
}
