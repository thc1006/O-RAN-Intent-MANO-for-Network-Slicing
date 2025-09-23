#!/bin/bash
# Format checking script for code consistency

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ERRORS=0

log_info() {
    echo -e "${GREEN}[FORMAT]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[FORMAT]${NC} $1"
}

log_error() {
    echo -e "${RED}[FORMAT]${NC} $1"
}

# Python formatting check
check_python_format() {
    log_info "Checking Python formatting..."

    # Find Python files
    local python_files=$(find "$PROJECT_ROOT" -name "*.py" -type f 2>/dev/null | grep -v venv | grep -v ".git" || true)

    if [ -z "$python_files" ]; then
        log_info "No Python files found"
        return 0
    fi

    if command -v black &> /dev/null; then
        log_info "Running black..."
        for file in $python_files; do
            if ! black --check --quiet "$file" 2>/dev/null; then
                log_error "Python formatting issues in: $file"
                log_warn "Run 'black $file' to fix"
                ERRORS=$((ERRORS + 1))
            fi
        done
    else
        log_warn "black not found (install for Python formatting)"
    fi

    # Check import sorting with isort
    if command -v isort &> /dev/null; then
        log_info "Checking import sorting..."
        for file in $python_files; do
            if ! isort --check-only --quiet "$file" 2>/dev/null; then
                log_warn "Import sorting issues in: $file"
                log_warn "Run 'isort $file' to fix"
            fi
        done
    fi
}

