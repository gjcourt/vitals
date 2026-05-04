package app

import (
	"context"
	"errors"
	"time"

	"vitals/internal/domain"
	"vitals/internal/ports/inbound"
	"vitals/internal/ports/outbound"
)

// chartsService implements inbound.ChartsService.
type chartsService struct {
	weightRepo outbound.WeightRepository
	waterRepo  outbound.WaterRepository
}

// NewChartsService creates a ChartsService backed by the given repositories.
func NewChartsService(wr outbound.WeightRepository, wa outbound.WaterRepository) inbound.ChartsService {
	return &chartsService{weightRepo: wr, waterRepo: wa}
}

// GetDaily returns per-day chart data for the last days days, with weights
// converted to the requested unit.
func (s *chartsService) GetDaily(ctx context.Context, userID int64, days int, unit string) ([]inbound.DayPoint, error) {
	if unit != "kg" && unit != "lb" {
		return nil, errors.New("unit must be \"kg\" or \"lb\"")
	}
	if days > 366 {
		days = 366
	}

	today := time.Now().In(time.Local)
	points := make([]inbound.DayPoint, 0, days)

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

		var wp *inbound.WeightPoint
		if entry != nil {
			val := entry.Value
			if entry.Unit != unit {
				val = domain.ConvertWeight(val, entry.Unit, unit)
			}
			wp = &inbound.WeightPoint{Value: val, Unit: unit}
		}

		points = append(points, inbound.DayPoint{Day: dayStr, WaterLiters: waterLiters, Weight: wp})
	}
	return points, nil
}
