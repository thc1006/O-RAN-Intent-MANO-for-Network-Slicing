#!/bin/bash
# Test runner script for O-RAN Intent MANO components

set -e

echo "Starting test execution for O-RAN Intent MANO..."
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test summary
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to run tests for a Go module
run_go_tests() {
    local module=$1
    echo -e "\n${YELLOW}Testing $module...${NC}"

    if [ -d "$module" ] && [ -f "$module/go.mod" ]; then
        cd "$module"

        # Run tests with coverage
        if go test ./... -v -short -cover 2>&1; then
            echo -e "${GREEN}✓ $module tests passed${NC}"
            ((PASSED_TESTS++))
        else
            echo -e "${RED}✗ $module tests failed${NC}"
            ((FAILED_TESTS++))
        fi

        ((TOTAL_TESTS++))
        cd - > /dev/null
    else
        echo -e "${YELLOW}⚠ $module not found or not a Go module${NC}"
    fi
}

# Function to run Python tests
run_python_tests() {
    local module=$1
    echo -e "\n${YELLOW}Testing Python module $module...${NC}"

    if [ -d "$module" ]; then
        if python -m pytest "$module" -v --tb=short 2>&1; then
            echo -e "${GREEN}✓ $module tests passed${NC}"
            ((PASSED_TESTS++))
        else
            echo -e "${RED}✗ $module tests failed or no tests found${NC}"
            ((FAILED_TESTS++))
        fi
        ((TOTAL_TESTS++))
    else
        echo -e "${YELLOW}⚠ $module not found${NC}"
    fi
}

# Test Go modules
echo -e "\n${YELLOW}Running Go module tests...${NC}"
run_go_tests "orchestrator"
run_go_tests "cn-dms"
run_go_tests "ran-dms"
run_go_tests "o2-client"
run_go_tests "tn"
run_go_tests "adapters/vnf-operator"

# Test Python modules
echo -e "\n${YELLOW}Running Python module tests...${NC}"
run_python_tests "nlp"
run_python_tests "experiments"

# Generate test report
echo -e "\n${YELLOW}Generating test report...${NC}"
cat > test-report.json << EOF
{
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "total_modules": $TOTAL_TESTS,
    "passed": $PASSED_TESTS,
    "failed": $FAILED_TESTS,
    "success_rate": $(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc -l 2>/dev/null || echo "0")
}
EOF

# Summary
echo -e "\n================================================"
echo -e "${YELLOW}TEST SUMMARY${NC}"
echo -e "Total modules tested: $TOTAL_TESTS"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "\n${RED}Some tests failed. Please review the output above.${NC}"
    exit 1
fi