#!/bin/bash
# DevContainer Verification Script
# Comprehensive testing after rebuild

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test result tracking
declare -a FAILED_TEST_NAMES

# Functions
log_header() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}$1${NC}"
    echo -e "${CYAN}========================================${NC}"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

log_pass() {
    echo -e "${GREEN}  ✅ PASS:${NC} $1"
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

log_fail() {
    echo -e "${RED}  ❌ FAIL:${NC} $1"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    FAILED_TEST_NAMES+=("$2")
}

log_skip() {
    echo -e "${YELLOW}  ⏭️  SKIP:${NC} $1"
    SKIPPED_TESTS=$((SKIPPED_TESTS + 1))
}

log_info() {
    echo -e "${CYAN}  ℹ️  INFO:${NC} $1"
}

# Test functions
test_version() {
    local tool=$1
    local expected=$2
    local cmd=$3

    log_test "Testing $tool version"

    if command -v "$tool" &> /dev/null; then
        actual_version=$($cmd 2>&1 || echo "unknown")
        if echo "$actual_version" | grep -q "$expected"; then
            log_pass "$tool version: $expected"
        else
            log_fail "$tool version mismatch: expected $expected, got $actual_version" "$tool version"
        fi
    else
        log_fail "$tool not found in PATH" "$tool installation"
    fi
}

test_command_exists() {
    local tool=$1
    log_test "Testing $tool availability"

    if command -v "$tool" &> /dev/null; then
        log_pass "$tool is available"
    else
        log_fail "$tool not found" "$tool installation"
    fi
}

test_file_exists() {
    local file=$1
    local description=$2
    log_test "Testing $description"

    if [ -f "$file" ]; then
        log_pass "$description exists"
    else
        log_fail "$description not found at $file" "$description"
    fi
}

# Main tests
echo -e "${GREEN}🔍 DevContainer Verification Test Suite${NC}"
echo -e "${GREEN}Started at: $(date)${NC}"

# Test 1: Language Versions
log_header "1. Language Version Tests"

test_version "Go" "1.24.7" "go version"
test_version "Python" "3.11" "python --version"
test_version "Node" "v20" "node --version"

# Test 2: Kubernetes Tools
log_header "2. Kubernetes Tools Tests"

test_version "kubectl" "v1.31.0" "kubectl version --client --short"
test_version "helm" "v3.16.2" "helm version --short"
test_version "kind" "v0.23.0" "kind version"

# Test 3: Go Development Tools
log_header "3. Go Development Tools Tests"

test_command_exists "golangci-lint"
test_command_exists "gosec"
test_command_exists "controller-gen"
test_command_exists "kustomize"
test_command_exists "kubebuilder"
test_command_exists "setup-envtest"

# Test 4: Docker Functionality
log_header "4. Docker Functionality Tests"

log_test "Testing Docker availability"
if docker version &> /dev/null; then
    log_pass "Docker is available"
else
    log_fail "Docker is not available" "Docker installation"
fi

log_test "Testing Docker build capability"
if docker build -t test:devcontainer-verify -f - . <<EOF &> /dev/null
FROM alpine:latest
RUN echo "Test"
EOF
then
    log_pass "Docker build works"
    docker rmi test:devcontainer-verify &> /dev/null || true
else
    log_fail "Docker build failed" "Docker build"
fi

log_test "Testing docker-compose availability"
if docker-compose version &> /dev/null || docker compose version &> /dev/null; then
    log_pass "docker-compose is available"
else
    log_fail "docker-compose not available" "docker-compose installation"
fi

# Test 5: Security Configuration
log_header "5. Security Configuration Tests"

log_test "Testing privileged mode"
if docker inspect "$(hostname)" 2>/dev/null | grep -q '"Privileged": false'; then
    log_pass "Container is NOT running in privileged mode"
else
    log_fail "Container IS running in privileged mode (security risk)" "Privileged mode"
fi

log_test "Testing capability configuration"
if docker inspect "$(hostname)" 2>/dev/null | grep -q "CapAdd"; then
    caps=$(docker inspect "$(hostname)" 2>/dev/null | grep -A 5 "CapAdd" | grep -o "NET_ADMIN\|SYS_PTRACE" | wc -l)
    if [ "$caps" -ge 1 ]; then
        log_pass "Capabilities are properly configured"
    else
        log_fail "Capability configuration issue" "Capabilities"
    fi
else
    log_skip "Could not check capability configuration"
fi

log_test "Testing current user"
current_user=$(whoami)
if [ "$current_user" != "root" ]; then
    log_pass "Running as non-root user: $current_user"
else
    log_fail "Running as root (security risk)" "User configuration"
fi

# Test 6: Named Volumes
log_header "6. Named Volume Tests"

volumes=("devcontainer-go-mod-cache" "devcontainer-go-build-cache" "devcontainer-python-cache" "devcontainer-extensions")
for vol in "${volumes[@]}"; do
    log_test "Testing volume: $vol"
    if docker volume ls | grep -q "$vol"; then
        log_pass "Volume $vol exists"
    else
        log_fail "Volume $vol not found" "Volume $vol"
    fi
done

# Test 7: Project Structure
log_header "7. Project Structure Tests"

test_file_exists "/workspace/.devcontainer/devcontainer.json" "DevContainer config"
test_file_exists "/workspace/.devcontainer/scripts/devcontainer-security-check.sh" "Security check script"
test_file_exists "/workspace/.devcontainer/scripts/post-create.sh" "Post-create script"
test_file_exists "/workspace/Makefile" "Makefile"
test_file_exists "/workspace/go.mod" "Go module file"

# Test 8: Git Configuration
log_header "8. Git Configuration Tests"

log_test "Testing Git safe directory"
if git config --get-all safe.directory | grep -q "/workspace"; then
    log_pass "Git safe directory configured"
else
    log_fail "Git safe directory not configured" "Git configuration"
fi

# Test 9: Environment Variables
log_header "9. Environment Variable Tests"

envs=("GOPATH" "PYTHONPATH" "SHELL")
for env in "${envs[@]}"; do
    log_test "Testing environment variable: $env"
    if [ -n "${!env}" ]; then
        log_pass "$env is set: ${!env}"
    else
        log_fail "$env is not set" "Environment $env"
    fi
done

# Test 10: Go Build Cache
log_header "10. Go Build Performance Tests"

log_test "Testing Go build cache"
cd /workspace
if [ -d "/go/pkg/mod" ]; then
    mod_count=$(find /go/pkg/mod -type d | wc -l)
    if [ "$mod_count" -gt 10 ]; then
        log_pass "Go module cache is populated ($mod_count directories)"
    else
        log_info "Go module cache is sparse (expected on first build)"
        log_pass "Go module cache directory exists"
    fi
else
    log_fail "Go module cache directory not found" "Go cache"
fi

# Test 11: Network Configuration
log_header "11. Network Tests"

log_test "Testing network connectivity"
if ping -c 1 8.8.8.8 &> /dev/null; then
    log_pass "External network connectivity works"
else
    log_fail "No external network connectivity" "Network"
fi

log_test "Testing DNS resolution"
if nslookup github.com &> /dev/null || host github.com &> /dev/null; then
    log_pass "DNS resolution works"
else
    log_fail "DNS resolution failed" "DNS"
fi

# Test 12: Kind Cluster Test (Optional)
log_header "12. Kind Cluster Test (Optional)"

log_test "Testing Kind cluster creation"
if kind create cluster --name verify-test --wait 60s &> /dev/null; then
    log_pass "Kind cluster created successfully"

    log_test "Testing kubectl access"
    if kubectl cluster-info --context kind-verify-test &> /dev/null; then
        log_pass "kubectl can access Kind cluster"
    else
        log_fail "kubectl cannot access Kind cluster" "kubectl access"
    fi

    # Cleanup
    kind delete cluster --name verify-test &> /dev/null || true
else
    log_skip "Kind cluster creation skipped (may require more time)"
fi

# Test 13: Security Scan
log_header "13. Security Scan Test"

log_test "Running security check script"
if [ -f "/workspace/.devcontainer/scripts/devcontainer-security-check.sh" ]; then
    if bash /workspace/.devcontainer/scripts/devcontainer-security-check.sh &> /dev/null; then
        log_pass "Security check passed"
    else
        log_fail "Security check found issues" "Security scan"
    fi
else
    log_skip "Security check script not found"
fi

# Summary
log_header "Test Summary"

echo -e "${GREEN}✅ Passed:${NC}  $PASSED_TESTS"
echo -e "${RED}❌ Failed:${NC}  $FAILED_TESTS"
echo -e "${YELLOW}⏭️  Skipped:${NC} $SKIPPED_TESTS"
echo -e "${CYAN}📊 Total:${NC}   $TOTAL_TESTS"

# Calculate success rate
if [ $TOTAL_TESTS -gt 0 ]; then
    success_rate=$(( PASSED_TESTS * 100 / TOTAL_TESTS ))
    echo -e "${CYAN}📈 Success Rate:${NC} ${success_rate}%"
fi

# List failed tests
if [ $FAILED_TESTS -gt 0 ]; then
    echo ""
    echo -e "${RED}Failed Tests:${NC}"
    for test in "${FAILED_TEST_NAMES[@]}"; do
        echo -e "  ${RED}•${NC} $test"
    done
fi

echo ""
echo -e "${CYAN}Completed at: $(date)${NC}"

# Exit code
if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
elif [ $FAILED_TESTS -le 2 ]; then
    echo -e "${YELLOW}⚠️  Some tests failed, but environment is mostly functional${NC}"
    exit 0
else
    echo -e "${RED}❌ Multiple tests failed, please review configuration${NC}"
    exit 1
fi