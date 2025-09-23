#!/bin/bash
# Validation script for E2E deployment suite
# Provides quick validation and smoke testing capabilities

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

# Configuration
DRY_RUN=${DRY_RUN:-false}
QUICK_MODE=${QUICK_MODE:-false}
VERBOSE=${VERBOSE:-false}

usage() {
    cat << EOF
Usage: $0 [OPTIONS] [COMMAND]

Validation script for E2E deployment suite

Commands:
    smoke           Run smoke tests (quick validation)
    prereqs         Check prerequisites only
    config          Validate configuration files
    scenarios       Validate scenario definitions
    thresholds      Validate threshold configurations
    all             Run all validations (default)

Options:
    --dry-run       Show what would be done without executing
    --quick         Skip time-intensive validations
    --verbose       Enable verbose output
    --help          Show this help message

Examples:
    $0 smoke                    # Quick smoke test
    $0 prereqs                  # Check prerequisites
    $0 config                   # Validate configs
    $0 --dry-run all           # Show validation plan
    $0 --quick --verbose all   # Quick verbose validation

EOF
}

# Smoke test - quick validation
run_smoke_test() {
    log_info "Running smoke test..."

    # Basic file structure
    validate_file_structure

    # Configuration syntax
    validate_config_syntax

    # Script permissions
    validate_script_permissions

    # Basic kubectl connectivity
    if kubectl cluster-info &>/dev/null; then
        log_info "✓ Kubernetes cluster accessible"
    else
        log_warning "✗ Kubernetes cluster not accessible"
        return 1
    fi

    log_info "✓ Smoke test passed"
    return 0
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    local failed=0

    # Required tools
    local tools=("kubectl" "yq" "bc" "python3")
    for tool in "${tools[@]}"; do
        if command -v "$tool" &>/dev/null; then
            log_info "✓ $tool available"
        else
            log_error "✗ $tool not found"
            ((failed++))
        fi
    done

    # Python dependencies
    if python3 -c "import json, subprocess, time, yaml" &>/dev/null; then
        log_info "✓ Python dependencies available"
    else
        log_warning "✗ Some Python dependencies missing"
        log_info "Install with: pip3 install pyyaml"
        ((failed++))
    fi

    # Kubernetes cluster
    if kubectl cluster-info &>/dev/null; then
        log_info "✓ Kubernetes cluster accessible"

        # Check for metrics server
        if kubectl top nodes &>/dev/null; then
            log_info "✓ Metrics server available"
        else
            log_warning "! Metrics server not available (metrics collection limited)"
        fi
    else
        log_error "✗ Kubernetes cluster not accessible"
        ((failed++))
    fi

    if [[ $failed -eq 0 ]]; then
        log_info "✓ All prerequisites satisfied"
        return 0
    else
        log_error "✗ $failed prerequisite(s) failed"
        return 1
    fi
}

# Validate file structure
validate_file_structure() {
    log_info "Validating file structure..."

    local required_files=(
        "run_suite.sh"
        "collect_metrics.py"
        "test_harness.py"
        "lib/common.sh"
        "lib/deploy_ran.sh"
        "lib/deploy_tn.sh"
        "lib/deploy_cn.sh"
        "config/thresholds.yaml"
        "config/monitoring.yaml"
        "scenarios/embb.yaml"
        "scenarios/urllc.yaml"
        "scenarios/miot.yaml"
    )

    local missing=0
    for file in "${required_files[@]}"; do
        if [[ -f "${SCRIPT_DIR}/$file" ]]; then
            log_debug "✓ $file exists"
        else
            log_error "✗ Missing file: $file"
            ((missing++))
        fi
    done

    # Required directories
    local required_dirs=("lib" "config" "scenarios" "results" "logs")
    for dir in "${required_dirs[@]}"; do
        if [[ -d "${SCRIPT_DIR}/$dir" ]]; then
            log_debug "✓ $dir/ directory exists"
        else
            log_warning "✗ Missing directory: $dir/"
            mkdir -p "${SCRIPT_DIR}/$dir"
            log_info "Created directory: $dir/"
        fi
    done

    if [[ $missing -eq 0 ]]; then
        log_info "✓ File structure valid"
        return 0
    else
        log_error "✗ $missing required file(s) missing"
        return 1
    fi
}

