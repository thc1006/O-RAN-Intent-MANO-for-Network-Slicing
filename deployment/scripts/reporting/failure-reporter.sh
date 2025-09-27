#!/bin/bash

# Comprehensive Failure Reporting for O-RAN MANO CI/CD Pipeline
# This script provides detailed failure analysis and reporting

set -euo pipefail

# Configuration
NAMESPACE="${NAMESPACE:-monitoring}"
REPORT_DIR="${REPORT_DIR:-./failure-reports}"
GITHUB_TOKEN="${GITHUB_TOKEN:-}"
SLACK_WEBHOOK="${SLACK_WEBHOOK:-}"
TEAMS_WEBHOOK="${TEAMS_WEBHOOK:-}"
EMAIL_SMTP_HOST="${EMAIL_SMTP_HOST:-}"
EMAIL_FROM="${EMAIL_FROM:-noreply@oran-mano.local}"
EMAIL_TO="${EMAIL_TO:-ops-team@oran-mano.local}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

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

# Initialize reporting
initialize_reporting() {
    log "Initializing failure reporting..."

    mkdir -p "$REPORT_DIR"

    # Create report metadata
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    export REPORT_ID="failure_report_$timestamp"
    export REPORT_FILE="$REPORT_DIR/${REPORT_ID}.md"
    export REPORT_JSON="$REPORT_DIR/${REPORT_ID}.json"

    success "Report ID: $REPORT_ID"
}

# Collect system information
collect_system_info() {
    log "Collecting system information..."

    local system_info
    system_info=$(cat << EOF
{
  "timestamp": "$(date -Iseconds)",
  "system": {
    "hostname": "$(hostname)",
    "kernel": "$(uname -r)",
    "os": "$(uname -o)",
    "uptime": "$(uptime -p)",
    "load_average": "$(uptime | awk -F'load average:' '{print $2}')"
  },
  "kubernetes": {
    "context": "$(kubectl config current-context 2>/dev/null || echo 'unknown')",
    "server_version": "$(kubectl version --short 2>/dev/null | grep 'Server Version' || echo 'unknown')",
    "node_count": $(kubectl get nodes --no-headers 2>/dev/null | wc -l || echo 0),
    "namespace_count": $(kubectl get namespaces --no-headers 2>/dev/null | wc -l || echo 0)
  }
}
EOF
    )

    echo "$system_info" > "$REPORT_JSON"
    success "System information collected"
}

# Collect cluster state
collect_cluster_state() {
    log "Collecting cluster state..."

    local cluster_state_file="$REPORT_DIR/${REPORT_ID}_cluster_state.yaml"

    {
        echo "# Cluster State Report - $(date)"
        echo "---"
        echo "# Namespaces"
        kubectl get namespaces -o yaml 2>/dev/null || echo "Failed to get namespaces"
        echo "---"
        echo "# Nodes"
        kubectl get nodes -o yaml 2>/dev/null || echo "Failed to get nodes"
        echo "---"
        echo "# Pods in monitoring namespace"
        kubectl get pods -n "$NAMESPACE" -o yaml 2>/dev/null || echo "Failed to get pods"
        echo "---"
        echo "# Services in monitoring namespace"
        kubectl get services -n "$NAMESPACE" -o yaml 2>/dev/null || echo "Failed to get services"
        echo "---"
        echo "# Events in monitoring namespace"
        kubectl get events -n "$NAMESPACE" --sort-by='.lastTimestamp' 2>/dev/null || echo "Failed to get events"
    } > "$cluster_state_file"

    # Update JSON report
    jq --arg file "$cluster_state_file" '.cluster_state_file = $file' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"

    success "Cluster state collected: $cluster_state_file"
}

