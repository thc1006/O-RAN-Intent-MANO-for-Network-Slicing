#!/bin/bash
# Comprehensive Test Runner for O-RAN Intent MANO
# Executes all unit tests with coverage reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WORKSPACE_DIR="${WORKSPACE_DIR:-/workspace}"
TEST_DIR="${WORKSPACE_DIR}/work_dir/tests"
REPORT_DIR="${WORKSPACE_DIR}/work_dir/reports"
COVERAGE_FILE="${REPORT_DIR}/coverage.out"
COVERAGE_HTML="${REPORT_DIR}/coverage.html"

echo -e "${BLUE}🧪 O-RAN Intent MANO Test Suite${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo ""

# Create reports directory
mkdir -p "${REPORT_DIR}"

# Track test results
FAILED_TESTS=()
PASSED_TESTS=()
TOTAL_TESTS=0

# Function to run a test suite
run_test_suite() {
    local test_name=$1
    local test_pattern=$2

    echo -e "${YELLOW}▶ Running ${test_name}...${NC}"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if go test -v -race -coverprofile="${REPORT_DIR}/coverage-${test_name}.out" \
        -run "${test_pattern}" "${TEST_DIR}" 2>&1 | tee "${REPORT_DIR}/${test_name}.log"; then
        echo -e "${GREEN}✓ ${test_name} PASSED${NC}"
        PASSED_TESTS+=("${test_name}")
    else
        echo -e "${RED}✗ ${test_name} FAILED${NC}"
        FAILED_TESTS+=("${test_name}")
    fi
    echo ""
}

# Change to test directory
cd "${TEST_DIR}"

# Run all test suites
run_test_suite "Nephio-Renderer" "TestNephioRenderer"
run_test_suite "O2-Client" "TestO2Client"
run_test_suite "ConfigWatcher" "TestConfigWatcher"
run_test_suite "ParseFloat64" "TestParseFloat64"
run_test_suite "VXLAN-Optimized" "TestVXLAN"

# Generate combined coverage report
echo -e "${BLUE}📊 Generating Coverage Report...${NC}"
go test -v -race -coverprofile="${COVERAGE_FILE}" -covermode=atomic "${TEST_DIR}" > /dev/null 2>&1 || true

if [ -f "${COVERAGE_FILE}" ]; then
    # Generate HTML coverage report
    go tool cover -html="${COVERAGE_FILE}" -o "${COVERAGE_HTML}"

    # Calculate coverage percentage
    COVERAGE=$(go tool cover -func="${COVERAGE_FILE}" | grep total | awk '{print $3}')
    echo -e "${GREEN}Coverage: ${COVERAGE}${NC}"
fi

# Print summary
echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}📋 Test Summary${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════${NC}"
echo -e "Total Tests: ${TOTAL_TESTS}"
echo -e "${GREEN}Passed: ${#PASSED_TESTS[@]}${NC}"
echo -e "${RED}Failed: ${#FAILED_TESTS[@]}${NC}"

if [ ${#PASSED_TESTS[@]} -gt 0 ]; then
    echo ""
    echo -e "${GREEN}✓ Passed Tests:${NC}"
    for test in "${PASSED_TESTS[@]}"; do
        echo -e "  ${GREEN}✓${NC} ${test}"
    done
fi

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo ""
    echo -e "${RED}✗ Failed Tests:${NC}"
    for test in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗${NC} ${test}"
    done
    echo ""
    echo -e "${RED}❌ Some tests failed. Check logs in ${REPORT_DIR}${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ All tests passed successfully!${NC}"
echo -e "${BLUE}📁 Reports available at: ${REPORT_DIR}${NC}"
echo ""

exit 0