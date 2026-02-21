package reporter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mchmarny/reputer/pkg/provider"
	"github.com/mchmarny/reputer/pkg/report"
)

// ListCommitAuthors returns a list of authors for the given repo and commit.
func ListCommitAuthors(ctx context.Context, opt *ListCommitAuthorsOptions) (retErr error) {
	if opt == nil {
		return errors.New("options must be specified")
	}

	if err := opt.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	q, err := report.MakeQuery(opt.Repo, opt.Commit, opt.Stats)
	if err != nil {
		return fmt.Errorf("error creating query for %s: %w", opt, err)
	}

	r, err := provider.GetAuthors(ctx, *q)
	if err != nil {
		return fmt.Errorf("error listing authors for %s: %w", opt, err)
	}

	f := os.Stdout
	if opt.File != "" {
		f, err = os.Create(opt.File)
		if err != nil {
			return fmt.Errorf("error creating file %s: %w", opt.File, err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil && retErr == nil {
				retErr = fmt.Errorf("error closing file %s: %w", opt.File, cerr)
			}
		}()
	}

	if err := json.NewEncoder(f).Encode(r); err != nil {
		return fmt.Errorf("error encoding authors for %s: %w", opt, err)
	}

	return nil
}
