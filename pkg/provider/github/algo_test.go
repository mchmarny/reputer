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
	followerRation := 100
	followingRation := 10

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
				Suspended: true,
			},
		},
		{
			name:      "date only",
			wantScore: 0.53,
			author: &report.Author{
				Created: time.Now().AddDate(-3, 0, 0),
			},
		},
		{
			name:      "positive ratio",
			wantScore: 0.18,
			author: &report.Author{
				Created:   time.Now().AddDate(-1, 0, 0),
				Followers: &followerRation,
				Following: &followingRation,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calculateReputation(tt.author)

			if err != nil && tt.wantError == nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.author != nil && tt.author.Reputation != tt.wantScore {
				t.Errorf("wrong reputation: got = %v, want %v", tt.author.Reputation, tt.wantScore)
			}
		})
	}
}
