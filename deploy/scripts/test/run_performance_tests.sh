#!/bin/bash
# O-RAN Intent-MANO Performance Test Suite
# Validates thesis performance targets: throughput and latency

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-/test-results}"

# Performance targets (from thesis)
TARGET_THROUGHPUT_EMBB="${TARGET_THROUGHPUT_EMBB:-4.57}"  # Mbps
TARGET_THROUGHPUT_URLLC="${TARGET_THROUGHPUT_URLLC:-2.77}"  # Mbps
TARGET_THROUGHPUT_MMTC="${TARGET_THROUGHPUT_MMTC:-0.93}"   # Mbps
TARGET_RTT_EMBB="${TARGET_RTT_EMBB:-16.1}"                 # ms
TARGET_RTT_URLLC="${TARGET_RTT_URLLC:-15.7}"               # ms
TARGET_RTT_MMTC="${TARGET_RTT_MMTC:-6.3}"                  # ms
MAX_DEPLOYMENT_TIME="${MAX_DEPLOYMENT_TIME:-600}"          # seconds

# Test parameters
IPERF_DURATION="${IPERF_DURATION:-60}"
PING_COUNT="${PING_COUNT:-100}"
LOAD_TEST_DURATION="${LOAD_TEST_DURATION:-300}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"; }
warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}"; }
error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}"; }
info() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*${NC}"; }

# Performance test results
PERFORMANCE_RESULTS=()

# Add performance result
add_performance_result() {
    local test_name="$1"
    local metric_type="$2"  # throughput or latency
    local measured_value="$3"
    local target_value="$4"
    local unit="$5"
    local status="$6"
    local details="$7"

    PERFORMANCE_RESULTS+=("$test_name,$metric_type,$measured_value,$target_value,$unit,$status,$details")

    if [ "$status" = "PASSED" ]; then
        log "Performance test $test_name PASSED: $measured_value $unit (target: $target_value $unit)"
    else
        warn "Performance test $test_name FAILED: $measured_value $unit (target: $target_value $unit) - $details"
    fi
}

# Deploy test workloads
deploy_test_workloads() {
    log "Deploying performance test workloads"

    # Deploy iperf3 servers on different clusters
    for cluster in edge01 edge02 regional; do
        kubectl config use-context "kind-$cluster"

        kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iperf3-server
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: iperf3-server
  template:
    metadata:
      labels:
        app: iperf3-server
    spec:
      containers:
      - name: iperf3
        image: networkstatic/iperf3:latest
        args: ["-s", "-p", "5201"]
        ports:
        - containerPort: 5201
          protocol: TCP
        - containerPort: 5201
          protocol: UDP
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
---
apiVersion: v1
kind: Service
metadata:
  name: iperf3-server
  namespace: default
spec:
  selector:
    app: iperf3-server
  ports:
  - name: tcp
    port: 5201
    targetPort: 5201
    protocol: TCP
  - name: udp
    port: 5201
    targetPort: 5201
    protocol: UDP
  type: ClusterIP
EOF

        kubectl wait --for=condition=Available deployment/iperf3-server --timeout=120s
        info "iperf3 server deployed on cluster: $cluster"
    done

    # Deploy test clients on central cluster
    kubectl config use-context "kind-central"

    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: iperf3-client
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: iperf3-client
  template:
    metadata:
      labels:
        app: iperf3-client
    spec:
      containers:
      - name: iperf3
        image: networkstatic/iperf3:latest
        command: ['sleep', 'infinity']
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
      - name: ping-client
        image: busybox:latest
        command: ['sleep', 'infinity']
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 100m
            memory: 128Mi
EOF

    kubectl wait --for=condition=Available deployment/iperf3-client --timeout=120s
    log "Test clients deployed"
}

