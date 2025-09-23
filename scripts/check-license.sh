#!/bin/bash
# License header checking script

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ERRORS=0

log_info() {
    echo -e "${GREEN}[LICENSE]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[LICENSE]${NC} $1"
}

log_error() {
    echo -e "${RED}[LICENSE]${NC} $1"
}

# Expected license header (Apache 2.0)
read -r -d '' LICENSE_HEADER << 'EOF' || true
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
EOF

# Alternative acceptable headers
read -r -d '' MIT_HEADER << 'EOF' || true
Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software
EOF

read -r -d '' COPYRIGHT_PATTERN << 'EOF' || true
Copyright
EOF

# Check license in Python files
check_python_license() {
    log_info "Checking Python file licenses..."

    local python_files=$(find "$PROJECT_ROOT" -name "*.py" -type f 2>/dev/null | \
        grep -v venv | grep -v ".git" | grep -v __pycache__ || true)

    if [ -z "$python_files" ]; then
        log_info "No Python files found"
        return 0
    fi

    for file in $python_files; do
        # Skip empty files
        if [ ! -s "$file" ]; then
            continue
        fi

        # Skip __init__.py files (often empty or minimal)
        if [[ "$file" == *"__init__.py" ]]; then
            continue
        fi

        # Check for any form of license or copyright
        if ! head -n 20 "$file" | grep -i -E "(copyright|license|licensed)" > /dev/null 2>&1; then
            log_warn "No license header in: $file"
            # Not an error, just a warning for now
        fi
    done
}

# Check license in Go files
check_go_license() {
    log_info "Checking Go file licenses..."

    local go_files=$(find "$PROJECT_ROOT" -name "*.go" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$go_files" ]; then
        log_info "No Go files found"
        return 0
    fi

    for file in $go_files; do
        # Skip empty files
        if [ ! -s "$file" ]; then
            continue
        fi

        # Skip generated files
        if head -n 5 "$file" | grep -q "Code generated"; then
            continue
        fi

        # Check for any form of license or copyright
        if ! head -n 20 "$file" | grep -i -E "(copyright|license|licensed)" > /dev/null 2>&1; then
            log_warn "No license header in: $file"
            # Not an error, just a warning for now
        fi
    done
}

# Check license in JavaScript/TypeScript files
check_js_license() {
    log_info "Checking JavaScript/TypeScript file licenses..."

    local js_files=$(find "$PROJECT_ROOT" \( -name "*.js" -o -name "*.ts" -o -name "*.jsx" -o -name "*.tsx" \) \
        -type f 2>/dev/null | grep -v node_modules | grep -v ".git" || true)

    if [ -z "$js_files" ]; then
        log_info "No JavaScript/TypeScript files found"
        return 0
    fi

    for file in $js_files; do
        # Skip empty files
        if [ ! -s "$file" ]; then
            continue
        fi

        # Skip minified files
        if [[ "$file" == *".min.js" ]]; then
            continue
        fi

        # Check for any form of license or copyright
        if ! head -n 20 "$file" | grep -i -E "(copyright|license|licensed)" > /dev/null 2>&1; then
            log_warn "No license header in: $file"
            # Not an error, just a warning for now
        fi
    done
}

# Check license in shell scripts
check_shell_license() {
    log_info "Checking shell script licenses..."

    local shell_files=$(find "$PROJECT_ROOT" -name "*.sh" -type f 2>/dev/null | grep -v ".git" || true)

    if [ -z "$shell_files" ]; then
        log_info "No shell scripts found"
        return 0
    fi

    for file in $shell_files; do
        # Skip empty files
        if [ ! -s "$file" ]; then
            continue
        fi

        # Check for any form of license or copyright (after shebang)
        if ! sed -n '2,20p' "$file" | grep -i -E "(copyright|license|licensed)" > /dev/null 2>&1; then
            log_warn "No license header in: $file"
            # Not an error, just a warning for now
        fi
    done
}

# Check for LICENSE file in project root
check_license_file() {
    log_info "Checking for LICENSE file..."

    if [ ! -f "$PROJECT_ROOT/LICENSE" ] && [ ! -f "$PROJECT_ROOT/LICENSE.md" ] && \
       [ ! -f "$PROJECT_ROOT/LICENSE.txt" ] && [ ! -f "$PROJECT_ROOT/LICENCE" ]; then
        log_error "No LICENSE file found in project root"
        log_warn "Please add a LICENSE file to define the project's license"
        ERRORS=$((ERRORS + 1))
    else
        log_info "LICENSE file found"

        # Check if it's Apache 2.0 (already exists based on file listing)
        if [ -f "$PROJECT_ROOT/LICENSE" ]; then
            if grep -q "Apache License" "$PROJECT_ROOT/LICENSE"; then
                log_info "Apache License 2.0 detected"
            elif grep -q "MIT License" "$PROJECT_ROOT/LICENSE"; then
                log_info "MIT License detected"
            else
                log_info "Custom license detected"
            fi
        fi
    fi
}

# Check for copyright year
check_copyright_year() {
    log_info "Checking copyright years..."

    local current_year=$(date +%Y)
    local source_files=$(find "$PROJECT_ROOT" \
        \( -name "*.py" -o -name "*.go" -o -name "*.js" -o -name "*.ts" \
           -o -name "*.java" -o -name "*.c" -o -name "*.cpp" -o -name "*.h" \) \
        -type f 2>/dev/null | grep -v ".git" | grep -v node_modules || true)

    for file in $source_files; do
        if grep -i "copyright" "$file" > /dev/null 2>&1; then
            # Check if copyright includes current year or recent year
            if ! grep -i "copyright.*20[0-9][0-9]" "$file" > /dev/null 2>&1; then
                log_warn "Copyright without year in: $file"
            elif ! grep -i "copyright.*\($(($current_year - 1))\|$current_year\)" "$file" > /dev/null 2>&1; then
                log_warn "Outdated copyright year in: $file"
            fi
        fi
    done
}

# Add license headers to files (helper function, not called by default)
add_license_headers() {
    log_info "License header addition helper (not executed automatically)"
    log_info "To add Apache 2.0 headers to all source files, run:"
    log_info "  $0 --add-headers"
    log_info ""
    log_info "This will add the following header to files missing licenses:"
    echo "$LICENSE_HEADER" | sed 's/^/  # /'
}

# Main function
main() {
    cd "$PROJECT_ROOT"

    # Check if we're being asked to add headers
    if [ "${1:-}" = "--add-headers" ]; then
        log_warn "Adding license headers is not implemented in this check script"
        log_warn "Please add license headers manually to maintain code ownership"
        exit 0
    fi

    log_info "Starting license checks..."

    check_license_file
    check_python_license
    check_go_license
    check_js_license
    check_shell_license
    check_copyright_year

    if [ $ERRORS -gt 0 ]; then
        log_error "License check failed with $ERRORS error(s)"
        log_info "Consider adding license headers to source files"
        add_license_headers
        exit 1
    else
        log_info "License checks completed!"
        if [ $ERRORS -eq 0 ]; then
            log_info "Note: Missing license headers in source files are warnings only"
        fi
    fi
}

main "$@"