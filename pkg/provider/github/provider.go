package github

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	hub "github.com/google/go-github/v72/github"
	"github.com/mchmarny/reputer/pkg/report"
	"golang.org/x/sync/errgroup"
)

const (
	pageSize       = 100
	hoursInDay     = 24
	maxConcurrency = 10
)

// ListAuthors is a GitHub commit provider.
func ListAuthors(ctx context.Context, q report.Query) (*report.Report, error) {
	slog.Debug("list authors",
		"owner", q.Owner,
		"repo", q.Repo,
		"commit", q.Commit,
		"stats", q.Stats)

	client, err := getClient()
	if err != nil {
		return nil, fmt.Errorf("error creating GitHub client: %w", err)
	}

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
			return nil, fmt.Errorf("error listing commits for %s/%s: %w", q.Owner, q.Name, err)
		}
		waitForRateLimit(r)

		slog.Debug("commit list",
			"page_num", opts.Page,
			"page_size", opts.PerPage,
			"page_next", r.NextPage,
			"page_last", r.LastPage,
			"status", r.StatusCode,
			"rate_limit", r.Rate.Limit,
			"rate_remaining", r.Rate.Remaining)

		for _, c := range p {
			if c.Author == nil {
				continue
			}

			login := c.GetAuthor().GetLogin()
			if _, ok := list[login]; !ok {
				list[login] = report.MakeAuthor(login)
			}

			list[login].Stats.Commits++
			if v := c.GetCommit().GetVerification(); v == nil || v.Verified == nil || !*v.Verified {
				list[login].Stats.UnverifiedCommits++
			}

			// Track most recent commit date per author (commits arrive newest-first).
			if list[login].Stats.LastCommitDays == 0 {
				if cd := c.GetCommit().GetCommitter().GetDate(); !cd.IsZero() {
					days := int64(math.Ceil(time.Now().UTC().Sub(cd.Time).Hours() / hoursInDay))
					if days < 0 {
						days = 0
					}
					list[login].Stats.LastCommitDays = days
				}
			}

			totalCommitCounter++
		}

		if len(p) < pageSize {
			break
		}

		pageCounter++
	}

	rpt := &report.Report{
		Repo:         q.Repo,
		AtCommit:     q.Commit,
		GeneratedOn:  time.Now().UTC(),
		TotalCommits: totalCommitCounter,
	}

	rpt.Meta = &report.Meta{
		ModelVersion: report.ModelVersion,
		Categories: []report.CategoryWeight{
			{Name: "code_provenance", Weight: CategoryProvenanceWeight},
			{Name: "identity", Weight: CategoryIdentityWeight},
			{Name: "engagement", Weight: CategoryEngagementWeight},
			{Name: "community", Weight: CategoryCommunityWeight},
		},
	}

	totalContributors := len(list)

	var mu sync.Mutex
	authors := make([]*report.Author, 0, len(list))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxConcurrency)

	for _, a := range list {
		g.Go(func() error {
			if err := loadAuthor(gctx, client, a, q.Stats, totalCommitCounter, totalContributors, q.Owner); err != nil {
				return err
			}
			mu.Lock()
			authors = append(authors, a)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("error loading authors: %w", err)
	}

	rpt.TotalContributors = int64(totalContributors)
	rpt.Contributors = authors

	return rpt, nil
}

// loadAuthor loads the author details.
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int, owner string) error {
	if client == nil {
		return fmt.Errorf("client must be specified")
	}
	if a == nil {
		return fmt.Errorf("author must be specified")
	}

	u, r, err := client.Users.Get(ctx, a.Username)
	if err != nil {
		return fmt.Errorf("error getting user %s: %w", a.Username, err)
	}
	waitForRateLimit(r)

	slog.Debug("user",
		"page_next", r.NextPage,
		"page_last", r.LastPage,
		"status", r.StatusCode,
		"rate_limit", r.Rate.Limit,
		"rate_remaining", r.Rate.Remaining)

	a.Stats.Suspended = u.SuspendedAt != nil
	a.Stats.StrongAuth = u.GetTwoFactorAuthentication()
	a.Stats.CommitsVerified = a.Stats.UnverifiedCommits == 0 // not used by scoring; exposed in JSON for display
	a.Stats.Followers = int64(u.GetFollowers())
	a.Stats.Following = int64(u.GetFollowing())
	a.Stats.PublicRepos = int64(u.GetPublicRepos())
	a.Stats.PrivateRepos = u.GetTotalPrivateRepos()

	dc := u.GetCreatedAt().Time
	a.Stats.AgeDays = int64(math.Ceil(time.Now().UTC().Sub(dc).Hours() / hoursInDay))
	a.Context.Created = dc.Format(time.RFC3339)

	if u.Name != nil {
		a.Context.Name = u.GetName()
	}

	if u.Email != nil {
		a.Context.Email = u.GetEmail()
	}

	if u.Company != nil {
		a.Context.Company = u.GetCompany()
	}

	// Org membership check -- graceful degradation on error.
	isMember, orgResp, memberErr := client.Organizations.IsMember(ctx, owner, a.Username)
	waitForRateLimit(orgResp)
	if memberErr != nil {
		slog.Debug(fmt.Sprintf("org membership check [%s/%s]: %v", owner, a.Username, memberErr))
	} else {
		a.Stats.OrgMember = isMember
	}

	calculateReputation(a, totalCommits, totalContributors)

	if !stats {
		a.Stats = nil
		a.Context = nil
	}

	return nil
}
