#!/bin/bash
# Fix imports to use standard ethereum package

echo "Fixing imports from luxfi/geth to ethereum/go-ethereum..."

# Find and replace in all Go files
find . -name "*.go" -type f -exec sed -i 's|github.com/luxfi/geth|github.com/ethereum/go-ethereum|g' {} \;

echo "Done!"