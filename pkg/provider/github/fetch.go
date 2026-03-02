package github

import (
	"context"
	"fmt"
	"log/slog"

	hub "github.com/google/go-github/v72/github"
)

// prStats holds merged and closed-without-merge PR counts.
type prStats struct {
	Merged int64
	Closed int64
}

// fetchPRStats returns global PR merge/close counts for a user.
// Uses GitHub search API: 2 calls per user.
func fetchPRStats(ctx context.Context, client *hub.Client, username string) prStats {
	var stats prStats

	mergedResult, mergedResp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr is:merged", username),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search merged PRs for %s: %v", username, err))
		return stats
	}
	waitForRateLimit(mergedResp)
	if mergedResult.Total != nil {
		stats.Merged = int64(*mergedResult.Total)
	}

	closedResult, closedResp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr is:unmerged is:closed", username),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search closed PRs for %s: %v", username, err))
		return stats
	}
	waitForRateLimit(closedResp)
	if closedResult.Total != nil {
		stats.Closed = int64(*closedResult.Total)
	}

	return stats
}

// fetchRecentPRRepoCount returns the number of distinct repos the user
// opened PRs in, based on recent public events (last ~90 days).
func fetchRecentPRRepoCount(ctx context.Context, client *hub.Client, username string) int64 {
	repos := make(map[string]struct{})
	page := 1

	for page <= 3 {
		events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, username, true,
			&hub.ListOptions{Page: page, PerPage: pageSize})
		if err != nil {
			slog.Debug(fmt.Sprintf("list events for %s page %d: %v", username, page, err))
			break
		}
		waitForRateLimit(resp)

		for _, e := range events {
			if e.GetType() == "PullRequestEvent" && e.Repo != nil {
				repos[e.Repo.GetName()] = struct{}{}
			}
		}

		if len(events) < pageSize {
			break
		}
		page++
	}

	return int64(len(repos))
}

// fetchForkedRepoCount returns the number of the user's owned repos that are forks.
func fetchForkedRepoCount(ctx context.Context, client *hub.Client, username string) int64 {
	var forked int64
	page := 1

	for page <= 3 {
		repos, resp, err := client.Repositories.ListByUser(ctx, username,
			&hub.RepositoryListByUserOptions{
				Type:        "owner",
				ListOptions: hub.ListOptions{Page: page, PerPage: pageSize},
			})
		if err != nil {
			slog.Debug(fmt.Sprintf("list repos for %s page %d: %v", username, page, err))
			break
		}
		waitForRateLimit(resp)

		for _, r := range repos {
			if r.GetFork() {
				forked++
			}
		}

		if len(repos) < pageSize {
			break
		}
		page++
	}

	return forked
}

// fetchAuthorAssociation returns the author_association for a user in a repo.
// Queries the user's PRs in the target repo and reads the association from the first result.
func fetchAuthorAssociation(ctx context.Context, client *hub.Client, username, owner, repo string) string {
	result, resp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr repo:%s/%s", username, owner, repo),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search PRs for %s in %s/%s: %v", username, owner, repo, err))
		return ""
	}
	waitForRateLimit(resp)

	if result.Total != nil && *result.Total > 0 && len(result.Issues) > 0 {
		return result.Issues[0].GetAuthorAssociation()
	}

	return ""
}