# Collect pod logs
collect_pod_logs() {
    log "Collecting pod logs..."

    local logs_dir="$REPORT_DIR/${REPORT_ID}_logs"
    mkdir -p "$logs_dir"

    # Get all pods in monitoring namespace
    local pods
    pods=$(kubectl get pods -n "$NAMESPACE" -o name 2>/dev/null || echo "")

    if [ -z "$pods" ]; then
        warning "No pods found in namespace $NAMESPACE"
        return
    fi

    local log_files=()

    for pod in $pods; do
        local pod_name
        pod_name=$(echo "$pod" | cut -d'/' -f2)

        log "Collecting logs for pod: $pod_name"

        # Get current logs
        kubectl logs "$pod" -n "$NAMESPACE" --tail=500 > "$logs_dir/${pod_name}_current.log" 2>/dev/null || echo "Failed to get current logs for $pod_name" > "$logs_dir/${pod_name}_current.log"

        # Get previous logs (if container restarted)
        kubectl logs "$pod" -n "$NAMESPACE" --previous --tail=500 > "$logs_dir/${pod_name}_previous.log" 2>/dev/null || echo "No previous logs for $pod_name" > "$logs_dir/${pod_name}_previous.log"

        # Get container logs for multi-container pods
        local containers
        containers=$(kubectl get pod "$pod_name" -n "$NAMESPACE" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")

        for container in $containers; do
            if [ -n "$container" ]; then
                kubectl logs "$pod_name" -n "$NAMESPACE" -c "$container" --tail=500 > "$logs_dir/${pod_name}_${container}.log" 2>/dev/null || true
            fi
        done

        log_files+=("$logs_dir/${pod_name}_current.log")
    done

    # Update JSON report
    local log_files_json
    log_files_json=$(printf '%s\n' "${log_files[@]}" | jq -R . | jq -s .)
    jq --argjson files "$log_files_json" '.log_files = $files' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"

    success "Pod logs collected in: $logs_dir"
}

# Analyze pod failures
analyze_pod_failures() {
    log "Analyzing pod failures..."

    local failed_pods
    failed_pods=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running -o json 2>/dev/null || echo '{"items":[]}')

    local pod_failures=()

    echo "$failed_pods" | jq -c '.items[]' | while read -r pod; do
        local pod_name
        pod_name=$(echo "$pod" | jq -r '.metadata.name')

        local pod_status
        pod_status=$(echo "$pod" | jq -r '.status.phase')

        local restart_count
        restart_count=$(echo "$pod" | jq -r '.status.containerStatuses[0].restartCount // 0')

        local last_state
        last_state=$(echo "$pod" | jq -r '.status.containerStatuses[0].lastState.terminated.reason // "Unknown"')

        local failure_info
        failure_info=$(cat << EOF
{
  "pod_name": "$pod_name",
  "status": "$pod_status",
  "restart_count": $restart_count,
  "last_termination_reason": "$last_state",
  "events": []
}
EOF
        )

        # Get related events
        local pod_events
        pod_events=$(kubectl get events -n "$NAMESPACE" --field-selector involvedObject.name="$pod_name" -o json 2>/dev/null || echo '{"items":[]}')

        local events_array
        events_array=$(echo "$pod_events" | jq '[.items[] | {reason: .reason, message: .message, timestamp: .firstTimestamp}]')

        failure_info=$(echo "$failure_info" | jq --argjson events "$events_array" '.events = $events')
        pod_failures+=("$failure_info")
    done

    # Update JSON report with pod failures
    if [ ${#pod_failures[@]} -gt 0 ]; then
        local failures_json
        failures_json=$(printf '%s\n' "${pod_failures[@]}" | jq -s .)
        jq --argjson failures "$failures_json" '.pod_failures = $failures' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"

        error "Found ${#pod_failures[@]} failed pods"
    else
        jq '.pod_failures = []' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"
        success "No failed pods found"
    fi
}

# Analyze service issues
analyze_service_issues() {
    log "Analyzing service issues..."

    local services
    services=$(kubectl get services -n "$NAMESPACE" -o json 2>/dev/null || echo '{"items":[]}')

    local service_issues=()

    echo "$services" | jq -c '.items[]' | while read -r service; do
        local service_name
        service_name=$(echo "$service" | jq -r '.metadata.name')

        local service_type
        service_type=$(echo "$service" | jq -r '.spec.type')

        # Check endpoints
        local endpoints
        endpoints=$(kubectl get endpoints "$service_name" -n "$NAMESPACE" -o json 2>/dev/null || echo '{"subsets":[]}')

        local endpoint_count
        endpoint_count=$(echo "$endpoints" | jq '.subsets[0].addresses | length // 0')

        if [ "$endpoint_count" -eq 0 ]; then
            local issue_info
            issue_info=$(cat << EOF
{
  "service_name": "$service_name",
  "service_type": "$service_type",
  "issue": "No endpoints available",
  "endpoint_count": $endpoint_count
}
EOF
            )
            service_issues+=("$issue_info")
        fi
    done

    # Update JSON report with service issues
    if [ ${#service_issues[@]} -gt 0 ]; then
        local issues_json
        issues_json=$(printf '%s\n' "${service_issues[@]}" | jq -s .)
        jq --argjson issues "$issues_json" '.service_issues = $issues' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"

        error "Found ${#service_issues[@]} service issues"
    else
        jq '.service_issues = []' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"
        success "No service issues found"
    fi
}

# Check monitoring stack health
check_monitoring_health() {
    log "Checking monitoring stack health..."

    local health_status
    health_status=$(cat << 'EOF'
{
  "prometheus": {"status": "unknown", "details": ""},
  "grafana": {"status": "unknown", "details": ""},
  "alertmanager": {"status": "unknown", "details": ""}
}
EOF
    )

    # Check Prometheus
    if kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus &>/dev/null; then
        local prom_pods
        prom_pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=prometheus --no-headers | grep -c "Running" || echo "0")

        if [ "$prom_pods" -gt 0 ]; then
            health_status=$(echo "$health_status" | jq '.prometheus.status = "healthy"')
            health_status=$(echo "$health_status" | jq --arg pods "$prom_pods" '.prometheus.details = ($pods + " pods running")')
        else
            health_status=$(echo "$health_status" | jq '.prometheus.status = "unhealthy"')
            health_status=$(echo "$health_status" | jq '.prometheus.details = "No running pods"')
        fi
    else
        health_status=$(echo "$health_status" | jq '.prometheus.status = "missing"')
        health_status=$(echo "$health_status" | jq '.prometheus.details = "No Prometheus pods found"')
    fi

    # Check Grafana
    if kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=grafana &>/dev/null; then
        local grafana_pods
        grafana_pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=grafana --no-headers | grep -c "Running" || echo "0")

        if [ "$grafana_pods" -gt 0 ]; then
            health_status=$(echo "$health_status" | jq '.grafana.status = "healthy"')
            health_status=$(echo "$health_status" | jq --arg pods "$grafana_pods" '.grafana.details = ($pods + " pods running")')
        else
            health_status=$(echo "$health_status" | jq '.grafana.status = "unhealthy"')
            health_status=$(echo "$health_status" | jq '.grafana.details = "No running pods"')
        fi
    else
        health_status=$(echo "$health_status" | jq '.grafana.status = "missing"')
        health_status=$(echo "$health_status" | jq '.grafana.details = "No Grafana pods found"')
    fi

    # Check AlertManager
    if kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager &>/dev/null; then
        local am_pods
        am_pods=$(kubectl get pods -n "$NAMESPACE" -l app.kubernetes.io/name=alertmanager --no-headers | grep -c "Running" || echo "0")

        if [ "$am_pods" -gt 0 ]; then
            health_status=$(echo "$health_status" | jq '.alertmanager.status = "healthy"')
            health_status=$(echo "$health_status" | jq --arg pods "$am_pods" '.alertmanager.details = ($pods + " pods running")')
        else
            health_status=$(echo "$health_status" | jq '.alertmanager.status = "unhealthy"')
            health_status=$(echo "$health_status" | jq '.alertmanager.details = "No running pods"')
        fi
    else
        health_status=$(echo "$health_status" | jq '.alertmanager.status = "missing"')
        health_status=$(echo "$health_status" | jq '.alertmanager.details = "No AlertManager pods found"')
    fi

    # Update JSON report
    jq --argjson health "$health_status" '.monitoring_health = $health' "$REPORT_JSON" > /tmp/report.json && mv /tmp/report.json "$REPORT_JSON"

    success "Monitoring health check completed"
}

# Generate markdown report
generate_markdown_report() {
    log "Generating markdown report..."

    local json_data
    json_data=$(cat "$REPORT_JSON")

    cat > "$REPORT_FILE" << EOF
# O-RAN MANO Failure Analysis Report

**Report ID**: $(echo "$json_data" | jq -r '.timestamp')
**Generated**: $(date)

## Executive Summary

$(
pod_failure_count=$(echo "$json_data" | jq '.pod_failures | length')
service_issue_count=$(echo "$json_data" | jq '.service_issues | length')

if [ "$pod_failure_count" -eq 0 ] && [ "$service_issue_count" -eq 0 ]; then
    echo "✅ **System Status**: Healthy - No critical issues detected"
else
    echo "❌ **System Status**: Issues Detected"
    [ "$pod_failure_count" -gt 0 ] && echo "- $pod_failure_count pod failures"
    [ "$service_issue_count" -gt 0 ] && echo "- $service_issue_count service issues"
fi
)

## System Information

- **Hostname**: $(echo "$json_data" | jq -r '.system.hostname')
- **Kernel**: $(echo "$json_data" | jq -r '.system.kernel')
- **Uptime**: $(echo "$json_data" | jq -r '.system.uptime')
- **Load Average**: $(echo "$json_data" | jq -r '.system.load_average')

## Kubernetes Cluster

- **Context**: $(echo "$json_data" | jq -r '.kubernetes.context')
- **Server Version**: $(echo "$json_data" | jq -r '.kubernetes.server_version')
- **Node Count**: $(echo "$json_data" | jq -r '.kubernetes.node_count')
- **Namespace Count**: $(echo "$json_data" | jq -r '.kubernetes.namespace_count')

## Monitoring Stack Health

$(
echo "$json_data" | jq -r '.monitoring_health | to_entries[] | "- **\(.key | ascii_upcase)**: \(.value.status) - \(.value.details)"'
)

## Pod Failures

$(
if [ "$(echo "$json_data" | jq '.pod_failures | length')" -eq 0 ]; then
    echo "✅ No pod failures detected"
else
    echo "❌ Pod failures detected:"
    echo ""
    echo "$json_data" | jq -r '.pod_failures[] | "### \(.pod_name)\n- **Status**: \(.status)\n- **Restart Count**: \(.restart_count)\n- **Last Termination**: \(.last_termination_reason)\n- **Recent Events**: \(.events | length) events\n"'
fi
)

## Service Issues

$(
if [ "$(echo "$json_data" | jq '.service_issues | length')" -eq 0 ]; then
    echo "✅ No service issues detected"
else
    echo "❌ Service issues detected:"
    echo ""
    echo "$json_data" | jq -r '.service_issues[] | "### \(.service_name)\n- **Type**: \(.service_type)\n- **Issue**: \(.issue)\n- **Endpoints**: \(.endpoint_count)\n"'
fi
)

## Files Generated

- **JSON Report**: $(basename "$REPORT_JSON")
- **Cluster State**: $(basename "$(echo "$json_data" | jq -r '.cluster_state_file // "N/A")")
- **Pod Logs**: $(basename "$REPORT_DIR/${REPORT_ID}_logs")

## Recommended Actions

$(
pod_failure_count=$(echo "$json_data" | jq '.pod_failures | length')
service_issue_count=$(echo "$json_data" | jq '.service_issues | length')

if [ "$pod_failure_count" -gt 0 ]; then
    echo "1. **Pod Failures**: Investigate failed pods and check logs for root cause"
fi

if [ "$service_issue_count" -gt 0 ]; then
    echo "2. **Service Issues**: Check service endpoints and pod selectors"
fi

if [ "$pod_failure_count" -eq 0 ] && [ "$service_issue_count" -eq 0 ]; then
    echo "1. **Monitor**: Continue monitoring system health"
    echo "2. **Performance**: Check performance metrics for optimization opportunities"
fi

echo "3. **Review**: Review recent changes that might have caused issues"
echo "4. **Escalate**: Escalate to engineering team if issues persist"
)

---
*This report was generated automatically by the O-RAN MANO failure reporting system.*
EOF

    success "Markdown report generated: $REPORT_FILE"
}

# Send notifications
send_notifications() {
    local severity="${1:-info}"
    local summary="${2:-Failure report generated}"

    log "Sending notifications (severity: $severity)..."

    # Send Slack notification
    if [ -n "$SLACK_WEBHOOK" ]; then
        send_slack_notification "$severity" "$summary"
    fi

    # Send Teams notification
    if [ -n "$TEAMS_WEBHOOK" ]; then
        send_teams_notification "$severity" "$summary"
    fi

    # Send email notification
    if [ -n "$EMAIL_SMTP_HOST" ]; then
        send_email_notification "$severity" "$summary"
    fi

    # Create GitHub issue
    if [ -n "$GITHUB_TOKEN" ]; then
        create_github_issue "$severity" "$summary"
    fi
}

# Send Slack notification
send_slack_notification() {
    local severity="$1"
    local summary="$2"

    local color="good"
    local emoji="✅"

    case "$severity" in
        "critical"|"error")
            color="danger"
            emoji="🚨"
            ;;
        "warning")
            color="warning"
            emoji="⚠️"
            ;;
    esac

    local payload
    payload=$(cat << EOF
{
  "text": "$emoji O-RAN MANO Failure Report",
  "attachments": [
    {
      "color": "$color",
      "title": "Failure Analysis Report",
      "fields": [
        {
          "title": "Summary",
          "value": "$summary",
          "short": false
        },
        {
          "title": "Report ID",
          "value": "$REPORT_ID",
          "short": true
        },
        {
          "title": "Timestamp",
          "value": "$(date)",
          "short": true
        }
      ],
      "actions": [
        {
          "type": "button",
          "text": "View Report",
          "url": "file://$REPORT_FILE"
        }
      ]
    }
  ]
}
EOF
    )

    curl -X POST -H 'Content-type: application/json' \
        --data "$payload" \
        "$SLACK_WEBHOOK" &>/dev/null && success "Slack notification sent" || warning "Failed to send Slack notification"
}

