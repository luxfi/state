# Lux Genesis Builder Makefile
# ============================
# A composable pipeline for blockchain data extraction, migration, and genesis generation

.PHONY: help all clean build test

# Default target shows help
help:
	@echo "Lux Genesis Builder - Composable Pipeline for Blockchain Data"
	@echo "============================================================="
	@echo ""
	@echo "QUICK START:"
	@echo "  make pipeline NETWORK=zoo         # Run complete pipeline for ZOO"
	@echo "  make pipeline NETWORK=lux         # Run complete pipeline for LUX"
	@echo "  make extract NETWORK=zoo          # Extract specific network"
	@echo "  make genesis NETWORK=zoo          # Build genesis for specific network"
	@echo ""
	@echo "EXTRACTION COMMANDS:"
	@echo "  make extract-chain CHAIN=<name>     # Extract any chain data"
	@echo "  make extract-lux                    # Extract LUX mainnet (96369)"
	@echo "  make extract-zoo                    # Extract ZOO mainnet (200200)"
	@echo "  make extract-spc                    # Extract SPC mainnet (36911)"
	@echo "  make extract-all                    # Extract all networks"
	@echo ""
	@echo "SCANNING COMMANDS (External Chains):"
	@echo "  make scan-bsc-zoo                   # Scan BSC for ZOO burns + eggs"
	@echo "  make scan-eth-nft                   # Scan ETH for Lux Genesis NFTs"
	@echo "  make scan-burns CHAIN=bsc TOKEN=0x... # Scan any token burns"
	@echo "  make scan-nfts CHAIN=eth NFT=0x...    # Scan any NFT holders"
	@echo ""
	@echo "ANALYSIS COMMANDS:"
	@echo "  make analyze-zoo                    # Analyze ZOO token distribution"
	@echo "  make analyze-spc                    # Analyze SPC token distribution"
	@echo "  make cross-reference                # Cross-reference all chains"
	@echo "  make validate-supply                # Validate token supplies"
	@echo ""
	@echo "GENESIS BUILDING:"
	@echo "  make genesis-lux                    # Build LUX genesis"
	@echo "  make genesis-zoo                    # Build ZOO genesis with BSC data"
	@echo "  make genesis-spc                    # Build SPC genesis (bootstrap)"
	@echo "  make genesis-all                    # Build all genesis files"
	@echo ""
	@echo "DEPLOYMENT:"
	@echo "  make deploy-lux                     # Deploy LUX network"
	@echo "  make deploy-l2 L2=zoo               # Deploy L2 (zoo/spc/hanzo)"
	@echo "  make deploy-all                     # Deploy all networks"
	@echo ""
	@echo "UTILITIES:"
	@echo "  make clean                          # Clean output directories"
	@echo "  make test                           # Run all tests"
	@echo "  make validate                       # Validate all genesis files"
	@echo "  make backup                         # Backup current genesis"
	@echo ""
	@echo "PIPELINES (Common Workflows):"
	@echo "  make pipeline-zoo                   # Complete ZOO migration pipeline"
	@echo "  make pipeline-fresh                 # Fresh network from scratch"
	@echo "  make pipeline-migrate               # Migrate existing networks"
	@echo ""
	@echo "ENVIRONMENT VARIABLES:"
	@echo "  BSC_RPC          BSC RPC endpoint (default: public)"
	@echo "  ETH_RPC          Ethereum RPC endpoint"
	@echo "  OUTPUT_DIR       Output directory (default: ./output)"
	@echo "  CHAIN_ID         Override chain ID"
	@echo "  VERBOSE          Enable verbose output"

# Configuration
OUTPUT_DIR ?= ./output
DATA_DIR ?= ./chaindata
EXPORT_DIR ?= $(OUTPUT_DIR)/exports
GENESIS_DIR ?= $(OUTPUT_DIR)/genesis
ANALYSIS_DIR ?= $(OUTPUT_DIR)/analysis

# Tools
TELEPORT := ./bin/teleport
ARCHEOLOGY := ./bin/archeology

# Networks
LUX_MAINNET := 96369
ZOO_MAINNET := 200200
SPC_MAINNET := 36911
HANZO_MAINNET := 36963

# Contract Addresses
ZOO_TOKEN_BSC := 0x0a6045b79151d0a54dbd5227082445750a023af2
EGG_NFT_BSC := 0x5bb68cf06289d54efde25155c88003be685356a8
LUX_NFT_ETH := 0x31e0f919c67cedd2bc3e294340dc900735810311
DEAD_ADDRESS := 0x000000000000000000000000000000000000dEaD

