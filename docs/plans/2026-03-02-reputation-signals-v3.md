# Reputation Signals v3.0.0 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 5 new signal groups (author association, PR acceptance rate, cross-repo burst, fork-only ratio, profile completeness) to the reputation scoring model and bump to v3.0.0.

**Architecture:** New signals are fetched via additional GitHub API calls (search/issues, user events, user repos) in the existing concurrent `loadAuthor` flow. The `score.Compute` function is rebalanced with a new Behavioral category. All existing helper functions (`logCurve`, `clampedRatio`, `expDecay`) are reused.

**Tech Stack:** Go, go-github/v72, testify/assert, errgroup

---

### Task 1: Add New Fields to score.Signals

**Files:**
- Modify: `pkg/score/score.go:49-64`

**Step 1: Write the failing test**

Add to `pkg/score/score_test.go` after the existing `TestModelVersion`:

```go
func TestModelVersionV3(t *testing.T) {
	assert.Equal(t, "3.0.0", ModelVersion)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/score/ -run TestModelVersionV3 -v`
Expected: FAIL — `"2.0.0"` != `"3.0.0"`

**Step 3: Update Signals struct and model version**

In `pkg/score/score.go`, change:
```go
const ModelVersion = "2.0.0"
```
to:
```go
const ModelVersion = "3.0.0"
```

Add new fields to the `Signals` struct (after `PrivateRepos`):

```go
type Signals struct {
	Suspended         bool   // Account suspended
	StrongAuth        bool   // 2FA enabled
	Commits           int64  // Author's commit count
	UnverifiedCommits int64  // Commits without verified signatures
	TotalCommits      int64  // Repo-wide total commits
	TotalContributors int    // Repo-wide contributor count
	AgeDays           int64  // Days since account creation
	OrgMember         bool   // Member of repo owner's org (kept for backward compat)
	LastCommitDays    int64  // Days since most recent commit
	Followers         int64  // GitHub follower count
	Following         int64  // Following count (for ratio)
	PublicRepos       int64  // Public repository count
	PrivateRepos      int64  // Private repository count

	// v3 signals
	AuthorAssociation string // OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR, NONE
	HasBio            bool   // Profile has bio
	HasCompany        bool   // Profile has company
	HasLocation       bool   // Profile has location
	HasWebsite        bool   // Profile has website/blog
	PRsMerged         int64  // Global merged PR count
	PRsClosed         int64  // Global closed-without-merge PR count
	RecentPRRepoCount int64  // Distinct repos with PR events in last 90 days
	ForkedRepos       int64  // Owned repos that are forks
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/score/ -run TestModelVersionV3 -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/score/score.go pkg/score/score_test.go
git commit -S -m "Add v3 signal fields to score.Signals struct"
```

---

### Task 2: Rebalance Weights and Update score.Compute

**Files:**
- Modify: `pkg/score/score.go` (constants, exported vars, Categories, Compute)

**Step 1: Write the failing tests**

Replace the weight sum test and add new category tests in `pkg/score/score_test.go`:

```go
func TestCategoryWeightsSum(t *testing.T) {
	sum := CategoryProvenanceWeight + CategoryIdentityWeight +
		CategoryEngagementWeight + CategoryCommunityWeight + CategoryBehavioralWeight
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/score/ -run "TestCategoryWeightsSum|TestCategories" -v`
Expected: FAIL — `CategoryBehavioralWeight` undefined, Categories returns 4 not 5

**Step 3: Rewrite constants, exported vars, Categories, and Compute**

Replace the constants block in `pkg/score/score.go`:

```go
const (
	// Category weights (sum to 1.0).
	provenanceWeight   = 0.30
	ageWeight          = 0.10
	associationWeight  = 0.05
	profileWeight      = 0.05
	proportionWeight   = 0.10
	recencyWeight      = 0.05
	prAcceptWeight     = 0.05
	followerWeight     = 0.05
	repoCountWeight    = 0.05
	burstWeight        = 0.10
	forkOnlyWeight     = 0.10

	// Ceilings and parameters.
	ageCeilDays           = 730
	followerRatioCeil     = 10.0
	repoCountCeil         = 30.0
	baseHalfLifeDays      = 90.0
	minHalfLifeMultiple   = 0.25
	provenanceFloor       = 0.1
	minProportionCeil     = 0.05
	minConfidenceCommits  = 30
	confCommitsPerContrib = 10
	noTFAMaxDiscount      = 0.5
	prCountCeil           = 20.0
	burstCeil             = 5.0
	forkOriginalCeil      = 5.0
)
```

