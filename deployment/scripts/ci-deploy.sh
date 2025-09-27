#!/bin/bash

# CI Deployment Script for O-RAN MANO Monitoring Stack
# This script creates a kind cluster, installs dependencies, and deploys the monitoring stack

set -euo pipefail

# Configuration
CLUSTER_NAME="${CLUSTER_NAME:-oran-test-cluster}"
NAMESPACE="${NAMESPACE:-monitoring}"
TIMEOUT="${TIMEOUT:-600}"
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-v2.47.0}"
GRAFANA_VERSION="${GRAFANA_VERSION:-10.1.0}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

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

# Error handling
cleanup_on_error() {
    local exit_code=$?
    error "Deployment failed with exit code $exit_code"

    if command -v kind &> /dev/null; then
        log "Collecting cluster logs for debugging..."
        mkdir -p "${PROJECT_ROOT}/deployment/logs"

        # Get cluster info
        kind get clusters | grep -q "$CLUSTER_NAME" && {
            kubectl cluster-info dump --output-directory="${PROJECT_ROOT}/deployment/logs/cluster-dump" || true
            kubectl get pods --all-namespaces -o wide > "${PROJECT_ROOT}/deployment/logs/pods.log" || true
            kubectl get events --all-namespaces --sort-by='.lastTimestamp' > "${PROJECT_ROOT}/deployment/logs/events.log" || true

            # Get logs from monitoring namespace
            kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus --tail=100 > "${PROJECT_ROOT}/deployment/logs/prometheus.log" || true
            kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=grafana --tail=100 > "${PROJECT_ROOT}/deployment/logs/grafana.log" || true
            kubectl logs -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager --tail=100 > "${PROJECT_ROOT}/deployment/logs/alertmanager.log" || true
        }

        # Cleanup cluster if requested
        if [ "${CLEANUP_ON_FAILURE:-true}" = "true" ]; then
            warning "Cleaning up cluster due to failure..."
            kind delete cluster --name "$CLUSTER_NAME" || true
        fi
    fi

    exit $exit_code
}

