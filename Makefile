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

# luxd binary path (used by import-monitor/status if node not running)
# Path to luxd (override with LUXD=... if needed)
LUXD       ?= $(HOME)/work/lux/node/build/luxd

# Default import settings
DATA_DIR   ?= $(HOME)/.luxd-import
NETWORK_ID ?= 96369
RPC_PORT   ?= 9630

# Target network (subdirectory under chaindata/), e.g. lux-mainnet-96369
NETWORK    ?= lux-mainnet-96369

# Default source chaindata path for import-chain-data
SRC        ?= chaindata/$(NETWORK)/db/pebbledb

# Build the unified genesis tool
.PHONY: all
all: build test

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

# Clean build artifacts and temp files
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(TMP_DIR)
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
	@echo "🚀 Importing chain data for network=$(NETWORK) from $(SRC)"
	@$(GENESIS_BIN) import chain-data $(SRC)

.PHONY: import-monitor
import-monitor:
	@echo "📡 Ensuring luxd is running for import monitoring..."
	@if ! pgrep -f 'luxd.*--data-dir=$(DATA_DIR)' >/dev/null 2>&1; then \
		if [ ! -f $(DATA_DIR)/chains/C/genesis.json ]; then \
			echo "⚠️  Data directory '$(DATA_DIR)' is not initialized. Running import-chain-data for network=$(NETWORK)..."; \
			$(MAKE) import-chain-data NETWORK=$(NETWORK); \
		fi; \
		echo "🔄 Starting luxd in normal mode..."; \
		$(LUXD) --network-id=$(NETWORK_ID) --data-dir=$(DATA_DIR) \
		  --http-host=0.0.0.0 --http-port=$(RPC_PORT) \
		  --staking-enabled=false --index-enabled=false \
		  --pruning-enabled=false --state-sync-enabled=false & \
		sleep 5; \
	fi
	@$(GENESIS_BIN) import monitor --rpc-url=http://localhost:$(RPC_PORT)

.PHONY: import-status
import-status:
	@echo "📡 Ensuring luxd is running for status check..."
	@if ! pgrep -f 'luxd.*--data-dir=$(DATA_DIR)' >/dev/null 2>&1; then \
		if [ ! -f $(DATA_DIR)/chains/C/genesis.json ]; then \
			echo "⚠️  Data directory '$(DATA_DIR)' is not initialized. Running import-chain-data for network=$(NETWORK)..."; \
			$(MAKE) import-chain-data NETWORK=$(NETWORK); \
		fi; \
		echo "🔄 Starting luxd in normal mode..."; \
		$(LUXD) --network-id=$(NETWORK_ID) --data-dir=$(DATA_DIR) \
		  --http-host=0.0.0.0 --http-port=$(RPC_PORT) \
		  --staking-enabled=false --index-enabled=false \
		  --pruning-enabled=false --state-sync-enabled=false & \
		sleep 5; \
	fi
	@$(GENESIS_BIN) import status --rpc-url=http://localhost:$(RPC_PORT)

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

# Help
.PHONY: help
help:
	@echo "Lux Genesis Migration Makefile"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build the unified genesis tool"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make clean-all          - Deep clean including test outputs"
	@echo ""
	@echo "Test targets:"
	@echo "  make test               - Run all tests"
	@echo "  make test filter=X      - Run tests matching filter X"
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

# Default target
.DEFAULT_GOAL := help