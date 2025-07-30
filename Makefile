##########################################################################
## constants – *change only if your repo layout moves*
##########################################################################
REPO        := $(CURDIR)
GENESIS_BIN := $(REPO)/bin/genesis
LUXD_BIN    := $(REPO)/node/build/luxd
CHAIN_ID    := X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3
VM_ID       := mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6
TIP_H       := 1082780
TIP_HASH    := 0x32dede1fc8e0f11ecde12fb42aef7933fc6c5fcf863bc277b5eac08ae4d461f0
EVM_DB      := $(REPO)/runtime/lux-96369-vm-ready/evm/pebbledb
NODE_DIR    := $(REPO)/runtime/node-data
##########################################################################
.PHONY: deps
deps:                                        ## download dependencies
	go mod download

.PHONY: build
build: deps                                  ## compile both binaries
	go build -o $(GENESIS_BIN) ./cmd/genesis
	go build -o $(LUXD_BIN)     ./node/main

##########################################################################
## "one‑shot" pipeline called from inside the container
##########################################################################
.PHONY: migrate
migrate: build
	$(GENESIS_BIN) import subnet \
		chaindata/lux-mainnet-96369/db/pebbledb \
		$(EVM_DB)
	$(GENESIS_BIN) repair delete-suffix $(EVM_DB) 6e --prefix 68
	$(GENESIS_BIN) rebuild-canonical $(EVM_DB)
	$(GENESIS_BIN) copy-to-node \
		--chain-id $(CHAIN_ID) \
		--vm-id    $(VM_ID) \
		--evm-db   $(EVM_DB) \
		--node-dir $(NODE_DIR) \
		--height   $(TIP_H) \
		--hash     $(TIP_HASH)

.PHONY: launch
launch:
	$(LUXD_BIN) \
		--network-id 96369 \
		--data-dir $(NODE_DIR) \
		--dev --http-host 0.0.0.0 --http-port 9630 --log-level info

##########################################################################
## Top level target for CI / local dev
##########################################################################
.PHONY: docker-run
docker-run:                                ## build image, mount repo, run pipeline
	cd .. && docker build -f genesis/docker/Dockerfile -t lux-genesis .
	cd .. && docker run --rm -it \
		-v $(shell dirname $(REPO)):/workspace:ro \
		-v $(REPO)/runtime:/workspace/genesis/runtime \
		-p 9630:9630 \
		--name lux \
		lux-genesis

### Docker helpers ###############################################
.PHONY: docker-build
docker-build:              ## builds the image
	docker compose build

.PHONY: docker-up
docker-up: docker-build    ## rebuild and start
	docker compose up -d

.PHONY: docker-clean
docker-clean:              ## stop and wipe runtime dir
	docker compose down
	rm -rf runtime/*

### Helper targets ###############################################
.PHONY: clean
clean:
	@echo "🧹 cleaning runtime directory..."
	@rm -rf $(REPO)/runtime

### Monitoring and verification ###############################################
.PHONY: monitor
monitor:
	@echo "📊 Monitoring C-Chain..."
	docker compose exec lux luxd monitor chain C

.PHONY: console
console:
	@echo "💻 Opening JS console..."
	docker compose exec lux geth attach http://127.0.0.1:9630/ext/bc/C/rpc

.PHONY: snapshot
snapshot:
	@echo "📸 Creating DB snapshot..."
	docker compose exec lux tar -C /opt/lux/runtime -czf - db > snapshot-$(shell date +%Y%m%d-%H%M%S).tgz
	@echo "✅ Snapshot created"

.PHONY: check-height
check-height:
	@echo "🔍 Checking block height..."
	@curl -s -H "content-type: application/json" \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber","params":[]}' \
		http://localhost:8080/ | jq -r '.result' | xargs printf "Height: %d\n"

.PHONY: check-balance
check-balance:
	@echo "💰 Checking president balance..."
	@curl -s -H "content-type: application/json" \
		-d '{"jsonrpc":"2.0","id":1,"method":"eth_getBalance","params":["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714","latest"]}' \
		http://localhost:8080/ | jq -r '.result'

.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Main workflow:"
	@echo "  make migrate-and-launch  - Complete migration pipeline and launch luxd"
	@echo "  make docker-run         - Build and run Docker container"
	@echo "  make docker-clean       - Stop container and clean resources"
	@echo ""
	@echo "Build targets:"
	@echo "  make build              - Build genesis binary"
	@echo "  make docker-build       - Build Docker image"
	@echo ""
	@echo "Migration steps:"
	@echo "  make migrate            - Run full migration pipeline"
	@echo "  make launch             - Launch luxd with migrated data"
	@echo ""
	@echo "Monitoring:"
	@echo "  make monitor            - Monitor blockchain activity"
	@echo "  make console            - Open JS console"
	@echo "  make check-height       - Check current block height"
	@echo "  make check-balance      - Check president's balance"
	@echo "  make snapshot           - Create DB snapshot"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean              - Clean runtime directory"
	@echo "  make help               - Show this help message"