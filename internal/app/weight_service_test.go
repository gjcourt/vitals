package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"biometrics/internal/app"
	"biometrics/internal/domain"
)

type mockWeightRepo struct {
	addFn    func(ctx context.Context, userID int64, v float64, u string, t time.Time) (int64, error)
	deleteFn func(ctx context.Context, userID int64) (bool, error)
	latestFn func(ctx context.Context, userID int64, day string) (*domain.WeightEntry, error)
	listFn   func(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error)
}

func (m *mockWeightRepo) AddWeightEvent(ctx context.Context, userID int64, v float64, u string, t time.Time) (int64, error) {
	if m.addFn != nil {
		return m.addFn(ctx, userID, v, u, t)
	}
	return 0, nil
}

func (m *mockWeightRepo) DeleteLatestWeightEvent(ctx context.Context, userID int64) (bool, error) {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, userID)
	}
	return false, nil
}

func (m *mockWeightRepo) LatestWeightForLocalDay(ctx context.Context, userID int64, day string) (*domain.WeightEntry, error) {
	if m.latestFn != nil {
		return m.latestFn(ctx, userID, day)
	}
	return nil, nil
}

func (m *mockWeightRepo) ListRecentWeightEvents(ctx context.Context, userID int64, limit int) ([]domain.WeightEntry, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, limit)
	}
	return nil, nil
}

func TestRecordWeight_Validation(t *testing.T) {
	svc := app.NewWeightService(&mockWeightRepo{})

	tests := []struct {
		name  string
		value float64
		unit  string
	}{
		{"zero value", 0, "kg"},
		{"negative value", -5, "kg"},
		{"bad unit", 80, "stones"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.RecordWeight(context.Background(), 1, tc.value, tc.unit)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRecordWeight_Success(t *testing.T) {
	entry := &domain.WeightEntry{ID: 1, Value: 80, Unit: "kg"}
	repo := &mockWeightRepo{
		addFn: func(_ context.Context, _ int64, _ float64, _ string, _ time.Time) (int64, error) {
			return 1, nil
		},
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) {
			return entry, nil
		},
	}
	svc := app.NewWeightService(repo)
	got, today, err := svc.RecordWeight(context.Background(), 1, 80, "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if today == "" {
		t.Fatal("expected today string")
	}
	if got == nil || got.ID != 1 {
		t.Fatalf("unexpected entry: %v", got)
	}
}

func TestRecordWeight_RepoError(t *testing.T) {
	repo := &mockWeightRepo{
		addFn: func(_ context.Context, _ int64, _ float64, _ string, _ time.Time) (int64, error) {
			return 0, errors.New("db down")
		},
	}
	svc := app.NewWeightService(repo)
	_, _, err := svc.RecordWeight(context.Background(), 1, 80, "kg")
	if err == nil {
		t.Fatal("expected error from repo")
	}
}

func TestGetTodayWeight(t *testing.T) {
	entry := &domain.WeightEntry{ID: 5, Value: 75, Unit: "kg"}
	repo := &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, day string) (*domain.WeightEntry, error) {
			if day != "2026-01-15" {
				t.Fatalf("unexpected day: %s", day)
			}
			return entry, nil
		},
	}
	svc := app.NewWeightService(repo)
	got, err := svc.GetTodayWeight(context.Background(), 1, "2026-01-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ID != 5 {
		t.Fatalf("unexpected entry: %v", got)
	}
}

func TestUndoLastWeight(t *testing.T) {
	repo := &mockWeightRepo{
		deleteFn: func(_ context.Context, _ int64) (bool, error) { return true, nil },
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) { return nil, nil },
	}
	svc := app.NewWeightService(repo)
	deleted, _, _, err := svc.UndoLast(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Fatal("expected deleted=true")
	}
}

func TestListRecentWeight_Error(t *testing.T) {
	repo := &mockWeightRepo{
		listFn: func(_ context.Context, _ int64, _ int) ([]domain.WeightEntry, error) {
			return nil, errors.New("db down")
		},
	}
	svc := app.NewWeightService(repo)
	_, err := svc.ListRecent(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}
