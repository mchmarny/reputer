VERSION    :=$(shell cat .version)
YAML_FILES :=$(shell find . ! -path "./vendor/*" -type f -regex ".*\.yaml" -print)

all: help

.PHONY: info
info: ## Prints all variables
	@echo "version:  $(VERSION)"
		
.PHONY: tidy
tidy: ## Updates the go modules and vendors all dependancies 
	go mod tidy
	go mod vendor

.PHONY: upgrade
upgrade: ## Upgrades all dependancies 
	go get -d -u ./...
	go mod tidy
	go mod vendor

.PHONY: test
test: tidy ## Runs unit tests
	go test -count=1 -race -covermode=atomic -coverprofile=cover.out ./...

.PHONY: lint
lint: lint-go lint-yaml ## Lints both Go and YAML files
	@echo "Completed Go and YAML lints"

.PHONY: lint
lint-go: ## Lints the entire project using Go
	golangci-lint -c .golangci.yaml run

.PHONY: lint-yaml
lint-yaml: ## Runs yamllint on all yaml files (brew install yamllint)
	yamllint -c .yamllint $(YAML_FILES)

.PHONY: build
build: tidy ## Builds CLI binary
	mkdir -p ./bin
	CGO_ENABLED=0 go build -trimpath \
	-ldflags="-w -s -X main.version=$(VERSION) \
	-extldflags '-static'" -mod vendor \
	-o bin/reputer cmd/main.go

.PHONY: snapshot
snapshot: test lint ## Runs test, lint before building snapshot distributables
	GITLAB_TOKEN="" goreleaser release --snapshot --clean --timeout 10m0s

.PHONY: vulncheck
vulncheck: ## Checks for soource vulnerabilities
	govulncheck -test ./...

.PHONY: tag
tag: ## Creates release tag 
	git tag -s -m "bump version" $(VERSION)
	git push origin $(VERSION)

.PHONY: tagless
tagless: ## Delete the current release tag and creates a new one
	git tag -d $(VERSION)
	git push --delete origin $(VERSION)

.PHONY: clean
clean: ## Cleans bin and temp directories
	go clean
	rm -fr ./vendor
	rm -fr ./bin

.PHONY: help
help: ## Display available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk \
		'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
