#!/bin/bash
# O-RAN Intent MANO Bootstrap Script
# Headless-friendly environment initialization

set -euo pipefail

# Colors for output (works in headless mode)
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Environment validation
validate_environment() {
    log_info "Validating environment..."

    # Check for .env file
    if [ ! -f "$PROJECT_ROOT/.env" ]; then
        if [ -f "$PROJECT_ROOT/.env.sample" ]; then
            log_warn ".env file not found, creating from .env.sample"
            cp "$PROJECT_ROOT/.env.sample" "$PROJECT_ROOT/.env"
            log_warn "Please update .env with your configuration"
        else
            log_warn "No .env or .env.sample found, creating template..."
            create_env_sample
        fi
    fi

    # Source environment variables
    if [ -f "$PROJECT_ROOT/.env" ]; then
        set -a
        source "$PROJECT_ROOT/.env"
        set +a
    fi

    # Validate no hardcoded secrets
    check_secrets
}

# Create .env.sample if it doesn't exist
create_env_sample() {
    cat > "$PROJECT_ROOT/.env.sample" << 'EOF'
# O-RAN Intent MANO Environment Configuration
# Copy this file to .env and update with your values

# Cluster Configuration
CLUSTER_NAME=oran-mano-local
CLUSTER_TYPE=kind  # kind or k3s
KUBECONFIG_PATH=${HOME}/.kube/config

# Network Configuration (no hardcoded IPs!)
# These will be discovered dynamically
SERVICE_SUBNET=""
POD_SUBNET=""
OVERLAY_NETWORK=""

# O-RAN Configuration
O2IMS_ENDPOINT=""  # Will be set dynamically
O2DMS_ENDPOINT=""  # Will be set dynamically
SMO_ENDPOINT=""    # Will be set dynamically

# GitOps Configuration
GIT_REPO_URL=""  # Your GitOps repository
GIT_BRANCH=main
GIT_USERNAME=""
GIT_TOKEN=""  # Use environment variable or secret manager

# Monitoring Configuration
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
METRICS_RETENTION_DAYS=7

# Development Settings
DEBUG_MODE=false
LOG_LEVEL=info
ENABLE_TRACING=false

# Resource Limits
MAX_SLICES=100
MAX_EDGE_SITES=50
MAX_REGIONAL_SITES=5

# Performance Targets
TARGET_DEPLOYMENT_TIME_MINUTES=10
TARGET_THROUGHPUT_HIGH_MBPS=4.57
TARGET_THROUGHPUT_MED_MBPS=2.77
TARGET_THROUGHPUT_LOW_MBPS=0.93
TARGET_RTT_HIGH_MS=16.1
TARGET_RTT_MED_MS=15.7
TARGET_RTT_LOW_MS=6.3

# Tool Versions (for reproducibility)
KIND_VERSION=v0.20.0
K3S_VERSION=v1.28.5+k3s1
KUBECTL_VERSION=v1.28.5
KPT_VERSION=v1.0.0-beta.49
NEPHIO_VERSION=v2.0.0
EOF

    log_info "Created .env.sample template"
}

# Check for hardcoded secrets or network configurations
check_secrets() {
    log_info "Checking for hardcoded secrets..."

    # Define patterns to check
    local patterns=(
        "password.*=.*['\"].*['\"]"
        "token.*=.*['\"].*['\"]"
        "secret.*=.*['\"].*['\"]"
        "key.*=.*['\"].*['\"]"
        "[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}"  # IP addresses
    )

    local found_issues=false

    # Check Python files
    if find "$PROJECT_ROOT" -name "*.py" -type f 2>/dev/null | head -1 > /dev/null; then
        for pattern in "${patterns[@]}"; do
            if grep -r -i -E "$pattern" --include="*.py" "$PROJECT_ROOT" 2>/dev/null | grep -v ".env" | grep -v ".git" | head -1 > /dev/null; then
                log_warn "Potential hardcoded values found in Python files (pattern: $pattern)"
                found_issues=true
            fi
        done
    fi

    # Check Go files
    if find "$PROJECT_ROOT" -name "*.go" -type f 2>/dev/null | head -1 > /dev/null; then
        for pattern in "${patterns[@]}"; do
            if grep -r -i -E "$pattern" --include="*.go" "$PROJECT_ROOT" 2>/dev/null | grep -v ".env" | grep -v ".git" | head -1 > /dev/null; then
                log_warn "Potential hardcoded values found in Go files (pattern: $pattern)"
                found_issues=true
            fi
        done
    fi

    # Check YAML files
    if find "$PROJECT_ROOT" -name "*.yaml" -o -name "*.yml" -type f 2>/dev/null | head -1 > /dev/null; then
        for pattern in "${patterns[@]}"; do
            if grep -r -i -E "$pattern" --include="*.yaml" --include="*.yml" "$PROJECT_ROOT" 2>/dev/null | grep -v ".env" | grep -v ".git" | head -1 > /dev/null; then
                log_warn "Potential hardcoded values found in YAML files (pattern: $pattern)"
                found_issues=true
            fi
        done
    fi

    if [ "$found_issues" = true ]; then
        log_warn "Please review and remove any hardcoded secrets or network configurations"
    else
        log_info "No obvious hardcoded secrets found"
    fi
}