# Test throughput between clusters
test_throughput() {
    local service_type="$1"
    local target_cluster="$2"
    local target_throughput="$3"

    log "Testing $service_type throughput to cluster $target_cluster"

    kubectl config use-context "kind-central"

    # Get target cluster service endpoint
    local target_endpoint=""
    case "$target_cluster" in
        "edge01")
            target_endpoint="172.18.0.2"  # Kind network IP
            ;;
        "edge02")
            target_endpoint="172.19.0.2"
            ;;
        "regional")
            target_endpoint="172.20.0.2"
            ;;
    esac

    # Run iperf3 throughput test
    local iperf_output=""
    if iperf_output=$(kubectl exec deployment/iperf3-client -c iperf3 -- \
        iperf3 -c "$target_endpoint" -p 5201 -t "$IPERF_DURATION" -f m -J 2>/dev/null); then

        # Parse throughput result
        local measured_throughput=$(echo "$iperf_output" | \
            jq -r '.end.sum_received.bits_per_second' 2>/dev/null | \
            awk '{printf "%.2f", $1/1000000}')

        if [ -n "$measured_throughput" ] && [ "$measured_throughput" != "null" ]; then
            # Check if throughput meets target
            local status="FAILED"
            local details="Below target"

            if awk "BEGIN {exit !($measured_throughput >= $target_throughput)}"; then
                status="PASSED"
                details="Target achieved"
            fi

            add_performance_result "throughput_${service_type}_${target_cluster}" \
                "throughput" "$measured_throughput" "$target_throughput" "Mbps" "$status" "$details"
        else
            add_performance_result "throughput_${service_type}_${target_cluster}" \
                "throughput" "0" "$target_throughput" "Mbps" "FAILED" "Failed to parse iperf3 output"
        fi
    else
        add_performance_result "throughput_${service_type}_${target_cluster}" \
            "throughput" "0" "$target_throughput" "Mbps" "FAILED" "iperf3 test failed"
    fi

    sleep 5  # Brief pause between tests
}

# Test latency between clusters
test_latency() {
    local service_type="$1"
    local target_cluster="$2"
    local target_rtt="$3"

    log "Testing $service_type latency to cluster $target_cluster"

    kubectl config use-context "kind-central"

    # Get target cluster service endpoint
    local target_endpoint=""
    case "$target_cluster" in
        "edge01")
            target_endpoint="172.18.0.2"
            ;;
        "edge02")
            target_endpoint="172.19.0.2"
            ;;
        "regional")
            target_endpoint="172.20.0.2"
            ;;
    esac

    # Run ping latency test
    local ping_output=""
    if ping_output=$(kubectl exec deployment/iperf3-client -c ping-client -- \
        ping -c "$PING_COUNT" "$target_endpoint" 2>/dev/null); then

        # Parse average RTT
        local measured_rtt=$(echo "$ping_output" | \
            grep "round-trip" | \
            awk -F'/' '{print $5}' | \
            awk '{printf "%.1f", $1}')

        if [ -n "$measured_rtt" ]; then
            # Check if latency meets target
            local status="FAILED"
            local details="Above target"

            if awk "BEGIN {exit !($measured_rtt <= $target_rtt)}"; then
                status="PASSED"
                details="Target achieved"
            fi

            add_performance_result "latency_${service_type}_${target_cluster}" \
                "latency" "$measured_rtt" "$target_rtt" "ms" "$status" "$details"
        else
            add_performance_result "latency_${service_type}_${target_cluster}" \
                "latency" "999" "$target_rtt" "ms" "FAILED" "Failed to parse ping output"
        fi
    else
        add_performance_result "latency_${service_type}_${target_cluster}" \
            "latency" "999" "$target_rtt" "ms" "FAILED" "Ping test failed"
    fi

    sleep 2  # Brief pause between tests
}

