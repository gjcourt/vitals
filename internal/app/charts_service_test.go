package app_test

import (
	"context"
	"testing"

	"biometrics/internal/app"
	"biometrics/internal/domain"
)

func TestGetDaily_BadUnit(t *testing.T) {
	svc := app.NewChartsService(&mockWeightRepo{}, &mockWaterRepo{})
	_, err := svc.GetDaily(context.Background(), 1, 7, "stones")
	if err == nil {
		t.Fatal("expected error for bad unit")
	}
}

func TestGetDaily_Success(t *testing.T) {
	wr := &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) {
			return &domain.WeightEntry{ID: 1, Value: 80, Unit: "kg"}, nil
		},
	}
	wa := &mockWaterRepo{
		totalFn: func(_ context.Context, _ int64, _ string) (float64, error) { return 2.5, nil },
	}

	svc := app.NewChartsService(wr, wa)
	points, err := svc.GetDaily(context.Background(), 1, 3, "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 3 {
		t.Fatalf("expected 3 points, got %d", len(points))
	}
	for _, p := range points {
		if p.WaterLiters != 2.5 {
			t.Errorf("expected waterLiters=2.5, got %v", p.WaterLiters)
		}
		if p.Weight == nil || p.Weight.Value != 80 {
			t.Errorf("expected weight 80, got %v", p.Weight)
		}
	}
}

func TestGetDaily_ConvertUnit(t *testing.T) {
	wr := &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) {
			return &domain.WeightEntry{ID: 1, Value: 100, Unit: "kg"}, nil
		},
	}
	wa := &mockWaterRepo{
		totalFn: func(_ context.Context, _ int64, _ string) (float64, error) { return 0, nil },
	}

	svc := app.NewChartsService(wr, wa)
	points, err := svc.GetDaily(context.Background(), 1, 1, "lb")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Weight == nil || points[0].Weight.Value < 220 || points[0].Weight.Value > 221 {
		t.Errorf("expected ~220.46 lb, got %v", points[0].Weight)
	}
}

func TestGetDaily_ClampsTo366(t *testing.T) {
	wr := &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) {
			return nil, nil
		},
	}
	wa := &mockWaterRepo{
		totalFn: func(_ context.Context, _ int64, _ string) (float64, error) { return 0, nil },
	}

	svc := app.NewChartsService(wr, wa)
	points, err := svc.GetDaily(context.Background(), 1, 500, "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 366 {
		t.Fatalf("expected 366 points (clamped), got %d", len(points))
	}
}

func TestGetDaily_NoWeight(t *testing.T) {
	wr := &mockWeightRepo{
		latestFn: func(_ context.Context, _ int64, _ string) (*domain.WeightEntry, error) {
			return nil, nil
		},
	}
	wa := &mockWaterRepo{
		totalFn: func(_ context.Context, _ int64, _ string) (float64, error) { return 1.0, nil },
	}

	svc := app.NewChartsService(wr, wa)
	points, err := svc.GetDaily(context.Background(), 1, 1, "kg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(points))
	}
	if points[0].Weight != nil {
		t.Errorf("expected nil weight, got %v", points[0].Weight)
	}
	if points[0].WaterLiters != 1.0 {
		t.Errorf("expected waterLiters=1.0, got %v", points[0].WaterLiters)
	}
}