Replace the exported vars:

```go
var (
	CategoryProvenanceWeight  = provenanceWeight
	CategoryIdentityWeight    = ageWeight + associationWeight + profileWeight
	CategoryEngagementWeight  = proportionWeight + recencyWeight + prAcceptWeight
	CategoryCommunityWeight   = followerWeight + repoCountWeight
	CategoryBehavioralWeight  = burstWeight + forkOnlyWeight
)
```

Replace `Categories()`:

```go
func Categories() []CategoryWeight {
	return []CategoryWeight{
		{Name: "code_provenance", Weight: CategoryProvenanceWeight},
		{Name: "identity", Weight: CategoryIdentityWeight},
		{Name: "engagement", Weight: CategoryEngagementWeight},
		{Name: "community", Weight: CategoryCommunityWeight},
		{Name: "behavioral", Weight: CategoryBehavioralWeight},
	}
}
```

Replace `Compute`:

```go
// Compute returns a reputation score in [0.0, 1.0] using the v3 weighted model.
func Compute(s Signals) float64 {
	if s.Suspended {
		slog.Debug("score - suspended")
		return 0
	}

	var rep float64

	// --- Category 1: Code Provenance (0.30) ---
	if s.Commits > 0 && s.TotalCommits > 0 {
		verifiedRatio := float64(s.Commits-s.UnverifiedCommits) / float64(s.Commits)
		proportion := float64(s.Commits) / float64(s.TotalCommits)
		propCeil := math.Max(1.0/float64(max(s.TotalContributors, 1)), minProportionCeil)

		securityMultiplier := 1.0
		if !s.StrongAuth {
			securityMultiplier = 1.0 - noTFAMaxDiscount*clampedRatio(proportion, propCeil)
		}

		rawScore := verifiedRatio * securityMultiplier
		if s.StrongAuth && rawScore < provenanceFloor {
			rawScore = provenanceFloor
		}

		rep += rawScore * provenanceWeight
		slog.Debug(fmt.Sprintf("provenance: %.4f (verified=%.2f, multiplier=%.2f)",
			rawScore*provenanceWeight, verifiedRatio, securityMultiplier))
	} else if s.StrongAuth {
		rep += provenanceFloor * provenanceWeight
		slog.Debug(fmt.Sprintf("provenance: %.4f (2FA floor, no commits)", provenanceFloor*provenanceWeight))
	}

	// --- Category 2: Identity (0.20) ---
	ageScore := logCurve(float64(s.AgeDays), ageCeilDays) * ageWeight
	rep += ageScore
	slog.Debug(fmt.Sprintf("age: %.4f (%d days)", ageScore, s.AgeDays))

	assocScore := associationScore(s.AuthorAssociation, s.OrgMember) * associationWeight
	rep += assocScore
	slog.Debug(fmt.Sprintf("association: %.4f (%s)", assocScore, s.AuthorAssociation))

	profileCount := 0
	if s.HasBio {
		profileCount++
	}
	if s.HasCompany {
		profileCount++
	}
	if s.HasLocation {
		profileCount++
	}
	if s.HasWebsite {
		profileCount++
	}
	profScore := float64(profileCount) / 4.0 * profileWeight
	rep += profScore
	slog.Debug(fmt.Sprintf("profile: %.4f (%d/4 fields)", profScore, profileCount))

	// --- Category 3: Engagement (0.20) ---
	if s.Commits > 0 && s.TotalCommits > 0 {
		proportion := float64(s.Commits) / float64(s.TotalCommits)
		propCeil := math.Max(1.0/float64(max(s.TotalContributors, 1)), minProportionCeil)

		confThreshold := float64(max(
			int64(s.TotalContributors)*int64(confCommitsPerContrib),
			int64(minConfidenceCommits),
		))
		confidence := math.Min(float64(s.TotalCommits)/confThreshold, 1.0)

		propScore := clampedRatio(proportion, propCeil) * confidence * proportionWeight
		rep += propScore
		slog.Debug(fmt.Sprintf("proportion: %.4f (prop=%.3f, ceil=%.3f, conf=%.3f)",
			propScore, proportion, propCeil, confidence))
	}

	numContrib := max(s.TotalContributors, 1)
	halfLifeMult := math.Max(1.0/math.Log(1+float64(numContrib)), minHalfLifeMultiple)
	if halfLifeMult > 1.0 {
		halfLifeMult = 1.0
	}
	halfLife := baseHalfLifeDays * halfLifeMult
	recScore := expDecay(float64(s.LastCommitDays), halfLife) * recencyWeight
	rep += recScore
	slog.Debug(fmt.Sprintf("recency: %.4f (%d days, halfLife=%.1f)",
		recScore, s.LastCommitDays, halfLife))

	totalTerminalPRs := s.PRsMerged + s.PRsClosed
	if totalTerminalPRs > 0 {
		mergeRate := float64(s.PRsMerged) / float64(totalTerminalPRs)
		confidence := logCurve(float64(totalTerminalPRs), prCountCeil)
		prScore := mergeRate * confidence * prAcceptWeight
		rep += prScore
		slog.Debug(fmt.Sprintf("pr_acceptance: %.4f (rate=%.2f, conf=%.2f)",
			prScore, mergeRate, confidence))
	}

	// --- Category 4: Community (0.10) ---
	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		rep += logCurve(ratio, followerRatioCeil) * followerWeight
		slog.Debug(fmt.Sprintf("followers: %.4f (ratio=%.2f)",
			logCurve(ratio, followerRatioCeil)*followerWeight, ratio))
	}

	totalRepos := float64(s.PublicRepos + s.PrivateRepos)
	rep += logCurve(totalRepos, repoCountCeil) * repoCountWeight
	slog.Debug(fmt.Sprintf("repos: %.4f (%d combined)",
		logCurve(totalRepos, repoCountCeil)*repoCountWeight, int64(totalRepos)))

	// --- Category 5: Behavioral (0.20) ---
	// Cross-repo burst: high PR activity across many repos relative to account age = suspicious
	if s.RecentPRRepoCount > 0 && s.AgeDays > 0 {
		ageMonths := math.Max(float64(s.AgeDays)/30.0, 1.0)
		burstRate := float64(s.RecentPRRepoCount) / ageMonths
		bScore := (1.0 - clampedRatio(burstRate, burstCeil)) * burstWeight
		rep += bScore
		slog.Debug(fmt.Sprintf("burst: %.4f (rate=%.2f, repos=%d)",
			bScore, burstRate, s.RecentPRRepoCount))
	} else {
		// No recent PR repos or brand new account: no burst detected = full score
		rep += burstWeight
		slog.Debug(fmt.Sprintf("burst: %.4f (no recent PR repos)", burstWeight))
	}

	// Fork-only: accounts with only forked repos and no original work
	totalOwnedRepos := s.PublicRepos + s.PrivateRepos
	if totalOwnedRepos > 0 {
		originalRepos := float64(totalOwnedRepos - s.ForkedRepos)
		fScore := clampedRatio(originalRepos, forkOriginalCeil) * forkOnlyWeight
		rep += fScore
		slog.Debug(fmt.Sprintf("fork_ratio: %.4f (original=%.0f, forked=%d)",
			fScore, originalRepos, s.ForkedRepos))
	} else {
		// No repos at all: score 0 for this signal
		slog.Debug("fork_ratio: 0.0000 (no repos)")
	}

	result := toFixed(rep, 2)
	slog.Debug(fmt.Sprintf("reputation: %.2f", result))
	return result
}

// associationScore maps GitHub's author_association to a [0, 1] score.
// Falls back to OrgMember boolean when association is empty.
func associationScore(assoc string, orgMember bool) float64 {
	switch assoc {
	case "OWNER", "MEMBER":
		return 1.0
	case "COLLABORATOR":
		return 0.8
	case "CONTRIBUTOR":
		return 0.5
	case "FIRST_TIME_CONTRIBUTOR":
		return 0.2
	case "NONE":
		return 0.0
	default:
		// Fallback: use legacy OrgMember boolean
		if orgMember {
			return 1.0
		}
		return 0.0
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./pkg/score/ -run "TestCategoryWeightsSum|TestCategories" -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/score/score.go pkg/score/score_test.go
git commit -S -m "Rebalance scoring model to v3.0.0 with behavioral category"
```

