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
brew tap mchmarny/reputer
brew install reputer
```

Go:

```shell
go install github.com/mchmarny/reputer@latest
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
    "model_version": "2.0.0",
    "categories": [
      { "name": "code_provenance", "weight": 0.35 },
      { "name": "identity", "weight": 0.25 },
      { "name": "engagement", "weight": 0.25 },
      { "name": "community", "weight": 0.15 }
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
        "org_member": true
      }
    }
  ]
}
```

## Scoring

Reputation is calculated using a **v2 risk-weighted categorical model** (model version `2.0.0`). Signals are grouped into four categories ranked by threat-model priority. Suspended users always score `0`.

### Categories

| Category | Weight | Signals |
|----------|--------|---------|
| Code Provenance | 0.35 | Commit verification ratio, 2FA security multiplier |
| Identity Authenticity | 0.25 | Account age, org membership |
| Engagement Depth | 0.25 | Commit proportion, recency |
| Community Standing | 0.15 | Follower/following ratio, repository count |

### Signals

| Signal | Weight | Ceiling / Curve | Details |
|--------|--------|-----------------|---------|
| Commit verification | 0.35 | verified/total ratio | 2FA acts as a security multiplier; without 2FA, core contributors are penalized up to 50%. 2FA holders get a floor of 0.1. |
| Account age | 0.15 | 730 days, log curve | Diminishing returns — early days matter more |
| Org membership | 0.10 | binary | Full weight if the author is a member of the repo owner org |
| Commit proportion | 0.15 | adaptive ceiling | Scaled by repo confidence (min 30 commits) |
| Recency | 0.10 | exponential decay | Base half-life of 90 days, adjusted by contributor count |
| Follower ratio | 0.10 | 10:1 ratio, log curve | `followers / following`; skipped if following is 0 |
| Repository count | 0.05 | 30 repos, log curve | Combined public + private repositories |

## GitHub Action

A composite action is provided to welcome first-time PR contributors with a comment that includes their reputation stats.

### Caller example

Add this to your repo at `.github/workflows/welcome.yaml`:

```yaml
name: welcome
on:
  pull_request_target:
    types: [opened]
permissions:
  pull-requests: write
  contents: read
jobs:
  welcome:
    runs-on: ubuntu-latest
    steps:
      - uses: mchmarny/reputer@main
```

### Inputs

| Input | Type | Default | Description |
|-------|------|---------|-------------|
| `welcome-message` | string | *(auto)* | Custom welcome text; supports `{author}` placeholder |
| `include-stats` | boolean | `true` | Whether to show the stats table |
| `reputer-version` | string | `latest` | Reputer release version to install (e.g. `v0.2.4`) |

The caller's `permissions` block grants `pull-requests: write` and `contents: read` to the automatic `GITHUB_TOKEN`. No additional secrets are needed.

### Behavior

1. Checks if the PR author is a first-time contributor
2. Installs reputer (pinned version or latest release)
3. Runs reputer against the calling repository
4. Posts a comment with:
   - Welcome message (first-time contributors only)
   - **Contributor Reputation** table if reputer finds the author in the repo history
   - **Contributor Profile** table (GitHub API fallback) if the author has no commits yet

The workflow uses `pull_request_target` context and does not check out PR code, so it is safe for use on public repositories.

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines and [DEVELOPMENT.md](DEVELOPMENT.md) for development setup.

## License

[Apache License 2.0](LICENSE)

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
