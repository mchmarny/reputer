package github

import (
	"context"
	"math"
	"sync"
	"time"

	hub "github.com/google/go-github/v52/github"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	pageSize   = 100
	hoursInDay = 24
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
		"stats":  q.Stats,
	}).Debug("list authors")

	client := getClient()

	list := make(map[string]*report.Author)
	pageCounter := 1
	totalCommitCounter := int64(0)

	for {
		opts := &hub.CommitsListOptions{
			SHA: q.Commit,
			ListOptions: hub.ListOptions{
				Page:    pageCounter,
				PerPage: pageSize,
			},
		}

		p, r, err := client.Repositories.ListCommits(ctx, q.Owner, q.Name, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing commits for %s/%s", q.Owner, q.Name)
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
			list[c.GetAuthor().GetLogin()].Stats.Commits++
			if !*c.GetCommit().GetVerification().Verified {
				list[c.GetAuthor().GetLogin()].Stats.UnverifiedCommits++
			}

			totalCommitCounter++
		}

		// check if we are done
		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	r := &report.Report{
		Repo:         q.Repo,
		AtCommit:     q.Commit,
		GeneratedOn:  time.Now().UTC(),
		TotalCommits: totalCommitCounter,
	}

	// convert map to slice
	authors := make([]*report.Author, 0)
	var wg sync.WaitGroup

	for _, a := range list {
		wg.Add(1)
		go func(a *report.Author) {
			defer wg.Done()
			loadAuthor(ctx, client, a, q.Stats)
			authors = append(authors, a)
		}(a)
	}

	// wait for all to finish
	wg.Wait()

	r.TotalContributors = int64(len(authors))
	r.Contributors = authors

	return r, nil
}

// loadAuthor loads the author details.
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool) {
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

	// optional fields
	a.Stats.Suspended = u.SuspendedAt != nil
	a.Stats.StrongAuth = u.GetTwoFactorAuthentication()
	a.Stats.CommitsVerified = a.Stats.UnverifiedCommits == 0
	a.Stats.Followers = int64(u.GetFollowers())
	a.Stats.Following = int64(u.GetFollowing())
	a.Stats.PublicRepos = int64(u.GetPublicRepos())
	a.Stats.PrivateRepos = u.GetTotalPrivateRepos()

	dc := u.GetCreatedAt().Time
	a.Stats.AgeDays = int64(math.Ceil(time.Now().UTC().Sub(dc).Hours() / hoursInDay))
	a.Context["created"] = dc.Format(time.RFC3339)

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

	// remove stats if not requested
	if !stats {
		a.Stats = nil
		a.Context = nil
	}
}
