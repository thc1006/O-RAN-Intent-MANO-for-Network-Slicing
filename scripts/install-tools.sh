#!/bin/bash
# Tool installation script for O-RAN Intent MANO development
# Headless-friendly and cross-platform

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Convert architecture names
case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm"
        ;;
esac

log_info() {
    echo -e "${GREEN}[INSTALL]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[INSTALL]${NC} $1"
}

log_error() {
    echo -e "${RED}[INSTALL]${NC} $1"
}

# Check if running with sufficient permissions
check_permissions() {
    if [ "$OS" = "linux" ] && [ "$EUID" -ne 0 ] && [ -z "${DEVCONTAINER:-}" ]; then
        log_warn "Some installations may require sudo permissions"
        log_info "You may be prompted for your password"
    fi
}

# Install kubectl
install_kubectl() {
    log_info "Checking kubectl..."

    if command -v kubectl &> /dev/null; then
        local current_version=$(kubectl version --client --short 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        log_info "kubectl already installed: $current_version"
        return 0
    fi

    log_info "Installing kubectl..."

    local kubectl_version="${KUBECTL_VERSION:-v1.28.5}"
    local url="https://dl.k8s.io/release/${kubectl_version}/bin/${OS}/${ARCH}/kubectl"

    if [ "$OS" = "windows" ]; then
        url="https://dl.k8s.io/release/${kubectl_version}/bin/windows/amd64/kubectl.exe"
    fi

    curl -LO "$url" 2>/dev/null || wget -q "$url"

    if [ "$OS" = "windows" ]; then
        chmod +x kubectl.exe
        mkdir -p "$HOME/bin"
        mv kubectl.exe "$HOME/bin/"
    else
        chmod +x kubectl
        sudo mv kubectl /usr/local/bin/ 2>/dev/null || mv kubectl "$HOME/.local/bin/"
    fi

    log_info "kubectl installed successfully"
}

# Install Kind
install_kind() {
    log_info "Checking Kind..."

    if command -v kind &> /dev/null; then
        local current_version=$(kind --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' || echo "unknown")
        log_info "Kind already installed: $current_version"
        return 0
    fi

    log_info "Installing Kind..."

    local kind_version="${KIND_VERSION:-v0.20.0}"
    local url="https://github.com/kubernetes-sigs/kind/releases/download/${kind_version}/kind-${OS}-${ARCH}"

    if [ "$OS" = "windows" ]; then
        url="https://github.com/kubernetes-sigs/kind/releases/download/${kind_version}/kind-windows-amd64.exe"
    fi

    curl -Lo kind "$url" 2>/dev/null || wget -q -O kind "$url"

    if [ "$OS" = "windows" ]; then
        chmod +x kind
        mkdir -p "$HOME/bin"
        mv kind "$HOME/bin/kind.exe"
    else
        chmod +x kind
        sudo mv kind /usr/local/bin/ 2>/dev/null || mv kind "$HOME/.local/bin/"
    fi

    log_info "Kind installed successfully"
}

# Install kpt
install_kpt() {
    log_info "Checking kpt..."

    if command -v kpt &> /dev/null; then
        local current_version=$(kpt version 2>/dev/null | head -1 || echo "unknown")
        log_info "kpt already installed: $current_version"
        return 0
    fi

    log_info "Installing kpt..."

    local kpt_version="${KPT_VERSION:-v1.0.0-beta.49}"
    local url="https://github.com/GoogleContainerTools/kpt/releases/download/${kpt_version}/kpt_${OS}_${ARCH}"

    if [ "$OS" = "darwin" ]; then
        url="https://github.com/GoogleContainerTools/kpt/releases/download/${kpt_version}/kpt_darwin_${ARCH}"
    elif [ "$OS" = "windows" ]; then
        url="https://github.com/GoogleContainerTools/kpt/releases/download/${kpt_version}/kpt_windows_${ARCH}.exe"
    fi

    curl -Lo kpt "$url" 2>/dev/null || wget -q -O kpt "$url"

    if [ "$OS" = "windows" ]; then
        chmod +x kpt
        mkdir -p "$HOME/bin"
        mv kpt "$HOME/bin/kpt.exe"
    else
        chmod +x kpt
        sudo mv kpt /usr/local/bin/ 2>/dev/null || mv kpt "$HOME/.local/bin/"
    fi

    log_info "kpt installed successfully"
}

# Install Python tools
install_python_tools() {
    log_info "Installing Python development tools..."

    if ! command -v python3 &> /dev/null && ! command -v python &> /dev/null; then
        log_error "Python not found. Please install Python 3.8+ first"
        return 1
    fi

    local python_cmd="python3"
    if ! command -v python3 &> /dev/null; then
        python_cmd="python"
    fi

    # Upgrade pip
    $python_cmd -m pip install --upgrade pip --quiet 2>/dev/null || true

    # Install development tools
    local python_tools=(
        "black"
        "ruff"
        "mypy"
        "pytest"
        "pytest-cov"
        "pytest-mock"
        "pyyaml"
        "requests"
        "pydantic"
        "click"
        "rich"
        "python-dotenv"
    )

    for tool in "${python_tools[@]}"; do
        log_info "Installing $tool..."
        $python_cmd -m pip install --quiet "$tool" 2>/dev/null || log_warn "Failed to install $tool"
    done

    log_info "Python tools installed"
}

# Install Go tools
install_go_tools() {
    log_info "Installing Go development tools..."

    if ! command -v go &> /dev/null; then
        log_warn "Go not found. Skipping Go tools installation"
        log_info "Install Go from https://golang.org/dl/"
        return 0
    fi

    # Install Go tools
    log_info "Installing golangci-lint..."
    if ! command -v golangci-lint &> /dev/null; then
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
            sh -s -- -b $(go env GOPATH)/bin 2>/dev/null || log_warn "Failed to install golangci-lint"
    fi

    log_info "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest 2>/dev/null || log_warn "Failed to install goimports"

    log_info "Installing delve debugger..."
    go install github.com/go-delve/delve/cmd/dlv@latest 2>/dev/null || log_warn "Failed to install delve"

    log_info "Installing mockgen..."
    go install github.com/golang/mock/mockgen@latest 2>/dev/null || log_warn "Failed to install mockgen"

    log_info "Go tools installed"
}

# Install shell tools
install_shell_tools() {
    log_info "Installing shell development tools..."

    # shellcheck
    if ! command -v shellcheck &> /dev/null; then
        log_info "Installing shellcheck..."
        if [ "$OS" = "linux" ]; then
            if command -v apt-get &> /dev/null; then
                sudo apt-get update && sudo apt-get install -y shellcheck 2>/dev/null || log_warn "Failed to install shellcheck"
            elif command -v yum &> /dev/null; then
                sudo yum install -y ShellCheck 2>/dev/null || log_warn "Failed to install shellcheck"
            else
                # Download binary
                local sc_version="v0.9.0"
                local sc_url="https://github.com/koalaman/shellcheck/releases/download/${sc_version}/shellcheck-${sc_version}.${OS}.x86_64.tar.xz"
                curl -Lo shellcheck.tar.xz "$sc_url" 2>/dev/null
                tar -xf shellcheck.tar.xz
                sudo mv "shellcheck-${sc_version}/shellcheck" /usr/local/bin/ 2>/dev/null || mv "shellcheck-${sc_version}/shellcheck" "$HOME/.local/bin/"
                rm -rf shellcheck.tar.xz "shellcheck-${sc_version}"
            fi
        elif [ "$OS" = "darwin" ]; then
            if command -v brew &> /dev/null; then
                brew install shellcheck 2>/dev/null || log_warn "Failed to install shellcheck"
            fi
        fi
    else
        log_info "shellcheck already installed"
    fi

    # shfmt
    if ! command -v shfmt &> /dev/null; then
        log_info "Installing shfmt..."
        if command -v go &> /dev/null; then
            go install mvdan.cc/sh/v3/cmd/shfmt@latest 2>/dev/null || log_warn "Failed to install shfmt"
        fi
    else
        log_info "shfmt already installed"
    fi
}

# Install YAML/JSON tools
install_yaml_json_tools() {
    log_info "Installing YAML/JSON tools..."

    # yamllint
    if ! command -v yamllint &> /dev/null; then
        log_info "Installing yamllint..."
        if command -v pip3 &> /dev/null; then
            pip3 install --quiet yamllint 2>/dev/null || log_warn "Failed to install yamllint"
        elif command -v pip &> /dev/null; then
            pip install --quiet yamllint 2>/dev/null || log_warn "Failed to install yamllint"
        fi
    else
        log_info "yamllint already installed"
    fi

    # jq
    if ! command -v jq &> /dev/null; then
        log_info "Installing jq..."
        if [ "$OS" = "linux" ]; then
            if command -v apt-get &> /dev/null; then
                sudo apt-get update && sudo apt-get install -y jq 2>/dev/null || log_warn "Failed to install jq"
            elif command -v yum &> /dev/null; then
                sudo yum install -y jq 2>/dev/null || log_warn "Failed to install jq"
            else
                # Download binary
                local jq_url="https://github.com/stedolan/jq/releases/download/jq-1.7/jq-linux64"
                curl -Lo jq "$jq_url" 2>/dev/null
                chmod +x jq
                sudo mv jq /usr/local/bin/ 2>/dev/null || mv jq "$HOME/.local/bin/"
            fi
        elif [ "$OS" = "darwin" ]; then
            if command -v brew &> /dev/null; then
                brew install jq 2>/dev/null || log_warn "Failed to install jq"
            fi
        fi
    else
        log_info "jq already installed"
    fi

    # yq
    if ! command -v yq &> /dev/null; then
        log_info "Installing yq..."
        local yq_version="v4.35.2"
        local yq_url="https://github.com/mikefarah/yq/releases/download/${yq_version}/yq_${OS}_${ARCH}"
        curl -Lo yq "$yq_url" 2>/dev/null || wget -q -O yq "$yq_url"
        chmod +x yq
        sudo mv yq /usr/local/bin/ 2>/dev/null || mv yq "$HOME/.local/bin/"
    else
        log_info "yq already installed"
    fi
}

# Install Docker tools
install_docker_tools() {
    log_info "Checking Docker tools..."

    # hadolint
    if ! command -v hadolint &> /dev/null; then
        log_info "Installing hadolint..."
        local hadolint_version="v2.12.0"
        local hadolint_url="https://github.com/hadolint/hadolint/releases/download/${hadolint_version}/hadolint-${OS}-x86_64"

        if [ "$OS" = "darwin" ]; then
            hadolint_url="https://github.com/hadolint/hadolint/releases/download/${hadolint_version}/hadolint-Darwin-x86_64"
        elif [ "$OS" = "windows" ]; then
            hadolint_url="https://github.com/hadolint/hadolint/releases/download/${hadolint_version}/hadolint-Windows-x86_64.exe"
        fi

        curl -Lo hadolint "$hadolint_url" 2>/dev/null || wget -q -O hadolint "$hadolint_url"
        chmod +x hadolint
        sudo mv hadolint /usr/local/bin/ 2>/dev/null || mv hadolint "$HOME/.local/bin/"
    else
        log_info "hadolint already installed"
    fi
}

# Install spell checkers
install_spell_checkers() {
    log_info "Installing spell checkers..."

    if [ "$OS" = "linux" ]; then
        if command -v apt-get &> /dev/null; then
            if ! command -v aspell &> /dev/null; then
                log_info "Installing aspell..."
                sudo apt-get update && sudo apt-get install -y aspell aspell-en 2>/dev/null || log_warn "Failed to install aspell"
            fi
        elif command -v yum &> /dev/null; then
            if ! command -v aspell &> /dev/null; then
                log_info "Installing aspell..."
                sudo yum install -y aspell aspell-en 2>/dev/null || log_warn "Failed to install aspell"
            fi
        fi
    elif [ "$OS" = "darwin" ]; then
        if command -v brew &> /dev/null; then
            if ! command -v aspell &> /dev/null; then
                log_info "Installing aspell..."
                brew install aspell 2>/dev/null || log_warn "Failed to install aspell"
            fi
        fi
    fi
}

# Create local bin directory
setup_local_bin() {
    log_info "Setting up local bin directory..."

    mkdir -p "$HOME/.local/bin"

    # Add to PATH if not already there
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        log_info "Adding $HOME/.local/bin to PATH"

        # Add to appropriate shell config
        if [ -f "$HOME/.bashrc" ]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
        fi
        if [ -f "$HOME/.zshrc" ]; then
            echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
        fi

        export PATH="$HOME/.local/bin:$PATH"
    fi
}

# Main installation function
main() {
    log_info "Starting tool installation..."
    log_info "OS: $OS, Architecture: $ARCH"

    check_permissions
    setup_local_bin

    # Core Kubernetes tools
    install_kubectl
    install_kind
    install_kpt

    # Development tools
    install_python_tools
    install_go_tools
    install_shell_tools
    install_yaml_json_tools
    install_docker_tools
    install_spell_checkers

    log_info "Tool installation complete!"
    log_info ""
    log_info "Installed tools summary:"
    echo "  - kubectl: $(command -v kubectl &> /dev/null && echo '✓' || echo '✗')"
    echo "  - kind: $(command -v kind &> /dev/null && echo '✓' || echo '✗')"
    echo "  - kpt: $(command -v kpt &> /dev/null && echo '✓' || echo '✗')"
    echo "  - black: $(command -v black &> /dev/null && echo '✓' || echo '✗')"
    echo "  - ruff: $(command -v ruff &> /dev/null && echo '✓' || echo '✗')"
    echo "  - golangci-lint: $(command -v golangci-lint &> /dev/null && echo '✓' || echo '✗')"
    echo "  - shellcheck: $(command -v shellcheck &> /dev/null && echo '✓' || echo '✗')"
    echo "  - yamllint: $(command -v yamllint &> /dev/null && echo '✓' || echo '✗')"
    echo "  - jq: $(command -v jq &> /dev/null && echo '✓' || echo '✗')"
    echo "  - hadolint: $(command -v hadolint &> /dev/null && echo '✓' || echo '✗')"
    echo "  - aspell: $(command -v aspell &> /dev/null && echo '✓' || echo '✗')"

    log_info ""
    log_info "You may need to restart your shell or run 'source ~/.bashrc' for PATH changes"
}

# Run main function
main "$@"