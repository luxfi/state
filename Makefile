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

# Build all tools
all: build test

# Build migration tools
build: clean
	@echo "Building migration tools..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/add-evm-prefix-to-blocks add-evm-prefix-to-blocks.go
	$(GOBUILD) -o $(BIN_DIR)/check-head-pointers check-head-pointers.go
	$(GOBUILD) -o $(BIN_DIR)/create-synthetic-blockchain create-synthetic-blockchain.go
	$(GOBUILD) -o $(BIN_DIR)/replay-consensus-pebble replay-consensus-pebble.go
	$(GOBUILD) -o $(BIN_DIR)/analyze-key-structure analyze-key-structure.go
	$(GOBUILD) -o $(BIN_DIR)/check-canonical-keys check-canonical-keys.go
	$(GOBUILD) -o $(BIN_DIR)/find-canonical-mappings find-canonical-mappings.go
	$(GOBUILD) -o $(BIN_DIR)/dump-sample-keys dump-sample-keys.go
	@echo "Build complete!"

# Run Ginkgo tests with optional filter
test: build
	@echo "Setting up test environment..."
	@mkdir -p $(TMP_DIR)
	@mkdir -p $(TEST_DIR)
	@if [ -f migration_test.go ]; then mv migration_test.go $(TEST_DIR)/; fi
	@echo "Running Ginkgo tests..."
ifdef filter
	@echo "Running tests matching: $(filter)"
	@cd $(TEST_DIR) && $(GINKGO) run -v --focus="$(filter)" --fail-fast
else
	@echo "Running all tests..."
	@cd $(TEST_DIR) && $(GINKGO) run -v --randomize-all --randomize-suites
endif

# Run specific test phases
test-step1:
	@$(MAKE) test filter="Step 1"

test-step2:
	@$(MAKE) test filter="Step 2"

test-step3:
	@$(MAKE) test filter="Step 3"

test-step4:
	@$(MAKE) test filter="Step 4"

test-step5:
	@$(MAKE) test filter="Step 5"

test-integration:
	@$(MAKE) test filter="Integration"

test-performance:
	@$(MAKE) test filter="Performance"

test-edge-cases:
	@$(MAKE) test filter="Edge Cases"

# Run migration pipeline step by step
migrate-step-by-step: build
	@echo "Running migration pipeline step by step..."
	@$(MAKE) test-step1
	@echo "\nPress Enter to continue to Step 2..."
	@read dummy
	@$(MAKE) test-step2
	@echo "\nPress Enter to continue to Step 3..."
	@read dummy
	@$(MAKE) test-step3
	@echo "\nPress Enter to continue to Step 4..."
	@read dummy
	@$(MAKE) test-step4
	@echo "\nPress Enter to continue to Step 5..."
	@read dummy
	@$(MAKE) test-step5
	@echo "\nPress Enter to run integration test..."
	@read dummy
	@$(MAKE) test-integration

# Install Ginkgo if not present
install-ginkgo:
	@echo "Installing Ginkgo test framework..."
	@$(GOGET) github.com/onsi/ginkgo/v2/ginkgo
	@$(GOGET) github.com/onsi/gomega/...

# Clean build artifacts and temp files
clean:
	@echo "Cleaning..."
	@rm -rf $(BIN_DIR)
	@rm -rf $(TMP_DIR)
	@find . -name "*.test" -delete
	@find . -name "*.out" -delete
	@echo "Clean complete!"

# Clean everything including test outputs
clean-all: clean
	@rm -rf $(TEST_DIR)/$(TMP_DIR)
	@echo "Deep clean complete!"

# Run a specific migration tool
run-prefix-migration:
	@if [ -z "$(src)" ] || [ -z "$(dst)" ]; then \
		echo "Usage: make run-prefix-migration src=<source-db> dst=<destination-db>"; \
		exit 1; \
	fi
	@$(BIN_DIR)/add-evm-prefix-to-blocks $(src) $(dst)

run-synthetic-blockchain:
	@if [ -z "$(state)" ] || [ -z "$(output)" ]; then \
		echo "Usage: make run-synthetic-blockchain state=<state-db> output=<output-db> [blocks=1082780] [chainid=96369]"; \
		exit 1; \
	fi
	@$(BIN_DIR)/create-synthetic-blockchain --state $(state) --output $(output) --blocks $${blocks:-1082780} --chainid $${chainid:-96369}

run-consensus-replay:
	@if [ -z "$(evm)" ] || [ -z "$(state)" ]; then \
		echo "Usage: make run-consensus-replay evm=<evm-db> state=<state-db> [tip=1082780] [batch=10000]"; \
		exit 1; \
	fi
	@$(BIN_DIR)/replay-consensus-pebble --evm $(evm) --state $(state) --tip $${tip:-1082780} --batch $${batch:-10000}

# Check database structure
check-db:
	@if [ -z "$(db)" ]; then \
		echo "Usage: make check-db db=<database-path>"; \
		exit 1; \
	fi
	@echo "Checking head pointers..."
	@$(BIN_DIR)/check-head-pointers $(db)
	@echo "\nAnalyzing key structure..."
	@$(BIN_DIR)/analyze-key-structure $(db) | head -50
	@echo "\nChecking canonical keys..."
	@$(BIN_DIR)/check-canonical-keys $(db)

# Help
help:
	@echo "Lux Genesis Migration Makefile"
	@echo ""
	@echo "Main commands:"
	@echo "  make all                 - Build tools and run all tests"
	@echo "  make build               - Build all migration tools"
	@echo "  make test                - Run all Ginkgo tests"
	@echo "  make test filter=<text>  - Run tests matching filter"
	@echo ""
	@echo "Step-by-step testing:"
	@echo "  make test-step1          - Test subnet data creation"
	@echo "  make test-step2          - Test EVM prefix migration"
	@echo "  make test-step3          - Test synthetic blockchain creation"
	@echo "  make test-step4          - Test consensus state generation"
	@echo "  make test-step5          - Test verification tools"
	@echo "  make test-integration    - Test full pipeline"
	@echo "  make test-performance    - Run performance benchmarks"
	@echo "  make test-edge-cases     - Test error handling"
	@echo "  make migrate-step-by-step - Interactive step-by-step migration"
	@echo ""
	@echo "Migration tools:"
	@echo "  make run-prefix-migration src=<db> dst=<db>"
	@echo "  make run-synthetic-blockchain state=<db> output=<db> [blocks=N] [chainid=N]"
	@echo "  make run-consensus-replay evm=<db> state=<db> [tip=N] [batch=N]"
	@echo "  make check-db db=<path>  - Analyze database structure"
	@echo ""
	@echo "Other commands:"
	@echo "  make install-ginkgo      - Install Ginkgo test framework"
	@echo "  make clean               - Clean build artifacts"
	@echo "  make clean-all           - Clean everything"

.PHONY: all build test clean clean-all help install-ginkgo \
        test-step1 test-step2 test-step3 test-step4 test-step5 \
        test-integration test-performance test-edge-cases \
        migrate-step-by-step run-prefix-migration \
        run-synthetic-blockchain run-consensus-replay check-db