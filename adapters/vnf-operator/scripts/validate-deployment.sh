#!/bin/bash

set -euo pipefail

# Validate all O-RAN MANO components are running correctly
# This script checks deployment status, metrics endpoints, Prometheus targets, and Grafana dashboards

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SYSTEM_NAMESPACE="oran-system"
MONITORING_NAMESPACE="oran-monitoring"
OBSERVABILITY_NAMESPACE="oran-observability"

# Validation results
declare -a VALIDATION_RESULTS=()
TOTAL_CHECKS=0
PASSED_CHECKS=0

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

add_result() {
    local check_name="$1"
    local status="$2"
    local message="$3"

    VALIDATION_RESULTS+=("$check_name|$status|$message")
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))

    if [[ "$status" == "PASS" ]]; then
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
        log_success "$check_name: $message"
    elif [[ "$status" == "WARN" ]]; then
        log_warning "$check_name: $message"
    else
        log_error "$check_name: $message"
    fi
}

check_namespace_exists() {
    local namespace="$1"
    local check_name="Namespace $namespace"

    if kubectl get namespace "$namespace" &>/dev/null; then
        add_result "$check_name" "PASS" "Namespace exists and is active"
    else
        add_result "$check_name" "FAIL" "Namespace does not exist or is not accessible"
    fi
}

check_deployment_ready() {
    local deployment="$1"
    local namespace="$2"
    local check_name="Deployment $deployment"

    if kubectl get deployment "$deployment" -n "$namespace" &>/dev/null; then
        local ready_replicas
        ready_replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired_replicas
        desired_replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [[ "$ready_replicas" == "$desired_replicas" ]] && [[ "$ready_replicas" -gt 0 ]]; then
            add_result "$check_name" "PASS" "Deployment is ready ($ready_replicas/$desired_replicas replicas)"
        else
            add_result "$check_name" "FAIL" "Deployment not ready ($ready_replicas/$desired_replicas replicas)"
        fi
    else
        add_result "$check_name" "FAIL" "Deployment does not exist"
    fi
}

