#!/bin/bash

set -euo pipefail

# Deploy complete monitoring stack for O-RAN MANO
# This script deploys Prometheus Operator, ServiceMonitors, Grafana, and AlertManager

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MONITORING_DIR="$PROJECT_ROOT/monitoring"
DEPLOYMENT_DIR="$PROJECT_ROOT/deployment"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROMETHEUS_NAMESPACE="oran-monitoring"
SYSTEM_NAMESPACE="oran-system"
OBSERVABILITY_NAMESPACE="oran-observability"
HELM_RELEASE_NAME="oran-monitoring"
PROMETHEUS_CHART_VERSION="25.8.0"
GRAFANA_CHART_VERSION="7.0.8"

# Functions
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

wait_for_pods() {
    local namespace=$1
    local label_selector=$2
    local timeout=${3:-300}

    log_info "Waiting for pods in namespace $namespace with selector $label_selector..."
    if kubectl wait --for=condition=Ready pods -l "$label_selector" -n "$namespace" --timeout="${timeout}s"; then
        log_success "Pods are ready in namespace $namespace"
    else
        log_error "Timeout waiting for pods in namespace $namespace"
        return 1
    fi
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is available and configured
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        return 1
    fi

    # Check if helm is available
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed or not in PATH"
        return 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        return 1
    fi

    log_success "Prerequisites check passed"
}

create_namespaces() {
    log_info "Creating namespaces..."

    # Apply namespace manifests
    kubectl apply -f "$DEPLOYMENT_DIR/kubernetes/namespaces.yaml"

    # Wait for namespaces to be ready
    kubectl wait --for=condition=Ready namespace/$PROMETHEUS_NAMESPACE --timeout=60s
    kubectl wait --for=condition=Ready namespace/$SYSTEM_NAMESPACE --timeout=60s
    kubectl wait --for=condition=Ready namespace/$OBSERVABILITY_NAMESPACE --timeout=60s

    log_success "Namespaces created successfully"
}

add_helm_repositories() {
    log_info "Adding required Helm repositories..."

    # Add Prometheus Community repository
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

    # Add Grafana repository
    helm repo add grafana https://grafana.github.io/helm-charts

    # Update repositories
    helm repo update

    log_success "Helm repositories added and updated"
}

install_prometheus_operator() {
    log_info "Installing Prometheus Operator..."

    # Install Prometheus Operator using Helm
    helm upgrade --install prometheus-operator prometheus-community/kube-prometheus-stack \
        --namespace "$PROMETHEUS_NAMESPACE" \
        --values "$MONITORING_DIR/prometheus/values.yaml" \
        --version "$PROMETHEUS_CHART_VERSION" \
        --wait \
        --timeout 600s

    # Wait for Prometheus Operator to be ready
    wait_for_pods "$PROMETHEUS_NAMESPACE" "app.kubernetes.io/name=prometheus-operator" 300

    log_success "Prometheus Operator installed successfully"
}

deploy_prometheus_rules() {
    log_info "Deploying Prometheus alerting rules..."

    # Apply PrometheusRule CRD
    kubectl apply -f "$MONITORING_DIR/prometheus/prometheus-rules.yaml"

    log_success "Prometheus rules deployed successfully"
}

