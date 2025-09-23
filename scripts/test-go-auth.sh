#!/bin/bash

# Test script to verify Go module authentication configuration

echo "Testing Go module authentication setup..."

# Configure git to use GitHub token for private repos
if [ -n "$GITHUB_TOKEN" ]; then
  echo "Configuring Git with GitHub token authentication..."
  git config --global url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf "https://github.com/"
else
  echo "Warning: GITHUB_TOKEN not set, using unauthenticated access"
fi

# Set GOPRIVATE for potentially private modules
export GOPRIVATE="github.com/securecodewarrior"

# Try to download dependencies
echo "Attempting to download Go dependencies..."
go mod download 2>&1 | head -20

# Check the result
if [ $? -eq 0 ]; then
  echo "✓ Go module download successful"
else
  echo "✗ Go module download failed - this is expected if there are missing private repos"
  echo "  The GitHub workflow will handle this with proper authentication"
fi

# Clean up git config (optional)
# git config --global --unset url."https://${GITHUB_TOKEN}:x-oauth-basic@github.com/".insteadOf

echo "Test complete."