# Test E2E deployment time
test_deployment_time() {
    log "Testing E2E deployment time"

    local start_time=$(date +%s)
    local test_slice_name="perf-test-slice-$(date +%s)"

    # Create a test network slice
    kubectl config use-context "kind-central"

    local slice_config=$(cat <<EOF
{
  "name": "$test_slice_name",
  "type": "embb",
  "sla": {
    "throughput": "$TARGET_THROUGHPUT_EMBB",
    "latency": "$TARGET_RTT_EMBB"
  },
  "coverage": {
    "areas": ["edge01", "regional"]
  },
  "vnfs": [
    {
      "type": "ran",
      "location": "edge01"
    },
    {
      "type": "cn",
      "location": "regional"
    }
  ]
}
EOF
)

    # Submit slice creation request
    local creation_response=""
    if creation_response=$(kubectl exec -n oran-mano deployment/oran-orchestrator -- \
        curl -s -X POST -H "Content-Type: application/json" \
        -d "$slice_config" http://localhost:8080/api/v1/slices 2>/dev/null); then

        # Monitor slice deployment progress
        local max_wait=$MAX_DEPLOYMENT_TIME
        local wait_time=0
        local deployment_complete=false

        while [ $wait_time -lt $max_wait ]; do
            # Check if slice is ready
            local slice_status=$(kubectl exec -n oran-mano deployment/oran-orchestrator -- \
                curl -s "http://localhost:8080/api/v1/slices/$test_slice_name/status" 2>/dev/null || echo "")

            if echo "$slice_status" | grep -q "Ready"; then
                deployment_complete=true
                break
            elif echo "$slice_status" | grep -q "Failed"; then
                break
            fi

            sleep 10
            wait_time=$((wait_time + 10))
        done

        local end_time=$(date +%s)
        local measured_time=$((end_time - start_time))

        if [ "$deployment_complete" = true ]; then
            local status="PASSED"
            local details="Deployment completed successfully"

            if [ $measured_time -gt $MAX_DEPLOYMENT_TIME ]; then
                status="FAILED"
                details="Deployment time exceeded target"
            fi

            add_performance_result "e2e_deployment_time" \
                "deployment" "$measured_time" "$MAX_DEPLOYMENT_TIME" "seconds" "$status" "$details"
        else
            add_performance_result "e2e_deployment_time" \
                "deployment" "$measured_time" "$MAX_DEPLOYMENT_TIME" "seconds" "FAILED" "Deployment did not complete"
        fi

        # Cleanup test slice
        kubectl exec -n oran-mano deployment/oran-orchestrator -- \
            curl -s -X DELETE "http://localhost:8080/api/v1/slices/$test_slice_name" 2>/dev/null || true

    else
        local end_time=$(date +%s)
        local measured_time=$((end_time - start_time))
        add_performance_result "e2e_deployment_time" \
            "deployment" "$measured_time" "$MAX_DEPLOYMENT_TIME" "seconds" "FAILED" "Failed to create slice"
    fi
}

# Test TN bandwidth control
test_tn_bandwidth_control() {
    log "Testing Transport Network bandwidth control"

    kubectl config use-context "kind-edge01"

    # Apply traffic shaping policy
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: tn-test-policy
  namespace: oran-edge
data:
  policy.yaml: |
    interfaces:
    - name: "eth0"
      bandwidth: "100Mbps"
      latency: "5ms"
      qos_classes:
      - name: "embb"
        bandwidth: "60Mbps"
        priority: 1
      - name: "urllc"
        bandwidth: "30Mbps"
        priority: 2
      - name: "mmtc"
        bandwidth: "10Mbps"
        priority: 3
EOF

    # Wait for TN agent to apply policy
    sleep 10

    # Check if traffic control rules are applied
    local tc_output=""
    if tc_output=$(kubectl exec -n oran-edge deployment/oran-tn-agent -- \
        tc qdisc show 2>/dev/null); then

        if echo "$tc_output" | grep -q "htb\|netem"; then
            add_performance_result "tn_bandwidth_control" \
                "configuration" "1" "1" "applied" "PASSED" "TC rules applied successfully"
        else
            add_performance_result "tn_bandwidth_control" \
                "configuration" "0" "1" "applied" "FAILED" "TC rules not found"
        fi
    else
        add_performance_result "tn_bandwidth_control" \
            "configuration" "0" "1" "applied" "FAILED" "Unable to check TC rules"
    fi

    # Cleanup
    kubectl delete configmap tn-test-policy -n oran-edge 2>/dev/null || true
}

# Test load handling
test_load_handling() {
    log "Testing system load handling"

    kubectl config use-context "kind-central"

    # Create multiple concurrent slice requests
    local start_time=$(date +%s)
    local num_slices=5
    local pids=()

    for i in $(seq 1 $num_slices); do
        (
            local slice_name="load-test-slice-$i"
            local slice_config=$(cat <<EOF
{
  "name": "$slice_name",
  "type": "mmtc",
  "sla": {
    "throughput": "$TARGET_THROUGHPUT_MMTC",
    "latency": "$TARGET_RTT_MMTC"
  },
  "coverage": {
    "areas": ["edge01"]
  },
  "vnfs": [
    {
      "type": "ran",
      "location": "edge01"
    }
  ]
}
EOF
)

            kubectl exec -n oran-mano deployment/oran-orchestrator -- \
                curl -s -X POST -H "Content-Type: application/json" \
                -d "$slice_config" http://localhost:8080/api/v1/slices >/dev/null 2>&1
        ) &
        pids+=($!)
    done

    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done

    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))

    # Check orchestrator responsiveness
    local health_response=""
    if health_response=$(kubectl exec -n oran-mano deployment/oran-orchestrator -- \
        curl -s "http://localhost:8080/health" 2>/dev/null); then

        if echo "$health_response" | grep -q "ok\|healthy"; then
            add_performance_result "load_handling" \
                "responsiveness" "$total_time" "60" "seconds" "PASSED" "System remained responsive under load"
        else
            add_performance_result "load_handling" \
                "responsiveness" "$total_time" "60" "seconds" "FAILED" "System unresponsive under load"
        fi
    else
        add_performance_result "load_handling" \
            "responsiveness" "$total_time" "60" "seconds" "FAILED" "Health check failed under load"
    fi

    # Cleanup load test slices
    for i in $(seq 1 $num_slices); do
        kubectl exec -n oran-mano deployment/oran-orchestrator -- \
            curl -s -X DELETE "http://localhost:8080/api/v1/slices/load-test-slice-$i" 2>/dev/null || true
    done
}

