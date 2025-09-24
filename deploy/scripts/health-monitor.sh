#!/bin/bash
# O-RAN Intent-MANO Health Monitor
# Continuous health monitoring and alerting

set -euo pipefail

# Configuration
MONITOR_INTERVAL="${MONITOR_INTERVAL:-30}"
RESULTS_DIR="${RESULTS_DIR:-/results}"
LOG_LEVEL="${LOG_LEVEL:-INFO}"

# Service endpoints to monitor
declare -A SERVICES=(
    ["orchestrator"]="http://orchestrator:8080/health"
    ["vnf-operator"]="http://vnf-operator:8081/healthz"
    ["o2-client"]="http://o2-client:8080/health"
    ["tn-manager"]="http://tn-manager:8080/health"
    ["tn-agent-edge01"]="http://tn-agent-edge01:8080/health"
    ["tn-agent-edge02"]="http://tn-agent-edge02:8080/health"
    ["ran-dms"]="http://ran-dms:8080/health"
    ["cn-dms"]="http://cn-dms:8080/health"
    ["prometheus"]="http://prometheus:9090/-/healthy"
    ["grafana"]="http://grafana:3000/api/health"
)

# Metrics endpoints
declare -A METRICS=(
    ["orchestrator"]="http://orchestrator:9090/metrics"
    ["vnf-operator"]="http://vnf-operator:8080/metrics"
    ["o2-client"]="http://o2-client:9090/metrics"
    ["tn-manager"]="http://tn-manager:9090/metrics"
    ["ran-dms"]="http://ran-dms:9090/metrics"
    ["cn-dms"]="http://cn-dms:9090/metrics"
)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log() {
    local level="$1"
    shift
    echo "$(date '+%Y-%m-%d %H:%M:%S') [$level] $*"
}

log_info() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO)$ ]] && log "INFO" "$@"
}

log_warn() {
    [[ "$LOG_LEVEL" =~ ^(DEBUG|INFO|WARN)$ ]] && log "WARN" "$@" >&2
}

log_error() {
    log "ERROR" "$@" >&2
}

log_debug() {
    [[ "$LOG_LEVEL" == "DEBUG" ]] && log "DEBUG" "$@"
}

# Install required tools
install_dependencies() {
    apk add --no-cache curl wget jq nc-openbsd >/dev/null 2>&1 || true
}

# Check service health
check_service_health() {
    local service="$1"
    local endpoint="$2"
    local timeout="${3:-5}"

    log_debug "Checking health for $service: $endpoint"

    if curl -f -s --max-time "$timeout" "$endpoint" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# Get service metrics
get_service_metrics() {
    local service="$1"
    local endpoint="$2"
    local timeout="${3:-10}"

    log_debug "Fetching metrics for $service: $endpoint"

    if curl -f -s --max-time "$timeout" "$endpoint" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Check network connectivity between services
check_connectivity() {
    local from_service="$1"
    local to_service="$2"
    local to_port="${3:-8080}"

    log_debug "Checking connectivity: $from_service -> $to_service:$to_port"

    if nc -z -w 5 "$to_service" "$to_port" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Perform comprehensive health check
perform_health_check() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local results_file="$RESULTS_DIR/health-check-$(date +%s).json"

    log_info "Performing comprehensive health check..."

    # Initialize results
    cat > "$results_file" << EOF
{
  "timestamp": "$timestamp",
  "monitor_version": "1.0.0",
  "services": {
EOF

    local service_count=0
    local healthy_count=0

    # Check each service
    for service in "${!SERVICES[@]}"; do
        local endpoint="${SERVICES[$service]}"
        local status="UNKNOWN"
        local response_time="0"
        local error_message=""

        service_count=$((service_count + 1))

        # Measure response time
        local start_time=$(date +%s%3N)

        if check_service_health "$service" "$endpoint"; then
            status="HEALTHY"
            healthy_count=$((healthy_count + 1))
        else
            status="UNHEALTHY"
            error_message="Health check failed"
        fi

        local end_time=$(date +%s%3N)
        response_time=$((end_time - start_time))

        # Add comma if not first service
        if [ $service_count -gt 1 ]; then
            echo "    }," >> "$results_file"
        fi

        cat >> "$results_file" << EOF
    "$service": {
      "status": "$status",
      "endpoint": "$endpoint",
      "response_time_ms": $response_time,
      "error": "$error_message",
      "last_check": "$timestamp"
EOF
    done

    cat >> "$results_file" << EOF
    }
  },
  "summary": {
    "total_services": $service_count,
    "healthy_services": $healthy_count,
    "unhealthy_services": $((service_count - healthy_count)),
    "overall_status": "$([[ $healthy_count -eq $service_count ]] && echo "HEALTHY" || echo "UNHEALTHY")"
  }
}
EOF

    log_info "Health check completed: $healthy_count/$service_count services healthy"

    # Log unhealthy services
    if [ $healthy_count -lt $service_count ]; then
        log_warn "Unhealthy services detected:"
        for service in "${!SERVICES[@]}"; do
            if ! check_service_health "$service" "${SERVICES[$service]}" 2; then
                log_error "  - $service (${SERVICES[$service]})"
            fi
        done
    fi

    echo "$results_file"
}

# Check connectivity between services
check_service_connectivity() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local results_file="$RESULTS_DIR/connectivity-check-$(date +%s).json"

    log_info "Checking inter-service connectivity..."

    cat > "$results_file" << EOF
{
  "timestamp": "$timestamp",
  "connectivity_tests": [
EOF

    local test_count=0
    local passed_count=0

    # Define connectivity tests
    local tests=(
        "orchestrator:ran-dms:8080"
        "orchestrator:cn-dms:8080"
        "vnf-operator:ran-dms:8080"
        "o2-client:ran-dms:8080"
        "o2-client:cn-dms:8080"
        "tn-manager:orchestrator:8080"
        "tn-agent-edge01:tn-manager:8080"
        "tn-agent-edge02:tn-manager:8080"
        "tn-agent-edge01:tn-agent-edge02:8080"
        "tn-agent-edge02:tn-agent-edge01:8080"
    )

    for test in "${tests[@]}"; do
        IFS=':' read -r from_service to_service to_port <<< "$test"

        test_count=$((test_count + 1))

        local status="FAIL"
        local error_message=""

        if check_connectivity "$from_service" "$to_service" "$to_port"; then
            status="PASS"
            passed_count=$((passed_count + 1))
        else
            error_message="Connection failed"
        fi

        # Add comma if not first test
        if [ $test_count -gt 1 ]; then
            echo "    }," >> "$results_file"
        fi

        cat >> "$results_file" << EOF
    {
      "test": "$from_service -> $to_service:$to_port",
      "status": "$status",
      "error": "$error_message",
      "timestamp": "$timestamp"
EOF
    done

    cat >> "$results_file" << EOF
    }
  ],
  "summary": {
    "total_tests": $test_count,
    "passed_tests": $passed_count,
    "failed_tests": $((test_count - passed_count)),
    "success_rate": "$(( (passed_count * 100) / test_count ))%"
  }
}
EOF

    log_info "Connectivity check completed: $passed_count/$test_count tests passed"

    echo "$results_file"
}

# Collect system metrics
collect_system_metrics() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local results_file="$RESULTS_DIR/system-metrics-$(date +%s).json"

    log_info "Collecting system metrics..."

    cat > "$results_file" << EOF
{
  "timestamp": "$timestamp",
  "system": {
    "hostname": "$(hostname)",
    "uptime": "$(cat /proc/uptime | cut -d' ' -f1)",
    "load_average": "$(cat /proc/loadavg | cut -d' ' -f1-3)",
    "memory": {
EOF

    # Memory information
    local mem_total=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local mem_free=$(grep MemFree /proc/meminfo | awk '{print $2}')
    local mem_available=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
    local mem_used=$((mem_total - mem_free))

    cat >> "$results_file" << EOF
      "total_kb": $mem_total,
      "used_kb": $mem_used,
      "free_kb": $mem_free,
      "available_kb": $mem_available,
      "usage_percent": $(( (mem_used * 100) / mem_total ))
    },
    "disk": {
EOF

    # Disk information
    local disk_info=$(df -k / | tail -1)
    local disk_total=$(echo "$disk_info" | awk '{print $2}')
    local disk_used=$(echo "$disk_info" | awk '{print $3}')
    local disk_available=$(echo "$disk_info" | awk '{print $4}')

    cat >> "$results_file" << EOF
      "total_kb": $disk_total,
      "used_kb": $disk_used,
      "available_kb": $disk_available,
      "usage_percent": $(( (disk_used * 100) / disk_total ))
    }
  },
  "services_metrics": {
EOF

    local metric_count=0

    # Collect metrics from each service
    for service in "${!METRICS[@]}"; do
        local endpoint="${METRICS[$service]}"

        metric_count=$((metric_count + 1))

        # Add comma if not first service
        if [ $metric_count -gt 1 ]; then
            echo "    }," >> "$results_file"
        fi

        echo "    \"$service\": {" >> "$results_file"

        if metrics_data=$(get_service_metrics "$service" "$endpoint"); then
            # Extract key metrics
            local http_requests=$(echo "$metrics_data" | grep -E "^http_requests_total" | head -1 | awk '{print $2}' || echo "0")
            local http_duration=$(echo "$metrics_data" | grep -E "^http_request_duration_seconds" | head -1 | awk '{print $2}' || echo "0")
            local process_cpu=$(echo "$metrics_data" | grep -E "^process_cpu_seconds_total" | head -1 | awk '{print $2}' || echo "0")
            local process_memory=$(echo "$metrics_data" | grep -E "^process_resident_memory_bytes" | head -1 | awk '{print $2}' || echo "0")

            cat >> "$results_file" << EOF
      "status": "available",
      "http_requests_total": $http_requests,
      "http_request_duration_seconds": $http_duration,
      "process_cpu_seconds_total": $process_cpu,
      "process_resident_memory_bytes": $process_memory
EOF
        else
            cat >> "$results_file" << EOF
      "status": "unavailable",
      "error": "Failed to fetch metrics"
EOF
        fi
    done

    cat >> "$results_file" << EOF
    }
  }
}
EOF

    log_info "System metrics collected"

    echo "$results_file"
}

