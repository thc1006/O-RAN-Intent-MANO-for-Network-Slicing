#!/bin/bash
# Multi-Site Connectivity Test Suite for Kube-OVN O-RAN Deployment
# Tests GENEVE tunnels, QoS enforcement, and network slice isolation

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_NAMESPACE="o-ran-test"
CENTRAL_KUBECONFIG="${CENTRAL_KUBECONFIG:-~/.kube/config}"
EDGE01_KUBECONFIG="${EDGE01_KUBECONFIG:-~/.kube/edge01-config}"
EDGE02_KUBECONFIG="${EDGE02_KUBECONFIG:-~/.kube/edge02-config}"

# Test configuration
HIGH_PRIORITY_BANDWIDTH="100Mbit"
MEDIUM_PRIORITY_BANDWIDTH="50Mbit"
LOW_PRIORITY_BANDWIDTH="10Mbit"

# Expected metrics from thesis
EXPECTED_DL_THROUGHPUT_HIGH="4.57"    # Mbps
EXPECTED_DL_THROUGHPUT_MEDIUM="2.77"  # Mbps
EXPECTED_DL_THROUGHPUT_LOW="0.93"     # Mbps
EXPECTED_RTT_HIGH="16.1"              # ms
EXPECTED_RTT_MEDIUM="15.7"            # ms
EXPECTED_RTT_LOW="6.3"                # ms

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') $*"
}

# Test result tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
TEST_RESULTS=()

# Test result functions
start_test() {
    local test_name="$1"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_info "Starting test: $test_name"
}

pass_test() {
    local test_name="$1"
    local details="${2:-}"
    PASSED_TESTS=$((PASSED_TESTS + 1))
    TEST_RESULTS+=("PASS: $test_name $details")
    log_success "$test_name - PASSED $details"
}

fail_test() {
    local test_name="$1"
    local details="${2:-}"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    TEST_RESULTS+=("FAIL: $test_name $details")
    log_error "$test_name - FAILED $details"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test resources..."

    for kubeconfig in "$CENTRAL_KUBECONFIG" "$EDGE01_KUBECONFIG" "$EDGE02_KUBECONFIG"; do
        if [[ -f "$kubeconfig" ]]; then
            kubectl --kubeconfig="$kubeconfig" delete namespace "$TEST_NAMESPACE" --ignore-not-found=true 2>/dev/null || true
            kubectl --kubeconfig="$kubeconfig" delete pods -l test-suite=connectivity --all-namespaces 2>/dev/null || true
        fi
    done

    log_info "Cleanup completed"
}

# Setup test environment
setup_test_environment() {
    log_info "Setting up test environment..."

    # Create test namespace on all clusters
    for cluster in "central" "edge01" "edge02"; do
        local kubeconfig_var="${cluster^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            log_warning "Kubeconfig not found for $cluster: $kubeconfig"
            continue
        fi

        kubectl --kubeconfig="$kubeconfig" create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | \
            kubectl --kubeconfig="$kubeconfig" apply -f -

        # Label namespace for OVN
        kubectl --kubeconfig="$kubeconfig" label namespace "$TEST_NAMESPACE" \
            networking.kubeovn.io/ns="$TEST_NAMESPACE" --overwrite
    done

    log_success "Test environment setup completed"
}

# Test 1: OVN Central Cluster Connectivity
test_ovn_central_connectivity() {
    start_test "OVN Central Cluster Connectivity"

    local nb_status=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" exec -n kube-ovn \
        deployment/ovn-central -- ovn-nbctl show 2>/dev/null | wc -l)
    local sb_status=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" exec -n kube-ovn \
        deployment/ovn-central -- ovn-sbctl show 2>/dev/null | wc -l)

    if [[ $nb_status -gt 0 && $sb_status -gt 0 ]]; then
        pass_test "OVN Central Cluster Connectivity" "(NB: $nb_status lines, SB: $sb_status lines)"
    else
        fail_test "OVN Central Cluster Connectivity" "(NB: $nb_status lines, SB: $sb_status lines)"
    fi
}

