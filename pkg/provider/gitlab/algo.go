package gitlab

import (
	"github.com/mchmarny/reputer/pkg/report"
	lab "github.com/xanzy/go-gitlab"
)

const (
	// weight should add up to 1
	deletionWeight = 0.1
)

func calculateReputation(s *lab.CommitStats) float64 {
	if s == nil {
		return 0
	}

	var rep float64

	// TODO: add more factors
	if s.Deletions > 0 {
		rep += deletionWeight
	}

	return report.ToFixed(rep, 2)
}
