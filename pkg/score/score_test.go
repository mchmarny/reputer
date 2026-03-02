package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategoryWeightsSum(t *testing.T) {
	sum := CategoryProvenanceWeight + CategoryIdentityWeight + CategoryEngagementWeight + CategoryCommunityWeight
	assert.InDelta(t, 1.0, sum, 0.001, "category weights must sum to 1.0, got %f", sum)
}

func TestCategories(t *testing.T) {
	cats := Categories()
	assert.Len(t, cats, 4)
	var total float64
	for _, c := range cats {
		assert.NotEmpty(t, c.Name)
		assert.Greater(t, c.Weight, 0.0)
		total += c.Weight
	}
	assert.InDelta(t, 1.0, total, 0.001)
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

func TestLogCurve(t *testing.T) {
	tests := []struct {
		name string
		val  float64
		ceil float64
		want float64
	}{
		{"zero val", 0, 10, 0},
		{"negative val", -5, 10, 0},
		{"zero ceil", 5, 0, 0},
		{"negative ceil", 5, -10, 0},
		{"at ceiling", 730, 730, 1.0},
		{"over ceiling", 1000, 730, 1.0},
		{"1 day of 730", 1, 730, 0.105},
		{"30 days of 730", 30, 730, 0.521},
		{"365 days of 730", 365, 730, 0.895},
		{"ratio 3:1 of 10", 3, 10, 0.578},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := logCurve(tt.val, tt.ceil)
			assert.InDelta(t, tt.want, got, 0.01, "logCurve(%v, %v)", tt.val, tt.ceil)
		})
	}
}

func TestExpDecay(t *testing.T) {
	tests := []struct {
		name     string
		val      float64
		halfLife float64
		want     float64
	}{
		{"zero val", 0, 90, 1.0},
		{"at half-life", 90, 90, 0.5},
		{"double half-life", 180, 90, 0.25},
		{"30 days hl=90", 30, 90, 0.794},
		{"365 days hl=90", 365, 90, 0.060},
		{"zero half-life", 10, 0, 0},
		{"negative half-life", 10, -5, 0},
		{"negative val", -10, 90, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expDecay(tt.val, tt.halfLife)
			assert.InDelta(t, tt.want, got, 0.01, "expDecay(%v, %v)", tt.val, tt.halfLife)
		})
	}
}

func TestCompute(t *testing.T) {
	tests := []struct {
		name      string
		signals   Signals
		wantScore float64
	}{
		{
			name:      "zero-value signals",
			signals:   Signals{TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.10,
		},
		{
			name:      "suspended user",
			signals:   Signals{Suspended: true, StrongAuth: true, AgeDays: 1000, TotalCommits: 100, TotalContributors: 5},
			wantScore: 0.0,
		},
		{
			name: "max signals",
			signals: Signals{
				StrongAuth:        true,
				Commits:           50,
				UnverifiedCommits: 0,
				TotalCommits:      100,
				TotalContributors: 5,
				AgeDays:           730,
				OrgMember:         true,
				LastCommitDays:    0,
				Followers:         100,
				Following:         10,
				PublicRepos:       30,
				PrivateRepos:      15,
			},
			wantScore: 1.00,
		},
		{
			name:      "org member only",
			signals:   Signals{OrgMember: true, TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.20,
		},
		{
			name: "2FA floor only",
			signals: Signals{
				StrongAuth:        true,
				Commits:           10,
				UnverifiedCommits: 10,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			wantScore: 0.29,
		},
		{
			name: "no 2FA core contributor all signed",
			signals: Signals{
				StrongAuth:        false,
				Commits:           10,
				UnverifiedCommits: 0,
				TotalCommits:      20,
				TotalContributors: 2,
			},
			wantScore: 0.38,
		},
		{
			name: "no 2FA peripheral all signed",
			signals: Signals{
				StrongAuth:        false,
				Commits:           1,
				UnverifiedCommits: 0,
				TotalCommits:      100,
				TotalContributors: 20,
			},
			wantScore: 0.43,
		},
		{
			name:      "age 30 days only",
			signals:   Signals{AgeDays: 30, TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.18,
		},
		{
			name:      "age 365 days only",
			signals:   Signals{AgeDays: 365, TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.23,
		},
		{
			name:      "follower ratio 3:1",
			signals:   Signals{Followers: 30, Following: 10, TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.16,
		},
		{
			name:      "5 repos combined",
			signals:   Signals{PublicRepos: 3, PrivateRepos: 2, TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.13,
		},
		{
			name: "engagement only - at ceiling",
			signals: Signals{
				Commits:           10,
				LastCommitDays:    0,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			wantScore: 0.42,
		},
		{
			name: "low confidence repo",
			signals: Signals{
				Commits:           3,
				LastCommitDays:    0,
				TotalCommits:      5,
				TotalContributors: 3,
			},
			wantScore: 0.30,
		},
		{
			name: "recency 90 days solo repo",
			signals: Signals{
				Commits:           5,
				LastCommitDays:    90,
				TotalCommits:      10,
				TotalContributors: 1,
			},
			wantScore: 0.34,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compute(tt.signals)
			assert.InDelta(t, tt.wantScore, got, 0.01,
				"%s: got=%.4f want=%.4f", tt.name, got, tt.wantScore)
		})
	}
}

func TestModelVersion(t *testing.T) {
	assert.Equal(t, "2.0.0", ModelVersion)
}
