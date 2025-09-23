#!/bin/bash
# Lint checking script for Python, Go, YAML, and Shell

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ERRORS=0

log_info() {
    echo -e "${GREEN}[LINT]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[LINT]${NC} $1"
}

log_error() {
    echo -e "${RED}[LINT]${NC} $1"
}

# Python linting
lint_python() {
    log_info "Checking Python files..."

    # Find Python files
    local python_files=$(find "$PROJECT_ROOT" -name "*.py" -type f 2>/dev/null | grep -v venv | grep -v ".git" || true)

    if [ -z "$python_files" ]; then
        log_info "No Python files found"
        return 0
    fi

    # Check if ruff is available
    if command -v ruff &> /dev/null; then
        log_info "Running ruff..."
        if ! ruff check $python_files 2>/dev/null; then
            log_error "Python linting failed (ruff)"
            ERRORS=$((ERRORS + 1))
        fi
    elif command -v flake8 &> /dev/null; then
        log_info "Running flake8..."
        if ! flake8 $python_files --max-line-length=120 --ignore=E203,W503 2>/dev/null; then
            log_error "Python linting failed (flake8)"
            ERRORS=$((ERRORS + 1))
        fi
    else
        log_warn "No Python linter found (install ruff or flake8)"
    fi

    # Type checking with mypy if available
    if command -v mypy &> /dev/null; then
        log_info "Running mypy type checking..."
        if ! mypy $python_files --ignore-missing-imports 2>/dev/null; then
            log_warn "Type checking warnings found"
        fi
    fi
}

# Go linting
lint_go() {
    log_info "Checking Go files..."

    # Find Go modules
    local go_modules=$(find "$PROJECT_ROOT" -name "go.mod" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$go_modules" ]; then
        log_info "No Go modules found"
        return 0
    fi

    # Check each Go module
    for mod in $go_modules; do
        local mod_dir=$(dirname "$mod")
        log_info "Checking Go module: $mod_dir"

        cd "$mod_dir"

        # Run go vet
        if command -v go &> /dev/null; then
            log_info "Running go vet..."
            if ! go vet ./... 2>/dev/null; then
                log_error "Go vet failed in $mod_dir"
                ERRORS=$((ERRORS + 1))
            fi
        fi

        # Run golangci-lint if available
        if command -v golangci-lint &> /dev/null; then
            log_info "Running golangci-lint..."
            if ! golangci-lint run --timeout=5m 2>/dev/null; then
                log_error "golangci-lint failed in $mod_dir"
                ERRORS=$((ERRORS + 1))
            fi
        else
            log_warn "golangci-lint not found (recommended for Go linting)"
        fi

        cd "$PROJECT_ROOT"
    done
}

# YAML linting
lint_yaml() {
    log_info "Checking YAML files..."

    # Find YAML files
    local yaml_files=$(find "$PROJECT_ROOT" -name "*.yaml" -o -name "*.yml" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$yaml_files" ]; then
        log_info "No YAML files found"
        return 0
    fi

    if command -v yamllint &> /dev/null; then
        log_info "Running yamllint..."

        # Create yamllint config if not exists
        if [ ! -f "$PROJECT_ROOT/.yamllint" ]; then
            cat > "$PROJECT_ROOT/.yamllint" << 'EOF'
---
extends: default

rules:
  line-length:
    max: 120
    level: warning
  comments:
    min-spaces-from-content: 1
  comments-indentation: disable
  document-start: disable
  truthy:
    allowed-values: ['true', 'false', 'yes', 'no', 'on', 'off']
EOF
        fi

        for file in $yaml_files; do
            if ! yamllint -c "$PROJECT_ROOT/.yamllint" "$file" 2>/dev/null; then
                log_error "YAML linting failed: $file"
                ERRORS=$((ERRORS + 1))
            fi
        done
    else
        log_warn "yamllint not found (install for YAML validation)"
    fi
}

# Shell script linting
lint_shell() {
    log_info "Checking shell scripts..."

    # Find shell scripts
    local shell_files=$(find "$PROJECT_ROOT" -name "*.sh" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$shell_files" ]; then
        log_info "No shell scripts found"
        return 0
    fi

    if command -v shellcheck &> /dev/null; then
        log_info "Running shellcheck..."
        for file in $shell_files; do
            if ! shellcheck "$file" 2>/dev/null; then
                log_error "Shell linting failed: $file"
                ERRORS=$((ERRORS + 1))
            fi
        done
    else
        log_warn "shellcheck not found (install for shell script validation)"
    fi
}

# Dockerfile linting
lint_docker() {
    log_info "Checking Dockerfiles..."

    # Find Dockerfiles
    local docker_files=$(find "$PROJECT_ROOT" -name "Dockerfile*" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$docker_files" ]; then
        log_info "No Dockerfiles found"
        return 0
    fi

    if command -v hadolint &> /dev/null; then
        log_info "Running hadolint..."
        for file in $docker_files; do
            if ! hadolint "$file" 2>/dev/null; then
                log_error "Dockerfile linting failed: $file"
                ERRORS=$((ERRORS + 1))
            fi
        done
    else
        log_warn "hadolint not found (install for Dockerfile validation)"
    fi
}

# Main function
main() {
    cd "$PROJECT_ROOT"

    log_info "Starting lint checks..."

    lint_python
    lint_go
    lint_yaml
    lint_shell
    lint_docker

    if [ $ERRORS -gt 0 ]; then
        log_error "Lint check failed with $ERRORS error(s)"
        exit 1
    else
        log_info "All lint checks passed!"
    fi
}

main "$@"