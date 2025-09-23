# GitOps Validation Framework Makefile
# O-RAN Intent MANO Project

SHELL := /bin/bash
.PHONY: help build test clean lint format vet deps install run-validation run-e2e docker-build docker-push

# Configuration
APP_NAME := gitops-validator
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# Go configuration
GO := go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Directories
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

# Docker configuration
DOCKER_REGISTRY ?= ghcr.io/thc1006
DOCKER_IMAGE := $(DOCKER_REGISTRY)/$(APP_NAME)
DOCKER_TAG := $(VERSION)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "GitOps Validation Framework"
	@echo "==========================="
	@echo ""
	@echo "Available targets:"
	@echo ""
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## /  /'

## deps: Download and install dependencies
deps:
	@echo -e "$(YELLOW)Installing dependencies...$(NC)"
	$(GO) mod download
	$(GO) mod tidy
	@echo -e "$(GREEN)Dependencies installed$(NC)"

## build: Build the validation framework binary
build: deps
	@echo -e "$(YELLOW)Building $(APP_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) ./main.go
	@echo -e "$(GREEN)Build complete: $(BUILD_DIR)/$(APP_NAME)$(NC)"

## build-all: Build for multiple platforms
build-all: deps
	@echo -e "$(YELLOW)Building for multiple platforms...$(NC)"
	@mkdir -p $(DIST_DIR)

	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-amd64 ./main.go

	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-linux-arm64 ./main.go

	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-windows-amd64.exe ./main.go

	# macOS AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-amd64 ./main.go

	# macOS ARM64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		$(GO) build $(LDFLAGS) -o $(DIST_DIR)/$(APP_NAME)-darwin-arm64 ./main.go

	@echo -e "$(GREEN)Multi-platform build complete$(NC)"
	@ls -la $(DIST_DIR)/

## test: Run all tests
test: deps
	@echo -e "$(YELLOW)Running tests...$(NC)"
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo -e "$(GREEN)Tests complete. Coverage report: $(COVERAGE_DIR)/coverage.html$(NC)"

## test-unit: Run unit tests only
test-unit: deps
	@echo -e "$(YELLOW)Running unit tests...$(NC)"
	$(GO) test -v -short ./...

## test-integration: Run integration tests
test-integration: deps
	@echo -e "$(YELLOW)Running integration tests...$(NC)"
	$(GO) test -v -tags=integration ./...

## benchmark: Run benchmarks
benchmark: deps
	@echo -e "$(YELLOW)Running benchmarks...$(NC)"
	$(GO) test -v -bench=. -benchmem ./...

## lint: Run linting
lint: deps
	@echo -e "$(YELLOW)Running linting...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo -e "$(RED)golangci-lint not found. Please install it.$(NC)"; \
		echo "Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)Linting complete$(NC)"

## format: Format Go code
format:
	@echo -e "$(YELLOW)Formatting code...$(NC)"
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi
	@echo -e "$(GREEN)Code formatted$(NC)"

## vet: Run go vet
vet: deps
	@echo -e "$(YELLOW)Running go vet...$(NC)"
	$(GO) vet ./...
	@echo -e "$(GREEN)Vet complete$(NC)"

## check: Run all quality checks
check: format vet lint test
	@echo -e "$(GREEN)All checks passed!$(NC)"

## install: Install the binary to GOPATH/bin
install: build
	@echo -e "$(YELLOW)Installing $(APP_NAME)...$(NC)"
	$(GO) install $(LDFLAGS) ./main.go
	@echo -e "$(GREEN)$(APP_NAME) installed to $(shell go env GOPATH)/bin$(NC)"

## run-validation: Run validation with default config
run-validation: build
	@echo -e "$(YELLOW)Running validation...$(NC)"
	./$(BUILD_DIR)/$(APP_NAME) --config=config.yaml --validate-only

## run-e2e: Run E2E pipeline
run-e2e: build
	@echo -e "$(YELLOW)Running E2E pipeline...$(NC)"
	./$(BUILD_DIR)/$(APP_NAME) --config=config.yaml --validate-only --enable-drift --enable-metrics

## run-continuous: Run continuous monitoring
run-continuous: build
	@echo -e "$(YELLOW)Starting continuous monitoring...$(NC)"
	./$(BUILD_DIR)/$(APP_NAME) --config=config.yaml --interval=5m

## docker-build: Build Docker image
docker-build: build
	@echo -e "$(YELLOW)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		.
	@echo -e "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

## docker-push: Push Docker image to registry
docker-push: docker-build
	@echo -e "$(YELLOW)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	docker push $(DOCKER_IMAGE):latest
	@echo -e "$(GREEN)Docker image pushed$(NC)"

## docker-run: Run Docker container locally
docker-run: docker-build
	@echo -e "$(YELLOW)Running Docker container...$(NC)"
	docker run --rm -it \
		-v $(PWD)/config.yaml:/app/config.yaml \
		-v $(HOME)/.kube:/root/.kube:ro \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		--config=/app/config.yaml --validate-only

## clean: Clean build artifacts
clean:
	@echo -e "$(YELLOW)Cleaning build artifacts...$(NC)"
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	$(GO) clean
	@echo -e "$(GREEN)Clean complete$(NC)"

## deps-update: Update dependencies
deps-update:
	@echo -e "$(YELLOW)Updating dependencies...$(NC)"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo -e "$(GREEN)Dependencies updated$(NC)"

## deps-vendor: Vendor dependencies
deps-vendor: deps
	@echo -e "$(YELLOW)Vendoring dependencies...$(NC)"
	$(GO) mod vendor
	@echo -e "$(GREEN)Dependencies vendored$(NC)"

## security-scan: Run security scan
security-scan:
	@echo -e "$(YELLOW)Running security scan...$(NC)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo -e "$(RED)gosec not found. Please install it.$(NC)"; \
		echo "Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)Security scan complete$(NC)"

## generate-mocks: Generate mocks for testing
generate-mocks:
	@echo -e "$(YELLOW)Generating mocks...$(NC)"
	@if command -v mockgen >/dev/null 2>&1; then \
		mockgen -source=validator.go -destination=mocks/mock_validator.go; \
	else \
		echo -e "$(RED)mockgen not found. Please install it.$(NC)"; \
		echo "Install with: go install github.com/golang/mock/mockgen@latest"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)Mocks generated$(NC)"

## docs: Generate documentation
docs:
	@echo -e "$(YELLOW)Generating documentation...$(NC)"
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Documentation server starting at http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo -e "$(RED)godoc not found. Please install it.$(NC)"; \
		echo "Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
		exit 1; \
	fi

## version: Show version information
version:
	@echo "App Name: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell go version)"

## info: Show build information
info:
	@echo "GitOps Validation Framework Build Information"
	@echo "============================================="
	@echo "App Name: $(APP_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Go Version: $(shell go version)"
	@echo "Platform: $(GOOS)/$(GOARCH)"
	@echo "CGO Enabled: $(CGO_ENABLED)"
	@echo ""
	@echo "Docker Configuration:"
	@echo "Registry: $(DOCKER_REGISTRY)"
	@echo "Image: $(DOCKER_IMAGE)"
	@echo "Tag: $(DOCKER_TAG)"

# Default target
.DEFAULT_GOAL := help