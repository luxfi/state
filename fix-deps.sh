#!/bin/bash
set -e

echo "Fixing dependencies..."

# Update go.mod to remove version conflicts
go mod edit -droprequire=github.com/luxfi/crypto
go mod edit -replace=github.com/luxfi/crypto=github.com/luxfi/crypto@v0.1.1

# Try to download and tidy
go mod download || true
go mod tidy -e || true

# If still failing, force update
go get -u github.com/luxfi/ids@v0.1.0
go get -u github.com/luxfi/node@v1.13.3

# Final tidy
go mod tidy

echo "Dependencies fixed!"