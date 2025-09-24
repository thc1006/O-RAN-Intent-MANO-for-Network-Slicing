#!/bin/bash
# O-RAN Intent-MANO Kubernetes Deployment Script
# Deploy to Kind cluster with Kube-OVN CNI

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CONFIG="${PROJECT_ROOT}/deploy/kind/kind-cluster-config.yaml"
K8S_MANIFESTS="${PROJECT_ROOT}/deploy/k8s"
CLUSTER_NAME="oran-mano"
KUBE_OVN_VERSION="v1.12.0"
PORCH_VERSION="v0.0.31"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    local deps=("kubectl" "kind" "helm" "docker")
    local missing=()

    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            missing+=("$dep")
        fi
    done

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        log_info "Please install missing dependencies"
        exit 1
    fi

    # Check Kind version
    local kind_version=$(kind version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' | head -1)
    log_info "Kind version: $kind_version"

    # Check kubectl version
    local kubectl_version=$(kubectl version --client --short | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+')
    log_info "kubectl version: $kubectl_version"

    log_success "All dependencies are available"
}

# Create Kind cluster
create_cluster() {
    log_info "Creating Kind cluster..."

    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warn "Cluster $CLUSTER_NAME already exists"
        log_info "Delete existing cluster? (y/N)"
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            kind delete cluster --name "$CLUSTER_NAME"
        else
            log_info "Using existing cluster"
            return 0
        fi
    fi

    # Create data directory
    mkdir -p /tmp/oran-mano-data
    chmod 755 /tmp/oran-mano-data

    # Create the cluster
    kind create cluster --name "$CLUSTER_NAME" --config "$KIND_CONFIG" --wait 5m

    # Set kubectl context
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"

    log_success "Kind cluster created successfully"
}

# Install Kube-OVN CNI
install_kube_ovn() {
    log_info "Installing Kube-OVN CNI..."

    # Download Kube-OVN manifests
    local kube_ovn_url="https://raw.githubusercontent.com/kubeovn/kube-ovn/${KUBE_OVN_VERSION}/dist/images/install.sh"

    # Download and run installation script
    curl -sSL "$kube_ovn_url" | bash

    # Wait for Kube-OVN to be ready
    log_info "Waiting for Kube-OVN to be ready..."
    kubectl wait --for=condition=ready pod -l app=ovn-central -n kube-system --timeout=300s
    kubectl wait --for=condition=ready pod -l app=ovs-ovn -n kube-system --timeout=300s
    kubectl wait --for=condition=ready pod -l app=kube-ovn-controller -n kube-system --timeout=300s

    log_success "Kube-OVN CNI installed successfully"
}

# Create namespace and RBAC
setup_namespace() {
    log_info "Setting up namespace and RBAC..."

    # Create namespace
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: oran-mano
  labels:
    name: oran-mano
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
---
apiVersion: v1
kind: Namespace
metadata:
  name: oran-monitoring
  labels:
    name: oran-monitoring
---
apiVersion: v1
kind: Namespace
metadata:
  name: oran-edge
  labels:
    name: oran-edge
EOF

    # Apply RBAC configurations
    if [[ -f "$K8S_MANIFESTS/base/rbac.yaml" ]]; then
        kubectl apply -f "$K8S_MANIFESTS/base/rbac.yaml"
        log_info "Applied RBAC configurations"
    fi

    # Apply network policies
    if [[ -f "$K8S_MANIFESTS/base/network-policies.yaml" ]]; then
        kubectl apply -f "$K8S_MANIFESTS/base/network-policies.yaml"
        log_info "Applied network policies"
    fi

    log_success "Namespace and RBAC setup completed"
}

# Install Prometheus and Grafana
install_monitoring() {
    log_info "Installing monitoring stack..."

    # Add Prometheus Helm repository
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo add grafana https://grafana.github.io/helm-charts
    helm repo update

    # Install Prometheus
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace oran-monitoring \
        --create-namespace \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.podMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.retention=7d \
        --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=10Gi \
        --set grafana.adminPassword=admin123 \
        --set grafana.service.type=NodePort \
        --set grafana.service.nodePort=30000 \
        --set prometheus.service.type=NodePort \
        --set prometheus.service.nodePort=30090 \
        --wait

    log_success "Monitoring stack installed"
}

# Install Porch (for GitOps)
install_porch() {
    log_info "Installing Porch..."

    # Create Porch namespace
    kubectl create namespace porch-system --dry-run=client -o yaml | kubectl apply -f -

    # Apply Porch manifests
    kubectl apply -f "https://github.com/nephio-project/porch/releases/download/${PORCH_VERSION}/porch-controllers.yaml"
    kubectl apply -f "https://github.com/nephio-project/porch/releases/download/${PORCH_VERSION}/porch-server.yaml"

    # Wait for Porch to be ready
    kubectl wait --for=condition=available deployment/porch-controllers -n porch-system --timeout=300s
    kubectl wait --for=condition=available deployment/porch-server -n porch-system --timeout=300s

    log_success "Porch installed successfully"
}

# Build and load Docker images
build_and_load_images() {
    log_info "Building and loading Docker images..."

    cd "$PROJECT_ROOT"

    # List of services to build
    local services=(
        "orchestrator"
        "vnf-operator"
        "o2-client"
        "tn-manager"
        "tn-agent"
        "ran-dms"
        "cn-dms"
    )

    for service in "${services[@]}"; do
        log_info "Building $service image..."

        local dockerfile="deploy/docker/$service/Dockerfile"
        if [[ "$service" == "vnf-operator" ]]; then
            dockerfile="deploy/docker/$service/Dockerfile.go1.24.7"
        fi

        docker build -t "oran-$service:latest" -f "$dockerfile" \
            --build-arg GO_VERSION=1.24.7 \
            --build-arg BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
            --build-arg VERSION="v1.0.0-k8s" \
            --build-arg COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')" \
            .

        # Load image into Kind cluster
        kind load docker-image "oran-$service:latest" --name "$CLUSTER_NAME"

        log_info "$service image built and loaded"
    done

    log_success "All images built and loaded successfully"
}

# Generate certificates and secrets
create_certificates() {
    log_info "Creating certificates and secrets..."

    local cert_dir="/tmp/oran-certs"
    mkdir -p "$cert_dir"

    # Generate CA certificate
    openssl genpkey -algorithm RSA -out "$cert_dir/ca.key"
    openssl req -new -x509 -key "$cert_dir/ca.key" -out "$cert_dir/ca.crt" \
        -days 365 -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN MANO/CN=ca"

    # Generate service certificates
    local services=("orchestrator" "ran-dms" "cn-dms" "vnf-operator")

    for service in "${services[@]}"; do
        openssl genpkey -algorithm RSA -out "$cert_dir/$service.key"
        openssl req -new -key "$cert_dir/$service.key" -out "$cert_dir/$service.csr" \
            -subj "/C=US/ST=CA/L=San Francisco/O=O-RAN MANO/CN=$service"
        openssl x509 -req -in "$cert_dir/$service.csr" -CA "$cert_dir/ca.crt" \
            -CAkey "$cert_dir/ca.key" -CAcreateserial -out "$cert_dir/$service.crt" -days 365
        rm "$cert_dir/$service.csr"
    done

    # Create Kubernetes secrets
    kubectl create secret tls oran-tls-secret \
        --cert="$cert_dir/ca.crt" \
        --key="$cert_dir/ca.key" \
        --namespace=oran-mano \
        --dry-run=client -o yaml | kubectl apply -f -

    kubectl create secret tls oran-vnf-operator-webhook-certs \
        --cert="$cert_dir/vnf-operator.crt" \
        --key="$cert_dir/vnf-operator.key" \
        --namespace=oran-mano \
        --dry-run=client -o yaml | kubectl apply -f -

    # Clean up temp directory
    rm -rf "$cert_dir"

    log_success "Certificates and secrets created"
}

# Deploy ORAN services
deploy_services() {
    log_info "Deploying O-RAN MANO services..."

    # Apply base manifests
    kubectl apply -f "$K8S_MANIFESTS/base/namespace.yaml" || true

    # Deploy services in order
    local services=(
        "ran-dms"
        "cn-dms"
        "orchestrator"
        "vnf-operator"
    )

    for service in "${services[@]}"; do
        if [[ -f "$K8S_MANIFESTS/base/$service.yaml" ]]; then
            log_info "Deploying $service..."
            kubectl apply -f "$K8S_MANIFESTS/base/$service.yaml"

            # Wait for deployment to be ready
            kubectl wait --for=condition=available deployment/oran-$service -n oran-mano --timeout=300s || true
        else
            log_warn "Manifest not found for $service: $K8S_MANIFESTS/base/$service.yaml"
        fi
    done

    # Create TN services (these don't have manifests yet, create them)
    create_tn_services

    log_success "O-RAN MANO services deployed"
}

# Create TN services
create_tn_services() {
    log_info "Creating TN services..."

    # TN Manager
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-tn-manager
  namespace: oran-mano
  labels:
    app: oran-tn-manager
spec:
  replicas: 1
  selector:
    matchLabels:
      app: oran-tn-manager
  template:
    metadata:
      labels:
        app: oran-tn-manager
    spec:
      containers:
      - name: tn-manager
        image: oran-tn-manager:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: ORCHESTRATOR_ENDPOINT
          value: "http://oran-orchestrator:8080"
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        securityContext:
          runAsNonRoot: true
          runAsUser: 65532
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: ["ALL"]
            add: ["NET_ADMIN", "NET_RAW"]
---
apiVersion: v1
kind: Service
metadata:
  name: oran-tn-manager
  namespace: oran-mano
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30084
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
  selector:
    app: oran-tn-manager
EOF

    # TN Agents
    for edge in "01" "02"; do
        local node_port=$((30084 + ${edge#0}))
        local iperf_port=$((30200 + ${edge#0}))

        kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-tn-agent-edge${edge}
  namespace: oran-edge
  labels:
    app: oran-tn-agent-edge${edge}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: oran-tn-agent-edge${edge}
  template:
    metadata:
      labels:
        app: oran-tn-agent-edge${edge}
    spec:
      containers:
      - name: tn-agent
        image: oran-tn-agent:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 5201
          name: iperf
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: NODE_ID
          value: "edge${edge}"
        - name: TN_MANAGER_ENDPOINT
          value: "http://oran-tn-manager.oran-mano:8080"
        - name: SITE_LOCATION
          value: "edge"
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        securityContext:
          runAsNonRoot: true
          runAsUser: 65532
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: ["ALL"]
            add: ["NET_ADMIN", "NET_RAW"]
---
apiVersion: v1
kind: Service
metadata:
  name: oran-tn-agent-edge${edge}
  namespace: oran-edge
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: ${node_port}
    name: http
  - port: 5201
    targetPort: 5201
    nodePort: ${iperf_port}
    name: iperf
  selector:
    app: oran-tn-agent-edge${edge}
EOF
    done

    # O2 Client
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-o2-client
  namespace: oran-mano
  labels:
    app: oran-o2-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app: oran-o2-client
  template:
    metadata:
      labels:
        app: oran-o2-client
    spec:
      containers:
      - name: o2-client
        image: oran-o2-client:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: LOG_LEVEL
          value: "debug"
        - name: O2IMS_ENDPOINT
          value: "http://localhost:5005"
        - name: O2DMS_ENDPOINT
          value: "http://oran-ran-dms:8080"
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        securityContext:
          runAsNonRoot: true
          runAsUser: 65532
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop: ["ALL"]
---
apiVersion: v1
kind: Service
metadata:
  name: oran-o2-client
  namespace: oran-mano
spec:
  type: NodePort
  ports:
  - port: 8080
    targetPort: 8080
    nodePort: 30083
    name: http
  - port: 9090
    targetPort: 9090
    name: metrics
  selector:
    app: oran-o2-client
EOF

    log_success "TN services created"
}

# Wait for all services to be ready
wait_for_services() {
    log_info "Waiting for all services to be ready..."

    # Wait for deployments to be available
    local deployments=(
        "oran-ran-dms"
        "oran-cn-dms"
        "oran-orchestrator"
        "oran-vnf-operator"
        "oran-tn-manager"
        "oran-o2-client"
    )

    for deployment in "${deployments[@]}"; do
        log_info "Waiting for $deployment..."
        kubectl wait --for=condition=available deployment/"$deployment" -n oran-mano --timeout=300s || log_warn "$deployment not ready"
    done

    # Wait for TN agents
    kubectl wait --for=condition=available deployment/oran-tn-agent-edge01 -n oran-edge --timeout=300s || log_warn "tn-agent-edge01 not ready"
    kubectl wait --for=condition=available deployment/oran-tn-agent-edge02 -n oran-edge --timeout=300s || log_warn "tn-agent-edge02 not ready"

    # Wait for pods to be ready
    kubectl wait --for=condition=ready pod -l app=oran-orchestrator -n oran-mano --timeout=180s || log_warn "Orchestrator pod not ready"
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=oran-vnf-operator -n oran-mano --timeout=180s || log_warn "VNF operator pod not ready"

    log_success "Services deployment completed"
}

# Run health checks
run_health_checks() {
    log_info "Running health checks..."

    # Check cluster status
    kubectl get nodes
    kubectl get pods -A

    # Test service endpoints
    local services=(
        "oran-orchestrator:8080:/health"
        "oran-ran-dms:8080:/health"
        "oran-cn-dms:8080:/health"
    )

    for service_info in "${services[@]}"; do
        IFS=':' read -r service port path <<< "$service_info"

        log_info "Testing $service endpoint..."
        if kubectl exec -n oran-mano deployment/"$service" -- wget -qO- --timeout=5 "http://localhost:${port}${path}" >/dev/null 2>&1; then
            log_success "$service health check passed"
        else
            log_warn "$service health check failed"
        fi
    done
}

# Show service information
show_service_info() {
    log_info "Service Information:"
    echo "┌─────────────────────────────────────────────────────────────────┐"
    echo "│                    O-RAN MANO Kubernetes Services               │"
    echo "├─────────────────────────────────────────────────────────────────┤"
    echo "│ Orchestrator:      http://localhost:8080                       │"
    echo "│ VNF Operator:      http://localhost:8081 (metrics)             │"
    echo "│ O2 Client:         http://localhost:8083                       │"
    echo "│ TN Manager:        http://localhost:8084                       │"
    echo "│ TN Agent Edge01:   http://localhost:8085                       │"
    echo "│ TN Agent Edge02:   http://localhost:8086                       │"
    echo "│ RAN DMS:           http://localhost:8087                       │"
    echo "│ CN DMS:            http://localhost:8088                       │"
    echo "├─────────────────────────────────────────────────────────────────┤"
    echo "│ Prometheus:        http://localhost:9090                       │"
    echo "│ Grafana:           http://localhost:3000                       │"
    echo "│   - User: admin / Password: admin123                           │"
    echo "└─────────────────────────────────────────────────────────────────┘"
    echo ""
    echo "Useful Commands:"
    echo "  kubectl get pods -n oran-mano"
    echo "  kubectl logs -f deployment/oran-orchestrator -n oran-mano"
    echo "  kubectl port-forward -n oran-mano svc/oran-orchestrator 8080:8080"
    echo ""
}

# Cleanup function
cleanup_cluster() {
    log_info "Cleaning up cluster..."

    kind delete cluster --name "$CLUSTER_NAME"
    rm -rf /tmp/oran-mano-data

    log_success "Cluster cleaned up"
}

# Show usage
show_usage() {
    cat << 'EOF'
O-RAN Intent-MANO Kubernetes Deployment Script

Usage: ./deploy-kubernetes.sh [COMMAND] [OPTIONS]

Commands:
  deploy         Deploy complete system (default)
  create         Create Kind cluster only
  build          Build and load images only
  install        Install services only
  clean          Delete cluster and cleanup
  health         Run health checks
  info           Show service information

Options:
  -h, --help     Show this help message
  -v, --verbose  Enable verbose output
  --skip-build   Skip building Docker images
  --skip-cni     Skip CNI installation

Examples:
  ./deploy-kubernetes.sh deploy
  ./deploy-kubernetes.sh create
  ./deploy-kubernetes.sh clean
EOF
}

# Main execution
main() {
    local command="${1:-deploy}"
    shift || true

    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -v|--verbose)
                set -x
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --skip-cni)
                SKIP_CNI=true
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    case "$command" in
        deploy)
            check_dependencies
            create_cluster
            [[ "${SKIP_CNI:-}" != "true" ]] && install_kube_ovn
            setup_namespace
            install_monitoring
            install_porch
            create_certificates
            [[ "${SKIP_BUILD:-}" != "true" ]] && build_and_load_images
            deploy_services
            wait_for_services
            run_health_checks
            show_service_info
            log_success "🎉 O-RAN MANO deployed successfully to Kubernetes!"
            ;;
        create)
            check_dependencies
            create_cluster
            [[ "${SKIP_CNI:-}" != "true" ]] && install_kube_ovn
            setup_namespace
            ;;
        build)
            check_dependencies
            build_and_load_images
            ;;
        install)
            check_dependencies
            install_monitoring
            install_porch
            create_certificates
            deploy_services
            wait_for_services
            ;;
        clean)
            cleanup_cluster
            ;;
        health)
            run_health_checks
            ;;
        info)
            show_service_info
            ;;
        *)
            log_error "Unknown command: $command"
            show_usage
            exit 1
            ;;
    esac
}

main "$@"