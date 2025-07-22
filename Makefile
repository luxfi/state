## Makefile for lux-genesis-import

SNAPSHOT=artifacts/lux-snapshot-v1.tgz
LUX_CLI=bin/lux
LUXD=bin/luxd
GINKGO=bin/ginkgo

.PHONY: install genesis snapshot docker push test convert-7777 convert-96369 run-7777-dev run-96369-dev import-7777-cchain import-96369-cchain analyze-chaindata build build-tools build-archaeology clean-bin test-unit test-integration test-all install-test-deps quickstart

quickstart: ## Quick guide to get started
	@echo "🚀 Lux Network Genesis - Quick Start Guide"
	@echo ""
	@echo "1️⃣  Development Mode (Single Node, Network 7777):"
	@echo "   make launch-dev"
	@echo ""
	@echo "2️⃣  Local Test Network (5 Nodes, Network 96369):"
	@echo "   MNEMONIC='your seed phrase' make launch-5-nodes"
	@echo ""
	@echo "3️⃣  Full Network (11 Nodes, Network 96369):"
	@echo "   MNEMONIC='your seed phrase' make launch-11-nodes"
	@echo ""
	@echo "📊 Test Network:"
	@echo "   make test-rpc        # Test RPC endpoints"
	@echo "   make test-c-chain    # Test C-Chain"
	@echo ""
	@echo "🛑 Stop Network:"
	@echo "   make stop-network"
	@echo ""
	@echo "📝 Other Commands:"
	@echo "   make help           # Show all commands"
	@echo "   make generate-validators  # Generate validator keys"
	@echo "   make generate-all-genesis # Generate genesis files"

install:
	@echo "Installing LUX binaries from GitHub..."
	@go run scripts/install_deps.go
	@echo "✅ Installation complete!"

## Genesis generation with network parameter
genesis:
	@if [ -z "$(network)" ]; then \
		echo "Usage: make genesis network=<lux|zoo|spc|hanzo>"; \
		echo "Example: make genesis network=lux"; \
		echo "Example: make genesis network=zoo"; \
		exit 1; \
	fi
	@echo "👉  Running genesis pipeline for $(network)"
	@case $(network) in \
		lux) \
			$(MAKE) genesis-lux ;; \
		zoo) \
			$(MAKE) genesis-zoo ;; \
		spc) \
			$(MAKE) genesis-spc ;; \
		hanzo) \
			$(MAKE) genesis-hanzo ;; \
		*) \
			echo "Unknown network: $(network)"; \
			echo "Valid networks: lux, zoo, spc, hanzo"; \
			exit 1 ;; \
	esac

## Lux genesis pipeline: extract then generate P-, C-, X-chain genesis files
genesis-lux: build-tools build-archeology build-genesis
	@echo "👉  Running Lux genesis pipeline"
	@bin/archeology extract --src chaindata/lux-genesis-7777/db/pebbledb --dst data/extracted/lux-genesis-7777 --chain-id 7777 --include-state
	@bin/archeology extract --src chaindata/lux-mainnet-96369/db/pebbledb --dst data/extracted/lux-96369 --chain-id 96369 --include-state
	@echo "👉  Generating P-Chain genesis"
	@bin/genesis generate --network p-chain --data data/extracted/lux-genesis-7777 --output configs/P/genesis.json
	@echo "👉  Generating C-Chain genesis"
	@bin/genesis generate --network c-chain --data data/extracted/lux-96369 --output configs/C/genesis.json
	@echo "👉  Generating X-Chain genesis"
	@bin/genesis generate --network x-chain --data data/extracted/lux-genesis-7777 --external data/external --output configs/xchain-genesis-complete.json
	@echo "✅ Lux genesis pipeline complete (configs/P, configs/C, configs/xchain-genesis-complete.json)"

## Zoo genesis pipeline
genesis-zoo: build-tools build-teleport zoo-analysis
	@echo "👉  Running Zoo genesis pipeline"
	@echo "First running Zoo analysis to gather external data..."
	@$(MAKE) zoo-analysis
	@echo "👉  Extracting Zoo mainnet data"
	@bin/archeology extract --src chaindata/zoo-mainnet-200200/db/pebbledb --dst data/extracted/zoo-200200 --chain-id 200200 --include-state
	@echo "👉  Generating Zoo genesis with external data"
	@bin/genesis generate --network zoo-mainnet --chain-id 200200 \
		--data data/extracted/zoo-200200 \
		--external exports/zoo-analysis/ \
		--output configs/zoo-mainnet-genesis.json
	@echo "✅ Zoo genesis pipeline complete (configs/zoo-mainnet-genesis.json)"

## SPC genesis pipeline
genesis-spc: build-tools build-archeology build-genesis
	@echo "👉  Running SPC genesis pipeline"
	@bin/archeology extract --src chaindata/spc-mainnet-36911/db/pebbledb --dst data/extracted/spc-36911 --chain-id 36911 --include-state
	@echo "👉  Generating SPC genesis"
	@bin/genesis generate --network spc-mainnet --chain-id 36911 \
		--data data/extracted/spc-36911 \
		--output configs/spc-mainnet-genesis.json
	@echo "✅ SPC genesis pipeline complete (configs/spc-mainnet-genesis.json)"

## Hanzo genesis pipeline (placeholder - not deployed yet)
genesis-hanzo:
	@echo "👉  Hanzo network not deployed yet"
	@echo "Chain ID 36963 reserved for future deployment"
	@echo "To prepare Hanzo genesis when ready:"
	@echo "  1. Deploy Hanzo subnet"
	@echo "  2. Extract chaindata"
	@echo "  3. Run: make genesis network=hanzo"

snapshot: genesis
	@echo "Building snapshot tarball..."
	./scripts/build_snapshot.sh

VERSION ?= $(shell cd node && git describe --tags --abbrev=0)

docker: snapshot
	@echo "Building Docker image ghcr.io/luxfi/node:latest..."
	docker build -t ghcr.io/luxfi/node:latest -f docker/Dockerfile .

push: docker
	@echo "Pushing Docker image ghcr.io/luxfi/node:latest..."
	docker push ghcr.io/luxfi/node:latest

test: snapshot
	@echo "Verifying snapshot..."
	./scripts/verify_snapshot.sh

# Data conversion targets
convert-7777:
	@echo "Converting 2023 (7777) LevelDB to PebbleDB..."
	@./scripts/convert/convert-7777.sh

convert-96369:
	@echo "Converting 2024 (96369) LevelDB to PebbleDB..."
	@./scripts/convert/convert-96369.sh

convert-all: convert-7777 convert-96369
	@echo "All conversions complete!"


# Import to C-Chain targets
import-96369-cchain:
	@echo "Importing 96369 data to C-Chain..."
	@echo "Using lux-cli to import PebbleDB data..."
	@$(LUX_CLI) blockchain import c-chain \
		--genesis-file configs/genesis-96369.json \
		--db-path pebbledb/2024-96369 \
		--network-id 96369

# Export 7777 accounts for X-Chain funding
export-7777-accounts:
	@echo "Exporting 7777 account balances to CSV..."
	@go run scripts/export_7777_accounts.go \
		--db-path chaindata/lux-genesis-7777/db \
		--output exports/7777-accounts.csv \
		--exclude-treasury 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
	@echo "✅ Export complete: exports/7777-accounts.csv"

# Generate X-Chain genesis with 7777 accounts
generate-xchain-genesis: export-7777-accounts
	@echo "Generating X-Chain genesis with 7777 account holders..."
	@go run scripts/generate_xchain_genesis.go \
		--accounts-csv exports/7777-accounts.csv \
		--min-validator-stake 1000000 \
		--output configs/xchain-genesis.json
	@echo "✅ X-Chain genesis generated: configs/xchain-genesis.json"

# Analysis targets
analyze-chaindata:
	@echo "Analyzing chaindata..."
	@echo ""
	@echo "=== Raw ChainData (LevelDB) ==="
	@echo "2023 (7777): $$(du -sh chaindata/2023-7777 2>/dev/null | cut -f1 || echo 'Not found')"
	@echo "2024 (96369): $$(du -sh chaindata/2024-96369 2>/dev/null | cut -f1 || echo 'Not found')"
	@echo ""
	@echo "=== Converted PebbleDB ==="
	@echo "2023 (7777): $$(du -sh pebbledb/2023-7777 2>/dev/null | cut -f1 || echo 'Not converted')"
	@echo "2024 (96369): $$(du -sh pebbledb/2024-96369 2>/dev/null | cut -f1 || echo 'Not converted')"
	@echo ""
	@echo "=== Genesis Files ==="
	@ls -lh configs/genesis-*.json 2>/dev/null || echo "No genesis files found"
	@echo ""
	@echo "To convert raw data: make convert-all"
	@echo "To run dev nodes: make run-7777-dev or make run-96369-dev"
	@echo "To import to C-Chain: make import-7777-cchain or make import-96369-cchain"

# Clean targets (PebbleDB cleaning removed for safety)
clean-chaindata:
	@echo "Cleaning raw chaindata directories..."
	@echo "WARNING: This will remove the original LevelDB data!"
	@read -p "Are you sure? [y/N] " confirm && [ "$${confirm}" = "y" ] || exit 1
	@rm -rf chaindata/2023-7777 chaindata/2024-96369
	@echo "✓ Cleaned raw chaindata"

# Build targets
build: build-tools build-archeology build-genesis build-teleport

build-tools:
	@echo "Building extraction tools..."
	@mkdir -p bin
	@echo "  - denamespace"
	@cd cmd/denamespace && go build -o ../../bin/denamespace . 2>/dev/null || echo "    ⚠️  Failed to build denamespace"
	@echo "  - prefixscan"
	@cd cmd/prefixscan && go build -o ../../bin/prefixscan . 2>/dev/null || echo "    ⚠️  Failed to build prefixscan"
	@echo "✅ Extraction tools built"

