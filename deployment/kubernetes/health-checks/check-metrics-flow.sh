#!/bin/bash
# O-RAN Monitoring Stack - End-to-End Metrics Flow Test
# This script validates complete metrics flow from scrape to visualization

set -euo pipefail

# Configuration
NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
PROMETHEUS_SERVICE="${PROMETHEUS_SERVICE:-prometheus}"
GRAFANA_SERVICE="${GRAFANA_SERVICE:-grafana}"
TIMEOUT="${TIMEOUT:-120}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASSWORD="${GRAFANA_PASSWORD:-admin}"
TEST_METRIC="${TEST_METRIC:-up}"
RETENTION_CHECK_HOURS="${RETENTION_CHECK_HOURS:-1}"

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

# Function to setup port forwards for both services
setup_port_forwards() {
    local prometheus_port=19090
    local grafana_port=13000

    # Find available ports
    while lsof -i :$prometheus_port &> /dev/null; do
        ((prometheus_port++))
    done

    while lsof -i :$grafana_port &> /dev/null; do
        ((grafana_port++))
    done

    # Start Prometheus port-forward
    kubectl port-forward -n "$NAMESPACE" service/"$PROMETHEUS_SERVICE" $prometheus_port:9090 &> /dev/null &
    local prometheus_pid=$!

    # Start Grafana port-forward
    kubectl port-forward -n "$NAMESPACE" service/"$GRAFANA_SERVICE" $grafana_port:3000 &> /dev/null &
    local grafana_pid=$!

    # Wait for port-forwards to be ready
    sleep 5

    # Verify port-forwards are working
    if ! lsof -i :$prometheus_port &> /dev/null; then
        log_error "Prometheus port-forward failed"
        kill $prometheus_pid $grafana_pid 2>/dev/null || true
        exit 1
    fi

    if ! lsof -i :$grafana_port &> /dev/null; then
        log_error "Grafana port-forward failed"
        kill $prometheus_pid $grafana_pid 2>/dev/null || true
        exit 1
    fi

    echo "$prometheus_pid:$prometheus_port:$grafana_pid:$grafana_port"
}

# Function to cleanup port forwards
cleanup_port_forwards() {
    local port_info="$1"
    IFS=':' read -r prometheus_pid prometheus_port grafana_pid grafana_port <<< "$port_info"

    kill $prometheus_pid $grafana_pid 2>/dev/null || true
    log_info "Cleaned up port-forward processes"
}

# Function to make Prometheus API calls
prometheus_query() {
    local query="$1"
    local prometheus_port="$2"

    curl -s --max-time "$TIMEOUT" "http://localhost:$prometheus_port/api/v1/query?query=$query"
}

# Function to make Grafana API calls
grafana_api_call() {
    local endpoint="$1"
    local grafana_port="$2"
    local method="${3:-GET}"

    curl -s --max-time "$TIMEOUT" -u "$GRAFANA_USER:$GRAFANA_PASSWORD" \
         -H "Content-Type: application/json" -X "$method" \
         "http://localhost:$grafana_port$endpoint"
}

# Test 1: Verify metrics are being scraped by Prometheus
test_metrics_scraping() {
    local prometheus_port="$1"

    log_info "Testing metrics scraping..."

    # Query basic metrics
    local basic_queries=("up" "prometheus_build_info" "node_cpu_seconds_total" "container_cpu_usage_seconds_total")

    for query in "${basic_queries[@]}"; do
        local response
        if response=$(prometheus_query "$query" "$prometheus_port"); then
            local status
            status=$(echo "$response" | jq -r '.status // "error"')

            if [[ "$status" == "success" ]]; then
                local result_count
                result_count=$(echo "$response" | jq '.data.result | length')

                if [[ $result_count -gt 0 ]]; then
                    log_success "  ✓ Query '$query' returned $result_count results"
                else
                    log_warning "  ✗ Query '$query' returned no results"
                fi
            else
                log_warning "  ✗ Query '$query' failed with status: $status"
            fi
        else
            log_error "  ✗ Query '$query' failed to execute"
            return 1
        fi
    done

    return 0
}

