#!/bin/bash

# Fix all imports from ethereum/go-ethereum to luxfi/geth
echo "Fixing imports from ethereum/go-ethereum to luxfi/geth..."

# Find all Go files and replace imports
find . -name "*.go" -type f -exec sed -i 's|github.com/ethereum/go-ethereum|github.com/luxfi/geth|g' {} \;

echo "Import fixes complete!"
echo "Running go mod tidy..."

# Clean up go.mod
go mod tidy

echo "Done!"