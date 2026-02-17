package app

import (
	"context"
	"errors"
	"time"

	"biometrics/internal/domain"
)

// WeightService encapsulates weight-tracking use cases.
type WeightService struct {
	repo domain.WeightRepository
}

// NewWeightService creates a WeightService backed by the given repository.
func NewWeightService(repo domain.WeightRepository) *WeightService {
	return &WeightService{repo: repo}
}

// GetTodayWeight returns the latest weight entry for the given local day.
func (s *WeightService) GetTodayWeight(ctx context.Context, userID int64, today string) (*domain.WeightEntry, error) {
	return s.repo.LatestWeightForLocalDay(ctx, userID, today)
}

// RecordWeight validates and stores a new weight measurement, returning the
// latest entry for today after the insert.
func (s *WeightService) RecordWeight(ctx context.Context, userID int64, value float64, unit string) (*domain.WeightEntry, string, error) {
	if value <= 0 {
		return nil, "", errors.New("value must be > 0")
	}
	if unit != "kg" && unit != "lb" {
		return nil, "", errors.New("unit must be \"kg\" or \"lb\"")
	}
	now := time.Now()
	today := now.In(time.Local).Format("2006-01-02")
	if _, err := s.repo.AddWeightEvent(ctx, userID, value, unit, now); err != nil {
		return nil, today, err
	}
	entry, err := s.repo.LatestWeightForLocalDay(ctx, userID, today)
	return entry, today, err
}

// ListRecent returns the most recent weight events up to limit.
func (s *WeightService) ListRecent(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	return s.repo.ListRecentWeightEvents(ctx, userID, limit)
}

// UndoLast deletes the most recent weight event and returns the new latest
// entry for today.
func (s *WeightService) UndoLast(ctx context.Context, userID int64) (bool, *domain.WeightEntry, string, error) {
	today := time.Now().In(time.Local).Format("2006-01-02")
	deleted, err := s.repo.DeleteLatestWeightEvent(ctx, userID)
	if err != nil {
		return false, nil, today, err
	}
	entry, _ := s.repo.LatestWeightForLocalDay(ctx, userID, today)
	return deleted, entry, today, nil
}
