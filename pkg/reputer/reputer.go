package reputer

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mchmarny/reputer/pkg/provider"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
)

// ListCommitAuthors returns a list of authors for the given repo and commit.
func ListCommitAuthors(ctx context.Context, opt *ListCommitAuthorsOptions) error {
	if opt == nil {
		return errors.New("options must be specified")
	}

	if err := opt.Validate(); err != nil {
		return errors.Wrap(err, "invalid options")
	}

	q, err := report.MakeQuery(opt.Repo, opt.Commit, opt.Stats)
	if err != nil {
		return errors.Wrapf(err, "error creating query for %s", opt)
	}

	r, err := provider.GetAuthors(ctx, *q)
	if err != nil {
		return errors.Wrapf(err, "error listing authors for %s", opt)
	}

	f := os.Stdout
	if opt.File != "" {
		f, err = os.Create(opt.File)
		if err != nil {
			return errors.Wrapf(err, "error creating file %s", opt.File)
		}
		defer f.Close()
	}

	if err := json.NewEncoder(f).Encode(r); err != nil {
		return errors.Wrapf(err, "error encoding authors for %s", opt)
	}

	return nil
}