# Send Teams notification
send_teams_notification() {
    local severity="$1"
    local summary="$2"

    local color="00FF00"
    local emoji="✅"

    case "$severity" in
        "critical"|"error")
            color="FF0000"
            emoji="🚨"
            ;;
        "warning")
            color="FFA500"
            emoji="⚠️"
            ;;
    esac

    local payload
    payload=$(cat << EOF
{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "$color",
  "summary": "O-RAN MANO Failure Report",
  "sections": [{
    "activityTitle": "$emoji O-RAN MANO Failure Analysis Report",
    "activitySubtitle": "$summary",
    "facts": [
      {
        "name": "Report ID",
        "value": "$REPORT_ID"
      },
      {
        "name": "Timestamp",
        "value": "$(date)"
      },
      {
        "name": "Severity",
        "value": "$severity"
      }
    ],
    "markdown": true
  }]
}
EOF
    )

    curl -X POST -H 'Content-type: application/json' \
        --data "$payload" \
        "$TEAMS_WEBHOOK" &>/dev/null && success "Teams notification sent" || warning "Failed to send Teams notification"
}

# Send email notification
send_email_notification() {
    local severity="$1"
    local summary="$2"

    if ! command -v sendmail &> /dev/null; then
        warning "sendmail not available, skipping email notification"
        return
    fi

    local subject="[O-RAN MANO] Failure Report - $severity"

    local email_content
    email_content=$(cat << EOF
To: $EMAIL_TO
From: $EMAIL_FROM
Subject: $subject

O-RAN MANO Failure Analysis Report

Summary: $summary
Report ID: $REPORT_ID
Timestamp: $(date)
Severity: $severity

Report Location: $REPORT_FILE

Please review the attached report for detailed information.

---
This is an automated message from the O-RAN MANO monitoring system.
EOF
    )

    echo "$email_content" | sendmail "$EMAIL_TO" && success "Email notification sent" || warning "Failed to send email notification"
}

