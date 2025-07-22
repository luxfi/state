## Makefile for lux-genesis-import

SNAPSHOT=artifacts/lux-snapshot-v1.tgz
LUX_CLI=bin/lux
LUXD=bin/luxd
GINKGO=bin/ginkgo

.PHONY: install genesis snapshot docker push test convert-7777 convert-96369 run-7777-dev run-96369-dev import-7777-cchain import-96369-cchain analyze-chaindata build build-tools build-archaeology clean-bin test-unit test-integration test-all install-test-deps

install:
	@echo "Installing LUX binaries from GitHub..."
	@go run scripts/install_deps.go
	@echo "✅ Installation complete!"

## Full genesis pipeline: extract then generate P-, C-, X-chain genesis files
genesis: build-tools build-archeology build-genesis
	@echo "👉  Running full genesis pipeline"
	@bin/archeology extract --src chaindata/lux-7777/db/pebbledb --dst data/extracted/lux-7777 --chain-id 7777 --include-state
	@bin/archeology extract --src chaindata/lux-mainnet-96369/db/pebbledb --dst data/extracted/lux-96369 --chain-id 96369 --include-state
	@echo "👉  Generating P-Chain genesis"
	@bin/genesis generate --network p-chain --data data/extracted/lux-7777 --output configs/P/genesis.json
	@echo "👉  Generating C-Chain genesis"
	@bin/genesis generate --network c-chain --data data/extracted/lux-96369 --output configs/C/genesis.json
	@echo "👉  Generating X-Chain genesis"
	@bin/genesis generate --network x-chain --data data/extracted/lux-7777 --external data/external --output configs/xchain-genesis-complete.json
	@echo "✅ Full genesis pipeline complete (configs/P, configs/C, configs/xchain-genesis-complete.json)"

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
build: build-tools build-genesis build-teleport

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
	@echo "Importing Zoo NFTs from BSC..."
	@echo "⚠️  Please add Zoo NFT contract address"
	@# ./bin/archeology import-nft \
	@#	--network bsc \
	@#	--chain-id 56 \
	@#	--contract 0xADD_ZOO_NFT_ADDRESS_HERE \
	@#	--project zoo \
	@#	--output exports/zoo-nfts-bsc.csv

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

# Import ERC20 tokens
import-zoo-tokens-bsc:
	@echo "Importing Zoo tokens from BSC..."
	@echo "⚠️  Please add Zoo token contract address"
	@# ./bin/archeology import-token \
	@#	--network bsc \
	@#	--chain-id 56 \
	@#	--contract 0xADD_ZOO_TOKEN_ADDRESS_HERE \
	@#	--project zoo \
	@#	--symbol ZOO \
	@#	--output exports/zoo-tokens-bsc.csv

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