---

### Task 3: Rewrite score_test.go TestCompute Cases for v3

**Files:**
- Modify: `pkg/score/score_test.go`

**Step 1: Rewrite all TestCompute test cases for v3 weights**

The test expectations must change because weights shifted. Replace the entire `TestCompute` function. Key test cases:

```go
func TestCompute(t *testing.T) {
	tests := []struct {
		name      string
		signals   Signals
		wantScore float64
	}{
		{
			name: "zero-value signals",
			// burst: no recent PR repos → burstWeight(0.10)
			// recency: lastCommitDays=0 → expDecay=1.0 → 0.05
			// total: 0.15
			signals:   Signals{TotalCommits: 100, TotalContributors: 10},
			wantScore: 0.15,
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
				AuthorAssociation: "MEMBER",
				HasBio:            true,
				HasCompany:        true,
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
			wantScore: 1.00,
		},
		{
			name: "hackerbot-claw profile",
			// 7-day account, no 2FA, no repos, no followers, 7 repos with PRs in 7 days
			signals: Signals{
				AgeDays:           7,
				RecentPRRepoCount: 7,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// age: logCurve(7,730)*0.10 ≈ 0.304*0.10 = 0.030
			// recency: 1.0*0.05 = 0.05
			// burst: ageMonths=max(7/30,1)=1, rate=7/1=7, clamped(7,5)=1.0, (1-1)*0.10=0
			// total ≈ 0.08
			wantScore: 0.08,
		},
		{
			name: "new legitimate contributor",
			// 60-day account, 2FA, has bio+company, 1 PR merged, 2 repos (0 forks)
			signals: Signals{
				StrongAuth:        true,
				AgeDays:           60,
				HasBio:            true,
				HasCompany:        true,
				PublicRepos:       2,
				PRsMerged:         1,
				PRsClosed:         0,
				RecentPRRepoCount: 1,
				ForkedRepos:       0,
				Commits:           5,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// This should score notably higher than hackerbot
			wantScore: 0.50,
		},
		{
			name: "association CONTRIBUTOR",
			signals: Signals{
				AuthorAssociation: "CONTRIBUTOR",
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// association: 0.5*0.05 = 0.025
			// recency: 1.0*0.05 = 0.05
			// burst: no recent PRs → 0.10
			// total ≈ 0.18
			wantScore: 0.18,
		},
		{
			name: "association FIRST_TIME_CONTRIBUTOR",
			signals: Signals{
				AuthorAssociation: "FIRST_TIME_CONTRIBUTOR",
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// association: 0.2*0.05 = 0.01
			// recency: 1.0*0.05 = 0.05
			// burst: no recent PRs → 0.10
			// total ≈ 0.16
			wantScore: 0.16,
		},
		{
			name: "profile completeness 2/4",
			signals: Signals{
				HasBio:     true,
				HasWebsite: true,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// profile: 2/4*0.05 = 0.025
			// recency: 0.05, burst: 0.10
			// total ≈ 0.18
			wantScore: 0.18,
		},
		{
			name: "high PR acceptance rate",
			signals: Signals{
				PRsMerged:         15,
				PRsClosed:         5,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// mergeRate=0.75, conf=logCurve(20,20)=1.0, score=0.75*1.0*0.05=0.0375
			// recency: 0.05, burst: 0.10
			// total ≈ 0.19
			wantScore: 0.19,
		},
		{
			name: "fork-only account",
			signals: Signals{
				PublicRepos: 5,
				ForkedRepos: 5,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// fork: original=0, score=0
			// repos: logCurve(5,30)*0.05 ≈ 0.026
			// recency: 0.05, burst: 0.10
			// total ≈ 0.18
			wantScore: 0.18,
		},
		{
			name: "burst rate high",
			signals: Signals{
				AgeDays:           30,
				RecentPRRepoCount: 10,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// age: logCurve(30,730)*0.10 ≈ 0.052
			// burst: ageMonths=1, rate=10, clamped(10,5)=1.0, (1-1)*0.10=0
			// recency: 0.05
			// total ≈ 0.10
			wantScore: 0.10,
		},
		{
			name: "org member fallback no association",
			signals: Signals{
				OrgMember:         true,
				TotalCommits:      100,
				TotalContributors: 10,
			},
			// association: fallback to orgMember=true → 1.0*0.05 = 0.05
			// recency: 0.05, burst: 0.10
			// total ≈ 0.20
			wantScore: 0.20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Compute(tt.signals)
			assert.InDelta(t, tt.wantScore, got, 0.02,
				"%s: got=%.4f want=%.4f", tt.name, got, tt.wantScore)
		})
	}
}
```