# Validate configuration syntax
validate_config_syntax() {
    log_info "Validating configuration syntax..."

    local config_files=(
        "config/thresholds.yaml"
        "config/monitoring.yaml"
        "scenarios/embb.yaml"
        "scenarios/urllc.yaml"
        "scenarios/miot.yaml"
    )

    local invalid=0
    for config in "${config_files[@]}"; do
        local config_path="${SCRIPT_DIR}/$config"
        if [[ -f "$config_path" ]]; then
            if yq eval '.' "$config_path" &>/dev/null; then
                log_debug "✓ $config syntax valid"
            else
                log_error "✗ Invalid YAML syntax: $config"
                ((invalid++))
            fi
        fi
    done

    if [[ $invalid -eq 0 ]]; then
        log_info "✓ Configuration syntax valid"
        return 0
    else
        log_error "✗ $invalid configuration file(s) have invalid syntax"
        return 1
    fi
}

# Validate script permissions
validate_script_permissions() {
    log_info "Validating script permissions..."

    local scripts=(
        "run_suite.sh"
        "test_harness.py"
        "lib/common.sh"
        "lib/deploy_ran.sh"
        "lib/deploy_tn.sh"
        "lib/deploy_cn.sh"
    )

    local not_executable=0
    for script in "${scripts[@]}"; do
        local script_path="${SCRIPT_DIR}/$script"
        if [[ -f "$script_path" ]]; then
            if [[ -x "$script_path" ]]; then
                log_debug "✓ $script is executable"
            else
                log_warning "! $script not executable, fixing..."
                chmod +x "$script_path"
                log_info "Fixed permissions: $script"
            fi
        fi
    done

    log_info "✓ Script permissions validated"
    return 0
}

# Validate scenario configurations
validate_scenarios() {
    log_info "Validating scenario configurations..."

    local scenarios=("embb" "urllc" "miot")
    local invalid=0

    for scenario in "${scenarios[@]}"; do
        local scenario_file="${SCRIPT_DIR}/scenarios/${scenario}.yaml"

        if [[ ! -f "$scenario_file" ]]; then
            log_error "✗ Missing scenario file: $scenario.yaml"
            ((invalid++))
            continue
        fi

        # Validate required fields
        local required_fields=(
            ".metadata.name"
            ".spec.qos.bandwidth"
            ".spec.qos.latency"
            ".network_functions.ran"
            ".network_functions.cn"
        )

        for field in "${required_fields[@]}"; do
            if yq eval "$field" "$scenario_file" &>/dev/null; then
                log_debug "✓ $scenario: $field present"
            else
                log_error "✗ $scenario: missing field $field"
                ((invalid++))
            fi
        done

        # Validate QoS values
        local bandwidth=$(yq eval '.spec.qos.bandwidth' "$scenario_file" 2>/dev/null || echo "0")
        local latency=$(yq eval '.spec.qos.latency' "$scenario_file" 2>/dev/null || echo "0")

        if (( $(echo "$bandwidth >= 0.5 && $bandwidth <= 5.0" | bc -l) )); then
            log_debug "✓ $scenario: bandwidth valid ($bandwidth Mbps)"
        else
            log_error "✗ $scenario: invalid bandwidth ($bandwidth Mbps)"
            ((invalid++))
        fi

        if (( $(echo "$latency >= 1.0 && $latency <= 20.0" | bc -l) )); then
            log_debug "✓ $scenario: latency valid ($latency ms)"
        else
            log_error "✗ $scenario: invalid latency ($latency ms)"
            ((invalid++))
        fi
    done

    if [[ $invalid -eq 0 ]]; then
        log_info "✓ All scenarios valid"
        return 0
    else
        log_error "✗ $invalid scenario validation(s) failed"
        return 1
    fi
}

