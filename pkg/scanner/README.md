# Scanner Package

The `scanner` package provides reusable blockchain scanning utilities for analyzing tokens, NFTs, and cross-chain balances. These tools are designed to be modular and can be used for any EVM-compatible blockchain.

## Components

### 1. Token Burn Scanner

Scans for ERC20 token burns to specific addresses (like dead address).

```go
config := &scanner.TokenBurnScanConfig{
    RPC:          "https://bsc-rpc-endpoint",
    TokenAddress: "0x0a6045b79151d0a54dbd5227082445750a023af2",
    BurnAddress:  scanner.DeadAddress,
    FromBlock:    0,
    ToBlock:      0, // 0 = latest
}

scanner, err := scanner.NewTokenBurnScanner(config)
burns, err := scanner.ScanBurns()
burnsByAddress, err := scanner.ScanBurnsByAddress() // Aggregated by burner
```

### 2. Token Transfer Scanner

Scans for token transfers to/from specific addresses.

```go
config := &scanner.TokenTransferScanConfig{
    RPC:             "https://bsc-rpc-endpoint",
    TokenAddress:    "0x0a6045b79151d0a54dbd5227082445750a023af2",
    TargetAddresses: []string{"0x28dad8427f127664365109c4a9406c8bc7844718"},
    Direction:       "to", // "to", "from", or "both"
}

scanner, err := scanner.NewTokenTransferScanner(config)
transfers, err := scanner.ScanTransfers()
balanceChanges := scanner.GetBalanceChanges(transfers)
```

### 3. NFT Holder Scanner

Scans for current NFT holders by processing all Transfer events.

```go
config := &scanner.NFTHolderScanConfig{
    RPC:             "https://bsc-rpc-endpoint",
    ContractAddress: "0x5bb68cf06289d54efde25155c88003be685356a8",
    IncludeTokenIDs: true,
}

scanner, err := scanner.NewNFTHolderScanner(config)
holders, err := scanner.ScanHolders()
topHolders, err := scanner.GetTopHolders(20)
distribution := scanner.GetHolderDistribution(holders)
```

### 4. Cross-Chain Balance Scanner

Checks token balances across multiple chains.

```go
config := &scanner.CrossChainBalanceScanConfig{
    Chains: []scanner.ChainConfig{
        {
            Name:         "BSC",
            ChainID:      56,
            RPC:          "https://bsc-rpc",
            TokenAddress: "0x0a6045b79151d0a54dbd5227082445750a023af2",
        },
        {
            Name:         "Zoo Mainnet",
            ChainID:      200200,
            RPC:          "http://localhost:9650/ext/bc/zoo/rpc",
            TokenAddress: "0x...",
        },
    },
}

scanner, err := scanner.NewCrossChainBalanceScanner(config)
balances, err := scanner.ScanBalances(addresses)
comparisons, err := scanner.CompareBalances(addresses)
```

## Export Utilities

The package includes various export functions:

```go
// Export to CSV
scanner.ExportTokenBurnsToCSV(burns, "burns.csv")
scanner.ExportTokenTransfersToCSV(transfers, "transfers.csv")
scanner.ExportNFTHoldersToCSV(holders, "holders.csv", metadata)
scanner.ExportCrossChainBalancesToCSV(balances, "balances.csv")

// Export to JSON
scanner.ExportToJSON(data, "output.json")

// Generate reports
scanner.GenerateSummaryReport("report.txt", sections)
```

## CLI Usage Examples

### Scan Zoo token burns on BSC
```bash
teleport scan-token-burns \
  --token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
  --burn-address 0x000000000000000000000000000000000000dEaD \
  --summarize \
  --output burns.csv
```

### Scan EGG NFT holders
```bash
teleport scan-nft-holders \
  --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
  --top 20 \
  --show-distribution \
  --output egg-holders.csv
```

### Scan token transfers to purchase address
```bash
teleport scan-token-transfers \
  --token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
  --target 0x28dad8427f127664365109c4a9406c8bc7844718 \
  --direction to \
  --show-balances \
  --output purchases.csv
```

### Check cross-chain balances
```bash
teleport check-cross-chain-balances \
  --source-chain BSC --source-rpc https://bsc-rpc \
  --source-token 0x0a6045b79151d0a54dbd5227082445750a023af2 \
  --target-chain "Zoo Mainnet" --target-rpc http://localhost:9650/ext/bc/zoo/rpc \
  --target-token 0x... \
  --address-file burners.txt \
  --compare
```

### Complete Zoo ecosystem analysis
```bash
teleport zoo-full-analysis --output-dir ./zoo-analysis
```

## Common Use Cases

1. **Token Migration Analysis**: Check which addresses have burned tokens on source chain but haven't received on target chain
2. **NFT Distribution**: Analyze NFT holder distribution and identify large holders
3. **Purchase Tracking**: Track payments to specific addresses (like NFT purchase addresses)
4. **Burn Analysis**: Identify and quantify token burns for migration or other purposes
5. **Cross-Chain Verification**: Verify token balances across multiple chains

## Performance Considerations

- All scanners use chunked processing (default 5000 blocks per chunk)
- Progress indicators show scan status
- Supports custom block ranges to limit scan scope
- Handles RPC rate limiting with retries