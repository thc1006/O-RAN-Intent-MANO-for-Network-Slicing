#!/bin/bash

# CI Validation Script for O-RAN MANO Monitoring Stack
# This script validates the monitoring stack deployment and configuration

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-monitoring}"
TIMEOUT="${TIMEOUT:-300}"
PROMETHEUS_PORT="${PROMETHEUS_PORT:-9090}"
GRAFANA_PORT="${GRAFANA_PORT:-3000}"
ALERTMANAGER_PORT="${ALERTMANAGER_PORT:-9093}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters for reporting
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Logging functions
log() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] ✅ $1${NC}"
    ((TESTS_PASSED++))
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⚠️  $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ❌ $1${NC}"
    ((TESTS_FAILED++))
}

test_step() {
    ((TESTS_TOTAL++))
    log "Test $TESTS_TOTAL: $1"
}

# Validate namespace and basic resources
validate_namespace() {
    test_step "Validating namespace and basic resources"

    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        success "Namespace '$NAMESPACE' exists"
    else
        error "Namespace '$NAMESPACE' does not exist"
        return 1
    fi

    # Check for required CRDs
    local crds=(
        "prometheuses.monitoring.coreos.com"
        "servicemonitors.monitoring.coreos.com"
        "alertmanagers.monitoring.coreos.com"
        "prometheusrules.monitoring.coreos.com"
    )

    for crd in "${crds[@]}"; do
        if kubectl get crd "$crd" &>/dev/null; then
            success "CRD '$crd' exists"
        else
            error "CRD '$crd' is missing"
        fi
    done
}