trap cleanup_on_error ERR

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."

    local missing_tools=()

    command -v kind &> /dev/null || missing_tools+=("kind")
    command -v kubectl &> /dev/null || missing_tools+=("kubectl")
    command -v helm &> /dev/null || missing_tools+=("helm")
    command -v docker &> /dev/null || missing_tools+=("docker")

    if [ ${#missing_tools[@]} -ne 0 ]; then
        error "Missing required tools: ${missing_tools[*]}"
        error "Please install the missing tools and try again"
        exit 1
    fi

    # Check if Docker is running
    if ! docker info &> /dev/null; then
        error "Docker daemon is not running"
        exit 1
    fi

    success "All prerequisites are satisfied"
}

# Create kind cluster
create_cluster() {
    log "Creating kind cluster: $CLUSTER_NAME"

    # Check if cluster already exists
    if kind get clusters | grep -q "^$CLUSTER_NAME$"; then
        warning "Cluster $CLUSTER_NAME already exists, deleting it..."
        kind delete cluster --name "$CLUSTER_NAME"
    fi

    # Create cluster configuration
    cat > /tmp/kind-config.yaml << 'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 9090
    hostPort: 9090
    protocol: TCP
  - containerPort: 3000
    hostPort: 3000
    protocol: TCP
  - containerPort: 9093
    hostPort: 9093
    protocol: TCP
- role: worker
  labels:
    worker-type: "monitoring"
- role: worker
  labels:
    worker-type: "applications"
EOF

    kind create cluster --name "$CLUSTER_NAME" --config /tmp/kind-config.yaml --wait 300s

    # Set kubectl context
    kubectl config use-context "kind-$CLUSTER_NAME"

    success "Kind cluster created successfully"
}

# Install core dependencies
install_dependencies() {
    log "Installing core dependencies..."

    # Install cert-manager
    log "Installing cert-manager..."
    kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager -n cert-manager
    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager-cainjector -n cert-manager
    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager-webhook -n cert-manager

    # Install NGINX Ingress Controller
    log "Installing NGINX Ingress Controller..."
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=300s

    # Create monitoring namespace
    kubectl create namespace "$NAMESPACE" || true

    success "Core dependencies installed"
}

# Install Prometheus Operator
install_prometheus_operator() {
    log "Installing Prometheus Operator..."

    # Add Helm repositories
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo update

    # Install Prometheus Operator with CRDs
    helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack \
        --namespace "$NAMESPACE" \
        --create-namespace \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.ruleSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.retention=7d \
        --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=10Gi \
        --set grafana.adminPassword=admin123 \
        --set grafana.service.type=NodePort \
        --set prometheus.service.type=NodePort \
        --set alertmanager.service.type=NodePort \
        --timeout "${TIMEOUT}s" \
        --wait

    success "Prometheus Operator installed"
}

# Deploy O-RAN monitoring components
deploy_oran_monitoring() {
    log "Deploying O-RAN monitoring components..."

    # Apply monitoring configurations if they exist
    if [ -d "${PROJECT_ROOT}/monitoring/prometheus" ]; then
        log "Applying Prometheus configurations..."
        find "${PROJECT_ROOT}/monitoring/prometheus" -name "*.yaml" -o -name "*.yml" | while read -r file; do
            kubectl apply -f "$file" -n "$NAMESPACE" || warning "Failed to apply $file"
        done
    fi

    if [ -d "${PROJECT_ROOT}/monitoring/grafana" ]; then
        log "Applying Grafana configurations..."
        find "${PROJECT_ROOT}/monitoring/grafana" -name "*.yaml" -o -name "*.yml" | while read -r file; do
            kubectl apply -f "$file" -n "$NAMESPACE" || warning "Failed to apply $file"
        done
    fi

    if [ -d "${PROJECT_ROOT}/monitoring/alerting" ]; then
        log "Applying alerting configurations..."
        find "${PROJECT_ROOT}/monitoring/alerting" -name "*.yaml" -o -name "*.yml" | while read -r file; do
            kubectl apply -f "$file" -n "$NAMESPACE" || warning "Failed to apply $file"
        done
    fi

    # Deploy ServiceMonitors for O-RAN components
    create_service_monitors

    success "O-RAN monitoring components deployed"
}

# Create ServiceMonitors for O-RAN components
create_service_monitors() {
    log "Creating ServiceMonitors for O-RAN components..."

    # Create ServiceMonitor for VNF Operator
    cat > /tmp/vnf-operator-servicemonitor.yaml << 'EOF'
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: vnf-operator
  namespace: monitoring
  labels:
    app: vnf-operator
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: vnf-operator
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
  namespaceSelector:
    matchNames:
    - default
    - vnf-operator-system
EOF

    kubectl apply -f /tmp/vnf-operator-servicemonitor.yaml || warning "Failed to create VNF Operator ServiceMonitor"

    # Create ServiceMonitor for Intent Management
    cat > /tmp/intent-management-servicemonitor.yaml << 'EOF'
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: intent-management
  namespace: monitoring
  labels:
    app: intent-management
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      app: intent-management
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
  namespaceSelector:
    matchNames:
    - default
    - intent-management-system
EOF

    kubectl apply -f /tmp/intent-management-servicemonitor.yaml || warning "Failed to create Intent Management ServiceMonitor"

    # Create generic ServiceMonitor for O-RAN components
    cat > /tmp/oran-components-servicemonitor.yaml << 'EOF'
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: oran-components
  namespace: monitoring
  labels:
    app: oran-components
    prometheus: kube-prometheus
spec:
  selector:
    matchLabels:
      oran.io/component: "true"
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
  - port: http-metrics
    interval: 30s
    path: /metrics
  namespaceSelector: {}
EOF

    kubectl apply -f /tmp/oran-components-servicemonitor.yaml || warning "Failed to create O-RAN Components ServiceMonitor"
}

# Wait for deployments to be ready
wait_for_deployments() {
    log "Waiting for deployments to be ready..."

    local deployments=(
        "deployment/prometheus-operator-kube-p-operator"
        "statefulset/prometheus-prometheus-operator-kube-p-prometheus"
        "deployment/prometheus-operator-grafana"
        "statefulset/alertmanager-prometheus-operator-kube-p-alertmanager"
    )

    for deployment in "${deployments[@]}"; do
        log "Waiting for $deployment..."
        kubectl wait --for=condition=Available --timeout="${TIMEOUT}s" "$deployment" -n "$NAMESPACE" || {
            error "Timeout waiting for $deployment"
            kubectl describe "$deployment" -n "$NAMESPACE"
            return 1
        }
    done

    # Wait for all pods to be ready
    log "Waiting for all pods to be ready..."
    kubectl wait --for=condition=Ready --timeout="${TIMEOUT}s" pods --all -n "$NAMESPACE"

    success "All deployments are ready"
}

# Verify monitoring stack
verify_monitoring_stack() {
    log "Verifying monitoring stack..."

    # Check Prometheus targets
    log "Checking Prometheus targets..."
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus 9090:9090 &
    local pf_pid=$!
    sleep 10

    # Test Prometheus API
    if curl -f http://localhost:9090/api/v1/targets &> /dev/null; then
        success "Prometheus API is accessible"
    else
        warning "Prometheus API is not accessible"
    fi

    kill $pf_pid 2>/dev/null || true

    # Check Grafana
    log "Checking Grafana..."
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-grafana 3000:80 &
    local gf_pid=$!
    sleep 10

    # Test Grafana API
    if curl -f http://localhost:3000/api/health &> /dev/null; then
        success "Grafana API is accessible"
    else
        warning "Grafana API is not accessible"
    fi

    kill $gf_pid 2>/dev/null || true

    # Display access information
    log "Access information:"
    echo "  Prometheus: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-prometheus 9090:9090"
    echo "  Grafana: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-grafana 3000:80"
    echo "  AlertManager: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-alertmanager 9093:9093"
    echo "  Grafana credentials: admin/admin123"
}

# Run validation tests
run_validation_tests() {
    log "Running validation tests..."

    # Check if validation script exists
    if [ -f "${PROJECT_ROOT}/monitoring/tests/ci-validation.sh" ]; then
        log "Running CI validation script..."
        chmod +x "${PROJECT_ROOT}/monitoring/tests/ci-validation.sh"
        "${PROJECT_ROOT}/monitoring/tests/ci-validation.sh"
    else
        warning "CI validation script not found, running basic checks..."

        # Basic validation
        kubectl get pods -n "$NAMESPACE"
        kubectl get services -n "$NAMESPACE"
        kubectl get servicemonitors -n "$NAMESPACE"

        # Check if Prometheus is scraping targets
        log "Checking Prometheus configuration..."
        kubectl get prometheus -n "$NAMESPACE" -o yaml | grep -A 10 "serviceMonitorSelector" || true
    fi

    success "Validation tests completed"
}

# Export cluster configuration
export_cluster_info() {
    log "Exporting cluster information..."

    mkdir -p "${PROJECT_ROOT}/deployment/test-results"

    # Export kubeconfig
    kind export kubeconfig --name "$CLUSTER_NAME" --kubeconfig "${PROJECT_ROOT}/deployment/test-results/kubeconfig"

    # Export cluster information
    cat > "${PROJECT_ROOT}/deployment/test-results/cluster-info.txt" << EOF
Cluster Name: $CLUSTER_NAME
Namespace: $NAMESPACE
Created: $(date)
Node Information:
$(kubectl get nodes -o wide)

Pod Information:
$(kubectl get pods -n "$NAMESPACE" -o wide)

Service Information:
$(kubectl get services -n "$NAMESPACE" -o wide)
EOF

    # Export access commands
    cat > "${PROJECT_ROOT}/deployment/test-results/access-commands.sh" << EOF
#!/bin/bash
# Access commands for the deployed monitoring stack

# Set kubectl context
export KUBECONFIG="${PROJECT_ROOT}/deployment/test-results/kubeconfig"

# Port forward commands
echo "Starting port forwards..."
kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-prometheus 9090:9090 &
kubectl port-forward -n $NAMESPACE svc/prometheus-operator-grafana 3000:80 &
kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-alertmanager 9093:9093 &

echo "Access URLs:"
echo "  Prometheus: http://localhost:9090"
echo "  Grafana: http://localhost:3000 (admin/admin123)"
echo "  AlertManager: http://localhost:9093"

echo "Press Ctrl+C to stop port forwards"
wait
EOF

    chmod +x "${PROJECT_ROOT}/deployment/test-results/access-commands.sh"

    success "Cluster information exported to ${PROJECT_ROOT}/deployment/test-results/"
}

# Main deployment function
main() {
    log "Starting CI deployment for O-RAN MANO monitoring stack"
    log "Cluster: $CLUSTER_NAME, Namespace: $NAMESPACE"

    check_prerequisites
    create_cluster
    install_dependencies
    install_prometheus_operator
    deploy_oran_monitoring
    wait_for_deployments
    verify_monitoring_stack
    run_validation_tests
    export_cluster_info

    success "🎉 CI deployment completed successfully!"
    log "Use the following commands to access the monitoring stack:"
    log "  Prometheus: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-prometheus 9090:9090"
    log "  Grafana: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-grafana 3000:80"
    log "  AlertManager: kubectl port-forward -n $NAMESPACE svc/prometheus-operator-kube-p-alertmanager 9093:9093"
    log "Cleanup: kind delete cluster --name $CLUSTER_NAME"
}

# Handle command line arguments
case "${1:-deploy}" in
    "deploy"|"")
        main
        ;;
    "cleanup")
        log "Cleaning up cluster: $CLUSTER_NAME"
        kind delete cluster --name "$CLUSTER_NAME" || true
        success "Cleanup completed"
        ;;
    "verify")
        verify_monitoring_stack
        ;;
    "help")
        echo "Usage: $0 [deploy|cleanup|verify|help]"
        echo "  deploy  - Deploy the monitoring stack (default)"
        echo "  cleanup - Delete the kind cluster"
        echo "  verify  - Verify the monitoring stack"
        echo "  help    - Show this help message"
        ;;
    *)
        error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac