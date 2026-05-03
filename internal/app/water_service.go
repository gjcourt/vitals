package app

import (
	"context"
	"errors"
	"time"

	"vitals/internal/domain"
	"vitals/internal/ports/inbound"
	"vitals/internal/ports/outbound"
)

// waterService implements inbound.WaterService.
type waterService struct {
	repo outbound.WaterRepository
}

// NewWaterService creates a WaterService backed by the given repository.
func NewWaterService(repo outbound.WaterRepository) inbound.WaterService {
	return &waterService{repo: repo}
}

// GetTodayTotal returns the total water intake in liters for the given local day.
func (s *waterService) GetTodayTotal(ctx context.Context, userID int64, today string) (float64, error) {
	return s.repo.WaterTotalForLocalDay(ctx, userID, today)
}

// RecordEvent validates and stores a water intake event.
func (s *waterService) RecordEvent(ctx context.Context, userID int64, deltaLiters float64) (int64, error) {
	if deltaLiters == 0 || deltaLiters < -10 || deltaLiters > 10 {
		return 0, errors.New("deltaLiters must be non-zero and within [-10, 10]")
	}
	return s.repo.AddWaterEvent(ctx, userID, deltaLiters, time.Now())
}

// ListRecent returns the most recent water events up to limit.
func (s *waterService) ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WaterEvent, error) {
	return s.repo.ListRecentWaterEvents(ctx, userID, limit)
}

// UndoLast deletes the most recent water event.
func (s *waterService) UndoLast(ctx context.Context, userID int64) (bool, int64, error) {
	items, err := s.repo.ListRecentWaterEvents(ctx, userID, 1)
	if err != nil {
		return false, 0, err
	}
	if len(items) == 0 {
		return false, 0, nil
	}
	if err := s.repo.DeleteWaterEvent(ctx, userID, items[0].ID); err != nil {
		return false, 0, err
	}
	return true, items[0].ID, nil
}
