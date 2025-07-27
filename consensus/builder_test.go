// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"testing"
	"time"
)

func TestBuilder(t *testing.T) {
	t.Run("default builder", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		if err := params.Validate(); err != nil {
			t.Fatalf("Default params invalid: %v", err)
		}
	})

	t.Run("from preset", func(t *testing.T) {
		builder := NewBuilder()
		builder, err := builder.FromPreset(MainnetNetwork)
		if err != nil {
			t.Fatalf("FromPreset failed: %v", err)
		}
		params, err := builder.Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		if params.K != MainnetParams.K {
			t.Errorf("K mismatch: got %d, want %d", params.K, MainnetParams.K)
		}
	})

	t.Run("for node count", func(t *testing.T) {
		tests := []struct {
			nodeCount int
			expectedK int
		}{
			{5, 5},
			{11, 11},
			{21, 21},
			{30, 30},
			{40, 21}, // Should cap and calculate
			{100, 27}, // Should cap and calculate
		}

		for _, tt := range tests {
			t.Run(string(rune(tt.nodeCount))+" nodes", func(t *testing.T) {
				builder := NewBuilder()
				params, err := builder.ForNodeCount(tt.nodeCount).Build()
				if err != nil {
					t.Fatalf("Build failed: %v", err)
				}
				if params.K != tt.expectedK {
					t.Errorf("K = %d, want %d for %d nodes", params.K, tt.expectedK, tt.nodeCount)
				}
				// Verify quorums are reasonable
				if params.AlphaPreference <= params.K/2 {
					t.Errorf("AlphaPreference too low: %d/%d", params.AlphaPreference, params.K)
				}
				if params.AlphaConfidence < params.AlphaPreference {
					t.Errorf("AlphaConfidence < AlphaPreference: %d < %d", 
						params.AlphaConfidence, params.AlphaPreference)
				}
			})
		}
	})

	t.Run("with quorum percentages", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.
			WithSampleSize(20).
			WithQuorumPercentages(70, 85).
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		expectedPref := 14 // 70% of 20
		expectedConf := 17 // 85% of 20
		
		if params.AlphaPreference != expectedPref {
			t.Errorf("AlphaPreference = %d, want %d", params.AlphaPreference, expectedPref)
		}
		if params.AlphaConfidence != expectedConf {
			t.Errorf("AlphaConfidence = %d, want %d", params.AlphaConfidence, expectedConf)
		}
	})

	t.Run("with target finality", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.
			WithTargetFinality(1*time.Second, 50).
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		// With 50ms latency, we should get ~20 rounds max for 1s finality
		if params.Beta > 20 {
			t.Errorf("Beta too high for 1s target: %d", params.Beta)
		}
		
		// Should maximize pipelining
		if params.ConcurrentRepolls < params.Beta {
			t.Errorf("Should maximize pipelining: %d < %d", 
				params.ConcurrentRepolls, params.Beta)
		}
	})

	t.Run("optimize for latency", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.
			WithSampleSize(21).
			WithBeta(20).
			OptimizeForLatency().
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		// Should reduce Beta
		if params.Beta != 8 {
			t.Errorf("Beta should be reduced to 8, got %d", params.Beta)
		}
		
		// Should maximize pipelining
		if params.ConcurrentRepolls != params.Beta {
			t.Errorf("Should maximize pipelining: %d != %d", 
				params.ConcurrentRepolls, params.Beta)
		}
		
		// Should increase parallelism
		if params.OptimalProcessing < 20 {
			t.Errorf("OptimalProcessing too low: %d", params.OptimalProcessing)
		}
	})

	t.Run("optimize for security", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.
			WithSampleSize(21).
			WithBeta(10).
			OptimizeForSecurity().
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		// Should increase Beta
		if params.Beta < 20 {
			t.Errorf("Beta should be increased to at least 20, got %d", params.Beta)
		}
		
		// Should increase quorums
		prefPercent := float64(params.AlphaPreference) / float64(params.K) * 100
		confPercent := float64(params.AlphaConfidence) / float64(params.K) * 100
		
		if prefPercent < 75 {
			t.Errorf("AlphaPreference too low: %.1f%%", prefPercent)
		}
		if confPercent < 80 {
			t.Errorf("AlphaConfidence too low: %.1f%%", confPercent)
		}
	})

	t.Run("optimize for throughput", func(t *testing.T) {
		builder := NewBuilder()
		params, err := builder.
			WithBeta(10).
			OptimizeForThroughput().
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		// Should maximize parallelism
		if params.OptimalProcessing < 32 {
			t.Errorf("OptimalProcessing too low: %d", params.OptimalProcessing)
		}
		if params.MaxOutstandingItems < 1024 {
			t.Errorf("MaxOutstandingItems too low: %d", params.MaxOutstandingItems)
		}
		
		// Should reduce processing timeout
		if params.MaxItemProcessingTime > 5*time.Second {
			t.Errorf("MaxItemProcessingTime too high: %v", params.MaxItemProcessingTime)
		}
	})

	t.Run("auto adjustment", func(t *testing.T) {
		builder := NewBuilder()
		// Set invalid values that should be auto-adjusted
		params, err := builder.
			WithSampleSize(10).
			WithQuorums(15, 8). // Both invalid
			Build()
		if err != nil {
			t.Fatalf("Build failed: %v", err)
		}
		
		// Should auto-adjust to valid values
		if params.AlphaPreference > params.K {
			t.Errorf("AlphaPreference not adjusted: %d > %d", 
				params.AlphaPreference, params.K)
		}
		if params.AlphaConfidence < params.AlphaPreference {
			t.Errorf("AlphaConfidence not adjusted: %d < %d", 
				params.AlphaConfidence, params.AlphaPreference)
		}
	})
}