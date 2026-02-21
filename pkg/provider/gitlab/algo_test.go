package gitlab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	lab "github.com/xanzy/go-gitlab"
)

func TestCalculateReputationNilStats(t *testing.T) {
	assert.Equal(t, float64(0), calculateReputation(nil))
}

func TestCalculateReputationWithDeletions(t *testing.T) {
	s := &lab.CommitStats{Deletions: 5, Additions: 10}
	assert.Equal(t, 0.1, calculateReputation(s))
}

func TestCalculateReputationNoDeletions(t *testing.T) {
	s := &lab.CommitStats{Additions: 10}
	assert.Equal(t, float64(0), calculateReputation(s))
}