# Create GitHub issue
create_github_issue() {
    local severity="$1"
    local summary="$2"

    if [ -z "$GITHUB_REPOSITORY" ]; then
        warning "GITHUB_REPOSITORY not set, skipping GitHub issue creation"
        return
    fi

    local title="[FAILURE] $summary"
    local body
    body=$(cat << EOF
## Failure Report

**Report ID**: $REPORT_ID
**Timestamp**: $(date)
**Severity**: $severity

### Summary
$summary

### Files Generated
- Report: $REPORT_FILE
- Data: $REPORT_JSON

### Next Steps
- [ ] Investigate root cause
- [ ] Implement fix
- [ ] Verify resolution
- [ ] Update monitoring/alerting if needed

---
*This issue was created automatically by the O-RAN MANO failure reporting system.*
EOF
    )

    local issue_payload
    issue_payload=$(jq -n \
        --arg title "$title" \
        --arg body "$body" \
        --argjson labels '["bug", "monitoring", "ci-cd"]' \
        '{title: $title, body: $body, labels: $labels}'
    )

    curl -X POST \
        -H "Authorization: token $GITHUB_TOKEN" \
        -H "Accept: application/vnd.github.v3+json" \
        "https://api.github.com/repos/$GITHUB_REPOSITORY/issues" \
        -d "$issue_payload" &>/dev/null && success "GitHub issue created" || warning "Failed to create GitHub issue"
}

