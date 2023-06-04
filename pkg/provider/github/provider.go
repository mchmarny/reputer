package github

import (
	"context"

	api "github.com/google/go-github/v52/github"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	pageSize = 1000
)

// ListAuthors is a GitHub commit provider.
func ListAuthors(ctx context.Context, owner, repo, commit string) ([]*report.Author, error) {
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo must be specified")
	}

	list := make(map[string]*report.Author)
	pageCounter := 1

	for {

		opts := &api.CommitsListOptions{
			SHA: commit,
			ListOptions: api.ListOptions{
				Page:    pageCounter,
				PerPage: pageSize,
			},
		}

		p, r, err := getClient().Repositories.ListCommits(ctx, owner, repo, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing commits for %s/%s", owner, repo)
		}

		log.Debug().
			Str("owner", owner).
			Str("repo", repo).
			Int("opt_page", opts.Page).
			Int("opt_size", opts.PerPage).
			Str("opt_token", opts.SHA).
			Int("page_next", r.NextPage).
			Int("page_last", r.LastPage).
			Str("status", r.Status).
			Int("status_code", r.StatusCode).
			Int("rate_limit", r.Rate.Limit).
			Int("rate_remaining", r.Rate.Remaining).
			Msg("list commits response")

		// loop through commits
		for _, c := range p {
			if c.Author == nil {
				continue
			}

			// check if author already in the list
			if _, ok := list[c.Author.GetLogin()]; !ok {
				list[c.Author.GetLogin()] = &report.Author{
					Login:   c.Author.GetLogin(),
					Name:    c.Author.Name,
					Email:   c.Author.Email,
					Company: c.Author.Company,
					Commits: make([]string, 0),
				}
			}

			// add commit to the list
			list[c.Author.GetLogin()].Commits = append(list[c.Author.GetLogin()].Commits, c.GetSHA())
		}

		// check if we are done
		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	// convert map to slice
	authors := make([]*report.Author, len(list))
	i := 0
	for _, a := range list {
		authors[i] = a
		i++
	}

	return authors, nil
}
