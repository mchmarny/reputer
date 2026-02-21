# Scoring Model v2 Design

## Problem

The current scoring algorithm uses 6 signals with ad-hoc weights and linear
response curves. Weights lack principled justification, correlated signals
double-count, and linear curves poorly model diminishing-returns signals like
account age. The model would not withstand ML community scrutiny.

## Goal

Replace the current model with a **risk-weighted categorical model** that:
- Groups signals into orthogonal threat categories
- Derives category weights from explicit threat-model ranking
- Uses domain-appropriate response curves (log, exponential decay, linear)
- Adapts dynamic parameters to repo context (team size, commit volume)
- Merges correlated signals to eliminate double-counting

Max score remains 1.0. Weights sum to 1.0.

## Threat Model

| Threat                        | Category              | What it answers                               |
|-------------------------------|-----------------------|-----------------------------------------------|
| Supply chain compromise       | **Code Provenance**   | Can we verify the provenance of their code?   |
| Fake/bot/sock puppet accounts | **Identity**          | Is this a real, accountable person?           |
| Drive-by / abandoned code     | **Engagement Depth**  | How invested is this person in THIS repo?     |
| Astroturfed reputation        | **Community Standing**| Do independent peers endorse this person?     |

## Category Weight Derivation

Categories ranked by impact: Provenance > Identity > Engagement > Community.

Justification:
- **Provenance is highest** -- directly measures the security property we care
  about (authenticated code). If commits aren't verified, identity alone
  doesn't prevent supply chain attacks.
- **Identity is second** -- if identity is fake, community and engagement
  signals are unreliable. But identity without provenance is insufficient.
- **Engagement is third** -- contextual to this repo, directly observable,
  harder to fake than community signals.
- **Community is lowest** -- most indirect, most gameable, weakest causal
  link to trustworthiness.

| Category           | Weight | Derivation                              |
|--------------------|--------|-----------------------------------------|
| Code Provenance    | 0.35   | Highest -- direct security signal       |
| Identity           | 0.25   | Second -- foundational trust            |
| Engagement Depth   | 0.25   | Third -- contextual investment          |
| Community Standing | 0.15   | Lowest -- indirect, gameable            |
| **Total**          | **1.00** |                                       |

## Signals

### Category 1: Code Provenance (0.35)

**Composite signal** merging commit verification and 2FA. These are correlated
(both measure authenticated code provenance) and interact -- signed commits
from a 2FA-protected account are worth more than the sum of the parts.

**Dimensions:**
1. Verification depth: fraction of commits cryptographically signed (0 to 1)
2. Account security: 2FA-enabled multiplier, dynamic by contribution proportion

**Formula:**
```
verifiedRatio = verifiedCommits / totalCommits
proportion = authorCommits / totalCommits
proportionCeiling = max(1.0 / numContributors, 0.05)

securityMultiplier = 2FA ? 1.0 : 1.0 - 0.5 * linear(proportion, proportionCeiling)
// Range for non-2FA: 0.90 (peripheral) to 0.50 (core contributor)

rawScore = verifiedRatio * securityMultiplier

// 2FA floor: account protection has baseline value
provenanceScore = 2FA ? max(rawScore, 0.1) : rawScore

score = provenanceScore * 0.35
```

**Dynamic security multiplier rationale:** A core contributor without 2FA is a
bigger supply chain risk than a peripheral one. The penalty scales with how
much of the repo's code they own.

| Scenario                        | verifiedRatio | multiplier | provenanceScore | * 0.35 |
|---------------------------------|---------------|------------|-----------------|--------|
| 2FA + 100% signed              | 1.0           | 1.0        | 1.0             | 0.350  |
| 2FA + 50% signed               | 0.5           | 1.0        | 0.5             | 0.175  |
| 2FA + 0% signed                | 0.0           | 1.0        | 0.1 (floor)     | 0.035  |
| No 2FA + 100% signed (core)    | 1.0           | 0.50       | 0.50            | 0.175  |
| No 2FA + 100% signed (periph)  | 1.0           | 0.90       | 0.90            | 0.315  |
| No 2FA + 50% signed (core)     | 0.5           | 0.50       | 0.25            | 0.088  |
| No 2FA + 0% signed             | 0.0           | any        | 0.0             | 0.000  |

