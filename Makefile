# proto2mcp — Generate type-safe MCP servers from Protobuf definitions
.DEFAULT_GOAL := help

# ──────────────────────────────────────────────
# Build
# ──────────────────────────────────────────────

.PHONY: build
build: ## Build the protoc plugin binary
	go build -o bin/protoc-gen-proto2mcp ./cmd/protoc-gen-proto2mcp

.PHONY: install
install: ## Install the plugin to $GOPATH/bin
	go install ./cmd/protoc-gen-proto2mcp

# ──────────────────────────────────────────────
# Test
# ──────────────────────────────────────────────

.PHONY: test
test: ## Run all tests with race detector
	go test -race -count=1 ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage report
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -func=coverage.txt

.PHONY: test-fuzz
test-fuzz: ## Run fuzz tests for 30 seconds each
	go test -fuzz=FuzzValidateToolName -fuzztime=30s ./pkg/extract/
	go test -fuzz=FuzzProtoKindToJSONType -fuzztime=30s ./pkg/extract/
	go test -fuzz=FuzzWellKnownSchema -fuzztime=30s ./pkg/extract/
	go test -fuzz=FuzzUnmarshalToolInput -fuzztime=30s ./pkg/mcpruntime/
	go test -fuzz=FuzzSanitizeErrorMessage -fuzztime=30s ./pkg/mcpruntime/

.PHONY: bench
bench: ## Run benchmarks
	go test -bench=. -benchmem ./...

# ──────────────────────────────────────────────
# Lint
# ──────────────────────────────────────────────

.PHONY: lint
lint: ## Run go vet
	go vet ./...

.PHONY: lint-full
lint-full: lint ## Run golangci-lint (requires golangci-lint installed)
	golangci-lint run ./...

# ──────────────────────────────────────────────
# Generate
# ──────────────────────────────────────────────

.PHONY: generate
generate: ## Regenerate proto files
	buf generate

.PHONY: golden-update
golden-update: ## Update golden test files
	go test -run TestGoldenFiles -update ./pkg/emit/
	go test -run TestE2EGolden -update ./pkg/emit/

# ──────────────────────────────────────────────
# Release
# ──────────────────────────────────────────────

.PHONY: release-dry
release-dry: ## Dry-run goreleaser
	goreleaser release --snapshot --clean

# ──────────────────────────────────────────────
# Clean
# ──────────────────────────────────────────────

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin/ dist/ coverage.txt

# ──────────────────────────────────────────────
# CI (composite targets)
# ──────────────────────────────────────────────

.PHONY: ci
ci: lint test build ## Run full CI pipeline locally (lint + test + build)

# ──────────────────────────────────────────────
# Help
# ──────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
