#!/bin/bash
# Latency Testing Script for Multi-Cluster Kube-OVN
# Tests pod-to-pod RTT and cross-site service connectivity

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NAMESPACE=${NAMESPACE:-default}
TEST_IMAGE=${TEST_IMAGE:-nicolaka/netshoot:latest}
PING_COUNT=${PING_COUNT:-100}
PING_INTERVAL=${PING_INTERVAL:-0.1}
SERVICE_PORT=${SERVICE_PORT:-8080}
RESULTS_FILE=${RESULTS_FILE:-latency_results.json}

# Expected delays (ms)
declare -A EXPECTED_DELAYS=(
    ["central-regional"]=7
    ["central-edge01"]=7
    ["central-edge02"]=7
    ["regional-edge01"]=5
    ["regional-edge02"]=5
    ["edge01-edge02"]=5
    ["same-cluster"]=0
)

# Tolerance for delay validation (ms)
DELAY_TOLERANCE=1

# Cluster configurations
declare -A CLUSTER_CONFIGS=(
    ["central"]="10.0.0.0/16"
    ["regional"]="10.1.0.0/16"
    ["edge01"]="10.2.0.0/16"
    ["edge02"]="10.3.0.0/16"
)

# Test results
declare -A TEST_RESULTS
FAILED_TESTS=0
PASSED_TESTS=0

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl."
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    # Check if Kube-OVN is installed
    if ! kubectl get crd subnets.kubeovn.io &> /dev/null; then
        log_error "Kube-OVN CRDs not found. Please install Kube-OVN first."
        exit 1
    fi

    log_info "Prerequisites check passed"
}

# Create test pods in each cluster
create_test_pods() {
    local cluster=$1
    local pod_name="latency-test-${cluster}"

    log_info "Creating test pod in cluster: ${cluster}"

    # Create pod with multi-NIC support if needed
    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: ${pod_name}
  namespace: ${NAMESPACE}
  labels:
    app: latency-test
    cluster: ${cluster}
  annotations:
    k8s.v1.cni.cncf.io/networks: |
      [
        {
          "name": "data-network",
          "interface": "eth1"
        }
      ]
spec:
  containers:
  - name: netshoot
    image: ${TEST_IMAGE}
    command: ["/bin/sleep", "3650d"]
    securityContext:
      capabilities:
        add:
        - NET_ADMIN
        - NET_RAW
    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "128Mi"
        cpu: "200m"
EOF

    # Wait for pod to be ready
    kubectl wait --for=condition=ready pod/${pod_name} -n ${NAMESPACE} --timeout=60s || {
        log_error "Pod ${pod_name} failed to start"
        return 1
    }

    # Get pod IP
    local pod_ip=$(kubectl get pod ${pod_name} -n ${NAMESPACE} -o jsonpath='{.status.podIP}')
    log_info "Pod ${pod_name} created with IP: ${pod_ip}"

    echo "${pod_ip}"
}

# Create test service
create_test_service() {
    local cluster=$1
    local service_name="latency-service-${cluster}"

    log_info "Creating test service in cluster: ${cluster}"

    cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Service
metadata:
  name: ${service_name}
  namespace: ${NAMESPACE}
  labels:
    app: latency-test
    cluster: ${cluster}
spec:
  selector:
    app: latency-test
    cluster: ${cluster}
  ports:
  - protocol: TCP
    port: ${SERVICE_PORT}
    targetPort: 8080
  type: ClusterIP
EOF

    # Start a simple HTTP server in the pod
    local pod_name="latency-test-${cluster}"
    kubectl exec -n ${NAMESPACE} ${pod_name} -- sh -c "python3 -m http.server 8080 &" 2>/dev/null || \
    kubectl exec -n ${NAMESPACE} ${pod_name} -- sh -c "nc -l -p 8080 &" 2>/dev/null || true

    local service_ip=$(kubectl get svc ${service_name} -n ${NAMESPACE} -o jsonpath='{.spec.clusterIP}')
    log_info "Service ${service_name} created with IP: ${service_ip}"

    echo "${service_ip}"
}