Also add a test for `associationScore`:

```go
func TestAssociationScore(t *testing.T) {
	tests := []struct {
		assoc     string
		orgMember bool
		want      float64
	}{
		{"OWNER", false, 1.0},
		{"MEMBER", false, 1.0},
		{"COLLABORATOR", false, 0.8},
		{"CONTRIBUTOR", false, 0.5},
		{"FIRST_TIME_CONTRIBUTOR", false, 0.2},
		{"NONE", false, 0.0},
		{"", true, 1.0},    // fallback to OrgMember
		{"", false, 0.0},   // fallback, no OrgMember
		{"UNKNOWN", false, 0.0}, // unrecognized → fallback
	}
	for _, tt := range tests {
		t.Run(tt.assoc, func(t *testing.T) {
			got := associationScore(tt.assoc, tt.orgMember)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}
```

**Step 2: Run all score tests**

Run: `go test ./pkg/score/ -v`
Expected: ALL PASS. If any `wantScore` values are slightly off, adjust within ±0.02 tolerance after verifying the math.

**Step 3: Commit**

```bash
git add pkg/score/score_test.go
git commit -S -m "Update score tests for v3.0.0 weight model"
```

---

### Task 4: Add New Fields to report.Stats

**Files:**
- Modify: `pkg/report/author.go:40-53`

