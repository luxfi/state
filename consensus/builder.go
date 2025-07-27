// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"fmt"
	"time"
)

// Builder provides a fluent interface for constructing consensus parameters
type Builder struct {
	params Parameters
}

// NewBuilder creates a new consensus parameters builder
func NewBuilder() *Builder {
	return &Builder{
		params: Parameters{
			// Start with sensible defaults
			K:                     11,
			AlphaPreference:       8,
			AlphaConfidence:       9,
			Beta:                  10,
			ConcurrentRepolls:     10,
			OptimalProcessing:     10,
			MaxOutstandingItems:   256,
			MaxItemProcessingTime: 10 * time.Second,
		},
	}
}

// FromPreset initializes the builder with a preset configuration
func (b *Builder) FromPreset(network NetworkType) (*Builder, error) {
	preset, err := GetPreset(network)
	if err != nil {
		return nil, err
	}
	b.params = preset
	return b, nil
}

// ForNodeCount configures parameters optimized for a specific node count
func (b *Builder) ForNodeCount(nodeCount int) *Builder {
	if nodeCount <= 0 {
		return b
	}
	
	// Set K to sample all nodes (or max reasonable sample)
	if nodeCount <= 30 {
		b.params.K = nodeCount
	} else {
		// For larger networks, sample a subset
		b.params.K = 20 + (nodeCount-30)/10
		if b.params.K > 50 {
			b.params.K = 50 // Cap at 50 for performance
		}
	}
	
	// Set quorum thresholds based on K
	// AlphaPreference: ~67% for good liveness
	b.params.AlphaPreference = (b.params.K * 2 / 3) + 1
	
	// AlphaConfidence: ~75-80% for strong safety
	b.params.AlphaConfidence = (b.params.K * 3 / 4) + 1
	
	// Adjust Beta based on network size
	// Larger networks need more rounds for security
	if nodeCount <= 5 {
		b.params.Beta = 11
	} else if nodeCount <= 11 {
		b.params.Beta = 20
	} else if nodeCount <= 21 {
		b.params.Beta = 31
	} else {
		b.params.Beta = 40
	}
	
	// Calculate optimal finality time
	expectedLatency := 50 // ms, assume decent network
	targetFinality := 10 * time.Second
	
	// Adjust concurrent repolls to meet target finality
	expectedRoundTime := time.Duration(expectedLatency) * time.Millisecond
	totalTime := time.Duration(b.params.Beta) * expectedRoundTime
	
	if totalTime > targetFinality {
		// Increase pipelining to reduce finality time
		b.params.ConcurrentRepolls = int(float64(b.params.Beta) * float64(expectedRoundTime) / float64(targetFinality))
		if b.params.ConcurrentRepolls < 4 {
			b.params.ConcurrentRepolls = 4
		}
		if b.params.ConcurrentRepolls > b.params.Beta {
			b.params.ConcurrentRepolls = b.params.Beta
		}
	}
	
	return b
}

// WithSampleSize sets the K parameter
func (b *Builder) WithSampleSize(k int) *Builder {
	b.params.K = k
	return b
}

// WithQuorums sets both quorum thresholds
func (b *Builder) WithQuorums(alphaPreference, alphaConfidence int) *Builder {
	b.params.AlphaPreference = alphaPreference
	b.params.AlphaConfidence = alphaConfidence
	return b
}

// WithQuorumPercentages sets quorums as percentages of K
func (b *Builder) WithQuorumPercentages(preferencePercent, confidencePercent float64) *Builder {
	b.params.AlphaPreference = int(float64(b.params.K) * preferencePercent / 100)
	if b.params.AlphaPreference < 1 {
		b.params.AlphaPreference = 1
	}
	
	b.params.AlphaConfidence = int(float64(b.params.K) * confidencePercent / 100)
	if b.params.AlphaConfidence < 1 {
		b.params.AlphaConfidence = 1
	}
	
	return b
}

// WithBeta sets the consecutive rounds threshold
func (b *Builder) WithBeta(beta int) *Builder {
	b.params.Beta = beta
	return b
}

// WithTargetFinality adjusts parameters to achieve target finality time
func (b *Builder) WithTargetFinality(target time.Duration, expectedNetworkLatencyMs int) *Builder {
	// Calculate how many rounds we can afford
	roundTime := time.Duration(expectedNetworkLatencyMs) * time.Millisecond
	maxRounds := int(target / roundTime)
	
	if maxRounds < 5 {
		maxRounds = 5 // Minimum for security
	}
	
	b.params.Beta = maxRounds
	
	// Maximize pipelining to achieve target
	b.params.ConcurrentRepolls = b.params.Beta
	if b.params.ConcurrentRepolls > 20 {
		b.params.ConcurrentRepolls = 20 // Reasonable upper limit
	}
	
	return b
}

// WithConcurrentRepolls sets the pipeline depth
func (b *Builder) WithConcurrentRepolls(repolls int) *Builder {
	b.params.ConcurrentRepolls = repolls
	return b
}

// WithOptimalProcessing sets the parallel processing parameter
func (b *Builder) WithOptimalProcessing(optimal int) *Builder {
	b.params.OptimalProcessing = optimal
	return b
}

// WithMaxOutstandingItems sets the max concurrent consensus items
func (b *Builder) WithMaxOutstandingItems(max int) *Builder {
	b.params.MaxOutstandingItems = max
	return b
}

// WithMaxItemProcessingTime sets the timeout for item processing
func (b *Builder) WithMaxItemProcessingTime(timeout time.Duration) *Builder {
	b.params.MaxItemProcessingTime = timeout
	return b
}

// OptimizeForLatency adjusts parameters for lowest latency
func (b *Builder) OptimizeForLatency() *Builder {
	// Reduce Beta for faster finality
	if b.params.Beta > 8 {
		b.params.Beta = 8
	}
	
	// Maximize pipelining
	b.params.ConcurrentRepolls = b.params.Beta
	
	// Increase parallelism
	b.params.OptimalProcessing = 20
	b.params.MaxOutstandingItems = 512
	
	return b
}

// OptimizeForSecurity adjusts parameters for maximum security
func (b *Builder) OptimizeForSecurity() *Builder {
	// Increase quorum thresholds
	b.params.AlphaPreference = (b.params.K * 3 / 4) + 1  // 75%+
	b.params.AlphaConfidence = (b.params.K * 4 / 5) + 1  // 80%+
	
	// Increase Beta for more confidence
	if b.params.Beta < 20 {
		b.params.Beta = 20
	}
	
	// Conservative processing limits
	b.params.MaxOutstandingItems = 128
	
	return b
}

// OptimizeForThroughput adjusts parameters for maximum throughput
func (b *Builder) OptimizeForThroughput() *Builder {
	// Maximize parallelism
	b.params.ConcurrentRepolls = b.params.Beta
	b.params.OptimalProcessing = 32
	b.params.MaxOutstandingItems = 1024
	
	// Reduce processing timeout for faster turnover
	b.params.MaxItemProcessingTime = 5 * time.Second
	
	return b
}

// Build validates and returns the constructed parameters
func (b *Builder) Build() (*Parameters, error) {
	// Auto-adjust if needed
	if b.params.AlphaPreference > b.params.K {
		b.params.AlphaPreference = b.params.K
	}
	if b.params.AlphaConfidence > b.params.K {
		b.params.AlphaConfidence = b.params.K
	}
	if b.params.AlphaConfidence < b.params.AlphaPreference {
		b.params.AlphaConfidence = b.params.AlphaPreference
	}
	
	if err := b.params.Validate(); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	
	return &b.params, nil
}