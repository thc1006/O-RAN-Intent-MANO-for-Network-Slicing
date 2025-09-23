#!/bin/bash
# Pre-commit hook for Dockerfile security validation
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Checking Dockerfile security...${NC}"

EXIT_CODE=0

# Find all Dockerfiles
DOCKERFILES=$(find . -name "Dockerfile" -not -path "./vendor/*" -not -path "./.git/*")

if [ -z "$DOCKERFILES" ]; then
    echo -e "${YELLOW}No Dockerfiles found to check.${NC}"
    exit 0
fi

for dockerfile in $DOCKERFILES; do
    echo -e "${GREEN}Checking: $dockerfile${NC}"

    # Check for security best practices
    ISSUES_FOUND=false

    # Check 1: Ensure non-root user
    if ! grep -q "USER.*[^0]" "$dockerfile"; then
        echo -e "${RED}❌ No non-root USER directive found in $dockerfile${NC}"
        ISSUES_FOUND=true
    fi

    # Check 2: No hardcoded secrets
    if grep -iE "(password|secret|key|token|api)" "$dockerfile" | grep -v "ENV.*FILE" | grep -v "ARG.*FILE"; then
        echo -e "${RED}❌ Potential hardcoded secrets found in $dockerfile${NC}"
        ISSUES_FOUND=true
    fi

    # Check 3: Use specific image tags (not latest)
    if grep -E "FROM.*:latest" "$dockerfile"; then
        echo -e "${YELLOW}⚠️  Using 'latest' tag in $dockerfile - consider pinning specific versions${NC}"
    fi

    # Check 4: No sudo usage
    if grep -q "sudo" "$dockerfile"; then
        echo -e "${RED}❌ Usage of 'sudo' found in $dockerfile - avoid sudo in containers${NC}"
        ISSUES_FOUND=true
    fi

    # Check 5: Use COPY instead of ADD where appropriate
    if grep -E "ADD.*\.(tar|zip|gz|bz2)" "$dockerfile" >/dev/null 2>&1; then
        # ADD is appropriate for archives
        :
    elif grep -q "ADD" "$dockerfile"; then
        echo -e "${YELLOW}⚠️  Consider using COPY instead of ADD in $dockerfile unless extracting archives${NC}"
    fi

    # Check 6: Ensure proper package cleanup
    if grep -q "apt-get install" "$dockerfile" && ! grep -q "apt-get clean\|rm.*apt.*lists" "$dockerfile"; then
        echo -e "${YELLOW}⚠️  Consider cleaning apt cache after installation in $dockerfile${NC}"
    fi

    # Check 7: Check for HEALTHCHECK
    if ! grep -q "HEALTHCHECK" "$dockerfile"; then
        echo -e "${YELLOW}⚠️  No HEALTHCHECK instruction found in $dockerfile${NC}"
    fi

    if [ "$ISSUES_FOUND" = true ]; then
        EXIT_CODE=1
    else
        echo -e "${GREEN}✅ $dockerfile passed security checks${NC}"
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All Dockerfiles passed security checks!${NC}"
else
    echo -e "${RED}Some Dockerfiles have security issues. Please fix them before committing.${NC}"
fi

exit $EXIT_CODE