package report

import "math"

// Round rounds a float64 to the nearest integer.
func Round(num float64) int {
	return int(math.Round(num))
}

// ToFixed truncates a float64 to the given precision.
func ToFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(Round(num*output)) / output
}
