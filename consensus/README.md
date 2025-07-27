# Consensus Parameters for Lux Network

This document describes the consensus parameters used across different Lux Network deployments.

## Overview

Lux Network uses an advanced snowball consensus algorithm with configurable parameters that affect:
- Transaction finalization time
- Network security
- Validator sampling

## Parameter Definitions

### K (Sample Size)
- **Description**: Number of nodes to query and sample in each consensus round
- **Impact**: Higher K increases security but may increase latency

### AlphaPreference 
- **Description**: Vote threshold to change your preference
- **Impact**: Controls how quickly nodes change their preferred decision

### AlphaConfidence
- **Description**: Vote threshold to increase confidence in a decision
- **Impact**: Affects how quickly consensus confidence builds

### Beta (Consecutive Samples)
- **Description**: Number of consecutive successful queries required for finalization
- **Impact**: Higher Beta increases security but increases finalization time

### ConcurrentRepolls
- **Description**: Target number of outstanding polls while processing
- **Impact**: Affects network utilization and consensus speed

## Network Configurations

### Mainnet (21 nodes)
```json
{
  "k": 20,
  "alphaPreference": 14,
  "alphaConfidence": 14,
  "beta": 31,
  "concurrentRepolls": 4
}
```
**Expected Consensus Time**: ~9.63 seconds

### Testnet (11 nodes) 
```json
{
  "k": 10,
  "alphaPreference": 7,
  "alphaConfidence": 7,
  "beta": 20,
  "concurrentRepolls": 4
}
```
**Expected Consensus Time**: ~6.3 seconds

### Local/Dev (5 nodes)
```json
{
  "k": 5,
  "alphaPreference": 3,
  "alphaConfidence": 3,
  "beta": 11,
  "concurrentRepolls": 4
}
```
**Expected Consensus Time**: ~3.69 seconds

## Calculation Formula

Expected consensus time can be approximated by:
```
Time = (Beta * QueryTimeout) / ConcurrentRepolls
```

Where QueryTimeout is typically 1.5 seconds.

## Security Considerations

1. **K must be <= Total Validators**: Cannot sample more nodes than exist
2. **AlphaPreference/Confidence**: Should be > K/2 for security
3. **Beta**: Higher values increase security against adversarial nodes
4. **Network Size**: Larger networks can support higher K values

## Migration Notes

When migrating from avalanchego:
- Consensus parameters remain largely compatible
- BadgerDB replaces LevelDB/PebbleDB for improved performance
- BLS signatures are used for validator operations