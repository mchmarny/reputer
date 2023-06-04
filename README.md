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

## Disclaimer

This is my personal project and it does not represent my employer. While I do my best to ensure that everything works, I take no responsibility for issues caused by this code.
