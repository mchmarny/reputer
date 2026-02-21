# Extract Scoring Library Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Extract the reputation scoring logic into a standalone `pkg/score` package with no provider dependencies.

**Architecture:** New `pkg/score` package owns the scoring model (constants, helpers, `Compute()`, `Categories()`). Existing `pkg/provider/github` becomes a thin adapter that builds `score.Signals` from API data and delegates. `pkg/report` re-exports `ModelVersion` and type-aliases `CategoryWeight` for backward compat.

**Tech Stack:** Go 1.26, `math`, `log/slog`, `testify/assert`

---

### Task 1: Create `pkg/score/score_test.go` (failing tests)

**Files:**
- Create: `pkg/score/score_test.go`

**Step 1: Write the test file**

This file contains every test that will validate the scoring package: helper functions, weight invariants, and full `Compute()` scenarios (ported from `pkg/provider/github/algo_test.go` plus new edge cases).

```go
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
			wantScore: 0.10, // recency only: expDecay(0,*)=1.0 * 0.10
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
			wantScore: 0.20, // 0.10 org + 0.10 recency
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
```

**Step 2: Run test to verify it fails**

Run: `go test ./pkg/score/ -v -count=1 2>&1 | head -20`
Expected: Compilation error — package `score` does not exist yet.

**Step 3: Commit**

```bash
git add pkg/score/score_test.go
git commit -S -m "add failing tests for pkg/score scoring library"
```

---

### Task 2: Create `pkg/score/score.go` (make tests pass)

**Files:**
- Create: `pkg/score/score.go`

**Step 1: Write the implementation**

Move all scoring logic from `pkg/provider/github/algo.go` into this new file. The `Compute` function takes a `Signals` struct instead of `*report.Author` + context args.

```go
package score

import (
	"fmt"
	"log/slog"
	"math"
)

// ModelVersion is the current scoring model version.
const ModelVersion = "2.0.0"

const (
	// Category weights (sum to 1.0).
	provenanceWeight = 0.35
	ageWeight        = 0.15
	orgMemberWeight  = 0.10
	proportionWeight = 0.15
	recencyWeight    = 0.10
	followerWeight   = 0.10
	repoCountWeight  = 0.05

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
)

// Exported category weights derived from signal constants above.
var (
	CategoryProvenanceWeight = provenanceWeight
	CategoryIdentityWeight   = ageWeight + orgMemberWeight
	CategoryEngagementWeight = proportionWeight + recencyWeight
	CategoryCommunityWeight  = followerWeight + repoCountWeight
)

// CategoryWeight describes a scoring category and its weight.
type CategoryWeight struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// Signals holds the raw inputs to the reputation model.
type Signals struct {
	Suspended         bool  // Account suspended
	StrongAuth        bool  // 2FA enabled
	Commits           int64 // Author's commit count
	UnverifiedCommits int64 // Commits without verified signatures
	TotalCommits      int64 // Repo-wide total commits
	TotalContributors int   // Repo-wide contributor count
	AgeDays           int64 // Days since account creation
	OrgMember         bool  // Member of repo owner's org
	LastCommitDays    int64 // Days since most recent commit
	Followers         int64 // GitHub follower count
	Following         int64 // Following count (for ratio)
	PublicRepos       int64 // Public repository count
	PrivateRepos      int64 // Private repository count
}

// Categories returns the model's scoring categories with their weights.
func Categories() []CategoryWeight {
	return []CategoryWeight{
		{Name: "code_provenance", Weight: CategoryProvenanceWeight},
		{Name: "identity", Weight: CategoryIdentityWeight},
		{Name: "engagement", Weight: CategoryEngagementWeight},
		{Name: "community", Weight: CategoryCommunityWeight},
	}
}

// Compute returns a reputation score in [0.0, 1.0] using the v2 weighted model.
func Compute(s Signals) float64 {
	if s.Suspended {
		slog.Debug("score - suspended")
		return 0
	}

	var rep float64

	// --- Category 1: Code Provenance (0.35) ---
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
			rep, verifiedRatio, securityMultiplier))
	} else if s.StrongAuth {
		rep += provenanceFloor * provenanceWeight
		slog.Debug(fmt.Sprintf("provenance: %.4f (2FA floor, no commits)", rep))
	}

	// --- Category 2: Identity Authenticity (0.25) ---
	rep += logCurve(float64(s.AgeDays), ageCeilDays) * ageWeight
	slog.Debug(fmt.Sprintf("age: %.4f (%d days)", rep, s.AgeDays))

	if s.OrgMember {
		rep += orgMemberWeight
	}
	slog.Debug(fmt.Sprintf("org: %.4f (member=%v)", rep, s.OrgMember))

	// --- Category 3: Engagement Depth (0.25) ---
	if s.Commits > 0 && s.TotalCommits > 0 {
		proportion := float64(s.Commits) / float64(s.TotalCommits)
		propCeil := math.Max(1.0/float64(max(s.TotalContributors, 1)), minProportionCeil)

		confThreshold := float64(max(
			int64(s.TotalContributors)*int64(confCommitsPerContrib),
			int64(minConfidenceCommits),
		))
		confidence := math.Min(float64(s.TotalCommits)/confThreshold, 1.0)

		rep += clampedRatio(proportion, propCeil) * confidence * proportionWeight
		slog.Debug(fmt.Sprintf("proportion: %.4f (prop=%.3f, ceil=%.3f, conf=%.3f)",
			rep, proportion, propCeil, confidence))
	}

	numContrib := max(s.TotalContributors, 1)
	halfLifeMult := math.Max(1.0/math.Log(1+float64(numContrib)), minHalfLifeMultiple)
	if halfLifeMult > 1.0 {
		halfLifeMult = 1.0
	}
	halfLife := baseHalfLifeDays * halfLifeMult
	rep += expDecay(float64(s.LastCommitDays), halfLife) * recencyWeight
	slog.Debug(fmt.Sprintf("recency: %.4f (%d days, halfLife=%.1f)",
		rep, s.LastCommitDays, halfLife))

	// --- Category 4: Community Standing (0.15) ---
	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		rep += logCurve(ratio, followerRatioCeil) * followerWeight
		slog.Debug(fmt.Sprintf("followers: %.4f (ratio=%.2f)", rep, ratio))
	}

	totalRepos := float64(s.PublicRepos + s.PrivateRepos)
	rep += logCurve(totalRepos, repoCountCeil) * repoCountWeight
	slog.Debug(fmt.Sprintf("repos: %.4f (%d combined)", rep, int64(totalRepos)))

	result := toFixed(rep, 2)
	slog.Debug(fmt.Sprintf("reputation: %.2f", result))
	return result
}

// clampedRatio maps val linearly into [0.0, 1.0] with ceil as the saturation point.
func clampedRatio(val, ceil float64) float64 {
	if ceil <= 0 || val <= 0 {
		return 0
	}
	if val >= ceil {
		return 1
	}
	return val / ceil
}

// logCurve maps val into [0.0, 1.0] with logarithmic diminishing returns.
func logCurve(val, ceil float64) float64 {
	if ceil <= 0 || val <= 0 {
		return 0
	}
	r := math.Log(1+val) / math.Log(1+ceil)
	if r >= 1 {
		return 1
	}
	return r
}

// expDecay models freshness with a half-life.
func expDecay(val, halfLife float64) float64 {
	if halfLife <= 0 {
		return 0
	}
	if val <= 0 {
		return 1
	}
	return math.Exp(-val * math.Ln2 / halfLife)
}

// toFixed truncates a float64 to the given precision.
func toFixed(num float64, precision int) float64 {
	output := math.Pow(10, float64(precision))
	return float64(int(math.Round(num*output))) / output
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./pkg/score/ -v -race -count=1`
Expected: All tests PASS.

**Step 3: Commit**

```bash
git add pkg/score/score.go
git commit -S -m "add pkg/score: standalone reputation scoring library"
```

---

### Task 3: Update `pkg/report/report.go` for backward compat

**Files:**
- Modify: `pkg/report/report.go`

**Step 1: Write the failing test**

Existing tests already cover `report.ModelVersion` usage. The change is mechanical — verify compilation and existing tests still pass.

**Step 2: Apply changes**

Replace the `ModelVersion` const and `CategoryWeight` type with delegations to `pkg/score`:

In `pkg/report/report.go`:

1. Add import: `"github.com/mchmarny/reputer/pkg/score"`
2. Replace `const ModelVersion = "2.0.0"` with `var ModelVersion = score.ModelVersion`
3. Replace the `CategoryWeight` struct definition with `type CategoryWeight = score.CategoryWeight`

The `Meta` struct stays unchanged — it already references `CategoryWeight` which is now an alias.

**Step 3: Run tests to verify they pass**

Run: `go test ./pkg/report/ -v -race -count=1`
Expected: All tests PASS (report tests don't directly test `CategoryWeight` fields beyond JSON, which aliases preserve).

**Step 4: Commit**

```bash
git add pkg/report/report.go
git commit -S -m "delegate report.ModelVersion and CategoryWeight to pkg/score"
```

---

### Task 4: Replace `pkg/provider/github/algo.go` with thin adapter

**Files:**
- Modify: `pkg/provider/github/algo.go`

**Step 1: Rewrite algo.go as adapter**

Replace the entire file with a thin adapter that builds `score.Signals` from `*report.Author` and calls `score.Compute()`:

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
	})
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./pkg/provider/github/ -v -race -count=1 -run TestCalculateReputation`
Expected: All 14 test cases PASS with identical scores.

**Step 3: Commit**

```bash
git add pkg/provider/github/algo.go
git commit -S -m "replace scoring logic in github adapter with score.Compute delegation"
```

---

### Task 5: Update `pkg/provider/github/algo_test.go`

**Files:**
- Modify: `pkg/provider/github/algo_test.go`

**Step 1: Remove tests that moved to pkg/score**

Remove `TestCategoryWeightsSum`, `TestClampedRatio`, `TestLogCurve`, `TestExpDecay` — these now live in `pkg/score/score_test.go`. Keep `TestCalculateReputation` — it validates the adapter builds `Signals` correctly from `report.Author`/`report.Stats`. Remove the `TestMain` function if it only existed for debug logging in helper tests (check if `TestCalculateReputation` still needs it; if so, keep it).

The remaining file should contain only `TestMain` (if needed) and `TestCalculateReputation`.

**Step 2: Run tests to verify they pass**

Run: `go test ./pkg/provider/github/ -v -race -count=1`
Expected: `TestCalculateReputation` PASS.

**Step 3: Commit**

```bash
git add pkg/provider/github/algo_test.go
git commit -S -m "remove scoring math tests moved to pkg/score"
```

---

### Task 6: Update `pkg/provider/github/provider.go` to use `score.Categories()`

**Files:**
- Modify: `pkg/provider/github/provider.go`

**Step 1: Apply changes**

In `pkg/provider/github/provider.go`:

1. Add import: `"github.com/mchmarny/reputer/pkg/score"`
2. Replace the Meta population block (lines 106-114) with:

```go
	rpt.Meta = &report.Meta{
		ModelVersion: score.ModelVersion,
		Categories:   score.Categories(),
	}
```

This removes the dependency on the `Category*Weight` vars that used to be exported from this package.

**Step 2: Run tests to verify they pass**

Run: `go test ./pkg/provider/github/ -v -race -count=1`
Expected: PASS.

**Step 3: Run full test suite**

Run: `go test ./... -race -count=1`
Expected: All packages PASS.

**Step 4: Commit**

```bash
git add pkg/provider/github/provider.go
git commit -S -m "use score.Categories() for report metadata in github provider"
```

---

### Task 7: Vendor, lint, and final validation

**Files:**
- Modify: `vendor/` (automatic)

**Step 1: Tidy and vendor**

Run: `make tidy`
Expected: Clean exit. `pkg/score` has no external deps so vendor won't change.

**Step 2: Run linter**

Run: `make lint`
Expected: Clean exit.

**Step 3: Run full test suite with coverage**

Run: `make test`
Expected: All tests pass with race detector. Coverage includes `pkg/score`.

**Step 4: Run snapshot build**

Run: `make snapshot`
Expected: Binary builds successfully.

**Step 5: Commit vendor changes if any**

```bash
git add -A vendor/
git commit -S -m "vendor: update after pkg/score extraction"
```

(Skip if no vendor changes.)
