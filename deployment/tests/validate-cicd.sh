#!/bin/bash

# Comprehensive CI/CD Pipeline Validation for O-RAN MANO
# This script validates all CI/CD components and their integration

set -euo pipefail

# Configuration
PROJECT_ROOT="${PROJECT_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)}"
TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-$PROJECT_ROOT/deployment/test-results}"
VALIDATION_MODE="${VALIDATION_MODE:-full}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

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

skip() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] ⏭️ $1${NC}"
    ((TESTS_SKIPPED++))
}

test_step() {
    ((TESTS_TOTAL++))
    log "Test $TESTS_TOTAL: $1"
}

# Initialize validation
initialize_validation() {
    log "Initializing CI/CD validation..."

    mkdir -p "$TEST_RESULTS_DIR"

    # Create test report
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    export VALIDATION_REPORT="$TEST_RESULTS_DIR/cicd_validation_$timestamp.json"

    # Initialize report structure
    cat > "$VALIDATION_REPORT" << 'EOF'
{
  "timestamp": "",
  "validation_mode": "",
  "summary": {
    "total_tests": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0,
    "success_rate": 0
  },
  "components": {},
  "issues": [],
  "recommendations": []
}
EOF

    # Update timestamp and mode
    local ts
    ts=$(date -Iseconds)
    jq --arg ts "$ts" --arg mode "$VALIDATION_MODE" '.timestamp = $ts | .validation_mode = $mode' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"

    success "Validation initialized. Report: $VALIDATION_REPORT"
}

