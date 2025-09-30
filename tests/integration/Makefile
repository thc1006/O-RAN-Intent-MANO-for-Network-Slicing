# Makefile for E2E Integration Tests

.PHONY: help test test-unit test-integration test-e2e test-all clean build up down logs coverage docker-build docker-clean

# Variables
DOCKER_COMPOSE := docker-compose -f docker-compose.e2e.yml
GO_TEST := go test -v -timeout=10m
GO_TEST_TAGS := -tags=integration
COVERAGE_FILE := output/coverage.out
COVERAGE_HTML := output/coverage.html

# Default target
help:
	@echo "O-RAN Intent MANO - E2E Integration Tests"
	@echo ""
	@echo "Available targets:"
	@echo "  make test              - Run all E2E tests with Docker Compose"
	@echo "  make test-local        - Run tests locally (without Docker)"
	@echo "  make test-short        - Run tests in short mode (skip integration)"
	@echo "  make test-coverage     - Run tests with coverage report"
	@echo "  make test-specific T=X - Run specific test matching pattern X"
	@echo "  make build             - Build test Docker image"
	@echo "  make up                - Start test environment"
	@echo "  make down              - Stop test environment"
	@echo "  make logs              - View test logs"
	@echo "  make clean             - Clean up test artifacts"
	@echo "  make coverage          - Generate coverage report"
	@echo "  make docker-build      - Build all Docker images"
	@echo "  make docker-clean      - Remove all Docker artifacts"
	@echo ""

# Run all E2E tests with Docker Compose
test: docker-build
	@echo "Running E2E integration tests..."
	$(DOCKER_COMPOSE) up --build --abort-on-container-exit
	$(DOCKER_COMPOSE) logs e2e-test
	@echo "Tests completed. Check output/ directory for results."

# Run tests locally without Docker
test-local:
	@echo "Running E2E tests locally..."
	@mkdir -p output
	$(GO_TEST) $(GO_TEST_TAGS) ./e2e/...

# Run tests in short mode
test-short:
	@echo "Running tests in short mode..."
	go test -short -v ./e2e/...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p output
	$(GO_TEST) $(GO_TEST_TAGS) -coverprofile=$(COVERAGE_FILE) ./e2e/...
	go tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Run specific test by name pattern
test-specific:
ifndef T
	@echo "Error: Test pattern not specified. Usage: make test-specific T=TestName"
	@exit 1
endif
	@echo "Running tests matching pattern: $(T)"
	$(GO_TEST) $(GO_TEST_TAGS) -run $(T) ./e2e/...

# Build test Docker image
build:
	@echo "Building test Docker image..."
	$(DOCKER_COMPOSE) build e2e-test

# Start test environment
up:
	@echo "Starting test environment..."
	$(DOCKER_COMPOSE) up -d
	@echo "Waiting for services to be ready..."
	@sleep 10
	@echo "Test environment ready!"

# Stop test environment
down:
	@echo "Stopping test environment..."
	$(DOCKER_COMPOSE) down

# View test logs
logs:
	$(DOCKER_COMPOSE) logs -f e2e-test

# Clean up test artifacts
clean:
	@echo "Cleaning up test artifacts..."
	rm -rf output/
	rm -rf testdata/generated-packages/
	$(DOCKER_COMPOSE) down -v
	@echo "Clean complete."

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@if [ ! -f $(COVERAGE_FILE) ]; then \
		echo "No coverage file found. Run 'make test-coverage' first."; \
		exit 1; \
	fi
	go tool cover -html=$(COVERAGE_FILE)

# Build all Docker images
docker-build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

# Clean Docker artifacts
docker-clean:
	@echo "Removing Docker artifacts..."
	$(DOCKER_COMPOSE) down -v --rmi all
	docker system prune -f

# Run provisioning test
test-provisioning:
	@echo "Running slice provisioning test..."
	$(GO_TEST) $(GO_TEST_TAGS) -run TestEndToEndSliceProvisioning ./e2e/...

# Run error handling test
test-errors:
	@echo "Running error handling test..."
	$(GO_TEST) $(GO_TEST_TAGS) -run TestEndToEndSliceProvisioningErrorHandling ./e2e/...

# Run edge case test
test-edge:
	@echo "Running edge case test..."
	$(GO_TEST) $(GO_TEST_TAGS) -run TestEndToEndSliceProvisioningEdgeCases ./e2e/...

# Run component integration test
test-components:
	@echo "Running component integration test..."
	$(GO_TEST) $(GO_TEST_TAGS) -run TestIntegrationWithMockedComponents ./e2e/...

# Health check for test environment
health-check:
	@echo "Checking test environment health..."
	@curl -f http://localhost:8080/mockserver/status || echo "Orchestrator: DOWN"
	@curl -f http://localhost:8081/mockserver/status || echo "VNF Operator: DOWN"
	@curl -f http://localhost:8082/mockserver/status || echo "Nephio Renderer: DOWN"
	@curl -f http://localhost:8083/mockserver/status || echo "TN Agent: DOWN"

# Initialize test data
init-testdata:
	@echo "Initializing test data..."
	@mkdir -p testdata/generated-packages/upf-package/resources
	@mkdir -p testdata/generated-packages/amf-package/resources
	@mkdir -p testdata/generated-packages/smf-package/resources
	@mkdir -p testdata/sql
	@mkdir -p output
	@echo "Test data directories created."

# Run linting on test code
lint:
	@echo "Running linters..."
	golangci-lint run ./e2e/...

# Format test code
fmt:
	@echo "Formatting test code..."
	go fmt ./e2e/...

# Vet test code
vet:
	@echo "Vetting test code..."
	go vet ./e2e/...

# Run all checks (fmt, vet, lint)
check: fmt vet lint
	@echo "All checks passed!"

# Watch tests (requires entr)
watch:
	@echo "Watching for changes..."
	find . -name "*.go" | entr -c make test-local

# Benchmark tests
benchmark:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./e2e/...

# Profile tests
profile:
	@echo "Profiling tests..."
	@mkdir -p output
	go test -cpuprofile=output/cpu.prof -memprofile=output/mem.prof ./e2e/...
	@echo "Profiles generated in output/"

# CI target for continuous integration
ci: docker-build test test-coverage
	@echo "CI pipeline completed successfully!"