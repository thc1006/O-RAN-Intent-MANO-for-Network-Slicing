#!/bin/bash
# O-RAN Intent-MANO Deployment Script
# Deploys all MANO components across multi-cluster environment

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
TEMP_DIR="/tmp/oran-mano-deploy"

# Configuration
CLUSTERS=("central" "edge01" "edge02" "regional")
HELM_TIMEOUT="600s"
IMAGE_TAG="${IMAGE_TAG:-latest}"
IMAGE_PREFIX="${IMAGE_PREFIX:-oran}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log() { echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $*${NC}"; }
warn() { echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $*${NC}"; }
error() { echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $*${NC}"; exit 1; }
info() { echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $*${NC}"; }

# Check prerequisites
check_prerequisites() {
    log "Checking deployment prerequisites..."

    # Check for required tools
    for tool in kubectl helm; do
        if ! command -v "$tool" &> /dev/null; then
            error "$tool is not installed or not in PATH"
        fi
    done

    # Check KUBECONFIG
    if [ -z "${KUBECONFIG:-}" ]; then
        error "KUBECONFIG environment variable not set"
    fi

    # Verify cluster connectivity
    for cluster in "${CLUSTERS[@]}"; do
        if ! kubectl config get-contexts | grep -q "kind-$cluster"; then
            error "Cluster context kind-$cluster not found"
        fi
    done

    log "Prerequisites check passed"
}

# Setup Helm repositories
setup_helm_repos() {
    log "Setting up Helm repositories..."

    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts 2>/dev/null || true
    helm repo add grafana https://grafana.github.io/helm-charts 2>/dev/null || true
    helm repo add istio https://istio-release.storage.googleapis.com/charts 2>/dev/null || true
    helm repo add jetstack https://charts.jetstack.io 2>/dev/null || true
    helm repo update

    log "Helm repositories configured"
}

# Create namespaces across clusters
create_namespaces() {
    log "Creating namespaces across clusters..."

    for cluster in "${CLUSTERS[@]}"; do
        kubectl config use-context "kind-$cluster"

        # Apply namespace definitions
        kubectl apply -f "$PROJECT_ROOT/deploy/k8s/base/namespace.yaml"

        info "Namespaces created on cluster: $cluster"
    done

    log "Namespaces creation completed"
}

# Deploy RBAC across clusters
deploy_rbac() {
    log "Deploying RBAC configurations..."

    for cluster in "${CLUSTERS[@]}"; do
        kubectl config use-context "kind-$cluster"

        # Apply RBAC definitions
        kubectl apply -f "$PROJECT_ROOT/deploy/k8s/base/rbac.yaml"

        info "RBAC deployed on cluster: $cluster"
    done

    log "RBAC deployment completed"
}

# Deploy core MANO components on central cluster
deploy_core_components() {
    log "Deploying core MANO components on central cluster..."

    kubectl config use-context "kind-central"

    # Deploy Orchestrator
    helm upgrade --install oran-orchestrator \
        "$PROJECT_ROOT/deploy/helm/charts/orchestrator" \
        --namespace oran-mano \
        --set image.repository="$IMAGE_PREFIX-orchestrator" \
        --set image.tag="$IMAGE_TAG" \
        --set config.integrations.o2dms.ran_endpoint="http://oran-ran-dms.oran-edge:8080" \
        --set config.integrations.o2dms.cn_endpoint="http://oran-cn-dms.oran-core:8080" \
        --wait --timeout="$HELM_TIMEOUT"

    # Deploy VNF Operator
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-vnf-operator
  namespace: oran-mano
  labels:
    app.kubernetes.io/name: oran-vnf-operator
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-vnf-operator
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-vnf-operator
    spec:
      serviceAccountName: oran-vnf-operator
      containers:
      - name: vnf-operator
        image: $IMAGE_PREFIX-vnf-operator:$IMAGE_TAG
        args:
        - --metrics-bind-address=0.0.0.0:8080
        - --health-probe-bind-address=0.0.0.0:8081
        - --leader-elect=true
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8081
          name: health
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
EOF

    # Deploy O2 Client
    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-o2-client
  namespace: oran-mano
  labels:
    app.kubernetes.io/name: oran-o2-client
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-o2-client
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-o2-client
    spec:
      serviceAccountName: oran-o2-client
      containers:
      - name: o2-client
        image: $IMAGE_PREFIX-o2-client:$IMAGE_TAG
        ports:
        - containerPort: 8080
          name: http
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: oran-o2-client
  namespace: oran-mano
spec:
  selector:
    app.kubernetes.io/name: oran-o2-client
  ports:
  - port: 8080
    targetPort: http
EOF

    # Wait for core components to be ready
    kubectl wait --for=condition=Available deployment/oran-orchestrator -n oran-mano --timeout="$HELM_TIMEOUT"
    kubectl wait --for=condition=Available deployment/oran-vnf-operator -n oran-mano --timeout="$HELM_TIMEOUT"
    kubectl wait --for=condition=Available deployment/oran-o2-client -n oran-mano --timeout="$HELM_TIMEOUT"

    log "Core MANO components deployed successfully"
}

# Deploy TN Manager on central cluster
deploy_tn_manager() {
    log "Deploying TN Manager on central cluster..."

    kubectl config use-context "kind-central"

    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-tn-manager
  namespace: oran-mano
  labels:
    app.kubernetes.io/name: oran-tn-manager
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-tn-manager
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-tn-manager
    spec:
      serviceAccountName: oran-tn-manager
      containers:
      - name: tn-manager
        image: $IMAGE_PREFIX-tn-manager:$IMAGE_TAG
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
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
          capabilities:
            add:
            - NET_ADMIN
---
apiVersion: v1
kind: Service
metadata:
  name: oran-tn-manager
  namespace: oran-mano
spec:
  selector:
    app.kubernetes.io/name: oran-tn-manager
  ports:
  - name: http
    port: 8080
    targetPort: http
  - name: metrics
    port: 9090
    targetPort: metrics
EOF

    kubectl wait --for=condition=Available deployment/oran-tn-manager -n oran-mano --timeout="$HELM_TIMEOUT"

    log "TN Manager deployed successfully"
}

# Deploy TN Agents on edge clusters
deploy_tn_agents() {
    log "Deploying TN Agents on edge clusters..."

    for cluster in edge01 edge02; do
        kubectl config use-context "kind-$cluster"

        kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: oran-tn-agent
  namespace: oran-edge
  labels:
    app.kubernetes.io/name: oran-tn-agent
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-tn-agent
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-tn-agent
    spec:
      serviceAccountName: oran-tn-agent
      hostNetwork: true
      containers:
      - name: tn-agent
        image: $IMAGE_PREFIX-tn-agent:$IMAGE_TAG
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: metrics
        env:
        - name: NODE_ID
          value: "$cluster"
        - name: TN_MANAGER_ENDPOINT
          value: "http://oran-tn-manager.oran-mano.svc.cluster.local:8080"
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
          privileged: true
          capabilities:
            add:
            - NET_ADMIN
            - NET_RAW
        volumeMounts:
        - name: host-proc
          mountPath: /host/proc
          readOnly: true
        - name: host-sys
          mountPath: /host/sys
          readOnly: true
      volumes:
      - name: host-proc
        hostPath:
          path: /proc
      - name: host-sys
        hostPath:
          path: /sys
      tolerations:
      - effect: NoSchedule
        operator: Exists
---
apiVersion: v1
kind: Service
metadata:
  name: oran-tn-agent
  namespace: oran-edge
spec:
  selector:
    app.kubernetes.io/name: oran-tn-agent
  ports:
  - name: http
    port: 8080
    targetPort: http
  - name: metrics
    port: 9090
    targetPort: metrics
EOF

        # Wait for TN agents to be ready
        kubectl wait --for=condition=Ready pods -l app.kubernetes.io/name=oran-tn-agent -n oran-edge --timeout="$HELM_TIMEOUT"

        info "TN Agent deployed on cluster: $cluster"
    done

    log "TN Agents deployment completed"
}

# Deploy DMS components
deploy_dms_components() {
    log "Deploying DMS components..."

    # Deploy RAN DMS on edge clusters
    for cluster in edge01 edge02; do
        kubectl config use-context "kind-$cluster"

        kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-ran-dms
  namespace: oran-edge
  labels:
    app.kubernetes.io/name: oran-ran-dms
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-ran-dms
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-ran-dms
    spec:
      containers:
      - name: ran-dms
        image: $IMAGE_PREFIX-ran-dms:$IMAGE_TAG
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8443
          name: https
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: oran-ran-dms
  namespace: oran-edge
spec:
  selector:
    app.kubernetes.io/name: oran-ran-dms
  ports:
  - name: http
    port: 8080
    targetPort: http
  - name: https
    port: 8443
    targetPort: https
EOF

        kubectl wait --for=condition=Available deployment/oran-ran-dms -n oran-edge --timeout="$HELM_TIMEOUT"
        info "RAN DMS deployed on cluster: $cluster"
    done

    # Deploy CN DMS on regional cluster
    kubectl config use-context "kind-regional"

    kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: oran-cn-dms
  namespace: oran-core
  labels:
    app.kubernetes.io/name: oran-cn-dms
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: oran-cn-dms
  template:
    metadata:
      labels:
        app.kubernetes.io/name: oran-cn-dms
    spec:
      containers:
      - name: cn-dms
        image: $IMAGE_PREFIX-cn-dms:$IMAGE_TAG
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8443
          name: https
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
---
apiVersion: v1
kind: Service
metadata:
  name: oran-cn-dms
  namespace: oran-core
spec:
  selector:
    app.kubernetes.io/name: oran-cn-dms
  ports:
  - name: http
    port: 8080
    targetPort: http
  - name: https
    port: 8443
    targetPort: https
EOF

    kubectl wait --for=condition=Available deployment/oran-cn-dms -n oran-core --timeout="$HELM_TIMEOUT"

    log "DMS components deployed successfully"
}

# Deploy monitoring stack
deploy_monitoring() {
    log "Deploying monitoring stack on central cluster..."

    kubectl config use-context "kind-central"

    # Deploy Prometheus
    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace oran-monitoring \
        --set prometheus.prometheusSpec.serviceMonitorSelectorNilUsesHelmValues=false \
        --set prometheus.prometheusSpec.retention=7d \
        --set grafana.adminPassword=admin \
        --set grafana.service.type=NodePort \
        --set grafana.service.nodePort=30300 \
        --wait --timeout="$HELM_TIMEOUT"

    log "Monitoring stack deployed successfully"
}

# Verify deployment
verify_deployment() {
    log "Verifying deployment across all clusters..."

    # Central cluster verification
    kubectl config use-context "kind-central"
    info "Verifying central cluster components..."

    for component in oran-orchestrator oran-vnf-operator oran-o2-client oran-tn-manager; do
        if kubectl get deployment "$component" -n oran-mano >/dev/null 2>&1; then
            local ready=$(kubectl get deployment "$component" -n oran-mano -o jsonpath='{.status.readyReplicas}')
            local desired=$(kubectl get deployment "$component" -n oran-mano -o jsonpath='{.spec.replicas}')
            if [ "$ready" = "$desired" ]; then
                info "✓ $component: $ready/$desired replicas ready"
            else
                warn "✗ $component: $ready/$desired replicas ready"
            fi
        else
            warn "✗ $component: deployment not found"
        fi
    done

    # Edge clusters verification
    for cluster in edge01 edge02; do
        kubectl config use-context "kind-$cluster"
        info "Verifying $cluster cluster components..."

        # Check TN Agent
        local tn_agent_ready=$(kubectl get daemonset oran-tn-agent -n oran-edge -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")
        local tn_agent_desired=$(kubectl get daemonset oran-tn-agent -n oran-edge -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")
        if [ "$tn_agent_ready" = "$tn_agent_desired" ] && [ "$tn_agent_ready" -gt 0 ]; then
            info "✓ TN Agent: $tn_agent_ready/$tn_agent_desired pods ready"
        else
            warn "✗ TN Agent: $tn_agent_ready/$tn_agent_desired pods ready"
        fi

        # Check RAN DMS
        local ran_dms_ready=$(kubectl get deployment oran-ran-dms -n oran-edge -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local ran_dms_desired=$(kubectl get deployment oran-ran-dms -n oran-edge -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
        if [ "$ran_dms_ready" = "$ran_dms_desired" ] && [ "$ran_dms_ready" -gt 0 ]; then
            info "✓ RAN DMS: $ran_dms_ready/$ran_dms_desired replicas ready"
        else
            warn "✗ RAN DMS: $ran_dms_ready/$ran_dms_desired replicas ready"
        fi
    done

    # Regional cluster verification
    kubectl config use-context "kind-regional"
    info "Verifying regional cluster components..."

    local cn_dms_ready=$(kubectl get deployment oran-cn-dms -n oran-core -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    local cn_dms_desired=$(kubectl get deployment oran-cn-dms -n oran-core -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")
    if [ "$cn_dms_ready" = "$cn_dms_desired" ] && [ "$cn_dms_ready" -gt 0 ]; then
        info "✓ CN DMS: $cn_dms_ready/$cn_dms_desired replicas ready"
    else
        warn "✗ CN DMS: $cn_dms_ready/$cn_dms_desired replicas ready"
    fi

    log "Deployment verification completed"
}

# Generate deployment report
generate_deployment_report() {
    log "Generating deployment report..."

    local report_file="$TEMP_DIR/deployment-report.json"
    mkdir -p "$TEMP_DIR"

    cat > "$report_file" <<EOF
{
  "deployment": {
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
    "image_tag": "$IMAGE_TAG",
    "clusters": [
EOF

    # Collect deployment status from all clusters
    local first=true
    for cluster in "${CLUSTERS[@]}"; do
        kubectl config use-context "kind-$cluster"

        if [ "$first" = true ]; then
            first=false
        else
            echo "," >> "$report_file"
        fi

        local components_json="[]"
        case "$cluster" in
            "central")
                components_json='[
                  {
                    "name": "oran-orchestrator",
                    "namespace": "oran-mano",
                    "ready": '$(kubectl get deployment oran-orchestrator -n oran-mano -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")',
                    "desired": '$(kubectl get deployment oran-orchestrator -n oran-mano -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")'
                  },
                  {
                    "name": "oran-vnf-operator",
                    "namespace": "oran-mano",
                    "ready": '$(kubectl get deployment oran-vnf-operator -n oran-mano -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")',
                    "desired": '$(kubectl get deployment oran-vnf-operator -n oran-mano -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")'
                  }
                ]'
                ;;
            "edge"*)
                components_json='[
                  {
                    "name": "oran-tn-agent",
                    "namespace": "oran-edge",
                    "ready": '$(kubectl get daemonset oran-tn-agent -n oran-edge -o jsonpath='{.status.numberReady}' 2>/dev/null || echo "0")',
                    "desired": '$(kubectl get daemonset oran-tn-agent -n oran-edge -o jsonpath='{.status.desiredNumberScheduled}' 2>/dev/null || echo "0")'
                  },
                  {
                    "name": "oran-ran-dms",
                    "namespace": "oran-edge",
                    "ready": '$(kubectl get deployment oran-ran-dms -n oran-edge -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")',
                    "desired": '$(kubectl get deployment oran-ran-dms -n oran-edge -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")'
                  }
                ]'
                ;;
            "regional")
                components_json='[
                  {
                    "name": "oran-cn-dms",
                    "namespace": "oran-core",
                    "ready": '$(kubectl get deployment oran-cn-dms -n oran-core -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")',
                    "desired": '$(kubectl get deployment oran-cn-dms -n oran-core -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")'
                  }
                ]'
                ;;
        esac

        cat >> "$report_file" <<EOF
      {
        "name": "$cluster",
        "context": "kind-$cluster",
        "components": $components_json
      }
EOF
    done

    cat >> "$report_file" <<EOF
    ]
  }
}
EOF

    log "Deployment report generated: $report_file"
}

# Main execution
main() {
    log "Starting O-RAN Intent-MANO deployment"

    # Parse command line arguments
    local deploy_monitoring=false
    local skip_verification=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --with-monitoring)
                deploy_monitoring=true
                shift
                ;;
            --skip-verification)
                skip_verification=true
                shift
                ;;
            --image-tag)
                IMAGE_TAG="$2"
                shift 2
                ;;
            --image-prefix)
                IMAGE_PREFIX="$2"
                shift 2
                ;;
            --help|-h)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --with-monitoring   Deploy monitoring stack"
                echo "  --skip-verification Skip deployment verification"
                echo "  --image-tag TAG     Container image tag (default: latest)"
                echo "  --image-prefix PREFIX Container image prefix (default: oran)"
                echo "  --help, -h          Show this help message"
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                ;;
        esac
    done

    # Execute deployment steps
    check_prerequisites
    setup_helm_repos
    create_namespaces
    deploy_rbac
    deploy_core_components
    deploy_tn_manager
    deploy_tn_agents
    deploy_dms_components

    if [ "$deploy_monitoring" = true ]; then
        deploy_monitoring
    fi

    if [ "$skip_verification" = false ]; then
        verify_deployment
    fi

    generate_deployment_report

    log "O-RAN Intent-MANO deployment completed successfully!"
    echo ""
    info "Access points:"
    echo "  - Orchestrator API: kubectl port-forward -n oran-mano service/oran-orchestrator 8080:8080"
    echo "  - Grafana (if deployed): kubectl port-forward -n oran-monitoring service/prometheus-grafana 3000:80"
    echo ""
    info "Next steps:"
    echo "1. Run integration tests: $PROJECT_ROOT/deploy/scripts/test/run_integration_tests.sh"
    echo "2. Run performance tests: $PROJECT_ROOT/deploy/scripts/test/run_performance_tests.sh"
    echo "3. Check deployment report: $TEMP_DIR/deployment-report.json"
}

# Execute main function
main "$@"