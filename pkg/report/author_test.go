package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeAuthor(t *testing.T) {
	a := MakeAuthor("testuser")
	require.NotNil(t, a)
	assert.Equal(t, "testuser", a.Username)
	assert.NotNil(t, a.Stats)
	assert.NotNil(t, a.Context)
	assert.Equal(t, float64(0), a.Reputation)
}

func TestAuthorStringNil(t *testing.T) {
	var a *Author
	assert.Equal(t, "<nil>", a.String())
}

func TestAuthorString(t *testing.T) {
	a := MakeAuthor("user1")
	s := a.String()
	assert.Contains(t, s, "user1")
}