# Measure RTT between pods
measure_rtt() {
    local source_pod=$1
    local target_ip=$2
    local expected_delay=$3
    local test_name=$4

    log_test "Testing RTT from ${source_pod} to ${target_ip} (expected: ${expected_delay}ms ±${DELAY_TOLERANCE}ms)"

    # Run ping test
    local ping_output=$(kubectl exec -n ${NAMESPACE} ${source_pod} -- \
        ping -c ${PING_COUNT} -i ${PING_INTERVAL} -q ${target_ip} 2>/dev/null || echo "FAILED")

    if [[ "${ping_output}" == "FAILED" ]]; then
        log_error "Ping test failed from ${source_pod} to ${target_ip}"
        TEST_RESULTS["${test_name}_status"]="FAILED"
        TEST_RESULTS["${test_name}_reason"]="Connectivity failed"
        ((FAILED_TESTS++))
        return 1
    fi

    # Parse ping statistics
    local packet_loss=$(echo "${ping_output}" | grep -oP '\d+(?=% packet loss)' || echo "100")
    local rtt_line=$(echo "${ping_output}" | grep "rtt min/avg/max/mdev")

    if [[ -z "${rtt_line}" ]]; then
        log_error "Could not parse RTT statistics"
        TEST_RESULTS["${test_name}_status"]="FAILED"
        TEST_RESULTS["${test_name}_reason"]="Parse error"
        ((FAILED_TESTS++))
        return 1
    fi

    # Extract RTT values (min/avg/max/mdev)
    local rtt_values=$(echo "${rtt_line}" | awk -F'=' '{print $2}' | awk '{print $1}')
    IFS='/' read -r min_rtt avg_rtt max_rtt mdev_rtt <<< "${rtt_values}"

    # Validate RTT against expected delay
    local avg_rtt_int=${avg_rtt%.*}
    local min_expected=$((expected_delay - DELAY_TOLERANCE))
    local max_expected=$((expected_delay + DELAY_TOLERANCE))

    TEST_RESULTS["${test_name}_avg_rtt"]="${avg_rtt}"
    TEST_RESULTS["${test_name}_min_rtt"]="${min_rtt}"
    TEST_RESULTS["${test_name}_max_rtt"]="${max_rtt}"
    TEST_RESULTS["${test_name}_packet_loss"]="${packet_loss}"
    TEST_RESULTS["${test_name}_expected"]="${expected_delay}"

    if [[ ${avg_rtt_int} -ge ${min_expected} ]] && [[ ${avg_rtt_int} -le ${max_expected} ]]; then
        log_info "✓ RTT: ${avg_rtt}ms (min: ${min_rtt}ms, max: ${max_rtt}ms) - PASSED"
        TEST_RESULTS["${test_name}_status"]="PASSED"
        ((PASSED_TESTS++))
        return 0
    else
        log_warn "✗ RTT: ${avg_rtt}ms (expected: ${expected_delay}±${DELAY_TOLERANCE}ms) - FAILED"
        TEST_RESULTS["${test_name}_status"]="FAILED"
        TEST_RESULTS["${test_name}_reason"]="RTT out of range"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test service connectivity
test_service_connectivity() {
    local source_pod=$1
    local service_ip=$2
    local test_name=$3

    log_test "Testing service connectivity from ${source_pod} to ${service_ip}:${SERVICE_PORT}"

    # Test TCP connectivity
    local nc_result=$(kubectl exec -n ${NAMESPACE} ${source_pod} -- \
        timeout 5 nc -zv ${service_ip} ${SERVICE_PORT} 2>&1 || echo "FAILED")

    if [[ "${nc_result}" == *"succeeded"* ]] || [[ "${nc_result}" == *"open"* ]]; then
        log_info "✓ Service ${service_ip}:${SERVICE_PORT} is reachable - PASSED"
        TEST_RESULTS["${test_name}_service"]="REACHABLE"
        ((PASSED_TESTS++))
        return 0
    else
        log_error "✗ Service ${service_ip}:${SERVICE_PORT} is not reachable - FAILED"
        TEST_RESULTS["${test_name}_service"]="UNREACHABLE"
        ((FAILED_TESTS++))
        return 1
    fi
}

# Test within same cluster (should have ~0ms delay)
test_same_cluster() {
    local cluster=$1

    log_info "Testing same-cluster latency in ${cluster}"

    # Create two pods in same cluster
    local pod1_name="latency-test-${cluster}-1"
    local pod2_name="latency-test-${cluster}-2"

    # Create first pod
    local pod1_ip=$(create_test_pods "${cluster}-1")

    # Create second pod
    local pod2_ip=$(create_test_pods "${cluster}-2")

    # Test RTT between pods in same cluster
    measure_rtt "${pod1_name}" "${pod2_ip}" 0 "${cluster}_internal"

    # Cleanup
    kubectl delete pod ${pod1_name} ${pod2_name} -n ${NAMESPACE} --ignore-not-found=true
}

# Run complete test suite
run_test_suite() {
    log_info "Starting multi-cluster latency test suite"

    local test_start_time=$(date +%s)

    # Create test infrastructure
    declare -A POD_IPS
    declare -A SERVICE_IPS

    # Create pods and services in each "cluster" (simulated with labels)
    for cluster in central regional edge01 edge02; do
        POD_IPS[${cluster}]=$(create_test_pods ${cluster})
        SERVICE_IPS[${cluster}]=$(create_test_service ${cluster})
    done

    # Test pod-to-pod latency between clusters
    log_info "Testing inter-cluster pod-to-pod latency..."

    # Central to other clusters
    measure_rtt "latency-test-central" "${POD_IPS[regional]}" ${EXPECTED_DELAYS["central-regional"]} "central_to_regional"
    measure_rtt "latency-test-central" "${POD_IPS[edge01]}" ${EXPECTED_DELAYS["central-edge01"]} "central_to_edge01"
    measure_rtt "latency-test-central" "${POD_IPS[edge02]}" ${EXPECTED_DELAYS["central-edge02"]} "central_to_edge02"

    # Regional to edge clusters
    measure_rtt "latency-test-regional" "${POD_IPS[edge01]}" ${EXPECTED_DELAYS["regional-edge01"]} "regional_to_edge01"
    measure_rtt "latency-test-regional" "${POD_IPS[edge02]}" ${EXPECTED_DELAYS["regional-edge02"]} "regional_to_edge02"

    # Edge to edge
    measure_rtt "latency-test-edge01" "${POD_IPS[edge02]}" ${EXPECTED_DELAYS["edge01-edge02"]} "edge01_to_edge02"

    # Test service connectivity
    log_info "Testing cross-cluster service connectivity..."

    for source_cluster in central regional edge01 edge02; do
        for target_cluster in central regional edge01 edge02; do
            if [[ ${source_cluster} != ${target_cluster} ]]; then
                test_service_connectivity "latency-test-${source_cluster}" \
                    "${SERVICE_IPS[${target_cluster}]}" \
                    "${source_cluster}_to_${target_cluster}_service"
            fi
        done
    done

    # Test same-cluster latency (should be ~0ms)
    test_same_cluster "central"

    local test_end_time=$(date +%s)
    local test_duration=$((test_end_time - test_start_time))

    # Generate results summary
    generate_results_summary ${test_duration}
}

# Generate JSON results file
generate_results_summary() {
    local duration=$1

    log_info "Generating test results summary..."

    # Create JSON results
    cat > ${RESULTS_FILE} <<EOF
{
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "duration_seconds": ${duration},
  "total_tests": $((PASSED_TESTS + FAILED_TESTS)),
  "passed_tests": ${PASSED_TESTS},
  "failed_tests": ${FAILED_TESTS},
  "success_rate": $(awk "BEGIN {printf \"%.2f\", ${PASSED_TESTS}*100/(${PASSED_TESTS}+${FAILED_TESTS})}")%,
  "test_results": {
EOF

    # Add individual test results
    local first=true
    for key in "${!TEST_RESULTS[@]}"; do
        if [[ ${first} == true ]]; then
            first=false
        else
            echo "," >> ${RESULTS_FILE}
        fi
        echo -n "    \"${key}\": \"${TEST_RESULTS[${key}]}\"" >> ${RESULTS_FILE}
    done

    cat >> ${RESULTS_FILE} <<EOF

  },
  "expected_delays": {
    "central_regional": ${EXPECTED_DELAYS["central-regional"]},
    "central_edge01": ${EXPECTED_DELAYS["central-edge01"]},
    "central_edge02": ${EXPECTED_DELAYS["central-edge02"]},
    "regional_edge01": ${EXPECTED_DELAYS["regional-edge01"]},
    "regional_edge02": ${EXPECTED_DELAYS["regional-edge02"]},
    "edge01_edge02": ${EXPECTED_DELAYS["edge01-edge02"]},
    "same_cluster": ${EXPECTED_DELAYS["same-cluster"]}
  }
}
EOF

    log_info "Results saved to ${RESULTS_FILE}"

    # Display summary
    echo ""
    echo "========================================="
    echo "         TEST SUMMARY                   "
    echo "========================================="
    echo -e "Total Tests:    $((PASSED_TESTS + FAILED_TESTS))"
    echo -e "Passed:         ${GREEN}${PASSED_TESTS}${NC}"
    echo -e "Failed:         ${RED}${FAILED_TESTS}${NC}"
    echo -e "Success Rate:   $(awk "BEGIN {printf \"%.2f\", ${PASSED_TESTS}*100/(${PASSED_TESTS}+${FAILED_TESTS})}")%"
    echo -e "Duration:       ${duration}s"
    echo "========================================="

    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        echo ""
        log_warn "Some tests failed. Review ${RESULTS_FILE} for details."
        return 1
    else
        log_info "All tests passed successfully!"
        return 0
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up test resources..."

    # Delete test pods
    kubectl delete pods -n ${NAMESPACE} -l app=latency-test --ignore-not-found=true

    # Delete test services
    kubectl delete services -n ${NAMESPACE} -l app=latency-test --ignore-not-found=true

    log_info "Cleanup complete"
}

# Main execution
main() {
    echo "==========================================="
    echo "  Multi-Cluster Kube-OVN Latency Testing  "
    echo "==========================================="
    echo ""

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --ping-count)
                PING_COUNT="$2"
                shift 2
                ;;
            --results-file)
                RESULTS_FILE="$2"
                shift 2
                ;;
            --cleanup-only)
                cleanup
                exit 0
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --namespace NAME      Kubernetes namespace to use (default: default)"
                echo "  --ping-count N        Number of ping packets (default: 100)"
                echo "  --results-file FILE   Output file for results (default: latency_results.json)"
                echo "  --cleanup-only        Only cleanup test resources"
                echo "  --help               Show this help message"
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Trap cleanup on exit
    trap cleanup EXIT

    # Check prerequisites
    check_prerequisites

    # Run test suite
    run_test_suite

    # Exit with appropriate code
    if [[ ${FAILED_TESTS} -gt 0 ]]; then
        exit 1
    else
        exit 0
    fi
}

# Run main function
main "$@"