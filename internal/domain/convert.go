package domain

const kgToLb = 2.2046226218

// ConvertWeight converts a weight value between "kg" and "lb".
// Returns v unchanged if from == to or if the units are unrecognised.
func ConvertWeight(v float64, from, to string) float64 {
	if from == to {
		return v
	}
	if from == "kg" && to == "lb" {
		return v * kgToLb
	}
	if from == "lb" && to == "kg" {
		return v / kgToLb
	}
	return v
}
