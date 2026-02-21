package github

import (
	"os"
	"testing"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	log.SetLevel(log.DebugLevel)
	os.Exit(m.Run())
}

func TestClampedRatio(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		max  float64
		want float64
	}{
		{"zero val", 0, 10, 0},
		{"negative val", -5, 10, 0},
		{"zero max", 5, 0, 0},
		{"negative max", 5, -10, 0},
		{"both zero", 0, 0, 0},
		{"half", 5, 10, 0.5},
		{"at max", 10, 10, 1},
		{"over max", 20, 10, 1},
		{"small fraction", 1, 10, 0.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampedRatio(tt.val, tt.max)
			assert.InDelta(t, tt.want, got, 0.001, "clampedRatio(%v, %v)", tt.val, tt.max)
		})
	}
}

func TestCalculateReputation(t *testing.T) {
	tests := []struct {
		name      string
		author    *report.Author
		wantScore float64
	}{
		{
			name: "suspended with stats",
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
			name: "zero stats",
			author: &report.Author{
				Username: "zero-stats",
				Stats:    &report.Stats{},
			},
			wantScore: 0,
		},
		{
			name: "2FA only",
			author: &report.Author{
				Username: "2fa-only",
				Stats: &report.Stats{
					StrongAuth: true,
				},
			},
			wantScore: 0.25,
		},
		{
			name: "age 182 days - partial",
			// 182/365 * 0.15 = 0.0747... → rounds to 0.07
			author: &report.Author{
				Username: "age-182",
				Stats: &report.Stats{
					AgeDays: 182,
				},
			},
			wantScore: 0.07,
		},
		{
			name: "age 365 days - full ceiling",
			// 365/365 * 0.15 = 0.15
			author: &report.Author{
				Username: "age-365",
				Stats: &report.Stats{
					AgeDays: 365,
				},
			},
			wantScore: 0.15,
		},
		{
			name: "age 1095 days - clamped at ceiling",
			// clamped to 1.0 * 0.15 = 0.15
			author: &report.Author{
				Username: "age-1095",
				Stats: &report.Stats{
					AgeDays: 1095,
				},
			},
			wantScore: 0.15,
		},
		{
			name: "10 public repos - partial",
			// 10/20 * 0.10 = 0.05
			author: &report.Author{
				Username: "pub-10",
				Stats: &report.Stats{
					PublicRepos: 10,
				},
			},
			wantScore: 0.05,
		},
		{
			name: "20 public repos - full ceiling",
			// 20/20 * 0.10 = 0.10
			author: &report.Author{
				Username: "pub-20",
				Stats: &report.Stats{
					PublicRepos: 20,
				},
			},
			wantScore: 0.10,
		},
		{
			name: "5 private repos - partial",
			// 5/10 * 0.10 = 0.05
			author: &report.Author{
				Username: "priv-5",
				Stats: &report.Stats{
					PrivateRepos: 5,
				},
			},
			wantScore: 0.05,
		},
		{
			name: "follower ratio 2:1 - partial",
			// ratio=2.0, 2/10 * 0.15 = 0.03
			author: &report.Author{
				Username: "ratio-2",
				Stats: &report.Stats{
					Followers: 20,
					Following: 10,
				},
			},
			wantScore: 0.03,
		},
		{
			name: "follower ratio 10:1 - full ceiling",
			// ratio=10.0, 10/10 * 0.15 = 0.15
			author: &report.Author{
				Username: "ratio-10",
				Stats: &report.Stats{
					Followers: 100,
					Following: 10,
				},
			},
			wantScore: 0.15,
		},
		{
			name: "follower ratio 0 following - skip",
			author: &report.Author{
				Username: "ratio-zero-following",
				Stats: &report.Stats{
					Followers: 100,
					Following: 0,
				},
			},
			wantScore: 0,
		},
		{
			name: "follower ratio 1:100 - rounds to 0",
			// ratio=0.01, 0.01/10 * 0.15 = 0.00015 → rounds to 0
			author: &report.Author{
				Username: "ratio-bad",
				Stats: &report.Stats{
					Followers: 1,
					Following: 100,
				},
			},
			wantScore: 0,
		},
		{
			name: "10/10 verified commits",
			// 10/10=1.0, clampedRatio(1.0,1.0)=1, 1*0.25=0.25
			author: &report.Author{
				Username: "verified-all",
				Stats: &report.Stats{
					Commits:           10,
					UnverifiedCommits: 0,
				},
			},
			wantScore: 0.25,
		},
		{
			name: "9/10 verified commits",
			// 9/10=0.9, 0.9*0.25=0.225 → rounds to 0.23
			author: &report.Author{
				Username: "verified-9",
				Stats: &report.Stats{
					Commits:           10,
					UnverifiedCommits: 1,
				},
			},
			wantScore: 0.23,
		},
		{
			name: "5/10 verified commits",
			// 5/10=0.5, 0.5*0.25=0.125 → rounds to 0.13
			author: &report.Author{
				Username: "verified-5",
				Stats: &report.Stats{
					Commits:           10,
					UnverifiedCommits: 5,
				},
			},
			wantScore: 0.13,
		},
		{
			name: "0/10 verified commits",
			// 0/10=0, clampedRatio(0,1.0)=0
			author: &report.Author{
				Username: "verified-0",
				Stats: &report.Stats{
					Commits:           10,
					UnverifiedCommits: 10,
				},
			},
			wantScore: 0,
		},
		{
			name: "all criteria max - full score",
			// 0.25+0.25+0.15+0.15+0.10+0.10 = 1.00
			author: &report.Author{
				Username: "max-score",
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           100,
					UnverifiedCommits: 0,
					AgeDays:           730,
					Followers:         200,
					Following:         10,
					PublicRepos:       30,
					PrivateRepos:      15,
				},
			},
			wantScore: 1.00,
		},
		{
			name: "realistic mid-level contributor",
			// 2FA: 0.25
			// age: 200/365*0.15 = 0.0821... → contributes 0.08 to rep
			// public: 8/20*0.10 = 0.04
			// private: 2/10*0.10 = 0.02
			// ratio: followers=15,following=10 → 1.5/10*0.15 = 0.0225
			// verified: 7/10=0.7 → 0.7*0.25 = 0.175
			// total = 0.25+0.0821+0.04+0.02+0.0225+0.175 = 0.5896 → 0.59
			author: &report.Author{
				Username: "mid-level",
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           10,
					UnverifiedCommits: 3,
					AgeDays:           200,
					Followers:         15,
					Following:         10,
					PublicRepos:       8,
					PrivateRepos:      2,
				},
			},
			wantScore: 0.59,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculateReputation(tt.author)
			if tt.author == nil {
				return
			}
			assert.Equalf(t, tt.wantScore, tt.author.Reputation,
				"%s - wrong reputation: got = %v, want %v",
				tt.author.Username, tt.author.Reputation, tt.wantScore)
		})
	}
}
