BINARY := tumanomir
BIN_DIR := bin

.PHONY: build vet test lint dogfood verify ci clean

build:
	go build -o $(BIN_DIR)/$(BINARY) ./cmd/tumanomir

vet:
	go vet ./...

test:
	go test ./...

lint:
	golangci-lint run

# Dogfood smoke test: the deterministic layer must gate its own spec
# cleanly (see CLAUDE.md "dogfood-смоук").
dogfood: build
	./$(BIN_DIR)/$(BINARY) check docs/requirements.md

verify: vet test

ci: build vet test lint dogfood

clean:
	rm -rf $(BIN_DIR)
