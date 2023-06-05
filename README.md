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
    "login": "xtreme-vikram-yadav",
    "created": "2013-07-22T15:54:52Z",
    "commits": [
      "c90fabf38796ec70286142c052b01570826d1a75"
    ],
    "name": "Vikram Yadav",
    "repos": 14,
    "gists": 0,
    "followers": 1,
    "following": 0
  },
  {
    "login": "davidhadas",
    "created": "2013-04-23T08:15:45Z",
    "commits": [
      "84fa64c75bd32662a623ecb9063797dfd5d624da"
    ],
    "name": "David Hadas",
    "email": "david.hadas@gmail.com",
    "company": "IBM Research",
    "repos": 61,
    "gists": 0,
    "followers": 6,
    "following": 0
  },
  ...
]
```

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