# RPC Endpoints
BSC_RPC ?= https://bsc-dataseed.binance.org/
ETH_RPC ?= https://eth.llamarpc.com

# Create output directories
$(OUTPUT_DIR) $(EXPORT_DIR) $(GENESIS_DIR) $(ANALYSIS_DIR):
	@mkdir -p $@

# Clean outputs
clean:
	@echo "🧹 Cleaning output directories..."
	@rm -rf $(OUTPUT_DIR)/*
	@echo "✅ Clean complete"

# Build tools if needed
build-tools:
	@echo "🔨 Building tools..."
	@cd .. && make build-teleport build-archeology
	@echo "✅ Tools built"

# ============ EXTRACTION COMMANDS ============

# Dynamic extraction based on NETWORK env var
extract:
ifdef NETWORK
	@$(MAKE) extract-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make extract NETWORK=zoo"
	@exit 1
endif

extract-lux: $(EXPORT_DIR)
	@echo "📦 Extracting LUX mainnet ($(LUX_MAINNET))..."
	@$(ARCHEOLOGY) extract \
		--source $(DATA_DIR)/lux-mainnet-$(LUX_MAINNET)/db/pebbledb \
		--destination $(EXPORT_DIR)/lux-$(LUX_MAINNET) \
		--chain-id $(LUX_MAINNET) \
		--include-state
	@echo "✅ LUX extraction complete"

extract-zoo: $(EXPORT_DIR)
	@echo "📦 Extracting ZOO mainnet ($(ZOO_MAINNET))..."
	@$(ARCHEOLOGY) extract \
		--source $(DATA_DIR)/zoo-mainnet-$(ZOO_MAINNET)/db/pebbledb \
		--destination $(EXPORT_DIR)/zoo-$(ZOO_MAINNET) \
		--chain-id $(ZOO_MAINNET) \
		--include-state
	@echo "✅ ZOO extraction complete"

extract-spc: $(EXPORT_DIR)
	@echo "📦 Extracting SPC mainnet ($(SPC_MAINNET))..."
	@$(ARCHEOLOGY) extract \
		--source $(DATA_DIR)/spc-mainnet-$(SPC_MAINNET)/db/pebbledb \
		--destination $(EXPORT_DIR)/spc-$(SPC_MAINNET) \
		--chain-id $(SPC_MAINNET) \
		--include-state
	@echo "✅ SPC extraction complete"

extract-all: extract-lux extract-zoo extract-spc
	@echo "✅ All extractions complete"

# Generic chain extraction
extract-chain: $(EXPORT_DIR)
ifndef CHAIN
	$(error CHAIN is not set. Usage: make extract-chain CHAIN=<name>)
endif
	@echo "📦 Extracting $(CHAIN)..."
	@$(ARCHEOLOGY) extract \
		--source $(DATA_DIR)/$(CHAIN)/db/pebbledb \
		--destination $(EXPORT_DIR)/$(CHAIN) \
		--include-state
	@echo "✅ $(CHAIN) extraction complete"

# ============ SCANNING COMMANDS ============

# Dynamic scanning based on NETWORK env var
scan:
ifdef NETWORK
	@$(MAKE) scan-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make scan NETWORK=zoo"
	@exit 1
endif

# Network-specific scans
scan-zoo: scan-bsc-zoo
scan-lux: scan-eth-nft
scan-spc:
	@echo "✅ SPC has no external chain scanning requirements"

scan-bsc-zoo: $(EXPORT_DIR)
	@echo "🔍 Scanning BSC for ZOO migration data..."
	@echo "  - Token burns to $(DEAD_ADDRESS)"
	@echo "  - EGG NFT holders (4.2M ZOO each)"
	@$(TELEPORT) zoo-migrate \
		--rpc $(BSC_RPC) \
		--include-burns \
		--include-egg-nfts \
		--output $(EXPORT_DIR)/zoo-bsc-migration.json
	@echo "✅ BSC ZOO scan complete"

scan-eth-nft: $(EXPORT_DIR)
	@echo "🔍 Scanning Ethereum for Lux Genesis NFTs..."
	@echo "  - NFT holders get validator rights"
	@echo "  - Contract: $(LUX_NFT_ETH)"
	@$(TELEPORT) scan-nft-holders \
		--chain ethereum \
		--rpc $(ETH_RPC) \
		--contract $(LUX_NFT_ETH) \
		--output $(EXPORT_DIR)/lux-nft-holders.csv
	@echo "✅ ETH NFT scan complete"

scan-burns: $(EXPORT_DIR)
ifndef TOKEN
	$(error TOKEN is not set. Usage: make scan-burns CHAIN=bsc TOKEN=0x...)
endif
	@echo "🔍 Scanning $(CHAIN) for token burns..."
	@$(TELEPORT) scan-token-burns \
		--rpc $($(shell echo $(CHAIN) | tr a-z A-Z)_RPC) \
		--token $(TOKEN) \
		--burn-address $(DEAD_ADDRESS) \
		--output $(EXPORT_DIR)/$(CHAIN)-burns.csv
	@echo "✅ Burn scan complete"

# ============ ANALYSIS COMMANDS ============

# Dynamic analysis based on NETWORK env var
analyze:
ifdef NETWORK
	@$(MAKE) analyze-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make analyze NETWORK=zoo"
	@exit 1
endif

analyze-zoo: $(ANALYSIS_DIR)
	@echo "📊 Analyzing ZOO token distribution..."
	@$(TELEPORT) analyze-distribution \
		--chain zoo \
		--data $(EXPORT_DIR)/zoo-$(ZOO_MAINNET) \
		--bsc-data $(EXPORT_DIR)/zoo-bsc-migration.json \
		--output $(ANALYSIS_DIR)/zoo-analysis.json
	@echo "✅ ZOO analysis complete"

analyze-lux: $(ANALYSIS_DIR)
	@echo "📊 Analyzing LUX token distribution..."
	@$(TELEPORT) analyze-distribution \
		--chain lux \
		--data $(EXPORT_DIR)/lux-$(LUX_MAINNET) \
		--nft-data $(EXPORT_DIR)/lux-nft-holders.csv \
		--output $(ANALYSIS_DIR)/lux-analysis.json
	@echo "✅ LUX analysis complete"

analyze-spc: $(ANALYSIS_DIR)
	@echo "📊 Analyzing SPC token distribution..."
	@$(TELEPORT) analyze-distribution \
		--chain spc \
		--data $(EXPORT_DIR)/spc-$(SPC_MAINNET) \
		--output $(ANALYSIS_DIR)/spc-analysis.json
	@echo "✅ SPC analysis complete"

validate-supply: analyze-zoo analyze-spc
	@echo "✓ Validating token supplies..."
	@$(TELEPORT) validate-supplies \
		--zoo $(ANALYSIS_DIR)/zoo-analysis.json \
		--spc $(ANALYSIS_DIR)/spc-analysis.json
	@echo "✅ Supply validation complete"

# ============ GENESIS BUILDING ============

# Dynamic genesis based on NETWORK env var
genesis:
ifdef NETWORK
	@$(MAKE) genesis-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make genesis NETWORK=zoo"
	@exit 1
endif

genesis-lux: $(GENESIS_DIR) extract-lux scan-eth-nft
	@echo "🏗️  Building LUX genesis..."
	@$(TELEPORT) build-genesis \
		--chain lux \
		--chain-id $(LUX_MAINNET) \
		--data $(EXPORT_DIR)/lux-$(LUX_MAINNET) \
		--nft-holders $(EXPORT_DIR)/lux-nft-holders.csv \
		--output $(GENESIS_DIR)/lux-mainnet-genesis.json
	@echo "✅ LUX genesis complete"

genesis-zoo: $(GENESIS_DIR) extract-zoo scan-bsc-zoo
	@echo "🏗️  Building ZOO genesis with BSC migration..."
	@$(TELEPORT) build-genesis \
		--chain zoo \
		--chain-id $(ZOO_MAINNET) \
		--data $(EXPORT_DIR)/zoo-$(ZOO_MAINNET) \
		--migration-data $(EXPORT_DIR)/zoo-bsc-migration.json \
		--output $(GENESIS_DIR)/zoo-mainnet-genesis.json
	@echo "✅ ZOO genesis complete"

genesis-spc: $(GENESIS_DIR)
	@echo "🏗️  Building SPC genesis (bootstrap)..."
	@$(TELEPORT) build-genesis \
		--chain spc \
		--chain-id $(SPC_MAINNET) \
		--bootstrap \
		--supply 10000000 \
		--output $(GENESIS_DIR)/spc-mainnet-genesis.json
	@echo "✅ SPC genesis complete"

genesis-all: genesis-lux genesis-zoo genesis-spc
	@echo "✅ All genesis files built"

# ============ DEPLOYMENT ============

# Dynamic deployment based on NETWORK env var
deploy:
ifdef NETWORK
	@$(MAKE) deploy-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make deploy NETWORK=zoo"
	@exit 1
endif

deploy-lux: genesis-lux
	@echo "🚀 Deploying LUX network..."
	@lux network create lux-mainnet \
		--genesis $(GENESIS_DIR)/lux-mainnet-genesis.json \
		--evm
	@echo "✅ LUX deployment complete"

deploy-zoo: genesis-zoo
	@echo "🚀 Deploying ZOO L2..."
	@lux subnet create zoo-mainnet \
		--genesis $(GENESIS_DIR)/zoo-mainnet-genesis.json \
		--evm
	@echo "✅ ZOO deployment complete"

deploy-spc: genesis-spc
	@echo "🚀 Deploying SPC L2..."
	@lux subnet create spc-mainnet \
		--genesis $(GENESIS_DIR)/spc-mainnet-genesis.json \
		--evm
	@echo "✅ SPC deployment complete"

deploy-l2: genesis-$(L2)
ifndef L2
	$(error L2 is not set. Usage: make deploy-l2 L2=zoo)
endif
	@echo "🚀 Deploying $(L2) L2..."
	@lux subnet create $(L2)-mainnet \
		--genesis $(GENESIS_DIR)/$(L2)-mainnet-genesis.json \
		--evm
	@echo "✅ $(L2) deployment complete"

deploy-all: deploy-lux
	@$(MAKE) deploy-l2 L2=zoo
	@$(MAKE) deploy-l2 L2=spc
	@echo "✅ All networks deployed"

# ============ PIPELINES ============

# Dynamic pipeline based on NETWORK env var
pipeline:
ifdef NETWORK
	@echo "🔄 Running pipeline for $(NETWORK)..."
	@$(MAKE) pipeline-$(NETWORK)
else
	@echo "❌ Please specify NETWORK. Usage: make pipeline NETWORK=zoo"
	@echo "   Available networks: lux, zoo, spc"
	@exit 1
endif

# Network-specific pipelines
pipeline-zoo:
	@echo "🔄 Running complete ZOO migration pipeline..."
	@$(MAKE) extract-zoo
	@$(MAKE) scan-bsc-zoo
	@$(MAKE) analyze-zoo
	@$(MAKE) genesis-zoo
	@echo "✅ ZOO pipeline complete!"

pipeline-lux:
	@echo "🔄 Running LUX network pipeline..."
	@$(MAKE) extract-lux
	@$(MAKE) scan-eth-nft
	@$(MAKE) genesis-lux
	@echo "✅ LUX pipeline complete!"

pipeline-spc:
	@echo "🔄 Running SPC network pipeline..."
	@$(MAKE) extract-spc
	@$(MAKE) analyze-spc
	@$(MAKE) genesis-spc
	@echo "✅ SPC pipeline complete!"

pipeline-fresh:
	@echo "🔄 Building fresh network from scratch..."
	@$(MAKE) clean
	@$(MAKE) build-tools
	@$(MAKE) extract-all
	@$(MAKE) genesis-all
	@$(MAKE) validate
	@echo "✅ Fresh network ready!"

pipeline-migrate:
	@echo "🔄 Running full migration pipeline..."
	@$(MAKE) extract-all
	@$(MAKE) scan-bsc-zoo
	@$(MAKE) scan-eth-nft
	@$(MAKE) analyze-zoo analyze-spc
	@$(MAKE) genesis-all
	@$(MAKE) validate
	@echo "✅ Migration complete!"

# ============ UTILITIES ============

validate: $(GENESIS_DIR)
	@echo "🔍 Validating all genesis files..."
	@for genesis in $(GENESIS_DIR)/*.json; do \
		echo "  Checking $$genesis..."; \
		$(ARCHEOLOGY) validate-genesis --file $$genesis || exit 1; \
	done
	@echo "✅ All genesis files valid"

backup:
	@echo "💾 Backing up genesis files..."
	@mkdir -p backups/$(shell date +%Y%m%d_%H%M%S)
	@cp -r $(GENESIS_DIR)/* backups/$(shell date +%Y%m%d_%H%M%S)/
	@echo "✅ Backup complete"

test:
	@echo "🧪 Running tests..."
	@cd .. && go test ./...
	@echo "✅ All tests passed"

# Aliases for common operations
zoo: pipeline-zoo
fresh: pipeline-fresh
migrate: pipeline-migrate

.DEFAULT_GOAL := help