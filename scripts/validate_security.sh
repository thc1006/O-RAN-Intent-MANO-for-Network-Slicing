#!/bin/bash

# Security Validation Script for O-RAN Intent-MANO Network Slicing
# Copyright 2024 O-RAN Intent MANO Project
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORT_FILE="${PROJECT_ROOT}/security-validation-report.json"
LOG_FILE="${PROJECT_ROOT}/security-validation.log"
EXIT_CODE=0

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m' # No Color

# Logging functions
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*" | tee -a "$LOG_FILE"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*" | tee -a "$LOG_FILE"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*" | tee -a "$LOG_FILE"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*" | tee -a "$LOG_FILE"
    EXIT_CODE=1
}

# Initialize report
init_report() {
    cat > "$REPORT_FILE" << EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "validation_version": "1.0.0",
  "project": "O-RAN Intent-MANO for Network Slicing",
  "commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "security_validation": {
    "go_error_handling": {
      "status": "pending",
      "issues": [],
      "files_checked": 0
    },
    "kubernetes_security": {
      "status": "pending",
      "network_policies": {
        "found": 0,
        "validated": 0,
        "issues": []
      },
      "security_policies": {
        "found": 0,
        "validated": 0,
        "issues": []
      },
      "rbac": {
        "found": 0,
        "validated": 0,
        "issues": []
      }
    },
    "secure_logging": {
      "status": "pending",
      "implementations_found": 0,
      "issues": []
    },
    "input_validation": {
      "status": "pending",
      "validators_found": 0,
      "issues": []
    },
    "container_security": {
      "status": "pending",
      "dockerfiles_checked": 0,
      "issues": []
    },
    "secrets_management": {
      "status": "pending",
      "issues": []
    }
  },
  "overall_status": "pending",
  "recommendations": []
}
EOF
}

# Update report section
update_report() {
    local section="$1"
    local status="$2"
    local data="$3"

    # Use jq to update the report
    if command -v jq >/dev/null 2>&1; then
        tmp_file=$(mktemp)
        jq "$section.status = \"$status\" | $section += $data" "$REPORT_FILE" > "$tmp_file" && mv "$tmp_file" "$REPORT_FILE"
    else
        log_warning "jq not available, report updates will be limited"
    fi
}

