#!/bin/bash
# Spell checking script for documentation and comments

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ERRORS=0

log_info() {
    echo -e "${GREEN}[SPELL]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[SPELL]${NC} $1"
}

log_error() {
    echo -e "${RED}[SPELL]${NC} $1"
}

# Create custom dictionary for technical terms
create_custom_dictionary() {
    local dict_file="$PROJECT_ROOT/.spellcheck-dictionary"

    if [ ! -f "$dict_file" ]; then
        cat > "$dict_file" << 'EOF'
# O-RAN and Network Slicing Terms
MANO
ORAN
O-RAN
Nephio
Porch
ConfigSync
Kube
Kubernetes
kubectl
kustomize
GitOps
VXLAN
O2ims
O2dms
SMO
RAN
CN
TN
VNF
NFV
ETSI
3GPP
SDN
CNI
CRD
CRDs
API
APIs
REST
gRPC
QoS
SLA
KPI
RTT
Mbps
ms

# Container and Cloud Terms
Docker
Dockerfile
devcontainer
containerd
OCI
Helm
yaml
yml
JSON
YAML
env
namespace
pod
pods
deployment
deployments
service
services
configmap
configmaps
secret
secrets
ingress
daemonset
statefulset
replicaset
PVC
PV
RBAC
webhook
webhooks
mutating
validating

# Programming Terms
Python
Golang
Go
JavaScript
TypeScript
async
await
pytest
mypy
ruff
flake8
black
isort
golangci
golangci-lint
gofmt
goimports
eslint
prettier
webpack
npm
pip
venv
virtualenv
struct
interface
func
def
init
middleware
handler
controller
router
schema
validator
serializer
deserializer

# Tool Names
git
GitHub
GitLab
Makefile
makefile
shellcheck
shfmt
hadolint
yamllint
markdownlint
mdl
jq
curl
wget
grep
sed
awk
vim
emacs
vscode
IDE

# Project-specific Terms
orchestrator
adapter
adapters
bootstrap
headless
linting
fmt
localhost
endpoint
endpoints
throughput
latency
jitter
multisite
multi-site
overlay
underlay
placement
lifecycle
telemetry
observability
tracing
metrics
monitoring
prometheus
grafana
opentelemetry
jaeger

# Common Abbreviations
URL
URLs
URI
URIs
UUID
UUIDs
ID
IDs
IP
IPs
TCP
UDP
HTTP
HTTPS
TLS
SSL
SSH
DNS
DHCP
CIDR
MAC
CPU
GPU
RAM
MB
GB
TB
ms
us
ns

# File Extensions and Paths
py
go
js
ts
jsx
tsx
sh
md
txt
log
json
yaml
yml
toml
ini
cfg
conf
config
src
pkg
cmd
bin
lib
dist
build
tmp
temp
usr
etc
var
opt
dev
proc
sys

# Versioning
v1
v2
v3
alpha
beta
rc
LTS
EOL
semver

# Company and Project Names
Anthropic
Claude
Linux
Ubuntu
Debian
Alpine
RedHat
RHEL
CentOS
macOS
Windows
WSL
EOF
        log_info "Created custom dictionary at $dict_file"
    fi

    echo "$dict_file"
}

# Check spelling in Markdown files
check_markdown_spelling() {
    log_info "Checking spelling in Markdown files..."

    local md_files=$(find "$PROJECT_ROOT" -name "*.md" -type f 2>/dev/null | grep -v ".git" | grep -v node_modules || true)

    if [ -z "$md_files" ]; then
        log_info "No Markdown files found"
        return 0
    fi

    local dict_file=$(create_custom_dictionary)

    # Try different spell checkers
    if command -v aspell &> /dev/null; then
        log_info "Using aspell for spell checking..."
        for file in $md_files; do
            # Extract words from markdown, skip code blocks
            local misspelled=$(cat "$file" | \
                sed '/^```/,/^```/d' | \
                sed 's/`[^`]*`//g' | \
                aspell list --mode=markdown --personal="$dict_file" 2>/dev/null | \
                sort -u)

            if [ -n "$misspelled" ]; then
                log_warn "Potential spelling issues in $file:"
                echo "$misspelled" | head -5 | sed 's/^/  - /'
                local count=$(echo "$misspelled" | wc -l)
                if [ $count -gt 5 ]; then
                    log_warn "  ... and $((count - 5)) more"
                fi
            fi
        done
    elif command -v hunspell &> /dev/null; then
        log_info "Using hunspell for spell checking..."
        for file in $md_files; do
            # hunspell approach (simplified)
            local misspelled=$(hunspell -l -p "$dict_file" "$file" 2>/dev/null | sort -u)
            if [ -n "$misspelled" ]; then
                log_warn "Potential spelling issues in $file:"
                echo "$misspelled" | head -5 | sed 's/^/  - /'
            fi
        done
    else
        log_warn "No spell checker found (install aspell or hunspell)"
        log_info "To install: apt-get install aspell aspell-en"
    fi
}

# Check spelling in Python docstrings and comments
check_python_spelling() {
    log_info "Checking spelling in Python comments and docstrings..."

    local python_files=$(find "$PROJECT_ROOT" -name "*.py" -type f 2>/dev/null | \
        grep -v venv | grep -v ".git" | grep -v __pycache__ || true)

    if [ -z "$python_files" ]; then
        log_info "No Python files found"
        return 0
    fi

    if ! command -v aspell &> /dev/null && ! command -v hunspell &> /dev/null; then
        log_info "Skipping Python spell check (no spell checker available)"
        return 0
    fi

    local dict_file=$(create_custom_dictionary)

    for file in $python_files; do
        # Extract comments and docstrings
        local comments=$(grep -E '^\s*#' "$file" | sed 's/^[[:space:]]*#[[:space:]]*//' || true)
        local docstrings=$(sed -n '/"""/,/"""/p; /\x27\x27\x27/,/\x27\x27\x27/p' "$file" || true)

        local all_text=$(echo -e "$comments\n$docstrings")

        if [ -n "$all_text" ]; then
            if command -v aspell &> /dev/null; then
                local misspelled=$(echo "$all_text" | \
                    aspell list --personal="$dict_file" 2>/dev/null | \
                    sort -u)

                if [ -n "$misspelled" ]; then
                    log_warn "Potential spelling issues in Python file $file"
                    # Don't show all, just note that there are issues
                fi
            fi
        fi
    done
}

# Check spelling in Go comments
check_go_spelling() {
    log_info "Checking spelling in Go comments..."

    local go_files=$(find "$PROJECT_ROOT" -name "*.go" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$go_files" ]; then
        log_info "No Go files found"
        return 0
    fi

    if ! command -v aspell &> /dev/null && ! command -v hunspell &> /dev/null; then
        log_info "Skipping Go spell check (no spell checker available)"
        return 0
    fi

    local dict_file=$(create_custom_dictionary)

    for file in $go_files; do
        # Extract comments
        local comments=$(grep -E '^\s*//' "$file" | sed 's|^[[:space:]]*//[[:space:]]*||' || true)

        if [ -n "$comments" ]; then
            if command -v aspell &> /dev/null; then
                local misspelled=$(echo "$comments" | \
                    aspell list --personal="$dict_file" 2>/dev/null | \
                    sort -u)

                if [ -n "$misspelled" ]; then
                    log_warn "Potential spelling issues in Go file $file"
                    # Don't show all, just note that there are issues
                fi
            fi
        fi
    done
}

# Check spelling in YAML comments
check_yaml_spelling() {
    log_info "Checking spelling in YAML comments..."

    local yaml_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) \
        -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$yaml_files" ]; then
        log_info "No YAML files found"
        return 0
    fi

    if ! command -v aspell &> /dev/null && ! command -v hunspell &> /dev/null; then
        log_info "Skipping YAML spell check (no spell checker available)"
        return 0
    fi

    local dict_file=$(create_custom_dictionary)

    for file in $yaml_files; do
        # Extract comments
        local comments=$(grep -E '^\s*#' "$file" | sed 's/^[[:space:]]*#[[:space:]]*//' || true)

        if [ -n "$comments" ]; then
            if command -v aspell &> /dev/null; then
                local misspelled=$(echo "$comments" | \
                    aspell list --personal="$dict_file" 2>/dev/null | \
                    sort -u)

                if [ -n "$misspelled" ] && [ $(echo "$misspelled" | wc -l) -gt 3 ]; then
                    log_warn "Potential spelling issues in YAML file $file"
                fi
            fi
        fi
    done
}

# Main function
main() {
    cd "$PROJECT_ROOT"

    log_info "Starting spell checks..."
    log_info "Note: Technical terms are excluded via custom dictionary"

    check_markdown_spelling
    check_python_spelling
    check_go_spelling
    check_yaml_spelling

    if [ $ERRORS -gt 0 ]; then
        log_error "Spell check found $ERRORS critical error(s)"
        exit 1
    else
        log_info "Spell check completed!"
        log_info "Review warnings above for potential spelling issues"
        log_info "Add technical terms to .spellcheck-dictionary if needed"
    fi
}

main "$@"