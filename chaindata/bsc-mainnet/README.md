# BSC Mainnet Data

This directory contains cached data from Binance Smart Chain mainnet.

## Zoo Token Data

Placeholder for Zoo token burn accounts and holder data from BSC.

### Known Information
- Zoo token was deployed on BSC
- Some accounts performed burns (sent tokens to 0x0)
- These burn addresses need to be credited on the new X-Chain

## Data Files

- `metadata.json` - Network and fetch metadata
- `finalized_block_*.json` - Block data at time of fetch
- `zoo_burn_accounts.csv` - Burn account addresses and amounts (to be populated)
- `zoo_holders_snapshot.json` - Current holder snapshot (requires additional tooling)

## TODO

1. Obtain correct Zoo token contract address on BSC
2. Fetch all burn transactions
3. Calculate burn amounts per address
4. Import into X-Chain genesis allocations