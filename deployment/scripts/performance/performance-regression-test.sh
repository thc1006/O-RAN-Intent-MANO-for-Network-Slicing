#!/bin/bash

# Performance Regression Detection for O-RAN MANO Monitoring Stack
# This script detects performance regressions in the monitoring infrastructure

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-monitoring}"
BASELINE_DIR="${BASELINE_DIR:-./performance-baselines}"
RESULTS_DIR="${RESULTS_DIR:-./performance-results}"
TEST_DURATION="${TEST_DURATION:-300}"  # 5 minutes
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Performance thresholds (configurable)
MAX_RESPONSE_TIME_MS=2000
MAX_CPU_USAGE_PERCENT=80
MAX_MEMORY_USAGE_PERCENT=80
MAX_QUERY_DURATION_MS=5000
MIN_SCRAPE_SUCCESS_RATE=95

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites for performance testing..."

    local missing_tools=()

    command -v kubectl &> /dev/null || missing_tools+=("kubectl")
    command -v curl &> /dev/null || missing_tools+=("curl")
    command -v jq &> /dev/null || missing_tools+=("jq")
    command -v bc &> /dev/null || missing_tools+=("bc")

    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        exit 1
    fi

    # Create directories
    mkdir -p "$BASELINE_DIR" "$RESULTS_DIR"

    success "Prerequisites check passed"
}

# Setup port forwarding
setup_port_forwarding() {
    log "Setting up port forwarding..."

    # Kill existing port forwards
    pkill -f "kubectl port-forward" 2>/dev/null || true
    sleep 2

    # Start Prometheus port forward
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus 9090:9090 &
    local prom_pid=$!

    # Start Grafana port forward
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-grafana 3000:80 &
    local grafana_pid=$!

    # Wait for services to be ready
    sleep 10

    # Test connectivity
    if ! curl -f "$PROMETHEUS_URL/-/healthy" &>/dev/null; then
        error "Cannot connect to Prometheus"
        kill $prom_pid $grafana_pid 2>/dev/null || true
        return 1
    fi

    if ! curl -f "$GRAFANA_URL/api/health" &>/dev/null; then
        error "Cannot connect to Grafana"
        kill $prom_pid $grafana_pid 2>/dev/null || true
        return 1
    fi

    success "Port forwarding established"
    echo "$prom_pid $grafana_pid" > /tmp/perf_test_pids
}

# Cleanup port forwarding
cleanup_port_forwarding() {
    if [ -f /tmp/perf_test_pids ]; then
        local pids
        pids=$(cat /tmp/perf_test_pids)
        for pid in $pids; do
            kill "$pid" 2>/dev/null || true
        done
        rm -f /tmp/perf_test_pids
    fi
}

