#!/bin/bash
# DevContainer Post-Create Script
# Runs once after container creation

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}🚀 Running post-create setup...${NC}"

# Set up Git configuration
if [ -d /workspace/.git ]; then
    git config --global --add safe.directory /workspace
    echo -e "${GREEN}✅ Git safe directory configured${NC}"
fi

# Verify tooling
echo -e "${YELLOW}Verifying installed tools...${NC}"
go version
python --version
kubectl version --client --short 2>/dev/null || echo "kubectl version check failed"
kind version 2>/dev/null || echo "kind not found"
helm version --short 2>/dev/null || echo "helm not found"

# Run security check
if [ -f /workspace/.devcontainer/scripts/devcontainer-security-check.sh ]; then
    echo ""
    bash /workspace/.devcontainer/scripts/devcontainer-security-check.sh || echo -e "${YELLOW}Security check completed with warnings${NC}"
fi

echo ""
echo -e "${GREEN}✅ Post-create setup complete!${NC}"
echo "Run 'make help' to see available commands"