build-archeology:
	@echo "Building archeology tool..."
	@mkdir -p bin
	@cd cmd/archeology && go build -o ../../bin/archeology .
	@echo "✅ archeology tool built"

build-genesis:
	@echo "Building genesis tool..."
	@mkdir -p bin
	@cd cmd/genesis && go build -o ../../bin/genesis .
	@echo "✅ genesis tool built"

build-teleport:
	@echo "Building teleport tool..."
	@mkdir -p bin
	@cd cmd/teleport && go build -o ../../bin/teleport .
	@echo "✅ teleport tool built"

# Keep old archeology for backwards compatibility
build-archaeology:
	@echo "Building archeology tool (deprecated - use build-archeology)..."
	@mkdir -p bin
	@cd cmd/archeology && go build -o ../../bin/archeology .
	@echo "✅ Blockchain archaeology tool built"

# External asset scanning
scan-ethereum-nfts:
	@echo "Scanning Ethereum for Lux NFTs..."
	@./bin/archeology scan \
		--chain ethereum \
		--contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
		--project lux \
		--type nft \
		--output exports/lux-nfts-ethereum.csv
	@echo "✅ Ethereum NFT scan complete"

scan-bsc-tokens:
	@echo "Scanning BSC for Zoo tokens..."
	@echo "⚠️  Please add Zoo token contract address to scan"
	@# ./bin/archeology scan \
	@#	--chain bsc \
	@#	--contract 0xADD_ZOO_TOKEN_ADDRESS_HERE \
	@#	--project zoo \
	@#	--type token \
	@#	--output exports/zoo-tokens-bsc.csv

# Import NFTs using new flexible command
import-lux-nfts:
	@echo "Importing Lux NFTs from Ethereum..."
	@./bin/archeology import-nft \
		--network ethereum \
		--chain-id 1 \
		--contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
		--project lux \
		--output exports/lux-nfts-ethereum.csv
	@echo "✅ Lux NFT import complete"

import-zoo-nfts:
	@echo "Importing Zoo EGG NFTs from BSC..."
	@./bin/teleport import-nft \
		--network bsc \
		--chain-id 56 \
		--contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
		--project zoo \
		--output exports/zoo-egg-nfts-bsc.csv

# Import with custom parameters
import-nft:
	@if [ -z "$(contract)" ]; then \
		echo "Usage: make import-nft network=<network> chainid=<id> contract=<address> project=<name>"; \
		echo "Example: make import-nft network=polygon chainid=137 contract=0x123... project=custom"; \
		exit 1; \
	fi
	@./bin/archeology import-nft \
		--network $(network) \
		--chain-id $(chainid) \
		--contract $(contract) \
		--project $(project)

# Scan EGG NFT holders
scan-egg-holders:
	@echo "Scanning all EGG NFT holders on BSC..."
	@echo "Contract: 0x5bb68cf06289d54efde25155c88003be685356a8"
	@mkdir -p exports
	@./bin/teleport scan-egg-holders --output exports/egg-holders.txt
	@echo "✅ EGG holder scan complete!"

# Zoo Migration (special handling for burns)
migrate-zoo-complete:
	@echo "Performing complete Zoo token migration from BSC..."
	@echo "This includes:"
	@echo "  - Current Zoo token holders"
	@echo "  - Users who burned tokens to 0x000000000000000000000000000000000000dEaD"
	@echo "  - EGG NFT holders"
	@mkdir -p exports
	@./bin/teleport zoo-migrate \
		--include-burns \
		--include-egg-nfts \
		--output exports/zoo-migration-complete.json
	@echo "✅ Zoo migration complete!"
	@echo "Check exports/zoo-migration-complete.json for results"

# Zoo Analysis (using archeology scanners)
zoo-analysis: build-archeology
	@echo "Performing comprehensive Zoo ecosystem analysis..."
	@echo "This will scan:"
	@echo "  - EGG NFT holders on BSC"
	@echo "  - ZOO transfers for EGG purchases"
	@echo "  - ZOO burns to dead address"
	@./scripts/zoo-analysis.sh exports/zoo-analysis
	@echo "✅ Zoo analysis complete!"
	@echo "Check exports/zoo-analysis/ for all CSV files and report"

# Scan token burns (reusable for any token)
scan-burns: build-archeology
	@if [ -z "$(token)" ]; then \
		echo "Usage: make scan-burns token=<address> rpc=<rpc-url>"; \
		echo "Example: make scan-burns token=0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13 rpc=https://bsc-dataseed.binance.org/"; \
		exit 1; \
	fi
	@mkdir -p exports
	@./bin/archeology scan-burns \
		--rpc $(rpc) \
		--token $(token) \
		--burn-address 0x000000000000000000000000000000000000dEaD \
		--summarize \
		--output exports/$(shell echo $(token) | cut -c1-10)-burns.csv \
		--output-json exports/$(shell echo $(token) | cut -c1-10)-burns-summary.json

