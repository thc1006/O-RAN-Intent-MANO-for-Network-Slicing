#!/bin/bash
# Test Results Collector Script
# Collects and consolidates test results from all services

set -euo pipefail

readonly RESULTS_DIR="/results"
readonly LOGS_DIR="/logs"
readonly TIMESTAMP="$(date +'%Y%m%d_%H%M%S')"

echo "Test Results Collection - ${TIMESTAMP}"
echo "======================================"

# Create results directory structure
mkdir -p "${RESULTS_DIR}/"{reports,logs,artifacts}

# Collect test logs from all services
echo "Collecting test logs..."
for service_log in "${LOGS_DIR}"/*; do
    if [[ -d "${service_log}" ]]; then
        service_name="$(basename "${service_log}")"
        echo "Collecting logs from ${service_name}..."
        cp -r "${service_log}" "${RESULTS_DIR}/logs/${service_name}-${TIMESTAMP}"
    fi
done

# Generate summary report
cat > "${RESULTS_DIR}/test-summary-${TIMESTAMP}.txt" << EOF
Test Execution Summary
======================
Timestamp: ${TIMESTAMP}
Test Environment: Docker Compose Test Suite

Services Tested:
- Test Framework
- Orchestrator
- VNF Operator
- O2 Client
- TN Manager
- TN Agent
- RAN DMS
- CN DMS

Results Location: ${RESULTS_DIR}
Logs Location: ${RESULTS_DIR}/logs/
EOF

echo "Test results collected in: ${RESULTS_DIR}"
echo "Summary available at: ${RESULTS_DIR}/test-summary-${TIMESTAMP}.txt"

# Keep only last 5 result sets to manage disk space
find "${RESULTS_DIR}" -name "*-*" -type d | sort | head -n -5 | xargs rm -rf 2>/dev/null || true

sleep infinity  # Keep container running for log collection