#!/bin/bash
# Pre-commit hook for running gosec security scanner
# Part of O-RAN Intent-MANO security scanning pipeline

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Running gosec security scan...${NC}"

# Check if gosec is installed
if ! command -v gosec &> /dev/null; then
    echo -e "${YELLOW}Installing gosec...${NC}"
    go install github.com/securego/gosec/v2/cmd/gosec@latest
fi

# Create temporary configuration file
TEMP_CONFIG=$(mktemp)
cat > "$TEMP_CONFIG" << 'EOF'
{
  "severity": "medium",
  "confidence": "medium",
  "exclude-generated": true,
  "exclude-dirs": [
    "vendor",
    "node_modules",
    ".git",
    "tests/golden"
  ],
  "exclude-rules": [
    "G104",
    "G304"
  ],
  "include-rules": [
    "G101", "G102", "G103", "G106", "G107", "G108", "G109", "G110",
    "G201", "G202", "G203", "G204", "G301", "G302", "G303", "G305",
    "G401", "G402", "G403", "G404", "G501", "G502", "G503", "G504",
    "G505", "G601"
  ]
}
EOF

# Run gosec scan
EXIT_CODE=0
if ! gosec -config="$TEMP_CONFIG" -fmt=text ./...; then
    EXIT_CODE=$?
    echo -e "${RED}Security issues found by gosec!${NC}"
    echo -e "${YELLOW}Please review and fix the security issues before committing.${NC}"
else
    echo -e "${GREEN}No security issues found by gosec.${NC}"
fi

# Cleanup
rm -f "$TEMP_CONFIG"

exit $EXIT_CODE