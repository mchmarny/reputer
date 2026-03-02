# Reputation Signals v3.0.0 Design

Motivation: the hackerbot-claw campaign (Feb 2026) demonstrated that autonomous bots
can exploit CI/CD pipelines across major open-source repos using new GitHub accounts
with no history. The current v2 model penalizes new/empty accounts but cannot
distinguish "new legitimate contributor" from "automated attack bot." This design adds
behavioral and identity signals to close that gap.

## New Signals

| Signal | Signals Field(s) | GitHub API | Cost |
|--------|------------------|------------|------|
| Author association | `AuthorAssociation string` | Commit objects (already fetched) | Free |
| PR acceptance rate | `PRsMerged, PRsClosed int64` | `GET /search/issues?q=author:{user}+type:pr` | 1 call/contributor |
| Cross-repo burst | `RecentPRRepoCount int64` | `GET /users/{user}/events/public` | 1-3 pages/contributor |
| Fork-only account | `ForkedRepos int64` | `GET /users/{user}/repos?type=owner` | 1-3 pages/contributor |
| Profile completeness | `HasBio, HasCompany, HasLocation, HasWebsite bool` | `Users.Get` (already fetched) | Free |

Net new API calls per contributor: ~2-6 (search + events + repos listing).

## Weight Model v3.0.0

| Category | v2 | v3 | Sub-signals |
|----------|----|----|-------------|
| Code Provenance | 0.35 | 0.30 | verified ratio, 2FA, security multiplier |
| Identity | 0.25 | 0.20 | age, author association, profile completeness |
| Engagement | 0.25 | 0.20 | proportion, recency, PR acceptance rate |
| Community | 0.15 | 0.10 | follower ratio, repo count |
| Behavioral (new) | -- | 0.20 | cross-repo burst penalty, fork-only ratio |

## Scoring Logic

### Author Association (Identity, 0.05)

Replaces binary `OrgMember` (was 0.10). Maps GitHub's `author_association` enum:

- `OWNER`/`MEMBER` -> 1.0
- `COLLABORATOR` -> 0.8
- `CONTRIBUTOR` -> 0.5
- `FIRST_TIME_CONTRIBUTOR` -> 0.2
- `NONE` -> 0.0

Weight: 0.05. The remaining 0.05 of old org weight absorbed into association scoring
(MEMBER/OWNER already scores 1.0 here).

### Profile Completeness (Identity, 0.05)

Count of filled fields (bio, company, location, website) / 4. Linear scaling.

### PR Acceptance Rate (Engagement, 0.05)

- `mergeRate = merged / (merged + closed)` (terminal-state PRs only)
- Confidence = `logCurve(merged + closed, prCountCeil)` where `prCountCeil = 20`
- Score = `mergeRate * confidence`
- Zero terminal PRs -> score 0 (unknown, not penalized via other signals)

### Cross-Repo Burst (Behavioral, 0.10)

- `burstRate = recentPRRepoCount / max(ageDays / 30, 1)`
- Score = `1.0 - clampedRatio(burstRate, burstCeil)` where `burstCeil = 5.0`
- High burst rate (many repos relative to account age) -> lower score
- Zero recent PR repos -> score 1.0 (no burst detected)

### Fork-Only Ratio (Behavioral, 0.10)

- `originalRepos = publicRepos - forkedRepos`
- Score = `clampedRatio(originalRepos, forkOriginalCeil)` where `forkOriginalCeil = 5.0`
- Accounts with only forks -> 0. Accounts with 5+ original repos -> 1.0.

## Data Model Changes

### score.Signals (new fields)

```go
AuthorAssociation string // OWNER, MEMBER, COLLABORATOR, CONTRIBUTOR, FIRST_TIME_CONTRIBUTOR, NONE
HasBio            bool
HasCompany        bool
HasLocation       bool
HasWebsite        bool
PRsMerged         int64
PRsClosed         int64
RecentPRRepoCount int64
ForkedRepos       int64
```

### report.Stats (new fields, matching Signals)

```go
AuthorAssociation string `json:"author_association,omitempty" yaml:"authorAssociation,omitempty"`
HasBio            bool   `json:"has_bio,omitempty" yaml:"hasBio,omitempty"`
HasLocation       bool   `json:"has_location,omitempty" yaml:"hasLocation,omitempty"`
HasWebsite        bool   `json:"has_website,omitempty" yaml:"hasWebsite,omitempty"`
PRsMerged         int64  `json:"prs_merged,omitempty" yaml:"prsMerged,omitempty"`
PRsClosed         int64  `json:"prs_closed,omitempty" yaml:"prsClosed,omitempty"`
RecentPRRepoCount int64  `json:"recent_pr_repo_count,omitempty" yaml:"recentPRRepoCount,omitempty"`
ForkedRepos       int64  `json:"forked_repos,omitempty" yaml:"forkedRepos,omitempty"`
```

Note: `HasCompany` already tracked via `AuthorContext.Company`; add bool to Stats for scoring.

## Implementation Steps

1. Add new fields to `score.Signals`
2. Add new fields to `report.Stats`
3. Update `score.Compute` with v3 weights and new category logic
4. Update `score_test.go` with new weight sum, new signal tests
5. Add GitHub API fetch functions: `fetchPRStats`, `fetchRecentEvents`, `fetchForkedRepoCount`
6. Extract `AuthorAssociation` from commit data during commit listing (already available)
7. Extract profile completeness from existing `Users.Get` response
8. Call new fetch functions concurrently in `loadAuthor`
9. Update `algo.go` to map new Stats fields to Signals
10. Update `algo_test.go` with new test cases
11. Bump `ModelVersion` to `3.0.0`

## Backward Compatibility

- `OrgMember` field remains in Stats JSON output for consumers
- `OrgMember` no longer used in scoring (replaced by `AuthorAssociation`)
- Model version bumped to 3.0.0 (breaking: scores not comparable to v2)
