#!/bin/bash
# Container Security Scanning Script
# Runs Trivy, Grype, and Snyk scans on all container images

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCAN_CONFIG="./security-scan.yaml"
REPORT_DIR="./security-reports"
IMAGES=(
    "oran-cn-dms"
    "oran-o2-client"
    "oran-orchestrator"
    "oran-ran-dms"
    "oran-test-framework"
    "oran-tn-agent"
    "oran-tn-manager"
    "oran-vnf-operator"
)

# Create report directory
mkdir -p "$REPORT_DIR"

echo -e "${GREEN}Starting container security scanning...${NC}"

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install security scanners
install_scanners() {
    echo -e "${YELLOW}Installing security scanners...${NC}"

    # Install Trivy
    if ! command_exists trivy; then
        echo "Installing Trivy..."
        if [[ "$OSTYPE" == "linux-gnu"* ]]; then
            curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.48.3
        elif [[ "$OSTYPE" == "darwin"* ]]; then
            brew install trivy
        fi
    fi

    # Install Grype
    if ! command_exists grype; then
        echo "Installing Grype..."
        curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
    fi

    # Install Snyk (requires npm)
    if ! command_exists snyk; then
        echo "Installing Snyk..."
        npm install -g snyk
    fi
}

# Function to run Trivy scan
run_trivy_scan() {
    local image=$1
    echo -e "${YELLOW}Running Trivy scan on $image...${NC}"

    trivy image \
        --severity HIGH,CRITICAL \
        --format json \
        --output "$REPORT_DIR/trivy-$image.json" \
        --config "$SCAN_CONFIG" \
        "$image:latest" || {
        echo -e "${RED}Trivy scan failed for $image${NC}"
        return 1
    }

    # Generate human-readable report
    trivy image \
        --severity HIGH,CRITICAL \
        --format table \
        --output "$REPORT_DIR/trivy-$image.txt" \
        "$image:latest"
}

# Function to run Grype scan
run_grype_scan() {
    local image=$1
    echo -e "${YELLOW}Running Grype scan on $image...${NC}"

    grype "$image:latest" \
        -o json \
        --file "$REPORT_DIR/grype-$image.json" \
        --fail-on high || {
        echo -e "${RED}Grype scan failed for $image${NC}"
        return 1
    }

    # Generate human-readable report
    grype "$image:latest" \
        -o table \
        --file "$REPORT_DIR/grype-$image.txt"
}

# Function to run Snyk scan
run_snyk_scan() {
    local image=$1
    echo -e "${YELLOW}Running Snyk scan on $image...${NC}"

    snyk container test "$image:latest" \
        --json \
        --severity-threshold=high \
        > "$REPORT_DIR/snyk-$image.json" || {
        echo -e "${RED}Snyk scan failed for $image${NC}"
        return 1
    }

    # Generate human-readable report
    snyk container test "$image:latest" \
        --severity-threshold=high \
        > "$REPORT_DIR/snyk-$image.txt"
}

# Function to generate summary report
generate_summary() {
    echo -e "${GREEN}Generating security summary report...${NC}"

    cat > "$REPORT_DIR/security-summary.md" << EOF
# Container Security Scan Summary

**Scan Date:** $(date)
**Images Scanned:** ${#IMAGES[@]}

## Scan Results

| Image | Trivy | Grype | Snyk | Status |
|-------|-------|-------|------|--------|
EOF

    for image in "${IMAGES[@]}"; do
        trivy_status="✅"
        grype_status="✅"
        snyk_status="✅"
        overall_status="✅ PASS"

        # Check if scan files exist and have vulnerabilities
        if [[ -f "$REPORT_DIR/trivy-$image.json" ]]; then
            vuln_count=$(jq '.Results[0].Vulnerabilities | length' "$REPORT_DIR/trivy-$image.json" 2>/dev/null || echo "0")
            if [[ $vuln_count -gt 0 ]]; then
                trivy_status="❌ ($vuln_count)"
                overall_status="❌ FAIL"
            fi
        else
            trivy_status="⚠️ NO SCAN"
        fi

        if [[ -f "$REPORT_DIR/grype-$image.json" ]]; then
            vuln_count=$(jq '.matches | length' "$REPORT_DIR/grype-$image.json" 2>/dev/null || echo "0")
            if [[ $vuln_count -gt 0 ]]; then
                grype_status="❌ ($vuln_count)"
                overall_status="❌ FAIL"
            fi
        else
            grype_status="⚠️ NO SCAN"
        fi

        if [[ -f "$REPORT_DIR/snyk-$image.json" ]]; then
            vuln_count=$(jq '.vulnerabilities | length' "$REPORT_DIR/snyk-$image.json" 2>/dev/null || echo "0")
            if [[ $vuln_count -gt 0 ]]; then
                snyk_status="❌ ($vuln_count)"
                overall_status="❌ FAIL"
            fi
        else
            snyk_status="⚠️ NO SCAN"
        fi

        echo "| $image | $trivy_status | $grype_status | $snyk_status | $overall_status |" >> "$REPORT_DIR/security-summary.md"
    done

    cat >> "$REPORT_DIR/security-summary.md" << EOF

## Security Recommendations

1. **Update base images** to latest secure versions
2. **Remove unnecessary packages** to minimize attack surface
3. **Use distroless images** where possible for production
4. **Implement runtime security** monitoring
5. **Regular vulnerability scanning** in CI/CD pipeline

## Files Generated

- \`trivy-*.json\` - Trivy vulnerability reports
- \`grype-*.json\` - Grype vulnerability reports
- \`snyk-*.json\` - Snyk vulnerability reports
- \`*-*.txt\` - Human-readable reports

EOF
}

# Main execution
main() {
    echo -e "${GREEN}O-RAN MANO Container Security Scanner${NC}"
    echo "======================================"

    # Install scanners if needed
    install_scanners

    # Build images first
    echo -e "${YELLOW}Building all images...${NC}"
    docker-compose build

    # Run scans on each image
    scan_failures=0
    for image in "${IMAGES[@]}"; do
        echo -e "\n${GREEN}Scanning $image...${NC}"

        # Run Trivy scan
        if ! run_trivy_scan "$image"; then
            ((scan_failures++))
        fi

        # Run Grype scan
        if ! run_grype_scan "$image"; then
            ((scan_failures++))
        fi

        # Run Snyk scan (if authenticated)
        if snyk auth status >/dev/null 2>&1; then
            if ! run_snyk_scan "$image"; then
                ((scan_failures++))
            fi
        else
            echo -e "${YELLOW}Snyk not authenticated, skipping Snyk scan for $image${NC}"
        fi
    done

    # Generate summary
    generate_summary

    echo -e "\n${GREEN}Security scanning completed!${NC}"
    echo "Reports available in: $REPORT_DIR"
    echo "Summary: $REPORT_DIR/security-summary.md"

    if [[ $scan_failures -gt 0 ]]; then
        echo -e "${RED}$scan_failures scan(s) failed. Check reports for details.${NC}"
        exit 1
    else
        echo -e "${GREEN}All scans passed successfully!${NC}"
    fi
}

# Run main function
main "$@"