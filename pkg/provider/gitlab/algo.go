package gitlab

import (
	"math"

	api "github.com/xanzy/go-gitlab"
)

const (
	// weight should add up to 1
	deletionWeight = 0.1
)

func calculateReputation(s *api.CommitStats) float64 {
	if s == nil {
		return 0
	}

	var rep float64

	// TODO: add more factors
	if s.Deletions > 0 {
		rep += deletionWeight
	}

	return toFixed(rep, 2)
}

func round(num float64) int {
	return int(num + math.Copysign(0.5, num))
}

func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(round(num*output)) / output
}