# Test Prometheus performance
test_prometheus_performance() {
    log "Testing Prometheus performance..."

    local results_file="$RESULTS_DIR/prometheus_performance_$(date +%Y%m%d_%H%M%S).json"

    # Initialize results
    cat > "$results_file" << 'EOF'
{
  "timestamp": "",
  "prometheus": {
    "response_time": {},
    "query_performance": {},
    "resource_usage": {},
    "scrape_performance": {}
  }
}
EOF

    # Update timestamp
    local timestamp
    timestamp=$(date -Iseconds)
    jq --arg ts "$timestamp" '.timestamp = $ts' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Test basic response time
    log "Testing Prometheus response time..."
    local response_time
    response_time=$(curl -w "%{time_total}" -s -o /dev/null "$PROMETHEUS_URL/-/healthy" || echo "999")
    response_time=$(echo "$response_time * 1000" | bc -l | cut -d. -f1)

    jq --argjson rt "$response_time" '.prometheus.response_time.health_check_ms = $rt' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Test query performance
    log "Testing Prometheus query performance..."
    local test_queries=(
        "up"
        "prometheus_config_last_reload_successful"
        "rate(prometheus_tsdb_head_samples_appended_total[5m])"
        "histogram_quantile(0.95, rate(prometheus_tsdb_query_duration_seconds_bucket[5m]))"
    )

    for query in "${test_queries[@]}"; do
        local start_time
        start_time=$(date +%s%N)

        local query_response
        query_response=$(curl -s "$PROMETHEUS_URL/api/v1/query" --data-urlencode "query=$query" || echo '{"status":"error"}')

        local end_time
        end_time=$(date +%s%N)

        local duration_ms
        duration_ms=$(( (end_time - start_time) / 1000000 ))

        local query_key
        query_key=$(echo "$query" | tr ' ()[]{}/.' '_' | tr -d '"')

        if echo "$query_response" | jq -e '.status == "success"' &>/dev/null; then
            jq --arg key "$query_key" --argjson duration "$duration_ms" '.prometheus.query_performance[$key] = {duration_ms: $duration, status: "success"}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"
        else
            jq --arg key "$query_key" --argjson duration "$duration_ms" '.prometheus.query_performance[$key] = {duration_ms: $duration, status: "error"}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"
        fi
    done

    # Test resource usage
    log "Testing Prometheus resource usage..."
    local cpu_query='rate(container_cpu_usage_seconds_total{pod=~"prometheus-.*"}[5m]) * 100'
    local memory_query='container_memory_working_set_bytes{pod=~"prometheus-.*"}'

    local cpu_response
    cpu_response=$(curl -s "$PROMETHEUS_URL/api/v1/query" --data-urlencode "query=$cpu_query" || echo '{"data":{"result":[]}}')

    local memory_response
    memory_response=$(curl -s "$PROMETHEUS_URL/api/v1/query" --data-urlencode "query=$memory_query" || echo '{"data":{"result":[]}}')

    # Extract resource values
    local cpu_usage
    cpu_usage=$(echo "$cpu_response" | jq -r '.data.result[0].value[1] // "0"' 2>/dev/null | head -1)

    local memory_usage
    memory_usage=$(echo "$memory_response" | jq -r '.data.result[0].value[1] // "0"' 2>/dev/null | head -1)

    jq --argjson cpu "$cpu_usage" --argjson mem "$memory_usage" '.prometheus.resource_usage = {cpu_percent: $cpu, memory_bytes: $mem}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Test scrape performance
    log "Testing Prometheus scrape performance..."
    local targets_response
    targets_response=$(curl -s "$PROMETHEUS_URL/api/v1/targets" || echo '{"data":{"activeTargets":[]}}')

    local total_targets
    total_targets=$(echo "$targets_response" | jq '.data.activeTargets | length' 2>/dev/null || echo "0")

    local healthy_targets
    healthy_targets=$(echo "$targets_response" | jq '[.data.activeTargets[] | select(.health == "up")] | length' 2>/dev/null || echo "0")

    local success_rate
    if [ "$total_targets" -gt 0 ]; then
        success_rate=$(echo "scale=2; $healthy_targets * 100 / $total_targets" | bc)
    else
        success_rate="0"
    fi

    jq --argjson total "$total_targets" --argjson healthy "$healthy_targets" --argjson rate "$success_rate" '.prometheus.scrape_performance = {total_targets: $total, healthy_targets: $healthy, success_rate_percent: $rate}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    success "Prometheus performance test completed: $results_file"
    echo "$results_file"
}

