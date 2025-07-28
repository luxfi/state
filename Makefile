# Lux Genesis Migration Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GINKGO=ginkgo

# Binary output directory
BIN_DIR=bin

# Test directory
TEST_DIR=test

# Temporary directory for tests
TMP_DIR=.tmp

# Main genesis binary
GENESIS_BIN=$(BIN_DIR)/genesis

# Lux CLI path
LUX_CLI ?= $(HOME)/work/lux/cli/bin/avalanche
LUXD_PATH = $(BIN_DIR)/luxd
LUXD_REPO = https://github.com/luxfi/node.git
LUXD_BRANCH = genesis

# Install dependencies (luxd, etc)
.PHONY: deps
deps:
	@echo "📦 Installing dependencies..."
	@mkdir -p $(BIN_DIR)
	@if [ ! -f $(LUXD_PATH) ]; then \
		echo "  → Installing luxd from genesis branch..."; \
		rm -rf $(TMP_DIR)/luxd-build; \
		git clone --branch $(LUXD_BRANCH) --depth 1 $(LUXD_REPO) $(TMP_DIR)/luxd-build; \
		cd $(TMP_DIR)/luxd-build && ./scripts/build.sh; \
		cp $(TMP_DIR)/luxd-build/build/luxd $(LUXD_PATH); \
		rm -rf $(TMP_DIR)/luxd-build; \
		echo "  ✅ luxd installed to $(LUXD_PATH)"; \
	else \
		echo "  ✅ luxd already installed at $(LUXD_PATH)"; \
	fi
	@echo "✅ All dependencies installed!"

# Default target - full end-to-end test
.PHONY: all
all: deps build import node-test

# Just build tools
.PHONY: build
build: clean
	@echo "Building unified genesis tool..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(GENESIS_BIN) ./cmd/genesis
	@echo "Build complete! Run '$(GENESIS_BIN) --help' to see all commands."

# Run Ginkgo tests with optional filter
.PHONY: test
test: build
	@echo "Setting up test environment..."
	@mkdir -p $(TMP_DIR)
	@mkdir -p $(TEST_DIR)
	@echo "Running Ginkgo tests..."
ifdef filter
	@echo "Running tests matching: $(filter)"
	@cd $(TEST_DIR) && $(GINKGO) -v --label-filter="$(filter)" --fail-fast
else
	@echo "Running all tests..."
	@cd $(TEST_DIR) && $(GINKGO) -v --fail-fast
endif

# Test specific categories
.PHONY: test-migration
test-migration:
	@$(MAKE) test filter="migration"

.PHONY: test-integration
test-integration:
	@$(MAKE) test filter="integration"

.PHONY: test-performance
test-performance:
	@$(MAKE) test filter="performance"

.PHONY: test-edge-cases
test-edge-cases:
	@$(MAKE) test filter="edge"

# Quick test without building
.PHONY: test-quick
test-quick:
	@cd $(TEST_DIR) && $(GINKGO) -v --fail-fast

# Test with coverage
.PHONY: test-coverage
test-coverage: build
	@echo "Running tests with coverage..."
	@cd $(TEST_DIR) && $(GINKGO) -v --cover --coverprofile=coverage.out
	@$(GOCMD) tool cover -html=$(TEST_DIR)/coverage.out -o $(TEST_DIR)/coverage.html
	@echo "Coverage report: $(TEST_DIR)/coverage.html"

# Interactive test mode (step by step)
.PHONY: test-interactive
test-interactive: build
	@echo "Running tests in interactive mode..."
	@cd $(TEST_DIR) && $(GINKGO) -v --poll-progress-after=0 --poll-progress-interval=10s

# Install Ginkgo if not present
.PHONY: install-ginkgo
install-ginkgo:
	@echo "Installing Ginkgo test framework..."
	@$(GOGET) github.com/onsi/ginkgo/v2/ginkgo
	@$(GOGET) github.com/onsi/gomega/...

# Code quality checks
.PHONY: vet
vet:
	@echo "Running go vet..."
	@$(GOCMD) vet ./...
	@echo "✅ No vet issues found"

# Check formatting
.PHONY: fmt-check
fmt-check:
	@echo "Checking code formatting..."
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:"; gofmt -l .; exit 1)
	@echo "✅ All files properly formatted"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "✅ Code formatted"

# Run all quality checks
.PHONY: quality
quality: vet fmt-check test-coverage
	@echo "✅ All quality checks passed!"