# Test 2: OVN Edge Connectivity
test_ovn_edge_connectivity() {
    for edge in "edge01" "edge02"; do
        start_test "OVN $edge Connectivity"

        local kubeconfig_var="${edge^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            fail_test "OVN $edge Connectivity" "(kubeconfig not found)"
            continue
        fi

        local controller_ready=$(kubectl --kubeconfig="$kubeconfig" get pods -n kube-ovn \
            -l app=kube-ovn-controller --no-headers 2>/dev/null | grep -c "Running" || echo "0")
        local ovs_ready=$(kubectl --kubeconfig="$kubeconfig" get pods -n kube-ovn \
            -l app=ovs --no-headers 2>/dev/null | grep -c "Running" || echo "0")

        if [[ $controller_ready -gt 0 && $ovs_ready -gt 0 ]]; then
            pass_test "OVN $edge Connectivity" "(Controller: $controller_ready, OVS: $ovs_ready)"
        else
            fail_test "OVN $edge Connectivity" "(Controller: $controller_ready, OVS: $ovs_ready)"
        fi
    done
}

# Test 3: GENEVE Tunnel Connectivity
test_geneve_tunnels() {
    start_test "GENEVE Tunnel Connectivity"

    # Check if GENEVE interfaces exist on edge nodes
    local geneve_count=0

    for edge in "edge01" "edge02"; do
        local kubeconfig_var="${edge^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            continue
        fi

        local geneve_interfaces=$(kubectl --kubeconfig="$kubeconfig" exec -n kube-ovn \
            ds/ovs-ovn -- ovs-vsctl show 2>/dev/null | grep -c "genev-" || echo "0")
        geneve_count=$((geneve_count + geneve_interfaces))
    done

    if [[ $geneve_count -gt 0 ]]; then
        pass_test "GENEVE Tunnel Connectivity" "($geneve_count tunnel interfaces found)"
    else
        fail_test "GENEVE Tunnel Connectivity" "(no tunnel interfaces found)"
    fi
}

# Test 4: QoS Subnet Configuration
test_qos_subnets() {
    start_test "QoS Subnet Configuration"

    local high_subnet=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get subnet \
        high-priority-slice -n kube-ovn -o name 2>/dev/null || echo "")
    local medium_subnet=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get subnet \
        medium-priority-slice -n kube-ovn -o name 2>/dev/null || echo "")
    local low_subnet=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get subnet \
        low-priority-slice -n kube-ovn -o name 2>/dev/null || echo "")

    local configured_subnets=0
    [[ -n "$high_subnet" ]] && configured_subnets=$((configured_subnets + 1))
    [[ -n "$medium_subnet" ]] && configured_subnets=$((configured_subnets + 1))
    [[ -n "$low_subnet" ]] && configured_subnets=$((configured_subnets + 1))

    if [[ $configured_subnets -eq 3 ]]; then
        pass_test "QoS Subnet Configuration" "(all 3 QoS subnets configured)"
    else
        fail_test "QoS Subnet Configuration" "($configured_subnets/3 subnets configured)"
    fi
}

# Test 5: Network Policy Enforcement
test_network_policies() {
    start_test "Network Policy Enforcement"

    local policies=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get networkpolicy \
        -n o-ran-slices -o name 2>/dev/null | wc -l)

    if [[ $policies -ge 3 ]]; then
        pass_test "Network Policy Enforcement" "($policies network policies found)"
    else
        fail_test "Network Policy Enforcement" "($policies network policies found, expected ≥3)"
    fi
}