**Step 1: Write the failing test**

Add to `pkg/report/author_test.go`:

```go
func TestMakeAuthorHasNewStatsFields(t *testing.T) {
	a := MakeAuthor("test")
	// Verify new v3 fields exist and are zero-valued
	assert.Empty(t, a.Stats.AuthorAssociation)
	assert.False(t, a.Stats.HasBio)
	assert.False(t, a.Stats.HasLocation)
	assert.False(t, a.Stats.HasWebsite)
	assert.Zero(t, a.Stats.PRsMerged)
	assert.Zero(t, a.Stats.PRsClosed)
	assert.Zero(t, a.Stats.RecentPRRepoCount)
	assert.Zero(t, a.Stats.ForkedRepos)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/report/ -run TestMakeAuthorHasNewStatsFields -v`
Expected: FAIL — fields don't exist

**Step 3: Add fields to Stats struct**

In `pkg/report/author.go`, add after the `OrgMember` field:

```go
type Stats struct {
	Suspended         bool   `json:"suspended,omitempty" yaml:"suspended,omitempty"`
	CommitsVerified   bool   `json:"verified_commits,omitempty" yaml:"verifiedCommits,omitempty"`
	StrongAuth        bool   `json:"strong_auth,omitempty" yaml:"strongAuth,omitempty"`
	AgeDays           int64  `json:"age_days" yaml:"ageDays"`
	Commits           int64  `json:"commits" yaml:"commits"`
	UnverifiedCommits int64  `json:"unverified_commits" yaml:"unverifiedCommits"`
	PublicRepos       int64  `json:"public_repos,omitempty" yaml:"publicRepos,omitempty"`
	PrivateRepos      int64  `json:"private_repos,omitempty" yaml:"privateRepos,omitempty"`
	Followers         int64  `json:"followers,omitempty" yaml:"followers,omitempty"`
	Following         int64  `json:"following,omitempty" yaml:"following,omitempty"`
	LastCommitDays    int64  `json:"last_commit_days,omitempty" yaml:"lastCommitDays,omitempty"`
	OrgMember         bool   `json:"org_member,omitempty" yaml:"orgMember,omitempty"`

	// v3 fields
	AuthorAssociation string `json:"author_association,omitempty" yaml:"authorAssociation,omitempty"`
	HasBio            bool   `json:"has_bio,omitempty" yaml:"hasBio,omitempty"`
	HasLocation       bool   `json:"has_location,omitempty" yaml:"hasLocation,omitempty"`
	HasWebsite        bool   `json:"has_website,omitempty" yaml:"hasWebsite,omitempty"`
	PRsMerged         int64  `json:"prs_merged,omitempty" yaml:"prsMerged,omitempty"`
	PRsClosed         int64  `json:"prs_closed,omitempty" yaml:"prsClosed,omitempty"`
	RecentPRRepoCount int64  `json:"recent_pr_repo_count,omitempty" yaml:"recentPRRepoCount,omitempty"`
	ForkedRepos       int64  `json:"forked_repos,omitempty" yaml:"forkedRepos,omitempty"`
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./pkg/report/ -run TestMakeAuthorHasNewStatsFields -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pkg/report/author.go pkg/report/author_test.go
git commit -S -m "Add v3 signal fields to report.Stats"
```

