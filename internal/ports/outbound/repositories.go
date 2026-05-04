// Package outbound holds the driven-port interfaces for persistent storage.
package outbound

import (
	"context"
	"time"

	"vitals/internal/domain"
)

// UserRepository is the driven port for user persistence.
type UserRepository interface {
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByID(ctx context.Context, id int64) (*domain.User, error)
	Create(ctx context.Context, username, passwordHash string) (*domain.User, error)
	Count(ctx context.Context) (int, error)
}

// SessionRepository is the driven port for session persistence.
type SessionRepository interface {
	Create(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error
	GetByToken(ctx context.Context, token string) (*domain.Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}

// WaterRepository is the driven port for water-event persistence.
type WaterRepository interface {
	AddWaterEvent(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error)
	DeleteWaterEvent(ctx context.Context, userID int64, id int64) error
	ListRecentWaterEvents(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error)
	WaterTotalForLocalDay(ctx context.Context, userID int64, localDay string) (float64, error)
}

// WeightRepository is the driven port for weight-event persistence.
type WeightRepository interface {
	AddWeightEvent(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error)
	DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error)
	LatestWeightForLocalDay(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error)
	ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error)
}
