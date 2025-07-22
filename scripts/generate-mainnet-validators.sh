#!/bin/bash
# Generate validator keys for Lux mainnet launch

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Lux Mainnet Validator Key Generation ===${NC}"
echo

# Check if genesis-builder exists
if [ ! -f "$PROJECT_ROOT/bin/genesis-builder" ]; then
    echo -e "${YELLOW}Building genesis-builder...${NC}"
    cd "$PROJECT_ROOT"
    go build -o bin/genesis-builder ./cmd/genesis-builder/
fi

# Create output directory
OUTPUT_DIR="$PROJECT_ROOT/validator-keys-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$OUTPUT_DIR"

echo -e "${GREEN}Generating 11 validator keys for mainnet...${NC}"
echo "Output directory: $OUTPUT_DIR"
echo

# Generate 11 validator keys
"$PROJECT_ROOT/bin/genesis-builder" \
    -generate-compatible \
    -account-count 11 \
    -save-keys "$OUTPUT_DIR/validators.json" \
    -save-keys-dir "$OUTPUT_DIR/keys"

echo
echo -e "${GREEN}✅ Validator keys generated successfully!${NC}"
echo

# Create a template for real validator configuration
cat > "$OUTPUT_DIR/mainnet-validators-real.json" << 'EOF'
[
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_1",
    "ethAddress": "0x1B475A4C983DfE4f32bbA4dE8DA8fd2c37f3A2A6",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_1",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_1",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 1 - Bootstrap Node 52.53.185.222"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_2",
    "ethAddress": "0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_2",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_2",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 2 - Bootstrap Node 52.53.185.223"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_3",
    "ethAddress": "0x8d5081153aE1cfb41f5c932fe0b6Beb7E159cF84",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_3",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_3",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 3 - Bootstrap Node 52.53.185.224"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_4",
    "ethAddress": "0xf8f12D0592e6d1bFe92ee16CaBCC4a6F26dAAe23",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_4",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_4",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 4 - Bootstrap Node 52.53.185.225"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_5",
    "ethAddress": "0xFb66808f708e1d4D7D43a8c75596e84f94e06806",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_5",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_5",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 5 - Bootstrap Node 52.53.185.226"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_6",
    "ethAddress": "0x313CF291c069C58D6bd61B0D672673462B8951bD",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_6",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_6",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 6 - Bootstrap Node 52.53.185.227"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_7",
    "ethAddress": "0xf7f52257a6143cE6BbD12A98eF2B0a3d0C648079",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_7",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_7",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 7 - Bootstrap Node 52.53.185.228"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_8",
    "ethAddress": "0xCA92ad0C91bd8DE640B9dAFfEB338ac908725142",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_8",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_8",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 8 - Bootstrap Node 52.53.185.229"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_9",
    "ethAddress": "0xB5B325df519eB58B7223d85aaeac8b56aB05f3d6",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_9",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_9",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 9 - Bootstrap Node 52.53.185.230"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_10",
    "ethAddress": "0xcf5288bEe8d8F63511C389D5015185FDEDe30e54",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_10",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_10",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 10 - Bootstrap Node 52.53.185.231"
  },
  {
    "nodeID": "REPLACE_WITH_GENERATED_NODEID_11",
    "ethAddress": "0x16204223fe4470f4B1F1dA19A368dC815736a3d7",
    "publicKey": "REPLACE_WITH_GENERATED_PUBLICKEY_11",
    "proofOfPossession": "REPLACE_WITH_GENERATED_POP_11",
    "weight": 1000000000000000000,
    "delegationFee": 20000,
    "_comment": "Validator 11 - Bootstrap Node 52.53.185.232"
  }
]
EOF

echo -e "${YELLOW}IMPORTANT NEXT STEPS:${NC}"
echo "1. The validator keys have been generated in: $OUTPUT_DIR"
echo "2. Each validator's staking keys are in: $OUTPUT_DIR/keys/validator-N/"
echo "3. Copy each validator's staking directory to their respective nodes"
echo "4. Update $OUTPUT_DIR/mainnet-validators-real.json with the generated NodeIDs and keys"
echo "5. Use the updated JSON file for genesis generation"
echo
echo -e "${RED}SECURITY WARNING:${NC}"
echo "- Keep the BLS private keys (bls.key) secure!"
echo "- The staking keys (staker.key) must be kept secure on each validator node"
echo "- Never share private keys or commit them to version control"
echo

# Show the generated validators
echo -e "${GREEN}Generated validators:${NC}"
cat "$OUTPUT_DIR/validators.json" | jq -r '.[] | "NodeID: \(.nodeID)"'