### Category 2: Identity Authenticity (0.25)

Two signals. Account age gets the premium -- hardest to fabricate (requires
waiting). Org membership is strong but context-dependent (only meaningful for
org-owned repos).

#### Signal: Account Age (0.15)

- **Curve:** Logarithmic -- first year matters most, going from year 5 to 6
  is negligible.
- **Ceiling:** 730 days (2 years)
- **Formula:** `logCurve(ageDays, 730) * 0.15`

| Days | logCurve score | * 0.15 |
|------|---------------|--------|
| 1    | 0.11          | 0.016  |
| 30   | 0.52          | 0.078  |
| 100  | 0.70          | 0.105  |
| 365  | 0.90          | 0.135  |
| 730  | 1.00          | 0.150  |

#### Signal: Org Membership (0.10)

- **Curve:** Binary
- **Source:** `Organizations.IsMember(ctx, owner, username)` -- 1 API call
- **Graceful degradation:** API error (user-owned repo, rate limit, 403) -> 0,
  logged at debug level. No special-casing of owner type.

### Category 3: Engagement Depth (0.25)

Two signals measuring investment in THIS repo. Commit proportion gets the
premium -- a contributor with 20% of commits who went dormant 60 days ago is
more trustworthy than one with 1% who committed yesterday.

#### Signal: Commit Proportion (0.15)

- **Curve:** Linear with dynamic ceiling and confidence ramp
- **Dynamic ceiling:** `max(1.0 / numContributors, 0.05)`
- **Confidence ramp:** Dampens signal in tiny repos where proportions are
  statistically meaningless.

```
ceiling = max(1.0 / numContributors, 0.05)
confidenceThreshold = max(numContributors * 10, 30)
confidence = min(totalCommits / confidenceThreshold, 1.0)
score = linear(proportion, ceiling) * confidence * 0.15
```

| Contributors | Ceiling | Confidence threshold |
|-------------|---------|---------------------|
| 1-3         | 33-100% | 30                  |
| 5           | 20%     | 50                  |
| 10          | 10%     | 100                 |
| 20+         | 5%      | 200+                |

#### Signal: Commit Recency (0.10)

- **Curve:** Exponential decay -- freshness decays naturally with a half-life
- **Dynamic half-life:** Adapts to team size. In a repo with many active
  contributors, going 90 days without a commit is more concerning than in a
  solo project.

```
baseHalfLife = 90  // days
halfLife = baseHalfLife * max(1.0 / log(1 + numContributors), 0.25)
score = exp(-daysSinceLastCommit * ln2 / halfLife) * 0.10
```

| Contributors | Half-life | Rationale                            |
|-------------|-----------|--------------------------------------|
| 1           | 90 days   | Solo maintainer, long cycles normal  |
| 5           | 50 days   | Small team, moderate cadence         |
| 10          | 38 days   | Active project, expect regular work  |
| 50+         | 23 days   | Floor -- large project, high cadence |

| Days since last commit | Score (halflife=90d) | Score (halflife=38d) |
|-----------------------|---------------------|---------------------|
| 0                     | 0.100               | 0.100               |
| 30                    | 0.079               | 0.058               |
| 90                    | 0.050               | 0.018               |
| 180                   | 0.025               | 0.003               |

### Category 4: Community Standing (0.15)

Weakest category. Two signals. Follower ratio gets the premium -- peer
endorsement is a stronger signal than repo count. Public and private repos
are merged into one signal to eliminate correlation-based double-counting.

#### Signal: Follower Ratio (0.10)

- **Curve:** Logarithmic -- going from 0:1 to 3:1 matters more than 7:1 to 10:1
- **Ceiling:** 10:1 ratio
- **Skipped if:** Following == 0
- **Formula:** `logCurve(followers/following, 10) * 0.10`

#### Signal: Repo Count (0.05)

- **Curve:** Logarithmic -- first few repos matter more than going from 25 to 30
- **Input:** `publicRepos + privateRepos` (combined to eliminate correlated double-counting)
- **Ceiling:** 30 repos
- **Formula:** `logCurve(publicRepos + privateRepos, 30) * 0.05`

## Response Curve Definitions

```
Linear:      f(v, c) = min(v / c, 1.0)           -- v >= 0, c > 0
Logarithmic: f(v, c) = min(log(1+v) / log(1+c), 1.0)  -- diminishing returns
Exp decay:   f(v, h) = exp(-v * ln(2) / h)       -- freshness with half-life
Binary:      f(v)    = v ? 1.0 : 0.0
```

Edge cases: if denominator <= 0 or value <= 0, return 0.

## Correlation Analysis

| Signal Pair                      | Correlated? | Mitigation                              |
|----------------------------------|-------------|-----------------------------------------|
| Public repos <-> Private repos   | Yes         | **Merged** into single repo count       |
| 2FA <-> Commit verification      | Yes         | **Merged** into composite provenance    |
| Followers <-> Following          | Partially   | Used as **ratio** -- decorrelates       |
| Account age <-> Repo count       | Weakly      | Different categories -- acceptable      |
| Commit proportion <-> Recency    | Weakly      | Same category, explicit budget sharing  |

## Data Model Changes

```go
type Stats struct {
    // Existing (unchanged)
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

    // New
    LastCommitDays int64 `json:"last_commit_days,omitempty"`
    OrgMember      bool  `json:"org_member,omitempty"`
}
```

`calculateReputation` signature:
```go
func calculateReputation(author *report.Author, totalCommits int64, totalContributors int)
```

## API Changes

| Call | Phase | Cost | Purpose |
|------|-------|------|---------|
| `Repositories.ListCommits` | Existing | 0 new | Capture commit timestamps (already iterating) |
| `Users.Get` | Existing | 0 new | All user profile data (already called) |
| `Organizations.IsMember` | Phase 2 | 1 per author | Org membership check |

## Implementation Phases

**Phase 1:** Zero-cost signals + model restructure
- New response curve functions (logCurve, expDecay)
- Merge 2FA + commit verification into composite provenance signal
- Add commit recency (capture timestamps in existing pagination loop)
- Add commit proportion (derive from existing data)
- Merge public + private repo counts
- Redistribute all weights
- Rewrite all tests

**Phase 2:** Low-cost API signal
- Add org membership check in loadAuthor
- Graceful degradation on error

## Files Changed

| File | Change |
|------|--------|
| `pkg/report/author.go` | Add `LastCommitDays`, `OrgMember` to Stats |
| `pkg/provider/github/algo.go` | New weights, curve functions, composite provenance, dynamic params, updated signature |
| `pkg/provider/github/algo_test.go` | Full rewrite -- new weights, new curves, new signals, dynamic parameter tests |
| `pkg/provider/github/provider.go` | Capture commit timestamps, org membership API, pass totals to scoring |

## Known Limitations

- **No ground truth** -- weights are informed priors, not trained parameters.
  Should be calibrated against labeled data if available in the future.
- **GitHub-centric** -- org membership and 2FA availability vary across
  providers.
- **Punishes infrequent high-impact contributors** -- someone who makes one
  critical commit per quarter scores low on engagement.
- **Org membership favors insiders** -- external contributors from other
  trusted orgs get no credit.
- **Survivorship bias** -- only scores people who committed, not those who
  were denied access.

## Model Metadata in Report Output

Include model version and category breakdown in report output so consumers
can understand how scores were derived:

```go
type ReportMeta struct {
    ModelVersion string            `json:"model_version"`
    Categories   []CategoryWeight  `json:"categories"`
}

type CategoryWeight struct {
    Name    string  `json:"name"`
    Weight  float64 `json:"weight"`
}
```

## Implementation Notes

- Half-life multiplier capped at 1.0 on the upper end (solo repos don't get
  extended half-lives beyond 90 days).
- Commit timestamps use committer date (not author date) for recency, as
  committer date reflects when the commit entered the branch.
- GitHub returns commits newest-first, so the first commit seen per author in
  the pagination loop is their most recent.
- Org membership uses `Organizations.IsMember` which requires the token to
  have `read:org` scope. Falls back to false on any error.
- `ReportMeta` was simplified to `Meta` in implementation to avoid
  `report.ReportMeta` stuttering.
- The `make snapshot` cosign signing step requires interactive Sigstore auth
  and will fail in non-interactive contexts. The build itself succeeds.
