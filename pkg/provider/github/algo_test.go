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
			name: "zero stats",
			author: &report.Author{
				Username: "zero",
				Stats:    &report.Stats{},
			},
			totalCommits:      100,
			totalContributors: 10,
			// LastCommitDays=0 => expDecay=1.0 => recency=0.10
			wantScore: 0.10,
		},
		{
			name: "max score",
			author: &report.Author{
				Username: "max",
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           50,
					UnverifiedCommits: 0,
					AgeDays:           730,
					OrgMember:         true,
					LastCommitDays:    0,
					Followers:         100,
					Following:         10,
					PublicRepos:       30,
					PrivateRepos:      15,
				},
			},
			totalCommits:      100,
			totalContributors: 5,
			wantScore:         1.00,
		},
		{
			// org member only: 0.10 org + 0.10 recency(0 days) = 0.20
			name: "org member only",
			author: &report.Author{
				Username: "org-only",
				Stats: &report.Stats{
					OrgMember: true,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.20,
		},
		{
			name: "2FA floor only",
			author: &report.Author{
				Username: "2fa-floor",
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           10,
					UnverifiedCommits: 10,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// provenance: floor=0.1 * 0.35 = 0.035
			// proportion: clamped(0.1/0.1)=1 * conf=1.0 * 0.15 = 0.15
			// recency: expDecay(0,37.5)=1.0 * 0.10 = 0.10
			// total = 0.035 + 0.15 + 0.10 = 0.285 => 0.29
			wantScore: 0.29,
		},
		{
			name: "no 2FA core contributor all signed",
			author: &report.Author{
				Username: "no-2fa-core",
				Stats: &report.Stats{
					StrongAuth:        false,
					Commits:           10,
					UnverifiedCommits: 0,
				},
			},
			totalCommits:      20,
			totalContributors: 2,
			// provenance: verified=1.0, prop=0.5, ceil=0.5, discount=0.5*1.0=0.5, mult=0.5
			//   raw=1.0*0.5=0.5, 0.5*0.35=0.175
			// proportion: clamped(0.5/0.5)=1 * conf=20/30=0.667 * 0.15 = 0.10
			// recency: halfLife=81.9, decay=1.0, 1.0*0.10=0.10
			// total = 0.175 + 0.10 + 0.10 = 0.375 => 0.38
			wantScore: 0.38,
		},
		{
			name: "no 2FA peripheral all signed",
			author: &report.Author{
				Username: "no-2fa-periph",
				Stats: &report.Stats{
					StrongAuth:        false,
					Commits:           1,
					UnverifiedCommits: 0,
				},
			},
			totalCommits:      100,
			totalContributors: 20,
			// provenance: verified=1.0, prop=0.01, ceil=0.05, discount=0.5*0.2=0.1, mult=0.9
			//   raw=1.0*0.9=0.9, 0.9*0.35=0.315
			// proportion: clamped(0.01/0.05)=0.2 * conf=100/200=0.5 * 0.15 = 0.015
			// recency: halfLife=29.6, decay=1.0, 1.0*0.10=0.10
			// total = 0.315 + 0.015 + 0.10 = 0.43
			wantScore: 0.43,
		},
		{
			name: "age 30 days only",
			author: &report.Author{
				Username: "age-30",
				Stats: &report.Stats{
					AgeDays: 30,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// age: logCurve(30,730)*0.15 = 0.521*0.15 = 0.078
			// recency: 1.0*0.10 = 0.10
			// total = 0.178 => 0.18
			wantScore: 0.18,
		},
		{
			name: "age 365 days only",
			author: &report.Author{
				Username: "age-365",
				Stats: &report.Stats{
					AgeDays: 365,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// age: logCurve(365,730)*0.15 = 0.895*0.15 = 0.134
			// recency: 1.0*0.10 = 0.10
			// total = 0.234 => 0.23
			wantScore: 0.23,
		},
		{
			name: "follower ratio 3:1",
			author: &report.Author{
				Username: "ratio-3",
				Stats: &report.Stats{
					Followers: 30,
					Following: 10,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// recency: 1.0*0.10 = 0.10
			// followers: logCurve(3,10)*0.10 = 0.578*0.10 = 0.058
			// total = 0.158 => 0.16
			wantScore: 0.16,
		},
		{
			name: "5 repos combined",
			author: &report.Author{
				Username: "repos-5",
				Stats: &report.Stats{
					PublicRepos:  3,
					PrivateRepos: 2,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// recency: 1.0*0.10 = 0.10
			// repos: logCurve(5,30)*0.05 = 0.522*0.05 = 0.026
			// total = 0.126 => 0.13
			wantScore: 0.13,
		},
		{
			name: "engagement only - at ceiling",
			author: &report.Author{
				Username: "engaged",
				Stats: &report.Stats{
					Commits:        10,
					LastCommitDays: 0,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			// provenance (no 2FA): verified=1.0, prop=0.1, ceil=0.1, discount=0.5*1=0.5, mult=0.5
			//   raw=1.0*0.5=0.5, 0.5*0.35=0.175
			// proportion: clamped(0.1/0.1)=1 * conf=1.0 * 0.15 = 0.15
			// recency: decay=1.0, 1.0*0.10=0.10
			// total = 0.175+0.15+0.10 = 0.425 => 0.42
			wantScore: 0.42,
		},
		{
			name: "low confidence repo",
			author: &report.Author{
				Username: "low-conf",
				Stats: &report.Stats{
					Commits:        3,
					LastCommitDays: 0,
				},
			},
			totalCommits:      5,
			totalContributors: 3,
			// provenance (no 2FA): verified=1.0, prop=3/5=0.6, ceil=max(1/3,0.05)=0.333
			//   discount=0.5*1.0=0.5, mult=0.5, raw=0.5, 0.5*0.35=0.175
			// proportion: clamped(0.6/0.333)=1.0 * conf=5/30=0.167 * 0.15 = 0.025
			// recency: halfLife=64.9, decay=1.0, 1.0*0.10=0.10
			// total = 0.175+0.025+0.10 = 0.30
			wantScore: 0.30,
		},
		{
			name: "recency 90 days solo repo",
			author: &report.Author{
				Username: "dormant-solo",
				Stats: &report.Stats{
					Commits:        5,
					LastCommitDays: 90,
				},
			},
			totalCommits:      10,
			totalContributors: 1,
			// provenance (no 2FA): verified=1.0, prop=5/10=0.5, ceil=max(1/1,0.05)=1.0
			//   discount=0.5*0.5=0.25, mult=0.75, raw=0.75, 0.75*0.35=0.2625
			// proportion: clamped(0.5/1.0)=0.5 * conf=10/30=0.333 * 0.15 = 0.025
			// recency: halfLife=90.0 (1/ln(2)=1.4427>1.0 capped), decay(90,90)=0.5, 0.5*0.10=0.05
			// total = 0.2625+0.025+0.05 = 0.3375 => 0.34
			wantScore: 0.34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculateReputation(tt.author, tt.totalCommits, tt.totalContributors)
			if tt.author == nil {
				return
			}
			assert.InDelta(t, tt.wantScore, tt.author.Reputation, 0.01,
				"%s: got=%.4f want=%.4f", tt.author.Username, tt.author.Reputation, tt.wantScore)
		})
	}
}
