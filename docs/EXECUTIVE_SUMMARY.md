# Executive Summary

This document provides a high-level overview of the Lux Network 2025 genesis transition, including the retirement of chain 7777, the launch of chain 96369 as the permanent C-Chain, the optional L1 upgrade path for existing subnets, and the preservation of historical data for transparency.

## Background

- **Chain 7777 (Original Mainnet)**: Launched in 2023, this chain is now retired. All account balances and state data are fully preserved for auditability.
- **Chain 96369 (New C-Chain)**: Launched in 2024 to replace the legacy chain. Clean architecture without technical debt, using PebbleDB for high performance.
- **Subnet Sovereignty**: Existing L2 subnets (ZOO, SPC, Hanzo) can optionally upgrade to their own L1 while remaining interoperable with Lux L1.
- **Historical Preservation**: The full 7777 genesis and state are maintained for transparency and optional historical node deployment.

## Scope & Audience

This document is intended for:

- **Developers** integrating genesis into CI/CD pipelines.
- **Researchers** auditing the transition and chain history.
- **Community Members** verifying allocations and running nodes.

## Key Milestones

| Phase                                    | Description                                                  |
|------------------------------------------|--------------------------------------------------------------|
| 7777 Genesis Preservation                | Retain original mainnet airdrop and allocations (chain 7777) |
| Denamespace & Extract C-Chain State      | Strip namespace prefixes and extract state for chain 96369   |
| Genesis Generation for Lux & Subnets     | Build final genesis for L1 and optional L1 subnets           |
| Network Launch (Mainnet & Testnet)       | Start Lux L1 mainnet (96369) and testnet (96368)             |