# Generate performance report
generate_performance_report() {
    log "Generating performance test report"

    local report_file="$TEST_RESULTS_DIR/performance-test-report.json"
    local html_report="$TEST_RESULTS_DIR/performance-test-report.html"

    # Calculate statistics
    local total_tests=${#PERFORMANCE_RESULTS[@]}
    local passed_tests=0
    local failed_tests=0

    for result in "${PERFORMANCE_RESULTS[@]}"; do
        IFS=',' read -r name metric_type measured target unit status details <<< "$result"
        if [ "$status" = "PASSED" ]; then
            ((passed_tests++))
        else
            ((failed_tests++))
        fi
    done

    # Create JSON report
    cat > "$report_file" <<EOF
{
  "test_suite": "O-RAN Intent-MANO Performance Tests",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "summary": {
    "total_tests": $total_tests,
    "passed": $passed_tests,
    "failed": $failed_tests,
    "success_rate": $(( passed_tests * 100 / total_tests ))
  },
  "targets": {
    "throughput": {
      "embb": $TARGET_THROUGHPUT_EMBB,
      "urllc": $TARGET_THROUGHPUT_URLLC,
      "mmtc": $TARGET_THROUGHPUT_MMTC
    },
    "latency": {
      "embb": $TARGET_RTT_EMBB,
      "urllc": $TARGET_RTT_URLLC,
      "mmtc": $TARGET_RTT_MMTC
    },
    "deployment_time": $MAX_DEPLOYMENT_TIME
  },
  "test_parameters": {
    "iperf_duration": $IPERF_DURATION,
    "ping_count": $PING_COUNT,
    "load_test_duration": $LOAD_TEST_DURATION
  },
  "results": [
EOF

    # Add test results
    local first=true
    for result in "${PERFORMANCE_RESULTS[@]}"; do
        IFS=',' read -r name metric_type measured target unit status details <<< "$result"

        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$report_file"
        fi

        cat >> "$report_file" <<EOF
    {
      "test_name": "$name",
      "metric_type": "$metric_type",
      "measured_value": $measured,
      "target_value": $target,
      "unit": "$unit",
      "status": "$status",
      "details": "$details"
    }
EOF
    done

    cat >> "$report_file" <<EOF
  ]
}
EOF

    log "Performance report generated: $report_file"
}

# Cleanup test environment
cleanup_test_environment() {
    log "Cleaning up test environment"

    for cluster in edge01 edge02 regional central; do
        kubectl config use-context "kind-$cluster"
        kubectl delete deployment iperf3-server iperf3-client --ignore-not-found=true --wait=true
        kubectl delete service iperf3-server --ignore-not-found=true
    done

    log "Test environment cleaned up"
}

# Main execution
main() {
    log "Starting O-RAN Intent-MANO Performance Tests"

    # Create results directory
    mkdir -p "$TEST_RESULTS_DIR"

    # Deploy test infrastructure
    deploy_test_workloads

    # Run performance tests
    info "Running throughput tests..."
    test_throughput "embb" "edge01" "$TARGET_THROUGHPUT_EMBB"
    test_throughput "urllc" "edge02" "$TARGET_THROUGHPUT_URLLC"
    test_throughput "mmtc" "regional" "$TARGET_THROUGHPUT_MMTC"

    info "Running latency tests..."
    test_latency "embb" "edge01" "$TARGET_RTT_EMBB"
    test_latency "urllc" "edge02" "$TARGET_RTT_URLLC"
    test_latency "mmtc" "regional" "$TARGET_RTT_MMTC"

    info "Running deployment time test..."
    test_deployment_time

    info "Running TN bandwidth control test..."
    test_tn_bandwidth_control

    info "Running load handling test..."
    test_load_handling

    # Generate reports
    generate_performance_report

    # Cleanup
    cleanup_test_environment

    # Summary
    local total_tests=${#PERFORMANCE_RESULTS[@]}
    local passed_tests=0
    local failed_tests=0

    for result in "${PERFORMANCE_RESULTS[@]}"; do
        if echo "$result" | grep -q ",PASSED,"; then
            ((passed_tests++))
        else
            ((failed_tests++))
        fi
    done

    log "Performance test suite completed"
    info "Total tests: $total_tests"
    info "Passed: $passed_tests"
    info "Failed: $failed_tests"
    info "Success rate: $(( passed_tests * 100 / total_tests ))%"

    if [ $failed_tests -eq 0 ]; then
        log "All performance tests PASSED! System meets thesis targets."
        exit 0
    else
        warn "Some performance tests FAILED. Review results for details."
        exit 1
    fi
}

# Execute main function
main "$@"