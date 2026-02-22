package github

import (
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/mchmarny/reputer/pkg/score"
)

// calculateReputation scores an author by delegating to the score package.
func calculateReputation(author *report.Author, totalCommits int64, totalContributors int) {
	if author == nil || author.Stats == nil {
		return
	}

	s := author.Stats

	author.Reputation = score.Compute(score.Signals{
		Suspended:         s.Suspended,
		StrongAuth:        s.StrongAuth,
		Commits:           s.Commits,
		UnverifiedCommits: s.UnverifiedCommits,
		TotalCommits:      totalCommits,
		TotalContributors: totalContributors,
		AgeDays:           s.AgeDays,
		OrgMember:         s.OrgMember,
		LastCommitDays:    s.LastCommitDays,
		Followers:         s.Followers,
		Following:         s.Following,
		PublicRepos:       s.PublicRepos,
		PrivateRepos:      s.PrivateRepos,
	})
}
