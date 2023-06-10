package reputer

import (
	"fmt"

	"github.com/pkg/errors"
)

type ListCommitAuthorsOptions struct {
	Repo   string
	Commit string
	Stats  bool
	File   string
}

func (l *ListCommitAuthorsOptions) Validate() error {
	if l == nil {
		return errors.New("options must be populated")
	}

	if l.Repo == "" {
		return errors.New("repo must be specified")
	}

	return nil
}

func (l *ListCommitAuthorsOptions) String() string {
	return fmt.Sprintf("repo: %s, commit: %s, stats: %t, file: %s",
		l.Repo, l.Commit, l.Stats, l.File)
}
