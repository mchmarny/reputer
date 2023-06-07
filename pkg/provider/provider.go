package provider

import (
	"context"
	"strings"
	"time"

	"github.com/mchmarny/reputer/pkg/provider/github"
	"github.com/mchmarny/reputer/pkg/provider/gitlab"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	repoNameParts = 3
)

var (
	providers = map[string]CommitProvider{
		"github.com": github.ListAuthors,
		"gitlab.com": gitlab.ListAuthors,
	}
)

// CommitProvider is a function that returns a list of authors for the given repo and commit.
type CommitProvider func(ctx context.Context, owner, repo, commit string) (*report.Report, error)

// GetAuthors returns a report of authors for the given repo and commit.
func GetAuthors(ctx context.Context, repo, commit string) (*report.Report, error) {
	if repo == "" {
		return nil, errors.New("repo must be specified")
	}

	start := time.Now()

	ri, err := parseRepoInfo(repo)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing repo info for %s", repo)
	}

	p, ok := providers[ri.kind]
	if !ok {
		return nil, errors.Errorf("unsupported git provider: %s", ri.kind)
	}

	r, err := p(ctx, ri.owner, ri.name, commit)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing authors for %s/%s", ri.owner, ri.name)
	}

	log.Debugf("listed %d commits from %d authors in %s", r.TotalCommits, r.TotalContributors, time.Since(start))

	return r, nil
}

func parseRepoInfo(s string) (*repoInfo, error) {
	// remove https:// if present
	s = strings.TrimPrefix(s, "https://")

	parts := strings.Split(s, "/")
	if len(parts) != repoNameParts {
		return nil, errors.Errorf("invalid format: %s", s)
	}

	return &repoInfo{
		kind:  parts[0],
		owner: parts[1],
		name:  parts[2],
	}, nil
}

type repoInfo struct {
	kind  string
	owner string
	name  string
}
