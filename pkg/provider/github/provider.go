package github

import (
	"context"
	"fmt"
	"sync"
	"time"

	hub "github.com/google/go-github/v52/github"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	pageSize = 100
)

// ListAuthors is a GitHub commit provider.
func ListAuthors(ctx context.Context, owner, repo, commit string) (*report.Report, error) {
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
	var commitCounter int64

	for {
		opts := &hub.CommitsListOptions{
			SHA: commit,
			ListOptions: hub.ListOptions{
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
			if _, ok := list[c.GetAuthor().GetLogin()]; !ok {
				list[c.GetAuthor().GetLogin()] = report.MakeAuthor(c.GetAuthor().GetLogin())
			}

			// add commit to the list
			list[c.GetAuthor().GetLogin()].Commits++
			if !*c.GetCommit().GetVerification().Verified {
				list[c.GetAuthor().GetLogin()].UnverifiedCommits++
			}

			commitCounter++
		}

		// check if we are done
		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	// convert map to slice
	authors := make([]*report.Author, 0)
	var wg sync.WaitGroup

	for _, a := range list {
		wg.Add(1)
		go func(a *report.Author) {
			defer wg.Done()
			loadAuthor(ctx, client, a)
			authors = append(authors, a)
		}(a)
	}

	wg.Wait()

	r := &report.Report{
		Repo:              fmt.Sprintf("github.com/%s/%s", owner, repo),
		AtCommit:          commit,
		GeneratedOn:       time.Now().UTC(),
		TotalCommits:      commitCounter,
		TotalContributors: int64(len(authors)),
		Contributors:      authors,
	}

	return r, nil
}

func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author) {
	if client == nil {
		log.Error("client must be specified")
		return
	}
	if a == nil {
		log.Error("author must be specified")
		return
	}

	u, r, err := client.Users.Get(ctx, a.Username)
	if err != nil {
		log.Errorf("error getting user %s: %v", a.Username, err)
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

	calculateReputation(a)
}
