# Consensus Parameter Tool

A sophisticated command-line tool for generating, validating, and analyzing Lux Network consensus parameters with built-in safety checks and optimization guidance.

## Features

- **Interactive Mode**: Guided parameter configuration with recommendations
- **Safety Analysis**: Comprehensive safety checks and production readiness validation
- **Parameter Guide**: Detailed explanations of all consensus parameters
- **Probability Analysis**: Calculate safety and liveness failure probabilities
- **Optimization Modes**: Pre-configured for latency, security, or throughput
- **Network-Aware**: Automatically adjusts parameters based on network characteristics

## Installation

```bash
cd genesis
go build -o bin/consensus ./cmd/consensus
```

## Usage

### Interactive Mode (Recommended for First-Time Users)

```bash
# Launch interactive configuration wizard
./bin/consensus -interactive
```

This will guide you through:
- Network size and characteristics
- Expected failure rates  
- Performance requirements
- Safety analysis and recommendations

### Generate Parameters from Preset

```bash
# Generate mainnet parameters with safety analysis
./bin/consensus -preset mainnet -output mainnet-consensus.json -summary -safety

# Generate testnet parameters  
./bin/consensus -preset testnet -output testnet-consensus.json -summary -safety

# Generate local network parameters
./bin/consensus -preset local -output local-consensus.json -summary -safety
```

### View Parameter Guide

```bash
# Show comprehensive parameter documentation
./bin/consensus -guide
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

## Safety Warnings and Best Practices

### Production Requirements

The tool enforces these safety requirements for production networks:

1. **Beta ≥ 4**: Minimum 4 consecutive rounds for adequate security
2. **AlphaConfidence ≥ 67% of K**: Classical BFT-level safety threshold
3. **Byzantine Tolerance ≤ 33%**: Network must tolerate less than 1/3 Byzantine nodes
4. **AlphaConfidence ≥ AlphaPreference**: Safety threshold must exceed liveness threshold

### Common Warning Scenarios

⚠️ **Low Fault Tolerance**
```
K=21, AlphaConfidence=20 → Can only tolerate 1 failure (4.8%)
Suggestion: Lower AlphaConfidence to 18 for better fault tolerance
```

⚠️ **Insufficient Beta**
```
Beta=3 → May compromise finality guarantees
Suggestion: Increase Beta to at least 4 for production use
```

⚠️ **Excessive Pipelining**
```
ConcurrentRepolls=20, Beta=10 → No benefit from excessive pipelining
Suggestion: Set ConcurrentRepolls=10 (same as Beta)
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
- `-interactive` - Run in interactive mode
- `-guide` - Show parameter guidance
- `-safety` - Perform safety analysis
- `-total-nodes` - Total nodes for safety analysis

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

## Network-Specific Recommendations

### Small Networks (5-10 nodes)
- Use K = total nodes for maximum security
- AlphaPreference ≈ 60-70% for good liveness
- AlphaConfidence ≈ 80% for strong safety
- Beta = 4-8 for quick finality

### Medium Networks (11-30 nodes)  
- K = total nodes (or slightly less for performance)
- AlphaPreference ≈ 67% (2/3 majority)
- AlphaConfidence ≈ 80-85%
- Beta = 8-20 based on security needs

### Large Networks (50+ nodes)
- K = 20-50 (sampling subset for scalability)
- AlphaPreference ≈ 70%
- AlphaConfidence ≈ 85%
- Beta = 10-30 for strong finality

### High-Performance Local Networks
- Maximize pipelining: ConcurrentRepolls = Beta
- Reduce Beta for faster finality (minimum 4)
- Increase MaxOutstandingItems for throughput
- Use shorter timeouts (but not less than 2× expected finality)

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

// Perform safety analysis
report := consensus.AnalyzeSafety(params, 21)
if report.Level >= consensus.SafetyCritical {
    log.Fatal("Parameters not safe:", report.Issues)
}

// Use the parameters in your node configuration
nodeConfig.ConsensusParams = params
```

## Understanding the Math

### Failure Probability
The probability of consensus failure decreases exponentially with Beta:
```
P(failure) ≈ (1 - AlphaConfidence/K)^Beta
```

For example, with K=21, AlphaConfidence=18, Beta=8:
- Single round failure: (3/21)^1 ≈ 14.3%
- Eight consecutive failures: (3/21)^8 ≈ 1.7×10^-7

### Expected Finality Time
```
Finality ≈ (Beta × NetworkLatency) / ConcurrentRepolls
```

With Beta=8, 50ms latency, ConcurrentRepolls=8:
- Finality ≈ (8 × 50ms) / 8 = 50ms (theoretical minimum)
- In practice, add processing overhead: ~100-200ms