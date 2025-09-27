#!/bin/bash
# O-RAN Monitoring Stack - Alert Rules and AlertManager Health Check
# This script validates alert rules are loaded and AlertManager is functional

set -euo pipefail

# Configuration
NAMESPACE="${MONITORING_NAMESPACE:-oran-monitoring}"
PROMETHEUS_SERVICE="${PROMETHEUS_SERVICE:-prometheus}"
ALERTMANAGER_SERVICE="${ALERTMANAGER_SERVICE:-alertmanager}"
TIMEOUT="${TIMEOUT:-60}"
MIN_ALERT_RULES="${MIN_ALERT_RULES:-5}"
EXPECTED_ALERT_GROUPS="${EXPECTED_ALERT_GROUPS:-oran.rules,kubernetes.rules,prometheus.rules}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Function to setup port forwards
setup_port_forwards() {
    local prometheus_port=19090
    local alertmanager_port=19093

    # Find available ports
    while lsof -i :$prometheus_port &> /dev/null; do
        ((prometheus_port++))
    done

    while lsof -i :$alertmanager_port &> /dev/null; do
        ((alertmanager_port++))
    done

    # Start Prometheus port-forward
    kubectl port-forward -n "$NAMESPACE" service/"$PROMETHEUS_SERVICE" $prometheus_port:9090 &> /dev/null &
    local prometheus_pid=$!

    # Start AlertManager port-forward
    kubectl port-forward -n "$NAMESPACE" service/"$ALERTMANAGER_SERVICE" $alertmanager_port:9093 &> /dev/null &
    local alertmanager_pid=$!

    # Wait for port-forwards to be ready
    sleep 5

    # Verify port-forwards are working
    if ! lsof -i :$prometheus_port &> /dev/null; then
        log_error "Prometheus port-forward failed"
        kill $prometheus_pid $alertmanager_pid 2>/dev/null || true
        exit 1
    fi

    if ! lsof -i :$alertmanager_port &> /dev/null; then
        log_error "AlertManager port-forward failed"
        kill $prometheus_pid $alertmanager_pid 2>/dev/null || true
        exit 1
    fi

    echo "$prometheus_pid:$prometheus_port:$alertmanager_pid:$alertmanager_port"
}

# Function to cleanup port forwards
cleanup_port_forwards() {
    local port_info="$1"
    IFS=':' read -r prometheus_pid prometheus_port alertmanager_pid alertmanager_port <<< "$port_info"

    kill $prometheus_pid $alertmanager_pid 2>/dev/null || true
    log_info "Cleaned up port-forward processes"
}

# Function to make Prometheus API calls
prometheus_api_call() {
    local endpoint="$1"
    local prometheus_port="$2"

    curl -s --max-time "$TIMEOUT" "http://localhost:$prometheus_port$endpoint"
}

# Function to make AlertManager API calls
alertmanager_api_call() {
    local endpoint="$1"
    local alertmanager_port="$2"

    curl -s --max-time "$TIMEOUT" "http://localhost:$alertmanager_port$endpoint"
}

