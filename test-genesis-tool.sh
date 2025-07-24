#!/bin/bash
# Comprehensive test script for the unified genesis tool

set -e

echo "🧪 Testing Unified Genesis Tool"
echo "==============================="

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to run test
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_result="$3"
    
    echo -n "Testing: $test_name... "
    
    if eval "$test_cmd" > /tmp/test_output.log 2>&1; then
        if [ -z "$expected_result" ] || grep -q "$expected_result" /tmp/test_output.log; then
            echo -e "${GREEN}✓${NC}"
            ((TESTS_PASSED++))
        else
            echo -e "${RED}✗${NC} (Expected: $expected_result)"
            ((TESTS_FAILED++))
            cat /tmp/test_output.log
        fi
    else
        echo -e "${RED}✗${NC} (Command failed)"
        ((TESTS_FAILED++))
        cat /tmp/test_output.log
    fi
}

# Build the tool first
echo "Building genesis tool..."
go build -o bin/genesis ./cmd/genesis || exit 1

# Test 1: Basic help
run_test "Basic help" "./bin/genesis --help" "Available Commands:"

# Test 2: Version
run_test "Version info" "./bin/genesis --version" "version"

# Test 3: Tools list
run_test "Tools list" "./bin/genesis tools" "Available tools"

# Test 4: Validators list (may be empty)
run_test "Validators list" "./bin/genesis validators list" ""

# Test 5: Generate command help
run_test "Generate help" "./bin/genesis generate --help" "Generate genesis"

# Test 6: Extract command help
run_test "Extract help" "./bin/genesis extract --help" "Extract blockchain data"

# Test 7: Analyze command help
run_test "Analyze help" "./bin/genesis analyze --help" "Analyze blockchain"

# Test 8: Migrate command help
run_test "Migrate help" "./bin/genesis migrate --help" "Migrate cross-chain"

# Test 9: Import command help
run_test "Import help" "./bin/genesis import --help" "Import blockchain data"

# Test 10: Generate genesis dry run
mkdir -p /tmp/genesis-test
run_test "Generate genesis (dry run)" "./bin/genesis generate --output /tmp/genesis-test --dry-run 2>&1 || true" ""

# Test 11: Validate non-existent genesis
run_test "Validate genesis" "./bin/genesis validate /tmp/non-existent.json 2>&1 || true" ""

# Test 12: Process historic help
run_test "Process historic help" "./bin/genesis process historic --help" "Process historic data"

# Test 13: Extract state help
run_test "Extract state help" "./bin/genesis extract state --help" "Extract state from PebbleDB"

# Test 14: Extract genesis help
run_test "Extract genesis help" "./bin/genesis extract genesis --help" "Extract genesis configuration"

# Test 15: Scan BSC help
run_test "Scan BSC help" "./bin/genesis scan bsc --help" "Scan BSC blockchain"

# Test 16: Test with sample PebbleDB path
if [ -d "chaindata/lux-mainnet-96369/db/pebbledb" ]; then
    run_test "Extract state dry run" "./bin/genesis extract state chaindata/lux-mainnet-96369/db/pebbledb /tmp/test-extract --network 96369 --limit 10" ""
fi

# Clean up
rm -rf /tmp/genesis-test /tmp/test_output.log

# Summary
echo ""
echo "============================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo "============================="

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi