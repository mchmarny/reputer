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

	client := getClient()

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

		p, r, err := client.Repositories.ListCommits(ctx, owner, repo, opts)
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
		if err := loadAuthor(ctx, client, a); err != nil {
			return nil, errors.Wrap(err, "error loading author")
		}
		authors[i] = a
		i++
	}

	return authors, nil
}

func loadAuthor(ctx context.Context, client *api.Client, author *report.Author) error {
	if client == nil {
		return errors.New("client must be specified")
	}
	if author == nil {
		return errors.New("author must be specified")
	}

	u, _, err := client.Users.Get(ctx, author.Login)
	if err != nil {
		return errors.Wrapf(err, "error getting user %s", author.Login)
	}

	author.Name = u.Name
	author.Email = u.Email
	author.Company = u.Company
	author.Repos = u.PublicRepos
	author.Gists = u.PublicGists
	author.Followers = u.Followers
	author.Following = u.Following
	author.Created = u.CreatedAt.Time
	author.Suspended = u.SuspendedAt != nil
	author.TwoFactorAuth = u.TwoFactorAuthentication

	return nil
}
