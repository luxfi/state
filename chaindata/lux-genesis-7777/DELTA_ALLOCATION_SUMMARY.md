# Lux 7777 Delta Allocation Summary

## Overview
This analysis identified addresses that held LUX tokens on chain 7777 but were never migrated to the 96369 mainnet.

## Results
- **Total addresses on Lux 7777**: 151
- **Addresses that also exist on 96369**: 82
- **Addresses unique to 7777 (delta)**: 69
- **Total LUX amount to allocate**: 2,023,306,456.547631502151489258 LUX

## Files Generated
1. `$HOME/work/lux/genesis/chaindata/lux-genesis-7777/7777-delta-allocations-for-xchain.csv`
   - Contains the 69 addresses and their balances that need to be allocated on X-Chain

## Key Findings
These 69 addresses were overlooked during the migration from chain 7777 to chain 96369. They should be allocated on the X-Chain to ensure they receive their rightful LUX tokens.

## Notable Large Balances
- `0x542bc5d9068c80a44b89a8ddeeb190326bd0d051`: 1,000,000,001 LUX
- `0x91bb0ed981e9436580c17557fe95ce8602b417dd`: 1,000,181,073.78 LUX
- `0xd65d95cfaa0553eb5226e9901fc05ec47a083aa2`: 10,000,000 LUX
- `0xa105a413c349b175d8b445da814a6a766843f6ac`: 5,557,054.23 LUX
- `0xc19d8d7bc52c9f36da48b0a80eb8015c7add7916`: 2,456,893.01 LUX
- `0xa8e216efa99fab75120f07a021779f5437a4601e`: 1,292,580.35 LUX
- `0x1990da4015f39a6a9e0ba9d334e940d7a017cf39`: 1,208,216.08 LUX

## Verification Process
1. Extracted all addresses from Lux 7777 airdrop file (151 addresses)
2. Extracted all addresses from Lux 96369 mainnet file (~12M addresses)
3. Found addresses that exist in 7777 but NOT in 96369 (69 addresses)
4. Verified that none of these 69 addresses ever held tokens on 96369

## Next Steps
These allocations should be included in the X-Chain genesis to ensure these overlooked addresses receive their LUX tokens.