# Test 2: Verify O-RAN specific metrics
test_oran_metrics() {
    local prometheus_port="$1"

    log_info "Testing O-RAN specific metrics..."

    # Check for O-RAN targets
    local targets_response
    if targets_response=$(curl -s --max-time "$TIMEOUT" "http://localhost:$prometheus_port/api/v1/targets"); then
        local oran_targets
        oran_targets=$(echo "$targets_response" | jq -r '.data.activeTargets[] | select(.labels.job | startswith("oran-")) | .labels.job' | sort -u)

        if [[ -n "$oran_targets" ]]; then
            log_success "Found O-RAN targets:"
            echo "$oran_targets" | while read -r target; do
                log_info "  - $target"
            done

            # Test O-RAN specific queries
            local oran_queries=(
                'up{job=~"oran.*"}'
                'prometheus_tsdb_head_series{job="prometheus"}'
                'rate(prometheus_http_requests_total[5m])'
            )

            for query in "${oran_queries[@]}"; do
                local response
                if response=$(prometheus_query "$query" "$prometheus_port"); then
                    local result_count
                    result_count=$(echo "$response" | jq '.data.result | length')
                    log_success "  ✓ O-RAN query '$query' returned $result_count results"
                else
                    log_warning "  ✗ O-RAN query '$query' failed"
                fi
            done
        else
            log_warning "No O-RAN targets found"
        fi
    else
        log_error "Failed to query Prometheus targets"
        return 1
    fi

    return 0
}

# Test 3: Verify data retention and historical queries
test_data_retention() {
    local prometheus_port="$1"

    log_info "Testing data retention..."

    # Query historical data
    local end_time=$(date +%s)
    local start_time=$((end_time - RETENTION_CHECK_HOURS * 3600))

    local range_query="up"
    local range_url="http://localhost:$prometheus_port/api/v1/query_range?query=$range_query&start=$start_time&end=$end_time&step=60"

    local response
    if response=$(curl -s --max-time "$TIMEOUT" "$range_url"); then
        local status
        status=$(echo "$response" | jq -r '.status // "error"')

        if [[ "$status" == "success" ]]; then
            local series_count
            series_count=$(echo "$response" | jq '.data.result | length')

            if [[ $series_count -gt 0 ]]; then
                local total_datapoints=0
                while IFS= read -r series; do
                    local datapoints
                    datapoints=$(echo "$series" | jq '.values | length')
                    total_datapoints=$((total_datapoints + datapoints))
                done < <(echo "$response" | jq -c '.data.result[]')

                log_success "Historical data: $series_count series with $total_datapoints total datapoints over ${RETENTION_CHECK_HOURS}h"
            else
                log_warning "No historical data found for the last ${RETENTION_CHECK_HOURS} hours"
            fi
        else
            log_error "Historical query failed with status: $status"
            return 1
        fi
    else
        log_error "Failed to execute historical query"
        return 1
    fi

    return 0
}

# Test 4: Verify Grafana can query Prometheus
test_grafana_prometheus_connection() {
    local grafana_port="$1"

    log_info "Testing Grafana to Prometheus connection..."

    # Get data sources
    local datasources_response
    if datasources_response=$(grafana_api_call "/api/datasources" "$grafana_port"); then
        local prometheus_datasource
        prometheus_datasource=$(echo "$datasources_response" | jq -r '.[] | select(.type == "prometheus") | .name' | head -1)

        if [[ -n "$prometheus_datasource" ]] && [[ "$prometheus_datasource" != "null" ]]; then
            log_success "Found Prometheus data source: $prometheus_datasource"

            # Test data source connectivity
            local ds_id
            ds_id=$(echo "$datasources_response" | jq -r '.[] | select(.type == "prometheus") | .id' | head -1)

            local test_response
            if test_response=$(grafana_api_call "/api/datasources/$ds_id/health" "$grafana_port" POST); then
                local test_status
                test_status=$(echo "$test_response" | jq -r '.status // "unknown"')

                if [[ "$test_status" == "success" ]]; then
                    log_success "Grafana can successfully connect to Prometheus"
                else
                    log_error "Grafana data source test failed: $test_status"
                    return 1
                fi
            else
                log_error "Failed to test Grafana data source connectivity"
                return 1
            fi
        else
            log_error "No Prometheus data source found in Grafana"
            return 1
        fi
    else
        log_error "Failed to retrieve Grafana data sources"
        return 1
    fi

    return 0
}

