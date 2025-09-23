#!/bin/bash

# Comprehensive Security Check Script for O-RAN Intent-MANO Network Slicing
# Copyright 2024 O-RAN Intent MANO Project
# SPDX-License-Identifier: Apache-2.0

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REPORT_FILE="${PROJECT_ROOT}/security-check-report.json"
LOG_FILE="${PROJECT_ROOT}/security-check.log"
TEMP_DIR="${PROJECT_ROOT}/.security-check-temp"
EXIT_CODE=0

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly PURPLE='\033[0;35m'
readonly CYAN='\033[0;36m'
readonly NC='\033[0m' # No Color

# Tool versions and requirements
readonly GOSEC_VERSION="2.18.2"
readonly CHECKOV_VERSION="3.0.0"
readonly KUBESEC_VERSION="2.14.0"

# Initialize logging
mkdir -p "$(dirname "$LOG_FILE")"
mkdir -p "$TEMP_DIR"
exec 1> >(tee -a "$LOG_FILE")
exec 2> >(tee -a "$LOG_FILE" >&2)

# Report structure
declare -A SECURITY_REPORT=(
    ["timestamp"]=""
    ["environment"]=""
    ["go_security_scan"]=""
    ["kubernetes_manifest_scan"]=""
    ["network_policy_validation"]=""
    ["security_context_validation"]=""
    ["image_specification_validation"]=""
    ["service_account_validation"]=""
    ["overall_status"]=""
    ["total_issues"]="0"
    ["critical_issues"]="0"
    ["high_issues"]="0"
    ["medium_issues"]="0"
    ["low_issues"]="0"
)

# Helper functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*" >&2
    EXIT_CODE=1
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

info() {
    echo -e "${CYAN}[INFO]${NC} $*"
}

# Check if running in CI environment
is_ci_environment() {
    [[ "${CI:-}" == "true" ]] || [[ -n "${GITHUB_ACTIONS:-}" ]] || [[ -n "${JENKINS_URL:-}" ]]
}

# Install required tools
install_security_tools() {
    log "Installing security scanning tools..."

    local tools_installed=0

    # Install gosec
    if ! command -v gosec &> /dev/null; then
        info "Installing gosec..."
        if is_ci_environment; then
            curl -sfL https://raw.githubusercontent.com/securecodewarrior/gosec/master/install.sh | sh -s -- -b "$(go env GOPATH)/bin" "v${GOSEC_VERSION}"
        else
            go install "github.com/securecodewarrior/gosec/v2/cmd/gosec@v${GOSEC_VERSION}"
        fi
        tools_installed=$((tools_installed + 1))
    fi

    # Install checkov
    if ! command -v checkov &> /dev/null; then
        info "Installing checkov..."
        if is_ci_environment; then
            pip3 install "checkov==${CHECKOV_VERSION}"
        else
            pip3 install --user "checkov==${CHECKOV_VERSION}"
        fi
        tools_installed=$((tools_installed + 1))
    fi

    # Install kubesec
    if ! command -v kubesec &> /dev/null; then
        info "Installing kubesec..."
        if is_ci_environment; then
            curl -sSX GET "https://api.github.com/repos/controlplaneio/kubesec/releases/latest" \
                | grep browser_download_url | grep linux-amd64 | cut -d'"' -f4 \
                | xargs curl -sSL -o /usr/local/bin/kubesec
            chmod +x /usr/local/bin/kubesec
        else
            local kubesec_url="https://github.com/controlplaneio/kubesec/releases/download/v${KUBESEC_VERSION}/kubesec_linux_amd64.tar.gz"
            curl -sSL "$kubesec_url" | tar -xz -C "$TEMP_DIR"
            sudo mv "$TEMP_DIR/kubesec" /usr/local/bin/
        fi
        tools_installed=$((tools_installed + 1))
    fi

    if [[ $tools_installed -gt 0 ]]; then
        success "Installed $tools_installed security tools"
    else
        info "All security tools already installed"
    fi
}