# Test Grafana performance
test_grafana_performance() {
    log "Testing Grafana performance..."

    local results_file="$RESULTS_DIR/grafana_performance_$(date +%Y%m%d_%H%M%S).json"

    # Initialize results
    cat > "$results_file" << 'EOF'
{
  "timestamp": "",
  "grafana": {
    "response_time": {},
    "api_performance": {},
    "dashboard_performance": {}
  }
}
EOF

    # Update timestamp
    local timestamp
    timestamp=$(date -Iseconds)
    jq --arg ts "$timestamp" '.timestamp = $ts' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Test basic response time
    log "Testing Grafana response time..."
    local response_time
    response_time=$(curl -w "%{time_total}" -s -o /dev/null "$GRAFANA_URL/api/health" || echo "999")
    response_time=$(echo "$response_time * 1000" | bc -l | cut -d. -f1)

    jq --argjson rt "$response_time" '.grafana.response_time.health_check_ms = $rt' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Test API endpoints
    log "Testing Grafana API performance..."
    local api_endpoints=(
        "/api/health"
        "/api/datasources"
        "/api/search"
        "/api/org"
    )

    for endpoint in "${api_endpoints[@]}"; do
        local start_time
        start_time=$(date +%s%N)

        local api_response
        api_response=$(curl -s "$GRAFANA_URL$endpoint" || echo "error")

        local end_time
        end_time=$(date +%s%N)

        local duration_ms
        duration_ms=$(( (end_time - start_time) / 1000000 ))

        local endpoint_key
        endpoint_key=$(echo "$endpoint" | tr '/' '_' | sed 's/^_//')

        if [ "$api_response" != "error" ]; then
            jq --arg key "$endpoint_key" --argjson duration "$duration_ms" '.grafana.api_performance[$key] = {duration_ms: $duration, status: "success"}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"
        else
            jq --arg key "$endpoint_key" --argjson duration "$duration_ms" '.grafana.api_performance[$key] = {duration_ms: $duration, status: "error"}' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"
        fi
    done

    success "Grafana performance test completed: $results_file"
    echo "$results_file"
}

# Load test monitoring stack
load_test_monitoring() {
    log "Running load test on monitoring stack..."

    local results_file="$RESULTS_DIR/load_test_$(date +%Y%m%d_%H%M%S).json"

    # Initialize results
    cat > "$results_file" << 'EOF'
{
  "timestamp": "",
  "load_test": {
    "duration_seconds": 0,
    "concurrent_requests": 0,
    "total_requests": 0,
    "successful_requests": 0,
    "failed_requests": 0,
    "average_response_time_ms": 0,
    "max_response_time_ms": 0,
    "requests_per_second": 0
  }
}
EOF

    local timestamp
    timestamp=$(date -Iseconds)
    jq --arg ts "$timestamp" '.timestamp = $ts' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    # Create load test script
    cat > /tmp/load_test.sh << 'EOF'
#!/bin/bash

PROMETHEUS_URL="$1"
GRAFANA_URL="$2"
DURATION="$3"
CONCURRENT="$4"

total_requests=0
successful_requests=0
failed_requests=0
total_response_time=0
max_response_time=0

start_time=$(date +%s)
end_time=$((start_time + DURATION))

# Array of URLs to test
urls=(
    "$PROMETHEUS_URL/-/healthy"
    "$PROMETHEUS_URL/api/v1/query?query=up"
    "$GRAFANA_URL/api/health"
    "$GRAFANA_URL/api/datasources"
)

# Function to make request
make_request() {
    local url="$1"
    local response_time
    response_time=$(curl -w "%{time_total}" -s -o /dev/null "$url" 2>/dev/null || echo "999")
    response_time_ms=$(echo "$response_time * 1000" | bc -l | cut -d. -f1)
    echo "$response_time_ms"
}

# Run concurrent requests
while [ $(date +%s) -lt $end_time ]; do
    for i in $(seq 1 "$CONCURRENT"); do
        (
            for url in "${urls[@]}"; do
                response_time=$(make_request "$url")
                echo "$response_time"
            done
        ) &
    done
    wait
    sleep 1
done
EOF

    chmod +x /tmp/load_test.sh

    log "Starting load test (duration: ${TEST_DURATION}s, concurrent: 5)..."

    # Run load test and capture results
    local load_results
    load_results=$(/tmp/load_test.sh "$PROMETHEUS_URL" "$GRAFANA_URL" "$TEST_DURATION" 5 2>/dev/null)

    # Process results
    local total_requests=0
    local successful_requests=0
    local failed_requests=0
    local total_response_time=0
    local max_response_time=0

    while read -r response_time; do
        if [ -n "$response_time" ] && [ "$response_time" != "999" ]; then
            ((total_requests++))
            if [ "$response_time" -lt "$MAX_RESPONSE_TIME_MS" ]; then
                ((successful_requests++))
            else
                ((failed_requests++))
            fi
            total_response_time=$((total_response_time + response_time))
            if [ "$response_time" -gt "$max_response_time" ]; then
                max_response_time="$response_time"
            fi
        else
            ((failed_requests++))
        fi
    done <<< "$load_results"

    # Calculate averages
    local average_response_time=0
    local requests_per_second=0

    if [ "$total_requests" -gt 0 ]; then
        average_response_time=$((total_response_time / total_requests))
        requests_per_second=$(echo "scale=2; $total_requests / $TEST_DURATION" | bc)
    fi

    # Update results file
    jq --argjson duration "$TEST_DURATION" \
       --argjson concurrent 5 \
       --argjson total "$total_requests" \
       --argjson successful "$successful_requests" \
       --argjson failed "$failed_requests" \
       --argjson avg "$average_response_time" \
       --argjson max "$max_response_time" \
       --argjson rps "$requests_per_second" \
       '.load_test = {
         duration_seconds: $duration,
         concurrent_requests: $concurrent,
         total_requests: $total,
         successful_requests: $successful,
         failed_requests: $failed,
         average_response_time_ms: $avg,
         max_response_time_ms: $max,
         requests_per_second: $rps
       }' "$results_file" > /tmp/results.json && mv /tmp/results.json "$results_file"

    rm -f /tmp/load_test.sh

    success "Load test completed: $results_file"
    echo "$results_file"
}

