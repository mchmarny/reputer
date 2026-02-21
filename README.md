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

## disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