# Main failure reporting function
main() {
    local failure_type="${1:-general}"
    local severity="${2:-warning}"
    local summary="${3:-Failure detected in O-RAN MANO system}"

    log "Starting comprehensive failure reporting"
    log "Type: $failure_type, Severity: $severity"

    initialize_reporting
    collect_system_info
    collect_cluster_state
    collect_pod_logs
    analyze_pod_failures
    analyze_service_issues
    check_monitoring_health
    generate_markdown_report

    # Determine final severity based on analysis
    local final_severity="$severity"
    local pod_failures
    local service_issues

    pod_failures=$(jq '.pod_failures | length' "$REPORT_JSON")
    service_issues=$(jq '.service_issues | length' "$REPORT_JSON")

    if [ "$pod_failures" -gt 0 ] || [ "$service_issues" -gt 0 ]; then
        final_severity="critical"
    fi

    send_notifications "$final_severity" "$summary"

    success "🎉 Failure reporting completed!"
    log "Report generated: $REPORT_FILE"
    log "JSON data: $REPORT_JSON"

    # Return exit code based on severity
    case "$final_severity" in
        "critical"|"error")
            return 1
            ;;
        *)
            return 0
            ;;
    esac
}

# Handle command line arguments
case "${1:-general}" in
    "deployment")
        main "deployment" "critical" "Deployment failure detected"
        ;;
    "monitoring")
        main "monitoring" "warning" "Monitoring stack issue detected"
        ;;
    "performance")
        main "performance" "warning" "Performance regression detected"
        ;;
    "general"|"")
        main "general" "${2:-info}" "${3:-System health check}"
        ;;
    "help")
        echo "Usage: $0 [failure-type] [severity] [summary]"
        echo ""
        echo "Failure Types:"
        echo "  deployment  - Deployment-related failures"
        echo "  monitoring  - Monitoring stack issues"
        echo "  performance - Performance regressions"
        echo "  general     - General system issues (default)"
        echo ""
        echo "Severity Levels:"
        echo "  critical - Critical system failure"
        echo "  error    - Error condition"
        echo "  warning  - Warning condition (default)"
        echo "  info     - Informational"
        echo ""
        echo "Environment Variables:"
        echo "  NAMESPACE       - Kubernetes namespace (default: monitoring)"
        echo "  REPORT_DIR      - Report output directory (default: ./failure-reports)"
        echo "  SLACK_WEBHOOK   - Slack webhook URL for notifications"
        echo "  TEAMS_WEBHOOK   - Teams webhook URL for notifications"
        echo "  GITHUB_TOKEN    - GitHub token for issue creation"
        echo "  EMAIL_SMTP_HOST - SMTP host for email notifications"
        ;;
    *)
        main "$1" "${2:-warning}" "${3:-Failure detected: $1}"
        ;;
esac