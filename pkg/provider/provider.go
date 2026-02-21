package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/mchmarny/reputer/pkg/provider/github"
	"github.com/mchmarny/reputer/pkg/provider/gitlab"
	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

var providers = map[string]CommitProvider{
	"github.com": github.ListAuthors,
	"gitlab.com": gitlab.ListAuthors,
}

// CommitProvider is a function that returns a list of authors for the given repo and commit.
type CommitProvider func(ctx context.Context, q report.Query) (*report.Report, error)

// GetAuthors returns a report of authors for the given repo and commit.
func GetAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	if err := q.Validate(); err != nil {
		return nil, fmt.Errorf("invalid query: %w", err)
	}

	start := time.Now()

	p, ok := providers[q.Kind]
	if !ok {
		return nil, fmt.Errorf("unsupported git provider: %s", q.Kind)
	}

	r, err := p(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("error listing authors with %v: %w", q, err)
	}

	r.SortAuthors()

	log.Debugf("listed %d commits from %d authors in %s",
		r.TotalCommits, r.TotalContributors, time.Since(start))

	return r, nil
}
