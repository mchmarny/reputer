package provider

import (
	"context"
	"testing"

	"github.com/mchmarny/reputer/pkg/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAuthorsUnsupportedProvider(t *testing.T) {
	q := report.Query{
		Repo:  "bitbucket.org/o/r",
		Kind:  "bitbucket.org",
		Owner: "o",
		Name:  "r",
	}

	_, err := GetAuthors(context.Background(), q)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported git provider")
}

func TestGetAuthorsEmptyQuery(t *testing.T) {
	_, err := GetAuthors(context.Background(), report.Query{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid query")
}
