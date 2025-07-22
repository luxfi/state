#!/bin/bash
# Comprehensive Zoo ecosystem analysis using archeology scanners

set -e

# Configuration
BSC_RPC="${BSC_RPC:-https://bsc-dataseed.binance.org/}"
ZOO_TOKEN="0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
EGG_NFT="0x5bb68cf06289d54efde25155c88003be685356a8"
EGG_PURCHASE_ADDR="0x28dad8427f127664365109c4a9406c8bc7844718"
DEAD_ADDR="0x000000000000000000000000000000000000dEaD"
OUTPUT_DIR="${1:-exports/zoo-analysis}"

# Block range (optional)
FROM_BLOCK="${FROM_BLOCK:-0}"
TO_BLOCK="${TO_BLOCK:-0}"

echo "=== Zoo Ecosystem Analysis ==="
echo "Output directory: $OUTPUT_DIR"
echo "BSC RPC: $BSC_RPC"
echo ""

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Step 1: Scan EGG NFT holders
echo "Step 1: Scanning EGG NFT holders..."
./bin/archeology scan-holders \
    --rpc "$BSC_RPC" \
    --contract "$EGG_NFT" \
    --type nft \
    --from-block "$FROM_BLOCK" \
    --to-block "$TO_BLOCK" \
    --output "$OUTPUT_DIR/egg_nft_holders.csv" \
    --output-json "$OUTPUT_DIR/egg_nft_holders.json" \
    --top 20 \
    --show-distribution

echo ""
echo "Step 2: Scanning ZOO transfers to EGG purchase address..."
./bin/archeology scan-transfers \
    --rpc "$BSC_RPC" \
    --token "$ZOO_TOKEN" \
    --target "$EGG_PURCHASE_ADDR" \
    --direction to \
    --from-block "$FROM_BLOCK" \
    --to-block "$TO_BLOCK" \
    --output "$OUTPUT_DIR/zoo_egg_purchases.csv" \
    --output-json "$OUTPUT_DIR/zoo_egg_purchases.json" \
    --show-balances

echo ""
echo "Step 3: Scanning ZOO burns to dead address..."
./bin/archeology scan-burns \
    --rpc "$BSC_RPC" \
    --token "$ZOO_TOKEN" \
    --burn-address "$DEAD_ADDR" \
    --from-block "$FROM_BLOCK" \
    --to-block "$TO_BLOCK" \
    --output "$OUTPUT_DIR/zoo_burns.csv" \
    --output-json "$OUTPUT_DIR/zoo_burns.json" \
    --summarize

# Step 4: Generate summary report
echo ""
echo "Step 4: Generating summary report..."
cat > "$OUTPUT_DIR/zoo_analysis_report.txt" << EOF
Zoo Ecosystem Analysis Report
============================
Generated: $(date)

Files Generated:
- egg_nft_holders.csv/json: Current EGG NFT holders
- zoo_egg_purchases.csv/json: ZOO transfers for EGG purchases
- zoo_burns.csv/json: ZOO burns to dead address

Key Addresses:
- ZOO Token: $ZOO_TOKEN
- EGG NFT: $EGG_NFT
- EGG Purchase: $EGG_PURCHASE_ADDR
- Burn Address: $DEAD_ADDR

Notes:
- Each EGG NFT represents 4,200,000 ZOO tokens
- Purchases should be multiples of 4.2M ZOO
- Burned ZOO needs to be tracked for mainnet delivery

Next Steps:
1. Cross-reference burns with Zoo mainnet (200200) balances
2. Identify burners who haven't received tokens on mainnet
3. Generate genesis allocations including burns
EOF

echo ""
echo "=== Analysis Complete ==="
echo "Results saved to: $OUTPUT_DIR"
echo ""
echo "Files created:"
ls -la "$OUTPUT_DIR"/*.csv "$OUTPUT_DIR"/*.json "$OUTPUT_DIR"/*.txt 2>/dev/null || true