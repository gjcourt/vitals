package app

import (
	"context"
	"errors"
	"time"

	"biometrics/internal/domain"
)

// ChartsService encapsulates chart data retrieval use cases.
type ChartsService struct {
	weightRepo domain.WeightRepository
	waterRepo  domain.WaterRepository
}

// NewChartsService creates a ChartsService backed by the given repositories.
func NewChartsService(wr domain.WeightRepository, wa domain.WaterRepository) *ChartsService {
	return &ChartsService{weightRepo: wr, waterRepo: wa}
}

// DayPoint is a single data point returned by GetDaily.
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

// GetDaily returns per-day chart data for the last days days, with weights
// converted to the requested unit.
func (s *ChartsService) GetDaily(ctx context.Context, userID int64, days int, unit string) ([]DayPoint, error) {
	if unit != "kg" && unit != "lb" {
		return nil, errors.New("unit must be \"kg\" or \"lb\"")
	}
	if days > 366 {
		days = 366
	}

	today := time.Now().In(time.Local)
	points := make([]DayPoint, 0, days)

	for i := days - 1; i >= 0; i-- {
		d := today.AddDate(0, 0, -i)
		dayStr := d.Format("2006-01-02")

		waterLiters, err := s.waterRepo.WaterTotalForLocalDay(ctx, userID, dayStr)
		if err != nil {
			return nil, err
		}

		entry, err := s.weightRepo.LatestWeightForLocalDay(ctx, userID, dayStr)
		if err != nil {
			return nil, err
		}

		var wp *WeightPoint
		if entry != nil {
			val := entry.Value
			if entry.Unit != unit {
				val = domain.ConvertWeight(val, entry.Unit, unit)
			}
			wp = &WeightPoint{Value: val, Unit: unit}
		}

		points = append(points, DayPoint{Day: dayStr, WaterLiters: waterLiters, Weight: wp})
	}
	return points, nil
}
