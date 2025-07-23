# L2 Deployment and Migration Summary

## Current Status

### ZOO Token
- **Current Genesis Data**: 6,889 accounts on genesis EVM + 17,000+ on BSC
- **Documented Supply**: 2 Trillion ZOO

### SPC Token
- **Current Genesis Data**: 42 accounts on genesis EVM
- **Documented Supply**: 100 Billion SPC

## BSC Migration Details (ZOO)

### Contract Addresses
- **ZOO Token on BSC**: `0x0a6045b79151d0a54dbd5227082445750a023af2` (confirmed in zoo_migration_scanner.go)
- **EGG NFT on BSC**: `0x5bb68cf06289d54efde25155c88003be685356a8`
- **Egg Purchase Address**: `0x28dad8427f127664365109c4a9406c8bc7844718`
- **Dead/Burn Address**: `0x000000000000000000000000000000000000dEaD`

### Migration Rules
1. **ZOO Token Burns**: Users who burned ZOO on BSC get 1:1 credit on ZOO chain
2. **EGG NFT Holders**: Each egg = 4.2M ZOO allocation
3. **Lux Ethereum NFT**: `0x31e0f919c67cedd2bc3e294340dc900735810311` (for validator rights)

## Existing Data Files

### Already Collected
- `output/genesis-analysis-20250722-060502/`
  - `egg_nft_holders.csv` - All egg NFT holders
  - `egg_nft_holders.json` - Detailed egg holdings (1,440 total supply)
  - `zoo_xchain_genesis_allocations.json` - ZOO allocations based on eggs
  - `xchain_egg_allocations.json` - X-Chain egg allocations

### BSC Data Directory
- `chaindata/bsc-mainnet/`
  - `zoo_burn_accounts.csv` - Placeholder for burn data
  - `zoo_burn_events.json` - Placeholder for burn events
  - `zoo_burns_parsed.json` - Empty file

## Tools Already Built

### Go Tools
- `pkg/bridge/zoo_migration_scanner.go` - Scans BSC for burns and holders
- `pkg/bridge/egg_nft_analyzer.go` - Analyzes egg NFT holdings
- `pkg/bridge/zoo_egg_cross_reference.go` - Cross-references BSC with ZOO chain
- `cmd/teleport/commands/scan_egg_holders.go` - CLI for egg holder scanning
- `cmd/teleport/commands/zoo_migrate.go` - Migration command

### Scripts
- `scripts/analysis/zoo-analysis.sh` - Comprehensive ZOO analysis
- `scripts/analysis/get-egg-holders-efficient.py` - Python egg holder scanner
- `scripts/analysis/get-zoo-burns-efficient.py` - Python burn scanner
- `scripts/fetch-bsc-data.sh` - Fetches BSC data

## Next Steps

1. **Run existing tools to collect BSC data**:
   ```bash
   # Scan egg holders (already done - check exports)
   ./bin/teleport scan-egg-holders --output exports/egg-holders-update.csv

   # Scan ZOO burns
   ./bin/teleport scan-token-burns \
     --token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
     --burn-address 0x000000000000000000000000000000000000dEaD \
     --output exports/zoo-burns.csv
   ```

2. **Generate final ZOO genesis with BSC data**:
   ```bash
   ./bin/teleport zoo-migrate \
     --burns exports/zoo-burns.csv \
     --eggs exports/egg-holders.csv \
     --output genesis-zoo/final-genesis.json
   ```

3. **For SPC**: Use historic chain data from 2024 Lux Network edition.
