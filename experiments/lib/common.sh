#!/bin/bash
# Common functions for deployment scripts

# Colors for logging
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO $(date +'%H:%M:%S')]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN $(date +'%H:%M:%S')]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR $(date +'%H:%M:%S')]${NC} $1"
}

log_debug() {
    if [[ "${DEBUG:-}" == "1" ]]; then
        echo -e "${BLUE}[DEBUG $(date +'%H:%M:%S')]${NC} $1"
    fi
}

# Check if required tools are available
check_prerequisites() {
    local tools=("kubectl" "yq" "bc")

    for tool in "${tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_error "Required tool not found: $tool"
            return 1
        fi
    done

    # Check kubectl connection
    if ! kubectl cluster-info &>/dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        return 1
    fi

    return 0
}

# Wait for deployment to be ready
wait_for_deployment() {
    local deployment=$1
    local namespace=${2:-default}
    local timeout=${3:-300}

    log_info "Waiting for deployment ${deployment} in namespace ${namespace}"

    if kubectl wait --for=condition=available \
        deployment/"${deployment}" \
        -n "${namespace}" \
        --timeout="${timeout}s"; then
        log_info "Deployment ${deployment} is ready"
        return 0
    else
        log_error "Deployment ${deployment} failed to become ready"
        return 1
    fi
}

# Wait for VNF to be ready
wait_for_vnf() {
    local vnf_name=$1
    local namespace=${2:-oran-system}
    local timeout=${3:-300}

    log_info "Waiting for VNF ${vnf_name} in namespace ${namespace}"

    # Check if VNF exists
    if ! kubectl get vnf "${vnf_name}" -n "${namespace}" &>/dev/null; then
        log_error "VNF ${vnf_name} not found"
        return 1
    fi

    # Wait for VNF to be ready
    local end_time=$(($(date +%s) + timeout))
    while [[ $(date +%s) -lt $end_time ]]; do
        local phase=$(kubectl get vnf "${vnf_name}" -n "${namespace}" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        case $phase in
            "Running"|"Ready")
                log_info "VNF ${vnf_name} is ready (phase: ${phase})"
                return 0
                ;;
            "Failed")
                log_error "VNF ${vnf_name} failed"
                return 1
                ;;
            *)
                log_debug "VNF ${vnf_name} phase: ${phase}"
                sleep 5
                ;;
        esac
    done

    log_error "VNF ${vnf_name} failed to become ready within ${timeout}s"
    return 1
}

# Wait for TNSlice to be active
wait_for_tnslice() {
    local slice_name=$1
    local namespace=${2:-default}
    local timeout=${3:-180}

    log_info "Waiting for TNSlice ${slice_name} in namespace ${namespace}"

    # Check if TNSlice exists
    if ! kubectl get tnslice "${slice_name}" -n "${namespace}" &>/dev/null; then
        log_error "TNSlice ${slice_name} not found"
        return 1
    fi

    # Wait for TNSlice to be active
    local end_time=$(($(date +%s) + timeout))
    while [[ $(date +%s) -lt $end_time ]]; do
        local phase=$(kubectl get tnslice "${slice_name}" -n "${namespace}" \
            -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")

        case $phase in
            "Active")
                log_info "TNSlice ${slice_name} is active"
                return 0
                ;;
            "Failed")
                log_error "TNSlice ${slice_name} failed"
                return 1
                ;;
            *)
                log_debug "TNSlice ${slice_name} phase: ${phase}"
                sleep 5
                ;;
        esac
    done

    log_error "TNSlice ${slice_name} failed to become active within ${timeout}s"
    return 1
}

# Get resource usage
get_pod_resource_usage() {
    local pod_name=$1
    local namespace=${2:-default}

    local metrics=$(kubectl top pod "${pod_name}" -n "${namespace}" --no-headers 2>/dev/null)
    if [[ -n "$metrics" ]]; then
        echo "$metrics"
    else
        echo "0m 0Mi"
    fi
}

# Get node resource usage
get_node_resource_usage() {
    local node_name=$1

    local metrics=$(kubectl top node "${node_name}" --no-headers 2>/dev/null)
    if [[ -n "$metrics" ]]; then
        echo "$metrics"
    else
        echo "0m 0% 0Mi 0%"
    fi
}

# Calculate percentage
calculate_percentage() {
    local actual=$1
    local target=$2

    if [[ "$target" == "0" ]]; then
        echo "0"
    else
        echo "scale=2; ($actual - $target) / $target * 100" | bc
    fi
}

# Check if namespace exists
ensure_namespace() {
    local namespace=$1

    if ! kubectl get namespace "$namespace" &>/dev/null; then
        log_info "Creating namespace: $namespace"
        kubectl create namespace "$namespace"
    fi
}

# Apply manifest with retry
apply_manifest() {
    local manifest=$1
    local retries=${2:-3}

    for ((i=1; i<=retries; i++)); do
        if kubectl apply -f "$manifest"; then
            return 0
        else
            log_warning "Apply failed (attempt $i/$retries), retrying..."
            sleep 5
        fi
    done

    log_error "Failed to apply manifest after $retries attempts"
    return 1
}

# Get scenario configuration value
get_scenario_config() {
    local scenario_file=$1
    local yaml_path=$2
    local default_value=${3:-""}

    if [[ -f "$scenario_file" ]]; then
        yq eval "$yaml_path" "$scenario_file" 2>/dev/null || echo "$default_value"
    else
        echo "$default_value"
    fi
}

# Simulate deployment delay based on mode
simulate_deployment_delay() {
    local base_delay=$1
    local mode=${2:-fast}

    case $mode in
        fast)
            # 80% of base delay
            local delay=$(echo "scale=0; $base_delay * 0.8" | bc)
            ;;
        slow)
            # 150% of base delay
            local delay=$(echo "scale=0; $base_delay * 1.5" | bc)
            ;;
        *)
            local delay=$base_delay
            ;;
    esac

    log_debug "Simulating deployment delay: ${delay}s (mode: $mode)"
    sleep "$delay"
}

# Export environment variables for child scripts
export -f log_info log_warning log_error log_debug
export -f wait_for_deployment wait_for_vnf wait_for_tnslice
export -f get_pod_resource_usage get_node_resource_usage
export -f calculate_percentage ensure_namespace apply_manifest
export -f get_scenario_config simulate_deployment_delay