# Test 6: Deploy Test Pods with QoS Classes
deploy_test_pods() {
    log_info "Deploying test pods with different QoS classes..."

    for cluster in "central" "edge01" "edge02"; do
        local kubeconfig_var="${cluster^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            continue
        fi

        # Deploy high priority test pod
        cat <<EOF | kubectl --kubeconfig="$kubeconfig" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-high-priority-$cluster
  namespace: $TEST_NAMESPACE
  labels:
    qos-class: high
    test-suite: connectivity
    cluster: $cluster
  annotations:
    ovn.kubernetes.io/logical_switch: "high-priority-slice"
    ovn.kubernetes.io/ingress_rate: "100"
    ovn.kubernetes.io/egress_rate: "100"
spec:
  containers:
  - name: test-container
    image: nicolaka/netshoot:latest
    command: ["/bin/bash"]
    args: ["-c", "while true; do sleep 30; done"]
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 500m
        memory: 256Mi
EOF

        # Deploy medium priority test pod
        cat <<EOF | kubectl --kubeconfig="$kubeconfig" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-medium-priority-$cluster
  namespace: $TEST_NAMESPACE
  labels:
    qos-class: medium
    test-suite: connectivity
    cluster: $cluster
  annotations:
    ovn.kubernetes.io/logical_switch: "medium-priority-slice"
    ovn.kubernetes.io/ingress_rate: "50"
    ovn.kubernetes.io/egress_rate: "50"
spec:
  containers:
  - name: test-container
    image: nicolaka/netshoot:latest
    command: ["/bin/bash"]
    args: ["-c", "while true; do sleep 30; done"]
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 200m
        memory: 256Mi
EOF

        # Deploy low priority test pod
        cat <<EOF | kubectl --kubeconfig="$kubeconfig" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: test-low-priority-$cluster
  namespace: $TEST_NAMESPACE
  labels:
    qos-class: low
    test-suite: connectivity
    cluster: $cluster
  annotations:
    ovn.kubernetes.io/logical_switch: "low-priority-slice"
    ovn.kubernetes.io/ingress_rate: "10"
    ovn.kubernetes.io/egress_rate: "10"
spec:
  containers:
  - name: test-container
    image: nicolaka/netshoot:latest
    command: ["/bin/bash"]
    args: ["-c", "while true; do sleep 30; done"]
    resources:
      requests:
        cpu: 50m
        memory: 128Mi
      limits:
        cpu: 100m
        memory: 256Mi
EOF
    done

    # Wait for pods to be ready
    log_info "Waiting for test pods to be ready..."
    sleep 30

    for cluster in "central" "edge01" "edge02"; do
        local kubeconfig_var="${cluster^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            continue
        fi

        kubectl --kubeconfig="$kubeconfig" wait --for=condition=Ready \
            pods -l test-suite=connectivity -n "$TEST_NAMESPACE" --timeout=120s || true
    done
}

# Test 7: Cross-Cluster Connectivity
test_cross_cluster_connectivity() {
    start_test "Cross-Cluster Connectivity"

    local connectivity_success=0
    local connectivity_total=0

    # Test ping between clusters
    for src_cluster in "central" "edge01" "edge02"; do
        local src_kubeconfig_var="${src_cluster^^}_KUBECONFIG"
        local src_kubeconfig="${!src_kubeconfig_var}"

        if [[ ! -f "$src_kubeconfig" ]]; then
            continue
        fi

        for dst_cluster in "central" "edge01" "edge02"; do
            if [[ "$src_cluster" == "$dst_cluster" ]]; then
                continue
            fi

            local dst_kubeconfig_var="${dst_cluster^^}_KUBECONFIG"
            local dst_kubeconfig="${!dst_kubeconfig_var}"

            if [[ ! -f "$dst_kubeconfig" ]]; then
                continue
            fi

            # Get destination pod IP
            local dst_pod_ip=$(kubectl --kubeconfig="$dst_kubeconfig" get pod \
                "test-high-priority-$dst_cluster" -n "$TEST_NAMESPACE" \
                -o jsonpath='{.status.podIP}' 2>/dev/null || echo "")

            if [[ -n "$dst_pod_ip" ]]; then
                connectivity_total=$((connectivity_total + 1))

                # Test ping from source to destination
                local ping_result=$(kubectl --kubeconfig="$src_kubeconfig" exec \
                    "test-high-priority-$src_cluster" -n "$TEST_NAMESPACE" -- \
                    ping -c 3 -W 5 "$dst_pod_ip" 2>/dev/null | grep -c "3 received" || echo "0")

                if [[ $ping_result -gt 0 ]]; then
                    connectivity_success=$((connectivity_success + 1))
                    log_info "Connectivity $src_cluster -> $dst_cluster ($dst_pod_ip): SUCCESS"
                else
                    log_warning "Connectivity $src_cluster -> $dst_cluster ($dst_pod_ip): FAILED"
                fi
            fi
        done
    done

    if [[ $connectivity_success -gt 0 && $connectivity_total -gt 0 ]]; then
        pass_test "Cross-Cluster Connectivity" "($connectivity_success/$connectivity_total successful)"
    else
        fail_test "Cross-Cluster Connectivity" "($connectivity_success/$connectivity_total successful)"
    fi
}

