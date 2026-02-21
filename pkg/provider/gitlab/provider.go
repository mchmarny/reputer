package gitlab

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	lab "gitlab.com/gitlab-org/api/client-go"
)

const (
	pageSize    int64 = 100
	httpTimeout       = 30 * time.Second
)

// ListAuthors is a GitLab commit provider.
func ListAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	slog.Warn("GitLab provider is a stub; reputation scores will be incomplete")

	slog.Debug("list authors",
		"owner", q.Owner,
		"repo", q.Repo,
		"commit", q.Commit)

	token := os.Getenv("GITLAB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITLAB_TOKEN environment variable must be set")
	}

	client, err := lab.NewClient(token, lab.WithHTTPClient(&http.Client{
		Timeout: httpTimeout,
	}))
	if err != nil {
		return nil, fmt.Errorf("error creating GitLab client: %w", err)
	}

	list := make(map[string]*report.Author)
	var totalCommitCounter int64
	var pageCounter int64 = 1

	for {
		opts := &lab.ListCommitsOptions{
			All:       lab.Ptr(true),
			WithStats: lab.Ptr(true),
			ListOptions: lab.ListOptions{
				Page:    pageCounter,
				PerPage: pageSize,
			},
		}

		p, r, err := client.Commits.ListCommits(q.Repo, opts, lab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("error listing commits for %s: %w", q.Repo, err)
		}

		slog.Debug("commit list", //nolint:gosec // G706: values are typed response metadata, not user input
			"asked_page", opts.Page,
			"asked_size", opts.PerPage,
			"got_size", r.ItemsPerPage,
			"got_next", r.NextPage,
			"got_current", r.CurrentPage,
			"got_status", r.StatusCode,
			"total_items", r.TotalItems,
			"total_pages", r.TotalPages)

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

		if int64(len(p)) < pageSize {
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
