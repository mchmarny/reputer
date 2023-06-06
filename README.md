# reputer

Contributor reputation reporting tool

> The algorithm currently used to score GitHub author reputation is for demonstration purposes only. 

## build

```shell
make build
```

## usage 

```shell
bin/reputer [flags]
```

options:

* `repo` (string) - Repo URI (e.g. github.com/owner/repo)
* `commit` (string) - Commit from which to start the report (optional, inclusive)
```

example: 

```shell
bin/reputer \
    --repo github.com/mchmarny/reputer \
    --commit 3c239456ef63b45322b7ccdceb7f835c01fba862
```

results in: 

> Note, the commits are only the commits in this repo since the `commit` (if provided)

```json
{
  "repo": "github.com/mchmarny/reputer",
  "at_commit": "23da8455b5e59f57576b7fd4d18b0ad7fc53596e",
  "generated_on": "2023-06-06T14:36:47.157072Z",
  "authors": [
    {
      "username": "mchmarny",
      "created": "2010-01-04T00:19:57Z",
      "public_repos": 148,
      "private_repos": 26,
      "followers": 231,
      "following": 8,
      "two_factor_auth": true,
      "reputation": 1,
      "context": {
        "company": "@Google",
        "name": "Mark Chmarny"
      },
      "commits": [
        {
          "sha": "003582a11a45ff4c2c08185f76bcc256f8fa9acb",
          "verified": true
        },
        ...
      ]
    },
    ...
  ]
}
```

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