# Validate all Prometheus targets are UP
validate_prometheus_targets() {
    test_step "Validating Prometheus targets"

    # Start port forward
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus "$PROMETHEUS_PORT:9090" &
    local pf_pid=$!
    sleep 5

    # Function to cleanup port forward
    cleanup_prometheus_pf() {
        kill $pf_pid 2>/dev/null || true
    }
    trap cleanup_prometheus_pf EXIT

    # Test Prometheus API connectivity
    local max_retries=10
    local retry=0
    while [ $retry -lt $max_retries ]; do
        if curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/query?query=up" &>/dev/null; then
            success "Prometheus API is accessible"
            break
        else
            ((retry++))
            if [ $retry -eq $max_retries ]; then
                error "Cannot connect to Prometheus API after $max_retries attempts"
                return 1
            fi
            sleep 2
        fi
    done

    # Get target status
    local targets_response
    targets_response=$(curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/targets" || echo '{"status":"error"}')

    if echo "$targets_response" | jq -e '.status == "success"' &>/dev/null; then
        local total_targets
        local healthy_targets
        local unhealthy_targets

        total_targets=$(echo "$targets_response" | jq '.data.activeTargets | length')
        healthy_targets=$(echo "$targets_response" | jq '[.data.activeTargets[] | select(.health == "up")] | length')
        unhealthy_targets=$((total_targets - healthy_targets))

        log "Target status: $healthy_targets/$total_targets healthy"

        if [ "$unhealthy_targets" -eq 0 ]; then
            success "All Prometheus targets are healthy"
        elif [ "$unhealthy_targets" -le 2 ]; then
            warning "$unhealthy_targets targets are unhealthy (within tolerance)"
        else
            error "$unhealthy_targets targets are unhealthy (exceeds tolerance)"

            # Show unhealthy targets
            echo "$targets_response" | jq -r '
                .data.activeTargets[] |
                select(.health != "up") |
                "  - Job: \(.labels.job), Instance: \(.labels.instance), Error: \(.lastError)"
            '
        fi

        # Validate specific O-RAN targets
        validate_oran_targets "$targets_response"
    else
        error "Failed to get Prometheus targets"
    fi

    cleanup_prometheus_pf
    trap - EXIT
}

# Validate O-RAN specific targets
validate_oran_targets() {
    local targets_response="$1"

    test_step "Validating O-RAN specific targets"

    # Expected O-RAN components
    local expected_jobs=(
        "kubernetes-apiservers"
        "kubernetes-nodes"
        "kubernetes-pods"
        "prometheus-operator-prometheus"
        "prometheus-operator-alertmanager"
        "prometheus-operator-grafana"
    )

    for job in "${expected_jobs[@]}"; do
        local job_targets
        job_targets=$(echo "$targets_response" | jq --arg job "$job" '[.data.activeTargets[] | select(.labels.job == $job)] | length')

        if [ "$job_targets" -gt 0 ]; then
            success "Found $job_targets targets for job '$job'"
        else
            warning "No targets found for job '$job'"
        fi
    done

    # Check for custom ServiceMonitors
    if kubectl get servicemonitors -n "$NAMESPACE" &>/dev/null; then
        local custom_servicemonitors
        custom_servicemonitors=$(kubectl get servicemonitors -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')

        if [ -n "$custom_servicemonitors" ]; then
            success "Found custom ServiceMonitors: $custom_servicemonitors"
        else
            warning "No custom ServiceMonitors found"
        fi
    fi
}

# Check Grafana datasource connectivity
validate_grafana_datasource() {
    test_step "Validating Grafana datasource connectivity"

    # Start port forward
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-grafana "$GRAFANA_PORT:80" &
    local pf_pid=$!
    sleep 5

    # Function to cleanup port forward
    cleanup_grafana_pf() {
        kill $pf_pid 2>/dev/null || true
    }
    trap cleanup_grafana_pf EXIT

    # Test Grafana API connectivity
    local max_retries=10
    local retry=0
    while [ $retry -lt $max_retries ]; do
        if curl -s "http://localhost:$GRAFANA_PORT/api/health" &>/dev/null; then
            success "Grafana API is accessible"
            break
        else
            ((retry++))
            if [ $retry -eq $max_retries ]; then
                error "Cannot connect to Grafana API after $max_retries attempts"
                cleanup_grafana_pf
                trap - EXIT
                return 1
            fi
            sleep 2
        fi
    done

    # Get Grafana credentials
    local grafana_password
    grafana_password=$(kubectl get secret -n "$NAMESPACE" prometheus-operator-grafana -o jsonpath='{.data.admin-password}' | base64 -d)

    # Test datasource connectivity (using basic auth)
    local datasources_response
    datasources_response=$(curl -s -u "admin:$grafana_password" "http://localhost:$GRAFANA_PORT/api/datasources" || echo '[]')

    if echo "$datasources_response" | jq -e '. | length > 0' &>/dev/null; then
        local prometheus_datasources
        prometheus_datasources=$(echo "$datasources_response" | jq '[.[] | select(.type == "prometheus")] | length')

        if [ "$prometheus_datasources" -gt 0 ]; then
            success "Found $prometheus_datasources Prometheus datasource(s) in Grafana"

            # Test datasource health
            echo "$datasources_response" | jq -r '.[] | select(.type == "prometheus") | "\(.id) \(.name)"' | while read -r id name; do
                local health_response
                health_response=$(curl -s -u "admin:$grafana_password" "http://localhost:$GRAFANA_PORT/api/datasources/$id/health" || echo '{"status":"error"}')

                if echo "$health_response" | jq -e '.status == "OK"' &>/dev/null; then
                    success "Datasource '$name' is healthy"
                else
                    error "Datasource '$name' health check failed"
                fi
            done
        else
            error "No Prometheus datasources found in Grafana"
        fi
    else
        error "Failed to get Grafana datasources"
    fi

    cleanup_grafana_pf
    trap - EXIT
}

# Test alert rules syntax
validate_alert_rules() {
    test_step "Validating Prometheus alert rules"

    # Get PrometheusRule resources
    if kubectl get prometheusrules -n "$NAMESPACE" &>/dev/null; then
        local rules
        rules=$(kubectl get prometheusrules -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')

        if [ -n "$rules" ]; then
            success "Found PrometheusRules: $rules"

            # Validate each rule
            for rule in $rules; do
                local rule_yaml
                rule_yaml=$(kubectl get prometheusrule "$rule" -n "$NAMESPACE" -o yaml)

                # Check if rule has proper structure
                if echo "$rule_yaml" | yq -e '.spec.groups[]' &>/dev/null; then
                    success "PrometheusRule '$rule' has valid structure"
                else
                    error "PrometheusRule '$rule' has invalid structure"
                fi
            done
        else
            warning "No PrometheusRules found"
        fi
    else
        warning "Cannot query PrometheusRules"
    fi

    # Check alerting rules in Prometheus
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus "$PROMETHEUS_PORT:9090" &
    local pf_pid=$!
    sleep 5

    local rules_response
    rules_response=$(curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/rules" || echo '{"status":"error"}')

    if echo "$rules_response" | jq -e '.status == "success"' &>/dev/null; then
        local total_rules
        total_rules=$(echo "$rules_response" | jq '[.data.groups[].rules[]] | length')

        if [ "$total_rules" -gt 0 ]; then
            success "Found $total_rules alerting/recording rules in Prometheus"

            # Check for rule evaluation errors
            local rule_errors
            rule_errors=$(echo "$rules_response" | jq '[.data.groups[].rules[] | select(.health != "ok" and .health != null)] | length')

            if [ "$rule_errors" -eq 0 ]; then
                success "All rules are evaluating correctly"
            else
                error "$rule_errors rules have evaluation errors"
            fi
        else
            warning "No alerting/recording rules found in Prometheus"
        fi
    else
        error "Failed to get Prometheus rules"
    fi

    kill $pf_pid 2>/dev/null || true
}

# Check ServiceMonitor selectors
validate_servicemonitor_selectors() {
    test_step "Validating ServiceMonitor selectors"

    # Get Prometheus configuration
    local prometheus_config
    prometheus_config=$(kubectl get prometheus -n "$NAMESPACE" -o yaml)

    if [ -n "$prometheus_config" ]; then
        success "Found Prometheus configuration"

        # Check serviceMonitorSelector
        if echo "$prometheus_config" | yq -e '.items[0].spec.serviceMonitorSelector' &>/dev/null; then
            success "ServiceMonitor selector is configured"
        else
            warning "ServiceMonitor selector not found or empty (will select all)"
        fi

        # Get all ServiceMonitors
        if kubectl get servicemonitors -A &>/dev/null; then
            local total_servicemonitors
            total_servicemonitors=$(kubectl get servicemonitors -A --no-headers | wc -l)
            success "Found $total_servicemonitors ServiceMonitors across all namespaces"

            # Check ServiceMonitors in monitoring namespace
            local monitoring_servicemonitors
            monitoring_servicemonitors=$(kubectl get servicemonitors -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)
            log "ServiceMonitors in monitoring namespace: $monitoring_servicemonitors"

            # Validate ServiceMonitor endpoints
            kubectl get servicemonitors -n "$NAMESPACE" -o yaml | yq -e '.items[].spec.endpoints[]' &>/dev/null && {
                success "ServiceMonitors have valid endpoint configurations"
            } || {
                warning "Some ServiceMonitors may have invalid endpoint configurations"
            }
        else
            error "Cannot query ServiceMonitors"
        fi
    else
        error "Cannot get Prometheus configuration"
    fi
}

# Check resource usage and limits
validate_resource_usage() {
    test_step "Validating resource usage and limits"

    # Check pod resource requests and limits
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -o jsonpath='{.items[*].metadata.name}')

    local pods_with_limits=0
    local total_pods=0

    for pod in $pods; do
        ((total_pods++))

        local has_limits
        has_limits=$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].resources.limits}')

        if [ -n "$has_limits" ]; then
            ((pods_with_limits++))
        fi
    done

    if [ $total_pods -gt 0 ]; then
        local percentage=$((pods_with_limits * 100 / total_pods))
        log "Pods with resource limits: $pods_with_limits/$total_pods ($percentage%)"

        if [ $percentage -ge 80 ]; then
            success "Most pods have resource limits configured"
        elif [ $percentage -ge 50 ]; then
            warning "Some pods are missing resource limits"
        else
            error "Many pods are missing resource limits"
        fi
    fi

    # Check node resource usage (if metrics-server is available)
    if kubectl top nodes &>/dev/null; then
        success "Node metrics are available"
        kubectl top nodes
    else
        warning "Node metrics not available (metrics-server not installed)"
    fi

    # Check pod resource usage
    if kubectl top pods -n "$NAMESPACE" &>/dev/null; then
        success "Pod metrics are available"
        kubectl top pods -n "$NAMESPACE"
    else
        warning "Pod metrics not available"
    fi
}

# Check storage and persistence
validate_storage() {
    test_step "Validating storage and persistence"

    # Check PVCs
    local pvcs
    pvcs=$(kubectl get pvc -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$pvcs" -gt 0 ]; then
        success "Found $pvcs Persistent Volume Claims"

        # Check PVC status
        kubectl get pvc -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name,STATUS:.status.phase,CAPACITY:.status.capacity.storage --no-headers | while read -r name status capacity; do
            if [ "$status" = "Bound" ]; then
                success "PVC '$name' is bound ($capacity)"
            else
                error "PVC '$name' is not bound (status: $status)"
            fi
        done
    else
        warning "No Persistent Volume Claims found"
    fi

    # Check StatefulSets
    local statefulsets
    statefulsets=$(kubectl get statefulsets -n "$NAMESPACE" --no-headers 2>/dev/null | wc -l)

    if [ "$statefulsets" -gt 0 ]; then
        success "Found $statefulsets StatefulSets"

        kubectl get statefulsets -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name,READY:.status.readyReplicas,REPLICAS:.spec.replicas --no-headers | while read -r name ready replicas; do
            if [ "$ready" = "$replicas" ]; then
                success "StatefulSet '$name' is ready ($ready/$replicas)"
            else
                error "StatefulSet '$name' is not ready ($ready/$replicas)"
            fi
        done
    else
        warning "No StatefulSets found"
    fi
}

# Check network connectivity
validate_network_connectivity() {
    test_step "Validating network connectivity"

    # Test service connectivity within cluster
    local services=(
        "prometheus-operator-kube-p-prometheus:9090"
        "prometheus-operator-grafana:80"
        "prometheus-operator-kube-p-alertmanager:9093"
    )

    for service in "${services[@]}"; do
        local service_name="${service%:*}"
        local port="${service##*:}"

        if kubectl get service "$service_name" -n "$NAMESPACE" &>/dev/null; then
            success "Service '$service_name' exists"

            # Test service endpoints
            local endpoints
            endpoints=$(kubectl get endpoints "$service_name" -n "$NAMESPACE" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null)

            if [ -n "$endpoints" ]; then
                local endpoint_count
                endpoint_count=$(echo "$endpoints" | wc -w)
                success "Service '$service_name' has $endpoint_count endpoints"
            else
                error "Service '$service_name' has no endpoints"
            fi
        else
            error "Service '$service_name' does not exist"
        fi
    done

    # Check ingress (if exists)
    if kubectl get ingress -n "$NAMESPACE" &>/dev/null; then
        local ingresses
        ingresses=$(kubectl get ingress -n "$NAMESPACE" --no-headers | wc -l)

        if [ "$ingresses" -gt 0 ]; then
            success "Found $ingresses ingress resources"
        fi
    fi
}

# Test metric queries
validate_metric_queries() {
    test_step "Validating metric queries"

    # Start port forward
    kubectl port-forward -n "$NAMESPACE" svc/prometheus-operator-kube-p-prometheus "$PROMETHEUS_PORT:9090" &
    local pf_pid=$!
    sleep 5

    # Function to cleanup port forward
    cleanup_metrics_pf() {
        kill $pf_pid 2>/dev/null || true
    }
    trap cleanup_metrics_pf EXIT

    # Test common PromQL queries
    local test_queries=(
        "up"
        "prometheus_config_last_reload_successful"
        "prometheus_tsdb_head_series"
        "rate(prometheus_tsdb_head_samples_appended_total[5m])"
        "kube_pod_info"
        "node_memory_MemAvailable_bytes"
    )

    local successful_queries=0

    for query in "${test_queries[@]}"; do
        local response
        response=$(curl -s "http://localhost:$PROMETHEUS_PORT/api/v1/query?query=$(echo "$query" | sed 's/ /%20/g')" || echo '{"status":"error"}')

        if echo "$response" | jq -e '.status == "success"' &>/dev/null; then
            local result_count
            result_count=$(echo "$response" | jq '.data.result | length')
            success "Query '$query' returned $result_count results"
            ((successful_queries++))
        else
            error "Query '$query' failed"
        fi
    done

    log "Successful queries: $successful_queries/${#test_queries[@]}"

    cleanup_metrics_pf
    trap - EXIT
}

# Generate validation report
generate_report() {
    log "Generating validation report..."

    local report_file="${PROJECT_ROOT}/deployment/test-results/validation-report.txt"
    mkdir -p "$(dirname "$report_file")"

    cat > "$report_file" << EOF
O-RAN MANO Monitoring Stack Validation Report
=============================================

Generated: $(date)
Namespace: $NAMESPACE
Cluster: $(kubectl config current-context)

Test Results Summary:
- Total Tests: $TESTS_TOTAL
- Passed: $TESTS_PASSED
- Failed: $TESTS_FAILED
- Success Rate: $(( TESTS_PASSED * 100 / TESTS_TOTAL ))%

Detailed Results:
EOF

    # Add pod status
    echo "" >> "$report_file"
    echo "Pod Status:" >> "$report_file"
    kubectl get pods -n "$NAMESPACE" -o wide >> "$report_file" 2>/dev/null || echo "Could not get pod status" >> "$report_file"

    # Add service status
    echo "" >> "$report_file"
    echo "Service Status:" >> "$report_file"
    kubectl get services -n "$NAMESPACE" >> "$report_file" 2>/dev/null || echo "Could not get service status" >> "$report_file"

    # Add ServiceMonitor status
    echo "" >> "$report_file"
    echo "ServiceMonitor Status:" >> "$report_file"
    kubectl get servicemonitors -n "$NAMESPACE" >> "$report_file" 2>/dev/null || echo "Could not get ServiceMonitor status" >> "$report_file"

    success "Validation report generated: $report_file"
}

# Main validation function
main() {
    log "Starting O-RAN MANO monitoring stack validation"
    log "Namespace: $NAMESPACE"

    # Run all validation tests
    validate_namespace
    validate_prometheus_targets
    validate_grafana_datasource
    validate_alert_rules
    validate_servicemonitor_selectors
    validate_resource_usage
    validate_storage
    validate_network_connectivity
    validate_metric_queries

    # Generate report
    generate_report

    # Final summary
    echo ""
    log "Validation Summary:"
    log "  Total Tests: $TESTS_TOTAL"
    log "  Passed: $TESTS_PASSED"
    log "  Failed: $TESTS_FAILED"

    if [ $TESTS_FAILED -eq 0 ]; then
        success "🎉 All validation tests passed!"
        exit 0
    elif [ $TESTS_FAILED -le 2 ]; then
        warning "⚠️  Some tests failed, but within acceptable limits"
        exit 0
    else
        error "❌ Too many tests failed - deployment may have issues"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-validate}" in
    "validate"|"")
        main
        ;;
    "targets")
        validate_prometheus_targets
        ;;
    "grafana")
        validate_grafana_datasource
        ;;
    "rules")
        validate_alert_rules
        ;;
    "metrics")
        validate_metric_queries
        ;;
    "help")
        echo "Usage: $0 [validate|targets|grafana|rules|metrics|help]"
        echo "  validate - Run all validation tests (default)"
        echo "  targets  - Validate Prometheus targets only"
        echo "  grafana  - Validate Grafana datasource only"
        echo "  rules    - Validate alert rules only"
        echo "  metrics  - Validate metric queries only"
        echo "  help     - Show this help message"
        ;;
    *)
        error "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac