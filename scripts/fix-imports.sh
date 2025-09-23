#!/bin/bash

# Fix all imports from old module path to new module path
echo "Fixing imports in Go files..."

# Find all .go files and replace the import paths
find . -name "*.go" -type f | while read -r file; do
    # Skip vendor and .git directories
    if [[ "$file" == *"vendor"* ]] || [[ "$file" == *".git"* ]]; then
        continue
    fi

    # Replace the old import path with the new one
    sed -i 's|github\.com/o-ran/intent-mano|github.com/thc1006/O-RAN-Intent-MANO-for-Network-Slicing|g' "$file"

    # Check if file was modified
    if git diff --quiet "$file" 2>/dev/null; then
        :
    else
        echo "Fixed imports in: $file"
    fi
done

echo "Import fix complete!"