# Validate threshold configurations
validate_thresholds() {
    log_info "Validating threshold configurations..."

    local thresholds_file="${SCRIPT_DIR}/config/thresholds.yaml"
    if [[ ! -f "$thresholds_file" ]]; then
        log_error "✗ Thresholds file not found"
        return 1
    fi

    local invalid=0

    # Check deployment time thresholds
    local series=("fast_series" "slow_series")
    local scenarios=("embb" "urllc" "miot")

    for series_name in "${series[@]}"; do
        for scenario in "${scenarios[@]}"; do
            local target=$(yq eval ".deployment_times.${series_name}.${scenario}.target" "$thresholds_file" 2>/dev/null || echo "null")
            local tolerance=$(yq eval ".deployment_times.${series_name}.${scenario}.tolerance" "$thresholds_file" 2>/dev/null || echo "null")

            if [[ "$target" != "null" && "$target" -gt 0 ]]; then
                log_debug "✓ ${series_name}.${scenario} target: ${target}s"
            else
                log_error "✗ Invalid target for ${series_name}.${scenario}: $target"
                ((invalid++))
            fi

            if [[ "$tolerance" != "null" && "$tolerance" -gt 0 ]]; then
                log_debug "✓ ${series_name}.${scenario} tolerance: ${tolerance}s"
            else
                log_error "✗ Invalid tolerance for ${series_name}.${scenario}: $tolerance"
                ((invalid++))
            fi
        done
    done

    # Check resource limits
    local cpu_max=$(yq eval '.resource_limits.smo.cpu_max_cores' "$thresholds_file" 2>/dev/null || echo "null")
    local memory_max=$(yq eval '.resource_limits.smo.memory_max_mb' "$thresholds_file" 2>/dev/null || echo "null")

    if [[ "$cpu_max" != "null" && $(echo "$cpu_max > 0" | bc -l) -eq 1 ]]; then
        log_debug "✓ SMO CPU limit: $cpu_max cores"
    else
        log_error "✗ Invalid SMO CPU limit: $cpu_max"
        ((invalid++))
    fi

    if [[ "$memory_max" != "null" && "$memory_max" -gt 0 ]]; then
        log_debug "✓ SMO memory limit: $memory_max MB"
    else
        log_error "✗ Invalid SMO memory limit: $memory_max"
        ((invalid++))
    fi

    if [[ $invalid -eq 0 ]]; then
        log_info "✓ Threshold configurations valid"
        return 0
    else
        log_error "✗ $invalid threshold validation(s) failed"
        return 1
    fi
}

# Main validation function
run_all_validations() {
    log_info "Running comprehensive validation..."

    local failed=0

    # Run individual validations
    check_prerequisites || ((failed++))
    validate_file_structure || ((failed++))
    validate_config_syntax || ((failed++))
    validate_script_permissions || ((failed++))
    validate_scenarios || ((failed++))
    validate_thresholds || ((failed++))

    if [[ $failed -eq 0 ]]; then
        log_info "✓ All validations passed"
        return 0
    else
        log_error "✗ $failed validation(s) failed"
        return 1
    fi
}

# Main execution
main() {
    local command=${1:-all}

    # Parse options
    while [[ $# -gt 0 ]]; do
        case $1 in
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --quick)
                QUICK_MODE=true
                shift
                ;;
            --verbose)
                DEBUG=1
                VERBOSE=true
                shift
                ;;
            --help)
                usage
                exit 0
                ;;
            smoke|prereqs|config|scenarios|thresholds|all)
                command=$1
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done

    if [[ "$DRY_RUN" == "true" ]]; then
        log_info "DRY RUN MODE - showing validation plan"
        echo "Would run: $command validation"
        exit 0
    fi

    log_info "Starting validation: $command"

    case $command in
        smoke)
            run_smoke_test
            ;;
        prereqs)
            check_prerequisites
            ;;
        config)
            validate_config_syntax
            ;;
        scenarios)
            validate_scenarios
            ;;
        thresholds)
            validate_thresholds
            ;;
        all)
            run_all_validations
            ;;
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac

    local exit_code=$?
    if [[ $exit_code -eq 0 ]]; then
        log_info "✓ Validation completed successfully"
    else
        log_error "✗ Validation failed"
    fi

    exit $exit_code
}

# Execute main function
main "$@"