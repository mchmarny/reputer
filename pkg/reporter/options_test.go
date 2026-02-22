package reporter

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
	assert.Equal(t, "json", o.Format)
}

func TestValidateFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantFmt string
		wantErr bool
	}{
		{name: "empty defaults to json", format: "", wantFmt: "json"},
		{name: "explicit json", format: "json", wantFmt: "json"},
		{name: "explicit yaml", format: "yaml", wantFmt: "yaml"},
		{name: "unsupported format", format: "xml", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := &ListCommitAuthorsOptions{Repo: "github.com/o/r", Format: tc.format}
			err := o.Validate()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported format")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantFmt, o.Format)
			}
		})
	}
}

func TestOptionsString(t *testing.T) {
	o := &ListCommitAuthorsOptions{
		Repo:   "github.com/o/r",
		Commit: "abc",
		Stats:  true,
		File:   "out.json",
		Format: "yaml",
	}
	s := o.String()
	assert.Contains(t, s, "github.com/o/r")
	assert.Contains(t, s, "abc")
	assert.Contains(t, s, "true")
	assert.Contains(t, s, "out.json")
	assert.Contains(t, s, "yaml")
}
