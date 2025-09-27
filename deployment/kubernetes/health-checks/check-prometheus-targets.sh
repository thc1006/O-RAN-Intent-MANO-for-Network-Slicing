#!/bin/bash
# O-RAN Monitoring Stack - Prometheus Targets Health Check
# This script verifies all Prometheus targets are UP and healthy

set -euo pipefail

# Configuration
NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
PROMETHEUS_SERVICE="${PROMETHEUS_SERVICE:-prometheus}"
TIMEOUT="${TIMEOUT:-60}"
MIN_HEALTHY_TARGETS="${MIN_HEALTHY_TARGETS:-3}"
EXPECTED_JOBS="${EXPECTED_JOBS:-kubernetes-apiservers,kubernetes-nodes,kubernetes-pods,oran-nlp,oran-orchestrator,oran-ran,oran-cn,oran-tn,oran-vnf-operator}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if kubectl is available
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    log_success "kubectl is available and connected to cluster"
}

# Function to check if namespace exists
check_namespace() {
    if ! kubectl get namespace "$NAMESPACE" &> /dev/null; then
        log_error "Namespace '$NAMESPACE' does not exist"
        exit 1
    fi

    log_success "Namespace '$NAMESPACE' exists"
}

# Function to check if Prometheus is running
check_prometheus_pod() {
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus --no-headers 2>/dev/null || true)

    if [[ -z "$pods" ]]; then
        log_error "No Prometheus pods found in namespace '$NAMESPACE'"
        exit 1
    fi

    local running_pods=0
    while IFS= read -r line; do
        if [[ -n "$line" ]]; then
            local status=$(echo "$line" | awk '{print $3}')
            local ready=$(echo "$line" | awk '{print $2}')

            if [[ "$status" == "Running" ]] && [[ "$ready" =~ ^[0-9]+/[0-9]+$ ]]; then
                local ready_count=$(echo "$ready" | cut -d'/' -f1)
                local total_count=$(echo "$ready" | cut -d'/' -f2)

                if [[ "$ready_count" == "$total_count" ]]; then
                    ((running_pods++))
                fi
            fi
        fi
    done <<< "$pods"

    if [[ $running_pods -eq 0 ]]; then
        log_error "No Prometheus pods are running and ready"
        exit 1
    fi

    log_success "$running_pods Prometheus pod(s) are running and ready"
}

# Function to get Prometheus targets via port-forward
get_prometheus_targets() {
    local port_forward_pid
    local local_port=19090

    # Start port-forward in background
    kubectl port-forward -n "$NAMESPACE" service/"$PROMETHEUS_SERVICE" $local_port:9090 &> /dev/null &
    port_forward_pid=$!

    # Wait for port-forward to be ready
    sleep 3

    # Make sure to cleanup port-forward on exit
    trap "kill $port_forward_pid 2>/dev/null || true" EXIT

    # Query targets API
    local targets_json
    if ! targets_json=$(curl -s --max-time $TIMEOUT "http://localhost:$local_port/api/v1/targets" 2>/dev/null); then
        log_error "Failed to query Prometheus targets API"
        kill $port_forward_pid 2>/dev/null || true
        exit 1
    fi

    # Kill port-forward
    kill $port_forward_pid 2>/dev/null || true

    echo "$targets_json"
}

