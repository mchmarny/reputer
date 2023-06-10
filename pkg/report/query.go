package report

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

const (
	repoNameParts = 3
)

// MakeQuery returns a new query for the given repo and commit.
func MakeQuery(repo, commit string, stats bool) (*Query, error) {
	if repo == "" {
		return nil, errors.New("repo must be specified")
	}

	// remove https:// if present
	repo = strings.TrimPrefix(repo, "https://")

	parts := strings.Split(repo, "/")
	if len(parts) != repoNameParts {
		return nil, errors.Errorf("invalid format: %s", repo)
	}

	q := &Query{
		Repo:   repo,
		Commit: commit,
		Stats:  stats,
		Kind:   parts[0],
		Owner:  parts[1],
		Name:   parts[2],
	}

	return q, nil
}

// Query is a query for a repo and commit.
type Query struct {
	// Repo is the repo to query (required).
	Repo string
	// Commit is the commit to query (optional).
	Commit string
	// File is the file to write the report to (optional).
	Stats bool

	// Kind is the kind of repo (e.g. github.com, gitlab.com, etc.).
	// Will be parsed from Repo.
	Kind string

	// Owner is the owner of the repo.
	// Will be parsed from Repo.
	Owner string

	// Name is the name of the repo.
	// Will be parsed from Repo.
	Name string
}

// String returns a string representation of the query.
func (q *Query) String() string {
	if q == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%v", *q)
}

// Validate validates the query.
func (q *Query) Validate() error {
	if q == nil {
		return errors.New("query must be specified")
	}

	if q.Repo == "" {
		return errors.New("repo must be specified")
	}

	if q.Kind == "" {
		return errors.New("kind must be specified")
	}

	if q.Owner == "" {
		return errors.New("owner must be specified")
	}

	if q.Name == "" {
		return errors.New("name must be specified")
	}

	return nil
}
