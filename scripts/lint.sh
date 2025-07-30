#!/bin/bash
set -e

echo "Running Go static analysis..."

# Run go fmt check
echo "Checking go fmt..."
UNFORMATTED=$(gofmt -l .)
if [ -n "$UNFORMATTED" ]; then
    echo "The following files are not formatted:"
    echo "$UNFORMATTED"
    exit 1
fi

# Run go vet
echo "Running go vet..."
go vet ./...

# Run go mod tidy check
echo "Checking go mod tidy..."
go mod tidy
if [ -n "$(git status --porcelain go.mod go.sum)" ]; then
    echo "go mod tidy modified go.mod or go.sum. Please run 'go mod tidy' and commit the changes."
    exit 1
fi

echo "All checks passed!"