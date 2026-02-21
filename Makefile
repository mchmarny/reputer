VERSION    :=$(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
COMMIT     :=$(shell git rev-parse --short HEAD)
BRANCH     :=$(shell git rev-parse --abbrev-ref HEAD)
GO_VERSION :=$(shell go env GOVERSION 2>/dev/null | sed 's/go//')
GOLINT_VER :=$(shell golangci-lint --version 2>/dev/null | awk '{print $$4}' || echo "not installed")
YAML_FILES :=$(shell find . ! -path "./vendor/*" -type f -regex ".*\.yaml" -print)
COVERAGE   ?=$(shell awk '/^target:/{print $$2}' .codecov.yaml 2>/dev/null)
ifeq ($(COVERAGE),)
COVERAGE := 30
endif

all: help

# =============================================================================
# Info
# =============================================================================

.PHONY: info
info: ## Prints project and toolchain info
	@echo "version:  $(VERSION)"
	@echo "commit:   $(COMMIT)"
	@echo "branch:   $(BRANCH)"
	@echo "go:       $(GO_VERSION)"
	@echo "linter:   $(GOLINT_VER)"

# =============================================================================
# Code Formatting & Dependencies
# =============================================================================

.PHONY: tidy
tidy: ## Formats code and updates Go module dependencies
	go fmt ./...
	go mod tidy
	go mod vendor

.PHONY: upgrade
upgrade: ## Upgrades all dependencies to latest versions
	go get -u ./...
	go mod tidy
	go mod vendor

.PHONY: fmt-check
fmt-check: ## Checks if code is formatted (CI-friendly, no modifications)
	@test -z "$$(gofmt -l .)" || (echo "Code is not formatted. Run 'make tidy' to fix:" && gofmt -l . && exit 1)
	@echo "Code formatting check passed"

# =============================================================================
# Quality
# =============================================================================

.PHONY: lint
lint: lint-go lint-yaml ## Lints the entire project (Go and YAML)
	@echo "Completed Go and YAML lints"

.PHONY: lint-go
lint-go: ## Lints Go files with go vet and golangci-lint
	GOFLAGS="-mod=vendor" go vet ./...
	GOFLAGS="-mod=vendor" golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Lints YAML files with yamllint (brew install yamllint)
	yamllint -c .yamllint.yaml $(YAML_FILES)

.PHONY: test-coverage
test-coverage: test ## Runs tests and enforces coverage threshold (default from .codecov.yaml)
	@coverage=$$(go tool cover -func=cover.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$coverage% (threshold: $(COVERAGE)%)"; \
	if [ $$(echo "$$coverage < $(COVERAGE)" | bc) -eq 1 ]; then \
		echo "ERROR: Coverage $$coverage% is below threshold $(COVERAGE)%"; \
		exit 1; \
	fi; \
	echo "Coverage check passed"

.PHONY: test
test: ## Runs unit tests with race detector and coverage
	GOFLAGS="-mod=vendor" go test -count=1 -race -timeout=5m -covermode=atomic -coverprofile=cover.out ./...

.PHONY: vulncheck
vulncheck: ## Checks for source vulnerabilities
	govulncheck -test ./...

.PHONY: qualify
qualify: test-coverage lint vulncheck ## Qualifies the codebase (test-coverage, lint, vulncheck)
	@echo "Codebase qualification completed"

# =============================================================================
# Build & Release
# =============================================================================

.PHONY: build
build: tidy ## Builds CLI binary
	mkdir -p ./bin
	CGO_ENABLED=0 go build -trimpath \
	-ldflags="-w -s \
	-X github.com/mchmarny/reputer/pkg/cli.version=$(VERSION) \
	-X github.com/mchmarny/reputer/pkg/cli.commit=$(COMMIT) \
	-X 'github.com/mchmarny/reputer/pkg/cli.date=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' \
	-extldflags '-static'" -mod vendor \
	-o bin/reputer cmd/main.go

.PHONY: snapshot
snapshot: test lint ## Runs test, lint before building snapshot distributables
	GITLAB_TOKEN="" goreleaser release --snapshot --clean --timeout 10m0s

.PHONY: bump-patch
bump-patch: ## Bumps patch version (v1.2.3 → v1.2.4)
	tools/bump patch

.PHONY: bump-minor
bump-minor: ## Bumps minor version (v1.2.3 → v1.3.0)
	tools/bump minor

.PHONY: bump-major
bump-major: ## Bumps major version (v1.2.3 → v2.0.0)
	tools/bump major

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Cleans build artifacts
	go clean
	rm -fr ./vendor
	rm -fr ./bin
	rm -f ./cover.out

.PHONY: help
help: ## Displays available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
