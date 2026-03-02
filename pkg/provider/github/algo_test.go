package github

import (
	"os"
	"testing"

	"github.com/mchmarny/reputer/pkg/logging"
	"github.com/mchmarny/reputer/pkg/report"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logging.SetDefaultLoggerWithLevel("github-test", "test", "debug")
	os.Exit(m.Run())
}

func TestCalculateReputation(t *testing.T) {
	tests := []struct {
		name              string
		author            *report.Author
		totalCommits      int64
		totalContributors int
		wantScore         float64
	}{
		{
			name:      "nil author",
			author:    nil,
			wantScore: 0,
		},
		{
			name: "nil stats",
			author: &report.Author{
				Username: "nil-stats",
				Stats:    nil,
			},
			wantScore: 0,
		},
		{
			name: "suspended",
			author: &report.Author{
				Username: "suspended",
				Stats: &report.Stats{
					Suspended:  true,
					StrongAuth: true,
					AgeDays:    1000,
				},
			},
			wantScore: 0,
		},
		{
			name: "max score",
			author: &report.Author{
				Username: "max",
				Context:  &report.AuthorContext{Company: "ACME"},
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           50,
					UnverifiedCommits: 0,
					AgeDays:           730,
					OrgMember:         true,
					AuthorAssociation: "MEMBER",
					HasBio:            true,
					HasLocation:       true,
					HasWebsite:        true,
					LastCommitDays:    0,
					Followers:         100,
					Following:         10,
					PublicRepos:       30,
					PrivateRepos:      15,
					PRsMerged:         20,
					PRsClosed:         0,
					RecentPRRepoCount: 2,
					ForkedRepos:       0,
				},
			},
			totalCommits:      100,
			totalContributors: 5,
			wantScore:         1.00,
		},
		{
			name: "hackerbot profile",
			author: &report.Author{
				Username: "hackerbot-claw",
				Context:  &report.AuthorContext{},
				Stats: &report.Stats{
					AgeDays:           7,
					RecentPRRepoCount: 7,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.08,
		},
		{
			name: "fork-only attacker",
			author: &report.Author{
				Username: "fork-only",
				Context:  &report.AuthorContext{},
				Stats: &report.Stats{
					AgeDays:           14,
					PublicRepos:       5,
					ForkedRepos:       5,
					RecentPRRepoCount: 5,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculateReputation(tt.author, tt.totalCommits, tt.totalContributors)
			if tt.author == nil {
				return
			}
			assert.InDelta(t, tt.wantScore, tt.author.Reputation, 0.02,
				"%s: got=%.4f want=%.4f", tt.author.Username, tt.author.Reputation, tt.wantScore)
		})
	}
}
