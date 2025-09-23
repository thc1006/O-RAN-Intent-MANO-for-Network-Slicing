#!/bin/bash
# Pre-commit hook for license header validation
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Checking license headers...${NC}"

EXIT_CODE=0

# Expected license header pattern (adjust as needed)
LICENSE_PATTERN="Copyright.*O-RAN Intent-MANO\|Licensed under.*Apache License\|SPDX-License-Identifier"

# Find all source files
SOURCE_FILES=$(find . -name "*.go" -o -name "*.py" -o -name "*.js" -o -name "*.ts" | grep -v vendor | grep -v node_modules | grep -v .git | grep -v "\.pb\.go$" | grep -v "_generated\.go$")

if [ -z "$SOURCE_FILES" ]; then
    echo -e "${YELLOW}No source files found to check.${NC}"
    exit 0
fi

MISSING_LICENSE=()

for source_file in $SOURCE_FILES; do
    # Check if file has license header (look in first 20 lines)
    if ! head -20 "$source_file" | grep -q "$LICENSE_PATTERN"; then
        MISSING_LICENSE+=("$source_file")
    fi
done

if [ ${#MISSING_LICENSE[@]} -eq 0 ]; then
    echo -e "${GREEN}All source files have proper license headers!${NC}"
else
    echo -e "${RED}❌ Files missing license headers:${NC}"
    for file in "${MISSING_LICENSE[@]}"; do
        echo "  - $file"
    done
    echo ""
    echo -e "${YELLOW}Please add appropriate license headers to the files listed above.${NC}"
    echo -e "${YELLOW}Example Go license header:${NC}"
    echo "/*"
    echo "Copyright 2024 O-RAN Intent-MANO Project"
    echo ""
    echo "Licensed under the Apache License, Version 2.0 (the \"License\");"
    echo "you may not use this file except in compliance with the License."
    echo "You may obtain a copy of the License at"
    echo ""
    echo "    http://www.apache.org/licenses/LICENSE-2.0"
    echo ""
    echo "Unless required by applicable law or agreed to in writing, software"
    echo "distributed under the License is distributed on an \"AS IS\" BASIS,"
    echo "WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied."
    echo "See the License for the specific language governing permissions and"
    echo "limitations under the License."
    echo "*/"
    EXIT_CODE=1
fi

exit $EXIT_CODE