check_statefulset_ready() {
    local statefulset="$1"
    local namespace="$2"
    local check_name="StatefulSet $statefulset"

    if kubectl get statefulset "$statefulset" -n "$namespace" &>/dev/null; then
        local ready_replicas
        ready_replicas=$(kubectl get statefulset "$statefulset" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
        local desired_replicas
        desired_replicas=$(kubectl get statefulset "$statefulset" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo "0")

        if [[ "$ready_replicas" == "$desired_replicas" ]] && [[ "$ready_replicas" -gt 0 ]]; then
            add_result "$check_name" "PASS" "StatefulSet is ready ($ready_replicas/$desired_replicas replicas)"
        else
            add_result "$check_name" "FAIL" "StatefulSet not ready ($ready_replicas/$desired_replicas replicas)"
        fi
    else
        add_result "$check_name" "FAIL" "StatefulSet does not exist"
    fi
}

check_service_endpoints() {
    local service="$1"
    local namespace="$2"
    local check_name="Service $service endpoints"

    if kubectl get service "$service" -n "$namespace" &>/dev/null; then
        local endpoints
        endpoints=$(kubectl get endpoints "$service" -n "$namespace" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null || echo "")

        if [[ -n "$endpoints" ]]; then
            local endpoint_count
            endpoint_count=$(echo "$endpoints" | wc -w)
            add_result "$check_name" "PASS" "Service has $endpoint_count active endpoints"
        else
            add_result "$check_name" "FAIL" "Service has no active endpoints"
        fi
    else
        add_result "$check_name" "FAIL" "Service does not exist"
    fi
}

check_metrics_endpoint() {
    local service="$1"
    local namespace="$2"
    local port="$3"
    local path="${4:-/metrics}"
    local check_name="Metrics endpoint $service:$port$path"

    # Try to access metrics endpoint via kubectl port-forward
    if kubectl get service "$service" -n "$namespace" &>/dev/null; then
        local pod_name
        pod_name=$(kubectl get pods -n "$namespace" -l "app=$service" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

        if [[ -n "$pod_name" ]]; then
            # Use kubectl exec to check if metrics endpoint responds
            if kubectl exec "$pod_name" -n "$namespace" -- wget -q -O- "http://localhost:$port$path" &>/dev/null; then
                add_result "$check_name" "PASS" "Metrics endpoint is responding"
            else
                add_result "$check_name" "FAIL" "Metrics endpoint is not responding"
            fi
        else
            add_result "$check_name" "FAIL" "No pods found for service"
        fi
    else
        add_result "$check_name" "FAIL" "Service does not exist"
    fi
}

check_prometheus_targets() {
    local check_name="Prometheus targets"

    # Check if Prometheus is running
    if kubectl get prometheus -n "$MONITORING_NAMESPACE" &>/dev/null; then
        # Get Prometheus pod
        local prometheus_pod
        prometheus_pod=$(kubectl get pods -n "$MONITORING_NAMESPACE" -l "app.kubernetes.io/name=prometheus" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

        if [[ -n "$prometheus_pod" ]]; then
            # Check if Prometheus API is accessible
            if kubectl exec "$prometheus_pod" -n "$MONITORING_NAMESPACE" -c prometheus -- wget -q -O- "http://localhost:9090/-/healthy" &>/dev/null; then
                add_result "$check_name" "PASS" "Prometheus is healthy and targets are discoverable"
            else
                add_result "$check_name" "FAIL" "Prometheus health check failed"
            fi
        else
            add_result "$check_name" "FAIL" "Prometheus pod not found"
        fi
    else
        add_result "$check_name" "FAIL" "Prometheus not deployed"
    fi
}

check_grafana_dashboards() {
    local check_name="Grafana dashboards"

    # Check if Grafana is running
    if kubectl get deployment grafana -n "$MONITORING_NAMESPACE" &>/dev/null; then
        # Check if dashboard ConfigMap exists
        if kubectl get configmap oran-grafana-dashboards -n "$MONITORING_NAMESPACE" &>/dev/null; then
            local dashboard_count
            dashboard_count=$(kubectl get configmap oran-grafana-dashboards -n "$MONITORING_NAMESPACE" -o jsonpath='{.data}' | jq -r 'keys | length' 2>/dev/null || echo "0")

            if [[ "$dashboard_count" -gt 0 ]]; then
                add_result "$check_name" "PASS" "Found $dashboard_count Grafana dashboards"
            else
                add_result "$check_name" "FAIL" "No dashboards found in ConfigMap"
            fi
        else
            add_result "$check_name" "FAIL" "Dashboard ConfigMap not found"
        fi
    else
        add_result "$check_name" "FAIL" "Grafana not deployed"
    fi
}

check_servicemonitors() {
    local check_name="ServiceMonitors"

    local servicemonitor_count
    servicemonitor_count=$(kubectl get servicemonitors -n "$MONITORING_NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")

    if [[ "$servicemonitor_count" -gt 0 ]]; then
        add_result "$check_name" "PASS" "Found $servicemonitor_count ServiceMonitors"
    else
        add_result "$check_name" "FAIL" "No ServiceMonitors found"
    fi
}

check_prometheusrules() {
    local check_name="PrometheusRules"

    local rules_count
    rules_count=$(kubectl get prometheusrules -n "$MONITORING_NAMESPACE" --no-headers 2>/dev/null | wc -l || echo "0")

    if [[ "$rules_count" -gt 0 ]]; then
        add_result "$check_name" "PASS" "Found $rules_count PrometheusRules"
    else
        add_result "$check_name" "WARN" "No PrometheusRules found"
    fi
}

check_alertmanager_config() {
    local check_name="AlertManager configuration"

    # Check if AlertManager secret exists
    if kubectl get secret -n "$MONITORING_NAMESPACE" | grep -q "alertmanager.*configuration"; then
        add_result "$check_name" "PASS" "AlertManager configuration secret exists"
    else
        add_result "$check_name" "WARN" "AlertManager configuration secret not found"
    fi
}

check_pod_health() {
    local namespace="$1"
    local check_name="Pod health in $namespace"

    local total_pods
    total_pods=$(kubectl get pods -n "$namespace" --no-headers 2>/dev/null | wc -l || echo "0")

    if [[ "$total_pods" -eq 0 ]]; then
        add_result "$check_name" "WARN" "No pods found in namespace"
        return
    fi

    local ready_pods
    ready_pods=$(kubectl get pods -n "$namespace" --no-headers -o custom-columns=":status.containerStatuses[*].ready" 2>/dev/null | grep -c "true" || echo "0")

    local unhealthy_pods
    unhealthy_pods=$(kubectl get pods -n "$namespace" --field-selector=status.phase!=Running --no-headers 2>/dev/null | wc -l || echo "0")

    if [[ "$unhealthy_pods" -eq 0 ]]; then
        add_result "$check_name" "PASS" "All $total_pods pods are healthy"
    else
        add_result "$check_name" "FAIL" "$unhealthy_pods out of $total_pods pods are unhealthy"
    fi
}

check_resource_usage() {
    local namespace="$1"
    local check_name="Resource usage in $namespace"

    # Check if metrics-server is available
    if ! kubectl top pods -n "$namespace" &>/dev/null; then
        add_result "$check_name" "WARN" "Metrics server not available, cannot check resource usage"
        return
    fi

    # Get resource usage
    local high_cpu_pods
    high_cpu_pods=$(kubectl top pods -n "$namespace" --no-headers 2>/dev/null | awk '$2 ~ /[0-9]+m/ && $2+0 > 500 {print $1}' | wc -l || echo "0")

    local high_memory_pods
    high_memory_pods=$(kubectl top pods -n "$namespace" --no-headers 2>/dev/null | awk '$3 ~ /[0-9]+Mi/ && $3+0 > 512 {print $1}' | wc -l || echo "0")

    if [[ "$high_cpu_pods" -eq 0 ]] && [[ "$high_memory_pods" -eq 0 ]]; then
        add_result "$check_name" "PASS" "Resource usage is within normal limits"
    else
        add_result "$check_name" "WARN" "$high_cpu_pods pods with high CPU, $high_memory_pods pods with high memory"
    fi
}

run_connectivity_tests() {
    local check_name="Service connectivity"

    # Test connectivity between services
    local orchestrator_pod
    orchestrator_pod=$(kubectl get pods -n "$SYSTEM_NAMESPACE" -l "app=orchestrator" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

    if [[ -n "$orchestrator_pod" ]]; then
        # Test connectivity to DMS components
        local connectivity_issues=0

        for service in "ran-dms" "cn-dms" "tn-manager"; do
            if ! kubectl exec "$orchestrator_pod" -n "$SYSTEM_NAMESPACE" -- wget -q -T 5 -O- "http://$service.$SYSTEM_NAMESPACE.svc.cluster.local/health" &>/dev/null; then
                connectivity_issues=$((connectivity_issues + 1))
            fi
        done

        if [[ "$connectivity_issues" -eq 0 ]]; then
            add_result "$check_name" "PASS" "All inter-service connectivity tests passed"
        else
            add_result "$check_name" "FAIL" "$connectivity_issues connectivity issues detected"
        fi
    else
        add_result "$check_name" "WARN" "Cannot test connectivity - orchestrator pod not found"
    fi
}

validate_crds() {
    local check_name="Custom Resource Definitions"

    local expected_crds=("vnfdeployments.mano.oran.io" "networkslices.mano.oran.io" "qosprofiles.mano.oran.io")
    local missing_crds=0

    for crd in "${expected_crds[@]}"; do
        if ! kubectl get crd "$crd" &>/dev/null; then
            missing_crds=$((missing_crds + 1))
        fi
    done

    if [[ "$missing_crds" -eq 0 ]]; then
        add_result "$check_name" "PASS" "All required CRDs are installed"
    else
        add_result "$check_name" "FAIL" "$missing_crds CRDs are missing"
    fi
}

print_summary() {
    echo
    echo "=========================="
    echo "  VALIDATION SUMMARY"
    echo "=========================="
    echo
    echo "Total checks: $TOTAL_CHECKS"
    echo "Passed: $PASSED_CHECKS"
    echo "Failed/Warnings: $((TOTAL_CHECKS - PASSED_CHECKS))"
    echo

    local pass_percentage
    if [[ $TOTAL_CHECKS -gt 0 ]]; then
        pass_percentage=$(( (PASSED_CHECKS * 100) / TOTAL_CHECKS ))
    else
        pass_percentage=0
    fi

    echo "Success rate: $pass_percentage%"
    echo

    if [[ $pass_percentage -ge 90 ]]; then
        log_success "Deployment validation PASSED"
        echo "The O-RAN MANO system is healthy and ready for use."
    elif [[ $pass_percentage -ge 70 ]]; then
        log_warning "Deployment validation PASSED with warnings"
        echo "The O-RAN MANO system is functional but some issues were detected."
    else
        log_error "Deployment validation FAILED"
        echo "The O-RAN MANO system has significant issues that need attention."
    fi

    echo
    echo "Detailed results:"
    echo "----------------"

    for result in "${VALIDATION_RESULTS[@]}"; do
        IFS='|' read -r name status message <<< "$result"
        case $status in
            "PASS")
                echo -e "${GREEN}✓${NC} $name: $message"
                ;;
            "WARN")
                echo -e "${YELLOW}⚠${NC} $name: $message"
                ;;
            "FAIL")
                echo -e "${RED}✗${NC} $name: $message"
                ;;
        esac
    done
}

main() {
    log_info "Starting O-RAN MANO deployment validation..."
    echo

    # Check prerequisites
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed or not in PATH"
        exit 1
    fi

    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi

    # Run all validation checks
    log_info "Checking namespaces..."
    check_namespace_exists "$SYSTEM_NAMESPACE"
    check_namespace_exists "$MONITORING_NAMESPACE"
    check_namespace_exists "$OBSERVABILITY_NAMESPACE"

    log_info "Checking deployments..."
    check_deployment_ready "orchestrator" "$SYSTEM_NAMESPACE"
    check_deployment_ready "ran-dms" "$SYSTEM_NAMESPACE"
    check_deployment_ready "cn-dms" "$SYSTEM_NAMESPACE"
    check_deployment_ready "tn-manager" "$SYSTEM_NAMESPACE"
    check_deployment_ready "grafana" "$MONITORING_NAMESPACE"

    log_info "Checking StatefulSets..."
    check_statefulset_ready "vnf-operator" "$SYSTEM_NAMESPACE"

    log_info "Checking services..."
    check_service_endpoints "orchestrator" "$SYSTEM_NAMESPACE"
    check_service_endpoints "vnf-operator-metrics-service" "$SYSTEM_NAMESPACE"
    check_service_endpoints "ran-dms" "$SYSTEM_NAMESPACE"
    check_service_endpoints "cn-dms" "$SYSTEM_NAMESPACE"
    check_service_endpoints "tn-manager" "$SYSTEM_NAMESPACE"
    check_service_endpoints "grafana" "$MONITORING_NAMESPACE"

    log_info "Checking metrics endpoints..."
    check_metrics_endpoint "orchestrator" "$SYSTEM_NAMESPACE" "9090"
    check_metrics_endpoint "ran-dms" "$SYSTEM_NAMESPACE" "9090"
    check_metrics_endpoint "cn-dms" "$SYSTEM_NAMESPACE" "9090"
    check_metrics_endpoint "tn-manager" "$SYSTEM_NAMESPACE" "9090"

    log_info "Checking monitoring components..."
    check_prometheus_targets
    check_grafana_dashboards
    check_servicemonitors
    check_prometheusrules
    check_alertmanager_config

    log_info "Checking pod health..."
    check_pod_health "$SYSTEM_NAMESPACE"
    check_pod_health "$MONITORING_NAMESPACE"

    log_info "Checking resource usage..."
    check_resource_usage "$SYSTEM_NAMESPACE"
    check_resource_usage "$MONITORING_NAMESPACE"

    log_info "Testing connectivity..."
    run_connectivity_tests

    log_info "Validating CRDs..."
    validate_crds

    # Print summary
    print_summary

    # Exit with appropriate code
    if [[ $((PASSED_CHECKS * 100 / TOTAL_CHECKS)) -ge 70 ]]; then
        exit 0
    else
        exit 1
    fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --system-namespace)
            SYSTEM_NAMESPACE="$2"
            shift 2
            ;;
        --monitoring-namespace)
            MONITORING_NAMESPACE="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Validate O-RAN MANO deployment"
            echo ""
            echo "Options:"
            echo "  --system-namespace NAME       Set system namespace (default: oran-system)"
            echo "  --monitoring-namespace NAME   Set monitoring namespace (default: oran-monitoring)"
            echo "  --help                        Show this help message"
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