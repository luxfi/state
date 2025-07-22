# Genesis Action Plan

Generated: Tue Jul 22 06:10:54 AM UTC 2025

## Completed Items ✓

1. **EGG NFT Holders Scanned**
   - Found 84 unique holders with 452 EGGs
   - Data saved to: egg_nft_holders.json
   - ZOO equivalent: 1898400000

2. **Bootstrap Validators Configured**
   - 11 validators with 1B LUX each
   - Config: chaindata/lux-mainnet-96369/bootnodes.json
   - 100-year vesting schedule

3. **Local Chaindata Available**
   - LUX 7777: chaindata/lux-genesis-7777
   - LUX 96369: chaindata/lux-mainnet-96369
   - ZOO 200200: chaindata/zoo-mainnet-200200

## Required Actions

### 1. Extract Local Chain Accounts
```bash
# Extract LUX 7777
./bin/archeology extract \
    --src chaindata/lux-genesis-7777/db/pebbledb \
    --dst data/extracted/lux-genesis-7777 \
    --chain-id 7777 \
    --include-state

# Extract LUX 96369
./bin/archeology extract \
    --src chaindata/lux-mainnet-96369/db/pebbledb \
    --dst data/extracted/lux-96369 \
    --chain-id 96369 \
    --include-state

# Extract ZOO 200200
./bin/archeology extract \
    --src chaindata/zoo-mainnet-200200/db/pebbledb \
    --dst data/extracted/zoo-200200 \
    --chain-id 200200 \
    --include-state
```

### 2. Get ZOO Burns Data
Options:
- Use BSC archive node with higher rate limits
- Use a paid BSC API service (Moralis, Covalent, etc.)
- Manually provide known burn addresses

### 3. Get LUX NFT Holders on Ethereum
```bash
# Need Ethereum RPC (Infura/Alchemy)
INFURA_KEY=YOUR_KEY ./bin/archeology scan-current-holders \
    --rpc https://mainnet.infura.io/v3/YOUR_KEY \
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
    --project lux \
    --output lux_nft_holders.csv
```

### 4. Generate X-Chain Genesis
Once all data is collected:
```bash
./bin/genesis generate \
    --network x-chain \
    --data ./data/extracted/ \
    --external ./exports/genesis-analysis-*/ \
    --validators chaindata/lux-mainnet-96369/bootnodes.json \
    --output ./genesis/x-chain-genesis.json
```

## Summary

- ✅ EGG NFT data collected (84 holders, 452 EGGs)
- ✅ Bootstrap validators configured (11 validators)
- ✅ Local chaindata available
- ⏳ Need to extract local chain accounts
- ⏳ Need BSC burns data (requires better RPC)
- ⏳ Need Ethereum NFT holder data
- ⏳ Final cross-reference and genesis generation

## Bootstrap Validator Addresses
0x9011E888251AB053B7bD1cdB598Db4f9DEd94714
0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59
0x8d5081153aE1cfb41f5c932fe0b6Beb7E159cF84
0xf8f12D0592e6d1bFe92ee16CaBCC4a6F26dAAe23
0xFb66808f708e1d4D7D43a8c75596e84f94e06806
0x313CF291c069C58D6bd61B0D672673462B8951bD
0xf7f52257a6143cE6BbD12A98eF2B0a3d0C648079
0xCA92ad0C91bd8DE640B9dAFfEB338ac908725142
0xB5B325df519eB58B7223d85aaeac8b56aB05f3d6
0xcf5288bEe8d8F63511C389D5015185FDEDe30e54
0x16204223fe4470f4B1F1dA19A368dC815736a3d7
