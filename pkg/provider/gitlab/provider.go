package gitlab

import (
	"context"
	"os"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	lab "github.com/xanzy/go-gitlab"
)

const (
	pageSize = 100
)

// ListAuthors is a GitHub commit provider.
func ListAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	if err := q.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid query")
	}

	log.WithFields(log.Fields{
		"owner":  q.Owner,
		"repo":   q.Repo,
		"commit": q.Commit,
	}).Debug("list authors")

	client, err := lab.NewClient(os.Getenv("GITLAB_TOKEN"))
	if err != nil {
		return nil, errors.Wrap(err, "error creating client")
	}

	list := make(map[string]*report.Author)
	var totalCommitCounter int64
	pageCounter := 1

	for {
		opts := &lab.ListCommitsOptions{
			All:       lab.Bool(true),
			WithStats: lab.Bool(true),
			ListOptions: lab.ListOptions{
				Page:    pageCounter,
				PerPage: pageSize,
			},
		}

		p, r, err := client.Commits.ListCommits(q.Repo, opts, lab.WithContext(ctx))
		if err != nil {
			return nil, errors.Wrapf(err, "error listing commits for %s", q.Repo)
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

		// loop through commits
		for _, c := range p {
			if _, ok := list[c.AuthorEmail]; !ok {
				a := report.MakeAuthor(c.AuthorEmail)
				a.Reputation = calculateReputation(c.Stats)
				a.Context["name"] = c.AuthorName

				list[c.AuthorEmail] = a
			}

			list[c.AuthorEmail].Stats.Commits++
			totalCommitCounter++
		}

		// check if we are done
		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	authors := make([]*report.Author, 0)
	for _, a := range list {
		authors = append(authors, a)
	}

	r := &report.Report{
		Repo:              q.Repo,
		AtCommit:          q.Commit,
		GeneratedOn:       time.Now().UTC(),
		TotalCommits:      totalCommitCounter,
		TotalContributors: int64(len(authors)),
		Contributors:      authors,
	}

	return r, nil
}