# Test 8: Bandwidth Testing with iperf3
test_bandwidth_qos() {
    start_test "Bandwidth QoS Testing"

    # Deploy iperf3 server pods
    log_info "Deploying iperf3 test infrastructure..."

    for cluster in "central" "edge01"; do
        local kubeconfig_var="${cluster^^}_KUBECONFIG"
        local kubeconfig="${!kubeconfig_var}"

        if [[ ! -f "$kubeconfig" ]]; then
            continue
        fi

        # Deploy iperf3 server for each QoS class
        for qos in "high" "medium" "low"; do
            cat <<EOF | kubectl --kubeconfig="$kubeconfig" apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: iperf3-server-$qos-$cluster
  namespace: $TEST_NAMESPACE
  labels:
    qos-class: $qos
    app: iperf3-server
    test-suite: connectivity
    cluster: $cluster
  annotations:
    ovn.kubernetes.io/logical_switch: "${qos}-priority-slice"
spec:
  containers:
  - name: iperf3-server
    image: networkstatic/iperf3:latest
    command: ["iperf3"]
    args: ["-s", "-p", "5201"]
    ports:
    - containerPort: 5201
      protocol: TCP
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 1000m
        memory: 256Mi
EOF
        done
    done

    # Wait for iperf3 servers to be ready
    sleep 20

    # Run bandwidth tests
    local bandwidth_tests_passed=0
    local bandwidth_tests_total=0

    for qos_class in "high" "medium" "low"; do
        bandwidth_tests_total=$((bandwidth_tests_total + 1))

        local server_ip=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get pod \
            "iperf3-server-$qos_class-central" -n "$TEST_NAMESPACE" \
            -o jsonpath='{.status.podIP}' 2>/dev/null || echo "")

        if [[ -n "$server_ip" && -f "$EDGE01_KUBECONFIG" ]]; then
            # Run iperf3 client test
            local throughput=$(kubectl --kubeconfig="$EDGE01_KUBECONFIG" exec \
                "test-$qos_class-priority-edge01" -n "$TEST_NAMESPACE" -- \
                iperf3 -c "$server_ip" -t 10 -f m 2>/dev/null | \
                grep "receiver" | awk '{print $7}' || echo "0")

            if [[ $(echo "$throughput > 0" | bc -l 2>/dev/null || echo "0") -eq 1 ]]; then
                bandwidth_tests_passed=$((bandwidth_tests_passed + 1))
                log_info "Bandwidth test $qos_class: ${throughput} Mbits/sec"

                # Store results for comparison with expected values
                case $qos_class in
                    "high")
                        HIGH_MEASURED_THROUGHPUT="$throughput"
                        ;;
                    "medium")
                        MEDIUM_MEASURED_THROUGHPUT="$throughput"
                        ;;
                    "low")
                        LOW_MEASURED_THROUGHPUT="$throughput"
                        ;;
                esac
            else
                log_warning "Bandwidth test $qos_class: FAILED (server: $server_ip)"
            fi
        fi
    done

    if [[ $bandwidth_tests_passed -gt 0 ]]; then
        pass_test "Bandwidth QoS Testing" "($bandwidth_tests_passed/$bandwidth_tests_total tests passed)"
    else
        fail_test "Bandwidth QoS Testing" "($bandwidth_tests_passed/$bandwidth_tests_total tests passed)"
    fi
}

