package domain

import "time"

// WeightEntry represents a single weight measurement.
type WeightEntry struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Day       string    `json:"day"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	CreatedAt time.Time `json:"createdAt"`
}
