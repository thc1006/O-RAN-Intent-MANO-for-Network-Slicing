#!/bin/bash
# RAN Domain Deployment Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# RAN deployment function
deploy_ran() {
    local scenario=$1
    local deployment_mode=${2:-fast}

    log_info "Deploying RAN for scenario: ${scenario}"

    # Load scenario configuration
    local scenario_file="${SCRIPT_DIR}/../scenarios/${scenario}.yaml"
    if [[ ! -f "$scenario_file" ]]; then
        log_error "Scenario file not found: $scenario_file"
        return 1
    fi

    # Extract RAN configuration
    local ran_count=$(yq eval '.network_functions.ran[0].count' "$scenario_file")
    local placement=$(yq eval '.network_functions.ran[0].placement' "$scenario_file")
    local priority=$(yq eval '.network_functions.ran[0].priority // "normal"' "$scenario_file")

    log_info "RAN Configuration: count=${ran_count}, placement=${placement}, priority=${priority}"

    # Start RAN deployment timer
    local start_time=$(date +%s.%N)

    # Deploy gNodeB components
    for ((i=1; i<=ran_count; i++)); do
        deploy_gnodeb "$scenario" "$i" "$placement" "$priority"
    done

    # Deploy additional RAN components based on scenario
    case $scenario in
        embb)
            deploy_ran_embb "$scenario"
            ;;
        urllc)
            deploy_ran_urllc "$scenario"
            ;;
        miot)
            deploy_ran_miot "$scenario"
            ;;
    esac

    # Wait for RAN readiness
    wait_for_ran_ready "$scenario"

    # Calculate deployment time
    local end_time=$(date +%s.%N)
    local duration=$(echo "$end_time - $start_time" | bc)

    log_info "RAN deployment completed in ${duration}s"
    echo "$duration"
}

# Deploy gNodeB
deploy_gnodeb() {
    local scenario=$1
    local instance=$2
    local placement=$3
    local priority=$4

    log_info "Deploying gNodeB-${instance} for ${scenario}"

    # Get bandwidth and latency from scenario
    local bandwidth=$(get_scenario_bandwidth "$scenario")
    local latency=$(get_scenario_latency "$scenario")

    kubectl apply -f - <<EOF
apiVersion: mano.oran.io/v1alpha1
kind: VNF
metadata:
  name: gnodeb-${scenario}-${instance}
  namespace: oran-system
  labels:
    experiment: e2e-test
    scenario: ${scenario}
    domain: ran
    component: gnodeb
    instance: "${instance}"
spec:
  type: RAN
  version: 1.0.0
  placement:
    cloudType: ${placement}
    maxLatency: 5
    preferredZones:
    - ${placement}-zone-${instance}
  resources:
    cpu: "1"
    memory: "2Gi"
    storage: "10Gi"
    networkBandwidth: 1000
  qos:
    bandwidth: ${bandwidth}
    latency: ${latency}
    jitter: 1.0
    reliability: 99.9
  targetClusters:
  - ${placement}01
  configData:
    SCENARIO: ${scenario}
    COMPONENT: gnodeb
    INSTANCE: "${instance}"
    PRIORITY: ${priority}
    CELL_ID: "$((1000 + instance))"
    PLMN: "00101"
    TAC: "000${instance}"
EOF

    log_info "gNodeB-${instance} deployment initiated"
}

# eMBB-specific RAN deployment
deploy_ran_embb() {
    local scenario=$1

    log_info "Deploying eMBB-specific RAN components"

    # Deploy CU (Centralized Unit)
    kubectl apply -f - <<EOF
apiVersion: mano.oran.io/v1alpha1
kind: VNF
metadata:
  name: cu-${scenario}
  namespace: oran-system
  labels:
    experiment: e2e-test
    scenario: ${scenario}
    domain: ran
    component: cu
spec:
  type: RAN
  version: 1.0.0
  placement:
    cloudType: edge
    maxLatency: 10
  resources:
    cpu: "2"
    memory: "4Gi"
    storage: "20Gi"
  qos:
    bandwidth: 4.57
    latency: 16.1
  configData:
    SCENARIO: ${scenario}
    COMPONENT: cu
    SPLIT_OPTION: "7-2x"
    BANDWIDTH_OPTIMIZATION: "true"
EOF

    # Deploy DU (Distributed Unit)
    kubectl apply -f - <<EOF
apiVersion: mano.oran.io/v1alpha1
kind: VNF
metadata:
  name: du-${scenario}
  namespace: oran-system
  labels:
    experiment: e2e-test
    scenario: ${scenario}
    domain: ran
    component: du
spec:
  type: RAN
  version: 1.0.0
  placement:
    cloudType: edge
    maxLatency: 5
  resources:
    cpu: "1"
    memory: "2Gi"
  qos:
    bandwidth: 4.57
    latency: 16.1
  configData:
    SCENARIO: ${scenario}
    COMPONENT: du
    FRONTHAUL_INTERFACE: "eth1"
EOF

    log_info "eMBB RAN components deployed"
}

