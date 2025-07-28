#!/bin/bash

echo "Building all genesis tools..."

# List of main tools to build
TOOLS=(
    "genesis:./cmd/genesis"
    "archeology:./cmd/archeology" 
    "extract-genesis:./cmd/extract-genesis"
    "extract-full-state:./cmd/extract-full-state"
    "read-state:./cmd/read-state"
)

# Build each tool
for tool_spec in "${TOOLS[@]}"; do
    IFS=':' read -r name path <<< "$tool_spec"
    echo -n "Building $name... "
    if go build -o "bin/$name" "$path" 2>/tmp/build-error.log; then
        echo "✓"
    else
        echo "✗"
        echo "Error building $name:"
        cat /tmp/build-error.log | head -10
        echo ""
    fi
done

# Build standalone tools
echo ""
echo "Building standalone migration tools..."

# Find all .go files in the root that look like tools
for file in *.go; do
    if [[ -f "$file" && "$file" != "fix-rawdb-constants.go" ]]; then
        name=$(basename "$file" .go)
        echo -n "Building $name... "
        if go build -o "bin/$name" "$file" 2>/tmp/build-error.log; then
            echo "✓"
        else
            echo "✗"
            echo "Error details:"
            cat /tmp/build-error.log | head -5
        fi
    fi
done

echo ""
echo "Build complete! Check bin/ directory for compiled tools."
ls -la bin/ | grep -E "^-rwx" | wc -l | xargs echo "Total executables:"