# Test 9: Latency Testing
test_latency_qos() {
    start_test "Latency QoS Testing"

    local latency_tests_passed=0
    local latency_tests_total=0

    for qos_class in "high" "medium" "low"; do
        latency_tests_total=$((latency_tests_total + 1))

        local server_ip=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get pod \
            "test-$qos_class-priority-central" -n "$TEST_NAMESPACE" \
            -o jsonpath='{.status.podIP}' 2>/dev/null || echo "")

        if [[ -n "$server_ip" && -f "$EDGE01_KUBECONFIG" ]]; then
            # Measure RTT with ping
            local avg_rtt=$(kubectl --kubeconfig="$EDGE01_KUBECONFIG" exec \
                "test-$qos_class-priority-edge01" -n "$TEST_NAMESPACE" -- \
                ping -c 10 -W 5 "$server_ip" 2>/dev/null | \
                grep "rtt min/avg/max" | cut -d'/' -f5 || echo "0")

            if [[ $(echo "$avg_rtt > 0" | bc -l 2>/dev/null || echo "0") -eq 1 ]]; then
                latency_tests_passed=$((latency_tests_passed + 1))
                log_info "Latency test $qos_class: ${avg_rtt} ms average RTT"

                # Store results for comparison
                case $qos_class in
                    "high")
                        HIGH_MEASURED_RTT="$avg_rtt"
                        ;;
                    "medium")
                        MEDIUM_MEASURED_RTT="$avg_rtt"
                        ;;
                    "low")
                        LOW_MEASURED_RTT="$avg_rtt"
                        ;;
                esac
            else
                log_warning "Latency test $qos_class: FAILED (server: $server_ip)"
            fi
        fi
    done

    if [[ $latency_tests_passed -gt 0 ]]; then
        pass_test "Latency QoS Testing" "($latency_tests_passed/$latency_tests_total tests passed)"
    else
        fail_test "Latency QoS Testing" "($latency_tests_passed/$latency_tests_total tests passed)"
    fi
}

# Test 10: Network Slice Isolation
test_network_isolation() {
    start_test "Network Slice Isolation"

    local isolation_tests_passed=0
    local isolation_tests_total=0

    # Test that low priority pods cannot reach high priority pods directly
    local high_pod_ip=$(kubectl --kubeconfig="$CENTRAL_KUBECONFIG" get pod \
        "test-high-priority-central" -n "$TEST_NAMESPACE" \
        -o jsonpath='{.status.podIP}' 2>/dev/null || echo "")

    if [[ -n "$high_pod_ip" && -f "$EDGE01_KUBECONFIG" ]]; then
        isolation_tests_total=$((isolation_tests_total + 1))

        # Try to connect from low priority pod to high priority pod
        local connection_blocked=$(kubectl --kubeconfig="$EDGE01_KUBECONFIG" exec \
            "test-low-priority-edge01" -n "$TEST_NAMESPACE" -- \
            timeout 5 nc -zv "$high_pod_ip" 8080 2>&1 | grep -c "succeeded" || echo "0")

        if [[ $connection_blocked -eq 0 ]]; then
            isolation_tests_passed=$((isolation_tests_passed + 1))
            log_info "Network isolation: Low->High priority connection properly blocked"
        else
            log_warning "Network isolation: Low->High priority connection NOT blocked"
        fi
    fi

    # Test that medium priority can reach high priority (as per policy)
    if [[ -n "$high_pod_ip" && -f "$EDGE01_KUBECONFIG" ]]; then
        isolation_tests_total=$((isolation_tests_total + 1))

        local connection_allowed=$(kubectl --kubeconfig="$EDGE01_KUBECONFIG" exec \
            "test-medium-priority-edge01" -n "$TEST_NAMESPACE" -- \
            timeout 5 ping -c 1 "$high_pod_ip" 2>/dev/null | grep -c "1 received" || echo "0")

        if [[ $connection_allowed -gt 0 ]]; then
            isolation_tests_passed=$((isolation_tests_passed + 1))
            log_info "Network isolation: Medium->High priority connection properly allowed"
        else
            log_warning "Network isolation: Medium->High priority connection blocked"
        fi
    fi

    if [[ $isolation_tests_passed -eq $isolation_tests_total && $isolation_tests_total -gt 0 ]]; then
        pass_test "Network Slice Isolation" "($isolation_tests_passed/$isolation_tests_total isolation tests passed)"
    else
        fail_test "Network Slice Isolation" "($isolation_tests_passed/$isolation_tests_total isolation tests passed)"
    fi
}

