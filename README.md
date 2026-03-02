# reputer

[![Build Status](https://github.com/mchmarny/reputer/actions/workflows/on-push.yaml/badge.svg)](https://github.com/mchmarny/reputer/actions/workflows/on-push.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mchmarny/reputer)](https://goreportcard.com/report/github.com/mchmarny/reputer)
[![Go Reference](https://pkg.go.dev/badge/github.com/mchmarny/reputer.svg)](https://pkg.go.dev/github.com/mchmarny/reputer)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

CLI tool that calculates contributor reputation scores from Git provider APIs. Currently supported providers: **GitHub** and **GitLab**.

Reputation is a value between `0` (no/low reputation) and `1.0` (high reputation). The scoring model uses only provider-sourced signals, so the score is best understood as an identity confidence indicator.

## Prerequisites

A personal access token for the target provider:

| Provider | Environment Variable | Required Scopes |
|----------|---------------------|-----------------|
| GitHub | `GITHUB_TOKEN` | `repo` (read) |
| GitLab | `GITLAB_TOKEN` | `read_api` |

## Install

Homebrew:

```shell
brew tap mchmarny/tap
brew install reputer
```

Go:

```shell
go install github.com/mchmarny/reputer/cmd/reputer@latest
```

Or download a binary from the [releases](https://github.com/mchmarny/reputer/releases) page.

## Usage

```shell
reputer [flags]
```

| Flag | Description |
|------|-------------|
| `--repo` | Repo URI (required, e.g. `github.com/owner/repo`) |
| `--commit` | Commit at which to end the report (optional, inclusive) |
| `--stats` | Include stats used to calculate reputation (optional) |
| `--file` | Write output to file at this path (optional, stdout if not specified) |
| `--format` | Output format: `json` or `yaml` (optional, default: `json`) |
| `--debug` | Turn on verbose logging (optional) |
| `--version` | Print version only (optional) |

Example:

```shell
export GITHUB_TOKEN=ghp_...
reputer --repo github.com/mchmarny/reputer
```

Output:

```json
{
  "repo": "github.com/mchmarny/reputer",
  "at_commit": "",
  "generated_on": "2025-06-10T14:49:19Z",
  "total_commits": 338,
  "total_contributors": 4,
  "meta": {
    "model_version": "3.0.0",
    "categories": [
      { "name": "code_provenance", "weight": 0.3 },
      { "name": "identity", "weight": 0.2 },
      { "name": "engagement", "weight": 0.2 },
      { "name": "community", "weight": 0.1 },
      { "name": "behavioral", "weight": 0.2 }
    ]
  },
  "contributors": [
    {
      "username": "mchmarny",
      "reputation": 1.0
    }
  ]
}
```

With `--stats`:

```json
{
  "contributors": [
    {
      "username": "mchmarny",
      "reputation": 1.0,
      "context": {
        "created": "2010-01-04T00:19:57Z",
        "name": "Mark Chmarny",
        "company": "@Company"
      },
      "stats": {
        "verified_commits": true,
        "strong_auth": true,
        "age_days": 5640,
        "commits": 282,
        "unverified_commits": 0,
        "public_repos": 149,
        "private_repos": 26,
        "followers": 231,
        "following": 8,
        "last_commit_days": 3,
        "org_member": true,
        "author_association": "MEMBER",
        "has_bio": true,
        "has_company": true,
        "has_location": true,
        "has_website": true,
        "prs_merged": 85,
        "prs_closed": 3,
        "recent_pr_repo_count": 2,
        "forked_repos": 1
      }
    }
  ]
}
```

## Scoring

Reputation is calculated using a **v3 risk-weighted categorical model** (model version `3.0.0`). Signals are grouped into five categories ranked by threat-model priority. Suspended users always score `0`.

### Categories

| Category | Weight | Signals |
|----------|--------|---------|
| Code Provenance | 0.30 | Commit verification ratio, 2FA security multiplier |
| Identity | 0.20 | Account age, author association, profile completeness |
| Engagement | 0.20 | Commit proportion, recency, PR acceptance rate |
| Community | 0.10 | Follower/following ratio, repository count |
| Behavioral | 0.20 | Cross-repo burst detection, fork-only ratio |

### Signals

| Signal | Weight | Ceiling / Curve | Details |
|--------|--------|-----------------|---------|
| Commit verification | 0.30 | verified/total ratio | 2FA acts as a security multiplier; without 2FA, core contributors are penalized up to 50%. 2FA holders get a floor of 0.1. |
| Account age | 0.10 | 730 days, log curve | Diminishing returns — early days matter more |
| Author association | 0.05 | enum mapping | OWNER/MEMBER→1.0, COLLABORATOR→0.8, CONTRIBUTOR→0.5, FIRST_TIME→0.2, NONE→0.0. Falls back to org membership. |
| Profile completeness | 0.05 | 4 fields, linear | Bio, company, location, website — count of filled fields / 4 |
| Commit proportion | 0.10 | adaptive ceiling | Scaled by repo confidence (min 30 commits) |
| Recency | 0.05 | exponential decay | Base half-life of 90 days, adjusted by contributor count |
| PR acceptance rate | 0.05 | 20 PRs, log curve | `merged / (merged + closed)` with confidence scaling |
| Follower ratio | 0.05 | 10:1 ratio, log curve | `followers / following`; skipped if following is 0 |
| Repository count | 0.05 | 30 repos, log curve | Combined public + private repositories |
| Cross-repo burst | 0.10 | 5.0 rate ceiling | Penalty for high PR activity across many repos relative to account age |
| Fork-only ratio | 0.10 | 5 original repos | Accounts with only forked repos and no original work score 0 |

## GitHub Action

A composite action that posts contributor reputation scores on pull requests.

### Caller example

Add this to your repo at `.github/workflows/reputation.yaml`:

```yaml
name: reputation
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  pull-requests: write
  contents: read
jobs:
  score:
    runs-on: ubuntu-latest
    steps:
      - uses: mchmarny/reputer@main
```

### Inputs

| Input | Type | Default | Description |
|-------|------|---------|-------------|
| `reputer-version` | string | `latest` | Reputer release version to install (e.g. `v0.2.4`) |

The caller's `permissions` block grants `pull-requests: write` and `contents: read` to the automatic `GITHUB_TOKEN`. No additional secrets are needed.

### Behavior

1. Installs reputer (pinned version or latest release, with checksum verification)
2. Runs reputer against the calling repository
3. Posts a **Contributor Reputation** comment with the PR author's v3 score and stats

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines and [DEVELOPMENT.md](DEVELOPMENT.md) for development setup.

## License

[Apache License 2.0](LICENSE)

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
