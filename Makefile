BINARY := unraid
BUILD_DIR := bin
DIST_DIR := dist
SCHEMA := graphql/schema.graphql
SCHEMA_HASH := graphql/schema.sha256

.PHONY: all build test lint clean generate generate-check schema-fetch schema-check build-all

all: lint test build

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/unraid

test:
	go test ./...

lint:
	go vet ./...
	golangci-lint run

clean:
	rm -rf $(BUILD_DIR)

generate:
	go generate ./internal/client/

## Check that generated files are up to date (run generate and diff).
generate-check: generate
	git diff --exit-code internal/client/generated.go internal/client/introspect_gen.go

## Fetch the latest schema from Apollo and update the stored hash.
schema-fetch:
	rover graph fetch Unraid-API@current > $(SCHEMA)
	shasum -a 256 $(SCHEMA) | awk '{print $$1}' > $(SCHEMA_HASH)
	@echo "Schema updated. Hash: $$(cat $(SCHEMA_HASH))"

## Check if the remote schema has changed since last fetch.
schema-check:
	@LATEST=$$(rover graph fetch Unraid-API@current | shasum -a 256 | awk '{print $$1}'); \
	STORED=$$(cat $(SCHEMA_HASH) 2>/dev/null || echo "none"); \
	if [ "$$LATEST" = "$$STORED" ]; then \
		echo "Schema is up to date. ($$STORED)"; \
	else \
		echo "Schema has changed!"; \
		echo "  Local:  $$STORED"; \
		echo "  Remote: $$LATEST"; \
		echo "Run 'make schema-fetch generate' to update."; \
		exit 1; \
	fi

# Cross-compilation targets
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-linux-amd64       ./cmd/unraid

build-linux-arm64:
	mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-linux-arm64       ./cmd/unraid

build-darwin-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin  GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-darwin-amd64      ./cmd/unraid

build-darwin-arm64:
	mkdir -p $(DIST_DIR)
	GOOS=darwin  GOARCH=arm64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-darwin-arm64      ./cmd/unraid

build-windows-amd64:
	mkdir -p $(DIST_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe ./cmd/unraid
