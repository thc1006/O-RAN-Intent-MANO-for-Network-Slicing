#!/bin/bash
# O-RAN Monitoring Stack - Grafana Dashboards Health Check
# This script validates Grafana dashboards are accessible and functional

set -euo pipefail

# Configuration
NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
GRAFANA_SERVICE="${GRAFANA_SERVICE:-grafana}"
TIMEOUT="${TIMEOUT:-60}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
MIN_DASHBOARDS="${MIN_DASHBOARDS:-1}"
EXPECTED_DASHBOARDS="${EXPECTED_DASHBOARDS:-O-RAN Intent Processing,Network Slice Performance,VNF Deployment Overview,Infrastructure Monitoring}"

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

# Function to check if Grafana is running
check_grafana_pod() {
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=grafana --no-headers 2>/dev/null || true)

    if [[ -z "$pods" ]]; then
        log_error "No Grafana pods found in namespace '$NAMESPACE'"
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
        log_error "No Grafana pods are running and ready"
        exit 1
    fi

    log_success "$running_pods Grafana pod(s) are running and ready"
}

# Function to get Grafana admin credentials
get_grafana_credentials() {
    # Try to get password from secret first
    local secret_password
    secret_password=$(kubectl get secret -n "$NAMESPACE" grafana-admin-credentials -o jsonpath='{.data.password}' 2>/dev/null | base64 -d 2>/dev/null || echo "")

    if [[ -n "$secret_password" ]]; then
        GRAFANA_PASSWORD="$secret_password"
        log_success "Retrieved Grafana password from Kubernetes secret"
    else
        log_warning "Using default Grafana credentials (admin/admin)"
    fi
}

# Function to create port-forward and return PID
setup_port_forward() {
    local local_port=13000
    local port_forward_pid

    # Find an available port
    while lsof -i :$local_port &> /dev/null; do
        ((local_port++))
    done

    # Start port-forward in background
    kubectl port-forward -n "$NAMESPACE" service/"$GRAFANA_SERVICE" $local_port:3000 &> /dev/null &
    port_forward_pid=$!

    # Wait for port-forward to be ready
    sleep 3

    # Test if port-forward is working
    if ! lsof -i :$local_port &> /dev/null; then
        log_error "Port-forward failed to start"
        kill $port_forward_pid 2>/dev/null || true
        exit 1
    fi

    echo "$port_forward_pid:$local_port"
}

# Function to make authenticated Grafana API calls
grafana_api_call() {
    local endpoint="$1"
    local local_port="$2"
    local method="${3:-GET}"
    local data="${4:-}"

    local curl_args=(
        -s
        --max-time "$TIMEOUT"
        -u "$GRAFANA_USER:$GRAFANA_PASSWORD"
        -H "Content-Type: application/json"
        -X "$method"
    )

    if [[ -n "$data" ]]; then
        curl_args+=(-d "$data")
    fi

    curl "${curl_args[@]}" "http://localhost:$local_port$endpoint"
}

# Function to check Grafana health
check_grafana_health() {
    local port_info="$1"
    local local_port="${port_info#*:}"

    log_info "Checking Grafana health..."

    local health_response
    if ! health_response=$(grafana_api_call "/api/health" "$local_port"); then
        log_error "Failed to query Grafana health API"
        return 1
    fi

    # Parse health response
    local database_status
    database_status=$(echo "$health_response" | jq -r '.database // "unknown"')

    if [[ "$database_status" == "ok" ]]; then
        log_success "Grafana database is healthy"
    else
        log_warning "Grafana database status: $database_status"
    fi

    # Check overall health
    local version
    version=$(echo "$health_response" | jq -r '.version // "unknown"')
    log_info "Grafana version: $version"

    return 0
}