deploy_service_monitors() {
    log_info "Deploying ServiceMonitors..."

    # Deploy all ServiceMonitors
    for servicemonitor in "$MONITORING_DIR/prometheus/servicemonitors"/*.yaml; do
        if [[ -f "$servicemonitor" ]]; then
            log_info "Applying $(basename "$servicemonitor")..."
            kubectl apply -f "$servicemonitor"
        fi
    done

    log_success "ServiceMonitors deployed successfully"
}

create_grafana_dashboards_configmap() {
    log_info "Creating Grafana dashboards ConfigMap..."

    # Create ConfigMap with all dashboard JSON files
    kubectl create configmap oran-grafana-dashboards \
        --from-file="$MONITORING_DIR/grafana/dashboards/" \
        --namespace="$PROMETHEUS_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -

    log_success "Grafana dashboards ConfigMap created"
}

deploy_grafana() {
    log_info "Deploying Grafana..."

    # Create dashboards ConfigMap first
    create_grafana_dashboards_configmap

    # Deploy Grafana
    kubectl apply -f "$MONITORING_DIR/grafana/deployment.yaml"

    # Wait for Grafana to be ready
    wait_for_pods "$PROMETHEUS_NAMESPACE" "app=grafana" 300

    log_success "Grafana deployed successfully"
}

configure_alertmanager() {
    log_info "Configuring AlertManager..."

    # Create AlertManager configuration secret
    kubectl create secret generic alertmanager-oran-monitoring-kube-prometheus-alertmanager \
        --from-file=alertmanager.yml="$MONITORING_DIR/alertmanager/config.yaml" \
        --namespace="$PROMETHEUS_NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -

    # Restart AlertManager to pick up new configuration
    kubectl rollout restart statefulset/alertmanager-oran-monitoring-kube-prometheus-alertmanager \
        -n "$PROMETHEUS_NAMESPACE" || true

    log_success "AlertManager configured successfully"
}

validate_deployment() {
    log_info "Validating deployment..."

    # Check if Prometheus is accessible
    log_info "Checking Prometheus..."
    if kubectl get prometheus -n "$PROMETHEUS_NAMESPACE" &> /dev/null; then
        log_success "Prometheus is deployed"
    else
        log_error "Prometheus is not accessible"
        return 1
    fi

    # Check if Grafana is accessible
    log_info "Checking Grafana..."
    if kubectl get deployment grafana -n "$PROMETHEUS_NAMESPACE" &> /dev/null; then
        log_success "Grafana is deployed"
    else
        log_error "Grafana is not accessible"
        return 1
    fi

    # Check if AlertManager is accessible
    log_info "Checking AlertManager..."
    if kubectl get alertmanager -n "$PROMETHEUS_NAMESPACE" &> /dev/null; then
        log_success "AlertManager is deployed"
    else
        log_error "AlertManager is not accessible"
        return 1
    fi

    # Check ServiceMonitors
    log_info "Checking ServiceMonitors..."
    local servicemonitor_count
    servicemonitor_count=$(kubectl get servicemonitors -n "$PROMETHEUS_NAMESPACE" --no-headers | wc -l)
    if [[ $servicemonitor_count -gt 0 ]]; then
        log_success "Found $servicemonitor_count ServiceMonitors"
    else
        log_warning "No ServiceMonitors found"
    fi

    # Check PrometheusRules
    log_info "Checking PrometheusRules..."
    local rules_count
    rules_count=$(kubectl get prometheusrules -n "$PROMETHEUS_NAMESPACE" --no-headers | wc -l)
    if [[ $rules_count -gt 0 ]]; then
        log_success "Found $rules_count PrometheusRules"
    else
        log_warning "No PrometheusRules found"
    fi

    log_success "Deployment validation completed"
}

display_access_info() {
    log_info "Deployment completed successfully!"
    echo
    echo "=== Access Information ==="
    echo
    echo "Prometheus:"
    echo "  kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/prometheus-kube-prometheus-prometheus 9090:9090"
    echo "  Access at: http://localhost:9090"
    echo
    echo "Grafana:"
    echo "  kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/grafana 3000:80"
    echo "  Access at: http://localhost:3000"
    echo "  Username: admin"
    echo "  Password: oran-admin-2024"
    echo
    echo "AlertManager:"
    echo "  kubectl port-forward -n $PROMETHEUS_NAMESPACE svc/alertmanager-operated 9093:9093"
    echo "  Access at: http://localhost:9093"
    echo
    echo "=== Monitoring Resources ==="
    echo "Namespaces: $SYSTEM_NAMESPACE, $PROMETHEUS_NAMESPACE, $OBSERVABILITY_NAMESPACE"
    echo "ServiceMonitors: $(kubectl get servicemonitors -n "$PROMETHEUS_NAMESPACE" --no-headers | wc -l)"
    echo "PrometheusRules: $(kubectl get prometheusrules -n "$PROMETHEUS_NAMESPACE" --no-headers | wc -l)"
    echo
}

cleanup_on_failure() {
    log_error "Deployment failed. Cleaning up..."

    # Remove Helm releases
    helm uninstall prometheus-operator -n "$PROMETHEUS_NAMESPACE" 2>/dev/null || true

    # Remove ConfigMaps
    kubectl delete configmap oran-grafana-dashboards -n "$PROMETHEUS_NAMESPACE" 2>/dev/null || true

    # Remove ServiceMonitors
    kubectl delete servicemonitors --all -n "$PROMETHEUS_NAMESPACE" 2>/dev/null || true

    # Remove PrometheusRules
    kubectl delete prometheusrules --all -n "$PROMETHEUS_NAMESPACE" 2>/dev/null || true
}

main() {
    log_info "Starting O-RAN MANO monitoring stack deployment..."

    # Set error trap
    trap cleanup_on_failure ERR

    # Execute deployment steps
    check_prerequisites
    create_namespaces
    add_helm_repositories
    install_prometheus_operator
    deploy_prometheus_rules
    deploy_service_monitors
    deploy_grafana
    configure_alertmanager
    validate_deployment
    display_access_info

    log_success "O-RAN MANO monitoring stack deployed successfully!"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --namespace)
            PROMETHEUS_NAMESPACE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Deploy O-RAN MANO monitoring stack"
            echo ""
            echo "Options:"
            echo "  --namespace NAME    Set monitoring namespace (default: oran-monitoring)"
            echo "  --dry-run          Show what would be deployed without actually deploying"
            echo "  --help             Show this help message"
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Execute main function
main "$@"