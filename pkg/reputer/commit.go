package reputer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mchmarny/reputer/pkg/provider"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
)

type ListCommitAuthorsOptions struct {
	Repo   string
	Commit string
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
	return fmt.Sprintf("repo: %s, commit: %s", l.Repo, l.Commit)
}

// ListCommitAuthors returns a list of authors for the given repo and commit.
func ListCommitAuthors(ctx context.Context, opt *ListCommitAuthorsOptions) error {
	if opt == nil {
		return errors.New("options must be specified")
	}

	if err := opt.Validate(); err != nil {
		return errors.Wrap(err, "invalid options")
	}

	list, err := provider.ListAuthors(ctx, opt.Repo, opt.Commit)
	if err != nil {
		return errors.Wrapf(err, "error listing authors for %s", opt)
	}

	rep := &report.Report{
		Repo:         opt.Repo,
		AtCommit:     opt.Commit,
		GeneratedOn:  time.Now().UTC(),
		Contributors: list,
	}

	f := os.Stdout
	if opt.File != "" {
		f, err = os.Create(opt.File)
		if err != nil {
			return errors.Wrapf(err, "error creating file %s", opt.File)
		}
		defer f.Close()
	}

	if err := json.NewEncoder(f).Encode(rep); err != nil {
		return errors.Wrapf(err, "error encoding authors for %s", opt)
	}

	return nil
}
