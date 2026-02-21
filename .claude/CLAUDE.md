# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## Project Overview

`reputer` is a CLI tool that calculates contributor reputation scores from Git provider APIs (GitHub, GitLab). It uses graduated proportional scoring — each signal contributes linearly between 0 and its weight ceiling, summing to a max of 1.0.

## Build & Test Commands

```
make test        # unit tests with -race and coverage
make lint        # golangci-lint + yamllint
make build       # build binary to ./bin/reputer
make tidy        # go mod tidy + vendor
make vulncheck   # govulncheck
make snapshot    # test + lint + goreleaser build
```

Run `make lint` and `make test` before every commit.

## Project Structure

```
cmd/                    CLI entry point
pkg/
  provider/             Provider abstraction (routes queries to backend)
    github/             GitHub: API client, reputation algo, tests
    gitlab/             GitLab: stub provider (documented TODO)
  report/               Data model (Author, Stats, Report, Query)
  reputer/              Orchestration (ListCommitAuthors, options)
```

## Non-Negotiable Rules

1. **Read before writing** — Never modify code you haven't read
2. **Tests must pass** — `make test` with race detector; never skip or disable tests
3. **Lint must pass** — `make lint` must be clean before committing
4. **Use project patterns** — Learn existing code before inventing new approaches
5. **3-strike rule** — After 3 failed fix attempts, stop and reassess

## Code Patterns

- **Errors**: `fmt.Errorf` with `%w` for wrapping; `errors.New` for simple errors
- **Logging**: `log/slog` via `pkg/logging`; structured key-value pairs
- **Testing**: `testify/assert` with table-driven tests
- **Dependencies**: vendored (`go mod vendor`); run `make tidy` after changes

## Git Configuration

- Commit to `main` branch (not `master`)
- Use `-S` to cryptographically sign commits
- Do NOT add `Co-Authored-By` lines
- Do not sign-off commits (no `-s` flag)

## Decision Framework

When choosing between approaches, prioritize:
1. **Testability** — Can it be unit tested without external dependencies?
2. **Readability** — Can another engineer understand it quickly?
3. **Consistency** — Does it match existing patterns in the codebase?
4. **Simplicity** — Is it the simplest solution that works?
5. **Reversibility** — Can it be easily changed later?
