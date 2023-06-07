# reputer

Reporting tool to calculate contributor reputation based on configurable algorithm for each provider. Currently supports github and gitlab providers.  

> Note: `reputation` is a value between 0 (no/low reputation) to 1.0 (high reputation). The algorithms used in this repo currently consider only the provider information about each contributor so the `reputation` is more a identity confidence score until additional/external data sources are introduced. 

## install 

```shell
brew tap mchmarny/reputer
brew install mchmarny/reputer/reputer
```

## usage 

```shell
Usage of reputer (v0.0.8):
  -repo string
    	Repo URI (required, e.g. github.com/owner/repo)
  -commit string
    	Commit at which to end the report (optional, inclusive)
  -file string
    	Write output to file at this path (optional, stdout if not specified)
  -debug
    	Turns logging verbose (optional, false)
```

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
  "generated_on": "2023-06-06T22:34:35.273897Z",
  "total_commits": 180,
  "total_contributors": 10,
  "contributors": [
    {
      "username": "mchmarny",
      "created": "2010-01-04T00:19:57Z",
      "public_repos": 148,
      "private_repos": 26,
      "followers": 231,
      "following": 8,
      "commits": 18,
      "verified_commits": true,
      "strong_auth": true,
      "reputation": 0.95,
      "context": {
        "company": "@Google",
        "name": "Mark Chmarny"
        ...
      }
    },
    ...
  ]
}
```

## disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
