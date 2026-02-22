# Development Guide

This guide covers project setup, architecture, development workflows, and tooling for contributors working on reputer.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Development Setup](#development-setup)
- [Project Architecture](#project-architecture)
- [Development Workflow](#development-workflow)
- [Make Targets Reference](#make-targets-reference)
- [Debugging](#debugging)

## Quick Start

```bash
# 1. Clone and setup
git clone https://github.com/mchmarny/reputer.git && cd reputer
make tidy           # Download dependencies

# 2. Develop
make test           # Run tests with race detector
make lint           # Run linters
make build          # Build binary

# 3. Before submitting PR
make qualify        # Full check: test-coverage + lint + vulncheck
```

## Prerequisites

### Required Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| **Go 1.26+** | Language runtime | [golang.org/dl](https://golang.org/dl/) |
| **make** | Build automation | Pre-installed on macOS; `apt install make` on Ubuntu/Debian |
| **git** | Version control | Pre-installed on most systems |

### Development Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| golangci-lint | Go linting | [golangci-lint.run](https://golangci-lint.run/welcome/install/) |
| yamllint | YAML linting | `brew install yamllint` or `pip install yamllint` |
| govulncheck | Vulnerability scanning | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| goreleaser | Release automation / local builds | [goreleaser.com](https://goreleaser.com/install/) |

## Development Setup

### Clone and Build

```bash
git clone https://github.com/mchmarny/reputer.git
cd reputer
```

Build for your current platform:

```bash
make build
```

The binary is output to `./bin/reputer`.

### Finalize Setup

After cloning:

```bash
# Download Go module dependencies
make tidy

# Run full qualification to ensure setup is correct
make qualify
```

## Project Architecture

### Directory Structure

```
reputer/
├── cmd/                    CLI entry point
├── pkg/
│   ├── provider/           Provider abstraction (routes queries to backend)
│   │   ├── github/         GitHub: API client, reputation algorithm, tests
│   │   └── gitlab/         GitLab: stub provider (documented TODO)
│   ├── report/             Data model (Author, Stats, Report, Query)
│   ├── reporter/           Output formatting (JSON file writer)
│   ├── reputer/            Orchestration (ListCommitAuthors, options)
│   └── score/              Standalone scoring model (Compute, Signals, Categories)
├── tools/                  Development scripts (bump)
├── .github/
│   ├── workflows/          CI/CD workflows (test, release, scan, score)
│   └── actions/            Composite actions (setup-build-tools)
├── .golangci.yaml          golangci-lint configuration
├── .yamllint.yaml          yamllint configuration
├── .goreleaser.yaml        GoReleaser configuration
└── .codecov.yaml           Codecov coverage configuration
```

### Key Components

#### CLI (`cmd/`)
Thin entry point that wires flags to the orchestration layer.

#### Provider (`pkg/provider/`)
Abstraction layer that routes queries to the correct backend (GitHub or GitLab). Each provider implements the reputation scoring algorithm using that platform's API.

#### GitHub Provider (`pkg/provider/github/`)
Full implementation with API client, graduated proportional scoring, rate-limit awareness, and pagination handling.

#### Report (`pkg/report/`)
Data model types: `Author`, `Stats`, `Report`, `Query`. Pure data structures with no external dependencies.

#### Reputer (`pkg/reputer/`)
Orchestration layer that coordinates providers and produces reports. Contains `ListCommitAuthors` and configuration options.

### Data Flow

```
Git Provider API --> provider (github/gitlab) --> reputer orchestration --> report --> stdout/file
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feat/add-new-signal
```

### 2. Make Changes

- Small, focused commits: each commit should address one logical change
- Test as you go: write tests alongside your code
- Read existing code in the package before modifying it

### 3. Run Tests

```bash
# Unit tests with race detector
make test
```

### 4. Lint Your Code

```bash
make lint
```

This runs `go vet`, `golangci-lint`, and `yamllint`.

### 5. Vulnerability Scan

```bash
make vulncheck
```

### 6. Full Qualification

Before submitting a PR, run everything:

```bash
make qualify
```

This runs: `test-coverage` -> `lint` -> `vulncheck`

## Make Targets Reference

### Quality and Testing

| Target | Description |
|--------|-------------|
| `make qualify` | Full qualification (test-coverage + lint + vulncheck) |
| `make test` | Unit tests with race detector and coverage |
| `make test-coverage` | Tests with coverage threshold enforcement (default from `.codecov.yaml`) |
| `make lint` | Go vet + golangci-lint + yamllint |
| `make lint-go` | Go vet + golangci-lint only |
| `make lint-yaml` | yamllint only |
| `make vulncheck` | Vulnerability scan with govulncheck |
| `make fmt-check` | Check code formatting (CI-friendly, no modifications) |

### Build and Release

| Target | Description |
|--------|-------------|
| `make build` | Build binary to `./bin/reputer` |
| `make snapshot` | Test + lint then goreleaser snapshot build |
| `make bump-patch` | Bump patch version (1.2.3 -> 1.2.4) |
| `make bump-minor` | Bump minor version (1.2.3 -> 1.3.0) |
| `make bump-major` | Bump major version (1.2.3 -> 2.0.0) |

### Code Maintenance

| Target | Description |
|--------|-------------|
| `make tidy` | Format code, update and vendor dependencies |
| `make upgrade` | Upgrade all dependencies to latest versions |

### Utilities

| Target | Description |
|--------|-------------|
| `make info` | Print project info (version, commit, Go version, linter) |
| `make clean` | Clean build artifacts, vendor, and coverage output |
| `make help` | Show all available targets |

## Debugging

### Common Issues

| Issue | Solution |
|-------|----------|
| Tests fail with race conditions | Check for shared state in goroutines |
| Linter errors | Run `make lint` and fix reported issues |
| Build failures | Run `make tidy` to update dependencies |
| API rate limit errors | Set `GITHUB_TOKEN` env var for higher limits |

### Debugging Tests

```bash
# Run specific test with verbose output
go test -v -mod=vendor ./pkg/provider/github/... -run TestSpecificFunction

# Run tests with race detector
go test -race -mod=vendor ./...

# Generate coverage report
go test -coverprofile=cover.out -mod=vendor ./...
go tool cover -html=cover.out
```

### Environment Variables

| Variable | Purpose |
|----------|---------|
| `GITHUB_TOKEN` | GitHub API authentication (higher rate limits) |
| `GITLAB_TOKEN` | GitLab API authentication |

## Additional Resources

- [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- [README.md](README.md) - Project overview and quick start
