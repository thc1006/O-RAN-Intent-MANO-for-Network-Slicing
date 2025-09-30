#!/bin/bash
# DevContainer Security Check Script
# Validates security configuration and scans for vulnerabilities

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔒 Running DevContainer Security Checks...${NC}"
echo ""

# Initialize counters
WARNINGS=0
ERRORS=0
CHECKS_PASSED=0

# Function to log results
log_pass() {
    echo -e "${GREEN}✅ PASS:${NC} $1"
    CHECKS_PASSED=$((CHECKS_PASSED + 1))
}

log_warn() {
    echo -e "${YELLOW}⚠️  WARN:${NC} $1"
    WARNINGS=$((WARNINGS + 1))
}

log_error() {
    echo -e "${RED}❌ ERROR:${NC} $1"
    ERRORS=$((ERRORS + 1))
}

log_info() {
    echo -e "${BLUE}ℹ️  INFO:${NC} $1"
}

# Check 1: Verify no privileged mode in devcontainer.json
echo "=== Checking DevContainer Configuration ==="
if grep -q '"--privileged"' /workspace/.devcontainer/devcontainer.json 2>/dev/null; then
    log_error "DevContainer is running in privileged mode!"
    log_info "This violates security best practices. Remove --privileged flag."
else
    log_pass "DevContainer is not running in privileged mode"
fi

# Check 2: Verify capability restrictions
if grep -q '"--cap-drop=ALL"' /workspace/.devcontainer/devcontainer.json 2>/dev/null; then
    log_pass "Capabilities are properly restricted (cap-drop=ALL)"
else
    log_warn "Capabilities are not restricted. Consider adding --cap-drop=ALL"
fi

# Check 3: Verify no-new-privileges security option
if grep -q '"--security-opt=no-new-privileges"' /workspace/.devcontainer/devcontainer.json 2>/dev/null; then
    log_pass "no-new-privileges security option is enabled"
else
    log_warn "no-new-privileges security option not found"
fi

# Check 4: Scan for secrets in environment
echo ""
echo "=== Scanning for Exposed Secrets ==="
if [ -f /workspace/.env ]; then
    log_warn ".env file detected in workspace"
    log_info "Ensure .env is in .gitignore and contains no real secrets"

    # Check if .env is in .gitignore
    if grep -q "^\.env$" /workspace/.gitignore 2>/dev/null; then
        log_pass ".env is properly listed in .gitignore"
    else
        log_error ".env is NOT in .gitignore - risk of committing secrets!"
    fi
else
    log_pass "No .env file found in workspace root"
fi

# Check 5: Run gitleaks if available
if command -v gitleaks &> /dev/null; then
    log_info "Running gitleaks secret scanner..."
    if gitleaks detect --no-git --verbose 2>&1 | grep -q "No leaks found"; then
        log_pass "No secrets detected by gitleaks"
    else
        log_warn "Potential secrets detected - review gitleaks output"
    fi
else
    log_info "gitleaks not installed - skipping secret scan"
fi

# Check 6: Verify Go dependencies for vulnerabilities
echo ""
echo "=== Checking Go Dependencies ==="
if command -v govulncheck &> /dev/null; then
    log_info "Running govulncheck on Go modules..."

    # Count vulnerable modules
    vuln_count=0
    while IFS= read -r modfile; do
        moddir=$(dirname "$modfile")
        if ! govulncheck "$moddir/..." &>/dev/null; then
            vuln_count=$((vuln_count + 1))
        fi
    done < <(find /workspace -name "go.mod" -type f 2>/dev/null)

    if [ $vuln_count -eq 0 ]; then
        log_pass "No Go vulnerabilities detected"
    else
        log_warn "Found vulnerabilities in $vuln_count Go module(s)"
    fi
else
    log_info "govulncheck not installed - skipping Go vulnerability scan"
fi