# uRLLC-specific RAN deployment
deploy_ran_urllc() {
    local scenario=$1

    log_info "Deploying uRLLC-specific RAN components"

    # Deploy edge-optimized CU
    kubectl apply -f - <<EOF
apiVersion: mano.oran.io/v1alpha1
kind: VNF
metadata:
  name: cu-${scenario}
  namespace: oran-system
  labels:
    experiment: e2e-test
    scenario: ${scenario}
    domain: ran
    component: cu
spec:
  type: RAN
  version: 1.0.0
  placement:
    cloudType: edge
    maxLatency: 1
  resources:
    cpu: "1"
    memory: "2Gi"
    storage: "10Gi"
  qos:
    bandwidth: 0.93
    latency: 6.3
    reliability: 99.999
  configData:
    SCENARIO: ${scenario}
    COMPONENT: cu
    LATENCY_OPTIMIZATION: "true"
    PREEMPTION_ENABLED: "true"
    PRIORITY: "10"
EOF

    log_info "uRLLC RAN components deployed"
}

# mIoT-specific RAN deployment
deploy_ran_miot() {
    local scenario=$1

    log_info "Deploying mIoT-specific RAN components"

    # Deploy connection-optimized CU
    kubectl apply -f - <<EOF
apiVersion: mano.oran.io/v1alpha1
kind: VNF
metadata:
  name: cu-${scenario}
  namespace: oran-system
  labels:
    experiment: e2e-test
    scenario: ${scenario}
    domain: ran
    component: cu
spec:
  type: RAN
  version: 1.0.0
  placement:
    cloudType: edge
    maxLatency: 20
  resources:
    cpu: "1"
    memory: "2Gi"
    storage: "15Gi"
  qos:
    bandwidth: 2.77
    latency: 15.7
  configData:
    SCENARIO: ${scenario}
    COMPONENT: cu
    MASSIVE_CONNECTIVITY: "true"
    CONNECTION_DENSITY: "high"
    ENERGY_EFFICIENCY: "true"
EOF

    log_info "mIoT RAN components deployed"
}

# Wait for RAN components to be ready
wait_for_ran_ready() {
    local scenario=$1
    local timeout=300  # 5 minutes

    log_info "Waiting for RAN components to be ready..."

    # Wait for all RAN VNFs
    if ! kubectl wait --for=condition=Ready vnf \
        -l scenario=${scenario},domain=ran \
        -n oran-system \
        --timeout=${timeout}s; then
        log_warning "Some RAN VNFs did not become ready within timeout"
    fi

    # Additional readiness checks
    local ready_count=$(kubectl get vnf -l scenario=${scenario},domain=ran -n oran-system \
        -o jsonpath='{.items[?(@.status.phase=="Ready")].metadata.name}' | wc -w)

    log_info "RAN readiness check: ${ready_count} components ready"

    # Simulate RAN-specific initialization delay
    case $scenario in
        embb)
            log_info "eMBB RAN initialization (bandwidth optimization)..."
            sleep 10
            ;;
        urllc)
            log_info "uRLLC RAN initialization (latency optimization)..."
            sleep 5
            ;;
        miot)
            log_info "mIoT RAN initialization (connection setup)..."
            sleep 8
            ;;
    esac
}

# Get scenario bandwidth
get_scenario_bandwidth() {
    local scenario=$1
    case $scenario in
        embb) echo "4.57" ;;
        urllc) echo "0.93" ;;
        miot) echo "2.77" ;;
        *) echo "1.0" ;;
    esac
}

# Get scenario latency
get_scenario_latency() {
    local scenario=$1
    case $scenario in
        embb) echo "16.1" ;;
        urllc) echo "6.3" ;;
        miot) echo "15.7" ;;
        *) echo "10.0" ;;
    esac
}

# Main execution when script is called directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    SCENARIO=${1:-embb}
    MODE=${2:-fast}

    duration=$(deploy_ran "$SCENARIO" "$MODE")
    log_info "RAN deployment duration: ${duration}s"
fi