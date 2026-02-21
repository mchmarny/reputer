# Extract Scoring Library — Design

## Problem

The reputation scoring logic is embedded in `pkg/provider/github/algo.go`. Consumers
with pre-collected contributor data cannot compute scores without running the full
commit-enumeration pipeline.

## Decision: Faithful Extraction into `pkg/score`

Create a new `pkg/score` package containing the scoring model with zero provider
dependencies. The existing GitHub provider becomes a thin adapter that builds
`score.Signals` from API data and delegates to `score.Compute()`.

## API Surface

```go
package score

const ModelVersion = "2.0.0"

type Signals struct {
    Suspended         bool
    StrongAuth        bool    // 2FA enabled
    Commits           int64   // Author's commit count
    UnverifiedCommits int64   // Unsigned commits
    TotalCommits      int64   // Repo-wide total
    TotalContributors int     // Repo-wide count
    AgeDays           int64   // Days since account creation
    OrgMember         bool    // Member of repo owner's org
    LastCommitDays    int64   // Days since most recent commit
    Followers         int64   // GitHub follower count
    Following         int64   // Following count (for ratio)
    PublicRepos       int64   // Public repository count
    PrivateRepos      int64   // Private repository count
}

type CategoryWeight struct {
    Name   string  `json:"name"`
    Weight float64 `json:"weight"`
}

// Exported category weights.
var (
    CategoryProvenanceWeight float64  // 0.35
    CategoryIdentityWeight   float64  // 0.25
    CategoryEngagementWeight float64  // 0.25
    CategoryCommunityWeight  float64  // 0.15
)

func Compute(s Signals) float64       // Returns score in [0.0, 1.0]
func Categories() []CategoryWeight    // Returns model category metadata
```

## Package Changes

### `pkg/score/score.go` (new)
- All constants, weights, helper functions (`clampedRatio`, `logCurve`, `expDecay`)
- `Signals` struct, `CategoryWeight` type, `Compute()`, `Categories()`
- Depends only on `math` and `log/slog`

### `pkg/score/score_test.go` (new)
- All math function tests from `algo_test.go`
- All `calculateReputation` test cases adapted to use `score.Compute(Signals{})`
- Additional edge cases: suspended→0.0, max signals→1.0, zero-value→minimum

### `pkg/report/report.go` (modified)
- `ModelVersion` → `var ModelVersion = score.ModelVersion`
- `CategoryWeight` → type alias `= score.CategoryWeight`
- `Meta` struct unchanged

### `pkg/provider/github/algo.go` (modified)
- Remove scoring logic, constants, helpers
- Thin adapter: build `score.Signals` from `*report.Author` + context, call `score.Compute()`

### `pkg/provider/github/algo_test.go` (modified)
- Remove math function tests (moved to score)
- Keep adapter-level test or simplify to verify delegation

### `pkg/provider/github/provider.go` (modified)
- Meta uses `score.ModelVersion` and `score.Categories()`
- Remove `Category*Weight` vars

## Dependency Graph

```
pkg/score       → math, log/slog (no provider deps)
pkg/report      → pkg/score (for ModelVersion, CategoryWeight)
provider/github → pkg/score, pkg/report
```

## What Does Not Change

- CLI interface, flags, output format
- `report.Author`, `report.Stats`, `report.AuthorContext`, `report.Query`
- `report.Report`, `report.Meta` struct shapes
- `reporter.ListCommitAuthors()` behavior
- `provider.GetAuthors()` routing
- Scoring model weights and formula
- All existing test assertions (identical scores)