---

### Task 5: Add GitHub API Fetch Functions

**Files:**
- Create: `pkg/provider/github/fetch.go`
- Create: `pkg/provider/github/fetch_test.go`

These functions fetch new signals from GitHub API. Each degrades gracefully on error (logs warning, returns zero values).

**Step 1: Write fetch.go**

```go
package github

import (
	"context"
	"fmt"
	"log/slog"

	hub "github.com/google/go-github/v72/github"
)

// prStats holds merged and closed-without-merge PR counts.
type prStats struct {
	Merged int64
	Closed int64 // closed without merge
}

// fetchPRStats returns global PR merge/close counts for a user.
// Uses GitHub search API: 2 calls per user.
func fetchPRStats(ctx context.Context, client *hub.Client, username string) prStats {
	var stats prStats

	// Merged PRs
	mergedResult, mergedResp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr is:merged", username),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search merged PRs for %s: %v", username, err))
		return stats
	}
	waitForRateLimit(mergedResp)
	if mergedResult.Total != nil {
		stats.Merged = int64(*mergedResult.Total)
	}

	// Closed without merge
	closedResult, closedResp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr is:unmerged is:closed", username),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search closed PRs for %s: %v", username, err))
		return stats
	}
	waitForRateLimit(closedResp)
	if closedResult.Total != nil {
		stats.Closed = int64(*closedResult.Total)
	}

	return stats
}

// fetchRecentPRRepoCount returns the number of distinct repos the user
// opened PRs in, based on recent public events (last ~90 days).
func fetchRecentPRRepoCount(ctx context.Context, client *hub.Client, username string) int64 {
	repos := make(map[string]struct{})
	page := 1

	for page <= 3 { // events API returns max 10 pages of 30, cap at 3 pages
		events, resp, err := client.Activity.ListEventsPerformedByUser(ctx, username, true,
			&hub.ListOptions{Page: page, PerPage: pageSize})
		if err != nil {
			slog.Debug(fmt.Sprintf("list events for %s page %d: %v", username, page, err))
			break
		}
		waitForRateLimit(resp)

		for _, e := range events {
			if e.GetType() == "PullRequestEvent" && e.Repo != nil {
				repos[e.Repo.GetName()] = struct{}{}
			}
		}

		if len(events) < pageSize {
			break
		}
		page++
	}

	return int64(len(repos))
}

// fetchForkedRepoCount returns the number of the user's owned repos that are forks.
func fetchForkedRepoCount(ctx context.Context, client *hub.Client, username string) int64 {
	var forked int64
	page := 1

	for page <= 3 { // cap at 3 pages (300 repos)
		repos, resp, err := client.Repositories.ListByUser(ctx, username,
			&hub.RepositoryListByUserOptions{
				Type:        "owner",
				ListOptions: hub.ListOptions{Page: page, PerPage: pageSize},
			})
		if err != nil {
			slog.Debug(fmt.Sprintf("list repos for %s page %d: %v", username, page, err))
			break
		}
		waitForRateLimit(resp)

		for _, r := range repos {
			if r.GetFork() {
				forked++
			}
		}

		if len(repos) < pageSize {
			break
		}
		page++
	}

	return forked
}

// fetchAuthorAssociation returns the author_association for a user in a repo.
// Queries the user's PRs in the target repo and reads the association from the first result.
func fetchAuthorAssociation(ctx context.Context, client *hub.Client, username, owner, repo string) string {
	result, resp, err := client.Search.Issues(ctx,
		fmt.Sprintf("author:%s type:pr repo:%s/%s", username, owner, repo),
		&hub.SearchOptions{ListOptions: hub.ListOptions{PerPage: 1}})
	if err != nil {
		slog.Debug(fmt.Sprintf("search PRs for %s in %s/%s: %v", username, owner, repo, err))
		return ""
	}
	waitForRateLimit(resp)

	if result.Total != nil && *result.Total > 0 && len(result.Issues) > 0 {
		return result.Issues[0].GetAuthorAssociation()
	}

	return ""
}
```

**Step 2: Write fetch_test.go (unit tests for the helper struct)**

```go
package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPRStatsZeroValue(t *testing.T) {
	var s prStats
	assert.Zero(t, s.Merged)
	assert.Zero(t, s.Closed)
}
```

