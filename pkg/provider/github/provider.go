package github

import (
	"context"

	api "github.com/google/go-github/v52/github"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	pageSize = 100
)

// ListAuthors is a GitHub commit provider.
func ListAuthors(ctx context.Context, owner, repo, commit string) ([]*report.Author, error) {
	if owner == "" || repo == "" {
		return nil, errors.New("owner and repo must be specified")
	}

	log.WithFields(log.Fields{
		"owner":  owner,
		"repo":   repo,
		"commit": commit,
	}).Debug("list authors")

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

		log.WithFields(log.Fields{
			"page_num":       opts.Page,
			"size_size":      opts.PerPage,
			"page_next":      r.NextPage,
			"page_last":      r.LastPage,
			"status":         r.StatusCode,
			"rate_limit":     r.Rate.Limit,
			"rate_remaining": r.Rate.Remaining,
		}).Debug("commit list")

		// loop through commits
		for _, c := range p {
			if c.Author == nil {
				continue
			}

			// check if author already in the list
			if _, ok := list[c.Author.GetLogin()]; !ok {
				list[c.Author.GetLogin()] = &report.Author{
					Username: c.Author.GetLogin(),
					Commits:  make([]*report.Commit, 0),
					Context:  make(map[string]interface{}),
				}
			}

			// add commit to the list
			list[c.Author.GetLogin()].Commits = append(list[c.Author.GetLogin()].Commits, &report.Commit{
				SHA:      c.GetSHA(),
				Verified: *c.GetCommit().GetVerification().Verified,
			})
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

	u, r, err := client.Users.Get(ctx, author.Username)
	if err != nil {
		return errors.Wrapf(err, "error getting user %s", author.Username)
	}

	log.WithFields(log.Fields{
		"page_next":      r.NextPage,
		"page_last":      r.LastPage,
		"status":         r.StatusCode,
		"rate_limit":     r.Rate.Limit,
		"rate_remaining": r.Rate.Remaining,
	}).Debug("user get")

	// required fields
	author.Created = u.CreatedAt.Time
	author.Suspended = u.SuspendedAt != nil
	author.PublicRepos = int64(u.GetPublicRepos())
	author.PrivateRepos = u.GetTotalPrivateRepos()
	author.Followers = int64(u.GetFollowers())
	author.Following = int64(u.GetFollowing())
	author.StrongAuth = u.GetTwoFactorAuthentication()

	// optional fields for context
	if u.Name != nil {
		author.Context["name"] = u.GetName()
	}

	if u.Email != nil {
		author.Context["email"] = u.GetEmail()
	}

	if u.Company != nil {
		author.Context["company"] = u.GetCompany()
	}

	if err := calculateReputation(author); err != nil {
		return errors.Wrap(err, "error calculating reputation")
	}

	return nil
}
