#!/bin/bash

# Use a valid BIP39 test mnemonic
export MNEMONIC="abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

echo "Testing with valid mnemonic..."
echo "Generating validators..."

make generate-validators

echo ""
echo "Generated validators:"
head -20 configs/mainnet-validators.json

echo ""
echo "Checking BLS key format:"
hexdump -C validator-keys/validator-1/bls.key | head -5