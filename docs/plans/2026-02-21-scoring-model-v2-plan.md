# Scoring Model v2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the ad-hoc 6-signal scoring algorithm with a risk-weighted categorical model using non-linear response curves, composite signals, and dynamic repo-context parameters.

**Architecture:** The algo layer (`algo.go`) is rewritten with new curve functions, a composite provenance signal, and a wider `calculateReputation` signature. The data model gains two fields. The provider captures commit timestamps and adds an org membership API call. All existing tests are rewritten to match new weights and curves.

**Tech Stack:** Go, go-github/v52, testify/assert, logrus

**Design doc:** `docs/plans/2026-02-21-scoring-model-v2-design.md`

---

### Task 1: Add new Stats fields to the data model

**Files:**
- Modify: `pkg/report/author.go:41-52`

**Step 1: Write the failing test**

No new test file needed. Existing tests compile-check the struct. We verify the new fields exist by using them in Task 2 tests. Skip to implementation.

**Step 2: Add fields to Stats struct**

In `pkg/report/author.go`, add two fields to `Stats` after the `Following` field (line 51):

```go
type Stats struct {
	Suspended         bool  `json:"suspended,omitempty"`
	CommitsVerified   bool  `json:"verified_commits,omitempty"`
	StrongAuth        bool  `json:"strong_auth,omitempty"`
	AgeDays           int64 `json:"age_days"`
	Commits           int64 `json:"commits"`
	UnverifiedCommits int64 `json:"unverified_commits"`
	PublicRepos       int64 `json:"public_repos,omitempty"`
	PrivateRepos      int64 `json:"private_repos,omitempty"`
	Followers         int64 `json:"followers,omitempty"`
	Following         int64 `json:"following,omitempty"`
	LastCommitDays    int64 `json:"last_commit_days,omitempty"`
	OrgMember         bool  `json:"org_member,omitempty"`
}
```

**Step 3: Run existing tests to verify no breakage**

Run: `make test`
Expected: PASS (new fields are zero-value by default)

**Step 4: Commit**

```bash
git add pkg/report/author.go
git commit -S -m "add LastCommitDays and OrgMember fields to Stats"
```

---

### Task 2: Add model metadata to Report

**Files:**
- Modify: `pkg/report/report.go:1-27`

**Step 1: Add metadata types and field**

In `pkg/report/report.go`, add the metadata types and a `Meta` field to `Report`:

```go
package report

import (
	"sort"
	"time"
)

const ModelVersion = "2.0.0"

// CategoryWeight describes a scoring category and its weight.
type CategoryWeight struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

// ReportMeta holds scoring model metadata.
type ReportMeta struct {
	ModelVersion string           `json:"model_version"`
	Categories   []CategoryWeight `json:"categories"`
}

// Report is the top-level output for a reputation query.
type Report struct {
	Repo              string      `json:"repo,omitempty"`
	AtCommit          string      `json:"at_commit,omitempty"`
	GeneratedOn       time.Time   `json:"generated_on,omitempty"`
	TotalCommits      int64       `json:"total_commits,omitempty"`
	TotalContributors int64       `json:"total_contributors,omitempty"`
	Meta              *ReportMeta `json:"meta,omitempty"`
	Contributors      []*Author   `json:"contributors,omitempty"`
}

// SortAuthors sorts the authors by username.
func (r *Report) SortAuthors() {
	if r.Contributors == nil {
		return
	}

	sort.Slice(r.Contributors, func(i, j int) bool {
		return r.Contributors[i].Username < r.Contributors[j].Username
	})
}
```

**Step 2: Run existing tests to verify no breakage**

Run: `make test`
Expected: PASS (Meta is omitempty, existing code doesn't set it)

**Step 3: Commit**

```bash
git add pkg/report/report.go
git commit -S -m "add model metadata to Report struct"
```

---

### Task 3: Implement response curve functions

**Files:**
- Modify: `pkg/provider/github/algo.go:1-34`
- Modify: `pkg/provider/github/algo_test.go:17-39`

**Step 1: Write failing tests for logCurve and expDecay**

In `pkg/provider/github/algo_test.go`, add two new test functions after `TestClampedRatio` (after line 39):

```go
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
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/mchmarny/dev/reputer && go test ./pkg/provider/github/ -run "TestLogCurve|TestExpDecay" -v`
Expected: FAIL -- `logCurve` and `expDecay` undefined

**Step 3: Implement the curve functions**

In `pkg/provider/github/algo.go`, add after the `clampedRatio` function (after line 34):

```go
// logCurve maps val into [0.0, 1.0] with logarithmic diminishing returns.
// f(v, c) = log(1+v) / log(1+c), clamped to [0, 1].
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
// f(v, h) = exp(-v * ln2 / h). Returns 1.0 at v=0, 0.5 at v=halfLife.
func expDecay(val, halfLife float64) float64 {
	if halfLife <= 0 {
		return 0
	}
	if val <= 0 {
		return 1
	}
	return math.Exp(-val * math.Ln2 / halfLife)
}
```

Add `"math"` to the imports in `algo.go`.

**Step 4: Run tests to verify they pass**

Run: `cd /Users/mchmarny/dev/reputer && go test ./pkg/provider/github/ -run "TestLogCurve|TestExpDecay" -v`
Expected: PASS

**Step 5: Run full test suite**

Run: `make test`
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/provider/github/algo.go pkg/provider/github/algo_test.go
git commit -S -m "add logCurve and expDecay response curve functions"
```

---

### Task 4: Rewrite scoring algorithm with new model

This is the core task. Replace all weight constants, rewrite `calculateReputation` with the new composite provenance signal, new curves, and the wider signature.

**Files:**
- Modify: `pkg/provider/github/algo.go` (full rewrite of constants and `calculateReputation`)
- Modify: `pkg/provider/github/algo_test.go` (full rewrite of `TestCalculateReputation`)

**Step 1: Write the failing tests**

Replace the entire `TestCalculateReputation` function in `algo_test.go` (lines 42-307). The new signature is `calculateReputation(author, totalCommits, totalContributors)`.

Key test scenarios (compute expected values from the design doc formulas):

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
			name:   "nil author",
			author: nil,
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
			wantScore:         0,
		},
		{
			// Provenance: 2FA + 100% signed (core, 50/100 commits in 5-person repo)
			//   verifiedRatio = 1.0, multiplier = 1.0, score = 1.0 * 0.35 = 0.35
			// Identity: age 730d -> logCurve(730,730)=1.0 * 0.15 = 0.15
			//   orgMember: true -> 0.10
			// Engagement: proportion = 50/100 = 0.5, ceiling = max(1/5, 0.05) = 0.20
			//   confidence = min(100 / max(5*10,30), 1) = min(100/50, 1) = 1.0
			//   linear(0.5, 0.20) = clamped to 1.0 * 1.0 * 0.15 = 0.15
			//   recency: 0 days, halfLife = 90 * max(1/log(6), 0.25) = 90 * 0.558 = 50.2
			//   expDecay(0, 50.2) = 1.0 * 0.10 = 0.10
			// Community: followers=100, following=10 -> ratio=10
			//   logCurve(10, 10) = 1.0 * 0.10 = 0.10
			//   repos = 30+15 = 45, logCurve(45, 30) = clamped 1.0 * 0.05 = 0.05
			// Total: 0.35 + 0.15 + 0.10 + 0.15 + 0.10 + 0.10 + 0.05 = 1.00
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
			// Provenance: 2FA + 0% signed -> floor 0.1 * 0.35 = 0.035
			// All other signals zero.
			// Total: 0.04 (rounded from 0.035)
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
			wantScore:         0.04,
		},
		{
			// Provenance: no 2FA, 100% signed, core contributor (10/20 commits, 2 contributors)
			//   proportion = 10/20 = 0.5, ceiling = max(1/2, 0.05) = 0.5
			//   multiplier = 1.0 - 0.5 * linear(0.5, 0.5) = 1.0 - 0.5 * 1.0 = 0.50
			//   rawScore = 1.0 * 0.50 = 0.50
			//   score = 0.50 * 0.35 = 0.175
			// Everything else zero.
			// Total: 0.18 (rounded from 0.175)
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
			wantScore:         0.18,
		},
		{
			// Provenance: no 2FA, 100% signed, peripheral (1/100 commits, 20 contributors)
			//   proportion = 1/100 = 0.01, ceiling = max(1/20, 0.05) = 0.05
			//   multiplier = 1.0 - 0.5 * linear(0.01, 0.05) = 1.0 - 0.5 * 0.2 = 0.90
			//   rawScore = 1.0 * 0.90 = 0.90
			//   score = 0.90 * 0.35 = 0.315
			// Everything else zero.
			// Total: 0.32 (rounded from 0.315)
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
			wantScore:         0.32,
		},
		{
			// Identity only: age 30 days
			// logCurve(30, 730) = log(31)/log(731) = 3.434/6.595 = 0.521
			// 0.521 * 0.15 = 0.0781
			// Total: 0.08
			name: "age 30 days only",
			author: &report.Author{
				Username: "age-30",
				Stats: &report.Stats{
					AgeDays: 30,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.08,
		},
		{
			// Identity only: age 365 days
			// logCurve(365, 730) = log(366)/log(731) = 5.903/6.595 = 0.895
			// 0.895 * 0.15 = 0.1343
			// Total: 0.13
			name: "age 365 days only",
			author: &report.Author{
				Username: "age-365",
				Stats: &report.Stats{
					AgeDays: 365,
				},
			},
			totalCommits:      100,
			totalContributors: 10,
			wantScore:         0.13,
		},
		{
			// Community: follower ratio 3:1
			// logCurve(3, 10) = log(4)/log(11) = 1.386/2.398 = 0.578
			// 0.578 * 0.10 = 0.0578
			// Total: 0.06
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
			wantScore:         0.06,
		},
		{
			// Community: repos combined = 5
			// logCurve(5, 30) = log(6)/log(31) = 1.791/3.434 = 0.521
			// 0.521 * 0.05 = 0.026
			// Total: 0.03
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
			wantScore:         0.03,
		},
		{
			// Engagement: proportion = 10/100 = 0.10, ceiling = max(1/10, 0.05) = 0.10
			//   confidence = min(100/max(10*10,30), 1) = min(100/100, 1) = 1.0
			//   linear(0.10, 0.10) = 1.0 * 1.0 * 0.15 = 0.15
			// Recency: 0 days, halfLife = 90 * max(1/log(11), 0.25) = 90 * 0.417 = 37.5
			//   expDecay(0, 37.5) = 1.0 * 0.10 = 0.10
			// Total: 0.25
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
			wantScore:         0.25,
		},
		{
			// Engagement: low confidence repo (5 total commits, 3 contributors)
			//   proportion = 3/5 = 0.6, ceiling = max(1/3, 0.05) = 0.333
			//   confidenceThreshold = max(3*10, 30) = 30
			//   confidence = min(5/30, 1) = 0.167
			//   linear(0.6, 0.333) = clamped 1.0 * 0.167 * 0.15 = 0.025
			// Recency: 0 days -> 0.10
			// Total: 0.13 (0.025 + 0.10 = 0.125)
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
			wantScore:         0.13,
		},
		{
			// Recency: 90 days, solo repo (halfLife = 90 * max(1/log(2), 0.25) = 90*1.0=90, capped to 1.0 mult so 90)
			//   Actually: 1/log(1+1) = 1/log(2) = 1/0.693 = 1.443, max(1.443, 0.25)=1.443
			//   But we cap multiplier at 1.0, so halfLife = 90 * 1.0 = 90?
			//   No -- the formula is max(1/log(1+n), 0.25) with no upper cap, so halfLife = 90 * 1.443 = 129.9
			//   expDecay(90, 129.9) = exp(-90 * 0.693 / 129.9) = exp(-0.480) = 0.619
			//   0.619 * 0.10 = 0.062
			//   proportion = 5/10 = 0.5, ceiling = max(1/1, 0.05) = 1.0
			//   confidence = min(10/max(1*10,30), 1) = min(10/30, 1) = 0.333
			//   linear(0.5, 1.0) = 0.5 * 0.333 * 0.15 = 0.025
			// Total: 0.09 (0.025 + 0.062 = 0.087)
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
			wantScore:         0.09,
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
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/mchmarny/dev/reputer && go test ./pkg/provider/github/ -run TestCalculateReputation -v`
Expected: FAIL -- wrong number of arguments to `calculateReputation`

**Step 3: Rewrite the algorithm**

Replace the entire contents of `pkg/provider/github/algo.go`:

```go
// Package github implements the GitHub reputation provider.
package github

import (
	"math"

	"github.com/mchmarny/reputer/pkg/report"
	log "github.com/sirupsen/logrus"
)

const (
	// Category weights (sum to 1.0).
	// Derived from threat-model ranking: Provenance > Identity > Engagement > Community.
	provenanceWeight  = 0.35
	ageWeight         = 0.15
	orgMemberWeight   = 0.10
	proportionWeight  = 0.15
	recencyWeight     = 0.10
	followerWeight    = 0.10
	repoCountWeight   = 0.05

	// Ceilings and parameters.
	ageCeilDays         = 730
	followerRatioCeil   = 10.0
	repoCountCeil       = 30.0
	baseHalfLifeDays    = 90.0
	minHalfLifeMultiple = 0.25
	provenanceFloor     = 0.1
	minProportionCeil   = 0.05
	minConfidenceCommits = 30
	confCommitsPerContrib = 10
	noTFAMaxDiscount    = 0.5
)

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

// calculateReputation scores an author using the v2 risk-weighted categorical model.
func calculateReputation(author *report.Author, totalCommits int64, totalContributors int) {
	if author == nil {
		return
	}

	s := author.Stats
	if s == nil {
		log.Debugf("score [%s] - no stats", author.Username)
		return
	}

	if s.Suspended {
		log.Debugf("score [%s] - suspended", author.Username)
		return
	}

	var rep float64

	// --- Category 1: Code Provenance (0.35) ---
	if s.Commits > 0 && totalCommits > 0 {
		verifiedRatio := float64(s.Commits-s.UnverifiedCommits) / float64(s.Commits)
		proportion := float64(s.Commits) / float64(totalCommits)
		propCeil := math.Max(1.0/float64(max(totalContributors, 1)), minProportionCeil)

		securityMultiplier := 1.0
		if !s.StrongAuth {
			securityMultiplier = 1.0 - noTFAMaxDiscount*clampedRatio(proportion, propCeil)
		}

		rawScore := verifiedRatio * securityMultiplier
		if s.StrongAuth && rawScore < provenanceFloor {
			rawScore = provenanceFloor
		}

		rep += rawScore * provenanceWeight
		log.Debugf("provenance [%s]: %.4f (verified=%.2f, multiplier=%.2f)", author.Username, rep, verifiedRatio, securityMultiplier)
	} else if s.StrongAuth {
		rep += provenanceFloor * provenanceWeight
		log.Debugf("provenance [%s]: %.4f (2FA floor, no commits)", author.Username, rep)
	}

	// --- Category 2: Identity Authenticity (0.25) ---
	rep += logCurve(float64(s.AgeDays), ageCeilDays) * ageWeight
	log.Debugf("age [%s]: %.4f (%d days)", author.Username, rep, s.AgeDays)

	if s.OrgMember {
		rep += orgMemberWeight
	}
	log.Debugf("org [%s]: %.4f (member=%v)", author.Username, rep, s.OrgMember)

	// --- Category 3: Engagement Depth (0.25) ---
	if s.Commits > 0 && totalCommits > 0 {
		proportion := float64(s.Commits) / float64(totalCommits)
		propCeil := math.Max(1.0/float64(max(totalContributors, 1)), minProportionCeil)

		confThreshold := float64(max(totalContributors*confCommitsPerContrib, minConfidenceCommits))
		confidence := math.Min(float64(totalCommits)/confThreshold, 1.0)

		rep += clampedRatio(proportion, propCeil) * confidence * proportionWeight
		log.Debugf("proportion [%s]: %.4f (prop=%.3f, ceil=%.3f, conf=%.3f)", author.Username, rep, proportion, propCeil, confidence)
	}

	numContrib := max(totalContributors, 1)
	halfLifeMult := math.Max(1.0/math.Log(1+float64(numContrib)), minHalfLifeMultiple)
	if halfLifeMult > 1.0 {
		halfLifeMult = 1.0
	}
	halfLife := baseHalfLifeDays * halfLifeMult
	rep += expDecay(float64(s.LastCommitDays), halfLife) * recencyWeight
	log.Debugf("recency [%s]: %.4f (%d days, halfLife=%.1f)", author.Username, rep, s.LastCommitDays, halfLife)

	// --- Category 4: Community Standing (0.15) ---
	if s.Following > 0 {
		ratio := float64(s.Followers) / float64(s.Following)
		rep += logCurve(ratio, followerRatioCeil) * followerWeight
		log.Debugf("followers [%s]: %.4f (ratio=%.2f)", author.Username, rep, ratio)
	}

	totalRepos := float64(s.PublicRepos + s.PrivateRepos)
	rep += logCurve(totalRepos, repoCountCeil) * repoCountWeight
	log.Debugf("repos [%s]: %.4f (%d combined)", author.Username, rep, int64(totalRepos))

	author.Reputation = report.ToFixed(rep, 2)
	log.Debugf("reputation [%s]: %.2f", author.Username, author.Reputation)
}
```

**Step 4: Run tests**

Run: `cd /Users/mchmarny/dev/reputer && go test ./pkg/provider/github/ -run TestCalculateReputation -v`
Expected: PASS (all scenarios within delta 0.01)

If any scenario fails, recalculate the expected value by hand using the exact formulas and adjust the test expectation. The formulas in the design doc are the source of truth.

**Step 5: Run full test suite + lint**

Run: `make test && make lint`
Expected: PASS. The `provider.go` call to `calculateReputation(a)` at line 171 will now fail to compile because the signature changed. That's expected -- Task 5 fixes it.

Actually, since `provider.go` calls `calculateReputation`, the build will fail. We need to temporarily update the call site. Add a TODO-style shim in provider.go to keep things compiling:

In `pkg/provider/github/provider.go` line 171, change:
```go
calculateReputation(a)
```
to:
```go
calculateReputation(a, 0, 0)
```

This is a temporary placeholder -- Task 5 will pass the real values.

**Step 6: Run full suite again**

Run: `make test && make lint`
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/provider/github/algo.go pkg/provider/github/algo_test.go pkg/provider/github/provider.go
git commit -S -m "rewrite scoring algorithm with v2 risk-weighted categorical model"
```

---

### Task 5: Capture commit timestamps and wire up totals in provider

**Files:**
- Modify: `pkg/provider/github/provider.go`

**Step 1: Write the failing test**

This task modifies the integration-level provider code. The scoring is already unit-tested. We verify correctness by running the full suite with real totals wired through. No new test file needed.

**Step 2: Update ListAuthors to capture commit timestamps**

In the commit pagination loop (lines 64-80 of `provider.go`), capture the most recent commit date per author:

Replace the inner loop body:
```go
		for _, c := range p {
			if c.Author == nil {
				continue
			}

			login := c.GetAuthor().GetLogin()
			if _, ok := list[login]; !ok {
				list[login] = report.MakeAuthor(login)
			}

			list[login].Stats.Commits++
			if !*c.GetCommit().GetVerification().Verified {
				list[login].Stats.UnverifiedCommits++
			}

			totalCommitCounter++
		}
```

With:
```go
		for _, c := range p {
			if c.Author == nil {
				continue
			}

			login := c.GetAuthor().GetLogin()
			if _, ok := list[login]; !ok {
				list[login] = report.MakeAuthor(login)
			}

			list[login].Stats.Commits++
			if !*c.GetCommit().GetVerification().Verified {
				list[login].Stats.UnverifiedCommits++
			}

			// Track most recent commit date per author (commits arrive newest-first).
			if list[login].Stats.LastCommitDays == 0 {
				if cd := c.GetCommit().GetCommitter().GetDate(); !cd.IsZero() {
					days := int64(math.Ceil(time.Now().UTC().Sub(cd.Time).Hours() / hoursInDay))
					if days < 0 {
						days = 0
					}
					list[login].Stats.LastCommitDays = days
				}
			}

			totalCommitCounter++
		}
```

Note: GitHub returns commits newest-first by default, so the first commit we see per author is their most recent.

**Step 3: Update loadAuthor to pass totals to calculateReputation**

Change the `loadAuthor` signature to accept totals:

```go
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int) error {
```

Replace the `calculateReputation` call (previously the temporary shim):
```go
	calculateReputation(a, totalCommits, totalContributors)
```

**Step 4: Update the errgroup call site in ListAuthors**

Replace the errgroup loop (lines 102-113) to pass totals:

```go
	totalContributors := len(list)

	for _, a := range list {
		a := a
		g.Go(func() error {
			if err := loadAuthor(gctx, client, a, q.Stats, totalCommitCounter, totalContributors); err != nil {
				return err
			}
			mu.Lock()
			authors = append(authors, a)
			mu.Unlock()
			return nil
		})
	}
```

And update the report assignment (line 119):
```go
	rpt.TotalContributors = int64(totalContributors)
```

**Step 5: Add model metadata to report**

After the `rpt` creation (after line 94), add:

```go
	rpt.Meta = &report.ReportMeta{
		ModelVersion: report.ModelVersion,
		Categories: []report.CategoryWeight{
			{Name: "code_provenance", Weight: provenanceWeight},
			{Name: "identity", Weight: ageWeight + orgMemberWeight},
			{Name: "engagement", Weight: proportionWeight + recencyWeight},
			{Name: "community", Weight: followerWeight + repoCountWeight},
		},
	}
```

**Step 6: Run full suite**

Run: `make test && make lint`
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/provider/github/provider.go
git commit -S -m "capture commit timestamps and wire scoring totals through provider"
```

---

### Task 6: Add org membership check (Phase 2)

**Files:**
- Modify: `pkg/provider/github/provider.go`

**Step 1: Add org membership check to loadAuthor**

In `loadAuthor`, after the existing user data extraction and before `calculateReputation`, add:

```go
	// Org membership check -- graceful degradation on error.
	isMember, _, memberErr := client.Organizations.IsMember(ctx, q.Owner, a.Username)
	if memberErr != nil {
		log.Debugf("org membership check [%s/%s]: %v", q.Owner, a.Username, memberErr)
	} else {
		a.Stats.OrgMember = isMember
	}
```

This requires passing `q.Owner` into `loadAuthor`. Update the signature:

```go
func loadAuthor(ctx context.Context, client *hub.Client, a *report.Author, stats bool, totalCommits int64, totalContributors int, owner string) error {
```

Update the call site in `ListAuthors`:

```go
			if err := loadAuthor(gctx, client, a, q.Stats, totalCommitCounter, totalContributors, q.Owner); err != nil {
```

**Step 2: Run full suite**

Run: `make test && make lint`
Expected: PASS

**Step 3: Commit**

```bash
git add pkg/provider/github/provider.go
git commit -S -m "add org membership check with graceful degradation"
```

---

### Task 7: Final validation

**Step 1: Run full quality gate**

Run: `make snapshot`
Expected: PASS (runs test + lint + goreleaser build)

**Step 2: Verify JSON output includes model metadata**

Run: `go run ./cmd/ --repo github.com/mchmarny/reputer --stats 2>/dev/null | head -20`
Expected: Output includes `"meta"` field with `model_version` and `categories`.

Note: This requires a valid `GITHUB_TOKEN` env var. If not available, verify by reading the code path instead.

**Step 3: Commit any final fixes**

If any fixes were needed, commit them.

---

### Task 8: Update design doc with final implementation notes

**Files:**
- Modify: `docs/plans/2026-02-21-scoring-model-v2-design.md`

**Step 1: Add implementation notes**

Append to the design doc:

```markdown
## Implementation Notes

- Half-life multiplier capped at 1.0 on the upper end (solo repos don't get
  extended half-lives beyond 90 days).
- Commit timestamps use committer date (not author date) for recency, as
  committer date reflects when the commit entered the branch.
- GitHub returns commits newest-first, so the first commit seen per author in
  the pagination loop is their most recent.
- Org membership uses `Organizations.IsMember` which requires the token to
  have `read:org` scope. Falls back to false on any error.
```

**Step 2: Commit**

```bash
git add docs/plans/2026-02-21-scoring-model-v2-design.md
git commit -S -m "add implementation notes to scoring model v2 design"
```