# Test 5: End-to-end query test through Grafana
test_e2e_query_via_grafana() {
    local grafana_port="$1"

    log_info "Testing end-to-end queries via Grafana..."

    # Get Prometheus data source ID
    local datasources_response
    datasources_response=$(grafana_api_call "/api/datasources" "$grafana_port")
    local prometheus_ds_id
    prometheus_ds_id=$(echo "$datasources_response" | jq -r '.[] | select(.type == "prometheus") | .id' | head -1)

    if [[ -z "$prometheus_ds_id" ]] || [[ "$prometheus_ds_id" == "null" ]]; then
        log_error "Cannot find Prometheus data source ID"
        return 1
    fi

    # Test queries through Grafana
    local test_queries=(
        "up"
        "rate(prometheus_http_requests_total[5m])"
        "avg_over_time(up[5m])"
    )

    for query in "${test_queries[@]}"; do
        # Create query payload
        local query_payload
        query_payload=$(jq -n --arg query "$query" --arg ds_id "$prometheus_ds_id" '{
            queries: [{
                datasource: {uid: $ds_id},
                expr: $query,
                refId: "A"
            }],
            from: "now-1h",
            to: "now"
        }')

        local response
        if response=$(grafana_api_call "/api/ds/query" "$grafana_port" POST "$query_payload"); then
            # Parse response
            local results
            results=$(echo "$response" | jq -r '.results.A.frames // []')

            if [[ "$results" != "[]" ]] && [[ "$results" != "null" ]]; then
                log_success "  ✓ E2E query '$query' returned data via Grafana"
            else
                log_warning "  ✗ E2E query '$query' returned no data via Grafana"
            fi
        else
            log_error "  ✗ E2E query '$query' failed via Grafana"
        fi
    done

    return 0
}

# Test 6: Verify alert rules evaluation
test_alert_rules() {
    local prometheus_port="$1"

    log_info "Testing alert rules evaluation..."

    local rules_response
    if rules_response=$(curl -s --max-time "$TIMEOUT" "http://localhost:$prometheus_port/api/v1/rules"); then
        local status
        status=$(echo "$rules_response" | jq -r '.status // "error"')

        if [[ "$status" == "success" ]]; then
            local total_rules=0
            local active_alerts=0

            while IFS= read -r group; do
                if [[ -n "$group" ]]; then
                    local group_rules
                    group_rules=$(echo "$group" | jq '.rules | length')
                    total_rules=$((total_rules + group_rules))

                    # Count active alerts
                    local group_alerts
                    group_alerts=$(echo "$group" | jq '[.rules[] | select(.alerts != null and (.alerts | length) > 0)] | length')
                    active_alerts=$((active_alerts + group_alerts))
                fi
            done < <(echo "$rules_response" | jq -c '.data.groups[]')

            log_success "Found $total_rules alert/recording rules"
            if [[ $active_alerts -gt 0 ]]; then
                log_info "$active_alerts rules have active alerts"
            else
                log_success "No active alerts (system is healthy)"
            fi
        else
            log_error "Failed to retrieve alert rules: $status"
            return 1
        fi
    else
        log_error "Failed to query alert rules"
        return 1
    fi

    return 0
}

