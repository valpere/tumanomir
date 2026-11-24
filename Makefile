BINARY := tumanomir
BIN_DIR := bin

.DEFAULT_GOAL := help

.PHONY: help build install vet test lint dogfood verify ci clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-10s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build bin/tumanomir
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/tumanomir

install: ## Install tumanomir into $GOBIN (or $GOPATH/bin)
	go install ./cmd/tumanomir

vet: ## Run go vet
	go vet ./...

test: ## Run go test
	go test ./...

lint: ## Run golangci-lint
	golangci-lint run

# Dogfood smoke test: the deterministic layer must gate its own spec
# cleanly (see CLAUDE.md "dogfood-смоук").
dogfood: build ## Build, then check docs/requirements.md (dogfood smoke test)
	./$(BIN_DIR)/$(BINARY) check docs/requirements.md

verify: vet test ## vet + test

ci: build vet test lint dogfood ## build + vet + test + lint + dogfood

clean: ## Remove bin/
	rm -rf $(BIN_DIR)
