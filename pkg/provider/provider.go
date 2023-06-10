package provider

import (
	"context"
	"time"

	"github.com/mchmarny/reputer/pkg/provider/github"
	"github.com/mchmarny/reputer/pkg/provider/gitlab"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

var (
	providers = map[string]CommitProvider{
		"github.com": github.ListAuthors,
		"gitlab.com": gitlab.ListAuthors,
	}
)

// CommitProvider is a function that returns a list of authors for the given repo and commit.
type CommitProvider func(ctx context.Context, q report.Query) (*report.Report, error)

// GetAuthors returns a report of authors for the given repo and commit.
func GetAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	if err := q.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid query")
	}

	start := time.Now()

	p, ok := providers[q.Kind]
	if !ok {
		return nil, errors.Errorf("unsupported git provider: %s", q.Kind)
	}

	r, err := p(ctx, q)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing authors with %v", q)
	}

	r.SortAuthors()

	log.Debugf("listed %d commits from %d authors in %s",
		r.TotalCommits, r.TotalContributors, time.Since(start))

	return r, nil
}