# Function to get and validate dashboards
get_dashboards() {
    local port_info="$1"
    local local_port="${port_info#*:}"

    log_info "Retrieving Grafana dashboards..."

    local dashboards_response
    if ! dashboards_response=$(grafana_api_call "/api/search?type=dash-db" "$local_port"); then
        log_error "Failed to query Grafana dashboards API"
        return 1
    fi

    # Check if response is valid JSON array
    if ! echo "$dashboards_response" | jq -e 'type == "array"' &> /dev/null; then
        log_error "Invalid dashboards response format"
        return 1
    fi

    echo "$dashboards_response"
}

# Function to validate dashboard count and content
validate_dashboards() {
    local dashboards_json="$1"

    local dashboard_count
    dashboard_count=$(echo "$dashboards_json" | jq 'length')

    log_info "Found $dashboard_count dashboards"

    if [[ $dashboard_count -lt $MIN_DASHBOARDS ]]; then
        log_error "Only $dashboard_count dashboards found (minimum required: $MIN_DASHBOARDS)"
        return 1
    fi

    log_success "Dashboard count meets minimum requirement"

    # List all dashboards
    log_info "Available dashboards:"
    echo "$dashboards_json" | jq -r '.[] | "  - \(.title) (UID: \(.uid), ID: \(.id))"'

    # Check for expected dashboards
    IFS=',' read -ra expected_dashboards_array <<< "$EXPECTED_DASHBOARDS"
    local missing_dashboards=()
    local found_dashboards=()

    for expected_dashboard in "${expected_dashboards_array[@]}"; do
        local found=false
        while IFS= read -r dashboard_title; do
            if [[ -n "$dashboard_title" ]] && [[ "$dashboard_title" =~ $expected_dashboard ]]; then
                found=true
                found_dashboards+=("$expected_dashboard")
                break
            fi
        done < <(echo "$dashboards_json" | jq -r '.[].title')

        if [[ "$found" == false ]]; then
            missing_dashboards+=("$expected_dashboard")
        fi
    done

    if [[ ${#missing_dashboards[@]} -gt 0 ]]; then
        log_warning "Missing expected dashboards: ${missing_dashboards[*]}"
    else
        log_success "All expected dashboards are present"
    fi

    if [[ ${#found_dashboards[@]} -gt 0 ]]; then
        log_success "Found expected dashboards: ${found_dashboards[*]}"
    fi

    return 0
}

# Function to test dashboard rendering
test_dashboard_rendering() {
    local dashboards_json="$1"
    local port_info="$2"
    local local_port="${port_info#*:}"

    log_info "Testing dashboard rendering..."

    # Get a few dashboards to test
    local test_dashboards
    test_dashboards=$(echo "$dashboards_json" | jq -r '.[0:3] | .[] | .uid')

    local successful_renders=0
    local total_tests=0

    while IFS= read -r uid; do
        if [[ -n "$uid" ]]; then
            ((total_tests++))

            local dashboard_response
            if dashboard_response=$(grafana_api_call "/api/dashboards/uid/$uid" "$local_port"); then
                # Check if response contains dashboard data
                local dashboard_title
                dashboard_title=$(echo "$dashboard_response" | jq -r '.dashboard.title // "unknown"')

                if [[ "$dashboard_title" != "unknown" ]] && [[ "$dashboard_title" != "null" ]]; then
                    ((successful_renders++))
                    log_success "  ✓ Dashboard '$dashboard_title' rendered successfully"
                else
                    log_warning "  ✗ Dashboard $uid returned invalid data"
                fi
            else
                log_warning "  ✗ Dashboard $uid failed to render"
            fi
        fi
    done <<< "$test_dashboards"

    if [[ $total_tests -gt 0 ]]; then
        local success_rate=$((successful_renders * 100 / total_tests))
        log_info "Dashboard rendering success rate: $success_rate% ($successful_renders/$total_tests)"

        if [[ $success_rate -lt 80 ]]; then
            log_warning "Dashboard rendering success rate is below 80%"
            return 1
        else
            log_success "Dashboard rendering success rate is acceptable"
        fi
    fi

    return 0
}

# Function to check data sources
check_data_sources() {
    local port_info="$1"
    local local_port="${port_info#*:}"

    log_info "Checking Grafana data sources..."

    local datasources_response
    if ! datasources_response=$(grafana_api_call "/api/datasources" "$local_port"); then
        log_error "Failed to query Grafana data sources API"
        return 1
    fi

    local datasource_count
    datasource_count=$(echo "$datasources_response" | jq 'length')

    log_info "Found $datasource_count data source(s)"

    if [[ $datasource_count -eq 0 ]]; then
        log_warning "No data sources configured"
        return 1
    fi

    # List data sources and check their health
    local healthy_datasources=0

    while IFS= read -r datasource; do
        if [[ -n "$datasource" ]]; then
            local name
            local type
            local url
            name=$(echo "$datasource" | jq -r '.name')
            type=$(echo "$datasource" | jq -r '.type')
            url=$(echo "$datasource" | jq -r '.url // "not specified"')

            log_info "  Data source: $name (type: $type, url: $url)"

            # Test data source connectivity
            local ds_id
            ds_id=$(echo "$datasource" | jq -r '.id')

            local test_response
            if test_response=$(grafana_api_call "/api/datasources/$ds_id/health" "$local_port" POST '{}'); then
                local status
                status=$(echo "$test_response" | jq -r '.status // "unknown"')

                if [[ "$status" == "success" ]]; then
                    ((healthy_datasources++))
                    log_success "    ✓ Data source is healthy"
                else
                    log_warning "    ✗ Data source test failed: $status"
                fi
            else
                log_warning "    ✗ Could not test data source connectivity"
            fi
        fi
    done < <(echo "$datasources_response" | jq -c '.[]')

    if [[ $healthy_datasources -gt 0 ]]; then
        log_success "$healthy_datasources data source(s) are healthy"
    else
        log_warning "No healthy data sources found"
    fi

    return 0
}

# Function to check Grafana plugins
check_plugins() {
    local port_info="$1"
    local local_port="${port_info#*:}"

    log_info "Checking Grafana plugins..."

    local plugins_response
    if ! plugins_response=$(grafana_api_call "/api/plugins" "$local_port"); then
        log_warning "Could not retrieve plugins information"
        return 0
    fi

    local plugin_count
    plugin_count=$(echo "$plugins_response" | jq 'length')

    log_info "Found $plugin_count plugin(s) installed"

    # List installed plugins
    echo "$plugins_response" | jq -r '.[] | "  - \(.name) v\(.info.version) (\(.id))"' | head -10

    return 0
}

# Function to check dashboard panels for errors
validate_dashboard_panels() {
    local dashboards_json="$1"
    local port_info="$2"
    local local_port="${port_info#*:}"

    log_info "Validating dashboard panels..."

    # Take first dashboard for detailed validation
    local first_dashboard_uid
    first_dashboard_uid=$(echo "$dashboards_json" | jq -r '.[0].uid // empty')

    if [[ -z "$first_dashboard_uid" ]]; then
        log_warning "No dashboards available for panel validation"
        return 0
    fi

    local dashboard_detail
    if ! dashboard_detail=$(grafana_api_call "/api/dashboards/uid/$first_dashboard_uid" "$local_port"); then
        log_warning "Could not retrieve dashboard details for panel validation"
        return 0
    fi

    local panel_count
    panel_count=$(echo "$dashboard_detail" | jq '.dashboard.panels | length')

    if [[ $panel_count -gt 0 ]]; then
        log_success "Dashboard has $panel_count panels"

        # Check if panels have queries
        local panels_with_queries
        panels_with_queries=$(echo "$dashboard_detail" | jq '[.dashboard.panels[] | select(.targets != null and (.targets | length) > 0)] | length')

        log_info "$panels_with_queries panels have configured queries"
    else
        log_warning "Dashboard has no panels"
    fi

    return 0
}

# Function to cleanup port-forward
cleanup_port_forward() {
    local port_forward_pid="$1"

    if [[ -n "$port_forward_pid" ]] && kill -0 "$port_forward_pid" 2>/dev/null; then
        kill "$port_forward_pid" 2>/dev/null || true
        log_info "Cleaned up port-forward process"
    fi
}

# Main execution function
main() {
    log_info "=== O-RAN Grafana Dashboards Health Check ==="
    log_info "Namespace: $NAMESPACE"
    log_info "Service: $GRAFANA_SERVICE"
    log_info "Timeout: ${TIMEOUT}s"
    log_info "Minimum dashboards: $MIN_DASHBOARDS"
    echo

    # Pre-flight checks
    check_kubectl
    check_namespace
    check_grafana_pod
    get_grafana_credentials

    # Setup port-forward
    log_info "Setting up port-forward to Grafana..."
    local port_info
    port_info=$(setup_port_forward)
    local port_forward_pid="${port_info%:*}"

    # Ensure cleanup on exit
    trap "cleanup_port_forward $port_forward_pid" EXIT

    # Health checks
    if ! check_grafana_health "$port_info"; then
        log_error "Grafana health check failed"
        exit 1
    fi

    # Get dashboards
    local dashboards_json
    if ! dashboards_json=$(get_dashboards "$port_info"); then
        log_error "Failed to retrieve dashboards"
        exit 1
    fi

    # Validate dashboards
    if ! validate_dashboards "$dashboards_json"; then
        log_error "Dashboard validation failed"
        exit 1
    fi

    # Test rendering
    if ! test_dashboard_rendering "$dashboards_json" "$port_info"; then
        log_warning "Some dashboard rendering tests failed"
    fi

    # Additional checks
    check_data_sources "$port_info"
    check_plugins "$port_info"
    validate_dashboard_panels "$dashboards_json" "$port_info"

    echo
    log_success "=== Grafana dashboards health check completed successfully ==="

    # Output summary JSON for automation
    local summary
    summary=$(echo "$dashboards_json" | jq --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)" '{
        timestamp: $timestamp,
        total_dashboards: length,
        dashboard_titles: [.[] | .title],
        status: "healthy"
    }')

    echo "HEALTH_CHECK_SUMMARY: $summary"
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Health check failed with exit code $exit_code"

    # Cleanup port-forward if it exists
    if [[ -n "${port_forward_pid:-}" ]]; then
        cleanup_port_forward "$port_forward_pid"
    fi

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
O-RAN Grafana Dashboards Health Check

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE       Kubernetes namespace (default: oran-monitoring)
    -s, --service SERVICE          Grafana service name (default: grafana)
    -t, --timeout TIMEOUT         Query timeout in seconds (default: 60)
    -u, --user USER               Grafana username (default: admin)
    -p, --password PASSWORD       Grafana password (default: admin)
    -m, --min-dashboards MIN      Minimum dashboards required (default: 1)
    -d, --expected-dashboards LIST Comma-separated list of expected dashboard names
    -h, --help                    Show this help message

ENVIRONMENT VARIABLES:
    MONITORING_NAMESPACE          Same as --namespace
    GRAFANA_SERVICE              Same as --service
    TIMEOUT                      Same as --timeout
    GRAFANA_USER                 Same as --user
    GRAFANA_PASSWORD             Same as --password
    MIN_DASHBOARDS               Same as --min-dashboards
    EXPECTED_DASHBOARDS          Same as --expected-dashboards

EXAMPLES:
    # Basic health check
    $0

    # Check with custom credentials
    $0 --user myuser --password mypass

    # Check for specific dashboards
    $0 --expected-dashboards "O-RAN Overview,Network Performance"

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
                GRAFANA_SERVICE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -u|--user)
                GRAFANA_USER="$2"
                shift 2
                ;;
            -p|--password)
                GRAFANA_PASSWORD="$2"
                shift 2
                ;;
            -m|--min-dashboards)
                MIN_DASHBOARDS="$2"
                shift 2
                ;;
            -d|--expected-dashboards)
                EXPECTED_DASHBOARDS="$2"
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

    for cmd in kubectl curl jq lsof; do
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