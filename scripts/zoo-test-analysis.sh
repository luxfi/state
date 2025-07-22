#!/bin/bash
# Test Zoo analysis with limited block range

echo "Running Zoo analysis test with limited block range..."
echo "This will scan only 1000 blocks for testing"

./bin/teleport zoo-full-analysis \
    --bsc-rpc https://bsc-dataseed.binance.org/ \
    --from-block 20000000 \
    --to-block 20001000 \
    --output-dir exports/zoo-analysis-test

echo "Test analysis complete!"
echo "Check exports/zoo-analysis-test/ for results"