Note: The fetch functions call live APIs so full tests require integration test setup. The scoring logic is tested independently in pkg/score. These functions degrade gracefully by design.

**Step 3: Run test**

Run: `go test ./pkg/provider/github/ -run TestPRStatsZeroValue -v`
Expected: PASS

**Step 4: Commit**

```bash
git add pkg/provider/github/fetch.go pkg/provider/github/fetch_test.go
git commit -S -m "Add GitHub API fetch functions for v3 signals"
```

---

### Task 6: Wire New Signals into loadAuthor and algo.go

**Files:**
- Modify: `pkg/provider/github/provider.go:142-205` (loadAuthor)
- Modify: `pkg/provider/github/algo.go` (calculateReputation)

**Step 1: Update loadAuthor to call new fetch functions**

In `provider.go`, the `loadAuthor` function needs to:
1. Extract profile completeness from existing `Users.Get` response
2. Call `fetchPRStats`, `fetchRecentPRRepoCount`, `fetchForkedRepoCount`, `fetchAuthorAssociation` concurrently
3. Populate new Stats fields

Replace `loadAuthor` in `pkg/provider/github/provider.go`:

```go
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int, owner string) error {
	if client == nil {
		return fmt.Errorf("client must be specified")
	}
	if a == nil {
		return fmt.Errorf("author must be specified")
	}

	u, r, err := client.Users.Get(ctx, a.Username)
	if err != nil {
		return fmt.Errorf("error getting user %s: %w", a.Username, err)
	}
	waitForRateLimit(r)

	slog.Debug("user",
		"page_next", r.NextPage,
		"page_last", r.LastPage,
		"status", r.StatusCode,
		"rate_limit", r.Rate.Limit,
		"rate_remaining", r.Rate.Remaining)

	a.Stats.Suspended = u.SuspendedAt != nil
	a.Stats.StrongAuth = u.GetTwoFactorAuthentication()
	a.Stats.CommitsVerified = a.Stats.UnverifiedCommits == 0
	a.Stats.Followers = int64(u.GetFollowers())
	a.Stats.Following = int64(u.GetFollowing())
	a.Stats.PublicRepos = int64(u.GetPublicRepos())
	a.Stats.PrivateRepos = u.GetTotalPrivateRepos()

	dc := u.GetCreatedAt().Time
	a.Stats.AgeDays = int64(math.Ceil(time.Now().UTC().Sub(dc).Hours() / hoursInDay))
	a.Context.Created = dc.Format(time.RFC3339)

	if u.Name != nil {
		a.Context.Name = u.GetName()
	}
	if u.Email != nil {
		a.Context.Email = u.GetEmail()
	}
	if u.Company != nil {
		a.Context.Company = u.GetCompany()
	}

	// Profile completeness (from existing Users.Get response)
	a.Stats.HasBio = u.Bio != nil && *u.Bio != ""
	a.Stats.HasLocation = u.Location != nil && *u.Location != ""
	a.Stats.HasWebsite = u.Blog != nil && *u.Blog != ""

	// HasCompany: use existing Company context field
	a.Stats.HasCompany = u.Company != nil && *u.Company != "" // Note: a.Context.Company may not reflect this if u.Company was nil above

	// Org membership check -- graceful degradation on error.
	isMember, orgResp, memberErr := client.Organizations.IsMember(ctx, owner, a.Username)
	waitForRateLimit(orgResp)
	if memberErr != nil {
		slog.Debug(fmt.Sprintf("org membership check [%s/%s]: %v", owner, a.Username, memberErr))
	} else {
		a.Stats.OrgMember = isMember
	}

	// Fetch v3 signals concurrently
	var (
		prStatsMu   sync.Mutex
		prResult    prStats
		recentCount int64
		forkedCount int64
		assocResult string
	)

	sg, sgctx := errgroup.WithContext(ctx)

	sg.Go(func() error {
		result := fetchPRStats(sgctx, client, a.Username)
		prStatsMu.Lock()
		prResult = result
		prStatsMu.Unlock()
		return nil
	})

	sg.Go(func() error {
		recentCount = fetchRecentPRRepoCount(sgctx, client, a.Username)
		return nil
	})

	sg.Go(func() error {
		forkedCount = fetchForkedRepoCount(sgctx, client, a.Username)
		return nil
	})

	sg.Go(func() error {
		assocResult = fetchAuthorAssociation(sgctx, client, a.Username, owner, q.Name)
		return nil
	})

	if err := sg.Wait(); err != nil {
		slog.Debug(fmt.Sprintf("v3 signal fetch error for %s: %v", a.Username, err))
	}

	a.Stats.PRsMerged = prResult.Merged
	a.Stats.PRsClosed = prResult.Closed
	a.Stats.RecentPRRepoCount = recentCount
	a.Stats.ForkedRepos = forkedCount
	a.Stats.AuthorAssociation = assocResult

	calculateReputation(a, totalCommits, totalContributors)

	if !stats {
		a.Stats = nil
		a.Context = nil
	}

	return nil
}
```

