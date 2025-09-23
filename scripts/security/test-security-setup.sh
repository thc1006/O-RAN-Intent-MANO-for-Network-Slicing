#!/bin/bash
# Test script to validate security scanning setup
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Testing O-RAN Intent-MANO Security Setup${NC}"
echo "=================================================="

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to run tests
run_test() {
    local test_name="$1"
    local test_command="$2"

    echo -e "\n${YELLOW}Testing: $test_name${NC}"

    if eval "$test_command"; then
        echo -e "${GREEN}✅ PASSED: $test_name${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAILED: $test_name${NC}"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Configuration files validation
run_test "Pre-commit configuration validation" \
    "python -c 'import yaml; yaml.safe_load(open(\".pre-commit-config.yaml\"))' 2>/dev/null"

run_test "Hadolint configuration validation" \
    "python -c 'import yaml; yaml.safe_load(open(\".hadolint.yaml\"))' 2>/dev/null"

run_test "SAST configuration validation" \
    "node -e 'JSON.parse(require(\"fs\").readFileSync(\".sast-config.json\", \"utf8\"))' 2>/dev/null"

# Test 2: Security scripts validation
SECURITY_SCRIPTS=(
    "scripts/security/run-gosec.sh"
    "scripts/security/check-dockerfile-security.sh"
    "scripts/security/validate-network-policies.sh"
    "scripts/security/validate-rbac.sh"
    "scripts/security/validate-security-policies.sh"
    "scripts/security/check-license-headers.sh"
)

for script in "${SECURITY_SCRIPTS[@]}"; do
    run_test "Security script exists and is executable: $(basename $script)" \
        "[ -f '$script' ] && [ -x '$script' ]"

    run_test "Security script syntax: $(basename $script)" \
        "bash -n '$script'"
done

# Test 3: GitHub Actions workflow validation
WORKFLOWS=(
    ".github/workflows/ci.yml"
    ".github/workflows/security.yml"
    ".github/workflows/security-enhanced.yml"
)

for workflow in "${WORKFLOWS[@]}"; do
    run_test "Workflow exists: $(basename $workflow)" \
        "[ -f '$workflow' ]"

    run_test "Workflow has security permissions: $(basename $workflow)" \
        "grep -q 'security-events: write' '$workflow'"

    run_test "Workflow YAML syntax: $(basename $workflow)" \
        "python -c 'import yaml; yaml.safe_load(open(\"$workflow\"))' 2>/dev/null"
done

# Test 4: Security tool integration checks
run_test "Gosec integration in CI workflow" \
    "grep -q 'gosec' .github/workflows/ci.yml"

run_test "Checkov integration in security workflow" \
    "grep -q 'checkov' .github/workflows/security.yml"

run_test "Trivy integration in security workflow" \
    "grep -q 'trivy' .github/workflows/security.yml"

# Test 5: Documentation validation
run_test "Security documentation exists" \
    "[ -f '.claude/ci-security.md' ]"

run_test "Security documentation has required sections" \
    "grep -q 'Security Scanning Architecture' .claude/ci-security.md && \
     grep -q 'Pre-commit Security Hooks' .claude/ci-security.md && \
     grep -q 'Incident Response' .claude/ci-security.md"

# Test 6: File structure validation
run_test "Security scripts directory structure" \
    "[ -d 'scripts/security' ]"

run_test "Claude documentation directory" \
    "[ -d '.claude' ]"

# Test 7: Sample security policy validation
run_test "Network policies exist" \
    "find . -name '*network*policy*.yaml' -o -name '*network*policy*.yml' | grep -q '.'"

run_test "RBAC policies exist" \
    "find . -name '*rbac*.yaml' -o -name '*rbac*.yml' | grep -q '.'"

# Test 8: Pre-commit hook validation
run_test "Pre-commit hooks configuration completeness" \
    "grep -q 'gosec' .pre-commit-config.yaml && \
     grep -q 'checkov' .pre-commit-config.yaml && \
     grep -q 'hadolint' .pre-commit-config.yaml"

# Test 9: Security scan coverage
run_test "Go security scanning coverage" \
    "grep -q 'gosec' .github/workflows/ci.yml && \
     grep -q 'golangci-lint.*gosec' .github/workflows/ci.yml"

run_test "Container security scanning coverage" \
    "grep -q 'trivy' .github/workflows/security.yml && \
     grep -q 'grype' .github/workflows/security.yml"

run_test "Infrastructure security scanning coverage" \
    "grep -q 'checkov.*kubernetes' .github/workflows/security.yml && \
     grep -q 'checkov.*dockerfile' .github/workflows/security.yml"

# Test 10: Secrets detection setup
run_test "Secrets detection tools configured" \
    "grep -q 'gitleaks' .github/workflows/security.yml && \
     grep -q 'trufflehog' .github/workflows/security.yml"

# Test Summary
echo -e "\n${BLUE}📊 Test Summary${NC}"
echo "================"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
TOTAL_TESTS=$((TESTS_PASSED + TESTS_FAILED))
echo -e "${BLUE}Total Tests: $TOTAL_TESTS${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All security setup tests passed!${NC}"
    echo -e "${GREEN}The O-RAN Intent-MANO security scanning pipeline is properly configured.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some tests failed. Please review and fix the issues above.${NC}"
    PASS_RATE=$(( (TESTS_PASSED * 100) / TOTAL_TESTS ))
    echo -e "${YELLOW}Pass Rate: $PASS_RATE%${NC}"
    exit 1
fi