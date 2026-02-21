package reputer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNilOptions(t *testing.T) {
	var o *ListCommitAuthorsOptions
	err := o.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "options must be populated")
}

func TestValidateEmptyRepo(t *testing.T) {
	o := &ListCommitAuthorsOptions{}
	err := o.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo must be specified")
}

func TestValidateValid(t *testing.T) {
	o := &ListCommitAuthorsOptions{Repo: "github.com/o/r"}
	require.NoError(t, o.Validate())
}

func TestOptionsString(t *testing.T) {
	o := &ListCommitAuthorsOptions{
		Repo:   "github.com/o/r",
		Commit: "abc",
		Stats:  true,
		File:   "out.json",
	}
	s := o.String()
	assert.Contains(t, s, "github.com/o/r")
	assert.Contains(t, s, "abc")
	assert.Contains(t, s, "true")
	assert.Contains(t, s, "out.json")
}