# Validate GitHub workflows
validate_github_workflows() {
    test_step "Validating GitHub workflows"

    local workflows_dir="$PROJECT_ROOT/.github/workflows"
    local workflow_results="{\"status\": \"unknown\", \"workflows\": {}, \"issues\": []}"

    if [ ! -d "$workflows_dir" ]; then
        error "GitHub workflows directory not found: $workflows_dir"
        jq --argjson result "$workflow_results" '.components.github_workflows = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    local workflow_files
    workflow_files=$(find "$workflows_dir" -name "*.yml" -o -name "*.yaml" 2>/dev/null || echo "")

    if [ -z "$workflow_files" ]; then
        error "No workflow files found in $workflows_dir"
        workflow_results=$(echo "$workflow_results" | jq '.status = "error" | .issues += ["No workflow files found"]')
        jq --argjson result "$workflow_results" '.components.github_workflows = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    local valid_workflows=0
    local total_workflows=0

    for workflow_file in $workflow_files; do
        ((total_workflows++))
        local workflow_name
        workflow_name=$(basename "$workflow_file" .yml)

        log "Validating workflow: $workflow_name"

        # Check YAML syntax
        if ! yq eval '.' "$workflow_file" > /dev/null 2>&1; then
            error "Invalid YAML syntax in $workflow_file"
            workflow_results=$(echo "$workflow_results" | jq --arg name "$workflow_name" '.workflows[$name] = {status: "invalid_yaml", issues: ["Invalid YAML syntax"]}')
            continue
        fi

        # Check required fields
        local issues=()

        # Check for 'on' trigger
        if ! yq eval '.on' "$workflow_file" > /dev/null 2>&1; then
            issues+=("Missing 'on' trigger")
        fi

        # Check for jobs
        if ! yq eval '.jobs' "$workflow_file" > /dev/null 2>&1; then
            issues+=("Missing 'jobs' section")
        fi

        # Check for checkout action in jobs
        local has_checkout
        has_checkout=$(yq eval '.jobs[].steps[] | select(.uses | test("actions/checkout"))' "$workflow_file" 2>/dev/null | wc -l)

        if [ "$has_checkout" -eq 0 ]; then
            issues+=("No checkout action found")
        fi

        # Check for environment variables and secrets
        local uses_secrets
        uses_secrets=$(yq eval '.jobs[].steps[] | select(.env) | .env | keys[]' "$workflow_file" 2>/dev/null | grep -c "secrets\." || echo "0")

        # Validate specific workflows
        case "$workflow_name" in
            "deploy-monitoring")
                # Check for monitoring-specific validations
                if ! grep -q "prometheus" "$workflow_file"; then
                    issues+=("Missing Prometheus validation")
                fi
                if ! grep -q "grafana" "$workflow_file"; then
                    issues+=("Missing Grafana validation")
                fi
                ;;
            "validate-metrics")
                # Check for metrics validation
                if ! grep -q "metrics" "$workflow_file"; then
                    issues+=("Missing metrics validation")
                fi
                ;;
        esac

        if [ ${#issues[@]} -eq 0 ]; then
            ((valid_workflows++))
            success "Workflow $workflow_name is valid"
            workflow_results=$(echo "$workflow_results" | jq --arg name "$workflow_name" '.workflows[$name] = {status: "valid", issues: []}')
        else
            error "Workflow $workflow_name has issues: ${issues[*]}"
            local issues_json
            issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
            workflow_results=$(echo "$workflow_results" | jq --arg name "$workflow_name" --argjson issues "$issues_json" '.workflows[$name] = {status: "invalid", issues: $issues}')
        fi
    done

    if [ "$valid_workflows" -eq "$total_workflows" ]; then
        workflow_results=$(echo "$workflow_results" | jq '.status = "valid"')
        success "All GitHub workflows are valid ($valid_workflows/$total_workflows)"
    else
        workflow_results=$(echo "$workflow_results" | jq '.status = "partially_valid"')
        warning "Some GitHub workflows have issues ($valid_workflows/$total_workflows valid)"
    fi

    jq --argjson result "$workflow_results" '.components.github_workflows = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Validate deployment scripts
validate_deployment_scripts() {
    test_step "Validating deployment scripts"

    local scripts_dir="$PROJECT_ROOT/deployment/scripts"
    local script_results="{\"status\": \"unknown\", \"scripts\": {}, \"issues\": []}"

    if [ ! -d "$scripts_dir" ]; then
        error "Deployment scripts directory not found: $scripts_dir"
        jq --argjson result "$script_results" '.components.deployment_scripts = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    local script_files
    script_files=$(find "$scripts_dir" -name "*.sh" -type f 2>/dev/null || echo "")

    if [ -z "$script_files" ]; then
        error "No deployment scripts found in $scripts_dir"
        script_results=$(echo "$script_results" | jq '.status = "error" | .issues += ["No script files found"]')
        jq --argjson result "$script_results" '.components.deployment_scripts = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    local valid_scripts=0
    local total_scripts=0

    for script_file in $script_files; do
        ((total_scripts++))
        local script_name
        script_name=$(basename "$script_file")

        log "Validating script: $script_name"

        local issues=()

        # Check if script is executable
        if [ ! -x "$script_file" ]; then
            issues+=("Script is not executable")
        fi

        # Check bash syntax
        if ! bash -n "$script_file" 2>/dev/null; then
            issues+=("Invalid bash syntax")
        fi

        # Check for shebang
        if ! head -n 1 "$script_file" | grep -q "^#!/bin/bash"; then
            issues+=("Missing or incorrect shebang")
        fi

        # Check for set -euo pipefail
        if ! grep -q "set -euo pipefail" "$script_file"; then
            issues+=("Missing 'set -euo pipefail'")
        fi

        # Check for help/usage information
        if ! grep -q -E "(help|usage)" "$script_file"; then
            issues+=("Missing help/usage information")
        fi

        # Check for error handling
        if ! grep -q -E "(trap|error)" "$script_file"; then
            issues+=("Limited error handling")
        fi

        # Validate specific scripts
        case "$script_name" in
            "ci-deploy.sh")
                if ! grep -q "kind" "$script_file"; then
                    issues+=("Missing kind cluster setup")
                fi
                if ! grep -q "prometheus" "$script_file"; then
                    issues+=("Missing Prometheus deployment")
                fi
                ;;
            "ci-validation.sh")
                if ! grep -q "kubectl" "$script_file"; then
                    issues+=("Missing kubectl commands")
                fi
                ;;
            "*rollback*.sh")
                if ! grep -q "backup" "$script_file"; then
                    issues+=("Missing backup functionality")
                fi
                ;;
        esac

        if [ ${#issues[@]} -eq 0 ]; then
            ((valid_scripts++))
            success "Script $script_name is valid"
            script_results=$(echo "$script_results" | jq --arg name "$script_name" '.scripts[$name] = {status: "valid", issues: []}')
        else
            warning "Script $script_name has issues: ${issues[*]}"
            local issues_json
            issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
            script_results=$(echo "$script_results" | jq --arg name "$script_name" --argjson issues "$issues_json" '.scripts[$name] = {status: "issues", issues: $issues}')
        fi
    done

    if [ "$valid_scripts" -eq "$total_scripts" ]; then
        script_results=$(echo "$script_results" | jq '.status = "valid"')
        success "All deployment scripts are valid ($valid_scripts/$total_scripts)"
    else
        script_results=$(echo "$script_results" | jq '.status = "partially_valid"')
        warning "Some deployment scripts have issues ($valid_scripts/$total_scripts valid)"
    fi

    jq --argjson result "$script_results" '.components.deployment_scripts = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Validate Terraform configuration
validate_terraform_config() {
    test_step "Validating Terraform configuration"

    local terraform_dir="$PROJECT_ROOT/deployment/terraform"
    local terraform_results="{\"status\": \"unknown\", \"files\": {}, \"issues\": []}"

    if [ ! -d "$terraform_dir" ]; then
        skip "Terraform directory not found: $terraform_dir"
        jq --argjson result "$terraform_results" '.components.terraform = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    if ! command -v terraform &> /dev/null; then
        skip "Terraform not installed, skipping validation"
        terraform_results=$(echo "$terraform_results" | jq '.status = "skipped" | .issues += ["Terraform not installed"]')
        jq --argjson result "$terraform_results" '.components.terraform = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    cd "$terraform_dir"

    local issues=()

    # Check for required files
    local required_files=("main.tf" "variables.tf" "outputs.tf")
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            issues+=("Missing required file: $file")
        fi
    done

    # Validate Terraform syntax
    log "Running terraform validate..."
    if terraform init -backend=false &>/dev/null && terraform validate &>/dev/null; then
        success "Terraform configuration is valid"
        terraform_results=$(echo "$terraform_results" | jq '.status = "valid"')
    else
        error "Terraform validation failed"
        issues+=("Terraform validation failed")
        terraform_results=$(echo "$terraform_results" | jq '.status = "invalid"')
    fi

    # Check for best practices
    if ! grep -q "required_version" main.tf; then
        issues+=("Missing Terraform version constraint")
    fi

    if ! grep -q "required_providers" main.tf; then
        issues+=("Missing provider version constraints")
    fi

    # Check for security issues
    if grep -q "password.*=" variables.tf && ! grep -q "sensitive.*=.*true" variables.tf; then
        issues+=("Passwords should be marked as sensitive")
    fi

    if [ ${#issues[@]} -gt 0 ]; then
        local issues_json
        issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
        terraform_results=$(echo "$terraform_results" | jq --argjson issues "$issues_json" '.issues = $issues')
    fi

    cd - > /dev/null

    jq --argjson result "$terraform_results" '.components.terraform = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Validate Kustomize configuration
validate_kustomize_config() {
    test_step "Validating Kustomize configuration"

    local kustomize_dir="$PROJECT_ROOT/monitoring/kustomize"
    local kustomize_results="{\"status\": \"unknown\", \"overlays\": {}, \"issues\": []}"

    if [ ! -d "$kustomize_dir" ]; then
        skip "Kustomize directory not found: $kustomize_dir"
        jq --argjson result "$kustomize_results" '.components.kustomize = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    if ! command -v kustomize &> /dev/null; then
        skip "Kustomize not installed, skipping validation"
        kustomize_results=$(echo "$kustomize_results" | jq '.status = "skipped" | .issues += ["Kustomize not installed"]')
        jq --argjson result "$kustomize_results" '.components.kustomize = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    local valid_overlays=0
    local total_overlays=0

    # Check base configuration
    if [ -f "$kustomize_dir/base/kustomization.yaml" ]; then
        log "Validating base kustomization..."
        if kustomize build "$kustomize_dir/base" > /dev/null 2>&1; then
            success "Base kustomization is valid"
            kustomize_results=$(echo "$kustomize_results" | jq '.overlays.base = {status: "valid", issues: []}')
        else
            error "Base kustomization is invalid"
            kustomize_results=$(echo "$kustomize_results" | jq '.overlays.base = {status: "invalid", issues: ["Build failed"]}')
        fi
    fi

    # Check overlay configurations
    local overlay_dirs
    overlay_dirs=$(find "$kustomize_dir/overlays" -maxdepth 1 -type d 2>/dev/null | grep -v "/overlays$" || echo "")

    for overlay_dir in $overlay_dirs; do
        ((total_overlays++))
        local overlay_name
        overlay_name=$(basename "$overlay_dir")

        log "Validating overlay: $overlay_name"

        local issues=()

        if [ ! -f "$overlay_dir/kustomization.yaml" ]; then
            issues+=("Missing kustomization.yaml")
        else
            if ! kustomize build "$overlay_dir" > /dev/null 2>&1; then
                issues+=("Kustomize build failed")
            fi
        fi

        if [ ${#issues[@]} -eq 0 ]; then
            ((valid_overlays++))
            success "Overlay $overlay_name is valid"
            kustomize_results=$(echo "$kustomize_results" | jq --arg name "$overlay_name" '.overlays[$name] = {status: "valid", issues: []}')
        else
            error "Overlay $overlay_name has issues: ${issues[*]}"
            local issues_json
            issues_json=$(printf '%s\n' "${issues[@]}" | jq -R . | jq -s .)
            kustomize_results=$(echo "$kustomize_results" | jq --arg name "$overlay_name" --argjson issues "$issues_json" '.overlays[$name] = {status: "invalid", issues: $issues}')
        fi
    done

    if [ "$valid_overlays" -eq "$total_overlays" ]; then
        kustomize_results=$(echo "$kustomize_results" | jq '.status = "valid"')
        success "All Kustomize overlays are valid ($valid_overlays/$total_overlays)"
    else
        kustomize_results=$(echo "$kustomize_results" | jq '.status = "partially_valid"')
        warning "Some Kustomize overlays have issues ($valid_overlays/$total_overlays valid)"
    fi

    jq --argjson result "$kustomize_results" '.components.kustomize = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Validate monitoring configuration
validate_monitoring_config() {
    test_step "Validating monitoring configuration"

    local monitoring_dir="$PROJECT_ROOT/monitoring"
    local monitoring_results="{\"status\": \"unknown\", \"components\": {}, \"issues\": []}"

    if [ ! -d "$monitoring_dir" ]; then
        error "Monitoring directory not found: $monitoring_dir"
        jq --argjson result "$monitoring_results" '.components.monitoring_config = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
        return
    fi

    # Check Prometheus configuration
    local prometheus_dir="$monitoring_dir/prometheus"
    if [ -d "$prometheus_dir" ]; then
        log "Validating Prometheus configuration..."
        local prom_issues=()

        # Check for YAML files
        local prom_files
        prom_files=$(find "$prometheus_dir" -name "*.yaml" -o -name "*.yml" 2>/dev/null || echo "")

        for file in $prom_files; do
            if ! yq eval '.' "$file" > /dev/null 2>&1; then
                prom_issues+=("Invalid YAML in $(basename "$file")")
            fi
        done

        if [ ${#prom_issues[@]} -eq 0 ]; then
            monitoring_results=$(echo "$monitoring_results" | jq '.components.prometheus = {status: "valid", issues: []}')
        else
            local issues_json
            issues_json=$(printf '%s\n' "${prom_issues[@]}" | jq -R . | jq -s .)
            monitoring_results=$(echo "$monitoring_results" | jq --argjson issues "$issues_json" '.components.prometheus = {status: "invalid", issues: $issues}')
        fi
    fi

    # Check Grafana configuration
    local grafana_dir="$monitoring_dir/grafana"
    if [ -d "$grafana_dir" ]; then
        log "Validating Grafana configuration..."
        local grafana_issues=()

        # Check for dashboard JSON files
        local dashboard_files
        dashboard_files=$(find "$grafana_dir" -name "*.json" 2>/dev/null || echo "")

        for file in $dashboard_files; do
            if ! jq empty "$file" 2>/dev/null; then
                grafana_issues+=("Invalid JSON in $(basename "$file")")
            fi
        done

        if [ ${#grafana_issues[@]} -eq 0 ]; then
            monitoring_results=$(echo "$monitoring_results" | jq '.components.grafana = {status: "valid", issues: []}')
        else
            local issues_json
            issues_json=$(printf '%s\n' "${grafana_issues[@]}" | jq -R . | jq -s .)
            monitoring_results=$(echo "$monitoring_results" | jq --argjson issues "$issues_json" '.components.grafana = {status: "invalid", issues: $issues}')
        fi
    fi

    # Check AlertManager configuration
    local alerting_dir="$monitoring_dir/alerting"
    if [ -d "$alerting_dir" ]; then
        log "Validating AlertManager configuration..."
        local alert_issues=()

        # Check for YAML files
        local alert_files
        alert_files=$(find "$alerting_dir" -name "*.yaml" -o -name "*.yml" 2>/dev/null || echo "")

        for file in $alert_files; do
            if ! yq eval '.' "$file" > /dev/null 2>&1; then
                alert_issues+=("Invalid YAML in $(basename "$file")")
            fi
        done

        if [ ${#alert_issues[@]} -eq 0 ]; then
            monitoring_results=$(echo "$monitoring_results" | jq '.components.alertmanager = {status: "valid", issues: []}')
        else
            local issues_json
            issues_json=$(printf '%s\n' "${alert_issues[@]}" | jq -R . | jq -s .)
            monitoring_results=$(echo "$monitoring_results" | jq --argjson issues "$issues_json" '.components.alertmanager = {status: "invalid", issues: $issues}')
        fi
    fi

    # Overall status
    local has_errors
    has_errors=$(echo "$monitoring_results" | jq '[.components[] | select(.status == "invalid")] | length')

    if [ "$has_errors" -eq 0 ]; then
        monitoring_results=$(echo "$monitoring_results" | jq '.status = "valid"')
        success "Monitoring configuration is valid"
    else
        monitoring_results=$(echo "$monitoring_results" | jq '.status = "partially_valid"')
        warning "Some monitoring components have issues"
    fi

    jq --argjson result "$monitoring_results" '.components.monitoring_config = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Validate dependencies
validate_dependencies() {
    test_step "Validating CI/CD dependencies"

    local deps_results="{\"status\": \"unknown\", \"tools\": {}, \"issues\": []}"

    # Required tools
    local required_tools=(
        "kubectl:kubectl"
        "helm:helm"
        "docker:docker"
        "kind:kind"
        "yq:yq"
        "jq:jq"
        "curl:curl"
    )

    local missing_tools=()
    local available_tools=()

    for tool_spec in "${required_tools[@]}"; do
        local tool_name="${tool_spec%:*}"
        local tool_command="${tool_spec#*:}"

        if command -v "$tool_command" &> /dev/null; then
            local version
            case "$tool_command" in
                "kubectl")
                    version=$(kubectl version --client --short 2>/dev/null | cut -d' ' -f3 || echo "unknown")
                    ;;
                "helm")
                    version=$(helm version --short 2>/dev/null | cut -d' ' -f1 || echo "unknown")
                    ;;
                "docker")
                    version=$(docker --version 2>/dev/null | cut -d' ' -f3 | tr -d ',' || echo "unknown")
                    ;;
                *)
                    version=$($tool_command --version 2>/dev/null | head -1 || echo "unknown")
                    ;;
            esac

            available_tools+=("$tool_name:$version")
            deps_results=$(echo "$deps_results" | jq --arg tool "$tool_name" --arg ver "$version" '.tools[$tool] = {status: "available", version: $ver}')
        else
            missing_tools+=("$tool_name")
            deps_results=$(echo "$deps_results" | jq --arg tool "$tool_name" '.tools[$tool] = {status: "missing", version: null}')
        fi
    done

    if [ ${#missing_tools[@]} -eq 0 ]; then
        deps_results=$(echo "$deps_results" | jq '.status = "complete"')
        success "All required dependencies are available"
    else
        deps_results=$(echo "$deps_results" | jq '.status = "incomplete"')
        error "Missing dependencies: ${missing_tools[*]}"
        local missing_json
        missing_json=$(printf '%s\n' "${missing_tools[@]}" | jq -R . | jq -s .)
        deps_results=$(echo "$deps_results" | jq --argjson missing "$missing_json" '.issues = ["Missing tools: " + ($missing | join(", "))]')
    fi

    jq --argjson result "$deps_results" '.components.dependencies = $result' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Generate recommendations
generate_recommendations() {
    log "Generating recommendations..."

    local recommendations=()

    # Analyze the report for issues
    local github_status
    github_status=$(jq -r '.components.github_workflows.status // "unknown"' "$VALIDATION_REPORT")

    local scripts_status
    scripts_status=$(jq -r '.components.deployment_scripts.status // "unknown"' "$VALIDATION_REPORT")

    local deps_status
    deps_status=$(jq -r '.components.dependencies.status // "unknown"' "$VALIDATION_REPORT")

    # Generate specific recommendations
    if [ "$github_status" != "valid" ]; then
        recommendations+=("Review and fix GitHub workflow files for syntax and completeness")
    fi

    if [ "$scripts_status" != "valid" ]; then
        recommendations+=("Improve deployment scripts by adding error handling and documentation")
    fi

    if [ "$deps_status" != "complete" ]; then
        recommendations+=("Install missing dependencies before running CI/CD pipelines")
    fi

    # Add general recommendations
    recommendations+=("Regularly test CI/CD pipelines in development environment")
    recommendations+=("Implement automated testing for all pipeline components")
    recommendations+=("Monitor pipeline performance and optimize as needed")

    # Update report
    local recommendations_json
    recommendations_json=$(printf '%s\n' "${recommendations[@]}" | jq -R . | jq -s .)
    jq --argjson recs "$recommendations_json" '.recommendations = $recs' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Update final summary
update_summary() {
    log "Updating validation summary..."

    local success_rate=0
    if [ "$TESTS_TOTAL" -gt 0 ]; then
        success_rate=$(echo "scale=2; $TESTS_PASSED * 100 / $TESTS_TOTAL" | bc)
    fi

    jq --argjson total "$TESTS_TOTAL" \
       --argjson passed "$TESTS_PASSED" \
       --argjson failed "$TESTS_FAILED" \
       --argjson skipped "$TESTS_SKIPPED" \
       --argjson rate "$success_rate" \
       '.summary = {
         total_tests: $total,
         passed: $passed,
         failed: $failed,
         skipped: $skipped,
         success_rate: $rate
       }' "$VALIDATION_REPORT" > /tmp/report.json && mv /tmp/report.json "$VALIDATION_REPORT"
}

# Generate markdown report
generate_markdown_report() {
    log "Generating markdown report..."

    local markdown_report="${VALIDATION_REPORT%.json}.md"
    local json_data
    json_data=$(cat "$VALIDATION_REPORT")

    cat > "$markdown_report" << EOF
# O-RAN MANO CI/CD Pipeline Validation Report

**Generated**: $(date)
**Validation Mode**: $(echo "$json_data" | jq -r '.validation_mode')

## Summary

- **Total Tests**: $(echo "$json_data" | jq -r '.summary.total_tests')
- **Passed**: $(echo "$json_data" | jq -r '.summary.passed')
- **Failed**: $(echo "$json_data" | jq -r '.summary.failed')
- **Skipped**: $(echo "$json_data" | jq -r '.summary.skipped')
- **Success Rate**: $(echo "$json_data" | jq -r '.summary.success_rate')%

## Component Status

### GitHub Workflows
**Status**: $(echo "$json_data" | jq -r '.components.github_workflows.status // "not_tested"')

$(
if [ "$(echo "$json_data" | jq -r '.components.github_workflows.status')" != "null" ]; then
    echo "$json_data" | jq -r '.components.github_workflows.workflows // {} | to_entries[] | "- **\(.key)**: \(.value.status)"'
fi
)

### Deployment Scripts
**Status**: $(echo "$json_data" | jq -r '.components.deployment_scripts.status // "not_tested"')

$(
if [ "$(echo "$json_data" | jq -r '.components.deployment_scripts.status')" != "null" ]; then
    echo "$json_data" | jq -r '.components.deployment_scripts.scripts // {} | to_entries[] | "- **\(.key)**: \(.value.status)"'
fi
)

### Dependencies
**Status**: $(echo "$json_data" | jq -r '.components.dependencies.status // "not_tested"')

$(
if [ "$(echo "$json_data" | jq -r '.components.dependencies.status')" != "null" ]; then
    echo "$json_data" | jq -r '.components.dependencies.tools // {} | to_entries[] | "- **\(.key)**: \(.value.status) (\(.value.version // "unknown"))"'
fi
)

## Recommendations

$(
echo "$json_data" | jq -r '.recommendations[]? | "- " + .'
)

## Files Generated

- **JSON Report**: $(basename "$VALIDATION_REPORT")
- **Markdown Report**: $(basename "$markdown_report")

---
*This report was generated automatically by the O-RAN MANO CI/CD validation system.*
EOF

    success "Markdown report generated: $markdown_report"
}

# Main validation function
main() {
    local mode="${1:-$VALIDATION_MODE}"
    VALIDATION_MODE="$mode"

    log "Starting O-RAN MANO CI/CD pipeline validation"
    log "Mode: $VALIDATION_MODE"

    initialize_validation

    case "$VALIDATION_MODE" in
        "full")
            validate_github_workflows
            validate_deployment_scripts
            validate_terraform_config
            validate_kustomize_config
            validate_monitoring_config
            validate_dependencies
            ;;
        "workflows")
            validate_github_workflows
            ;;
        "scripts")
            validate_deployment_scripts
            ;;
        "terraform")
            validate_terraform_config
            ;;
        "kustomize")
            validate_kustomize_config
            ;;
        "monitoring")
            validate_monitoring_config
            ;;
        "dependencies")
            validate_dependencies
            ;;
        *)
            error "Unknown validation mode: $VALIDATION_MODE"
            echo "Supported modes: full, workflows, scripts, terraform, kustomize, monitoring, dependencies"
            exit 1
            ;;
    esac

    generate_recommendations
    update_summary
    generate_markdown_report

    # Final summary
    echo ""
    log "Validation Summary:"
    log "  Total Tests: $TESTS_TOTAL"
    log "  Passed: $TESTS_PASSED"
    log "  Failed: $TESTS_FAILED"
    log "  Skipped: $TESTS_SKIPPED"

    if [ "$TESTS_FAILED" -eq 0 ]; then
        success "🎉 All CI/CD components are valid!"
        exit 0
    else
        error "❌ CI/CD validation failed with $TESTS_FAILED issues"
        log "See report for details: $VALIDATION_REPORT"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-full}" in
    "full"|"workflows"|"scripts"|"terraform"|"kustomize"|"monitoring"|"dependencies")
        main "$1"
        ;;
    "help")
        echo "Usage: $0 [validation-mode]"
        echo ""
        echo "Validation Modes:"
        echo "  full         - Complete validation of all components (default)"
        echo "  workflows    - Validate GitHub workflows only"
        echo "  scripts      - Validate deployment scripts only"
        echo "  terraform    - Validate Terraform configuration only"
        echo "  kustomize    - Validate Kustomize configuration only"
        echo "  monitoring   - Validate monitoring configuration only"
        echo "  dependencies - Validate required dependencies only"
        echo "  help         - Show this help message"
        echo ""
        echo "Environment Variables:"
        echo "  PROJECT_ROOT        - Project root directory"
        echo "  TEST_RESULTS_DIR    - Test results directory"
        ;;
    *)
        error "Unknown validation mode: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac