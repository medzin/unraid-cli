BINARY := unraid
BUILD_DIR := bin
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
build-all: build-linux-amd64 build-macos-amd64 build-macos-arm64 build-windows-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/unraid

build-macos-amd64:
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-macos-amd64 ./cmd/unraid

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY)-macos-arm64 ./cmd/unraid

build-windows-amd64:
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/unraid
