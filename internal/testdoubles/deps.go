package testdoubles

import (
	"context"
	"time"

	"vitals/internal/domain"
	"vitals/internal/ports/outbound"
)

// FakeUserRepository is a function-field fake for outbound.UserRepository.
type FakeUserRepository struct {
	GetByUsernameFn func(ctx context.Context, username string) (*domain.User, error)
	GetByIDFn       func(ctx context.Context, id int64) (*domain.User, error)
	CreateFn        func(ctx context.Context, username, passwordHash string) (*domain.User, error)
	CountFn         func(ctx context.Context) (int, error)
}

var _ outbound.UserRepository = (*FakeUserRepository)(nil)

// GetByUsername delegates to GetByUsernameFn if set, otherwise returns nil.
func (f *FakeUserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	if f.GetByUsernameFn != nil {
		return f.GetByUsernameFn(ctx, username)
	}
	return nil, nil
}

// GetByID delegates to GetByIDFn if set, otherwise returns nil.
func (f *FakeUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	if f.GetByIDFn != nil {
		return f.GetByIDFn(ctx, id)
	}
	return nil, nil
}

// Create delegates to CreateFn if set, otherwise returns a zero-value User.
func (f *FakeUserRepository) Create(ctx context.Context, username, passwordHash string) (*domain.User, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, username, passwordHash)
	}
	return &domain.User{Username: username}, nil
}

// Count delegates to CountFn if set, otherwise returns zero.
func (f *FakeUserRepository) Count(ctx context.Context) (int, error) {
	if f.CountFn != nil {
		return f.CountFn(ctx)
	}
	return 0, nil
}

// FakeSessionRepository is a function-field fake for outbound.SessionRepository.
type FakeSessionRepository struct {
	CreateFn        func(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error
	GetByTokenFn    func(ctx context.Context, token string) (*domain.Session, error)
	DeleteFn        func(ctx context.Context, token string) error
	DeleteExpiredFn func(ctx context.Context) error
}

var _ outbound.SessionRepository = (*FakeSessionRepository)(nil)

// Create delegates to CreateFn if set, otherwise returns nil.
func (f *FakeSessionRepository) Create(ctx context.Context, userID int64, token, userAgent, ip string, expiresAt time.Time) error {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, userID, token, userAgent, ip, expiresAt)
	}
	return nil
}

// GetByToken delegates to GetByTokenFn if set, otherwise returns nil.
func (f *FakeSessionRepository) GetByToken(ctx context.Context, token string) (*domain.Session, error) {
	if f.GetByTokenFn != nil {
		return f.GetByTokenFn(ctx, token)
	}
	return nil, nil
}

// Delete delegates to DeleteFn if set, otherwise returns nil.
func (f *FakeSessionRepository) Delete(ctx context.Context, token string) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, token)
	}
	return nil
}

// DeleteExpired delegates to DeleteExpiredFn if set, otherwise returns nil.
func (f *FakeSessionRepository) DeleteExpired(ctx context.Context) error {
	if f.DeleteExpiredFn != nil {
		return f.DeleteExpiredFn(ctx)
	}
	return nil
}

// FakeWaterRepository is a function-field fake for outbound.WaterRepository.
type FakeWaterRepository struct {
	AddWaterEventFn         func(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error)
	DeleteWaterEventFn      func(ctx context.Context, userID int64, id int64) error
	ListRecentWaterEventsFn func(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error)
	WaterTotalForLocalDayFn func(ctx context.Context, userID int64, localDay string) (float64, error)
}

var _ outbound.WaterRepository = (*FakeWaterRepository)(nil)

// AddWaterEvent delegates to AddWaterEventFn if set, otherwise returns zero.
func (f *FakeWaterRepository) AddWaterEvent(ctx context.Context, userID int64, deltaLiters float64, createdAt time.Time) (int64, error) {
	if f.AddWaterEventFn != nil {
		return f.AddWaterEventFn(ctx, userID, deltaLiters, createdAt)
	}
	return 0, nil
}

// DeleteWaterEvent delegates to DeleteWaterEventFn if set, otherwise returns nil.
func (f *FakeWaterRepository) DeleteWaterEvent(ctx context.Context, userID int64, id int64) error {
	if f.DeleteWaterEventFn != nil {
		return f.DeleteWaterEventFn(ctx, userID, id)
	}
	return nil
}

// ListRecentWaterEvents delegates to ListRecentWaterEventsFn if set, otherwise returns nil.
func (f *FakeWaterRepository) ListRecentWaterEvents(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error) {
	if f.ListRecentWaterEventsFn != nil {
		return f.ListRecentWaterEventsFn(ctx, userID, limit)
	}
	return nil, nil
}

// WaterTotalForLocalDay delegates to WaterTotalForLocalDayFn if set, otherwise returns zero.
func (f *FakeWaterRepository) WaterTotalForLocalDay(ctx context.Context, userID int64, localDay string) (float64, error) {
	if f.WaterTotalForLocalDayFn != nil {
		return f.WaterTotalForLocalDayFn(ctx, userID, localDay)
	}
	return 0, nil
}

// FakeWeightRepository is a function-field fake for outbound.WeightRepository.
type FakeWeightRepository struct {
	AddWeightEventFn          func(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error)
	DeleteLatestWeightEventFn func(ctx context.Context, userID int64) (bool, error)
	LatestWeightForLocalDayFn func(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error)
	ListRecentWeightEventsFn  func(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error)
}

var _ outbound.WeightRepository = (*FakeWeightRepository)(nil)

// AddWeightEvent delegates to AddWeightEventFn if set, otherwise returns zero.
func (f *FakeWeightRepository) AddWeightEvent(ctx context.Context, userID int64, value float64, unit string, createdAt time.Time) (int64, error) {
	if f.AddWeightEventFn != nil {
		return f.AddWeightEventFn(ctx, userID, value, unit, createdAt)
	}
	return 0, nil
}

// DeleteLatestWeightEvent delegates to DeleteLatestWeightEventFn if set, otherwise returns false.
func (f *FakeWeightRepository) DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error) {
	if f.DeleteLatestWeightEventFn != nil {
		return f.DeleteLatestWeightEventFn(ctx, userID)
	}
	return false, nil
}

// LatestWeightForLocalDay delegates to LatestWeightForLocalDayFn if set, otherwise returns nil.
func (f *FakeWeightRepository) LatestWeightForLocalDay(ctx context.Context, userID int64, localDay string) (*domain.WeightEntry, error) {
	if f.LatestWeightForLocalDayFn != nil {
		return f.LatestWeightForLocalDayFn(ctx, userID, localDay)
	}
	return nil, nil
}

// ListRecentWeightEvents delegates to ListRecentWeightEventsFn if set, otherwise returns nil.
func (f *FakeWeightRepository) ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	if f.ListRecentWeightEventsFn != nil {
		return f.ListRecentWeightEventsFn(ctx, userID, limit)
	}
	return nil, nil
}

// ServerDeps aggregates all outbound-port fakes for unit tests.
type ServerDeps struct {
	Users    *FakeUserRepository
	Sessions *FakeSessionRepository
	Water    *FakeWaterRepository
	Weight   *FakeWeightRepository
}

// NewServerDeps returns a ServerDeps with all fakes initialised to safe zero-value defaults.
func NewServerDeps() *ServerDeps {
	return &ServerDeps{
		Users:    &FakeUserRepository{},
		Sessions: &FakeSessionRepository{},
		Water:    &FakeWaterRepository{},
		Weight:   &FakeWeightRepository{},
	}
}