# Validate Go code with gosec
validate_go_security() {
    log "Running Go security analysis with gosec..."

    local go_files
    go_files=$(find "$PROJECT_ROOT" -name "*.go" -not -path "*/vendor/*" -not -path "*/.git/*" | wc -l)

    if [[ $go_files -eq 0 ]]; then
        warning "No Go files found for security analysis"
        SECURITY_REPORT["go_security_scan"]="SKIPPED"
        return 0
    fi

    info "Found $go_files Go files to analyze"

    local gosec_output="${TEMP_DIR}/gosec-report.json"
    local gosec_txt="${TEMP_DIR}/gosec-report.txt"

    # Run gosec with comprehensive checks
    if gosec -fmt json -out "$gosec_output" -stdout -nosec=false -tests -severity=medium \
        -confidence=medium -exclude-generated \
        -include="G101,G102,G103,G104,G106,G107,G108,G109,G110,G201,G202,G203,G204,G301,G302,G303,G304,G305,G306,G307,G401,G402,G403,G404,G501,G502,G503,G504,G505,G601" \
        "$PROJECT_ROOT/..." 2> "$gosec_txt"; then

        # Parse results
        local issues_count
        issues_count=$(jq -r '.Stats.files_scanned // 0' "$gosec_output" 2>/dev/null || echo "0")
        local found_issues
        found_issues=$(jq -r '.Issues | length' "$gosec_output" 2>/dev/null || echo "0")

        if [[ $found_issues -gt 0 ]]; then
            warning "Found $found_issues security issues in Go code"

            # Categorize issues by severity
            local critical high medium low
            critical=$(jq -r '.Issues | map(select(.severity == "HIGH" and .confidence == "HIGH")) | length' "$gosec_output" 2>/dev/null || echo "0")
            high=$(jq -r '.Issues | map(select(.severity == "MEDIUM" and .confidence == "HIGH")) | length' "$gosec_output" 2>/dev/null || echo "0")
            medium=$(jq -r '.Issues | map(select(.severity == "LOW" and .confidence == "HIGH")) | length' "$gosec_output" 2>/dev/null || echo "0")
            low=$(jq -r '.Issues | map(select(.confidence == "MEDIUM" or .confidence == "LOW")) | length' "$gosec_output" 2>/dev/null || echo "0")

            SECURITY_REPORT["critical_issues"]=$((${SECURITY_REPORT["critical_issues"]} + critical))
            SECURITY_REPORT["high_issues"]=$((${SECURITY_REPORT["high_issues"]} + high))
            SECURITY_REPORT["medium_issues"]=$((${SECURITY_REPORT["medium_issues"]} + medium))
            SECURITY_REPORT["low_issues"]=$((${SECURITY_REPORT["low_issues"]} + low))

            # Display top issues
            info "Top Go security issues:"
            jq -r '.Issues[] | "  - \(.rule_id): \(.details) (File: \(.file), Line: \(.line))"' "$gosec_output" 2>/dev/null | head -10

            if [[ $critical -gt 0 ]] || [[ $high -gt 0 ]]; then
                error "Critical or high-severity security issues found in Go code"
                SECURITY_REPORT["go_security_scan"]="FAILED"
            else
                warning "Medium/low-severity security issues found in Go code"
                SECURITY_REPORT["go_security_scan"]="WARNING"
            fi
        else
            success "No security issues found in Go code"
            SECURITY_REPORT["go_security_scan"]="PASSED"
        fi
    else
        error "gosec scan failed"
        SECURITY_REPORT["go_security_scan"]="ERROR"
    fi

    SECURITY_REPORT["total_issues"]=$((${SECURITY_REPORT["total_issues"]} + found_issues))
}