# Test 1: Verify alert rules are loaded in Prometheus
test_alert_rules_loaded() {
    local prometheus_port="$1"

    log_info "Testing alert rules loading..."

    local rules_response
    if ! rules_response=$(prometheus_api_call "/api/v1/rules" "$prometheus_port"); then
        log_error "Failed to query Prometheus rules API"
        return 1
    fi

    # Parse rules response
    local status
    status=$(echo "$rules_response" | jq -r '.status // "error"')

    if [[ "$status" != "success" ]]; then
        log_error "Prometheus rules API returned status: $status"
        return 1
    fi

    # Count total rules
    local total_rules=0
    local alert_rules=0
    local recording_rules=0
    local groups_count=0

    while IFS= read -r group; do
        if [[ -n "$group" ]]; then
            ((groups_count++))
            local group_name
            group_name=$(echo "$group" | jq -r '.name')

            while IFS= read -r rule; do
                if [[ -n "$rule" ]]; then
                    ((total_rules++))
                    local rule_type
                    rule_type=$(echo "$rule" | jq -r '.type')

                    if [[ "$rule_type" == "alerting" ]]; then
                        ((alert_rules++))
                    elif [[ "$rule_type" == "recording" ]]; then
                        ((recording_rules++))
                    fi
                fi
            done < <(echo "$group" | jq -c '.rules[]?')
        fi
    done < <(echo "$rules_response" | jq -c '.data.groups[]?')

    log_info "Found $groups_count rule groups with $total_rules total rules"
    log_info "  - Alert rules: $alert_rules"
    log_info "  - Recording rules: $recording_rules"

    if [[ $total_rules -lt $MIN_ALERT_RULES ]]; then
        log_error "Only $total_rules rules found (minimum required: $MIN_ALERT_RULES)"
        return 1
    fi

    log_success "Alert rules are properly loaded"

    # Check for expected rule groups
    IFS=',' read -ra expected_groups_array <<< "$EXPECTED_ALERT_GROUPS"
    local missing_groups=()
    local found_groups=()

    for expected_group in "${expected_groups_array[@]}"; do
        local found=false
        while IFS= read -r group_name; do
            if [[ -n "$group_name" ]] && [[ "$group_name" =~ $expected_group ]]; then
                found=true
                found_groups+=("$expected_group")
                break
            fi
        done < <(echo "$rules_response" | jq -r '.data.groups[].name')

        if [[ "$found" == false ]]; then
            missing_groups+=("$expected_group")
        fi
    done

    if [[ ${#missing_groups[@]} -gt 0 ]]; then
        log_warning "Missing expected rule groups: ${missing_groups[*]}"
    else
        log_success "All expected rule groups are present"
    fi

    # Display rule details
    log_info "Rule group details:"
    echo "$rules_response" | jq -r '.data.groups[] | "  \(.name): \(.rules | length) rules (interval: \(.interval // "unknown"))"'

    return 0
}

# Test 2: Verify alert rules syntax and evaluation
test_alert_rules_evaluation() {
    local prometheus_port="$1"

    log_info "Testing alert rule evaluation..."

    local rules_response
    rules_response=$(prometheus_api_call "/api/v1/rules" "$prometheus_port")

    # Check for rules with evaluation errors
    local rules_with_errors=0
    local rules_with_alerts=0

    while IFS= read -r rule; do
        if [[ -n "$rule" ]]; then
            local rule_name
            local rule_type
            local last_error
            local alerts

            rule_name=$(echo "$rule" | jq -r '.name')
            rule_type=$(echo "$rule" | jq -r '.type')
            last_error=$(echo "$rule" | jq -r '.lastError // empty')
            alerts=$(echo "$rule" | jq -r '.alerts // []')

            if [[ -n "$last_error" ]]; then
                ((rules_with_errors++))
                log_warning "  Rule '$rule_name' has evaluation error: $last_error"
            fi

            if [[ "$rule_type" == "alerting" ]] && [[ "$alerts" != "[]" ]]; then
                local alert_count
                alert_count=$(echo "$alerts" | jq 'length')
                if [[ $alert_count -gt 0 ]]; then
                    ((rules_with_alerts++))
                    log_info "  Alert rule '$rule_name' has $alert_count active alerts"
                fi
            fi
        fi
    done < <(echo "$rules_response" | jq -c '.data.groups[].rules[]?')

    if [[ $rules_with_errors -gt 0 ]]; then
        log_error "$rules_with_errors rules have evaluation errors"
        return 1
    else
        log_success "All rules are evaluating without errors"
    fi

    if [[ $rules_with_alerts -gt 0 ]]; then
        log_warning "$rules_with_alerts alert rules are currently firing"
    else
        log_success "No alerts are currently firing (system is healthy)"
    fi

    return 0
}

# Test 3: Verify O-RAN specific alert rules
test_oran_alert_rules() {
    local prometheus_port="$1"

    log_info "Testing O-RAN specific alert rules..."

    local rules_response
    rules_response=$(prometheus_api_call "/api/v1/rules" "$prometheus_port")

    # Look for O-RAN specific rules
    local oran_rules=()
    while IFS= read -r rule; do
        if [[ -n "$rule" ]]; then
            local rule_name
            local rule_expr
            rule_name=$(echo "$rule" | jq -r '.name')
            rule_expr=$(echo "$rule" | jq -r '.query // .expr // empty')

            # Check if rule is O-RAN related
            if [[ "$rule_name" =~ [Oo][Rr][Aa][Nn] ]] || [[ "$rule_expr" =~ oran_ ]]; then
                oran_rules+=("$rule_name")
            fi
        fi
    done < <(echo "$rules_response" | jq -c '.data.groups[].rules[]?')

    if [[ ${#oran_rules[@]} -gt 0 ]]; then
        log_success "Found ${#oran_rules[@]} O-RAN specific alert rules:"
        for rule in "${oran_rules[@]}"; do
            log_info "  - $rule"
        done
    else
        log_warning "No O-RAN specific alert rules found"
    fi

    # Check for common O-RAN alert patterns
    local expected_oran_patterns=(
        "intent.*processing"
        "slice.*deployment"
        "vnf.*placement"
        "network.*slice"
    )

    local found_patterns=()
    for pattern in "${expected_oran_patterns[@]}"; do
        local pattern_found=false
        for rule in "${oran_rules[@]}"; do
            if [[ "$rule" =~ $pattern ]]; then
                pattern_found=true
                found_patterns+=("$pattern")
                break
            fi
        done
    done

    if [[ ${#found_patterns[@]} -gt 0 ]]; then
        log_success "Found O-RAN alert patterns: ${found_patterns[*]}"
    fi

    return 0
}

# Test 4: Verify AlertManager is accessible and healthy
test_alertmanager_health() {
    local alertmanager_port="$1"

    log_info "Testing AlertManager health..."

    # Test basic connectivity
    local status_response
    if ! status_response=$(alertmanager_api_call "/api/v2/status" "$alertmanager_port"); then
        log_error "Failed to query AlertManager status API"
        return 1
    fi

    # Parse status response
    local version_info
    version_info=$(echo "$status_response" | jq -r '.versionInfo // {}')

    if [[ "$version_info" != "{}" ]]; then
        local version
        version=$(echo "$version_info" | jq -r '.version // "unknown"')
        log_success "AlertManager is healthy (version: $version)"
    else
        log_error "AlertManager returned invalid status response"
        return 1
    fi

    # Test configuration
    local config_response
    if config_response=$(alertmanager_api_call "/api/v2/status" "$alertmanager_port"); then
        local config_hash
        config_hash=$(echo "$config_response" | jq -r '.configHash // "unknown"')
        log_info "AlertManager config hash: $config_hash"
    fi

    return 0
}

# Test 5: Verify AlertManager configuration
test_alertmanager_config() {
    local alertmanager_port="$1"

    log_info "Testing AlertManager configuration..."

    # Get configuration
    local config_response
    if ! config_response=$(alertmanager_api_call "/api/v1/status" "$alertmanager_port"); then
        log_warning "Could not retrieve AlertManager configuration details"
        return 0
    fi

    # Check if configuration is valid
    local config_yaml
    config_yaml=$(echo "$config_response" | jq -r '.data.configYAML // empty')

    if [[ -n "$config_yaml" ]]; then
        # Basic configuration validation
        if echo "$config_yaml" | grep -q "route:"; then
            log_success "AlertManager has routing configuration"
        else
            log_warning "AlertManager routing configuration not found"
        fi

        if echo "$config_yaml" | grep -q "receivers:"; then
            log_success "AlertManager has receiver configuration"
        else
            log_warning "AlertManager receiver configuration not found"
        fi

        # Check for notification configurations
        local notification_types=()
        if echo "$config_yaml" | grep -q "email_configs:"; then
            notification_types+=("email")
        fi
        if echo "$config_yaml" | grep -q "slack_configs:"; then
            notification_types+=("slack")
        fi
        if echo "$config_yaml" | grep -q "webhook_configs:"; then
            notification_types+=("webhook")
        fi

        if [[ ${#notification_types[@]} -gt 0 ]]; then
            log_success "AlertManager notification types: ${notification_types[*]}"
        else
            log_warning "No notification configurations found"
        fi
    else
        log_warning "Could not retrieve AlertManager configuration YAML"
    fi

    return 0
}

# Test 6: Test alert firing and silencing
test_alert_management() {
    local alertmanager_port="$1"

    log_info "Testing alert management..."

    # Get current alerts
    local alerts_response
    if ! alerts_response=$(alertmanager_api_call "/api/v2/alerts" "$alertmanager_port"); then
        log_error "Failed to query AlertManager alerts API"
        return 1
    fi

    # Parse alerts
    local alert_count
    alert_count=$(echo "$alerts_response" | jq 'length')

    log_info "Current alerts: $alert_count"

    if [[ $alert_count -gt 0 ]]; then
        # Show alert summary
        echo "$alerts_response" | jq -r '.[] | "  - \(.labels.alertname // "unknown") (\(.status.state))"' | head -10

        # Check alert states
        local firing_alerts
        local silenced_alerts
        firing_alerts=$(echo "$alerts_response" | jq '[.[] | select(.status.state == "active")] | length')
        silenced_alerts=$(echo "$alerts_response" | jq '[.[] | select(.status.state == "suppressed")] | length')

        log_info "  - Firing: $firing_alerts"
        log_info "  - Silenced: $silenced_alerts"
    else
        log_success "No active alerts (system is healthy)"
    fi

    # Get silences
    local silences_response
    if silences_response=$(alertmanager_api_call "/api/v2/silences" "$alertmanager_port"); then
        local silence_count
        silence_count=$(echo "$silences_response" | jq 'length')

        if [[ $silence_count -gt 0 ]]; then
            log_info "Active silences: $silence_count"
            echo "$silences_response" | jq -r '.[] | "  - \(.comment // "no comment") (expires: \(.endsAt))"' | head -5
        else
            log_info "No active silences"
        fi
    fi

    return 0
}

# Test 7: Test AlertManager cluster status (if clustered)
test_alertmanager_cluster() {
    local alertmanager_port="$1"

    log_info "Testing AlertManager cluster status..."

    local status_response
    if status_response=$(alertmanager_api_call "/api/v2/status" "$alertmanager_port"); then
        local cluster_status
        cluster_status=$(echo "$status_response" | jq -r '.cluster // {}')

        if [[ "$cluster_status" != "{}" ]]; then
            local cluster_size
            local cluster_name
            cluster_size=$(echo "$cluster_status" | jq -r '.peers // [] | length')
            cluster_name=$(echo "$cluster_status" | jq -r '.name // "unknown"')

            if [[ $cluster_size -gt 1 ]]; then
                log_success "AlertManager is running in cluster mode with $cluster_size peers"
                log_info "Cluster name: $cluster_name"
            else
                log_info "AlertManager is running in standalone mode"
            fi
        else
            log_info "AlertManager cluster information not available"
        fi
    fi

    return 0
}

# Test 8: Verify Prometheus-AlertManager connectivity
test_prometheus_alertmanager_connection() {
    local prometheus_port="$1"

    log_info "Testing Prometheus to AlertManager connectivity..."

    # Query AlertManager status from Prometheus
    local alertmanagers_response
    if ! alertmanagers_response=$(prometheus_api_call "/api/v1/alertmanagers" "$prometheus_port"); then
        log_error "Failed to query Prometheus alertmanagers API"
        return 1
    fi

    local status
    status=$(echo "$alertmanagers_response" | jq -r '.status // "error"')

    if [[ "$status" != "success" ]]; then
        log_error "Prometheus alertmanagers API returned status: $status"
        return 1
    fi

    # Check AlertManager endpoints
    local active_alertmanagers
    local dropped_alertmanagers
    active_alertmanagers=$(echo "$alertmanagers_response" | jq '.data.activeAlertmanagers | length')
    dropped_alertmanagers=$(echo "$alertmanagers_response" | jq '.data.droppedAlertmanagers | length')

    log_info "Active AlertManagers: $active_alertmanagers"
    log_info "Dropped AlertManagers: $dropped_alertmanagers"

    if [[ $active_alertmanagers -eq 0 ]]; then
        log_error "No active AlertManager endpoints found in Prometheus"
        return 1
    fi

    # Show AlertManager details
    echo "$alertmanagers_response" | jq -r '.data.activeAlertmanagers[] | "  - \(.url) (health: \(.health // "unknown"))"'

    log_success "Prometheus can communicate with AlertManager"

    return 0
}

# Main execution function
main() {
    log_info "=== O-RAN Alert Rules and AlertManager Health Check ==="
    log_info "Namespace: $NAMESPACE"
    log_info "Prometheus Service: $PROMETHEUS_SERVICE"
    log_info "AlertManager Service: $ALERTMANAGER_SERVICE"
    log_info "Timeout: ${TIMEOUT}s"
    log_info "Minimum alert rules: $MIN_ALERT_RULES"
    echo

    # Setup port forwards
    log_info "Setting up port-forwards..."
    local port_info
    port_info=$(setup_port_forwards)
    IFS=':' read -r prometheus_pid prometheus_port alertmanager_pid alertmanager_port <<< "$port_info"

    log_success "Port-forwards established (Prometheus: $prometheus_port, AlertManager: $alertmanager_port)"

    # Ensure cleanup on exit
    trap "cleanup_port_forwards $port_info" EXIT

    # Run all tests
    local test_results=()

    # Test 1: Alert Rules Loading
    if test_alert_rules_loaded "$prometheus_port"; then
        test_results+=("rules_loading:PASS")
    else
        test_results+=("rules_loading:FAIL")
    fi

    # Test 2: Alert Rules Evaluation
    if test_alert_rules_evaluation "$prometheus_port"; then
        test_results+=("rules_evaluation:PASS")
    else
        test_results+=("rules_evaluation:FAIL")
    fi

    # Test 3: O-RAN Alert Rules
    if test_oran_alert_rules "$prometheus_port"; then
        test_results+=("oran_rules:PASS")
    else
        test_results+=("oran_rules:FAIL")
    fi

    # Test 4: AlertManager Health
    if test_alertmanager_health "$alertmanager_port"; then
        test_results+=("alertmanager_health:PASS")
    else
        test_results+=("alertmanager_health:FAIL")
    fi

    # Test 5: AlertManager Configuration
    if test_alertmanager_config "$alertmanager_port"; then
        test_results+=("alertmanager_config:PASS")
    else
        test_results+=("alertmanager_config:FAIL")
    fi

    # Test 6: Alert Management
    if test_alert_management "$alertmanager_port"; then
        test_results+=("alert_management:PASS")
    else
        test_results+=("alert_management:FAIL")
    fi

    # Test 7: AlertManager Cluster
    if test_alertmanager_cluster "$alertmanager_port"; then
        test_results+=("alertmanager_cluster:PASS")
    else
        test_results+=("alertmanager_cluster:FAIL")
    fi

    # Test 8: Prometheus-AlertManager Connection
    if test_prometheus_alertmanager_connection "$prometheus_port"; then
        test_results+=("prometheus_alertmanager:PASS")
    else
        test_results+=("prometheus_alertmanager:FAIL")
    fi

    # Summary
    echo
    log_info "=== Test Results Summary ==="
    local passed=0
    local failed=0

    for result in "${test_results[@]}"; do
        local test_name="${result%:*}"
        local test_status="${result#*:}"

        if [[ "$test_status" == "PASS" ]]; then
            log_success "✓ $test_name"
            ((passed++))
        else
            log_error "✗ $test_name"
            ((failed++))
        fi
    done

    echo
    log_info "Passed: $passed, Failed: $failed"

    # Generate summary JSON
    local summary
    summary=$(jq -n --argjson passed "$passed" --argjson failed "$failed" --arg timestamp "$(date -u +%Y-%m-%dT%H:%M:%S.%3NZ)" '{
        timestamp: $timestamp,
        total_tests: ($passed + $failed),
        passed: $passed,
        failed: $failed,
        success_rate: (($passed / ($passed + $failed)) * 100),
        status: (if $failed == 0 then "healthy" else "degraded" end)
    }')

    echo "HEALTH_CHECK_SUMMARY: $summary"

    if [[ $failed -eq 0 ]]; then
        log_success "=== All alert system tests passed! ==="
        exit 0
    else
        log_error "=== Some alert system tests failed ==="
        exit 1
    fi
}

# Error handling
handle_error() {
    local exit_code=$?
    log_error "Alert system test failed with exit code $exit_code"

    # Cleanup port-forwards if they exist
    if [[ -n "${port_info:-}" ]]; then
        cleanup_port_forwards "$port_info"
    fi

    exit $exit_code
}

trap handle_error ERR

# Help function
show_help() {
    cat << EOF
O-RAN Alert Rules and AlertManager Health Check

Usage: $0 [OPTIONS]

OPTIONS:
    -n, --namespace NAMESPACE       Kubernetes namespace (default: oran-monitoring)
    -p, --prometheus SERVICE       Prometheus service name (default: prometheus)
    -a, --alertmanager SERVICE     AlertManager service name (default: alertmanager)
    -t, --timeout TIMEOUT         Query timeout in seconds (default: 60)
    -m, --min-rules MIN           Minimum alert rules required (default: 5)
    -g, --expected-groups GROUPS  Comma-separated list of expected rule groups
    -h, --help                    Show this help message

ENVIRONMENT VARIABLES:
    MONITORING_NAMESPACE          Same as --namespace
    PROMETHEUS_SERVICE           Same as --prometheus
    ALERTMANAGER_SERVICE         Same as --alertmanager
    TIMEOUT                      Same as --timeout
    MIN_ALERT_RULES              Same as --min-rules
    EXPECTED_ALERT_GROUPS        Same as --expected-groups

EXAMPLES:
    # Basic alert system check
    $0

    # Check with custom minimum rules
    $0 --min-rules 10

    # Check for specific rule groups
    $0 --expected-groups "oran.rules,kubernetes.rules"

EXIT CODES:
    0  - All tests passed
    1  - Some tests failed
    2  - Invalid arguments

EOF
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -p|--prometheus)
                PROMETHEUS_SERVICE="$2"
                shift 2
                ;;
            -a|--alertmanager)
                ALERTMANAGER_SERVICE="$2"
                shift 2
                ;;
            -t|--timeout)
                TIMEOUT="$2"
                shift 2
                ;;
            -m|--min-rules)
                MIN_ALERT_RULES="$2"
                shift 2
                ;;
            -g|--expected-groups)
                EXPECTED_ALERT_GROUPS="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done
}

# Check dependencies
check_dependencies() {
    local missing_deps=()

    for cmd in kubectl curl jq lsof; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again"
        exit 1
    fi
}

# Entry point
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    parse_args "$@"
    check_dependencies
    main
fi