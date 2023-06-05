# reputer

Contributor reputation reporting tool

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
    --repo github.com/knative/serving \
    --commit 3c239456ef63b45322b7ccdceb7f835c01fba862
```

results in: 

```json
[
  {
    "login": "mchmarny",
    "created": "2010-01-04T00:19:57Z",
    "days": 4900,
    "commits": [
      "003582a11a45ff4c2c08185f76bcc256f8fa9acb",
      ...
    ],
    "name": "Mark Chmarny",
    "company": "@Google",
    "repos": 148,
    "gists": 4,
    "followers": 230,
    "following": 8,
    "two_factor_auth": true,
    "reputation": 2.36
  }
  ...
]
```

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