# Check Go error handling patterns
validate_go_error_handling() {
    log_info "Validating Go error handling patterns..."

    local issues=()
    local files_checked=0

    # Find all Go files
    while IFS= read -r -d '' go_file; do
        ((files_checked++))

        # Check for proper error handling patterns
        local line_num=0
        while IFS= read -r line; do
            ((line_num++))

            # Check for unhandled errors (basic pattern)
            if echo "$line" | grep -qE '^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*,\s*err\s*:=.*$' && \
               ! tail -n +$((line_num + 1)) "$go_file" | head -n 5 | grep -qE '(if\s+err\s*!=\s*nil|return.*err)'; then
                issues+=("{\"file\": \"$go_file\", \"line\": $line_num, \"issue\": \"Potential unhandled error\"}")
            fi

            # Check for log injection vulnerabilities
            if echo "$line" | grep -qE 'log\.(Print|Fatal|Panic).*\+.*user.*input' || \
               echo "$line" | grep -qE 'fmt\.(Print|Sprint).*\+.*\$\{'; then
                issues+=("{\"file\": \"$go_file\", \"line\": $line_num, \"issue\": \"Potential log injection vulnerability\"}")
            fi

            # Check for SQL injection patterns
            if echo "$line" | grep -qE 'db\.(Query|Exec).*\+.*\$\{' || \
               echo "$line" | grep -qE 'fmt\.Sprintf.*SELECT.*\+'; then
                issues+=("{\"file\": \"$go_file\", \"line\": $line_num, \"issue\": \"Potential SQL injection vulnerability\"}")
            fi

            # Check for command injection patterns
            if echo "$line" | grep -qE 'exec\.Command.*\+.*user' || \
               echo "$line" | grep -qE 'os\.system.*\+'; then
                issues+=("{\"file\": \"$go_file\", \"line\": $line_num, \"issue\": \"Potential command injection vulnerability\"}")
            fi

        done < "$go_file"

    done < <(find "$PROJECT_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" -print0)

    # Check for proper use of security package
    local security_usage_count=0
    security_usage_count=$(grep -r "pkg/security" "$PROJECT_ROOT" --include="*.go" | wc -l)

    if [ "$security_usage_count" -lt 5 ]; then
        issues+=("{\"file\": \"global\", \"line\": 0, \"issue\": \"Limited usage of security package ($security_usage_count occurrences)\"}")
    fi

    local status="passed"
    if [ ${#issues[@]} -gt 0 ]; then
        status="failed"
        log_error "Found ${#issues[@]} Go security issues"
        for issue in "${issues[@]}"; do
            log_error "  $issue"
        done
    else
        log_success "Go error handling validation passed ($files_checked files checked)"
    fi

    # Update report
    local issues_json=""
    if [ ${#issues[@]} -gt 0 ]; then
        issues_json="[$(IFS=','; echo "${issues[*]}")]"
    else
        issues_json="[]"
    fi

    update_report ".security_validation.go_error_handling" "$status" "{\"files_checked\": $files_checked, \"issues\": $issues_json}"
}

# Validate Kubernetes security configurations
validate_kubernetes_security() {
    log_info "Validating Kubernetes security configurations..."

    local k8s_dir="$PROJECT_ROOT/deploy/k8s"
    local issues=()

    # Check NetworkPolicies
    local netpol_count=0
    local netpol_validated=0
    local netpol_issues=()

    if [ -d "$k8s_dir" ]; then
        while IFS= read -r -d '' policy_file; do
            ((netpol_count++))
            log_info "Checking NetworkPolicy: $policy_file"

            # Validate NetworkPolicy structure
            if grep -q "kind: NetworkPolicy" "$policy_file"; then
                # Check for proper ingress/egress rules
                if grep -q "policyTypes:" "$policy_file" &&
                   (grep -q "- Ingress" "$policy_file" || grep -q "- Egress" "$policy_file"); then
                    ((netpol_validated++))
                    log_success "NetworkPolicy validated: $(basename "$policy_file")"
                else
                    netpol_issues+=("{\"file\": \"$policy_file\", \"issue\": \"Missing or incomplete policyTypes\"}")
                fi

                # Check for overly permissive rules
                if grep -q "podSelector: {}" "$policy_file" && grep -q "from: \[\]" "$policy_file"; then
                    netpol_issues+=("{\"file\": \"$policy_file\", \"issue\": \"Overly permissive rule - allows all traffic\"}")
                fi
            fi
        done < <(find "$k8s_dir" -name "*network*policy*.yaml" -o -name "*netpol*.yaml" -print0)
    fi

    # Check Security Policies
    local secpol_count=0
    local secpol_validated=0
    local secpol_issues=()

    if [ -d "$k8s_dir" ]; then
        while IFS= read -r -d '' policy_file; do
            ((secpol_count++))
            log_info "Checking Security Policy: $policy_file"

            # Validate PodSecurityPolicy or PodSecurity standards
            if grep -q "kind: PodSecurityPolicy\|PodSecurityConfiguration" "$policy_file"; then
                # Check for security controls
                if grep -q "allowPrivilegeEscalation.*false\|enforce.*restricted" "$policy_file" &&
                   grep -q "runAsNonRoot.*true\|runAsUser" "$policy_file"; then
                    ((secpol_validated++))
                    log_success "Security Policy validated: $(basename "$policy_file")"
                else
                    secpol_issues+=("{\"file\": \"$policy_file\", \"issue\": \"Missing security controls\"}")
                fi
            fi
        done < <(find "$k8s_dir" -name "*security*.yaml" -o -name "*pod-security*.yaml" -print0)
    fi

    # Check RBAC
    local rbac_count=0
    local rbac_validated=0
    local rbac_issues=()

    if [ -d "$k8s_dir" ]; then
        while IFS= read -r -d '' rbac_file; do
            ((rbac_count++))
            log_info "Checking RBAC: $rbac_file"

            # Validate RBAC structure
            if grep -q "kind: Role\|ClusterRole\|RoleBinding\|ClusterRoleBinding" "$rbac_file"; then
                # Check for overly broad permissions
                if grep -q "resources: \[\"*\"\]\|verbs: \[\"*\"\]" "$rbac_file"; then
                    rbac_issues+=("{\"file\": \"$rbac_file\", \"issue\": \"Overly broad permissions detected\"}")
                else
                    ((rbac_validated++))
                    log_success "RBAC validated: $(basename "$rbac_file")"
                fi

                # Check for proper least privilege
                if grep -q "apiGroups: \[\"*\"\]" "$rbac_file"; then
                    rbac_issues+=("{\"file\": \"$rbac_file\", \"issue\": \"Excessive API group permissions\"}")
                fi
            fi
        done < <(find "$k8s_dir" -name "*rbac*.yaml" -o -name "*role*.yaml" -print0)
    fi

    # Determine overall status
    local k8s_status="passed"
    local total_issues=$((${#netpol_issues[@]} + ${#secpol_issues[@]} + ${#rbac_issues[@]}))

    if [ $total_issues -gt 0 ]; then
        k8s_status="failed"
        log_error "Found $total_issues Kubernetes security issues"
    else
        log_success "Kubernetes security validation passed"
    fi

    # Update report
    local netpol_issues_json="[]"
    local secpol_issues_json="[]"
    local rbac_issues_json="[]"

    if [ ${#netpol_issues[@]} -gt 0 ]; then
        netpol_issues_json="[$(IFS=','; echo "${netpol_issues[*]}")]"
    fi
    if [ ${#secpol_issues[@]} -gt 0 ]; then
        secpol_issues_json="[$(IFS=','; echo "${secpol_issues[*]}")]"
    fi
    if [ ${#rbac_issues[@]} -gt 0 ]; then
        rbac_issues_json="[$(IFS=','; echo "${rbac_issues[*]}")]"
    fi

    update_report ".security_validation.kubernetes_security" "$k8s_status" "{
        \"network_policies\": {\"found\": $netpol_count, \"validated\": $netpol_validated, \"issues\": $netpol_issues_json},
        \"security_policies\": {\"found\": $secpol_count, \"validated\": $secpol_validated, \"issues\": $secpol_issues_json},
        \"rbac\": {\"found\": $rbac_count, \"validated\": $rbac_validated, \"issues\": $rbac_issues_json}
    }"

    log_info "NetworkPolicies: $netpol_validated/$netpol_count validated"
    log_info "Security Policies: $secpol_validated/$secpol_count validated"
    log_info "RBAC: $rbac_validated/$rbac_count validated"
}

# Validate secure logging implementation
validate_secure_logging() {
    log_info "Validating secure logging implementation..."

    local implementations=0
    local issues=()

    # Check for secure logging package usage
    if [ -f "$PROJECT_ROOT/pkg/security/logging.go" ]; then
        ((implementations++))
        log_success "Found secure logging package: pkg/security/logging.go"

        # Check for key security functions
        local required_functions=("SanitizeForLog" "SafeLogf" "SanitizeErrorForLog" "validateLogMessage")
        for func in "${required_functions[@]}"; do
            if grep -q "func.*$func" "$PROJECT_ROOT/pkg/security/logging.go"; then
                log_success "Found secure logging function: $func"
            else
                issues+=("{\"file\": \"pkg/security/logging.go\", \"issue\": \"Missing function: $func\"}")
            fi
        done

        # Check for log injection protection
        if grep -q "containsLogInjectionPatterns\|log injection" "$PROJECT_ROOT/pkg/security/logging.go"; then
            log_success "Log injection protection implemented"
        else
            issues+=("{\"file\": \"pkg/security/logging.go\", \"issue\": \"No log injection protection found\"}")
        fi
    else
        issues+=("{\"file\": \"pkg/security/logging.go\", \"issue\": \"Secure logging package not found\"}")
    fi

    # Check for usage of secure logging in applications
    local usage_count=0
    usage_count=$(grep -r "pkg/security.*logging\|SanitizeForLog\|SafeLogf" "$PROJECT_ROOT" --include="*.go" --exclude-dir=vendor | wc -l)

    if [ "$usage_count" -lt 3 ]; then
        issues+=("{\"file\": \"global\", \"issue\": \"Limited usage of secure logging functions ($usage_count occurrences)\"}")
    else
        log_success "Secure logging functions used in $usage_count locations"
    fi

    local status="passed"
    if [ ${#issues[@]} -gt 0 ]; then
        status="failed"
        log_error "Found ${#issues[@]} secure logging issues"
    else
        log_success "Secure logging validation passed"
    fi

    local issues_json="[]"
    if [ ${#issues[@]} -gt 0 ]; then
        issues_json="[$(IFS=','; echo "${issues[*]}")]"
    fi

    update_report ".security_validation.secure_logging" "$status" "{\"implementations_found\": $implementations, \"issues\": $issues_json}"
}

# Validate input validation implementation
validate_input_validation() {
    log_info "Validating input validation implementation..."

    local validators=0
    local issues=()

    # Check for input validation package
    if [ -f "$PROJECT_ROOT/pkg/security/validation.go" ]; then
        ((validators++))
        log_success "Found input validation package: pkg/security/validation.go"

        # Check for key validation functions
        local validation_functions=("ValidateNetworkInterface" "ValidateIPAddress" "ValidateFilePath" "ValidateCommand")
        for func in "${validation_functions[@]}"; do
            if grep -q "func.*$func" "$PROJECT_ROOT/pkg/security/validation.go"; then
                log_success "Found validation function: $func"
            else
                issues+=("{\"file\": \"pkg/security/validation.go\", \"issue\": \"Missing validation function: $func\"}")
            fi
        done
    else
        issues+=("{\"file\": \"pkg/security/validation.go\", \"issue\": \"Input validation package not found\"}")
    fi

    # Check for subprocess security
    if [ -f "$PROJECT_ROOT/pkg/security/subprocess.go" ]; then
        ((validators++))
        log_success "Found subprocess security package: pkg/security/subprocess.go"

        if grep -q "SafeCommand\|CommandValidator" "$PROJECT_ROOT/pkg/security/subprocess.go"; then
            log_success "Safe subprocess execution implemented"
        else
            issues+=("{\"file\": \"pkg/security/subprocess.go\", \"issue\": \"Safe subprocess execution not found\"}")
        fi
    else
        issues+=("{\"file\": \"pkg/security/subprocess.go\", \"issue\": \"Subprocess security package not found\"}")
    fi

    local status="passed"
    if [ ${#issues[@]} -gt 0 ]; then
        status="failed"
        log_error "Found ${#issues[@]} input validation issues"
    else
        log_success "Input validation passed ($validators validators found)"
    fi

    local issues_json="[]"
    if [ ${#issues[@]} -gt 0 ]; then
        issues_json="[$(IFS=','; echo "${issues[*]}")]"
    fi

    update_report ".security_validation.input_validation" "$status" "{\"validators_found\": $validators, \"issues\": $issues_json}"
}

# Validate container security
validate_container_security() {
    log_info "Validating container security configurations..."

    local dockerfiles_checked=0
    local issues=()

    # Find and check Dockerfiles
    while IFS= read -r -d '' dockerfile; do
        ((dockerfiles_checked++))
        log_info "Checking Dockerfile: $dockerfile"

        # Check for security best practices
        local dockerfile_issues=()

        # Check for non-root user
        if ! grep -q "USER.*[0-9]\+\|USER.*[a-z]" "$dockerfile"; then
            dockerfile_issues+=("No non-root user specified")
        fi

        # Check for COPY/ADD with proper permissions
        if grep -q "COPY.*--chown" "$dockerfile" || grep -q "RUN chown" "$dockerfile"; then
            log_success "Proper file ownership in $(basename "$dockerfile")"
        else
            dockerfile_issues+=("No explicit file ownership management")
        fi

        # Check for minimal base images
        if grep -q "FROM.*alpine\|FROM.*distroless\|FROM.*scratch" "$dockerfile"; then
            log_success "Using minimal base image in $(basename "$dockerfile")"
        else
            dockerfile_issues+=("Not using minimal base image")
        fi

        # Check for secrets in build
        if grep -qE "ARG.*SECRET\|ENV.*PASSWORD\|ENV.*TOKEN" "$dockerfile"; then
            dockerfile_issues+=("Potential secrets in build arguments/environment")
        fi

        # Check for package updates
        if grep -q "apt.*update.*upgrade\|apk.*update.*upgrade\|yum.*update" "$dockerfile"; then
            log_success "Package updates found in $(basename "$dockerfile")"
        else
            dockerfile_issues+=("No package updates found")
        fi

        # Add issues to global list
        for issue in "${dockerfile_issues[@]}"; do
            issues+=("{\"file\": \"$dockerfile\", \"issue\": \"$issue\"}")
        done

        if [ ${#dockerfile_issues[@]} -eq 0 ]; then
            log_success "Dockerfile security validation passed: $(basename "$dockerfile")"
        else
            log_warning "Found ${#dockerfile_issues[@]} issues in $(basename "$dockerfile")"
        fi

    done < <(find "$PROJECT_ROOT" -name "Dockerfile*" -not -path "*/.git/*" -print0)

    local status="passed"
    if [ ${#issues[@]} -gt 0 ]; then
        status="warning"  # Container issues are warnings, not failures
        log_warning "Found ${#issues[@]} container security recommendations"
    else
        log_success "Container security validation passed ($dockerfiles_checked Dockerfiles checked)"
    fi

    local issues_json="[]"
    if [ ${#issues[@]} -gt 0 ]; then
        issues_json="[$(IFS=','; echo "${issues[*]}")]"
    fi

    update_report ".security_validation.container_security" "$status" "{\"dockerfiles_checked\": $dockerfiles_checked, \"issues\": $issues_json}"
}

# Validate secrets management
validate_secrets_management() {
    log_info "Validating secrets management..."

    local issues=()

    # Check for hardcoded secrets in source code
    log_info "Scanning for hardcoded secrets..."

    # Common secret patterns
    local secret_patterns=(
        "password.*=.*['\"][^'\"]{8,}"
        "secret.*=.*['\"][^'\"]{8,}"
        "key.*=.*['\"][^'\"]{20,}"
        "token.*=.*['\"][^'\"]{16,}"
        "api[_-]?key.*=.*['\"][^'\"]{16,}"
        "access[_-]?key.*=.*['\"][^'\"]{16,}"
        "private[_-]?key.*=.*['\"][^'\"]{20,}"
    )

    for pattern in "${secret_patterns[@]}"; do
        while IFS= read -r match; do
            if [ -n "$match" ]; then
                # Exclude test files and mock data
                if [[ ! "$match" == *"test"* ]] && [[ ! "$match" == *"mock"* ]] && [[ ! "$match" == *"example"* ]]; then
                    issues+=("{\"file\": \"$(echo "$match" | cut -d: -f1)\", \"line\": \"$(echo "$match" | cut -d: -f2)\", \"issue\": \"Potential hardcoded secret\"}")
                fi
            fi
        done < <(grep -rn -E "$pattern" "$PROJECT_ROOT" --include="*.go" --include="*.py" --include="*.yaml" --include="*.yml" --exclude-dir=.git --exclude-dir=vendor 2>/dev/null || true)
    done

    # Check for proper secret management in Kubernetes
    local k8s_secrets_found=false
    if [ -d "$PROJECT_ROOT/deploy/k8s" ]; then
        if find "$PROJECT_ROOT/deploy/k8s" -name "*.yaml" -exec grep -l "kind: Secret" {} \; | head -1 >/dev/null 2>&1; then
            k8s_secrets_found=true
            log_success "Kubernetes Secret resources found"
        fi

        # Check for external secret management
        if find "$PROJECT_ROOT/deploy/k8s" -name "*.yaml" -exec grep -l "external-secrets\|sealed-secrets\|vault" {} \; | head -1 >/dev/null 2>&1; then
            log_success "External secret management detected"
        else
            issues+=("{\"file\": \"deploy/k8s\", \"issue\": \"No external secret management detected\"}")
        fi
    fi

    # Check for .env files with secrets
    while IFS= read -r -d '' env_file; do
        if grep -qE "SECRET|PASSWORD|TOKEN|KEY" "$env_file"; then
            # Check if it's a .env.sample or .env.example
            if [[ "$env_file" == *.sample ]] || [[ "$env_file" == *.example ]]; then
                log_success "Found template env file: $(basename "$env_file")"
            else
                issues+=("{\"file\": \"$env_file\", \"issue\": \"Env file may contain secrets\"}")
            fi
        fi
    done < <(find "$PROJECT_ROOT" -name ".env*" -not -path "*/.git/*" -print0)

    local status="passed"
    if [ ${#issues[@]} -gt 0 ]; then
        status="failed"
        log_error "Found ${#issues[@]} secrets management issues"
    else
        log_success "Secrets management validation passed"
    fi

    local issues_json="[]"
    if [ ${#issues[@]} -gt 0 ]; then
        issues_json="[$(IFS=','; echo "${issues[*]}")]"
    fi

    update_report ".security_validation.secrets_management" "$status" "{\"issues\": $issues_json}"
}

# Generate recommendations
generate_recommendations() {
    log_info "Generating security recommendations..."

    local recommendations=()

    # Check current report status
    if command -v jq >/dev/null 2>&1; then
        # Go error handling recommendations
        local go_status
        go_status=$(jq -r '.security_validation.go_error_handling.status' "$REPORT_FILE")
        if [ "$go_status" = "failed" ]; then
            recommendations+=("\"Implement proper error handling patterns in Go code using the security package\"")
            recommendations+=("\"Use secure logging functions from pkg/security/logging for all error messages\"")
        fi

        # Kubernetes security recommendations
        local k8s_netpol_count
        k8s_netpol_count=$(jq -r '.security_validation.kubernetes_security.network_policies.validated' "$REPORT_FILE")
        if [ "$k8s_netpol_count" -lt 3 ]; then
            recommendations+=("\"Implement comprehensive NetworkPolicies for all components\"")
            recommendations+=("\"Review and tighten existing NetworkPolicy rules\"")
        fi

        # Container security recommendations
        local container_status
        container_status=$(jq -r '.security_validation.container_security.status' "$REPORT_FILE")
        if [ "$container_status" = "warning" ]; then
            recommendations+=("\"Use minimal base images (alpine, distroless) for all containers\"")
            recommendations+=("\"Implement non-root users in all Dockerfiles\"")
            recommendations+=("\"Add security scanning to CI/CD pipeline\"")
        fi

        # Secrets management recommendations
        local secrets_status
        secrets_status=$(jq -r '.security_validation.secrets_management.status' "$REPORT_FILE")
        if [ "$secrets_status" = "failed" ]; then
            recommendations+=("\"Implement external secret management (Vault, External Secrets Operator)\"")
            recommendations+=("\"Remove any hardcoded secrets from source code\"")
            recommendations+=("\"Use Kubernetes Secrets with proper RBAC controls\"")
        fi
    fi

    # Add general recommendations
    recommendations+=("\"Implement regular security scanning in CI/CD pipeline\"")
    recommendations+=("\"Conduct periodic security audits of the codebase\"")
    recommendations+=("\"Keep all dependencies updated and scan for vulnerabilities\"")
    recommendations+=("\"Implement runtime security monitoring\"")

    # Update report with recommendations
    local recommendations_json=""
    if [ ${#recommendations[@]} -gt 0 ]; then
        recommendations_json="[$(IFS=','; echo "${recommendations[*]}")]"
    else
        recommendations_json="[]"
    fi

    update_report ".recommendations" "" "$recommendations_json"
}

# Finalize report
finalize_report() {
    log_info "Finalizing security validation report..."

    # Determine overall status
    local overall_status="passed"

    if command -v jq >/dev/null 2>&1; then
        local failed_checks
        failed_checks=$(jq -r '[.security_validation[] | select(.status == "failed")] | length' "$REPORT_FILE")
        if [ "$failed_checks" -gt 0 ]; then
            overall_status="failed"
        fi
    fi

    # Update overall status
    update_report ".overall_status" "$overall_status" "{}"

    log_info "Security validation report saved to: $REPORT_FILE"
    log_info "Security validation log saved to: $LOG_FILE"

    if [ "$overall_status" = "passed" ]; then
        log_success "Overall security validation: PASSED"
    else
        log_error "Overall security validation: FAILED"
    fi
}

# Main execution
main() {
    log_info "Starting O-RAN Intent-MANO Security Validation"
    log_info "Project root: $PROJECT_ROOT"

    # Clean up previous runs
    rm -f "$REPORT_FILE" "$LOG_FILE"

    # Initialize
    init_report

    # Run validation checks
    validate_go_error_handling
    validate_kubernetes_security
    validate_secure_logging
    validate_input_validation
    validate_container_security
    validate_secrets_management

    # Generate recommendations and finalize
    generate_recommendations
    finalize_report

    # Display summary
    echo ""
    log_info "=== SECURITY VALIDATION SUMMARY ==="
    if command -v jq >/dev/null 2>&1 && [ -f "$REPORT_FILE" ]; then
        jq -r '.security_validation | to_entries[] | "\(.key): \(.value.status)"' "$REPORT_FILE" | while read -r line; do
            local check_name=$(echo "$line" | cut -d: -f1)
            local check_status=$(echo "$line" | cut -d: -f2 | tr -d ' ')
            case "$check_status" in
                "passed") log_success "$check_name: PASSED" ;;
                "failed") log_error "$check_name: FAILED" ;;
                "warning") log_warning "$check_name: WARNING" ;;
                *) log_info "$check_name: $check_status" ;;
            esac
        done
    fi

    echo ""
    log_info "Detailed report available at: $REPORT_FILE"
    log_info "Full log available at: $LOG_FILE"

    exit $EXIT_CODE
}

# Help function
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Security validation script for O-RAN Intent-MANO Network Slicing project.

Options:
    -h, --help     Show this help message
    -v, --verbose  Enable verbose output
    --report-only  Only generate report, don't exit with error code

This script validates:
- Go error handling patterns
- Kubernetes security configurations
- Secure logging implementation
- Input validation
- Container security
- Secrets management

Report will be saved to: security-validation-report.json
Log will be saved to: security-validation.log
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -v|--verbose)
            set -x
            shift
            ;;
        --report-only)
            EXIT_CODE=0  # Don't exit with error even if validation fails
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Check dependencies
if ! command -v git >/dev/null 2>&1; then
    log_warning "git not found - commit information will not be available"
fi

# Run main function
main "$@"