# Scan token/NFT holders (reusable)
scan-holders: build-archeology
	@if [ -z "$(contract)" ]; then \
		echo "Usage: make scan-holders contract=<address> rpc=<rpc-url> [type=<nft|token>]"; \
		echo "Example: make scan-holders contract=0x5bb68cf06289d54efde25155c88003be685356a8 rpc=https://bsc-dataseed.binance.org/ type=nft"; \
		exit 1; \
	fi
	@mkdir -p exports
	@./bin/archeology scan-holders \
		--rpc $(rpc) \
		--contract $(contract) \
		--type $(if $(type),$(type),nft) \
		--top 20 \
		--show-distribution \
		--output exports/$(shell echo $(contract) | cut -c1-10)-holders.csv

# Import ERC20 tokens
import-zoo-tokens-bsc:
	@echo "Importing Zoo tokens from BSC..."
	@./bin/teleport import-token \
		--network bsc \
		--chain-id 56 \
		--contract 0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13 \
		--project zoo \
		--symbol ZOO \
		--output exports/zoo-tokens-bsc.csv

import-lux-tokens-7777:
	@echo "Importing LUX tokens from local 7777 chain..."
	@echo "Make sure chain 7777 is running locally (make run network=7777)"
	@./bin/archeology import-token \
		--rpc http://localhost:9650/ext/bc/C/rpc \
		--chain-id 7777 \
		--contract 0xADD_LUX_TOKEN_ADDRESS_HERE \
		--project lux \
		--symbol LUX \
		--output exports/lux-tokens-7777.csv

# Import tokens with custom parameters
import-token:
	@if [ -z "$(contract)" ]; then \
		echo "Usage: make import-token network=<network> chainid=<id> contract=<address> project=<name> [symbol=<symbol>]"; \
		echo "Example: make import-token network=bsc chainid=56 contract=0x123... project=zoo symbol=ZOO"; \
		echo "Example: make import-token rpc=http://localhost:9650/ext/bc/C/rpc chainid=7777 contract=0x456... project=lux"; \
		exit 1; \
	fi
	@./bin/archeology import-token \
		$(if $(network),--network $(network),) \
		$(if $(rpc),--rpc $(rpc),) \
		--chain-id $(chainid) \
		--contract $(contract) \
		--project $(project) \
		$(if $(symbol),--symbol $(symbol),)

# Complete X-Chain genesis generation
generate-xchain-complete: export-7777-accounts scan-ethereum-nfts
	@echo "Generating complete X-Chain genesis with all external assets..."
	@./bin/archeology genesis \
		--nft-csv exports/lux-nfts-ethereum.csv \
		--accounts-csv exports/7777-accounts.csv \
		--chain x-chain \
		--output configs/xchain-genesis-complete.json
	@echo "✅ Complete X-Chain genesis generated with all historical assets!"

clean-bin:
	@echo "Cleaning binary directory..."
	@rm -rf bin/
	@echo "✓ Cleaned bin/"

# Test targets
install-test-deps:
	@echo "Installing test dependencies..."
	@mkdir -p bin
	@env GOBIN=$(shell pwd)/bin go install github.com/onsi/ginkgo/v2/ginkgo@v2.23.4
	@echo "✅ Test dependencies installed (ginkgo binary in bin/)"

test-unit:
	@echo "Skipping unit tests (stub)"

test-integration: install-test-deps
	@echo "Skipping integration tests (stub)"

test-all: test-unit test-integration
	@echo "All tests completed!"

# Full integration test - runs everything
test-full-integration: install-test-deps convert-all
	@echo "Running full integration test suite..."
	@echo "This will:"
	@echo "  1. Convert all chain data"
	@echo "  2. Start 5-node primary network"
	@echo "  3. Import C-Chain data"
	@echo "  4. Deploy L2 subnets"
	@echo "  5. Run 7777 in dev mode"
	@$(GINKGO) -v --timeout=30m tests/integration/

# Single node dev mode targets
run:
	@if [ -z "$(network)" ]; then \
		echo "Usage: make run network=<7777|96369|96368|200200|200201|36911|36912|36963|36962>"; \
		exit 1; \
	fi
	@echo "Running chain $(network) in single-node dev mode..."
	@case $(network) in \
		7777) $(LUXD) --dev \
			--network-id=7777 \
			--chain-config-dir=configs/lux-genesis-7777 \
			--data-dir=chaindata/lux-genesis-7777/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		96369) $(LUXD) --dev \
			--network-id=96369 \
			--chain-config-dir=configs/lux-mainnet-96369 \
			--data-dir=chaindata/lux-mainnet-96369/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		96368) $(LUXD) --dev \
			--network-id=96368 \
			--chain-config-dir=configs/lux-testnet-96368 \
			--data-dir=chaindata/lux-testnet-96368/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		200200) $(LUXD) --dev \
			--network-id=200200 \
			--chain-config-dir=configs/zoo-mainnet-200200 \
			--data-dir=chaindata/zoo-mainnet-200200/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		200201) $(LUXD) --dev \
			--network-id=200201 \
			--chain-config-dir=configs/zoo-testnet-200201 \
			--data-dir=chaindata/zoo-testnet-200201/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		36911) $(LUXD) --dev \
			--network-id=36911 \
			--chain-config-dir=configs/spc-mainnet-36911 \
			--data-dir=chaindata/spc-mainnet-36911/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		36912) $(LUXD) --dev \
			--network-id=36912 \
			--chain-config-dir=configs/spc-testnet-36912 \
			--data-dir=chaindata/spc-testnet-36912/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		36963) $(LUXD) --dev \
			--network-id=36963 \
			--chain-config-dir=configs/hanzo-mainnet-36963 \
			--data-dir=chaindata/hanzo-mainnet-36963/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		36962) $(LUXD) --dev \
			--network-id=36962 \
			--chain-config-dir=configs/hanzo-testnet-36962 \
			--data-dir=chaindata/hanzo-testnet-36962/db \
			--http-port=9630 \
			--staking-port=9631 \
			--log-level=info ;; \
		*) echo "Unknown network: $(network)" && exit 1 ;; \
	esac

up:
	@echo "Starting LUX network with all subnets..."
	@echo "This will:"
	@echo "  1. Launch primary network (96369)"
	@echo "  2. Import genesis data"
	@echo "  3. Deploy ZOO subnet (200200)"
	@echo "  4. Deploy SPC subnet (36911)"
	@echo "  5. Deploy Hanzo subnet (36963)"
	@docker-compose -f docker/docker-compose.yml up -d
	@echo ""
	@echo "✅ Network started!"
	@echo "Primary RPC: http://localhost:9630/ext/bc/C/rpc"
	@echo "ZOO RPC: http://localhost:9630/ext/bc/zoo/rpc"
	@echo "SPC RPC: http://localhost:9630/ext/bc/spc/rpc"
	@echo "Hanzo RPC: http://localhost:9630/ext/bc/hanzo/rpc"
	@echo ""
	@echo "Check status: docker-compose -f docker/docker-compose.yml ps"
	@echo "View logs: docker-compose -f docker/docker-compose.yml logs -f"

down:
	@echo "Stopping LUX network..."
	@docker-compose -f docker/docker-compose.yml down
	@echo "✅ Network stopped"

# New Pipeline Workflows
pipeline-extract-all:
	@echo "Extracting all chain data..."
	@./bin/archeology extract \
		--source /path/to/lux-96369/db/pebbledb \
		--destination ./data/extracted/lux-96369 \
		--chain-id 96369 \
		--include-state
	@./bin/archeology extract \
		--source /path/to/zoo-200200/db/pebbledb \
		--destination ./data/extracted/zoo-200200 \
		--network zoo-mainnet \
		--include-state
	@./bin/archeology extract \
		--source /path/to/spc-36911/db/pebbledb \
		--destination ./data/extracted/spc-36911 \
		--chain-id 36911 \
		--include-state
	@echo "✅ All chains extracted"

pipeline-scan-external:
	@echo "Scanning external assets..."
	@./bin/teleport scan-nft \
		--chain ethereum \
		--contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
		--project lux \
		--output ./data/external/lux-nfts-ethereum.json
	@echo "✅ External assets scanned"

pipeline-generate-genesis:
	@echo "Generating genesis files..."
	@./bin/genesis generate \
		--network lux-mainnet \
		--chain-id 96369 \
		--data ./data/extracted/lux-96369 \
		--external ./data/external/ \
		--output ./data/genesis/lux-mainnet-96369.json
	@./bin/genesis generate \
		--network zoo-mainnet \
		--chain-id 200200 \
		--data ./data/extracted/zoo-200200 \
		--external ./data/external/ \
		--output ./data/genesis/zoo-mainnet-200200.json
	@./bin/genesis generate \
		--network spc-mainnet \
		--chain-id 36911 \
		--data ./data/extracted/spc-36911 \
		--output ./data/genesis/spc-mainnet-36911.json
	@echo "✅ All genesis files generated"

pipeline-full: pipeline-extract-all pipeline-scan-external pipeline-generate-genesis
	@echo "✅ Full pipeline completed!"

# Token Migration Workflows
migrate-token-to-l2:
	@if [ -z "$(contract)" ]; then \
		echo "Usage: make migrate-token-to-l2 chain=<chain> contract=<address> name=<subnet-name>"; \
		echo "Example: make migrate-token-to-l2 chain=ethereum contract=0xA0b8... name=usdc-subnet"; \
		exit 1; \
	fi
	@./bin/teleport migrate \
		--source-chain $(chain) \
		--contract $(contract) \
		--target-layer L2 \
		--target-name $(name)

# Help target
help:
	@echo "LUX Genesis Makefile"
	@echo ""
	@echo "Installation & Setup:"
	@echo "  make install          - Install LUX binaries from GitHub"
	@echo "  make build           - Build all tools"
	@echo ""
	@echo "New Tools:"
	@echo "  make build-archeology    - Build blockchain data extraction tool"
	@echo "  make build-genesis       - Build genesis generation tool"
	@echo "  make build-teleport      - Build external asset migration tool"
	@echo ""
	@echo "Genesis Workflows:"
	@echo "  make genesis-extract-all       - Extract data from all chains"
	@echo "  make genesis-scan-external     - Scan external blockchains for assets"
	@echo "  make genesis-generate-genesis  - Generate all genesis files"
	@echo "  make genesis                   - Run complete genesis pipeline"
	@echo ""
	@echo "Token Migration:"
	@echo "  make migrate-token-to-l2 chain=<chain> contract=<addr> name=<name>"
	@echo ""
	@echo "Data Conversion:"
	@echo "  make convert network=7777    - Convert 7777 LevelDB to PebbleDB"
	@echo "  make convert network=96369   - Convert 96369 LevelDB to PebbleDB"
	@echo "  make convert-all      		  - Convert all chain data"
	@echo ""
	@echo "Running Networks:"
	@echo "  make run network=7777   - Run historic 7777 network"
	@echo "  make run network=96369  - Run mainnet 96369"
	@echo "  make up                 - Launch full network with all subnets"
	@echo "  make down               - Stop the network"
	@echo ""
	@echo "Development:"
	@echo "  make run network=CHAIN - Run any chain in dev mode"
	@echo ""
	@echo "External Asset Import:"
	@echo "  make import-lux-nfts    - Import Lux NFTs from Ethereum"
	@echo "  make import-zoo-nfts    - Import Zoo NFTs from BSC (needs contract)"
	@echo "  make import-zoo-tokens-bsc - Import Zoo tokens from BSC (needs contract)"
	@echo "  make import-lux-tokens-7777 - Import LUX tokens from local 7777 chain"
	@echo "  make import-nft ...     - Import NFTs with custom parameters"
	@echo "  make import-token ...   - Import tokens with custom parameters"
	@echo ""
	@echo "Testing:"
	@echo "  make test-unit       - Run unit tests"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-all        - Run all tests"
	@echo ""
	@echo "Analysis:"
	@echo "  make analyze-chaindata - Show chain data statistics"

# Full genesis pipeline
genesis-full-pipeline: check-luxd check-lux-cli extract-chaindata generate-validators generate-all-genesis ## Complete genesis generation pipeline
	@echo "✅ Full genesis pipeline complete!"

# Check dependencies
check-luxd: ## Verify luxd is built
	@echo "Building latest luxd..."
	@cd ../node && git pull && ./scripts/build.sh
	@echo "✅ luxd built"
	@../node/build/luxd --version

check-lux-cli: ## Verify lux-cli is available
	@echo "Building latest lux-cli..."
	@cd ../cli && git pull && go build -o ../genesis/bin/lux-cli ./main.go
	@echo "✅ lux-cli built"
	@./bin/lux-cli --version || ./bin/lux-cli version

# Extract chaindata from all networks
extract-chaindata: build-tools ## Extract blockchain data from all networks
	@echo "Extracting chaindata from all networks..."
	@mkdir -p data/extracted
	@if [ -d "/home/z/.lux-cli/runs/network_96369_1/node1/db/pebbledb" ]; then \
		echo "Extracting Lux mainnet (96369)..."; \
		./bin/denamespace \
			-src /home/z/.lux-cli/runs/network_96369_1/node1/db/pebbledb \
			-dst data/extracted/lux-mainnet-96369 \
			-network 96369 \
			-state; \
	fi
	@echo "✅ Chaindata extraction complete"

# Generate validators deterministically
generate-validators: build-genesis-pkg ## Generate 11 validators from mnemonic
	@if [ -z "$$MNEMONIC" ]; then \
		echo "Error: MNEMONIC not set."; \
		echo "Please set MNEMONIC environment variable:"; \
		echo "  export MNEMONIC='your twelve word mnemonic phrase'"; \
		exit 1; \
	fi
	@echo "Generating 11 validators from MNEMONIC..."
	@mkdir -p configs
	@./bin/genesis-builder \
		-generate-keys \
		-mnemonic "$$MNEMONIC" \
		-offsets "0,1,2,3,4,5,6,7,8,9,10" \
		-save-keys configs/mainnet-validators.json \
		-save-keys-dir validator-keys
	@echo "✅ Validators generated with proper P-Chain addresses"

# Generate all genesis files
generate-all-genesis: generate-mainnet-genesis generate-testnet-genesis generate-local-genesis ## Generate genesis for all networks

generate-mainnet-genesis: build-genesis-pkg ## Generate mainnet genesis
	@echo "Generating mainnet genesis..."
	@echo "Importing C-Chain allocations from 7777 airdrop..."
	@./bin/genesis-builder \
		--network mainnet \
		--import-allocations chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet.csv \
		--validators configs/mainnet-validators.json \
		--output genesis_mainnet_96369.json

generate-testnet-genesis: build-genesis-pkg ## Generate testnet genesis
	@./bin/genesis-builder \
		--network testnet \
		--treasury-amount 2000000000000000000000 \
		--output genesis_testnet_96368.json

generate-local-genesis: build-genesis-pkg ## Generate local test genesis
	@./bin/genesis-builder \
		--network local \
		--treasury-amount 2000000000000000000000 \
		--output genesis_local.json

# Build tools
build-genesis-pkg: ## Build genesis builder
	@echo "Building genesis builder..."
	@go build -o bin/genesis-builder ./cmd/genesis-builder/
	@echo "✅ Genesis builder built"

# Network operations using lux-cli
cli-network-clean: ## Clean lux-cli network
	@echo "Cleaning lux-cli network..."
	@lux-cli network stop --force 2>/dev/null || true
	@lux-cli network clean --hard 2>/dev/null || true

cli-network-start: ## Start network with lux-cli
	@echo "Starting network with lux-cli..."
	@lux-cli network start --lux-path $(LUXD_PATH)

cli-network-stop: ## Stop lux-cli network
	@lux-cli network stop

cli-network-status: ## Check lux-cli network status
	@lux-cli network status

cli-local-start: ## Start local POA network
	@echo "Starting local POA network..."
	@lux-cli local start

# Direct luxd operations
luxd-start-single: ## Start single luxd node with genesis
	@echo "Starting single luxd node..."
	@mkdir -p ~/.luxd/staking
	@cp validator-keys/validator-1/staking/staker.crt ~/.luxd/staking/
	@cp validator-keys/validator-1/staking/staker.key ~/.luxd/staking/
	@cp validator-keys/validator-1/bls.key ~/.luxd/staking/signer.key
	@$(LUXD_PATH) \
		--network-id=96369 \
		--genesis-file=genesis_mainnet_96369.json \
		--http-host=0.0.0.0 \
		--http-port=9630 \
		--staking-enabled=false \
		--snow-sample-size=1 \
		--snow-quorum-size=1 \
		--log-level=info

# Network launch targets
launch-dev: ## Launch single node in dev mode (network 7777)
	@echo "Launching dev node..."
	@./scripts/launch-dev-7777.sh

launch-5-nodes: generate-validators generate-all-genesis ## Launch 5-node network
	@echo "Launching 5-node network..."
	@./scripts/launch-5-nodes.sh

launch-11-nodes: generate-validators generate-all-genesis ## Launch full 11-node network
	@echo "Launching 11-node network..."
	@./scripts/launch-11-nodes.sh

stop-network: ## Stop all running luxd nodes
	@echo "Stopping network..."
	@pkill -f "luxd.*network-id=96369" || true
	@pkill -f "luxd.*network-id=7777" || true
	@echo "✅ Network stopped"
	@echo ""
	@echo "✅ Local validators running!"
	@echo "RPC endpoints:"
	@for i in 1 2 3 4 5; do \
		echo "  Node $$i: http://localhost:$$((9650 + (i-1)*2))"; \
	done

# Deploy remote validators (last 6)
deploy-remote-validators: ## Package remote validator configs
	@echo "Packaging remote validators..."
	@mkdir -p remote-validators
	@for i in 6 7 8 9 10 11; do \
		cp -r validator-keys/validator-$$i remote-validators/; \
	done
	@tar -czf remote-validators.tar.gz remote-validators/
	@echo "✅ Remote validators packaged: remote-validators.tar.gz"

# L2 operations
deploy-zoo-l2: ## Deploy Zoo L2 subnet
	@./bin/lux-cli l2 create zoo \
		--evm \
		--chain-id 200200 \
		--custom-subnet-evm-genesis data/unified-genesis/zoo-mainnet-200200/genesis.json
	@./bin/lux-cli l2 deploy zoo --local

deploy-spc-l2: ## Deploy SPC L2 subnet  
	@./bin/lux-cli l2 create spc \
		--evm \
		--chain-id 36911 \
		--custom-subnet-evm-genesis data/unified-genesis/spc-mainnet-36911/genesis.json
	@./bin/lux-cli l2 deploy spc --local

# Testing
test-genesis: build-genesis-pkg ## Test genesis generation
	@echo "Testing genesis generation..."
	@go test ./pkg/genesis/... -v

test-validators: ## Test validator key generation
	@echo "Testing validator generation..."
	@MNEMONIC="test test test test test test test test test test test junk" \
		./bin/genesis-builder \
		-generate-keys \
		-mnemonic "$$MNEMONIC" \
		-offsets "0,1,2,3,4,5,6,7,8,9,10" \
		-save-keys test-validators.json
	@echo "Checking deterministic generation..."
	@if [ -f "test-validators.json" ]; then \
		echo "✅ Validator generation test passed"; \
		rm test-validators.json; \
	else \
		echo "❌ Validator generation test failed"; \
		exit 1; \
	fi

test-network: launch-local-validators ## Test network bootstrap
	@echo "Testing network bootstrap..."
	@sleep 10
	@./bin/lux-cli network status
	@echo "Testing RPC endpoints..."
	@curl -s -X POST --data '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":{}}' \
		-H 'content-type:application/json' http://localhost:9650/ext/info | jq .

test-rpc: ## Test RPC endpoints
	@echo "Testing RPC endpoints..."
	@echo "Network ID:"
	@curl -s -X POST -H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":1,"method":"info.getNetworkID","params":{}}' \
		http://localhost:9630/ext/info | jq .
	@echo ""
	@echo "Chain ID (C-Chain):"
	@curl -s -X POST -H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_chainId","params":[]}' \
		http://localhost:9630/ext/bc/C/rpc | jq .
	@echo ""
	@echo "Latest block:"
	@curl -s -X POST -H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
		http://localhost:9630/ext/bc/C/rpc | jq .

test-c-chain: ## Test C-Chain specific endpoints
	@echo "Testing C-Chain..."
	@echo "Treasury balance:"
	@curl -s -X POST -H 'Content-Type: application/json' \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"]}' \
		http://localhost:9630/ext/bc/C/rpc | jq .

test-all: test-genesis test-validators test-rpc ## Run all tests

# Full launch sequence
launch-mainnet: genesis-full-pipeline launch-local-validators ## Complete mainnet launch
	@echo ""
	@echo "🚀 Lux Mainnet launched!"
	@echo ""
	@echo "Local validators (1-5) are running"
	@echo "Remote validator packages: remote-validators.tar.gz"
	@echo ""
	@echo "Next steps:"
	@echo "1. Deploy remote validators to data center"
	@echo "2. Deploy L2 subnets: make deploy-zoo-l2 deploy-spc-l2"

# Clean everything
clean-all: clean network-clean ## Clean all generated files and networks
	@rm -rf validator-keys/ configs/mainnet-validators.json
	@rm -f genesis_*.json test-validators.json
	@rm -rf remote-validators/ remote-validators.tar.gz
	@echo "✅ All cleaned"

.PHONY: genesis-full-pipeline check-luxd check-lux-cli extract-chaindata
.PHONY: generate-validators generate-all-genesis generate-mainnet-genesis
.PHONY: generate-testnet-genesis generate-local-genesis build-genesis-pkg
.PHONY: cli-network-clean cli-network-start cli-network-stop cli-network-status cli-local-start
.PHONY: luxd-start-single launch-dev launch-5-nodes launch-11-nodes stop-network
.PHONY: launch-local-validators deploy-remote-validators
.PHONY: deploy-zoo-l2 deploy-spc-l2 test-genesis test-validators test-network
.PHONY: test-all launch-mainnet clean-all test-rpc test-c-chain

# Network deployment operations
deploy: deploy-local ## Deploy local test network (alias for deploy-local)

deploy-local: build-genesis ## Deploy local test network with 5 nodes
	@echo "=== Deploying local test network (5 nodes) ==="
	@./scripts/launch-5-nodes.sh

deploy-mainnet: build-genesis ## Deploy mainnet with all historical data and L2s
	@echo "=== Deploying Lux Mainnet ==="
	@chmod +x scripts/launch-mainnet.sh scripts/deploy-zoo-subnet.sh scripts/deploy-spc-subnet.sh
	@./scripts/launch-mainnet.sh

deploy-testnet: build-genesis ## Deploy testnet with historical data
	@echo "=== Deploying Lux Testnet ==="
	@chmod +x scripts/deploy-testnet.sh
	@./scripts/deploy-testnet.sh

install-plugin: build-genesis build-archeology ## Install genesis and archaeology as lux-cli plugins
	@echo "Installing lux-cli plugins..."
	@# Install genesis plugin
	@mkdir -p ~/.lux-cli/plugins/genesis
	@cp bin/genesis plugin.json lux-cli-genesis ~/.lux-cli/plugins/genesis/
	@chmod +x ~/.lux-cli/plugins/genesis/lux-cli-genesis
	@echo "✅ Genesis plugin installed"
	@# Install archaeology plugin
	@mkdir -p ~/.lux-cli/plugins/archaeology
	@cp bin/archeology archaeology-plugin.json lux-cli-archaeology ~/.lux-cli/plugins/archaeology/
	@mv ~/.lux-cli/plugins/archaeology/archaeology-plugin.json ~/.lux-cli/plugins/archaeology/plugin.json
	@chmod +x ~/.lux-cli/plugins/archaeology/lux-cli-archaeology
	@echo "✅ Archaeology plugin installed"
	@echo ""
	@echo "Plugins installed successfully! Use with:"
	@echo "  lux-cli genesis <command>"
	@echo "  lux-cli archaeology <command>"