# Test 7: Verify metrics cardinality is within limits
test_metrics_cardinality() {
    local prometheus_port="$1"

    log_info "Testing metrics cardinality..."

    # Query total series count
    local series_response
    if series_response=$(prometheus_query "prometheus_tsdb_head_series" "$prometheus_port"); then
        local series_count
        series_count=$(echo "$series_response" | jq -r '.data.result[0].value[1] // "0"')

        log_info "Total time series: $series_count"

        # Check if within reasonable limits (adjust as needed)
        local max_series=100000
        if [[ $(echo "$series_count < $max_series" | bc) -eq 1 ]]; then
            log_success "Series count is within acceptable limits"
        else
            log_warning "High series count detected: $series_count (limit: $max_series)"
        fi

        # Check cardinality by job
        local cardinality_response
        if cardinality_response=$(prometheus_query 'count by (job) (group by (__name__, job) ({__name__!=""}))' "$prometheus_port"); then
            log_info "Cardinality by job:"
            echo "$cardinality_response" | jq -r '.data.result[] | "  \(.metric.job // "unknown"): \(.value[1])"' | sort -k2 -nr
        fi
    else
        log_warning "Could not query series count"
    fi

    return 0
}

# Test 8: Performance validation
test_query_performance() {
    local prometheus_port="$1"

    log_info "Testing query performance..."

    local test_queries=(
        "up"
        "rate(prometheus_http_requests_total[5m])"
        "histogram_quantile(0.95, rate(prometheus_http_request_duration_seconds_bucket[5m]))"
    )

    for query in "${test_queries[@]}"; do
        local start_time=$(date +%s%3N)

        if prometheus_query "$query" "$prometheus_port" > /dev/null; then
            local end_time=$(date +%s%3N)
            local duration=$((end_time - start_time))

            if [[ $duration -lt 1000 ]]; then  # Less than 1 second
                log_success "  ✓ Query '$query' completed in ${duration}ms"
            else
                log_warning "  ⚠ Query '$query' took ${duration}ms (> 1s)"
            fi
        else
            log_error "  ✗ Query '$query' failed"
        fi
    done

    return 0
}

