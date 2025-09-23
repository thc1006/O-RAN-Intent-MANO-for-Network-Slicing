#!/bin/bash
# O-RAN Intent-MANO Multi-Cluster Setup Script
# Creates all Kind clusters and configures multi-cluster networking

set -euo pipefail

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

# Configuration
CLUSTERS=("edge01" "edge02" "regional" "central")
KIND_CONFIG_DIR="$PROJECT_ROOT/deploy/kind/configs"
TEMP_DIR="/tmp/oran-mano-setup"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}"
    exit 1
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*${NC}"
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    # Check for required tools
    for tool in kind kubectl docker helm; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed or not in PATH"
        fi
    done

    # Check Docker is running
    if ! docker info &> /dev/null; then
        error "Docker is not running"
    fi

    # Check available resources
    local memory_mb=$(free -m | awk 'NR==2{printf "%.0f", $7}')
    if [ "$memory_mb" -lt 8192 ]; then
        warn "Less than 8GB available memory. Performance may be impacted."
    fi

    log "Prerequisites check passed"
}

# Create temporary directories
setup_directories() {
    log "Setting up directories..."

    mkdir -p "$TEMP_DIR"
    mkdir -p /tmp/{edge01,edge02,regional,central}-{logs,data}
    mkdir -p /tmp/{edge01,edge02,regional,central}-worker{1,2,3}-{logs,data}

    log "Directories created"
}

# Clean up existing clusters
cleanup_clusters() {
    log "Cleaning up existing clusters..."

    for cluster in "${CLUSTERS[@]}"; do
        if kind get clusters | grep -q "^$cluster$"; then
            info "Deleting existing cluster: $cluster"
            kind delete cluster --name "$cluster" || warn "Failed to delete cluster $cluster"
        fi
    done

    log "Cleanup completed"
}

# Create individual cluster
create_cluster() {
    local cluster_name="$1"
    local config_file="$KIND_CONFIG_DIR/${cluster_name}-cluster.yaml"

    log "Creating cluster: $cluster_name"

    if [ ! -f "$config_file" ]; then
        error "Configuration file not found: $config_file"
    fi

    # Create cluster
    kind create cluster --config "$config_file" --wait 300s

    # Wait for cluster to be ready
    local kubeconfig="$(kind get kubeconfig-path --name="$cluster_name")"
    export KUBECONFIG="$kubeconfig"

    info "Waiting for cluster $cluster_name to be ready..."
    kubectl wait --for=condition=Ready nodes --all --timeout=300s

    log "Cluster $cluster_name created and ready"
}

# Install Kube-OVN CNI
install_kube_ovn() {
    local cluster_name="$1"

    log "Installing Kube-OVN CNI on cluster: $cluster_name"

    # Switch to cluster context
    kubectl config use-context "kind-$cluster_name"

    # Determine subnet based on cluster
    local pod_subnet=""
    local service_subnet=""
    case "$cluster_name" in
        "edge01")
            pod_subnet="10.244.0.0/16"
            service_subnet="10.96.0.0/12"
            ;;
        "edge02")
            pod_subnet="10.245.0.0/16"
            service_subnet="10.97.0.0/12"
            ;;
        "regional")
            pod_subnet="10.246.0.0/16"
            service_subnet="10.98.0.0/12"
            ;;
        "central")
            pod_subnet="10.247.0.0/16"
            service_subnet="10.99.0.0/12"
            ;;
    esac

    # Download and apply Kube-OVN
    local kube_ovn_version="v1.12.0"
    curl -sSL "https://raw.githubusercontent.com/kubeovn/kube-ovn/$kube_ovn_version/dist/images/install.sh" | \
        POD_CIDR="$pod_subnet" \
        SVC_CIDR="$service_subnet" \
        bash

    # Wait for Kube-OVN to be ready
    info "Waiting for Kube-OVN pods to be ready..."
    kubectl wait --for=condition=Ready pods -n kube-system -l app=ovn-central --timeout=300s
    kubectl wait --for=condition=Ready pods -n kube-system -l app=ovs-ovn --timeout=300s
    kubectl wait --for=condition=Ready pods -n kube-system -l app=kube-ovn-controller --timeout=300s

    log "Kube-OVN installed on cluster: $cluster_name"
}

# Setup inter-cluster networking
setup_inter_cluster_networking() {
    log "Setting up inter-cluster networking..."

    # Create network bridges for cluster communication
    for i in "${!CLUSTERS[@]}"; do
        local cluster="${CLUSTERS[$i]}"
        local bridge_name="br-$cluster"

        # Create bridge if it doesn't exist
        if ! docker network ls | grep -q "$bridge_name"; then
            info "Creating network bridge: $bridge_name"
            docker network create \
                --driver bridge \
                --subnet="172.2$i.0.0/16" \
                --gateway="172.2$i.0.1" \
                "$bridge_name"
        fi

        # Connect cluster nodes to bridge
        local nodes=$(kind get nodes --name="$cluster")
        for node in $nodes; do
            if ! docker network inspect "$bridge_name" | grep -q "$node"; then
                info "Connecting node $node to bridge $bridge_name"
                docker network connect "$bridge_name" "$node" || warn "Failed to connect $node to $bridge_name"
            fi
        done
    done

    log "Inter-cluster networking configured"
}

