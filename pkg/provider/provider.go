package provider

import (
	"context"
	"strings"

	"github.com/mchmarny/reputer/pkg/provider/github"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
)

const (
	repoNameParts = 3
)

var (
	providers = map[string]CommitProvider{
		"github.com": github.ListAuthors,
	}
)

// CommitProvider is a function that returns a list of authors for the given repo and commit.
type CommitProvider func(ctx context.Context, owner, repo, from, to string) ([]*report.Author, error)

// ListAuthors returns a list of authors for the given repo and commit.
func ListAuthors(ctx context.Context, repo, from, to string) ([]*report.Author, error) {
	if repo == "" {
		return nil, errors.New("repo must be specified")
	}

	parts := strings.Split(repo, "/")
	if len(parts) != repoNameParts {
		return nil, errors.Errorf("invalid repo name: %s", repo)
	}

	gitProvider := parts[0]
	owner := parts[1]
	repoName := parts[2]

	p, ok := providers[gitProvider]
	if !ok {
		return nil, errors.Errorf("unsupported git provider: %s", gitProvider)
	}

	list, err := p(ctx, owner, repoName, from, to)
	if err != nil {
		return nil, errors.Wrapf(err, "error listing authors for %s/%s", owner, repoName)
	}

	return list, nil
}
