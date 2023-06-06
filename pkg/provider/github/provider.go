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

			// committer and author could be different
			// that only means that the commit was cherry-picked or re-based (or both)
			log.WithFields(log.Fields{
				"author":         c.GetAuthor().GetLogin(),
				"author_date":    c.GetAuthor().GetCreatedAt(),
				"committer":      c.GetCommitter().GetLogin(),
				"committer_date": c.GetCommitter().GetCreatedAt(),
			}).Debug("commit")

			// check if author already in the list
			if _, ok := list[c.GetAuthor().GetLogin()]; !ok {
				list[c.GetAuthor().GetLogin()] = report.MakeAuthor(c.GetAuthor().GetLogin())
			}

			// add commit to the list
			list[c.GetAuthor().GetLogin()].Commits++
			if !*c.GetCommit().GetVerification().Verified {
				list[c.GetAuthor().GetLogin()].UnverifiedCommits++
			}
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

func loadAuthor(ctx context.Context, client *api.Client, a *report.Author) error {
	if client == nil {
		return errors.New("client must be specified")
	}
	if a == nil {
		return errors.New("author must be specified")
	}

	u, r, err := client.Users.Get(ctx, a.Username)
	if err != nil {
		return errors.Wrapf(err, "error getting user %s", a.Username)
	}

	log.WithFields(log.Fields{
		"page_next":      r.NextPage,
		"page_last":      r.LastPage,
		"status":         r.StatusCode,
		"rate_limit":     r.Rate.Limit,
		"rate_remaining": r.Rate.Remaining,
	}).Debug("user")

	// required fields
	a.Created = u.CreatedAt.Time
	a.Suspended = u.SuspendedAt != nil
	a.PublicRepos = int64(u.GetPublicRepos())
	a.PrivateRepos = u.GetTotalPrivateRepos()
	a.Followers = int64(u.GetFollowers())
	a.Following = int64(u.GetFollowing())
	a.StrongAuth = u.GetTwoFactorAuthentication()
	a.CommitsVerified = a.UnverifiedCommits == 0

	// optional fields for context
	if u.Name != nil {
		a.Context["name"] = u.GetName()
	}

	if u.Email != nil {
		a.Context["email"] = u.GetEmail()
	}

	if u.Company != nil {
		a.Context["company"] = u.GetCompany()
	}

	if err := calculateReputation(a); err != nil {
		return errors.Wrap(err, "error calculating reputation")
	}

	return nil
}