# Function to parse and validate targets
validate_targets() {
    local targets_json="$1"

    # Check if response is valid JSON
    if ! echo "$targets_json" | jq empty 2>/dev/null; then
        log_error "Invalid JSON response from Prometheus targets API"
        return 1
    fi

    # Check API status
    local status
    status=$(echo "$targets_json" | jq -r '.status // empty')
    if [[ "$status" != "success" ]]; then
        log_error "Prometheus API returned status: $status"
        return 1
    fi

    # Get active targets
    local active_targets
    active_targets=$(echo "$targets_json" | jq '.data.activeTargets // []')

    if [[ "$active_targets" == "[]" ]]; then
        log_error "No active targets found"
        return 1
    fi

    # Count total and healthy targets
    local total_targets
    local healthy_targets
    total_targets=$(echo "$active_targets" | jq 'length')
    healthy_targets=$(echo "$active_targets" | jq '[.[] | select(.health == "up")] | length')

    log_info "Total targets: $total_targets"
    log_info "Healthy targets: $healthy_targets"

    if [[ $healthy_targets -lt $MIN_HEALTHY_TARGETS ]]; then
        log_error "Only $healthy_targets targets are healthy (minimum required: $MIN_HEALTHY_TARGETS)"
        return 1
    fi

    # Check for unhealthy targets
    local unhealthy_targets
    unhealthy_targets=$(echo "$active_targets" | jq '[.[] | select(.health != "up")]')
    local unhealthy_count
    unhealthy_count=$(echo "$unhealthy_targets" | jq 'length')

    if [[ $unhealthy_count -gt 0 ]]; then
        log_warning "$unhealthy_count targets are unhealthy:"
        echo "$unhealthy_targets" | jq -r '.[] | "  - Job: \(.labels.job // "unknown"), Instance: \(.labels.instance // "unknown"), Health: \(.health), Error: \(.lastError // "none")"'
    fi

    # Check for expected jobs
    local found_jobs
    found_jobs=$(echo "$active_targets" | jq -r '[.[] | .labels.job] | unique | .[]' | tr '\n' ',' | sed 's/,$//')

    log_info "Found jobs: $found_jobs"

    IFS=',' read -ra expected_jobs_array <<< "$EXPECTED_JOBS"
    local missing_jobs=()

    for job in "${expected_jobs_array[@]}"; do
        if ! echo "$found_jobs" | grep -q "$job"; then
            missing_jobs+=("$job")
        fi
    done

    if [[ ${#missing_jobs[@]} -gt 0 ]]; then
        log_warning "Missing expected jobs: ${missing_jobs[*]}"
    else
        log_success "All expected jobs are present"
    fi

    # Display detailed target information
    log_info "Target details:"
    echo "$active_targets" | jq -r '.[] | "  \(.labels.job // "unknown") - \(.labels.instance // "unknown") - \(.health) - \(.lastScrape // "never")"' | sort

    return 0
}

# Function to check target scrape intervals
check_scrape_intervals() {
    local targets_json="$1"

    log_info "Checking scrape intervals..."

    # Get targets with recent scrapes (within last 2 minutes)
    local recent_scrapes
    recent_scrapes=$(echo "$targets_json" | jq --arg cutoff "$(date -d '2 minutes ago' -u +%Y-%m-%dT%H:%M:%S.%3NZ)" '
        [.data.activeTargets[] | select(.lastScrape > $cutoff)] | length')

    local total_active
    total_active=$(echo "$targets_json" | jq '.data.activeTargets | length')

    if [[ $recent_scrapes -lt $((total_active / 2)) ]]; then
        log_warning "Only $recent_scrapes out of $total_active targets have been scraped recently"
    else
        log_success "$recent_scrapes out of $total_active targets have recent scrapes"
    fi
}

# Function to validate target labels
validate_target_labels() {
    local targets_json="$1"

    log_info "Validating target labels..."

    # Check for required labels
    local targets_missing_job
    targets_missing_job=$(echo "$targets_json" | jq '[.data.activeTargets[] | select(.labels.job == null or .labels.job == "")] | length')

    if [[ $targets_missing_job -gt 0 ]]; then
        log_warning "$targets_missing_job targets are missing 'job' label"
    fi

    # Check for O-RAN specific labels
    local oran_targets
    oran_targets=$(echo "$targets_json" | jq '[.data.activeTargets[] | select(.labels.job | startswith("oran-"))] | length')

    if [[ $oran_targets -gt 0 ]]; then
        log_success "$oran_targets O-RAN specific targets found"

        # Check for O-RAN component labels
        echo "$targets_json" | jq -r '.data.activeTargets[] | select(.labels.job | startswith("oran-")) | "  \(.labels.job) - component: \(.labels.component // "missing"), oran_service: \(.labels.oran_service // "missing")"'
    else
        log_warning "No O-RAN specific targets found"
    fi
}

# Function to test target connectivity
test_target_connectivity() {
    local targets_json="$1"

    log_info "Testing basic target connectivity..."

    # Get unique endpoints from healthy targets
    local endpoints
    endpoints=$(echo "$targets_json" | jq -r '.data.activeTargets[] | select(.health == "up") | .scrapeUrl' | head -5)

    local reachable=0
    local total=0

    while IFS= read -r endpoint; do
        if [[ -n "$endpoint" ]]; then
            ((total++))
            if curl -s --max-time 5 "$endpoint" &> /dev/null; then
                ((reachable++))
                log_success "  ✓ $endpoint"
            else
                log_warning "  ✗ $endpoint (not reachable directly)"
            fi
        fi
    done <<< "$endpoints"

    if [[ $total -gt 0 ]]; then
        log_info "Direct connectivity test: $reachable/$total endpoints reachable"
    fi
}

# Function to check for duplicate targets
check_duplicate_targets() {
    local targets_json="$1"

    log_info "Checking for duplicate targets..."

    local duplicates
    duplicates=$(echo "$targets_json" | jq -r '
        .data.activeTargets |
        group_by(.scrapeUrl) |
        map(select(length > 1)) |
        map({url: .[0].scrapeUrl, count: length}) |
        .[]? |
        "  \(.url) appears \(.count) times"
    ')

    if [[ -n "$duplicates" ]]; then
        log_warning "Duplicate targets found:"
        echo "$duplicates"
    else
        log_success "No duplicate targets found"
    fi
}

# Main execution function
main() {
    log_info "=== O-RAN Prometheus Targets Health Check ==="
    log_info "Namespace: $NAMESPACE"
    log_info "Service: $PROMETHEUS_SERVICE"
    log_info "Timeout: ${TIMEOUT}s"
    log_info "Minimum healthy targets: $MIN_HEALTHY_TARGETS"
    echo

    # Pre-flight checks
    check_kubectl
    check_namespace
    check_prometheus_pod

    # Get targets data
    log_info "Querying Prometheus targets..."
    local targets_json
    targets_json=$(get_prometheus_targets)

    # Validate targets
    if validate_targets "$targets_json"; then
        log_success "Target validation passed"
    else
        log_error "Target validation failed"
        exit 1
    fi

    # Additional checks
    check_scrape_intervals "$targets_json"
    validate_target_labels "$targets_json"
    check_duplicate_targets "$targets_json"
    test_target_connectivity "$targets_json"

    echo
    log_success "=== Prometheus targets health check completed successfully ==="

    # Output summary JSON for automation
    local summary
    summary=$(echo "$targets_json" | jq '{
        timestamp: now | todateiso8601,
        total_targets: .data.activeTargets | length,
        healthy_targets: [.data.activeTargets[] | select(.health == "up")] | length,
        unhealthy_targets: [.data.activeTargets[] | select(.health != "up")] | length,
        jobs: [.data.activeTargets[] | .labels.job] | unique,
        status: "healthy"
    }')

    echo "HEALTH_CHECK_SUMMARY: $summary"
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Health check failed with exit code $exit_code"

    # Output failure summary
    local error_summary='{
        "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)'",
        "status": "failed",
        "exit_code": '$exit_code'
    }'

    echo "HEALTH_CHECK_SUMMARY: $error_summary"
    exit $exit_code
}

trap handle_error ERR

# Help function
show_help() {
    cat << EOF
O-RAN Prometheus Targets Health Check

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE       Kubernetes namespace (default: oran-monitoring)
    -s, --service SERVICE          Prometheus service name (default: prometheus)
    -t, --timeout TIMEOUT         Query timeout in seconds (default: 60)
    -m, --min-targets MIN          Minimum healthy targets required (default: 3)
    -j, --expected-jobs JOBS       Comma-separated list of expected jobs
    -h, --help                     Show this help message

ENVIRONMENT VARIABLES:
    MONITORING_NAMESPACE           Same as --namespace
    PROMETHEUS_SERVICE             Same as --service
    TIMEOUT                        Same as --timeout
    MIN_HEALTHY_TARGETS           Same as --min-targets
    EXPECTED_JOBS                 Same as --expected-jobs

EXAMPLES:
    # Basic health check
    $0

    # Check with custom namespace
    $0 --namespace my-monitoring

    # Check with minimum 5 healthy targets
    $0 --min-targets 5

    # Check for specific jobs
    $0 --expected-jobs "kubernetes-apiservers,oran-nlp,oran-orchestrator"

EXIT CODES:
    0  - All checks passed
    1  - Health check failed
    2  - Invalid arguments

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -s|--service)
                PROMETHEUS_SERVICE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -m|--min-targets)
                MIN_HEALTHY_TARGETS="$2"
                shift 2
                ;;
            -j|--expected-jobs)
                EXPECTED_JOBS="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    for cmd in kubectl curl jq; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again"
        exit 1
    fi
}

# Entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_args "$@"
    check_dependencies
    main
fi