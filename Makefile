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
	@echo "  make up                           # Launch full historic genesis network"
	@echo "  make up NETWORK=<name>            # Launch a single network (e.g., zoo, spc, hanzo)"
	@echo ""
	@echo "COMMON WORKFLOWS:"
	@echo "  make build-primary                # Build EVM and Node for clean genesis"
	@echo "  make devnet                       # Launch 3-node devnet with clean genesis"
	@echo "  make devnet-test                  # Test devnet health and features"
	@echo "  make diag                         # Quick diagnose historic database"
	@echo "  make migrate-complete             # Full migration workflow"
	@echo "  make genesis-help                 # Show all genesis commands"
	@echo "  ./genesis diagnose /path/to/db    # Direct CLI usage"
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
	@echo "  make genesis-all                    # Build all genesis files

# Validator Management
validators-generate: build-genesis
	@echo "🔑 Generating 11 new validator keys..."
	@./bin/genesis validators generate \
		--offsets 0,1,2,3,4,5,6,7,8,9,10 \
		--save-keys configs/mainnet/validators.json
	@echo "✅ 11 new validators generated and saved to configs/mainnet/validators.json"
	@echo ""

# Network Deployment Commands
deploy: deploy-network

deploy-network:
ifndef NETWORK
	@echo "❌ Please specify NETWORK=mainnet|testnet|local"
	@echo "Usage: make deploy NETWORK=mainnet"
	@exit 1
endif
ifeq ($(NETWORK),mainnet)
	@$(MAKE) deploy-mainnet
else ifeq ($(NETWORK),testnet)
	@$(MAKE) deploy-testnet
else ifeq ($(NETWORK),local)
	@$(MAKE) deploy-local
else
	@echo "❌ Invalid network: $(NETWORK)"
	@exit 1
endif

# Deploy mainnet (21 nodes with historic data)
deploy-mainnet: build-node
	@echo "🚀 Deploying Lux Mainnet (21 nodes)"
	@# Generate validators if not exists
	@if [ ! -f "$(HOME)/.luxd/keys/mainnet/validators.json" ]; then \
		echo "Generating mainnet validators..."; \
		$(HOME)/.luxd/genkeys mainnet; \
	fi
	@# Create genesis if not exists
	@if [ ! -f "$(HOME)/.luxd/genesis/mainnet/genesis.json" ]; then \
		echo "Creating mainnet genesis..."; \
		$(HOME)/.luxd/create_genesis_with_validators.py mainnet; \
	fi
	@# Launch 21 nodes
	@$(MAKE) launch-mainnet-21

# Deploy testnet (11 nodes with faster consensus)
deploy-testnet: build-node
	@echo "🚀 Deploying Lux Testnet (11 nodes)"
	@# Generate validators if not exists
	@if [ ! -f "$(HOME)/.luxd/keys/testnet/validators.json" ]; then \
		echo "Generating testnet validators..."; \
		$(HOME)/.luxd/genkeys testnet --count 11; \
	fi
	@# Create genesis if not exists
	@if [ ! -f "$(HOME)/.luxd/genesis/testnet/genesis.json" ]; then \
		echo "Creating testnet genesis..."; \
		$(HOME)/.luxd/create_genesis_with_validators.py testnet; \
	fi
	@# Launch 11 nodes with testnet config
	@$(MAKE) launch-testnet-11

# Deploy local development network (5 nodes)
deploy-local: build-node
	@echo "🚀 Deploying Local Development Network (5 nodes)"
	@# Generate validators if not exists
	@if [ ! -f "$(HOME)/.luxd/keys/local/validators.json" ]; then \
		echo "Generating local validators..."; \
		$(HOME)/.luxd/genkeys local; \
	fi
	@# Create genesis if not exists
	@if [ ! -f "$(HOME)/.luxd/genesis/local/genesis.json" ]; then \
		echo "Creating local genesis..."; \
		$(HOME)/.luxd/create_genesis_with_validators.py local; \
	fi
	@# Launch 5 nodes
	@$(MAKE) launch-local-5

# Launch mainnet with 21 nodes
launch-mainnet-21: kill-node
	@echo "🚀 Starting 21-node mainnet..."
	@$(MAKE) start-nodes NETWORK=mainnet NUM_NODES=21 BASE_PORT=9650 \
		CONSENSUS_K=21 CONSENSUS_ALPHA_PREF=13 CONSENSUS_ALPHA_CONF=18 CONSENSUS_BETA=8 \
		CONCURRENT_REPOLLS=8 OPTIMAL_PROCESSING=10 MAX_PROCESSING_TIME=9630000000

# Launch testnet with 11 nodes and faster consensus
launch-testnet-11: kill-node
	@echo "🚀 Starting 11-node testnet..."
	@$(MAKE) start-nodes NETWORK=testnet NUM_NODES=11 BASE_PORT=9680 \
		CONSENSUS_K=11 CONSENSUS_ALPHA_PREF=8 CONSENSUS_ALPHA_CONF=9 CONSENSUS_BETA=10 \
		CONCURRENT_REPOLLS=10 OPTIMAL_PROCESSING=10 MAX_PROCESSING_TIME=6300000000

# Launch local with 5 nodes
launch-local-5: kill-node
	@echo "🚀 Starting 5-node local network..."
	@$(MAKE) start-nodes NETWORK=local NUM_NODES=5 BASE_PORT=9710 \
		CONSENSUS_K=5 CONSENSUS_ALPHA_PREF=3 CONSENSUS_ALPHA_CONF=4 CONSENSUS_BETA=5 \
		CONCURRENT_REPOLLS=5 OPTIMAL_PROCESSING=5 MAX_PROCESSING_TIME=3690000000

# Generic node launcher
start-nodes:
	@echo "Starting $(NUM_NODES) nodes for $(NETWORK)..."
	@mkdir -p $(HOME)/.luxd/networks/$(NETWORK)
	@# Start bootstrap node
	@echo "Starting bootstrap node..."
	@$(MAKE) start-single-node NODE_ID=1 NETWORK=$(NETWORK) \
		HTTP_PORT=$(BASE_PORT) STAKING_PORT=$$(($(BASE_PORT)+1000)) \
		BOOTSTRAP_IPS="" BOOTSTRAP_IDS=""
	@sleep 5
	@# Get bootstrap info
	@BOOTSTRAP_IP="127.0.0.1:$$(($(BASE_PORT)+1000))"
	@BOOTSTRAP_ID=$$(cat $(HOME)/.luxd/keys/$(NETWORK)/validators.json | jq -r '.validators[0].nodeId')
	@# Start remaining nodes
	@for i in $$(seq 2 $(NUM_NODES)); do \
		echo "Starting node $$i..."; \
		$(MAKE) start-single-node NODE_ID=$$i NETWORK=$(NETWORK) \
			HTTP_PORT=$$(($(BASE_PORT)+$$i-1)) STAKING_PORT=$$(($(BASE_PORT)+1000+$$i)) \
			BOOTSTRAP_IPS=$$BOOTSTRAP_IP BOOTSTRAP_IDS=$$BOOTSTRAP_ID; \
		sleep 1; \
	done
	@echo "✅ All $(NUM_NODES) nodes started for $(NETWORK)"
	@echo "Primary RPC: http://localhost:$(BASE_PORT)"

# Start a single node
start-single-node:
	@NODE_DIR="$(HOME)/.luxd/networks/$(NETWORK)/node$(NODE_ID)"
	@mkdir -p $$NODE_DIR/logs
	@# Copy C-Chain data for mainnet/testnet
	@if [ "$(NETWORK)" = "mainnet" ] && [ $(NODE_ID) -eq 1 ]; then \
		if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then \
			echo "Copying mainnet C-Chain data..."; \
			mkdir -p $$NODE_DIR/db/C/db; \
			cp -r chaindata/lux-mainnet-96369/db/pebbledb $$NODE_DIR/db/C/db/; \
		fi; \
	fi
	@if [ "$(NETWORK)" = "testnet" ] && [ $(NODE_ID) -eq 1 ]; then \
		if [ -d "chaindata/lux-testnet-96368/db/pebbledb" ]; then \
			echo "Copying testnet C-Chain data..."; \
			mkdir -p $$NODE_DIR/db/C/db; \
			cp -r chaindata/lux-testnet-96368/db/pebbledb $$NODE_DIR/db/C/db/; \
		fi; \
	fi
	@# Determine network ID
	@if [ "$(NETWORK)" = "mainnet" ]; then NETWORK_ID=96369; \
	elif [ "$(NETWORK)" = "testnet" ]; then NETWORK_ID=96368; \
	else NETWORK_ID=96370; fi
	@# Start node
	@nohup ../node/build/luxd \
		--network-id=$$NETWORK_ID \
		--data-dir="$$NODE_DIR" \
		--staking-tls-cert-file="$(HOME)/.luxd/keys/$(NETWORK)/staker$(NODE_ID).crt" \
		--staking-tls-key-file="$(HOME)/.luxd/keys/$(NETWORK)/staker$(NODE_ID).key" \
		--staking-signer-key-file="$(HOME)/.luxd/keys/$(NETWORK)/signer$(NODE_ID).key" \
		--http-host=0.0.0.0 \
		--http-port=$(HTTP_PORT) \
		--staking-port=$(STAKING_PORT) \
		--public-ip=127.0.0.1 \
		--sybil-protection-enabled=false \
		--consensus-sample-size=$(CONSENSUS_K) \
		--consensus-quorum-size=$(CONSENSUS_ALPHA_CONF) \
		--consensus-k=$(CONSENSUS_K) \
		--consensus-alpha-preference=$(CONSENSUS_ALPHA_PREF) \
		--consensus-alpha-confidence=$(CONSENSUS_ALPHA_CONF) \
		--consensus-beta=$(CONSENSUS_BETA) \
		--consensus-concurrent-repolls=$(CONCURRENT_REPOLLS) \
		--consensus-optimal-processing=$(OPTIMAL_PROCESSING) \
		--consensus-max-item-processing-time=$(MAX_PROCESSING_TIME) \
		--api-admin-enabled \
		--api-keystore-enabled \
		--api-metrics-enabled \
		--log-level=info \
		--log-dir="$$NODE_DIR/logs" \
		--genesis-file="$(HOME)/.luxd/genesis/$(NETWORK)/genesis.json" \
		$(if $(BOOTSTRAP_IPS),--bootstrap-ips="$(BOOTSTRAP_IPS)",) \
		$(if $(BOOTSTRAP_IDS),--bootstrap-ids="$(BOOTSTRAP_IDS)",) \
		> "$$NODE_DIR/node.log" 2>&1 &
	@echo $$! > "$$NODE_DIR/node.pid"
	@echo "Node $(NODE_ID) started (PID: $$(cat $$NODE_DIR/node.pid))"

# Stop networks
stop:
ifndef NETWORK
	@echo "❌ Please specify NETWORK=mainnet|testnet|local"
	@exit 1
endif
	@echo "🛑 Stopping $(NETWORK) network..."
	@NETWORK_DIR="$(HOME)/.luxd/networks/$(NETWORK)"
	@if [ -d "$$NETWORK_DIR" ]; then \
		for pidfile in $$NETWORK_DIR/node*/node.pid; do \
			if [ -f "$$pidfile" ]; then \
				PID=$$(cat $$pidfile); \
				if kill -0 $$PID 2>/dev/null; then \
					echo "Stopping node (PID: $$PID)"; \
					kill -TERM $$PID; \
				fi; \
				rm -f $$pidfile; \
			fi; \
		done; \
		echo "✅ $(NETWORK) network stopped"; \
	else \
		echo "ℹ️  No $(NETWORK) network running"; \
	fi

# Network status
network-status:
	@echo "📊 Network Status"
	@echo "================"
	@for network in mainnet testnet local; do \
		echo ""; \
		echo "$$network:"; \
		NETWORK_DIR="$(HOME)/.luxd/networks/$$network"; \
		if [ -d "$$NETWORK_DIR" ]; then \
			RUNNING=0; \
			TOTAL=$$(ls -d $$NETWORK_DIR/node* 2>/dev/null | wc -l); \
			for pidfile in $$NETWORK_DIR/node*/node.pid; do \
				if [ -f "$$pidfile" ]; then \
					PID=$$(cat $$pidfile); \
					if kill -0 $$PID 2>/dev/null; then \
						RUNNING=$$((RUNNING + 1)); \
					fi; \
				fi; \
			done; \
			echo "  Running: $$RUNNING/$$TOTAL nodes"; \
		else \
			echo "  Not deployed"; \
		fi; \
	done

	@echo "MIGRATION COMMANDS:"
	@echo "  make migrate                        # Migrate historic chain data"
	@echo "  make migrate-genesis                # Extract genesis from historic data"
	@echo "  make migrate-complete               # Run complete migration workflow"
	@echo "  make migrate-dry-run                # Preview migration without changes"
	@echo ""
	@echo "DIAGNOSTIC COMMANDS:"
	@echo "  make diagnose DB_PATH=/path/to/db  # Diagnose any database"
	@echo "  make diagnose-historic              # Diagnose historic LUX database"
	@echo "  make count-keys DB_PATH=/path/to/db# Count keys in database"
	@echo "  make show-pointers DB_PATH=/path    # Show pointer keys"
	@echo "  make copy-pointers SRC_DB=... DST_DB=... # Copy pointer keys"
	@echo "  make inspect-sst                    # Inspect SST files in restored DB"
	@echo ""
	@echo "LAUNCH COMMANDS (Full Network):"
	@echo "  make launch                         # Launch full network (primary + L2s)"
	@echo "  make launch-full                    # Same as 'make launch'"
	@echo "  make launch-primary                 # Launch only LUX primary network"
	@echo "  make launch-docker                  # Launch full network with Docker (recommended)"
	@echo "  make launch-test                    # Launch test configuration"
	@echo "  make kill-node                      # Stop all running nodes"
	@echo "  make network-info                   # Show network information"
	@echo ""
	@echo "DEPLOYMENT (Individual):"
	@echo "  make deploy-lux                     # Deploy LUX network"
	@echo "  make deploy-l2 L2=zoo               # Deploy L2 (zoo/spc/hanzo)"
	@echo "  make deploy-all                     # Deploy all networks"
	@echo ""
	@echo "UTILITIES:"
	@echo "  make clean                          # Clean output directories"
	@echo "  make test                           # Run all tests"
	@echo "  make validate                       # Validate all genesis files"
	@echo "  make backup                         # Backup current genesis"
	@echo "  make genesis-cmd CMD=\"...\"          # Run custom genesis command"
	@echo "  make namespace ARGS=\"...\"         # Run namespace tool"
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
ARCHEOLOGY := ./bin/archaeology

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
	@cd .. && make build-teleport build-archaeology
	@echo "✅ Tools built"

# Build unified genesis tool
build: build-genesis build-extract-genesis

build-genesis:
	@echo "🔨 Building unified genesis tool..."
	@go work use .
	@go build -o bin/genesis ./cmd/genesis
	@echo "✅ Genesis tool built"

build-extract-genesis:
	@echo "🔨 Building extract-genesis tool..."
	@go build -o bin/extract-genesis ./cmd/extract-genesis
	@echo "✅ Extract-genesis tool built"

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

# Build primary network with clean genesis
build-primary: build-evm build-node
	@echo "✅ Primary network build complete"

build-evm:
	@echo "🔨 Building EVM..."
	@cd ../evm && make

build-node:
	@echo "🔨 Building Node..."
	@cd ../node && make

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

# ============ LAUNCH COMMANDS ============

# Primary network configuration
NODE_DIR ?= ../node
CLI_DIR ?= ../cli
DATA_DIR ?= ../.devnet
IMPORT_DIR ?= $(OUTPUT_DIR)/import-ready
LUX_POA_CHAIN_ID := 96369
LUX_PRIMARY_CHAIN_ID := 43125

# Clean genesis devnet configuration
devnet: build-primary kill-node
	@echo "🚀 Launching clean genesis devnet..."
	@mkdir -p $(DATA_DIR)/{node0,node1,node2}/{db,logs}
	@$(MAKE) devnet-configs
	@$(MAKE) devnet-start
	@echo "✅ Devnet launched!"
	@echo ""
	@echo "Node endpoints:"
	@echo "  - Node 0: http://127.0.0.1:9650"
	@echo "  - Node 1: http://127.0.0.1:9652"
	@echo "  - Node 2: http://127.0.0.1:9654"

devnet-configs:
	@for i in 0 1 2; do \
		HTTP_PORT=$$((9650 + i*2)); \
		STAKING_PORT=$$((9651 + i*2)); \
		cat > $(DATA_DIR)/node$$i.json <<EOF \
{ \
  "network-id": "43125", \
  "db-dir": "$(DATA_DIR)/node$$i/db", \
  "log-dir": "$(DATA_DIR)/node$$i/logs", \
  "http-port": $$HTTP_PORT, \
  "staking-port": $$STAKING_PORT, \
  "bootstrap-ips": "", \
  "bootstrap-ids": "", \
  "genesis-file": "./lux-pchain-genesis.json", \
  "snow-mixed-query-num-push-vdr": 1, \
  "consensus-shutdown-timeout": "1s" \
} \
EOF; \
	done

devnet-start:
	@cd $(NODE_DIR) && ./build/luxd --config-file="$(DATA_DIR)/node0.json" > "$(DATA_DIR)/node0/output.log" 2>&1 &
	@sleep 5
	@cd $(NODE_DIR) && ./build/luxd --config-file="$(DATA_DIR)/node1.json" > "$(DATA_DIR)/node1/output.log" 2>&1 &
	@cd $(NODE_DIR) && ./build/luxd --config-file="$(DATA_DIR)/node2.json" > "$(DATA_DIR)/node2/output.log" 2>&1 &
	@echo "PID files created in $(DATA_DIR)/"

devnet-test:
	@echo "🧪 Testing devnet features..."
	@curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"health.health"}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/health | jq .
	@curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type:application/json;' http://127.0.0.1:9650/ext/bc/C/rpc | jq .

# Kill any existing processes
kill-node:
	@echo "🛑 Stopping existing nodes..."
	@pkill -f luxd || true
	@pkill -f avalanche || true
	@sleep 2

# Prepare import-ready data
prepare-import: $(IMPORT_DIR)
	@echo "🔧 Preparing genesis data for import..."
	@mkdir -p $(IMPORT_DIR)/{lux,zoo,spc,hanzo}/{C,L2}
	
	# LUX mainnet - prepare C-Chain data
	@if [ -d "$(DATA_DIR)/lux-mainnet-$(LUX_POA_CHAIN_ID)/db/pebbledb" ]; then \
		echo "  ✅ Found LUX mainnet chaindata"; \
		cp -r $(DATA_DIR)/lux-mainnet-$(LUX_POA_CHAIN_ID)/db/pebbledb $(IMPORT_DIR)/lux/C/chaindata; \
		if [ -f "$(DATA_DIR)/configs/lux-mainnet-$(LUX_POA_CHAIN_ID)/genesis.json" ]; then \
			cp $(DATA_DIR)/configs/lux-mainnet-$(LUX_POA_CHAIN_ID)/genesis.json $(IMPORT_DIR)/lux/C/genesis.json; \
		fi; \
	elif [ -d "chaindata/lux-mainnet-$(LUX_POA_CHAIN_ID)/db/pebbledb" ]; then \
		echo "  ✅ Found LUX mainnet chaindata in local directory"; \
		cp -r chaindata/lux-mainnet-$(LUX_POA_CHAIN_ID)/db/pebbledb $(IMPORT_DIR)/lux/C/chaindata; \
		if [ -f "chaindata/configs/lux-mainnet-$(LUX_POA_CHAIN_ID)/genesis.json" ]; then \
			cp chaindata/configs/lux-mainnet-$(LUX_POA_CHAIN_ID)/genesis.json $(IMPORT_DIR)/lux/C/genesis.json; \
		fi; \
	fi
	
	# ZOO L2 - prepare with BSC migration
	@if [ -d "$(DATA_DIR)/zoo-mainnet-$(ZOO_MAINNET)/db/pebbledb" ]; then \
		echo "  ✅ Found ZOO mainnet chaindata"; \
		cp -r $(DATA_DIR)/zoo-mainnet-$(ZOO_MAINNET)/db/pebbledb $(IMPORT_DIR)/zoo/L2/chaindata; \
		if [ -f "$(DATA_DIR)/configs/zoo-mainnet-$(ZOO_MAINNET)/genesis.json" ]; then \
			cp $(DATA_DIR)/configs/zoo-mainnet-$(ZOO_MAINNET)/genesis.json $(IMPORT_DIR)/zoo/L2/genesis.json; \
		fi; \
	elif [ -d "chaindata/zoo-mainnet-$(ZOO_MAINNET)/db/pebbledb" ]; then \
		echo "  ✅ Found ZOO mainnet chaindata in local directory"; \
		cp -r chaindata/zoo-mainnet-$(ZOO_MAINNET)/db/pebbledb $(IMPORT_DIR)/zoo/L2/chaindata; \
		if [ -f "chaindata/configs/zoo-mainnet-$(ZOO_MAINNET)/genesis.json" ]; then \
			cp chaindata/configs/zoo-mainnet-$(ZOO_MAINNET)/genesis.json $(IMPORT_DIR)/zoo/L2/genesis.json; \
		fi; \
	fi
	
	# SPC L2 - prepare chaindata
	@if [ -d "$(DATA_DIR)/spc-mainnet-$(SPC_MAINNET)/db/pebbledb" ]; then \
		echo "  ✅ Found SPC mainnet chaindata"; \
		cp -r $(DATA_DIR)/spc-mainnet-$(SPC_MAINNET)/db/pebbledb $(IMPORT_DIR)/spc/L2/chaindata; \
		if [ -f "$(DATA_DIR)/configs/spc-mainnet-$(SPC_MAINNET)/genesis.json" ]; then \
			cp $(DATA_DIR)/configs/spc-mainnet-$(SPC_MAINNET)/genesis.json $(IMPORT_DIR)/spc/L2/genesis.json; \
		fi; \
	elif [ -d "chaindata/spc-mainnet-$(SPC_MAINNET)/db/pebbledb" ]; then \
		echo "  ✅ Found SPC mainnet chaindata in local directory"; \
		cp -r chaindata/spc-mainnet-$(SPC_MAINNET)/db/pebbledb $(IMPORT_DIR)/spc/L2/chaindata; \
		if [ -f "chaindata/configs/spc-mainnet-$(SPC_MAINNET)/genesis.json" ]; then \
			cp chaindata/configs/spc-mainnet-$(SPC_MAINNET)/genesis.json $(IMPORT_DIR)/spc/L2/genesis.json; \
		fi; \
	fi
	
	# Hanzo L2 - fresh genesis only
	@echo "  📄 Creating fresh Hanzo genesis..."
	@echo '{"chainId": $(HANZO_MAINNET), "homesteadBlock": 0, "eip150Block": 0, "eip155Block": 0, "eip158Block": 0, "byzantiumBlock": 0, "constantinopleBlock": 0, "petersburgBlock": 0, "istanbulBlock": 0, "muirGlacierBlock": 0, "subnetEVMTimestamp": 0}' > $(IMPORT_DIR)/hanzo/L2/genesis.json
	
	@echo "✅ Import preparation complete"

# Launch LUX primary network with POA automining
launch-lux: kill-node
	@echo "🚀 Launching LUX mainnet (Chain ID: $(LUX_PRIMARY_CHAIN_ID)+$(LUX_POA_CHAIN_ID))..."
	@cd $(NODE_DIR) && nohup ./build/luxd \
		--network-id=$(LUX_POA_CHAIN_ID) \
		--data-dir="$(DATA_DIR)" \
		--chain-config-content='{"C": {"chainId": $(LUX_PRIMARY_CHAIN_ID), "state-sync-enabled": false, "pruning-enabled": false}}' \
		--http-host=0.0.0.0 \
		--http-port=9650 \
		--staking-enabled=false \
		--sybil-protection-enabled=false \
		--bootstrap-ips="" \
		--bootstrap-ids="" \
		--public-ip=127.0.0.1 \
		--snow-sample-size=1 \
		--snow-quorum-size=1 \
		--snow-virtuous-commit-threshold=1 \
		--snow-rogue-commit-threshold=1 \
		--snow-concurrent-repolls=1 \
		--index-enabled \
		--db-dir="$(DATA_DIR)/db" \
		> $(OUTPUT_DIR)/lux-mainnet.log 2>&1 &
	@sleep 10
	@if curl -s -X POST --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' -H 'content-type:application/json;' http://localhost:9650/ext/bc/C/rpc > /dev/null; then \
		echo "✅ LUX mainnet running on chain ID $(LUX_PRIMARY_CHAIN_ID)"; \
	else \
		echo "❌ Failed to start LUX mainnet"; \
		exit 1; \
	fi

# Create and deploy L2s with existing or fresh data
create-l2s: 
	@echo "🚀 Creating L2s..."
	@cd $(CLI_DIR) && \
	export AVALANCHE_NETWORK=Local && \
	export AVALANCHE_CHAIN_ID=$(LUX_POA_CHAIN_ID) && \
	\
	echo "Creating ZOO L2 (with existing data)..." && \
	./bin/lux blockchain create zoo \
		--evm \
		--chain-id=$(ZOO_MAINNET) \
		--token-symbol=ZOO \
		--genesis-file=$(IMPORT_DIR)/zoo/L2/genesis.json \
		--force && \
	\
	echo "Creating SPC L2 (with existing data)..." && \
	./bin/lux blockchain create spc \
		--evm \
		--chain-id=$(SPC_MAINNET) \
		--token-symbol=SPC \
		--genesis-file=$(IMPORT_DIR)/spc/L2/genesis.json \
		--force && \
	\
	echo "Creating Hanzo L2 (fresh deployment)..." && \
	./bin/lux blockchain create hanzo \
		--evm \
		--chain-id=$(HANZO_MAINNET) \
		--token-symbol=AI \
		--genesis-file=$(IMPORT_DIR)/hanzo/L2/genesis.json \
		--force

deploy-l2s:
	@echo "🚀 Deploying L2s to local network..."
	@cd $(CLI_DIR) && \
	export AVALANCHE_NETWORK=Local && \
	\
	echo "Deploying ZOO L2..." && \
	./bin/lux blockchain deploy zoo --local --avalanchego-version latest && \
	\
	echo "Deploying SPC L2..." && \
	./bin/lux blockchain deploy spc --local --avalanchego-version latest && \
	\
	echo "Deploying Hanzo L2..." && \
	./bin/lux blockchain deploy hanzo --local --avalanchego-version latest

