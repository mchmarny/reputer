# reputer

Reporting tool to calculate contributor reputation based on graduated scoring algorithm for each provider. Currently supported providers: `github` and `gitlab`.

> Note: `reputation` is a value between 0 (no/low reputation) to 1.0 (high reputation). Each signal (account age, repos, commit verification, follower ratio, 2FA) contributes proportionally rather than as binary pass/fail. The algorithms consider only provider information about each contributor so the `reputation` is more an identity confidence score until additional/external data sources are introduced.

## install

```shell
brew tap mchmarny/reputer
brew install mchmarny/reputer/reputer
```

## usage

```shell
reputer [flags]
```

Supported flags:

* `--repo` - Repo URI (required, e.g. github.com/owner/repo)
* `--commit` - Commit at which to end the report (optional, inclusive)
* `--stats` - Includes stats used to calculate reputation (optional)
* `--file` - Write output to file at this path (optional, stdout if not specified)
* `--debug` - Turns logging verbose (optional)
* `--version` - Prints version only (optional)

example:

```shell
reputer \
    --repo github.com/mchmarny/reputer \
    --commit 3c239456ef63b45322b7ccdceb7f835c01fba862
```

results in:

```json
{
  "repo": "github.com/mchmarny/reputer",
  "generated_on": "2023-06-10T14:49:19.417079Z",
  "total_commits": 338,
  "total_contributors": 4,
  "contributors": [
    {
      "username": "mchmarny",
      "reputation": 1.0
    },
    ...
  ]
}
```

Same command with `--stats`

```json
{
  "repo": "github.com/mchmarny/reputer",
  "generated_on": "2023-06-10T14:49:19.417079Z",
  "total_commits": 338,
  "total_contributors": 4,
  "contributors": [
    {
      "username": "mchmarny",
      "reputation": 1.0,
      "context": {
        "company": "@Company",
        "created": "2010-01-04T00:19:57Z",
        "name": "Mark Chmarny"
      },
      "stats": {
        "verified_commits": true,
        "strong_auth": true,
        "age_days": 4906,
        "commits": 282,
        "unverified_commits": 0,
        "public_repos": 149,
        "private_repos": 26,
        "followers": 231,
        "following": 8
      }
    },
    ...
  ]
}
```

## scoring

Reputation is a value between `0` (no/low reputation) and `1.0` (high reputation), calculated using graduated proportional scoring. Each signal contributes linearly between 0 and its weight ceiling:

| Signal | Weight | Ceiling | Notes |
|--------|--------|---------|-------|
| 2FA (strong auth) | 0.25 | — | Binary: full weight if enabled |
| Commit verification | 0.25 | 100% verified | Ratio of verified to total commits |
| Follower/following ratio | 0.15 | 10:1 | Skipped if following is 0 |
| Account age | 0.15 | 365 days | Linear ramp to 1 year |
| Public repositories | 0.10 | 20 repos | |
| Private repositories | 0.10 | 10 repos | |

Each signal is clamped: `min(1.0, value / ceiling) × weight`. The final score is the sum of all weighted signals. Suspended users always score `0`.

## github actions

A reusable workflow is provided to welcome first-time PR contributors with a comment that includes their reputation stats.

### caller example

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
    uses: mchmarny/reputer/.github/workflows/welcome.yaml@main
```

### inputs

| Input | Type | Default | Description |
|-------|------|---------|-------------|
| `welcome-message` | string | *(auto)* | Custom welcome text; supports `{author}` placeholder |
| `include-stats` | boolean | `true` | Whether to show the stats table |
| `reputer-version` | string | `latest` | Reputer release version to install (e.g. `v0.2.4`) |

The caller's `permissions` block grants `pull-requests: write` and `contents: read` to the automatic `GITHUB_TOKEN`. No additional secrets are needed.

### behavior

1. Checks if the PR author is a first-time contributor
2. Installs reputer (pinned version or latest release)
3. Runs reputer against the calling repository
4. Posts a comment with:
   - Welcome message (first-time contributors only)
   - **Contributor Reputation** table if reputer finds the author in the repo history
   - **Contributor Profile** table (GitHub API fallback) if the author has no commits in the repo yet

The workflow uses `pull_request_target` context and does not check out PR code, so it is safe for use on public repositories.

## disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