**Important:** `loadAuthor` needs access to `q.Name` for the `fetchAuthorAssociation` call. The function signature must be updated to accept the repo name:

Change signature from:
```go
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int, owner string) error {
```
to:
```go
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int, owner, repoName string) error {
```

And update the call site in `ListAuthors` from:
```go
if err := loadAuthor(gctx, client, a, q.Stats, totalCommitCounter, totalContributors, q.Owner); err != nil {
```
to:
```go
if err := loadAuthor(gctx, client, a, q.Stats, totalCommitCounter, totalContributors, q.Owner, q.Name); err != nil {
```

Also add `"sync"` and `"golang.org/x/sync/errgroup"` imports if not already present (they are already imported in this file).

**Step 2: Update calculateReputation in algo.go**

Replace `pkg/provider/github/algo.go`:

```go
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

		// v3 signals
		AuthorAssociation: s.AuthorAssociation,
		HasBio:            s.HasBio,
		HasCompany:        s.HasCompany,
		HasLocation:       s.HasLocation,
		HasWebsite:        s.HasWebsite,
		PRsMerged:         s.PRsMerged,
		PRsClosed:         s.PRsClosed,
		RecentPRRepoCount: s.RecentPRRepoCount,
		ForkedRepos:       s.ForkedRepos,
	})
}
```

**Step 3: Run all tests**

Run: `go test ./pkg/... -v -count=1`
Expected: ALL PASS

**Step 4: Commit**

```bash
git add pkg/provider/github/provider.go pkg/provider/github/algo.go
git commit -S -m "Wire v3 signal fetches into loadAuthor and scoring"
```

---

### Task 7: Update algo_test.go for v3

**Files:**
- Modify: `pkg/provider/github/algo_test.go`

**Step 1: Rewrite algo test cases for v3 weights**

All test cases need updated `wantScore` values since the weight model changed. Add new test cases for v3 signals:

```go
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
				Stats: &report.Stats{
					StrongAuth:        true,
					Commits:           50,
					UnverifiedCommits: 0,
					AgeDays:           730,
					OrgMember:         true,
					AuthorAssociation: "MEMBER",
					HasBio:            true,
					HasCompany:        true,
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
				Stats: &report.Stats{
					AgeDays:           14,
					PublicRepos:       5,
					ForkedRepos:       5,
					RecentPRRepoCount: 5,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.07,
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
```

**Step 2: Run tests**

Run: `go test ./pkg/provider/github/ -v`
Expected: ALL PASS

**Step 3: Commit**

```bash
git add pkg/provider/github/algo_test.go
git commit -S -m "Update algo tests for v3.0.0 scoring model"
```

---

### Task 8: Lint, Vet, and Full Test Pass

**Files:** None (validation only)

**Step 1: Run full test suite**

Run: `make test`
Expected: ALL PASS with race detector

**Step 2: Run linter**

Run: `make lint`
Expected: Clean

**Step 3: Fix any issues found**

Address lint warnings or test failures. Common issues:
- Unused imports after refactoring
- Float comparison tolerance adjustments in tests
- Missing error checks flagged by linter

**Step 4: Commit fixes if any**

```bash
git add -A
git commit -S -m "Fix lint and test issues for v3.0.0"
```

---

### Task 9: Final Verification and Cleanup

**Step 1: Run make build**

Run: `make build`
Expected: Clean build in `./dist`

**Step 2: Verify JSON output includes new fields**

If a `GITHUB_TOKEN` is available, run:
```bash
./dist/reputer --repo github/mchmarny/reputer --stats
```

Verify the JSON output includes new fields: `author_association`, `has_bio`, `prs_merged`, etc.

**Step 3: Commit any final adjustments**

```bash
git add -A
git commit -S -m "Final v3.0.0 cleanup and verification"
```