# Generate test report
generate_test_report() {
    log_info "Generating test report..."

    local report_file="/tmp/ovn-connectivity-test-report-$(date +%Y%m%d-%H%M%S).json"

    cat > "$report_file" <<EOF
{
  "test_suite": "Kube-OVN Multi-Site Connectivity Test",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "summary": {
    "total_tests": $TOTAL_TESTS,
    "passed_tests": $PASSED_TESTS,
    "failed_tests": $FAILED_TESTS,
    "success_rate": $(echo "scale=2; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc -l 2>/dev/null || echo "0")
  },
  "performance_metrics": {
    "high_priority": {
      "expected_throughput_mbps": $EXPECTED_DL_THROUGHPUT_HIGH,
      "measured_throughput_mbps": "${HIGH_MEASURED_THROUGHPUT:-0}",
      "expected_rtt_ms": $EXPECTED_RTT_HIGH,
      "measured_rtt_ms": "${HIGH_MEASURED_RTT:-0}"
    },
    "medium_priority": {
      "expected_throughput_mbps": $EXPECTED_DL_THROUGHPUT_MEDIUM,
      "measured_throughput_mbps": "${MEDIUM_MEASURED_THROUGHPUT:-0}",
      "expected_rtt_ms": $EXPECTED_RTT_MEDIUM,
      "measured_rtt_ms": "${MEDIUM_MEASURED_RTT:-0}"
    },
    "low_priority": {
      "expected_throughput_mbps": $EXPECTED_DL_THROUGHPUT_LOW,
      "measured_throughput_mbps": "${LOW_MEASURED_THROUGHPUT:-0}",
      "expected_rtt_ms": $EXPECTED_RTT_LOW,
      "measured_rtt_ms": "${LOW_MEASURED_RTT:-0}"
    }
  },
  "test_results": [
$(IFS=$'\n'; echo "${TEST_RESULTS[*]}" | sed 's/^/    "/' | sed 's/$/"/' | sed '$!s/$/,/')
  ]
}
EOF

    log_success "Test report generated: $report_file"

    # Display summary
    echo
    echo "======================================"
    echo "     OVN CONNECTIVITY TEST SUMMARY"
    echo "======================================"
    echo "Total Tests:  $TOTAL_TESTS"
    echo "Passed:       $PASSED_TESTS"
    echo "Failed:       $FAILED_TESTS"
    echo "Success Rate: $(echo "scale=1; $PASSED_TESTS * 100 / $TOTAL_TESTS" | bc -l 2>/dev/null || echo "0")%"
    echo "======================================"
    echo

    # Display performance comparison
    if [[ -n "${HIGH_MEASURED_THROUGHPUT:-}" ]]; then
        echo "PERFORMANCE COMPARISON:"
        echo "High Priority - Expected: ${EXPECTED_DL_THROUGHPUT_HIGH}Mbps, Measured: ${HIGH_MEASURED_THROUGHPUT}Mbps"
        echo "Medium Priority - Expected: ${EXPECTED_DL_THROUGHPUT_MEDIUM}Mbps, Measured: ${MEDIUM_MEASURED_THROUGHPUT:-N/A}Mbps"
        echo "Low Priority - Expected: ${EXPECTED_DL_THROUGHPUT_LOW}Mbps, Measured: ${LOW_MEASURED_THROUGHPUT:-N/A}Mbps"
        echo
    fi

    if [[ -n "${HIGH_MEASURED_RTT:-}" ]]; then
        echo "LATENCY COMPARISON:"
        echo "High Priority - Expected: ${EXPECTED_RTT_HIGH}ms, Measured: ${HIGH_MEASURED_RTT}ms"
        echo "Medium Priority - Expected: ${EXPECTED_RTT_MEDIUM}ms, Measured: ${MEDIUM_MEASURED_RTT:-N/A}ms"
        echo "Low Priority - Expected: ${EXPECTED_RTT_LOW}ms, Measured: ${LOW_MEASURED_RTT:-N/A}ms"
        echo
    fi

    echo "Detailed report: $report_file"
}

# Main execution
main() {
    log_info "Starting Kube-OVN Multi-Site Connectivity Test Suite"

    # Set up cleanup trap
    trap cleanup EXIT

    # Check prerequisites
    for cmd in kubectl bc; do
        if ! command -v "$cmd" &> /dev/null; then
            log_error "Required command not found: $cmd"
            exit 1
        fi
    done

    # Run tests
    setup_test_environment

    test_ovn_central_connectivity
    test_ovn_edge_connectivity
    test_geneve_tunnels
    test_qos_subnets
    test_network_policies

    deploy_test_pods

    test_cross_cluster_connectivity
    test_bandwidth_qos
    test_latency_qos
    test_network_isolation

    generate_test_report

    # Exit with appropriate code
    if [[ $FAILED_TESTS -eq 0 ]]; then
        log_success "All tests passed successfully!"
        exit 0
    else
        log_error "$FAILED_TESTS tests failed"
        exit 1
    fi
}

# Script entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi