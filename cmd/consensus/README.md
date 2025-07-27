# Consensus Parameter Tool

A command-line tool for generating and validating Lux Network consensus parameters.

## Installation

```bash
cd genesis
go build -o bin/consensus ./cmd/consensus
```

## Usage

### Generate Parameters from Preset

```bash
# Generate mainnet parameters
./bin/consensus -preset mainnet -output mainnet-consensus.json -summary

# Generate testnet parameters  
./bin/consensus -preset testnet -output testnet-consensus.json -summary

# Generate local network parameters
./bin/consensus -preset local -output local-consensus.json -summary
```

### Generate Parameters for Node Count

```bash
# Generate optimized parameters for 21 nodes
./bin/consensus -nodes 21 -output consensus-21.json -summary

# Generate for 11 nodes with custom target finality
./bin/consensus -nodes 11 -target-finality 5s -output consensus-11.json -summary
```

### Custom Parameters

```bash
# Manually specify all parameters
./bin/consensus \
  -k 15 \
  -alpha-pref 10 \
  -alpha-conf 12 \
  -beta 20 \
  -concurrent 20 \
  -output custom.json \
  -summary
```

### Optimization Modes

```bash
# Optimize for lowest latency
./bin/consensus -preset mainnet -optimize latency -output low-latency.json

# Optimize for maximum security  
./bin/consensus -preset mainnet -optimize security -output high-security.json

# Optimize for highest throughput
./bin/consensus -preset mainnet -optimize throughput -output high-throughput.json
```

### Validate Existing Parameters

```bash
# Validate a consensus parameters file
./bin/consensus -validate consensus-params.json
```

## Examples

### 1. Generate Parameters for a 15-Node Network

```bash
./bin/consensus -nodes 15 -summary
```

Output:
```
{
  "k": 15,
  "alphaPreference": 11,
  "alphaConfidence": 12,
  "beta": 25,
  "concurrentRepolls": 4,
  "optimalProcessing": 10,
  "maxOutstandingItems": 256,
  "maxItemProcessingTime": "10s"
}

Consensus Parameters Summary:
- Sample Size (K): 15 nodes
- Preference Quorum: 11/15 (73.3%) - tolerates 4 failures
- Confidence Quorum: 12/15 (80.0%) - tolerates 3 failures  
- Finalization Rounds (Beta): 25
- Concurrent Polls: 4
- Expected Finality: 1.56s (50ms network), 3.12s (100ms network)
- Max Outstanding Items: 256
- Max Item Processing Time: 10s
```

### 2. Optimize for Sub-Second Finality

```bash
./bin/consensus \
  -nodes 10 \
  -target-finality 800ms \
  -network-latency 30 \
  -optimize latency \
  -summary
```

### 3. Create High-Security Parameters

```bash
./bin/consensus \
  -preset mainnet \
  -optimize security \
  -alpha-conf 18 \
  -beta 40 \
  -output high-security.json
```

## Parameter Reference

### Command Line Flags

- `-preset` - Use preset configuration (mainnet, testnet, local)
- `-nodes` - Number of nodes in the network
- `-k` - Sample size (number of nodes to query)
- `-alpha-pref` - Preference quorum threshold
- `-alpha-conf` - Confidence quorum threshold  
- `-beta` - Consecutive rounds threshold
- `-concurrent` - Concurrent repolls
- `-optimize` - Optimization mode (latency, security, throughput)
- `-output` - Output file path (JSON)
- `-summary` - Show parameter summary
- `-validate` - Validate parameters from file
- `-target-finality` - Target finality time (e.g., 500ms, 1s)
- `-network-latency` - Expected network latency in ms (default: 50)

### Parameter Guidelines

1. **K (Sample Size)**
   - Should be ≤ total number of nodes
   - Larger K = more security but higher overhead
   - Typical: 50-100% of validator set

2. **AlphaPreference**
   - Should be > K/2 for liveness
   - Typical: 67-75% of K
   - Lower = faster preference changes

3. **AlphaConfidence**  
   - Should be ≥ AlphaPreference
   - Typical: 75-85% of K
   - Higher = stronger safety guarantee

4. **Beta**
   - More rounds = stronger finality guarantee
   - Typical: 10-40 rounds
   - Trade-off between security and latency

5. **ConcurrentRepolls**
   - Should be ≤ Beta
   - Higher = better throughput via pipelining
   - Typical: 4-20

## Integration Example

```go
import "github.com/luxfi/genesis/consensus"

// Use builder to create custom parameters
builder := consensus.NewBuilder()
params, err := builder.
    ForNodeCount(21).
    WithTargetFinality(5 * time.Second, 50).
    OptimizeForSecurity().
    Build()

if err != nil {
    log.Fatal(err)
}

// Use the parameters in your node configuration
nodeConfig.ConsensusParams = params
```