# Get network information
network-info:
	@echo "📊 Network Information"
	@echo "===================="
	@cd $(CLI_DIR) && \
	\
	echo "LUX Primary Network:" && \
	echo "  Chain ID: $(LUX_PRIMARY_CHAIN_ID) (presented as)" && \
	echo "  Network ID: $(LUX_POA_CHAIN_ID) (actual)" && \
	echo "  RPC: http://localhost:9650/ext/bc/C/rpc" && \
	echo "" && \
	\
	./bin/lux blockchain list && \
	echo "" && \
	\
	if ./bin/lux blockchain describe zoo 2>/dev/null | grep -q "Blockchain ID"; then \
		echo "ZOO L2:" && \
		./bin/lux blockchain describe zoo | grep -E "(Chain ID|Blockchain ID|RPC URL)" && \
		echo ""; \
	fi && \
	\
	if ./bin/lux blockchain describe spc 2>/dev/null | grep -q "Blockchain ID"; then \
		echo "SPC L2:" && \
		./bin/lux blockchain describe spc | grep -E "(Chain ID|Blockchain ID|RPC URL)" && \
		echo ""; \
	fi && \
	\
	if ./bin/lux blockchain describe hanzo 2>/dev/null | grep -q "Blockchain ID"; then \
		echo "Hanzo L2:" && \
		./bin/lux blockchain describe hanzo | grep -E "(Chain ID|Blockchain ID|RPC URL)" && \
		echo ""; \
	fi

# Main launch targets
launch: launch-full
	@echo "✅ Full network launched!"

launch-docker:
	@echo "🐳 Launching network with Docker..."
	@NETWORK=$(NETWORK) docker-compose -f docker/compose.yml up --build
	@echo "✅ Docker network launched!"

launch-full: prepare-import launch-lux create-l2s deploy-l2s network-info
	@echo "✅ Full Lux network with L2s launched successfully!"

launch-primary: launch-lux
	@echo "✅ LUX primary network launched!"

launch-test: kill-node
	@echo "🧪 Launching test configuration..."
	@$(MAKE) launch-lux
	@echo "✅ Test network ready!"

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

deploy-hanzo:
	@echo "🚀 Deploying Hanzo L2 (fresh)..."
	@lux subnet create hanzo-mainnet \
		--evm \
		--chain-id $(HANZO_MAINNET) \
		--token-symbol AI
	@echo "✅ Hanzo deployment complete"

deploy-all: deploy-lux
	@$(MAKE) deploy-l2 L2=zoo
	@$(MAKE) deploy-l2 L2=spc
	@$(MAKE) deploy-l2 L2=hanzo
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

test-genesis: build
	@echo "🧪 Testing unified genesis tool..."
	@./bin/genesis --help > /dev/null
	@./bin/genesis tools > /dev/null
	@./bin/genesis validators list > /dev/null || true
	@./bin/genesis extract --help > /dev/null
	@./bin/genesis analyze --help > /dev/null
	@./bin/genesis migrate --help > /dev/null
	@echo "✅ Genesis tool tests passed"

# ============ MIGRATION COMMANDS ============

# Migrate historic chain data to new blockchain ID
migrate: build-genesis
	@echo "🔄 Migrating historic chain data..."
	@./bin/genesis migrate \
		/home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ
	@echo "✅ Migration complete"

# Extract genesis from historic data only
migrate-genesis: build-genesis
	@echo "📤 Extracting genesis from historic data..."
	@./bin/genesis read \
		/home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ \
		--write-config --show-id --raw --pointers
	@echo "✅ Genesis extraction complete"

# ============ DIAGNOSTIC COMMANDS ============

# Diagnose blockchain database
diagnose: build-genesis
	@echo "🔍 Diagnosing blockchain database..."
	@./bin/genesis diagnose $(DB_PATH)

# Diagnose historic database
diagnose-historic: build-genesis
	@./bin/genesis diagnose /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ

# Count keys in database
count-keys: build-genesis
	@echo "📊 Counting keys in database..."
	@./bin/genesis count $(DB_PATH) --all

# Show pointer keys
show-pointers: build-genesis
	@echo "🔑 Showing pointer keys..."
	@./bin/genesis pointers show $(DB_PATH)

