#!/bin/bash
# Aggressive cleanup for mainnet launch

echo "🧹 Complete cleanup for mainnet launch..."

# Remove all runtime/output/test data
rm -rf runtime/
rm -rf output/
rm -rf build/

# Archive old scripts
mkdir -p .archive
mv scripts/* .archive/ 2>/dev/null || true
echo "Use the unified genesis tool instead: ./bin/genesis --help" > scripts/README.md

# Archive old tools
mv tools/* .archive/ 2>/dev/null || true
rmdir tools 2>/dev/null || true

# Remove old cmd directories (keep only genesis)
find cmd -type d -name "bin" -exec rm -rf {} + 2>/dev/null
rm -rf cmd/archeology cmd/teleport cmd/namespace cmd/migrate* cmd/extract* cmd/fetch* cmd/import* cmd/process* cmd/read* cmd/subnet* cmd/test* cmd/build* cmd/genesis-builder cmd/genesis-cli cmd/prefixscan 2>/dev/null

# Clean up test artifacts
find test -name ".tmp" -type d -exec rm -rf {} + 2>/dev/null
find test -name "*.out" -delete 2>/dev/null
find test -name "*.test" -delete 2>/dev/null
rm -rf test/\$HOME

# Remove old tree file
rm -f tree.txt

# Remove temp files
rm -f cleanup_final.sh
rm -f verify_migrated_data.go
rm -f consensus.go

# Clean chaindata LOCK files
find chaindata -name "LOCK" -type f -delete 2>/dev/null || true

echo ""
echo "✅ Cleanup complete!"
echo ""
echo "Clean structure:"
echo "├── chaindata/     # Source blockchain data"
echo "├── configs/       # Network configurations" 
echo "├── cmd/genesis/   # Genesis tool source"
echo "├── pkg/           # Go packages"
echo "├── test/          # Test files"
echo "├── deployments/   # Deployment configs"
echo "├── docker/        # Docker configs"
echo "├── docs/          # Documentation"
echo "├── bin/           # Built binaries"
echo "├── scripts/       # (archived)"
echo "├── Makefile       # Build system"
echo "├── README.md      # Main documentation"
echo "└── LLM.md         # AI guide"
echo ""
echo "To run: make"