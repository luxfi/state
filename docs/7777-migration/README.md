# LUX 7777 Chain Migration

## Overview

Documentation for migrating the historic LUX 7777 blockchain data to the modern 96369 mainnet.

## Process Summary

The migration extracts ~888,834 blocks and 151 accounts from the December 2023 Avalanche unified database format, converting it to modern PebbleDB format suitable for import.

## Key Files

- [Extraction Guide](./extraction-guide.md) - Step-by-step extraction process
- [Airdrop Data](./airdrop-data.md) - Account balance information

## Statistics

- **Original Chain ID**: 7777
- **Blocks Extracted**: 888,834
- **Total Accounts**: 151
- **Database Size**: ~441MB (converted)
- **Conversion Time**: ~20 seconds