# Compare with baseline
compare_with_baseline() {
    local current_results="$1"
    local baseline_file="$BASELINE_DIR/baseline.json"

    log "Comparing with baseline performance..."

    if [ ! -f "$baseline_file" ]; then
        warning "No baseline file found. Creating baseline from current results..."
        cp "$current_results" "$baseline_file"
        return 0
    fi

    local comparison_file="$RESULTS_DIR/comparison_$(date +%Y%m%d_%H%M%S).json"

    # Compare Prometheus performance
    local current_prom_response
    current_prom_response=$(jq -r '.prometheus.response_time.health_check_ms // 0' "$current_results")

    local baseline_prom_response
    baseline_prom_response=$(jq -r '.prometheus.response_time.health_check_ms // 0' "$baseline_file")

    local prom_regression=false
    if [ "$current_prom_response" -gt "$((baseline_prom_response * 150 / 100))" ]; then
        prom_regression=true
        error "Prometheus response time regression detected: ${current_prom_response}ms vs baseline ${baseline_prom_response}ms"
    fi

    # Compare scrape success rate
    local current_success_rate
    current_success_rate=$(jq -r '.prometheus.scrape_performance.success_rate_percent // 0' "$current_results")

    local baseline_success_rate
    baseline_success_rate=$(jq -r '.prometheus.scrape_performance.success_rate_percent // 0' "$baseline_file")

    local scrape_regression=false
    if (( $(echo "$current_success_rate < $baseline_success_rate - 5" | bc -l) )); then
        scrape_regression=true
        error "Prometheus scrape success rate regression detected: ${current_success_rate}% vs baseline ${baseline_success_rate}%"
    fi

    # Create comparison report
    cat > "$comparison_file" << EOF
{
  "timestamp": "$(date -Iseconds)",
  "regression_detected": $(if [ "$prom_regression" = true ] || [ "$scrape_regression" = true ]; then echo "true"; else echo "false"; fi),
  "prometheus": {
    "response_time": {
      "current_ms": $current_prom_response,
      "baseline_ms": $baseline_prom_response,
      "regression": $prom_regression
    },
    "scrape_success_rate": {
      "current_percent": $current_success_rate,
      "baseline_percent": $baseline_success_rate,
      "regression": $scrape_regression
    }
  }
}
EOF

    if [ "$prom_regression" = true ] || [ "$scrape_regression" = true ]; then
        error "Performance regression detected! See comparison: $comparison_file"
        return 1
    else
        success "No performance regression detected"
        return 0
    fi
}

