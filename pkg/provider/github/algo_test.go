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

func TestCalculateReputation(t *testing.T) {
	tests := []struct {
		name      string
		author    *report.Author
		wantScore float64
		wantError error
	}{
		{
			name:      "suspended",
			wantScore: 0,
			author: &report.Author{
				Username: "suspended",
				Stats: &report.Stats{
					Suspended: true,
				},
			},
		},
		{
			name:      "age only",
			wantScore: 0.1,
			author: &report.Author{
				Username: "date-only",
				Stats: &report.Stats{
					AgeDays: 1095,
				},
			},
		},
		{
			name:      "positive ratio",
			wantScore: 0.15,
			author: &report.Author{
				Username: "positive-ratio",
				Stats: &report.Stats{
					AgeDays:   90,
					Followers: 100,
					Following: 10,
				},
			},
		},
		{
			name:      "negative ratio",
			wantScore: 0,
			author: &report.Author{
				Username: "negative-ratio",
				Stats: &report.Stats{
					AgeDays:   90,
					Followers: 10,
					Following: 100,
				},
			},
		},
		{
			name:      "public repos",
			wantScore: 0.05,
			author: &report.Author{
				Username: "public-repos",
				Stats: &report.Stats{
					AgeDays:     90,
					PublicRepos: 10,
				},
			},
		},
		{
			name:      "2FA",
			wantScore: 0.35,
			author: &report.Author{
				Username: "two-factor",
				Stats: &report.Stats{
					StrongAuth: true,
					AgeDays:    90,
				},
			},
		},
		{
			name:      "max score - all criteria met",
			wantScore: 0.95,
			author: &report.Author{
				Username: "max-score",
				Stats: &report.Stats{
					StrongAuth:      true,
					CommitsVerified: true,
					AgeDays:         365,
					Followers:       100,
					Following:       10,
					PublicRepos:     10,
					PrivateRepos:    5,
				},
			},
		},
		{
			name:      "nil author",
			wantScore: 0,
			author:    nil,
		},
		{
			name:      "nil stats",
			wantScore: 0,
			author: &report.Author{
				Username: "nil-stats",
				Stats:    nil,
			},
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
