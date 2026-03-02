package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPRStatsZeroValue(t *testing.T) {
	var s prStats
	assert.Zero(t, s.Merged)
	assert.Zero(t, s.Closed)
}
