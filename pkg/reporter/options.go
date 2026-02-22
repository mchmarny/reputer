package reporter

import (
	"errors"
	"fmt"
)

// ListCommitAuthorsOptions configures a reputation report query.
type ListCommitAuthorsOptions struct {
	Repo   string
	Commit string
	Stats  bool
	File   string
	Format string
}

// Validate checks that required fields are populated.
func (l *ListCommitAuthorsOptions) Validate() error {
	if l == nil {
		return errors.New("options must be populated")
	}

	if l.Repo == "" {
		return errors.New("repo must be specified")
	}

	switch l.Format {
	case "", "json", "yaml":
	default:
		return fmt.Errorf("unsupported format: %s (must be json or yaml)", l.Format)
	}

	if l.Format == "" {
		l.Format = "json"
	}

	return nil
}

func (l *ListCommitAuthorsOptions) String() string {
	return fmt.Sprintf("repo: %s, commit: %s, stats: %t, file: %s, format: %s",
		l.Repo, l.Commit, l.Stats, l.File, l.Format)
}
