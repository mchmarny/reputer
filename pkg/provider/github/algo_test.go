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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculateReputation(tt.author)
			assert.NotNil(t, tt.author)
			assert.Equalf(t, tt.wantScore, tt.author.Reputation,
				"%s - wrong reputation: got = %v, want %v",
				tt.author.Username, tt.author.Reputation, tt.wantScore)
		})
	}
}
