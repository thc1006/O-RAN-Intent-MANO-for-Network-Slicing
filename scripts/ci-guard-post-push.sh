#!/bin/bash
# CI Guardian v2025-09 Post-push Helper
# Automatically watches CI and handles failures

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛡️  CI Guardian v2025-09 - Post-push CI monitoring${NC}"
echo "==========================================================="

# Check if gh CLI is available
if ! command -v gh &> /dev/null; then
    echo -e "${RED}❌ Error: 'gh' CLI is not installed${NC}"
    echo -e "${YELLOW}   Install from: https://cli.github.com/${NC}"
    exit 1
fi

# Check if we're in a git repository
if ! git rev-parse --git-dir &> /dev/null; then
    echo -e "${RED}❌ Error: Not in a git repository${NC}"
    exit 1
fi

echo -e "${BLUE}🔄 Watching GitHub Actions run...${NC}"
echo -e "${YELLOW}   Press Ctrl+C to stop watching${NC}"

# Watch the CI run
if gh run watch --exit-status --interval 10 --compact; then
    echo -e "${GREEN}✅ GitHub Actions completed successfully!${NC}"
    exit 0
else
    echo -e "${RED}❌ GitHub Actions failed!${NC}"

    # Ask user if they want to rerun failed jobs
    echo -e "${YELLOW}🔄 Would you like to rerun failed jobs? (y/N)${NC}"
    read -r response

    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}🔄 Rerunning failed jobs...${NC}"
        if gh run rerun --failed-jobs-only; then
            echo -e "${GREEN}✅ Successfully restarted failed jobs${NC}"
            echo -e "${BLUE}🔄 Watching rerun...${NC}"
            gh run watch --exit-status --interval 10 --compact
        else
            echo -e "${RED}❌ Failed to rerun jobs${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}⚠️  CI failures not addressed. Please check the logs.${NC}"
        exit 1
    fi
fi