# Go formatting check
check_go_format() {
    log_info "Checking Go formatting..."

    # Find Go files
    local go_files=$(find "$PROJECT_ROOT" -name "*.go" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$go_files" ]; then
        log_info "No Go files found"
        return 0
    fi

    if command -v gofmt &> /dev/null; then
        log_info "Running gofmt..."
        for file in $go_files; do
            if [ -n "$(gofmt -l "$file" 2>/dev/null)" ]; then
                log_error "Go formatting issues in: $file"
                log_warn "Run 'gofmt -w $file' to fix"
                ERRORS=$((ERRORS + 1))
            fi
        done
    fi

    # Check imports with goimports
    if command -v goimports &> /dev/null; then
        log_info "Checking Go imports..."
        for file in $go_files; do
            if [ -n "$(goimports -l "$file" 2>/dev/null)" ]; then
                log_warn "Go import issues in: $file"
                log_warn "Run 'goimports -w $file' to fix"
            fi
        done
    fi
}

# YAML formatting check
check_yaml_format() {
    log_info "Checking YAML formatting..."

    # Find YAML files
    local yaml_files=$(find "$PROJECT_ROOT" \( -name "*.yaml" -o -name "*.yml" \) -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$yaml_files" ]; then
        log_info "No YAML files found"
        return 0
    fi

    for file in $yaml_files; do
        # Check for tabs (YAML should use spaces)
        if grep -q $'\t' "$file"; then
            log_error "YAML file contains tabs: $file"
            log_warn "YAML files should use spaces, not tabs"
            ERRORS=$((ERRORS + 1))
        fi

        # Check for trailing whitespace
        if grep -q '[[:space:]]$' "$file"; then
            log_warn "Trailing whitespace in: $file"
        fi
    done
}

# JSON formatting check
check_json_format() {
    log_info "Checking JSON formatting..."

    # Find JSON files
    local json_files=$(find "$PROJECT_ROOT" -name "*.json" -type f 2>/dev/null | grep -v ".git" | grep -v node_modules || true)

    if [ -z "$json_files" ]; then
        log_info "No JSON files found"
        return 0
    fi

    for file in $json_files; do
        if command -v python3 &> /dev/null; then
            if ! python3 -m json.tool "$file" > /dev/null 2>&1; then
                log_error "Invalid JSON format: $file"
                ERRORS=$((ERRORS + 1))
            else
                # Check if file is pretty-printed
                local original=$(cat "$file" | tr -d '[:space:]')
                local formatted=$(python3 -m json.tool "$file" | tr -d '[:space:]')
                if [ "$original" != "$formatted" ]; then
                    log_warn "JSON not properly formatted: $file"
                    log_warn "Run 'python3 -m json.tool $file > temp && mv temp $file' to fix"
                fi
            fi
        elif command -v jq &> /dev/null; then
            if ! jq . "$file" > /dev/null 2>&1; then
                log_error "Invalid JSON format: $file"
                ERRORS=$((ERRORS + 1))
            fi
        fi
    done
}

# Markdown formatting check
check_markdown_format() {
    log_info "Checking Markdown formatting..."

    # Find Markdown files
    local md_files=$(find "$PROJECT_ROOT" -name "*.md" -type f 2>/dev/null | grep -v ".git" | grep -v node_modules || true)

    if [ -z "$md_files" ]; then
        log_info "No Markdown files found"
        return 0
    fi

    if command -v markdownlint &> /dev/null || command -v mdl &> /dev/null; then
        for file in $md_files; do
            if command -v markdownlint &> /dev/null; then
                if ! markdownlint "$file" 2>/dev/null; then
                    log_warn "Markdown formatting issues in: $file"
                fi
            elif command -v mdl &> /dev/null; then
                if ! mdl "$file" 2>/dev/null; then
                    log_warn "Markdown formatting issues in: $file"
                fi
            fi
        done
    else
        log_info "No Markdown linter found (optional)"
    fi

    # Check for trailing whitespace
    for file in $md_files; do
        if grep -q '[[:space:]]$' "$file"; then
            log_warn "Trailing whitespace in: $file"
        fi
    done
}

# Shell script formatting check
check_shell_format() {
    log_info "Checking shell script formatting..."

    # Find shell scripts
    local shell_files=$(find "$PROJECT_ROOT" -name "*.sh" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$shell_files" ]; then
        log_info "No shell scripts found"
        return 0
    fi

    if command -v shfmt &> /dev/null; then
        log_info "Running shfmt..."
        for file in $shell_files; do
            if ! shfmt -d "$file" > /dev/null 2>&1; then
                log_error "Shell formatting issues in: $file"
                log_warn "Run 'shfmt -w $file' to fix"
                ERRORS=$((ERRORS + 1))
            fi
        done
    else
        log_info "shfmt not found (optional for shell formatting)"
    fi

    # Check for trailing whitespace
    for file in $shell_files; do
        if grep -q '[[:space:]]$' "$file"; then
            log_warn "Trailing whitespace in: $file"
        fi
    done
}

# General formatting checks
check_general_format() {
    log_info "Checking general formatting rules..."

    # Check for files without final newline
    local all_text_files=$(find "$PROJECT_ROOT" \
        \( -name "*.py" -o -name "*.go" -o -name "*.js" -o -name "*.ts" \
           -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" \
           -o -name "*.md" -o -name "*.sh" -o -name "*.txt" \) \
        -type f 2>/dev/null | grep -v ".git" | grep -v node_modules || true)

    for file in $all_text_files; do
        if [ -s "$file" ] && [ -z "$(tail -c 1 "$file")" ]; then
            : # File ends with newline, good
        else
            if [ -s "$file" ]; then
                log_warn "Missing final newline: $file"
            fi
        fi
    done

    # Check for files with CRLF line endings (should be LF)
    for file in $all_text_files; do
        if file "$file" 2>/dev/null | grep -q "CRLF"; then
            log_error "CRLF line endings found: $file"
            log_warn "Convert to LF line endings"
            ERRORS=$((ERRORS + 1))
        fi
    done
}

# Main function
main() {
    cd "$PROJECT_ROOT"

    log_info "Starting format checks..."

    check_python_format
    check_go_format
    check_yaml_format
    check_json_format
    check_markdown_format
    check_shell_format
    check_general_format

    if [ $ERRORS -gt 0 ]; then
        log_error "Format check failed with $ERRORS error(s)"
        exit 1
    else
        log_info "All format checks passed!"
    fi
}

main "$@"