# Generate performance report
generate_performance_report() {
    local prometheus_results="$1"
    local grafana_results="$2"
    local load_test_results="$3"

    log "Generating performance report..."

    local report_file="$RESULTS_DIR/performance_report_$(date +%Y%m%d_%H%M%S).md"

    cat > "$report_file" << EOF
# O-RAN MANO Monitoring Performance Report

**Generated**: $(date)
**Test Duration**: ${TEST_DURATION} seconds

## Summary

$(if compare_with_baseline "$prometheus_results" &>/dev/null; then echo "✅ **No performance regressions detected**"; else echo "❌ **Performance regressions detected**"; fi)

## Prometheus Performance

- **Health Check Response Time**: $(jq -r '.prometheus.response_time.health_check_ms' "$prometheus_results")ms
- **Scrape Success Rate**: $(jq -r '.prometheus.scrape_performance.success_rate_percent' "$prometheus_results")%
- **Total Targets**: $(jq -r '.prometheus.scrape_performance.total_targets' "$prometheus_results")
- **Healthy Targets**: $(jq -r '.prometheus.scrape_performance.healthy_targets' "$prometheus_results")

### Query Performance
EOF

    # Add query performance details
    jq -r '.prometheus.query_performance | to_entries[] | "- **\(.key)**: \(.value.duration_ms)ms (\(.value.status))"' "$prometheus_results" >> "$report_file"

    cat >> "$report_file" << EOF

## Grafana Performance

- **Health Check Response Time**: $(jq -r '.grafana.response_time.health_check_ms' "$grafana_results")ms

### API Performance
EOF

    # Add API performance details
    jq -r '.grafana.api_performance | to_entries[] | "- **\(.key)**: \(.value.duration_ms)ms (\(.value.status))"' "$grafana_results" >> "$report_file"

    cat >> "$report_file" << EOF

## Load Test Results

- **Total Requests**: $(jq -r '.load_test.total_requests' "$load_test_results")
- **Successful Requests**: $(jq -r '.load_test.successful_requests' "$load_test_results")
- **Failed Requests**: $(jq -r '.load_test.failed_requests' "$load_test_results")
- **Average Response Time**: $(jq -r '.load_test.average_response_time_ms' "$load_test_results")ms
- **Max Response Time**: $(jq -r '.load_test.max_response_time_ms' "$load_test_results")ms
- **Requests Per Second**: $(jq -r '.load_test.requests_per_second' "$load_test_results")

## Thresholds

- Max Response Time: ${MAX_RESPONSE_TIME_MS}ms
- Max CPU Usage: ${MAX_CPU_USAGE_PERCENT}%
- Max Memory Usage: ${MAX_MEMORY_USAGE_PERCENT}%
- Min Scrape Success Rate: ${MIN_SCRAPE_SUCCESS_RATE}%

## Files Generated

- Prometheus Results: $prometheus_results
- Grafana Results: $grafana_results
- Load Test Results: $load_test_results
- Performance Report: $report_file
EOF

    success "Performance report generated: $report_file"
}

# Main performance test function
main() {
    local action="${1:-full}"

    log "Starting O-RAN MANO monitoring performance regression test"
    log "Action: $action, Duration: ${TEST_DURATION}s"

    # Set up error handling
    trap cleanup_port_forwarding EXIT

    check_prerequisites

    case "$action" in
        "full")
            setup_port_forwarding

            local prometheus_results
            prometheus_results=$(test_prometheus_performance)

            local grafana_results
            grafana_results=$(test_grafana_performance)

            local load_test_results
            load_test_results=$(load_test_monitoring)

            # Compare with baseline
            if ! compare_with_baseline "$prometheus_results"; then
                error "Performance regression detected!"
                exit 1
            fi

            generate_performance_report "$prometheus_results" "$grafana_results" "$load_test_results"
            ;;
        "prometheus")
            setup_port_forwarding
            test_prometheus_performance
            ;;
        "grafana")
            setup_port_forwarding
            test_grafana_performance
            ;;
        "load")
            setup_port_forwarding
            load_test_monitoring
            ;;
        "baseline")
            setup_port_forwarding
            local results
            results=$(test_prometheus_performance)
            cp "$results" "$BASELINE_DIR/baseline.json"
            success "Baseline created: $BASELINE_DIR/baseline.json"
            ;;
        *)
            error "Unknown action: $action"
            echo "Usage: $0 [full|prometheus|grafana|load|baseline]"
            exit 1
            ;;
    esac

    success "Performance regression test completed!"
}

# Execute main function
main "${1:-full}"