# Validate Kubernetes manifests with checkov and kubesec
validate_kubernetes_manifests() {
    log "Running Kubernetes manifest security analysis..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -not -path "*/.git/*" -not -path "*/node_modules/*" | grep -E "(k8s|kubernetes|deploy|chart)" | head -20)

    if [[ -z "$k8s_files" ]]; then
        warning "No Kubernetes manifest files found"
        SECURITY_REPORT["kubernetes_manifest_scan"]="SKIPPED"
        return 0
    fi

    info "Found Kubernetes manifests to analyze"

    local checkov_output="${TEMP_DIR}/checkov-report.json"
    local kubesec_output="${TEMP_DIR}/kubesec-report.json"
    local total_issues=0
    local failed_files=0

    # Run checkov on Kubernetes files
    if echo "$k8s_files" | xargs checkov -f --framework kubernetes \
        --output json --output-file "$checkov_output" \
        --check CKV_K8S_1,CKV_K8S_2,CKV_K8S_3,CKV_K8S_4,CKV_K8S_5,CKV_K8S_6,CKV_K8S_7,CKV_K8S_8,CKV_K8S_9,CKV_K8S_10 \
        --check CKV_K8S_11,CKV_K8S_12,CKV_K8S_13,CKV_K8S_14,CKV_K8S_15,CKV_K8S_16,CKV_K8S_17,CKV_K8S_18,CKV_K8S_19,CKV_K8S_20 \
        --check CKV_K8S_21,CKV_K8S_22,CKV_K8S_23,CKV_K8S_25,CKV_K8S_28,CKV_K8S_29,CKV_K8S_30,CKV_K8S_31,CKV_K8S_37,CKV_K8S_38 \
        --check CKV_K8S_40,CKV_K8S_43,CKV_K8S_49 2>/dev/null || true; then

        # Parse checkov results
        local checkov_failed
        checkov_failed=$(jq -r '.results.failed_checks | length' "$checkov_output" 2>/dev/null || echo "0")
        total_issues=$((total_issues + checkov_failed))

        if [[ $checkov_failed -gt 0 ]]; then
            warning "Checkov found $checkov_failed security issues"
            info "Top checkov issues:"
            jq -r '.results.failed_checks[] | "  - \(.check_id): \(.check_name) (File: \(.file_path))"' "$checkov_output" 2>/dev/null | head -5
        fi
    fi

    # Run kubesec on individual YAML files
    echo '[]' > "$kubesec_output"
    while IFS= read -r yaml_file; do
        if [[ -f "$yaml_file" ]]; then
            local file_result
            file_result=$(kubesec scan "$yaml_file" 2>/dev/null || echo '[]')

            # Check if file has security issues
            local score
            score=$(echo "$file_result" | jq -r '.[0].score // 0' 2>/dev/null || echo "0")

            if [[ $(echo "$score < 0" | bc -l 2>/dev/null || echo "1") -eq 1 ]]; then
                failed_files=$((failed_files + 1))
                warning "Security issues in $yaml_file (score: $score)"

                # Show critical advisories
                echo "$file_result" | jq -r '.[0].scoring.critical[]?.reason // empty' 2>/dev/null | while read -r reason; do
                    echo "    CRITICAL: $reason"
                done
            fi
        fi
    done <<< "$k8s_files"

    # Update totals
    SECURITY_REPORT["total_issues"]=$((${SECURITY_REPORT["total_issues"]} + total_issues))

    if [[ $total_issues -gt 0 ]] || [[ $failed_files -gt 0 ]]; then
        if [[ $failed_files -gt 5 ]] || [[ $total_issues -gt 20 ]]; then
            error "Critical Kubernetes security issues found"
            SECURITY_REPORT["kubernetes_manifest_scan"]="FAILED"
        else
            warning "Some Kubernetes security issues found"
            SECURITY_REPORT["kubernetes_manifest_scan"]="WARNING"
        fi
    else
        success "Kubernetes manifests passed security validation"
        SECURITY_REPORT["kubernetes_manifest_scan"]="PASSED"
    fi
}

# Check NetworkPolicy presence and correctness
validate_network_policies() {
    log "Validating NetworkPolicy configurations..."

    local netpol_files
    netpol_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: NetworkPolicy" {} \; 2>/dev/null || true)

    if [[ -z "$netpol_files" ]]; then
        error "No NetworkPolicy resources found - network segmentation required"
        SECURITY_REPORT["network_policy_validation"]="FAILED"
        return 1
    fi

    local total_policies=0
    local valid_policies=0
    local invalid_policies=0

    while IFS= read -r netpol_file; do
        if [[ -f "$netpol_file" ]]; then
            info "Validating NetworkPolicy in $netpol_file"

            # Extract NetworkPolicy resources
            local policies
            policies=$(yq eval 'select(.kind == "NetworkPolicy")' "$netpol_file" 2>/dev/null || true)

            if [[ -n "$policies" ]]; then
                total_policies=$((total_policies + 1))

                # Check for required fields
                local has_podSelector has_policyTypes has_ingress_or_egress
                has_podSelector=$(echo "$policies" | yq eval '.spec.podSelector != null' - 2>/dev/null || echo "false")
                has_policyTypes=$(echo "$policies" | yq eval '.spec.policyTypes != null' - 2>/dev/null || echo "false")
                has_ingress_or_egress=$(echo "$policies" | yq eval '.spec.ingress != null or .spec.egress != null' - 2>/dev/null || echo "false")

                if [[ "$has_podSelector" == "true" ]] && [[ "$has_policyTypes" == "true" ]] && [[ "$has_ingress_or_egress" == "true" ]]; then
                    valid_policies=$((valid_policies + 1))

                    # Check for overly permissive policies
                    local is_permissive
                    is_permissive=$(echo "$policies" | yq eval '.spec.podSelector == {} and (.spec.ingress[].from == [] or .spec.egress[].to == [])' - 2>/dev/null || echo "false")

                    if [[ "$is_permissive" == "true" ]]; then
                        warning "Overly permissive NetworkPolicy found in $netpol_file"
                    fi
                else
                    invalid_policies=$((invalid_policies + 1))
                    error "Invalid NetworkPolicy configuration in $netpol_file"
                    echo "  Missing: podSelector($has_podSelector), policyTypes($has_policyTypes), ingress/egress($has_ingress_or_egress)"
                fi
            fi
        fi
    done <<< "$netpol_files"

    info "NetworkPolicy validation: $valid_policies valid, $invalid_policies invalid out of $total_policies total"

    if [[ $invalid_policies -gt 0 ]]; then
        error "Invalid NetworkPolicy configurations found"
        SECURITY_REPORT["network_policy_validation"]="FAILED"
    elif [[ $total_policies -eq 0 ]]; then
        error "No NetworkPolicy resources found"
        SECURITY_REPORT["network_policy_validation"]="FAILED"
    else
        success "NetworkPolicy validation passed"
        SECURITY_REPORT["network_policy_validation"]="PASSED"
    fi
}

# Validate container security contexts
validate_security_contexts() {
    log "Validating container security contexts..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: \(Deployment\|StatefulSet\|DaemonSet\|Pod\)" {} \; 2>/dev/null || true)

    if [[ -z "$k8s_files" ]]; then
        warning "No workload resources found for security context validation"
        SECURITY_REPORT["security_context_validation"]="SKIPPED"
        return 0
    fi

    local total_containers=0
    local secure_containers=0
    local insecure_containers=0

    while IFS= read -r k8s_file; do
        if [[ -f "$k8s_file" ]]; then
            # Extract containers from workload resources
            local containers
            containers=$(yq eval '.spec.template.spec.containers[]? // .spec.containers[]?' "$k8s_file" 2>/dev/null || true)

            if [[ -n "$containers" ]]; then
                local container_count
                container_count=$(echo "$containers" | yq eval 'length' - 2>/dev/null || echo "0")
                total_containers=$((total_containers + container_count))

                # Check each container's security context
                echo "$containers" | yq eval '.[]' - 2>/dev/null | while read -r container; do
                    local has_security_context runAsNonRoot readOnlyRootFilesystem allowPrivilegeEscalation

                    has_security_context=$(echo "$container" | yq eval '.securityContext != null' - 2>/dev/null || echo "false")
                    runAsNonRoot=$(echo "$container" | yq eval '.securityContext.runAsNonRoot == true' - 2>/dev/null || echo "false")
                    readOnlyRootFilesystem=$(echo "$container" | yq eval '.securityContext.readOnlyRootFilesystem == true' - 2>/dev/null || echo "false")
                    allowPrivilegeEscalation=$(echo "$container" | yq eval '.securityContext.allowPrivilegeEscalation == false' - 2>/dev/null || echo "false")

                    if [[ "$has_security_context" == "true" ]] && [[ "$runAsNonRoot" == "true" ]] &&
                       [[ "$readOnlyRootFilesystem" == "true" ]] && [[ "$allowPrivilegeEscalation" == "true" ]]; then
                        secure_containers=$((secure_containers + 1))
                    else
                        insecure_containers=$((insecure_containers + 1))
                        warning "Insecure container found in $k8s_file"
                        echo "  Security context issues: securityContext($has_security_context), runAsNonRoot($runAsNonRoot), readOnlyRootFilesystem($readOnlyRootFilesystem), allowPrivilegeEscalation($allowPrivilegeEscalation)"
                    fi
                done
            fi
        fi
    done <<< "$k8s_files"

    info "Security context validation: $secure_containers secure, $insecure_containers insecure out of $total_containers total containers"

    if [[ $insecure_containers -gt 0 ]]; then
        if [[ $insecure_containers -gt $((total_containers / 2)) ]]; then
            error "Majority of containers have insecure security contexts"
            SECURITY_REPORT["security_context_validation"]="FAILED"
        else
            warning "Some containers have insecure security contexts"
            SECURITY_REPORT["security_context_validation"]="WARNING"
        fi
    elif [[ $total_containers -eq 0 ]]; then
        warning "No containers found for security context validation"
        SECURITY_REPORT["security_context_validation"]="SKIPPED"
    else
        success "All containers have secure security contexts"
        SECURITY_REPORT["security_context_validation"]="PASSED"
    fi
}

# Validate image specifications (digest/tag format)
validate_image_specifications() {
    log "Validating container image specifications..."

    local k8s_files
    k8s_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "image:" {} \; 2>/dev/null || true)

    if [[ -z "$k8s_files" ]]; then
        warning "No files with container images found"
        SECURITY_REPORT["image_specification_validation"]="SKIPPED"
        return 0
    fi

    local total_images=0
    local secure_images=0
    local insecure_images=0

    while IFS= read -r k8s_file; do
        if [[ -f "$k8s_file" ]]; then
            # Extract image references
            local images
            images=$(grep -o 'image: .*' "$k8s_file" 2>/dev/null | sed 's/image: //' | tr -d '"' || true)

            while IFS= read -r image; do
                if [[ -n "$image" ]]; then
                    total_images=$((total_images + 1))

                    # Check image format
                    if [[ "$image" =~ @sha256:[a-f0-9]{64}$ ]]; then
                        # Image uses digest - most secure
                        secure_images=$((secure_images + 1))
                        info "✓ Secure image (digest): $image"
                    elif [[ "$image" =~ :[a-zA-Z0-9_.-]+$ ]] && [[ ! "$image" =~ :latest$ ]]; then
                        # Image uses specific tag (not latest) - acceptable
                        secure_images=$((secure_images + 1))
                        info "✓ Acceptable image (tagged): $image"
                    elif [[ "$image" =~ :latest$ ]] || [[ ! "$image" =~ : ]]; then
                        # Image uses latest tag or no tag - insecure
                        insecure_images=$((insecure_images + 1))
                        warning "✗ Insecure image (latest/no tag): $image in $k8s_file"
                    else
                        # Unknown format
                        insecure_images=$((insecure_images + 1))
                        warning "✗ Unknown image format: $image in $k8s_file"
                    fi
                fi
            done <<< "$images"
        fi
    done <<< "$k8s_files"

    info "Image validation: $secure_images secure, $insecure_images insecure out of $total_images total images"

    if [[ $insecure_images -gt 0 ]]; then
        if [[ $insecure_images -gt $((total_images / 3)) ]]; then
            error "Too many insecure image specifications found"
            SECURITY_REPORT["image_specification_validation"]="FAILED"
        else
            warning "Some insecure image specifications found"
            SECURITY_REPORT["image_specification_validation"]="WARNING"
        fi
    elif [[ $total_images -eq 0 ]]; then
        warning "No container images found for validation"
        SECURITY_REPORT["image_specification_validation"]="SKIPPED"
    else
        success "All images use secure specifications"
        SECURITY_REPORT["image_specification_validation"]="PASSED"
    fi
}

# Validate Service Account configurations
validate_service_accounts() {
    log "Validating Service Account configurations..."

    local sa_files
    sa_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -exec grep -l "kind: ServiceAccount\|serviceAccount:" {} \; 2>/dev/null || true)

    if [[ -z "$sa_files" ]]; then
        warning "No Service Account configurations found"
        SECURITY_REPORT["service_account_validation"]="SKIPPED"
        return 0
    fi

    local total_sa=0
    local secure_sa=0
    local insecure_sa=0

    while IFS= read -r sa_file; do
        if [[ -f "$sa_file" ]]; then
            # Check ServiceAccount resources
            local sa_resources
            sa_resources=$(yq eval 'select(.kind == "ServiceAccount")' "$sa_file" 2>/dev/null || true)

            if [[ -n "$sa_resources" ]]; then
                total_sa=$((total_sa + 1))

                # Check automountServiceAccountToken
                local automount
                automount=$(echo "$sa_resources" | yq eval '.automountServiceAccountToken == false' - 2>/dev/null || echo "false")

                if [[ "$automount" == "true" ]]; then
                    secure_sa=$((secure_sa + 1))
                    info "✓ Secure ServiceAccount (automount disabled): $sa_file"
                else
                    insecure_sa=$((insecure_sa + 1))
                    warning "✗ Insecure ServiceAccount (automount enabled): $sa_file"
                fi
            fi

            # Check workload resources using ServiceAccounts
            local workloads
            workloads=$(yq eval 'select(.kind == "Deployment" or .kind == "StatefulSet" or .kind == "DaemonSet" or .kind == "Pod")' "$sa_file" 2>/dev/null || true)

            if [[ -n "$workloads" ]]; then
                # Check if serviceAccount is explicitly set
                local sa_name
                sa_name=$(echo "$workloads" | yq eval '.spec.template.spec.serviceAccount // .spec.serviceAccount // empty' - 2>/dev/null || true)

                if [[ -n "$sa_name" ]] && [[ "$sa_name" != "default" ]]; then
                    info "✓ Workload uses custom ServiceAccount: $sa_name in $sa_file"
                else
                    warning "✗ Workload uses default ServiceAccount in $sa_file"
                fi
            fi
        fi
    done <<< "$sa_files"

    info "ServiceAccount validation: $secure_sa secure, $insecure_sa insecure out of $total_sa total"

    if [[ $insecure_sa -gt 0 ]]; then
        warning "Some Service Accounts have security issues"
        SECURITY_REPORT["service_account_validation"]="WARNING"
    elif [[ $total_sa -eq 0 ]]; then
        warning "No Service Accounts found for validation"
        SECURITY_REPORT["service_account_validation"]="SKIPPED"
    else
        success "All Service Accounts are properly configured"
        SECURITY_REPORT["service_account_validation"]="PASSED"
    fi
}

# Generate final security report
generate_security_report() {
    log "Generating security compliance report..."

    # Set timestamp and environment
    SECURITY_REPORT["timestamp"]=$(date -Iseconds)
    SECURITY_REPORT["environment"]=$(if is_ci_environment; then echo "CI"; else echo "LOCAL"; fi)

    # Calculate overall status
    local failed_checks=0
    local warning_checks=0
    local passed_checks=0

    for check in "go_security_scan" "kubernetes_manifest_scan" "network_policy_validation" "security_context_validation" "image_specification_validation" "service_account_validation"; do
        case "${SECURITY_REPORT[$check]}" in
            "FAILED"|"ERROR") failed_checks=$((failed_checks + 1)) ;;
            "WARNING") warning_checks=$((warning_checks + 1)) ;;
            "PASSED") passed_checks=$((passed_checks + 1)) ;;
        esac
    done

    if [[ $failed_checks -gt 0 ]]; then
        SECURITY_REPORT["overall_status"]="FAILED"
    elif [[ $warning_checks -gt 0 ]]; then
        SECURITY_REPORT["overall_status"]="WARNING"
    else
        SECURITY_REPORT["overall_status"]="PASSED"
    fi

    # Generate JSON report
    cat > "$REPORT_FILE" << EOF
{
  "timestamp": "${SECURITY_REPORT[timestamp]}",
  "environment": "${SECURITY_REPORT[environment]}",
  "overall_status": "${SECURITY_REPORT[overall_status]}",
  "summary": {
    "total_issues": ${SECURITY_REPORT[total_issues]},
    "critical_issues": ${SECURITY_REPORT[critical_issues]},
    "high_issues": ${SECURITY_REPORT[high_issues]},
    "medium_issues": ${SECURITY_REPORT[medium_issues]},
    "low_issues": ${SECURITY_REPORT[low_issues]}
  },
  "checks": {
    "go_security_scan": "${SECURITY_REPORT[go_security_scan]}",
    "kubernetes_manifest_scan": "${SECURITY_REPORT[kubernetes_manifest_scan]}",
    "network_policy_validation": "${SECURITY_REPORT[network_policy_validation]}",
    "security_context_validation": "${SECURITY_REPORT[security_context_validation]}",
    "image_specification_validation": "${SECURITY_REPORT[image_specification_validation]}",
    "service_account_validation": "${SECURITY_REPORT[service_account_validation]}"
  },
  "statistics": {
    "failed_checks": $failed_checks,
    "warning_checks": $warning_checks,
    "passed_checks": $passed_checks
  }
}
EOF

    # Display summary
    echo
    echo "======================== SECURITY COMPLIANCE REPORT ========================"
    echo "Timestamp: ${SECURITY_REPORT[timestamp]}"
    echo "Environment: ${SECURITY_REPORT[environment]}"
    echo "Overall Status: ${SECURITY_REPORT[overall_status]}"
    echo
    echo "Summary:"
    echo "  Total Issues: ${SECURITY_REPORT[total_issues]}"
    echo "  Critical: ${SECURITY_REPORT[critical_issues]}"
    echo "  High: ${SECURITY_REPORT[high_issues]}"
    echo "  Medium: ${SECURITY_REPORT[medium_issues]}"
    echo "  Low: ${SECURITY_REPORT[low_issues]}"
    echo
    echo "Check Results:"
    echo "  Go Security Scan: ${SECURITY_REPORT[go_security_scan]}"
    echo "  Kubernetes Manifest Scan: ${SECURITY_REPORT[kubernetes_manifest_scan]}"
    echo "  NetworkPolicy Validation: ${SECURITY_REPORT[network_policy_validation]}"
    echo "  Security Context Validation: ${SECURITY_REPORT[security_context_validation]}"
    echo "  Image Specification Validation: ${SECURITY_REPORT[image_specification_validation]}"
    echo "  Service Account Validation: ${SECURITY_REPORT[service_account_validation]}"
    echo
    echo "Statistics:"
    echo "  Failed Checks: $failed_checks"
    echo "  Warning Checks: $warning_checks"
    echo "  Passed Checks: $passed_checks"
    echo "=========================================================================="
    echo

    success "Security report generated: $REPORT_FILE"

    # Set exit code based on results
    if [[ "${SECURITY_REPORT[overall_status]}" == "FAILED" ]]; then
        error "Security compliance check FAILED"
        EXIT_CODE=1
    elif [[ "${SECURITY_REPORT[overall_status]}" == "WARNING" ]]; then
        warning "Security compliance check completed with WARNINGS"
        EXIT_CODE=0  # Don't fail CI for warnings
    else
        success "Security compliance check PASSED"
        EXIT_CODE=0
    fi
}

# Cleanup function
cleanup() {
    if [[ -d "$TEMP_DIR" ]]; then
        rm -rf "$TEMP_DIR"
    fi
}

# Main execution
main() {
    trap cleanup EXIT

    log "Starting comprehensive security check for O-RAN Intent-MANO..."
    log "Project root: $PROJECT_ROOT"
    log "Environment: $(if is_ci_environment; then echo "CI/CD"; else echo "Local"; fi)"

    # Install required tools
    install_security_tools

    # Run security validations
    validate_go_security
    validate_kubernetes_manifests
    validate_network_policies
    validate_security_contexts
    validate_image_specifications
    validate_service_accounts

    # Generate final report
    generate_security_report

    exit $EXIT_CODE
}

# Script usage
usage() {
    cat << EOF
Usage: $0 [options]

Options:
    -h, --help              Show this help message
    -v, --verbose           Enable verbose output
    --skip-install          Skip tool installation
    --report-file FILE      Custom report file location
    --temp-dir DIR          Custom temporary directory

Examples:
    $0                      Run full security check
    $0 --skip-install       Run without installing tools
    $0 --verbose            Run with detailed output

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
        --skip-install)
            SKIP_INSTALL=1
            shift
            ;;
        --report-file)
            REPORT_FILE="$2"
            shift 2
            ;;
        --temp-dir)
            TEMP_DIR="$2"
            shift 2
            ;;
        *)
            error "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Run main function
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi