package domain

import "time"

// WaterEvent represents a single water intake/decrement event.
type WaterEvent struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	DeltaLiters float64   `json:"deltaLiters"`
	CreatedAt   time.Time `json:"createdAt"`
}
