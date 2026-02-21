package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRound(t *testing.T) {
	assert.Equal(t, 2, Round(1.5))
	assert.Equal(t, 1, Round(1.4))
	assert.Equal(t, -2, Round(-1.5))
}

func TestToFixed(t *testing.T) {
	assert.Equal(t, 1.23, ToFixed(1.234, 2))
	assert.Equal(t, 1.24, ToFixed(1.235, 2))
	assert.Equal(t, 0.0, ToFixed(0.0, 2))
}
