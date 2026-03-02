package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCategoryWeightsSum(t *testing.T) {
	sum := CategoryProvenanceWeight + CategoryIdentityWeight + CategoryEngagementWeight + CategoryCommunityWeight + CategoryBehavioralWeight
	assert.InDelta(t, 1.0, sum, 0.001, "category weights must sum to 1.0, got %f", sum)
}

func TestCategories(t *testing.T) {
	cats := Categories()
	assert.Len(t, cats, 5)
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

func TestAssociationScore(t *testing.T) {
	tests := []struct {
		name             string
		assoc            string
		orgMember        bool
		trustedOrgMember bool
		want             float64
	}{
		{"OWNER", "OWNER", false, false, 1.0},
		{"MEMBER", "MEMBER", false, false, 1.0},
		{"COLLABORATOR", "COLLABORATOR", false, false, 0.8},
		{"CONTRIBUTOR", "CONTRIBUTOR", false, false, 0.5},
		{"FIRST_TIME_CONTRIBUTOR", "FIRST_TIME_CONTRIBUTOR", false, false, 0.2},
		{"NONE", "NONE", false, false, 0.0},
		{"empty with org", "", true, false, 1.0},
		{"empty without org", "", false, false, 0.0},
		{"unknown", "UNKNOWN", false, false, 0.0},
		// trusted org member cases
		{"trusted NONE", "NONE", false, true, 0.8},
		{"trusted FIRST_TIME", "FIRST_TIME_CONTRIBUTOR", false, true, 0.8},
		{"trusted CONTRIBUTOR", "CONTRIBUTOR", false, true, 0.8},
		{"trusted COLLABORATOR", "COLLABORATOR", false, true, 0.8},
		{"trusted OWNER", "OWNER", false, true, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := associationScore(tt.assoc, tt.orgMember, tt.trustedOrgMember)
			assert.InDelta(t, tt.want, got, 0.001)
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
			wantScore: 0.15,
		},
		{
			name:      "suspended user",
			signals:   Signals{Suspended: true, AgeDays: 1000, TotalCommits: 100, TotalContributors: 5},
			wantScore: 0.0,
		},
		{
			name: "max signals",
			signals: Signals{
				Commits:           50,
				UnverifiedCommits: 0,
				TotalCommits:      100,
				TotalContributors: 5,
				AgeDays:           730,
				OrgMember:         true,
				AuthorAssociation: "OWNER",
				HasBio:            true,
				HasCompany:        true,
				HasLocation:       true,
				HasWebsite:        true,
				LastCommitDays:    0,
				Followers:         100,
				Following:         10,
				PublicRepos:       30,
				PRsMerged:         100,
				PRsClosed:         0,
			},
			wantScore: 1.00,
		},
		{
			name: "hackerbot-claw profile",
			signals: Signals{
				AgeDays:           7,
				TotalCommits:      50,
				TotalContributors: 10,
				Commits:           5,
				UnverifiedCommits: 5,
				RecentPRRepoCount: 7,
				PublicRepos:       7,
				ForkedRepos:       7,
			},
			wantScore: 0.23,
		},
		{
			name: "new legitimate contributor",
			signals: Signals{
				AgeDays:           60,
				HasBio:            true,
				Commits:           5,
				UnverifiedCommits: 0,
				TotalCommits:      50,
				TotalContributors: 5,
				LastCommitDays:    3,
				PRsMerged:         8,
				PRsClosed:         2,
				PublicRepos:       3,
				RecentPRRepoCount: 2,
			},
			wantScore: 0.53,
		},
		{
			name: "association CONTRIBUTOR",
			signals: Signals{
				AuthorAssociation: "CONTRIBUTOR",
				AgeDays:           365,
				Commits:           10,
				TotalCommits:      100,
				TotalContributors: 10,
				LastCommitDays:    5,
				PublicRepos:       5,
			},
			wantScore: 0.74,
		},
		{
			name: "association FIRST_TIME_CONTRIBUTOR",
			signals: Signals{
				AuthorAssociation: "FIRST_TIME_CONTRIBUTOR",
				AgeDays:           30,
				Commits:           1,
				TotalCommits:      100,
				TotalContributors: 10,
				LastCommitDays:    1,
				PublicRepos:       1,
			},
			wantScore: 0.37,
		},
		{
			name: "profile completeness 2/4",
			signals: Signals{
				HasBio:            true,
				HasLocation:       true,
				AgeDays:           200,
				TotalCommits:      100,
				TotalContributors: 10,
				PublicRepos:       5,
			},
			wantScore: 0.45,
		},
		{
			name: "high PR acceptance rate",
			signals: Signals{
				PRsMerged:         50,
				PRsClosed:         2,
				AgeDays:           365,
				Commits:           20,
				TotalCommits:      100,
				TotalContributors: 10,
				LastCommitDays:    5,
				PublicRepos:       10,
			},
			wantScore: 0.78,
		},
		{
			name: "fork-only account",
			signals: Signals{
				AgeDays:           100,
				PublicRepos:       10,
				ForkedRepos:       10,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			wantScore: 0.32,
		},
		{
			name: "burst rate high",
			signals: Signals{
				AgeDays:           14,
				RecentPRRepoCount: 10,
				PublicRepos:       10,
				Commits:           3,
				TotalCommits:      50,
				TotalContributors: 5,
				LastCommitDays:    1,
			},
			wantScore: 0.39,
		},
		{
			name: "org member fallback no association",
			signals: Signals{
				OrgMember:         true,
				AgeDays:           500,
				Commits:           20,
				TotalCommits:      100,
				TotalContributors: 10,
				LastCommitDays:    2,
				PublicRepos:       8,
			},
			wantScore: 0.79,
		},
		{
			name: "trusted org member with NONE association",
			signals: Signals{
				TrustedOrgMember:  true,
				AuthorAssociation: "NONE",
				AgeDays:           500,
				Commits:           20,
				TotalCommits:      100,
				TotalContributors: 10,
				LastCommitDays:    2,
				PublicRepos:       8,
			},
			wantScore: 0.78,
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
	assert.Equal(t, "3.2.0", ModelVersion)
}
