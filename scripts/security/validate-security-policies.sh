#!/bin/bash
# Pre-commit hook for Security Policy validation
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Validating Security Policies...${NC}"

EXIT_CODE=0

# Find all security policy files
SECURITY_FILES=$(find . -path "./security/*" -name "*.yaml" -o -path "./security/*" -name "*.yml" -o -path "./deploy/k8s/*" -name "*security*.yaml" -o -path "./deploy/k8s/*" -name "*security*.yml" -o -path "./.github/workflows/*" -name "*.yml" | grep -v ".git")

if [ -z "$SECURITY_FILES" ]; then
    echo -e "${YELLOW}No security policy files found to validate.${NC}"
    exit 0
fi

for security_file in $SECURITY_FILES; do
    echo -e "${GREEN}Validating: $security_file${NC}"

    ISSUES_FOUND=false

    # Check 1: Validate YAML syntax
    if ! python3 -c "import yaml; yaml.safe_load(open('$security_file'))" 2>/dev/null; then
        echo -e "${RED}❌ Invalid YAML syntax in $security_file${NC}"
        ISSUES_FOUND=true
        continue
    fi

    # Check 2: Security context validation for Kubernetes manifests
    if grep -q "kind.*Pod\|kind.*Deployment\|kind.*StatefulSet\|kind.*DaemonSet" "$security_file"; then
        if ! grep -q "securityContext:" "$security_file"; then
            echo -e "${YELLOW}⚠️  No securityContext found in $security_file${NC}"
        fi

        # Check for runAsNonRoot
        if ! grep -q "runAsNonRoot.*true" "$security_file"; then
            echo -e "${YELLOW}⚠️  runAsNonRoot not set to true in $security_file${NC}"
        fi

        # Check for readOnlyRootFilesystem
        if ! grep -q "readOnlyRootFilesystem.*true" "$security_file"; then
            echo -e "${YELLOW}⚠️  readOnlyRootFilesystem not set to true in $security_file${NC}"
        fi

        # Check for privileged containers
        if grep -q "privileged.*true" "$security_file"; then
            echo -e "${RED}❌ Privileged container found in $security_file${NC}"
            ISSUES_FOUND=true
        fi

        # Check for allowPrivilegeEscalation
        if grep -q "allowPrivilegeEscalation.*true" "$security_file"; then
            echo -e "${RED}❌ allowPrivilegeEscalation set to true in $security_file${NC}"
            ISSUES_FOUND=true
        fi
    fi

    # Check 3: GitHub Actions workflow security
    if [[ "$security_file" == *".github/workflows"* ]]; then
        # Check for proper permissions
        if ! grep -q "permissions:" "$security_file"; then
            echo -e "${YELLOW}⚠️  No permissions specified in workflow $security_file${NC}"
        fi

        # Check for hardcoded secrets
        if grep -iE "(password|secret|key|token|api).*:" "$security_file" | grep -v "secrets\." | grep -v "github.token"; then
            echo -e "${RED}❌ Potential hardcoded secrets in workflow $security_file${NC}"
            ISSUES_FOUND=true
        fi

        # Check for third-party actions pinning
        if grep -E "uses:.*@(main|master|latest)" "$security_file"; then
            echo -e "${YELLOW}⚠️  Third-party actions not pinned to specific versions in $security_file${NC}"
        fi
    fi

    # Check 4: Pod Security Standards
    if grep -q "apiVersion.*policy/v1beta1\|apiVersion.*policy/v1" "$security_file" && grep -q "kind.*PodSecurityPolicy" "$security_file"; then
        # Check for privileged PSP
        if grep -q "privileged.*true" "$security_file"; then
            echo -e "${RED}❌ Privileged PodSecurityPolicy found in $security_file${NC}"
            ISSUES_FOUND=true
        fi

        # Check for host network/PID/IPC
        if grep -qE "(hostNetwork|hostPID|hostIPC).*true" "$security_file"; then
            echo -e "${YELLOW}⚠️  Host namespace access enabled in $security_file${NC}"
        fi
    fi

    # Check 5: Network Policies
    if grep -q "kind.*NetworkPolicy" "$security_file"; then
        # Check for empty podSelector (applies to all pods)
        if grep -q "podSelector: {}" "$security_file"; then
            echo -e "${YELLOW}⚠️  NetworkPolicy applies to all pods in $security_file - ensure this is intentional${NC}"
        fi
    fi

    # Check 6: Service mesh security policies
    if grep -qE "kind.*(AuthorizationPolicy|PeerAuthentication|RequestAuthentication)" "$security_file"; then
        # Check for permissive policies
        if grep -q "action.*ALLOW" "$security_file" && ! grep -q "rules:" "$security_file"; then
            echo -e "${YELLOW}⚠️  Permissive authorization policy without rules in $security_file${NC}"
        fi
    fi

    if [ "$ISSUES_FOUND" = true ]; then
        EXIT_CODE=1
    else
        echo -e "${GREEN}✅ $security_file passed validation${NC}"
    fi
done

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}All security policies are valid!${NC}"
else
    echo -e "${RED}Some security policies have issues. Please fix them before committing.${NC}"
fi

exit $EXIT_CODE