# Check 7: Verify container user is not root
echo ""
echo "=== Checking Container User ==="
current_user=$(whoami)
if [ "$current_user" = "root" ]; then
    log_error "Running as root user - security risk!"
else
    log_pass "Running as non-root user: $current_user"
fi

# Check 8: Verify required security tools are installed
echo ""
echo "=== Verifying Security Tools ==="
tools=("gitleaks" "gosec" "trivy" "kubebuilder" "setup-envtest")
for tool in "${tools[@]}"; do
    if command -v "$tool" &> /dev/null; then
        log_pass "$tool is installed"
    else
        log_warn "$tool is not installed"
    fi
done

# Check 9: Verify PATH doesn't include suspicious directories
echo ""
echo "=== Checking PATH Security ==="
if echo "$PATH" | grep -q "/tmp"; then
    log_warn "PATH includes /tmp directory - potential security risk"
else
    log_pass "PATH does not include insecure directories"
fi

# Check 10: Check for hardcoded IPs or credentials in code
echo ""
echo "=== Scanning for Hardcoded Values ==="
hardcoded_patterns=0

# Check for hardcoded IPs (excluding localhost and common examples)
if grep -r -n -E '[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}' \
    --include="*.go" --include="*.py" --include="*.yaml" \
    /workspace 2>/dev/null | \
    grep -v "127.0.0.1" | \
    grep -v "0.0.0.0" | \
    grep -v "example.com" | \
    grep -v "\.git/" | \
    head -1 > /dev/null; then
    log_warn "Potential hardcoded IP addresses found in code"
    hardcoded_patterns=$((hardcoded_patterns + 1))
fi

# Check for potential hardcoded passwords/tokens
if grep -r -i -n -E "(password|token|secret|api[_-]?key).*=.*['\"][^'\"]+['\"]" \
    --include="*.go" --include="*.py" \
    /workspace 2>/dev/null | \
    grep -v "\.git/" | \
    grep -v "_test\." | \
    head -1 > /dev/null; then
    log_warn "Potential hardcoded credentials found in code"
    hardcoded_patterns=$((hardcoded_patterns + 1))
fi

if [ $hardcoded_patterns -eq 0 ]; then
    log_pass "No obvious hardcoded values detected"
fi

# Check 11: Verify network isolation
echo ""
echo "=== Checking Network Configuration ==="
if docker network ls 2>/dev/null | grep -q "devcontainer"; then
    log_pass "DevContainer network exists"
else
    log_info "No dedicated devcontainer network found"
fi

# Check 12: Verify Go version matches CI
echo ""
echo "=== Checking Version Consistency ==="
go_version=$(go version 2>/dev/null | grep -oP 'go\K[0-9.]+' || echo "unknown")
expected_version="1.24.7"

if [ "$go_version" = "$expected_version" ]; then
    log_pass "Go version matches expected: $expected_version"
elif [ "$go_version" != "unknown" ]; then
    log_warn "Go version mismatch: found $go_version, expected $expected_version"
else
    log_info "Could not determine Go version"
fi

# Summary
echo ""
echo "========================================"
echo "  Security Check Summary"
echo "========================================"
echo -e "${GREEN}✅ Checks Passed: $CHECKS_PASSED${NC}"
echo -e "${YELLOW}⚠️  Warnings: $WARNINGS${NC}"
echo -e "${RED}❌ Errors: $ERRORS${NC}"
echo ""

if [ $ERRORS -gt 0 ]; then
    echo -e "${RED}CRITICAL: $ERRORS security error(s) detected!${NC}"
    echo "Please address errors before proceeding."
    exit 1
elif [ $WARNINGS -gt 5 ]; then
    echo -e "${YELLOW}NOTICE: Multiple warnings detected.${NC}"
    echo "Consider addressing warnings to improve security posture."
    exit 0
else
    echo -e "${GREEN}✅ Security checks completed successfully!${NC}"
    exit 0
fi