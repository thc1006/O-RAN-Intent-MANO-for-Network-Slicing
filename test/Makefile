# Makefile for Modern Go Testing - O-RAN Intent MANO Project
# This Makefile provides standardized test execution with 2025 best practices

# Configuration
PROJECT_ROOT := $(shell pwd)/..
TEST_TIMEOUT := 300s
COVERAGE_FILE := test-reports/coverage.out
HTML_COVERAGE := test-reports/coverage.html
MIN_COVERAGE := 80
BENCHMARK_TIME := 3s

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
BLUE := \033[0;34m
NC := \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

# Help target
help: ## Display this help message
	@echo -e "${BLUE}O-RAN Intent MANO - Modern Testing Makefile${NC}"
	@echo -e "${BLUE}===========================================${NC}"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-20s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

# Setup and initialization
init: ## Initialize test environment and dependencies
	@echo -e "${BLUE}Initializing test environment...${NC}"
	@mkdir -p test-reports testdata/golden testdata/fixtures
	@go mod tidy
	@go mod download
	@echo -e "${GREEN}✅ Test environment initialized${NC}"

# Clean targets
clean: ## Clean test artifacts and reports
	@echo -e "${YELLOW}Cleaning test artifacts...${NC}"
	@rm -rf test-reports/
	@rm -f coverage.out coverage.html
	@go clean -testcache
	@echo -e "${GREEN}✅ Test artifacts cleaned${NC}"

clean-cache: ## Clean Go test cache
	@echo -e "${YELLOW}Cleaning Go test cache...${NC}"
	@go clean -testcache
	@echo -e "${GREEN}✅ Test cache cleaned${NC}"

# Basic test execution
test: ## Run all tests with standard settings
	@echo -e "${BLUE}Running all tests...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -v -timeout $(TEST_TIMEOUT) ./...
	@echo -e "${GREEN}✅ All tests completed${NC}"

test-short: ## Run tests in short mode (faster, skip long-running tests)
	@echo -e "${BLUE}Running tests in short mode...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -short -v -timeout $(TEST_TIMEOUT) ./...
	@echo -e "${GREEN}✅ Short tests completed${NC}"

test-race: ## Run tests with race condition detection
	@echo -e "${BLUE}Running tests with race detection...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -race -v -timeout $(TEST_TIMEOUT) ./...
	@echo -e "${GREEN}✅ Race detection tests completed${NC}"

test-parallel: ## Run tests with maximum parallelization
	@echo -e "${BLUE}Running tests with parallel execution...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -parallel 8 -v -timeout $(TEST_TIMEOUT) ./...
	@echo -e "${GREEN}✅ Parallel tests completed${NC}"

# Coverage testing
coverage: ## Generate test coverage report
	@echo -e "${BLUE}Generating coverage report...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@cd $(PROJECT_ROOT) && go tool cover -func=$(COVERAGE_FILE) | tail -1
	@echo -e "${GREEN}✅ Coverage report generated: $(COVERAGE_FILE)${NC}"

coverage-html: coverage ## Generate HTML coverage report
	@echo -e "${BLUE}Generating HTML coverage report...${NC}"
	@cd $(PROJECT_ROOT) && go tool cover -html=$(COVERAGE_FILE) -o=$(HTML_COVERAGE)
	@echo -e "${GREEN}✅ HTML coverage report: $(HTML_COVERAGE)${NC}"
	@echo -e "${BLUE}Open in browser: file://$(shell cd $(PROJECT_ROOT) && pwd)/$(HTML_COVERAGE)${NC}"

coverage-check: coverage ## Check if coverage meets minimum threshold
	@echo -e "${BLUE}Checking coverage threshold ($(MIN_COVERAGE)%)...${NC}"
	@cd $(PROJECT_ROOT) && \
		COVERAGE=$$(go tool cover -func=$(COVERAGE_FILE) | tail -1 | grep -o '[0-9.]*%' | sed 's/%//'); \
		if (( $$(echo "$$COVERAGE >= $(MIN_COVERAGE)" | bc -l) )); then \
			echo -e "${GREEN}✅ Coverage ($$COVERAGE%) meets threshold${NC}"; \
		else \
			echo -e "${RED}❌ Coverage ($$COVERAGE%) below threshold ($(MIN_COVERAGE)%)${NC}"; \
			exit 1; \
		fi

# Specific test categories
test-unit: ## Run unit tests only
	@echo -e "${BLUE}Running unit tests...${NC}"
	@cd $(PROJECT_ROOT) && go test -short -run "^TestUnit|^Test.*Unit" -v ./...
	@echo -e "${GREEN}✅ Unit tests completed${NC}"

test-integration: ## Run integration tests only
	@echo -e "${BLUE}Running integration tests...${NC}"
	@cd $(PROJECT_ROOT) && go test -run "^TestIntegration|^Test.*Integration" -v ./...
	@echo -e "${GREEN}✅ Integration tests completed${NC}"

test-e2e: ## Run end-to-end tests only
	@echo -e "${BLUE}Running E2E tests...${NC}"
	@cd $(PROJECT_ROOT) && go test -run "^TestE2E|^Test.*E2E" -v ./...
	@echo -e "${GREEN}✅ E2E tests completed${NC}"

test-security: ## Run security tests only
	@echo -e "${BLUE}Running security tests...${NC}"
	@cd $(PROJECT_ROOT) && go test -run "^TestSecurity|^TestValidation.*Security|^Fuzz.*" -v ./...
	@echo -e "${GREEN}✅ Security tests completed${NC}"

# Benchmark testing
benchmark: ## Run benchmark tests
	@echo -e "${BLUE}Running benchmark tests...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -bench=. -benchtime=$(BENCHMARK_TIME) -benchmem ./... | tee test-reports/benchmark.txt
	@echo -e "${GREEN}✅ Benchmark tests completed${NC}"

benchmark-compare: ## Compare benchmark results with baseline
	@echo -e "${BLUE}Running benchmark comparison...${NC}"
	@mkdir -p test-reports
	@if [ -f test/benchmarks/baseline.txt ]; then \
		cd $(PROJECT_ROOT) && go test -bench=. -benchtime=$(BENCHMARK_TIME) ./... > test-reports/current.txt; \
		benchcmp test/benchmarks/baseline.txt test-reports/current.txt || true; \
	else \
		echo -e "${YELLOW}⚠️  No baseline found, creating new baseline${NC}"; \
		cd $(PROJECT_ROOT) && go test -bench=. -benchtime=$(BENCHMARK_TIME) ./... | tee test/benchmarks/baseline.txt; \
	fi

# Fuzz testing
fuzz: ## Run fuzz tests for 30 seconds each
	@echo -e "${BLUE}Running fuzz tests...${NC}"
	@cd $(PROJECT_ROOT) && \
		for fuzz in $$(grep -r "func Fuzz" --include="*_test.go" . | cut -d: -f1 | sort -u); do \
			pkg=$$(dirname $$fuzz | sed 's|^\./||'); \
			echo -e "${BLUE}Fuzzing package: $$pkg${NC}"; \
			timeout 30s go test -fuzz=Fuzz -fuzztime=30s ./$$pkg || true; \
		done
	@echo -e "${GREEN}✅ Fuzz testing completed${NC}"

# Package-specific tests
test-security-pkg: ## Run tests for security package specifically
	@echo -e "${BLUE}Testing security package...${NC}"
	@cd $(PROJECT_ROOT) && go test -v -coverprofile=test-reports/security-coverage.out ./pkg/security/...
	@cd $(PROJECT_ROOT) && go tool cover -func=test-reports/security-coverage.out
	@echo -e "${GREEN}✅ Security package tests completed${NC}"

test-orchestrator: ## Run tests for orchestrator package
	@echo -e "${BLUE}Testing orchestrator package...${NC}"
	@cd $(PROJECT_ROOT) && go test -v -coverprofile=test-reports/orchestrator-coverage.out ./orchestrator/...
	@cd $(PROJECT_ROOT) && go tool cover -func=test-reports/orchestrator-coverage.out
	@echo -e "${GREEN}✅ Orchestrator tests completed${NC}"

test-tn: ## Run tests for transport network package
	@echo -e "${BLUE}Testing TN package...${NC}"
	@cd $(PROJECT_ROOT) && go test -v -coverprofile=test-reports/tn-coverage.out ./tn/...
	@cd $(PROJECT_ROOT) && go tool cover -func=test-reports/tn-coverage.out
	@echo -e "${GREEN}✅ TN tests completed${NC}"

# Comprehensive testing
test-all: ## Run comprehensive test suite with coverage
	@echo -e "${BLUE}Running comprehensive test suite...${NC}"
	@$(MAKE) clean
	@$(MAKE) test-race
	@$(MAKE) coverage-html
	@$(MAKE) benchmark
	@$(MAKE) test-security
	@$(MAKE) fuzz
	@echo -e "${GREEN}✅ Comprehensive test suite completed${NC}"

# CI/CD targets
ci-test: ## Run tests suitable for CI/CD pipeline
	@echo -e "${BLUE}Running CI/CD test suite...${NC}"
	@mkdir -p test-reports
	@cd $(PROJECT_ROOT) && go test -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic -v -timeout $(TEST_TIMEOUT) ./...
	@$(MAKE) coverage-check
	@cd $(PROJECT_ROOT) && go tool cover -html=$(COVERAGE_FILE) -o=$(HTML_COVERAGE)
	@echo -e "${GREEN}✅ CI/CD tests completed${NC}"

ci-security: ## Run security tests for CI/CD
	@echo -e "${BLUE}Running CI/CD security tests...${NC}"
	@cd $(PROJECT_ROOT) && go test -run "^TestSecurity|^TestValidation.*Security" -v -timeout 120s ./...
	@echo -e "${GREEN}✅ CI/CD security tests completed${NC}"

# Reporting and analysis
report: ## Generate comprehensive test report
	@echo -e "${BLUE}Generating comprehensive test report...${NC}"
	@cd $(PROJECT_ROOT) && bash scripts/test_coverage_report.sh
	@echo -e "${GREEN}✅ Comprehensive report generated${NC}"

lint-tests: ## Lint test files for best practices
	@echo -e "${BLUE}Linting test files...${NC}"
	@cd $(PROJECT_ROOT) && \
		echo "Checking for t.Parallel() usage in parallel-safe tests..."; \
		echo "Checking for proper t.Cleanup() usage..."; \
		echo "Checking for testify usage patterns..."; \
		find . -name "*_test.go" -exec grep -L "t\.Parallel()" {} \; | head -10 | while read file; do \
			echo -e "${YELLOW}⚠️  Consider adding t.Parallel() to: $$file${NC}"; \
		done
	@echo -e "${GREEN}✅ Test linting completed${NC}"

# Development helpers
test-watch: ## Watch for changes and run tests (requires entr)
	@echo -e "${BLUE}Watching for changes (requires 'entr' tool)...${NC}"
	@if command -v entr >/dev/null 2>&1; then \
		find $(PROJECT_ROOT) -name "*.go" | entr -c $(MAKE) test-short; \
	else \
		echo -e "${RED}❌ 'entr' tool not found. Install with: apt-get install entr${NC}"; \
		exit 1; \
	fi

debug-test: ## Run a specific test with debugging enabled
	@echo -e "${BLUE}Debug mode - specify test with TEST_NAME=TestName${NC}"
	@if [ -z "$(TEST_NAME)" ]; then \
		echo -e "${RED}❌ Please specify TEST_NAME=TestName${NC}"; \
		exit 1; \
	fi
	@cd $(PROJECT_ROOT) && go test -v -run "$(TEST_NAME)" ./...

# Modernization helpers
modernize-check: ## Check test modernization status
	@echo -e "${BLUE}Checking test modernization status...${NC}"
	@cd $(PROJECT_ROOT) && \
		echo "=== Test Modernization Report ==="; \
		echo "t.Parallel() usage: $$(grep -r 't\.Parallel()' --include='*_test.go' . | wc -l)"; \
		echo "t.Cleanup() usage: $$(grep -r 't\.Cleanup(' --include='*_test.go' . | wc -l)"; \
		echo "testify/assert usage: $$(grep -r 'testify/assert' --include='*_test.go' . | wc -l)"; \
		echo "testify/suite usage: $$(grep -r 'testify/suite' --include='*_test.go' . | wc -l)"; \
		echo "Table-driven tests: $$(grep -r 'tests := \[\]struct' --include='*_test.go' . | wc -l)"; \
		echo "Benchmark tests: $$(grep -r 'func Benchmark' --include='*_test.go' . | wc -l)"; \
		echo "Fuzz tests: $$(grep -r 'func Fuzz' --include='*_test.go' . | wc -l)"
	@echo -e "${GREEN}✅ Modernization check completed${NC}"

# Utilities
deps: ## Download and verify test dependencies
	@echo -e "${BLUE}Downloading test dependencies...${NC}"
	@cd $(PROJECT_ROOT) && go mod download
	@cd $(PROJECT_ROOT) && go mod verify
	@echo -e "${GREEN}✅ Dependencies verified${NC}"

update-deps: ## Update test dependencies to latest versions
	@echo -e "${BLUE}Updating dependencies...${NC}"
	@cd $(PROJECT_ROOT) && go get -u ./...
	@cd $(PROJECT_ROOT) && go mod tidy
	@echo -e "${GREEN}✅ Dependencies updated${NC}"

# Documentation
docs: ## Generate test documentation
	@echo -e "${BLUE}Generating test documentation...${NC}"
	@cd $(PROJECT_ROOT) && go doc -all ./test/framework > docs/TEST_FRAMEWORK_DOCS.md
	@echo -e "${GREEN}✅ Test documentation generated${NC}"

# Phony targets
.PHONY: help init clean clean-cache test test-short test-race test-parallel \
        coverage coverage-html coverage-check test-unit test-integration test-e2e \
        test-security benchmark benchmark-compare fuzz test-security-pkg \
        test-orchestrator test-tn test-all ci-test ci-security report lint-tests \
        test-watch debug-test modernize-check deps update-deps docs