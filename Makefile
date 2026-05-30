.DEFAULT_GOAL := help

# overridable so CI can drive cross-compiles and race runs without forking the recipes
BINARY      ?= squares-api
BIN_DIR     ?= bin
MAIN        ?= ./cmd/main.go
OUT         ?= $(BIN_DIR)/$(BINARY)
COVER_FILE  ?= coverage.out
COVER_TOOL  := github.com/vladopajic/go-test-coverage/v2@latest
SWAG        := go run github.com/swaggo/swag/cmd/swag@v1.16.6
RACE        ?=
BUILD_FLAGS ?=
LDFLAGS     ?=

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'

.PHONY: run
run: ## Run the API locally
	go run $(MAIN)

.PHONY: build
build: ## Build the binary (override OUT/MAIN/BUILD_FLAGS/LDFLAGS for cross-compiles)
	go build $(BUILD_FLAGS) $(if $(LDFLAGS),-ldflags="$(LDFLAGS)",) -o $(OUT) $(MAIN)

.PHONY: verify
verify: ## Verify go module dependencies
	go mod verify

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run unit tests (no Docker required)
	go test $(RACE) ./internal/... ./cmd/...

.PHONY: test-integration
test-integration: ## Run integration tests (requires Docker)
	go test $(RACE) ./test/...

.PHONY: test-all
test-all: ## Run all tests
	go test $(RACE) ./...

.PHONY: cover
cover: ## Run unit tests with coverage and enforce the threshold
	go test $(RACE) -coverprofile=$(COVER_FILE) ./internal/... ./cmd/...
	go run $(COVER_TOOL) --config .testcoverage.yml

.PHONY: cover-html
cover-html: ## Generate an HTML coverage report from the coverage profile
	go tool cover -html=$(COVER_FILE) -o coverage.html

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: swag
swag: ## Regenerate swagger docs
	$(SWAG) init -g $(MAIN) -o docs

.PHONY: swag-check
swag-check: ## Fail if committed swagger docs are out of date
	$(SWAG) init -g $(MAIN) -o docs
	git diff --exit-code docs

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: deps
deps: ## Upgrade dependencies and tidy
	go get -u -t ./...
	go mod tidy

.PHONY: nats
nats: ## Port-forward a NATS server for local dev
	while true; do kubectl port-forward svc/nats 4222:4222 -n nats; done

.PHONY: clean
clean: ## Remove build and coverage artifacts
	rm -rf $(BIN_DIR) $(COVER_FILE) coverage.html