# Create directory structure
setup_directories() {
    log_info "Setting up directory structure..."

    local dirs=(
        "nlp/tests"
        "orchestrator/cmd"
        "orchestrator/pkg/placement"
        "orchestrator/tests"
        "adapters/vnf-operator/controllers"
        "adapters/vnf-operator/config"
        "adapters/vnf-operator/tests/golden"
        "ran-dms"
        "cn-dms"
        "tn/manager"
        "tn/agent"
        "tn/tests"
        "net/ovn"
        "net/tests"
        "experiments"
        "clusters/edge01"
        "clusters/edge02"
        "clusters/regional"
        "clusters/central"
        "clusters/local"
    )

    for dir in "${dirs[@]}"; do
        mkdir -p "$PROJECT_ROOT/$dir"
    done

    log_info "Directory structure created"
}

# Initialize Git hooks
setup_git_hooks() {
    log_info "Setting up Git hooks..."

    local hooks_dir="$PROJECT_ROOT/.git/hooks"

    # Pre-commit hook to check for secrets
    cat > "$hooks_dir/pre-commit" << 'EOF'
#!/bin/bash
# Pre-commit hook to prevent committing secrets

# Check for potential secrets
if git diff --cached --name-only | xargs grep -E -i "(password|token|secret|key).*=.*['\"].*['\"]" 2>/dev/null; then
    echo "ERROR: Potential secrets detected in staged files!"
    echo "Please remove hardcoded secrets before committing."
    exit 1
fi

# Check for hardcoded IPs
if git diff --cached --name-only | xargs grep -E "[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}" 2>/dev/null | grep -v "127.0.0.1" | grep -v "0.0.0.0"; then
    echo "WARNING: Hardcoded IP addresses detected in staged files!"
    echo "Consider using environment variables or configuration files."
fi

exit 0
EOF

    chmod +x "$hooks_dir/pre-commit"

    log_info "Git hooks configured"
}

# Install Python dependencies
setup_python() {
    log_info "Setting up Python environment..."

    # Create virtual environment if not in container
    if [ -z "${DEVCONTAINER:-}" ] && [ -z "${CODESPACES:-}" ]; then
        if [ ! -d "$PROJECT_ROOT/venv" ]; then
            python3 -m venv "$PROJECT_ROOT/venv"
            source "$PROJECT_ROOT/venv/bin/activate"
        fi
    fi

    # Install Python tools
    pip install --quiet --upgrade pip
    pip install --quiet \
        black \
        ruff \
        mypy \
        pytest \
        pytest-cov \
        pyyaml \
        requests \
        pydantic

    log_info "Python environment ready"
}

# Install Go tools
setup_go() {
    log_info "Setting up Go environment..."

    if command -v go &> /dev/null; then
        # Install Go tools
        go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || true
        go install golang.org/x/tools/cmd/goimports@latest 2>/dev/null || true
        go install github.com/go-delve/delve/cmd/dlv@latest 2>/dev/null || true

        log_info "Go environment ready"
    else
        log_warn "Go not found, skipping Go setup"
    fi
}

# Install Kubernetes tools
setup_kubernetes() {
    log_info "Setting up Kubernetes tools..."

    # These might already be installed in devcontainer
    local tools_needed=()

    if ! command -v kubectl &> /dev/null; then
        tools_needed+=("kubectl")
    fi

    if ! command -v kind &> /dev/null; then
        tools_needed+=("kind")
    fi

    if ! command -v kpt &> /dev/null; then
        tools_needed+=("kpt")
    fi

    if [ ${#tools_needed[@]} -gt 0 ]; then
        log_warn "Missing Kubernetes tools: ${tools_needed[*]}"
        log_info "Run 'make install-tools' to install missing tools"
    else
        log_info "Kubernetes tools ready"
    fi
}

# Create example files
create_examples() {
    log_info "Creating example files..."

    # Create example intent
    cat > "$PROJECT_ROOT/nlp/example_intent.json" << 'EOF'
{
  "intent": "Create a low-latency network slice for AR/VR gaming with guaranteed 10ms latency",
  "service_type": "gaming",
  "requirements": {
    "latency": "low",
    "bandwidth": "high",
    "reliability": "medium"
  }
}
EOF

    # Create Kind cluster config
    cat > "$PROJECT_ROOT/clusters/kind-config.yaml" << 'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: oran-mano-local
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "site=central,type=control"
  - role: worker
    kubeadmConfigPatches:
      - |
        kind: JoinConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "site=edge01,type=worker"
  - role: worker
    kubeadmConfigPatches:
      - |
        kind: JoinConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "site=edge02,type=worker"
networking:
  apiServerPort: 6443
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/16"
EOF

    log_info "Example files created"
}

# Main bootstrap function
main() {
    log_info "Starting O-RAN Intent MANO bootstrap..."

    # Change to project root
    cd "$PROJECT_ROOT"

    # Run bootstrap steps
    validate_environment
    setup_directories

    # Only setup git hooks if .git directory exists
    if [ -d "$PROJECT_ROOT/.git" ]; then
        setup_git_hooks
    fi

    setup_python
    setup_go
    setup_kubernetes
    create_examples

    # Create placeholder files for checks if they don't exist
    touch "$PROJECT_ROOT/README.md" 2>/dev/null || true

    log_info "Bootstrap complete!"
    log_info "Run 'make help' to see available commands"
    log_info "Run 'make check' to verify the environment"
}

# Run main function
main "$@"