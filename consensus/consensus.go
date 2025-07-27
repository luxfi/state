// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"encoding/json"
	"fmt"
	"time"
)

// Parameters defines the consensus parameters for Lux Network
type Parameters struct {
	// K is the number of nodes to query and sample in a round.
	K int `json:"k" yaml:"k"`
	
	// AlphaPreference is the vote threshold to change your preference.
	AlphaPreference int `json:"alphaPreference" yaml:"alphaPreference"`
	
	// AlphaConfidence is the vote threshold to increase your confidence.
	AlphaConfidence int `json:"alphaConfidence" yaml:"alphaConfidence"`
	
	// Beta is the number of consecutive successful queries required for finalization.
	Beta int `json:"beta" yaml:"beta"`
	
	// ConcurrentRepolls is the number of outstanding polls the engine will
	// target to have while there is something processing.
	ConcurrentRepolls int `json:"concurrentRepolls" yaml:"concurrentRepolls"`
	
	// OptimalProcessing is the number of items to process in parallel optimally.
	OptimalProcessing int `json:"optimalProcessing" yaml:"optimalProcessing"`
	
	// MaxOutstandingItems is the maximum number of consensus items that can be outstanding.
	MaxOutstandingItems int `json:"maxOutstandingItems" yaml:"maxOutstandingItems"`
	
	// MaxItemProcessingTime is the maximum time allowed for processing a single item.
	MaxItemProcessingTime time.Duration `json:"maxItemProcessingTime" yaml:"maxItemProcessingTime"`
}

// NetworkType represents different network configurations
type NetworkType string

const (
	MainnetNetwork NetworkType = "mainnet"
	TestnetNetwork NetworkType = "testnet"
	LocalNetwork   NetworkType = "local"
	CustomNetwork  NetworkType = "custom"
)

// Preset configurations for different networks
var (
	// MainnetParams are optimized for a global, decentralized network (21 nodes)
	MainnetParams = Parameters{
		K:                     21,  // Sample all 21 nodes
		AlphaPreference:       13,  // ~62% quorum - can tolerate up to 8 failures
		AlphaConfidence:       18,  // ~86% quorum - can tolerate up to 3 failures
		Beta:                  8,   // 8 rounds → 500ms finality
		ConcurrentRepolls:     8,   // Pipeline all 8 rounds
		OptimalProcessing:     10,
		MaxOutstandingItems:   369,
		MaxItemProcessingTime: 9630 * time.Millisecond, // 9.63 seconds timeout
	}

	// TestnetParams are balanced for testing with fewer nodes (11 nodes)
	TestnetParams = Parameters{
		K:                     11,  // Sample all 11 nodes
		AlphaPreference:       8,   // ~73% quorum - can tolerate up to 3 failures
		AlphaConfidence:       9,   // ~82% quorum - can tolerate up to 2 failures
		Beta:                  10,  // 10 rounds → 600ms finality
		ConcurrentRepolls:     10,  // Pipeline all 10 rounds
		OptimalProcessing:     10,
		MaxOutstandingItems:   256,
		MaxItemProcessingTime: 6900 * time.Millisecond, // 6.9 seconds timeout
	}

	// LocalParams are optimized for low-latency local networks (5 nodes)
	// Designed for 10Gbps networks with minimal latency
	LocalParams = Parameters{
		K:                     5,  // Sample all 5 nodes
		AlphaPreference:       4,  // 80% quorum - can tolerate 1 failure
		AlphaConfidence:       4,  // 80% quorum - can tolerate 1 failure
		Beta:                  4,  // 4 rounds → minimum latency
		ConcurrentRepolls:     4,  // Pipeline all 4 rounds
		OptimalProcessing:     32, // High parallelism for 10Gbps
		MaxOutstandingItems:   1024, // High throughput capacity
		MaxItemProcessingTime: 3690 * time.Millisecond, // 3.69 seconds timeout
	}

	// BenchmarkParams are for high-performance 10Gbps networks
	BenchmarkParams = Parameters{
		K:                     21,
		AlphaPreference:       13, // ~62% quorum
		AlphaConfidence:       18, // ~86% quorum
		Beta:                  8,  // ~500ms finality
		ConcurrentRepolls:     8,
		OptimalProcessing:     10,
		MaxOutstandingItems:   369,
		MaxItemProcessingTime: 96369 * time.Nanosecond, // ~96 microseconds
	}
)

// GetPreset returns preset parameters for a given network type
func GetPreset(network NetworkType) (Parameters, error) {
	switch network {
	case MainnetNetwork:
		return MainnetParams, nil
	case TestnetNetwork:
		return TestnetParams, nil
	case LocalNetwork:
		return LocalParams, nil
	default:
		return Parameters{}, fmt.Errorf("unknown network type: %s", network)
	}
}

// Validate checks if the parameters are valid
func (p *Parameters) Validate() error {
	if p.K <= 0 {
		return fmt.Errorf("K must be positive, got %d", p.K)
	}
	
	if p.AlphaPreference <= 0 || p.AlphaPreference > p.K {
		return fmt.Errorf("AlphaPreference must be between 1 and K (%d), got %d", p.K, p.AlphaPreference)
	}
	
	if p.AlphaConfidence <= 0 || p.AlphaConfidence > p.K {
		return fmt.Errorf("AlphaConfidence must be between 1 and K (%d), got %d", p.K, p.AlphaConfidence)
	}
	
	if p.AlphaConfidence < p.AlphaPreference {
		return fmt.Errorf("AlphaConfidence (%d) should not be less than AlphaPreference (%d)", 
			p.AlphaConfidence, p.AlphaPreference)
	}
	
	if p.Beta <= 0 {
		return fmt.Errorf("Beta must be positive, got %d", p.Beta)
	}
	
	if p.ConcurrentRepolls <= 0 {
		return fmt.Errorf("ConcurrentRepolls must be positive, got %d", p.ConcurrentRepolls)
	}
	
	if p.OptimalProcessing <= 0 {
		return fmt.Errorf("OptimalProcessing must be positive, got %d", p.OptimalProcessing)
	}
	
	if p.MaxOutstandingItems <= 0 {
		return fmt.Errorf("MaxOutstandingItems must be positive, got %d", p.MaxOutstandingItems)
	}
	
	if p.MaxItemProcessingTime <= 0 {
		return fmt.Errorf("MaxItemProcessingTime must be positive, got %v", p.MaxItemProcessingTime)
	}
	
	return nil
}

// CalculateExpectedFinality estimates the expected finality time
func (p *Parameters) CalculateExpectedFinality(networkLatencyMs int) time.Duration {
	// Formula: (Beta * (NetworkLatency + ProcessingTime)) / ConcurrentRepolls
	roundTime := time.Duration(networkLatencyMs) * time.Millisecond
	totalTime := time.Duration(p.Beta) * roundTime
	
	if p.ConcurrentRepolls > 1 {
		// Account for pipelining benefit
		pipelineFactor := float64(p.Beta) / float64(p.ConcurrentRepolls)
		if pipelineFactor < 1 {
			pipelineFactor = 1
		}
		totalTime = time.Duration(float64(totalTime) / pipelineFactor)
	}
	
	return totalTime
}

// CalculateFaultTolerance returns the fault tolerance for preference and confidence
func (p *Parameters) CalculateFaultTolerance() (preferenceToleranceNodes int, confidenceToleranceNodes int) {
	preferenceToleranceNodes = p.K - p.AlphaPreference
	confidenceToleranceNodes = p.K - p.AlphaConfidence
	return
}

// ToJSON converts parameters to JSON
func (p *Parameters) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON loads parameters from JSON
func FromJSON(data []byte) (*Parameters, error) {
	var p Parameters
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// Summary returns a human-readable summary of the parameters
func (p *Parameters) Summary() string {
	prefTolerance, confTolerance := p.CalculateFaultTolerance()
	finality50ms := p.CalculateExpectedFinality(50)
	finality100ms := p.CalculateExpectedFinality(100)
	
	return fmt.Sprintf(`Consensus Parameters Summary:
- Sample Size (K): %d nodes
- Preference Quorum: %d/%d (%.1f%%) - tolerates %d failures
- Confidence Quorum: %d/%d (%.1f%%) - tolerates %d failures  
- Finalization Rounds (Beta): %d
- Concurrent Polls: %d
- Expected Finality: %.2fs (50ms network), %.2fs (100ms network)
- Max Outstanding Items: %d
- Max Item Processing Time: %v`,
		p.K,
		p.AlphaPreference, p.K, float64(p.AlphaPreference)/float64(p.K)*100, prefTolerance,
		p.AlphaConfidence, p.K, float64(p.AlphaConfidence)/float64(p.K)*100, confTolerance,
		p.Beta,
		p.ConcurrentRepolls,
		finality50ms.Seconds(), finality100ms.Seconds(),
		p.MaxOutstandingItems,
		p.MaxItemProcessingTime)
}