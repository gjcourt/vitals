// Package inbound holds the driving-port interfaces exposed to HTTP adapters.
package inbound

import (
	"context"

	"vitals/internal/domain"
)

// AuthService is the driving port for authentication and session management.
type AuthService interface {
	Login(ctx context.Context, username, password, userAgent, ip string) (string, error)
	Logout(ctx context.Context, token string) error
	ValidateSession(ctx context.Context, token, userAgent string) (*domain.User, error)
	CreateInitialUser(ctx context.Context, username, password string) error
	ValidateForwardAuth(ctx context.Context, remoteUser string) (*domain.User, error)
	LoginWithUser(ctx context.Context, username, userAgent, ip string) (string, error)
}

// WaterService is the driving port for water-tracking use cases.
type WaterService interface {
	GetTodayTotal(ctx context.Context, userID int64, today string) (float64, error)
	RecordEvent(ctx context.Context, userID int64, deltaLiters float64) (int64, error)
	ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error)
	UndoLast(ctx context.Context, userID int64) (bool, int64, error)
}

// DayPoint is a single data point for chart data.
type DayPoint struct {
	Day         string       `json:"day"`
	WaterLiters float64      `json:"waterLiters"`
	Weight      *WeightPoint `json:"weight"`
}

// WeightPoint is the optional weight value within a DayPoint.
type WeightPoint struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// WeightService is the driving port for weight-tracking use cases.
type WeightService interface {
	GetTodayWeight(ctx context.Context, userID int64, today string) (*domain.WeightEntry, error)
	RecordWeight(ctx context.Context, userID int64, value float64, unit string) (*domain.WeightEntry, string, error)
	ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error)
	UndoLast(ctx context.Context, userID int64) (bool, *domain.WeightEntry, string, error)
}

// ChartsService is the driving port for chart data aggregation.
type ChartsService interface {
	GetDaily(ctx context.Context, userID int64, days int, unit string) ([]DayPoint, error)
}