# Load container images to clusters
load_images() {
    log "Loading container images to clusters..."

    # Build images if they don't exist
    local images=(
        "oran-orchestrator:latest"
        "oran-vnf-operator:latest"
        "oran-o2-client:latest"
        "oran-tn-manager:latest"
        "oran-tn-agent:latest"
        "oran-ran-dms:latest"
        "oran-cn-dms:latest"
    )

    for image in "${images[@]}"; do
        info "Checking image: $image"
        if ! docker images | grep -q "${image%:*}"; then
            warn "Image $image not found, skipping load"
            continue
        fi

        for cluster in "${CLUSTERS[@]}"; do
            info "Loading $image to cluster $cluster"
            kind load docker-image "$image" --name "$cluster" || warn "Failed to load $image to $cluster"
        done
    done

    log "Container images loaded"
}

# Install monitoring stack
install_monitoring() {
    local cluster_name="$1"

    log "Installing monitoring stack on cluster: $cluster_name"

    kubectl config use-context "kind-$cluster_name"

    # Create monitoring namespace
    kubectl create namespace oran-monitoring --dry-run=client -o yaml | kubectl apply -f -

    # Install Prometheus
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    helm repo update

    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace oran-monitoring \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.retention=7d \
        --set grafana.adminPassword=admin \
        --wait --timeout=600s

    log "Monitoring stack installed on cluster: $cluster_name"
}

# Verify cluster setup
verify_cluster() {
    local cluster_name="$1"

    log "Verifying cluster: $cluster_name"

    kubectl config use-context "kind-$cluster_name"

    # Check nodes
    local node_count=$(kubectl get nodes --no-headers | wc -l)
    info "Cluster $cluster_name has $node_count nodes"

    # Check system pods
    local system_pods_ready=$(kubectl get pods -n kube-system --no-headers | grep -c "Running")
    local system_pods_total=$(kubectl get pods -n kube-system --no-headers | wc -l)
    info "System pods ready: $system_pods_ready/$system_pods_total"

    # Check CNI
    if kubectl get pods -n kube-system -l app=kube-ovn-controller --no-headers | grep -q "Running"; then
        info "Kube-OVN CNI is running"
    else
        warn "Kube-OVN CNI is not running properly"
    fi

    log "Cluster $cluster_name verification completed"
}

# Generate kubeconfig for multi-cluster access
generate_multi_cluster_kubeconfig() {
    log "Generating multi-cluster kubeconfig..."

    local multi_config="$TEMP_DIR/multi-cluster-kubeconfig.yaml"

    # Merge all cluster configs
    export KUBECONFIG=""
    for cluster in "${CLUSTERS[@]}"; do
        local cluster_config=$(kind get kubeconfig-path --name="$cluster")
        export KUBECONFIG="$KUBECONFIG:$cluster_config"
    done

    # Create merged config
    kubectl config view --merge --flatten > "$multi_config"

    # Set default context to central
    kubectl --kubeconfig="$multi_config" config use-context kind-central

    info "Multi-cluster kubeconfig saved to: $multi_config"
    echo "export KUBECONFIG=\"$multi_config\"" > "$TEMP_DIR/kubeconfig.env"

    log "Multi-cluster kubeconfig generated"
}

# Main execution
main() {
    log "Starting O-RAN Intent-MANO multi-cluster setup"

    # Parse command line arguments
    local skip_cleanup=false
    local install_monitoring_flag=false
    local clusters_to_create=("${CLUSTERS[@]}")

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            --with-monitoring)
                install_monitoring_flag=true
                shift
                ;;
            --clusters)
                IFS=',' read -ra clusters_to_create <<< "$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-cleanup     Skip cleanup of existing clusters"
                echo "  --with-monitoring  Install monitoring stack"
                echo "  --clusters LIST    Comma-separated list of clusters to create"
                echo "  --help, -h         Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # Execute setup steps
    check_prerequisites
    setup_directories

    if [ "$skip_cleanup" = false ]; then
        cleanup_clusters
    fi

    # Create clusters
    for cluster in "${clusters_to_create[@]}"; do
        create_cluster "$cluster"
        install_kube_ovn "$cluster"

        if [ "$install_monitoring_flag" = true ] && [ "$cluster" = "central" ]; then
            install_monitoring "$cluster"
        fi

        verify_cluster "$cluster"
    done

    # Setup networking and finalize
    setup_inter_cluster_networking
    load_images
    generate_multi_cluster_kubeconfig

    log "O-RAN Intent-MANO multi-cluster setup completed successfully!"
    echo ""
    info "Next steps:"
    echo "1. Source the kubeconfig environment: source $TEMP_DIR/kubeconfig.env"
    echo "2. Verify clusters: kubectl config get-contexts"
    echo "3. Deploy MANO components: $PROJECT_ROOT/deploy/scripts/setup/deploy-mano.sh"
    echo ""
    info "Cluster endpoints:"
    for cluster in "${clusters_to_create[@]}"; do
        local api_server=$(kubectl --kubeconfig="$TEMP_DIR/multi-cluster-kubeconfig.yaml" config view -o jsonpath="{.clusters[?(@.name==\"kind-$cluster\")].cluster.server}")
        echo "  $cluster: $api_server"
    done
}

# Execute main function
main "$@"