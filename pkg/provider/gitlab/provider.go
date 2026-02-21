package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
	lab "github.com/xanzy/go-gitlab"
)

const (
	pageSize    = 100
	httpTimeout = 30 * time.Second
)

// ListAuthors is a GitLab commit provider.
func ListAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	log.WithFields(log.Fields{
		"owner":  q.Owner,
		"repo":   q.Repo,
		"commit": q.Commit,
	}).Debug("list authors")

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITLAB_TOKEN environment variable must be set")
	}

	client, err := lab.NewClient(token, lab.WithHTTPClient(&http.Client{ //nolint:staticcheck // TODO: migrate to gitlab.com/gitlab-org/api/client-go
		Timeout: httpTimeout,
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating GitLab client: %w", err)
	}

	list := make(map[string]*report.Author)
	var totalCommitCounter int64
	pageCounter := 1

	for {
		opts := &lab.ListCommitsOptions{
			All:       lab.Bool(true), //nolint:staticcheck // TODO: migrate to Ptr
			WithStats: lab.Bool(true), //nolint:staticcheck // TODO: migrate to Ptr
			ListOptions: lab.ListOptions{
				Page:    pageCounter,
				PerPage: pageSize,
			},
		}

		p, r, err := client.Commits.ListCommits(q.Repo, opts, lab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("error listing commits for %s: %w", q.Repo, err)
		}

		log.WithFields(log.Fields{
			"asked_page":  opts.Page,
			"asked_size":  opts.PerPage,
			"got_size":    r.ItemsPerPage,
			"got_next":    r.NextPage,
			"got_current": r.CurrentPage,
			"got_status":  r.StatusCode,
			"total_items": r.TotalItems,
			"total_pages": r.TotalPages,
		}).Debug("commit list")

		for _, c := range p {
			if _, ok := list[c.AuthorEmail]; !ok {
				a := report.MakeAuthor(c.AuthorEmail)
				a.Reputation = calculateReputation(c.Stats)
				a.Context.Name = c.AuthorName

				list[c.AuthorEmail] = a
			}

			list[c.AuthorEmail].Stats.Commits++
			totalCommitCounter++
		}

		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	authors := make([]*report.Author, 0, len(list))
	for _, a := range list {
		authors = append(authors, a)
	}

	rpt := &report.Report{
		Repo:              q.Repo,
		AtCommit:          q.Commit,
		GeneratedOn:       time.Now().UTC(),
		TotalCommits:      totalCommitCounter,
		TotalContributors: int64(len(authors)),
		Contributors:      authors,
	}

	return rpt, nil
}
