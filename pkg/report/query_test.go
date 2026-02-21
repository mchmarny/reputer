package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeQuery(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		commit  string
		stats   bool
		wantErr bool
		errMsg  string
		check   func(t *testing.T, q *Query)
	}{
		{
			name:   "valid 3-part URI",
			repo:   "github.com/owner/repo",
			commit: "abc123",
			stats:  true,
			check: func(t *testing.T, q *Query) {
				assert.Equal(t, "github.com", q.Kind)
				assert.Equal(t, "owner", q.Owner)
				assert.Equal(t, "repo", q.Name)
				assert.Equal(t, "abc123", q.Commit)
				assert.True(t, q.Stats)
			},
		},
		{
			name:   "strips https prefix",
			repo:   "https://github.com/owner/repo",
			commit: "",
			check: func(t *testing.T, q *Query) {
				assert.Equal(t, "github.com", q.Kind)
				assert.Equal(t, "owner", q.Owner)
				assert.Equal(t, "repo", q.Name)
			},
		},
		{
			name:   "strips http prefix",
			repo:   "http://github.com/owner/repo",
			commit: "",
			check: func(t *testing.T, q *Query) {
				assert.Equal(t, "github.com", q.Kind)
				assert.Equal(t, "owner", q.Owner)
				assert.Equal(t, "repo", q.Name)
			},
		},
		{
			name:    "too few parts",
			repo:    "github.com/owner",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "too many parts",
			repo:    "github.com/owner/repo/extra",
			wantErr: true,
			errMsg:  "invalid format",
		},
		{
			name:    "empty repo",
			repo:    "",
			wantErr: true,
			errMsg:  "repo must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := MakeQuery(tt.repo, tt.commit, tt.stats)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, q)
			if tt.check != nil {
				tt.check(t, q)
			}
		})
	}
}

func TestQueryValidate(t *testing.T) {
	tests := []struct {
		name    string
		query   *Query
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil query",
			query:   nil,
			wantErr: true,
			errMsg:  "query must be specified",
		},
		{
			name:    "empty repo",
			query:   &Query{},
			wantErr: true,
			errMsg:  "repo must be specified",
		},
		{
			name:    "missing kind",
			query:   &Query{Repo: "r"},
			wantErr: true,
			errMsg:  "kind must be specified",
		},
		{
			name:    "missing owner",
			query:   &Query{Repo: "r", Kind: "k"},
			wantErr: true,
			errMsg:  "owner must be specified",
		},
		{
			name:    "missing name",
			query:   &Query{Repo: "r", Kind: "k", Owner: "o"},
			wantErr: true,
			errMsg:  "name must be specified",
		},
		{
			name: "valid",
			query: &Query{
				Repo:  "github.com/o/n",
				Kind:  "github.com",
				Owner: "o",
				Name:  "n",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.query.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestQueryString(t *testing.T) {
	assert.Equal(t, "<nil>", (*Query)(nil).String())

	q := &Query{Repo: "github.com/o/n", Kind: "github.com", Owner: "o", Name: "n"}
	assert.Contains(t, q.String(), "github.com/o/n")
}
