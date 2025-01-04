SHELL := /bin/bash
.DEFAULT_GOAL := help

# Project variables
PROJECT_NAME := go-bootstrap
PKG := github.com/damianoneill/$(PROJECT_NAME)
PKG_LIST := $(shell go list ./... | grep -v /vendor/)
# Test specific package list - excludes examples and generated code
TEST_PKG_LIST := $(shell go list ./... | grep -v /vendor/ | grep -Ev '(/examples/|/mocks)')

# Tool versions
GOLANGCI_LINT_VERSION := v1.55.2

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(COMMIT_HASH) -X main.BuildTime=$(BUILD_TIME)"

# Directories
DIST_DIR := dist
COVERAGE_DIR := $(DIST_DIR)/coverage
EXAMPLES_DIR := $(DIST_DIR)/examples

# Set GOBIN to GOPATH/bin if not already set
export GOBIN ?= $(shell go env GOPATH)/bin
# Add GOBIN to PATH for all targets
export PATH := $(GOBIN):$(PATH)

.PHONY: help
help: ## Display this help message
	@cat $(MAKEFILE_LIST) | grep -e "^[a-zA-Z_\-]*: *.*## *" | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean: ## Clean build artifacts and generated code
	@echo "Cleaning build artifacts and generated code..."
	@rm -rf $(DIST_DIR)
	@rm -rf pkg/domain/mocks
	@find . -type f -name '*.test' -delete
	@find . -type f -name 'coverage.out' -delete
	@find . -type f -name 'coverage.html' -delete

.PHONY: dirs
dirs: ## Create required directories
	@echo "Creating required directories..."
	@mkdir -p $(COVERAGE_DIR)
	@mkdir -p $(EXAMPLES_DIR)
	@mkdir -p pkg/domain/mocks

.PHONY: tools
tools: ## Install development tools
	@echo "Installing tools..."
	@cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

.PHONY: lint
lint: ## Run linters
	@echo "Running linters..."
	@if ! command -v $(GOBIN)/golangci-lint &> /dev/null; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION); \
	fi
	@$(GOBIN)/golangci-lint run

.PHONY: fmt
fmt: ## Format code
	@echo "Formatting code..."
	@$(GOBIN)/goimports -w -local $(PKG) .
	@go fmt ./...

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

.PHONY: test
test: dirs ## Run tests with coverage
	@echo "Running tests..."
	@go test -race -cover -coverprofile=$(COVERAGE_DIR)/coverage.out $(TEST_PKG_LIST)
	@go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated: $(COVERAGE_DIR)/coverage.html"
	@go tool cover -func=$(COVERAGE_DIR)/coverage.out | grep total: | awk '{print "Total coverage: " $$3}'

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "Running integration tests..."
	@go test -tags=integration ./...

.PHONY: build-examples
build-examples: dirs ## Build example applications
	@echo "Building examples..."
	@find examples -name main.go -exec sh -c '\
		dir=$$(dirname "{}"); \
		name=$${dir#examples/}; \
		echo "Building $$name..."; \
		go build $(LDFLAGS) -o $(EXAMPLES_DIR)/$$name "{}";' \;

.PHONY: check
check: fmt vet lint test ## Run all checks

.PHONY: generate
generate: ## Run go generate
	@echo "Running go generate..."
	@go generate ./...

.PHONY: mock
mock: dirs ## Generate mocks
	@echo "Generating mocks..."
	@go generate ./pkg/domain/...

.PHONY: tidy
tidy: ## Tidy and verify go modules
	@echo "Tidying modules..."
	@go mod tidy
	@go mod verify

.PHONY: vuln
vuln: ## Check for vulnerabilities
	@echo "Checking for vulnerabilities..."
	@$(GOBIN)/govulncheck ./...

.PHONY: version
version: ## Display version information
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT_HASH)"
	@echo "Built:      $(BUILD_TIME)"

.PHONY: all
all: clean tools dirs mock check build-examples ## Run all targets

# Development targets
.PHONY: dev
dev: tidy tools mock ## Prepare for development
	@echo "Development environment ready"

# CI targets
.PHONY: ci
ci: check ## Run CI checks
	@echo "CI checks complete"