# Generate summary report
generate_summary_report() {
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local report_file="$RESULTS_DIR/health-summary-$(date +%Y%m%d).json"

    log_info "Generating summary report..."

    # Find latest health check results
    local latest_health=$(find "$RESULTS_DIR" -name "health-check-*.json" -type f 2>/dev/null | sort | tail -1)
    local latest_connectivity=$(find "$RESULTS_DIR" -name "connectivity-check-*.json" -type f 2>/dev/null | sort | tail -1)
    local latest_metrics=$(find "$RESULTS_DIR" -name "system-metrics-*.json" -type f 2>/dev/null | sort | tail -1)

    cat > "$report_file" << EOF
{
  "timestamp": "$timestamp",
  "report_type": "daily_summary",
  "health_status": {
EOF

    if [[ -f "$latest_health" ]]; then
        local healthy=$(jq -r '.summary.healthy_services' "$latest_health")
        local total=$(jq -r '.summary.total_services' "$latest_health")
        local overall=$(jq -r '.summary.overall_status' "$latest_health")

        cat >> "$report_file" << EOF
    "healthy_services": $healthy,
    "total_services": $total,
    "overall_status": "$overall",
    "last_check": "$(jq -r '.timestamp' "$latest_health")"
EOF
    else
        cat >> "$report_file" << EOF
    "healthy_services": 0,
    "total_services": 0,
    "overall_status": "UNKNOWN",
    "last_check": "never"
EOF
    fi

    cat >> "$report_file" << EOF
  },
  "connectivity_status": {
EOF

    if [[ -f "$latest_connectivity" ]]; then
        local passed=$(jq -r '.summary.passed_tests' "$latest_connectivity")
        local total=$(jq -r '.summary.total_tests' "$latest_connectivity")
        local success_rate=$(jq -r '.summary.success_rate' "$latest_connectivity")

        cat >> "$report_file" << EOF
    "passed_tests": $passed,
    "total_tests": $total,
    "success_rate": "$success_rate",
    "last_check": "$(jq -r '.timestamp' "$latest_connectivity")"
EOF
    else
        cat >> "$report_file" << EOF
    "passed_tests": 0,
    "total_tests": 0,
    "success_rate": "0%",
    "last_check": "never"
EOF
    fi

    cat >> "$report_file" << EOF
  },
  "system_status": {
EOF

    if [[ -f "$latest_metrics" ]]; then
        local mem_usage=$(jq -r '.system.memory.usage_percent' "$latest_metrics")
        local disk_usage=$(jq -r '.system.disk.usage_percent' "$latest_metrics")
        local uptime=$(jq -r '.system.uptime' "$latest_metrics")

        cat >> "$report_file" << EOF
    "memory_usage_percent": $mem_usage,
    "disk_usage_percent": $disk_usage,
    "uptime_seconds": $uptime,
    "last_check": "$(jq -r '.timestamp' "$latest_metrics")"
EOF
    else
        cat >> "$report_file" << EOF
    "memory_usage_percent": 0,
    "disk_usage_percent": 0,
    "uptime_seconds": 0,
    "last_check": "never"
EOF
    fi

    cat >> "$report_file" << EOF
  },
  "recommendations": [
EOF

    # Add recommendations based on current status
    local recommendations=()

    if [[ -f "$latest_health" ]] && [[ $(jq -r '.summary.unhealthy_services' "$latest_health") -gt 0 ]]; then
        recommendations+=("\"Investigate unhealthy services and restart if necessary\"")
    fi

    if [[ -f "$latest_connectivity" ]] && [[ $(jq -r '.summary.failed_tests' "$latest_connectivity") -gt 0 ]]; then
        recommendations+=("\"Check network configuration and firewall rules\"")
    fi

    if [[ -f "$latest_metrics" ]]; then
        local mem_usage=$(jq -r '.system.memory.usage_percent' "$latest_metrics")
        local disk_usage=$(jq -r '.system.disk.usage_percent' "$latest_metrics")

        if [[ $mem_usage -gt 80 ]]; then
            recommendations+=("\"Memory usage is high ($mem_usage%), consider scaling resources\"")
        fi

        if [[ $disk_usage -gt 80 ]]; then
            recommendations+=("\"Disk usage is high ($disk_usage%), clean up logs and data\"")
        fi
    fi

    if [[ ${#recommendations[@]} -eq 0 ]]; then
        recommendations+=("\"All systems operating normally\"")
    fi

    # Join recommendations with commas
    printf '    %s' "${recommendations[0]}"
    for ((i=1; i<${#recommendations[@]}; i++)); do
        printf ',\n    %s' "${recommendations[i]}"
    done

    cat >> "$report_file" << EOF

  ]
}
EOF

    log_info "Summary report generated: $report_file"

    echo "$report_file"
}

# Main monitoring loop
main_loop() {
    log_info "Starting O-RAN MANO Health Monitor"
    log_info "Monitor interval: ${MONITOR_INTERVAL}s"
    log_info "Results directory: $RESULTS_DIR"

    # Install dependencies
    install_dependencies

    # Create results directory
    mkdir -p "$RESULTS_DIR"

    # Main monitoring loop
    while true; do
        log_debug "Starting monitoring cycle..."

        # Perform health check
        health_result=$(perform_health_check)
        log_debug "Health check completed: $health_result"

        # Check connectivity
        connectivity_result=$(check_service_connectivity)
        log_debug "Connectivity check completed: $connectivity_result"

        # Collect system metrics
        metrics_result=$(collect_system_metrics)
        log_debug "System metrics collected: $metrics_result"

        # Generate daily summary (once per day)
        local current_date=$(date +%Y%m%d)
        local summary_file="$RESULTS_DIR/health-summary-$current_date.json"

        if [[ ! -f "$summary_file" ]] || [[ $(find "$summary_file" -mmin +1440 2>/dev/null) ]]; then
            summary_result=$(generate_summary_report)
            log_debug "Summary report generated: $summary_result"
        fi

        # Clean up old files (keep last 7 days)
        find "$RESULTS_DIR" -name "*.json" -type f -mtime +7 -delete 2>/dev/null || true

        log_debug "Monitoring cycle completed, waiting ${MONITOR_INTERVAL}s..."
        sleep "$MONITOR_INTERVAL"
    done
}

# Signal handlers
cleanup() {
    log_info "Received termination signal, shutting down..."
    exit 0
}

trap cleanup SIGTERM SIGINT

# Start monitoring
main_loop