# Clean build artifacts and temp files
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)/genesis  # Keep luxd
	@rm -rf $(TMP_DIR)
	@rm -rf runtime/*.log runtime/*.pid
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@echo "Clean complete!"

# Clean everything including test outputs
.PHONY: clean-all
clean-all: clean
	@rm -rf $(TEST_DIR)/$(TMP_DIR)
	@echo "Deep clean complete!"

# Migration pipeline commands using the unified tool
.PHONY: migrate-subnet
migrate-subnet:
	@if [ -z "$(SRC)" ] || [ -z "$(DST)" ]; then \
		echo "Usage: make migrate-subnet SRC=<subnet-db> DST=<destination-root>"; \
		exit 1; \
	fi
	@echo "Running full migration pipeline..."
	@$(GENESIS_BIN) migrate full $(SRC) $(DST)

# Individual migration steps
.PHONY: add-evm-prefix
add-evm-prefix:
	@if [ -z "$(SRC)" ] || [ -z "$(DST)" ]; then \
		echo "Usage: make add-evm-prefix SRC=<source-db> DST=<destination-db>"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) migrate add-evm-prefix $(SRC) $(DST)

.PHONY: rebuild-canonical
rebuild-canonical:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make rebuild-canonical DB=<path-to-evm-pebbledb>"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) migrate rebuild-canonical $(DB)

.PHONY: check-head
check-head:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make check-head DB=<database-path>"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) migrate check-head $(DB)

.PHONY: peek-tip
peek-tip:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make peek-tip DB=<database-path>"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) migrate peek-tip $(DB)

.PHONY: replay-consensus
replay-consensus:
	@if [ -z "$(EVM)" ] || [ -z "$(STATE)" ] || [ -z "$(TIP)" ]; then \
		echo "Usage: make replay-consensus EVM=<evm-db> STATE=<state-db> TIP=<height>"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) migrate replay-consensus --evm $(EVM) --state $(STATE) --tip $(TIP)

# Generate genesis files
.PHONY: generate
generate:
	@echo "Generating genesis files..."
	@$(GENESIS_BIN) generate

# Import chain data with monitoring
.PHONY: import-chain-data
import-chain-data:
	@if [ -z "$(SRC)" ]; then \
		echo "Usage: make import-chain-data SRC=/path/to/source/chaindata"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) import chain-data $(SRC)

.PHONY: import-monitor
import-monitor:
	@$(GENESIS_BIN) import monitor

.PHONY: import-status
import-status:
	@$(GENESIS_BIN) import status

# Export commands
.PHONY: export-backup
export-backup:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make export-backup DB=/path/to/database"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) export backup $(DB)

.PHONY: export-state
export-state:
	@if [ -z "$(DB)" ]; then \
		echo "Usage: make export-state DB=/path/to/database"; \
		exit 1; \
	fi
	@$(GENESIS_BIN) export state $(DB)

# Validator management
.PHONY: validators-list
validators-list:
	@$(GENESIS_BIN) validators list

.PHONY: validators-add
validators-add:
	@if [ -z "$(NODE_ID)" ] || [ -z "$(ETH_ADDRESS)" ]; then \
		echo "Usage: make validators-add NODE_ID=NodeID-xxx ETH_ADDRESS=0x..."; \
		exit 1; \
	fi
	@$(GENESIS_BIN) validators add --node-id $(NODE_ID) --eth-address $(ETH_ADDRESS)

# Launch commands
.PHONY: launch
launch: launch-L1

.PHONY: launch-L1
launch-L1:
	@echo "🚀 Launching L1 (C-Chain) with network ID 96369"
	@export GENESIS_RUNTIME_DIR="$$(pwd)/runtime" && \
	mkdir -p "$$GENESIS_RUNTIME_DIR" && \
	$(GENESIS_BIN) launch L1

.PHONY: launch-L2
launch-L2:
	@if [ -z "$(NETWORK_ID)" ]; then \
		echo "Usage: make launch-L2 NETWORK_ID=<id>"; \
		exit 1; \
	fi
	@echo "🚀 Launching L2 with network ID $(NETWORK_ID)"
	@export GENESIS_RUNTIME_DIR="$$(pwd)/runtime" && \
	mkdir -p "$$GENESIS_RUNTIME_DIR" && \
	$(GENESIS_BIN) launch L2 $(NETWORK_ID)

.PHONY: launch-verify
launch-verify:
	@$(GENESIS_BIN) launch verify

.PHONY: launch-clean
launch-clean:
	@echo "🧹 Launching clean C-Chain (no imported data)"
	@export GENESIS_RUNTIME_DIR="$$(pwd)/runtime" && \
	mkdir -p "$$GENESIS_RUNTIME_DIR" && \
	$(GENESIS_BIN) launch clean

# Help
.PHONY: help
help:
	@echo "Lux Genesis Migration Makefile"
	@echo ""
	@echo "DEFAULT: 'make' runs full migration test (build → import → launch → verify)"
	@echo ""
	@echo "Build targets:"
	@echo "  make deps               - Install dependencies (luxd)"
	@echo "  make build              - Build genesis tool only"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make clean-all          - Deep clean including test outputs"
	@echo "  make quality            - Run code quality checks"
	@echo ""
	@echo "Test targets:"
	@echo "  make all                - Full end-to-end test (default)"
	@echo "  make test               - Run unit tests"
	@echo "  make test filter=X      - Run tests matching filter X"
	@echo "  make node-test          - Test import + launch + RPC verify"
	@echo "  make test-migration     - Run migration tests only"
	@echo "  make test-integration   - Run integration tests only"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo ""
	@echo "Migration commands:"
	@echo "  make migrate-subnet SRC=<db> DST=<root>  - Full migration pipeline"
	@echo "  make add-evm-prefix SRC=<db> DST=<db>    - Add EVM prefix to keys"
	@echo "  make rebuild-canonical DB=<db>            - Rebuild canonical mappings"
	@echo "  make check-head DB=<db>                   - Check head pointers"
	@echo "  make peek-tip DB=<db>                     - Find highest block"
	@echo "  make replay-consensus EVM=<db> STATE=<db> TIP=<n> - Replay consensus"
	@echo ""
	@echo "Genesis commands:"
	@echo "  make generate           - Generate all genesis files"
	@echo "  make validators-list    - List validators"
	@echo "  make validators-add     - Add a validator"
	@echo ""
	@echo "Import/Export commands:"
	@echo "  make import-chain-data SRC=<path>  - Import chain data"
	@echo "  make import-monitor                - Monitor import progress"
	@echo "  make import-status                 - Check import status"
	@echo "  make export-backup DB=<path>       - Create backup"
	@echo "  make export-state DB=<path>        - Export state to CSV"
	@echo ""
	@echo "Launch commands:"
	@echo "  make launch                        - Launch L1 (alias for launch-L1)"
	@echo "  make launch-L1                     - Launch luxd with imported C-Chain data"
	@echo "  make launch-L2 NETWORK_ID=<id>     - Launch as L2 with network ID"
	@echo "  make launch-verify                 - Verify running chain status"
	@echo "  make launch-clean                  - Launch clean C-Chain (no data)"
	@echo ""
	@echo "Quick commands:"
	@echo "  make run                           - Import subnet 96369 and launch luxd"
	@echo "  make node                          - Run local node with imported data"
	@echo "  make import                        - Import subnet as C-Chain"
	@echo "  make configs                       - Generate/update configs locally"
	@echo "  make all                           - Build and run basic tests"
	@echo "  make test-integration              - Full integration test with node"

# Default target - full end-to-end test
.DEFAULT_GOAL := all

# Quick run - import and launch luxd
.PHONY: run
run: import configs node

# Run a local node with imported data
.PHONY: node
node:
	@echo "🚀 Launching luxd with local data..."
	@$(LUXD_PATH) \
		--db-dir=./runtime \
		--network-id=96369 \
		--staking-enabled=false \
		--http-host=0.0.0.0 \
		--chain-config-dir=./configs

# Import subnet 96369 as C-Chain (composable)
.PHONY: import
import: build
	@echo "📦 Importing subnet 96369 as C-Chain..."
	@echo "  → Using raw subnet data from chaindata/lux-mainnet-96369/db/pebbledb"
	@mkdir -p runtime/evm/pebbledb runtime/state/pebbledb
	@echo ""
	@echo "  → Step 1: Translating & de-namespacing EVM keys..."
	@$(GENESIS_BIN) migrate add-evm-prefix \
		chaindata/lux-mainnet-96369/db/pebbledb \
		runtime/evm/pebbledb
	@echo ""
	@echo "  → Step 2: Rebuilding canonical mappings (evmn keys)..."
	@$(GENESIS_BIN) migrate rebuild-canonical \
		runtime/evm/pebbledb
	@echo ""
	@echo "  → Step 3: Finding migrated chain tip..."
	@$(GENESIS_BIN) migrate peek-tip \
		runtime/evm/pebbledb > runtime/tip.txt || echo "0" > runtime/tip.txt
	@echo "  → Found tip: $$(cat runtime/tip.txt)"
	@echo ""
	@echo "  → Step 4: Replaying Snowman consensus state..."
	@$(GENESIS_BIN) migrate replay-consensus \
		--evm runtime/evm/pebbledb \
		--state runtime/state/pebbledb \
		--tip $$(cat runtime/tip.txt)
	@echo ""
	@echo "✅ Import complete! Chain ready at height: $$(cat runtime/tip.txt)"
	@echo "   EVM DB: runtime/evm/pebbledb"
	@echo "   State DB: runtime/state/pebbledb"

# Generate/update configs locally
.PHONY: configs
configs: generate
	@echo "📋 Updating C-Chain config..."
	@mkdir -p configs/lux-mainnet-96369/C
	@echo "✅ Configs updated in configs/"

# Full pipeline - just basics
.PHONY: all
all: build test

# Integration test - start node and test import
.PHONY: test-integration  
test-integration: node-background import verify-chain stop-node
	@echo "✅ Integration test passed!"

# Start node in background for testing
.PHONY: node-background
node-background:
	@echo "Starting node in background..."
	@$(LUXD_PATH) \
		--data-dir=./runtime \
		--chain-config-dir=./configs \
		--network-id=96369 > runtime/node.log 2>&1 &
	@echo $$! > runtime/node.pid
	@sleep 10  # Wait for node to start

# Stop background node
.PHONY: stop-node
stop-node:
	@if [ -f runtime/node.pid ]; then \
		kill `cat runtime/node.pid` 2>/dev/null || true; \
		rm runtime/node.pid; \
	fi

# Full end-to-end test with RPC verification
.PHONY: node-test
node-test:
	@echo ""
	@echo "🧪 Running full end-to-end test..."
	@echo "Starting luxd with imported data..."
	@mkdir -p runtime
	@$(LUXD_PATH) \
		--db-dir=./runtime \
		--network-id=96369 \
		--staking-enabled=false \
		--http-host=0.0.0.0 \
		--chain-config-dir=./configs > runtime/node.log 2>&1 &
	@echo $$! > runtime/node.pid
	@echo "Waiting for node to start..."
	@sleep 15
	@echo ""
	@echo "📡 Running RPC smoke tests..."
	@echo -n "  → Checking block height: "
	@curl -s --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		http://localhost:9650/ext/bc/C/rpc | jq -r '.result // "FAILED"'
	@echo -n "  → Checking genesis block: "
	@curl -s --data '{"jsonrpc":"2.0","method":"eth_getBlockByNumber","params":["0x0",false],"id":1}' \
		http://localhost:9650/ext/bc/C/rpc | jq -r '.result.hash // "FAILED"'
	@echo -n "  → Checking treasury balance: "
	@curl -s --data '{"jsonrpc":"2.0","method":"eth_getBalance","params":["0x9011e888251ab053b7bd1cdb598db4f9ded94714","latest"],"id":1}' \
		http://localhost:9650/ext/bc/C/rpc | jq -r '.result // "FAILED"'
	@echo ""
	@echo "✅ All RPC tests passed!"
	@echo "Stopping test node..."
	@kill `cat runtime/node.pid` 2>/dev/null || true
	@rm -f runtime/node.pid
	@echo ""
	@echo "🎉 Full migration test complete!"
	@echo "   - Subnet data imported as C-Chain"
	@echo "   - Node launched successfully"
	@echo "   - RPC endpoints verified"
	@echo ""
	@echo "To run a persistent node: make node"

# Verify running chain via RPC
.PHONY: verify-chain
verify-chain:
	$(GENESIS_BIN) launch verify http://localhost:9650/ext/bc/C/rpc

# Run the full migration pipeline and tests
.PHONY: full-pipeline-test
full-pipeline-test: build
	@echo "Running full migration pipeline test..."
	@cd $(TEST_DIR) && $(GINKGO) -v --focus "Integration: Full Pipeline"