# Copy pointer keys between databases
copy-pointers: build-genesis
ifndef SRC_DB
	$(error SRC_DB is not set. Usage: make copy-pointers SRC_DB=/path/to/source DST_DB=/path/to/dest)
endif
ifndef DST_DB
	$(error DST_DB is not set. Usage: make copy-pointers SRC_DB=/path/to/source DST_DB=/path/to/dest)
endif
	@echo "📋 Copying pointer keys from $(SRC_DB) to $(DST_DB)..."
	@./bin/genesis pointers copy $(SRC_DB) $(DST_DB)

# ============ ANALYSIS COMMANDS ============

# Inspect SST files in restored database
inspect-sst:
	@echo "🔍 Inspecting Restored Database Files"
	@echo "====================================="
	@echo ""
	@echo "📁 Database path: /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb"
	@echo ""
	@echo "📊 Database statistics:"
	@echo "  Total SST files: $$(ls -1 /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb/*.sst 2>/dev/null | wc -l)"
	@echo "  Total size: $$(du -sh /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb 2>/dev/null | cut -f1)"
	@echo ""
	@echo "📄 Sample SST files:"
	@ls -lh /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb/*.sst 2>/dev/null | head -5
	@echo ""
	@echo "⚠️  Note: This database is missing CURRENT/MANIFEST files"

# ============ COMPREHENSIVE MIGRATION WORKFLOW ============

# Complete migration workflow
migrate-complete: build-genesis
	@echo "🔄 Complete Historic Data Migration Workflow"
	@echo "==========================================="
	@echo ""
	@echo "Step 1: Diagnosing source database..."
	@$(MAKE) diagnose-historic || true
	@echo ""
	@echo "Step 2: Extracting genesis..."
	@$(MAKE) migrate-genesis
	@echo ""
	@echo "Step 3: Running migration..."
	@$(MAKE) migrate
	@echo ""
	@echo "✅ Migration workflow complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Review genesis at ~/.luxd/configs/C/genesis.json"
	@echo "2. Start node: cd /home/z/work/lux/node && ./build/luxd --http-port=9630"

# Dry run migration to see what would happen
migrate-dry-run: build-genesis
	@echo "🔍 Migration Dry Run"
	@./bin/genesis migrate \
		/home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ \
		--dry-run

# ============ HELPER COMMANDS ============

# Run genesis tool with custom command
genesis-cmd: build-genesis
ifndef CMD
	$(error CMD is not set. Usage: make genesis-cmd CMD="diagnose /path/to/db")
endif
	@./bin/genesis $(CMD)

# Run namespace tool directly
namespace: build-migrate
ifndef ARGS
	$(error ARGS is not set. Usage: make namespace ARGS="-src /path -dst /path -migrate-id")
endif
	@./bin/namespace $(ARGS)

# ============ DATABASE VARIABLES ============

# Default database paths
HISTORIC_DB ?= /home/z/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ
DB_PATH ?= $(HISTORIC_DB)

# Show all genesis tool commands
genesis-help: build-genesis
	@./bin/genesis --help

# Show specific genesis command help
genesis-help-cmd: build-genesis
ifndef CMD
	$(error CMD is not set. Usage: make genesis-help-cmd CMD=diagnose)
endif
	@./bin/genesis $(CMD) --help

# Show current status
status: 
	@echo "📊 Genesis Tool Status"
	@echo "====================="
	@echo ""
	@echo "✅ Genesis tool: $$(if [ -f ./bin/genesis ]; then echo 'Built'; else echo 'Not built'; fi)"
	@echo "✅ Historic DB: $(HISTORIC_DB)"
	@echo "✅ SST files: $$(ls -1 $(HISTORIC_DB)/db/pebbledb/*.sst 2>/dev/null | wc -l) files"
	@echo ""
	@echo "Available commands:"
	@echo "  ./genesis --help                  # Show all commands"
	@echo "  make diag                         # Diagnose historic DB"
	@echo "  make migrate-complete             # Run full migration"

# Aliases for common operations
zoo: pipeline-zoo
fresh: pipeline-fresh
diag: diagnose-historic

up:
ifeq ($(strip $(NETWORK)),)
	@echo "🚀 Launching full historic genesis network..."
	@$(MAKE) validators-generate
	@$(MAKE) genesis-all
	@$(MAKE) launch-docker
else
	@echo "🚀 Launching single network: $(NETWORK)..."
	@$(MAKE) genesis-$(NETWORK)
	@$(MAKE) launch-docker NETWORK=$(NETWORK)
endif

.DEFAULT_GOAL := help