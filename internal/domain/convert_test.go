package domain_test

import (
"math"
"testing"

"biometrics/internal/domain"
)

func almostEqual(a, b, epsilon float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestConvertWeight(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		from, to string
		want     float64
	}{
		{"kg to lb", 100.0, "kg", "lb", 220.46226218},
		{"lb to kg", 220.46226218, "lb", "kg", 100.0},
		{"same unit kg", 80.0, "kg", "kg", 80.0},
		{"same unit lb", 180.0, "lb", "lb", 180.0},
		{"unknown units", 50.0, "st", "kg", 50.0},
		{"zero value", 0, "kg", "lb", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
got := domain.ConvertWeight(tc.value, tc.from, tc.to)
if !almostEqual(got, tc.want, 0.001) {
t.Errorf("ConvertWeight(%v, %q, %q) = %v; want %v",
tc.value, tc.from, tc.to, got, tc.want)
}
})
	}
}
