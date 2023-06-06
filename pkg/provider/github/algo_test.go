package github

import (
	"os"
	"testing"
	"time"

	"github.com/mchmarny/reputer/pkg/report"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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
			name:      "nil",
			wantError: errors.New("author required"),
		},
		{
			name:      "suspended",
			wantScore: 0,
			author: &report.Author{
				Username:  "suspended",
				Suspended: true,
			},
		},
		{
			name:      "date only",
			wantScore: 0.1,
			author: &report.Author{
				Username: "date-only",
				Created:  time.Now().AddDate(-3, 0, 0),
			},
		},
		{
			name:      "positive ratio",
			wantScore: 0.15,
			author: &report.Author{
				Username:  "positive-ratio",
				Created:   time.Now().AddDate(0, -3, 0),
				Followers: 100,
				Following: 10,
			},
		},
		{
			name:      "negative ratio",
			wantScore: 0,
			author: &report.Author{
				Username:  "negative-ratio",
				Created:   time.Now().AddDate(0, -5, 0),
				Followers: 10,
				Following: 100,
			},
		},
		{
			name:      "public repos",
			wantScore: 0.05,
			author: &report.Author{
				Username:     "public-repos",
				Created:      time.Now().AddDate(0, -5, 0),
				PublicRepos:  10,
				PrivateRepos: 1,
			},
		},
		{
			name:      "2FA",
			wantScore: 0.35,
			author: &report.Author{
				Username:   "two-factor",
				Created:    time.Now().AddDate(0, -5, 0),
				StrongAuth: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calculateReputation(tt.author)
			if tt.author != nil && tt.author.Reputation != tt.wantScore {
				t.Errorf("%s - wrong reputation: got = %v, want %v",
					tt.author.Username, tt.author.Reputation, tt.wantScore)
			}
		})
	}
}