# Main execution function
main() {
    log_info "=== O-RAN End-to-End Metrics Flow Test ==="
    log_info "Namespace: $NAMESPACE"
    log_info "Prometheus Service: $PROMETHEUS_SERVICE"
    log_info "Grafana Service: $GRAFANA_SERVICE"
    log_info "Timeout: ${TIMEOUT}s"
    echo

    # Setup port forwards
    log_info "Setting up port-forwards..."
    local port_info
    port_info=$(setup_port_forwards)
    IFS=':' read -r prometheus_pid prometheus_port grafana_pid grafana_port <<< "$port_info"

    log_success "Port-forwards established (Prometheus: $prometheus_port, Grafana: $grafana_port)"

    # Ensure cleanup on exit
    trap "cleanup_port_forwards $port_info" EXIT

    # Run all tests
    local test_results=()

    # Test 1: Metrics Scraping
    if test_metrics_scraping "$prometheus_port"; then
        test_results+=("metrics_scraping:PASS")
    else
        test_results+=("metrics_scraping:FAIL")
    fi

    # Test 2: O-RAN Metrics
    if test_oran_metrics "$prometheus_port"; then
        test_results+=("oran_metrics:PASS")
    else
        test_results+=("oran_metrics:FAIL")
    fi

    # Test 3: Data Retention
    if test_data_retention "$prometheus_port"; then
        test_results+=("data_retention:PASS")
    else
        test_results+=("data_retention:FAIL")
    fi

    # Test 4: Grafana-Prometheus Connection
    if test_grafana_prometheus_connection "$grafana_port"; then
        test_results+=("grafana_prometheus:PASS")
    else
        test_results+=("grafana_prometheus:FAIL")
    fi

    # Test 5: E2E Queries via Grafana
    if test_e2e_query_via_grafana "$grafana_port"; then
        test_results+=("e2e_grafana:PASS")
    else
        test_results+=("e2e_grafana:FAIL")
    fi

    # Test 6: Alert Rules
    if test_alert_rules "$prometheus_port"; then
        test_results+=("alert_rules:PASS")
    else
        test_results+=("alert_rules:FAIL")
    fi

    # Test 7: Cardinality
    if test_metrics_cardinality "$prometheus_port"; then
        test_results+=("cardinality:PASS")
    else
        test_results+=("cardinality:FAIL")
    fi

    # Test 8: Performance
    if test_query_performance "$prometheus_port"; then
        test_results+=("performance:PASS")
    else
        test_results+=("performance:FAIL")
    fi

    # Summary
    echo
    log_info "=== Test Results Summary ==="
    local passed=0
    local failed=0

    for result in "${test_results[@]}"; do
        local test_name="${result%:*}"
        local test_status="${result#*:}"

        if [[ "$test_status" == "PASS" ]]; then
            log_success "✓ $test_name"
            ((passed++))
        else
            log_error "✗ $test_name"
            ((failed++))
        fi
    done

    echo
    log_info "Passed: $passed, Failed: $failed"

    # Generate summary JSON
    local summary
    summary=$(jq -n --argjson passed "$passed" --argjson failed "$failed" --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)" '{
        timestamp: $timestamp,
        total_tests: ($passed + $failed),
        passed: $passed,
        failed: $failed,
        success_rate: (($passed / ($passed + $failed)) * 100),
        status: (if $failed == 0 then "healthy" else "degraded" end)
    }')

    echo "HEALTH_CHECK_SUMMARY: $summary"

    if [[ $failed -eq 0 ]]; then
        log_success "=== All metrics flow tests passed! ==="
        exit 0
    else
        log_error "=== Some metrics flow tests failed ==="
        exit 1
    fi
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Metrics flow test failed with exit code $exit_code"

    # Cleanup port-forwards if they exist
    if [[ -n "${port_info:-}" ]]; then
        cleanup_port_forwards "$port_info"
    fi

    exit $exit_code
}

trap handle_error ERR

# Help function
show_help() {
    cat << EOF
O-RAN End-to-End Metrics Flow Test

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE       Kubernetes namespace (default: oran-monitoring)
    -p, --prometheus SERVICE       Prometheus service name (default: prometheus)
    -g, --grafana SERVICE          Grafana service name (default: grafana)
    -t, --timeout TIMEOUT         Query timeout in seconds (default: 120)
    -u, --user USER               Grafana username (default: admin)
    -w, --password PASSWORD       Grafana password (default: admin)
    -m, --metric METRIC           Test metric (default: up)
    -r, --retention HOURS         Retention check hours (default: 1)
    -h, --help                    Show this help message

ENVIRONMENT VARIABLES:
    MONITORING_NAMESPACE          Same as --namespace
    PROMETHEUS_SERVICE           Same as --prometheus
    GRAFANA_SERVICE             Same as --grafana
    TIMEOUT                     Same as --timeout
    GRAFANA_USER                Same as --user
    GRAFANA_PASSWORD            Same as --password
    TEST_METRIC                 Same as --metric
    RETENTION_CHECK_HOURS       Same as --retention

EXAMPLES:
    # Basic end-to-end test
    $0

    # Test with custom retention period
    $0 --retention 6

    # Test with custom credentials
    $0 --user myuser --password mypass

EXIT CODES:
    0  - All tests passed
    1  - Some tests failed
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
            -p|--prometheus)
                PROMETHEUS_SERVICE="$2"
                shift 2
                ;;
            -g|--grafana)
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
            -w|--password)
                GRAFANA_PASSWORD="$2"
                shift 2
                ;;
            -m|--metric)
                TEST_METRIC="$2"
                shift 2
                ;;
            -r|--retention)
                RETENTION_CHECK_HOURS="$2"
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

    for cmd in kubectl curl jq lsof bc; do
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