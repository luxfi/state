# Ethereum Mainnet Data

This directory contains cached data from Ethereum mainnet.

## Lux NFT Data

The Lux NFT holder data has been extracted and is available in:
- `../../../exports/lux-nft-analysis-20250723-014805/lux_nft_holders.csv`

Current holders (as of snapshot):
- 12 unique addresses
- 22 total NFTs
- Each NFT grants 1M LUX in validator rewards

## Data Structure

- `metadata.json` - Network and fetch metadata
- `finalized_block_*.json` - Block data at time of fetch
- `lux_nft_*.json/csv` - NFT holder data (if available)