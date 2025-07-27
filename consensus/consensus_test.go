// Copyright (C) 2025, Lux Industries Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package consensus

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPresetParameters(t *testing.T) {
	tests := []struct {
		name    string
		network NetworkType
		params  Parameters
	}{
		{
			name:    "mainnet",
			network: MainnetNetwork,
			params:  MainnetParams,
		},
		{
			name:    "testnet",
			network: TestnetNetwork,
			params:  TestnetParams,
		},
		{
			name:    "local",
			network: LocalNetwork,
			params:  LocalParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test GetPreset
			params, err := GetPreset(tt.network)
			if err != nil {
				t.Fatalf("GetPreset failed: %v", err)
			}

			// Validate parameters
			if err := params.Validate(); err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			// Test JSON serialization
			data, err := params.ToJSON()
			if err != nil {
				t.Fatalf("ToJSON failed: %v", err)
			}

			// Test JSON deserialization
			loaded, err := FromJSON(data)
			if err != nil {
				t.Fatalf("FromJSON failed: %v", err)
			}

			// Verify loaded params match original
			if loaded.K != params.K {
				t.Errorf("K mismatch: got %d, want %d", loaded.K, params.K)
			}
			
			// Test summary generation
			summary := params.Summary()
			if summary == "" {
				t.Error("Summary should not be empty")
			}
			
			t.Logf("Network: %s\n%s", tt.name, summary)
		})
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  Parameters
		wantErr bool
	}{
		{
			name: "valid params",
			params: Parameters{
				K:                     11,
				AlphaPreference:       8,
				AlphaConfidence:       9,
				Beta:                  10,
				ConcurrentRepolls:     10,
				OptimalProcessing:     10,
				MaxOutstandingItems:   256,
				MaxItemProcessingTime: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "invalid K",
			params: Parameters{
				K:                     0,
				AlphaPreference:       8,
				AlphaConfidence:       9,
				Beta:                  10,
				ConcurrentRepolls:     10,
				OptimalProcessing:     10,
				MaxOutstandingItems:   256,
				MaxItemProcessingTime: 10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "AlphaPreference > K",
			params: Parameters{
				K:                     11,
				AlphaPreference:       12,
				AlphaConfidence:       9,
				Beta:                  10,
				ConcurrentRepolls:     10,
				OptimalProcessing:     10,
				MaxOutstandingItems:   256,
				MaxItemProcessingTime: 10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "AlphaConfidence < AlphaPreference",
			params: Parameters{
				K:                     11,
				AlphaPreference:       9,
				AlphaConfidence:       8,
				Beta:                  10,
				ConcurrentRepolls:     10,
				OptimalProcessing:     10,
				MaxOutstandingItems:   256,
				MaxItemProcessingTime: 10 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateExpectedFinality(t *testing.T) {
	params := Parameters{
		K:                 11,
		AlphaPreference:   8,
		AlphaConfidence:   9,
		Beta:              10,
		ConcurrentRepolls: 10,
	}

	tests := []struct {
		name         string
		networkLatMs int
		minExpected  time.Duration
		maxExpected  time.Duration
	}{
		{
			name:         "low latency",
			networkLatMs: 50,
			minExpected:  400 * time.Millisecond,
			maxExpected:  600 * time.Millisecond,
		},
		{
			name:         "high latency",
			networkLatMs: 100,
			minExpected:  800 * time.Millisecond,
			maxExpected:  1200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finality := params.CalculateExpectedFinality(tt.networkLatMs)
			if finality < tt.minExpected || finality > tt.maxExpected {
				t.Errorf("Expected finality between %v and %v, got %v",
					tt.minExpected, tt.maxExpected, finality)
			}
		})
	}
}

func TestCalculateFaultTolerance(t *testing.T) {
	tests := []struct {
		name               string
		params             Parameters
		wantPrefTolerance  int
		wantConfTolerance  int
	}{
		{
			name:               "mainnet",
			params:             MainnetParams,
			wantPrefTolerance:  6, // 20 - 14
			wantConfTolerance:  6, // 20 - 14
		},
		{
			name:               "testnet",
			params:             TestnetParams,
			wantPrefTolerance:  3, // 10 - 7
			wantConfTolerance:  3, // 10 - 7
		},
		{
			name:               "local",
			params:             LocalParams,
			wantPrefTolerance:  2, // 5 - 3
			wantConfTolerance:  1, // 5 - 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefTolerance, confTolerance := tt.params.CalculateFaultTolerance()
			if prefTolerance != tt.wantPrefTolerance {
				t.Errorf("Preference tolerance = %d, want %d", prefTolerance, tt.wantPrefTolerance)
			}
			if confTolerance != tt.wantConfTolerance {
				t.Errorf("Confidence tolerance = %d, want %d", confTolerance, tt.wantConfTolerance)
			}
		})
	}
}

func TestJSONSerialization(t *testing.T) {
	original := Parameters{
		K:                     21,
		AlphaPreference:       13,
		AlphaConfidence:       18,
		Beta:                  8,
		ConcurrentRepolls:     8,
		OptimalProcessing:     10,
		MaxOutstandingItems:   369,
		MaxItemProcessingTime: 96369 * time.Nanosecond,
	}

	// Serialize to JSON
	data, err := json.MarshalIndent(original, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Deserialize from JSON
	var loaded Parameters
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Verify all fields match
	if loaded.K != original.K {
		t.Errorf("K mismatch: got %d, want %d", loaded.K, original.K)
	}
	if loaded.AlphaPreference != original.AlphaPreference {
		t.Errorf("AlphaPreference mismatch: got %d, want %d", loaded.AlphaPreference, original.AlphaPreference)
	}
	if loaded.AlphaConfidence != original.AlphaConfidence {
		t.Errorf("AlphaConfidence mismatch: got %d, want %d", loaded.AlphaConfidence, original.AlphaConfidence)
	}
	if loaded.Beta != original.Beta {
		t.Errorf("Beta mismatch: got %d, want %d", loaded.Beta, original.Beta)
	}
	if loaded.ConcurrentRepolls != original.ConcurrentRepolls {
		t.Errorf("ConcurrentRepolls mismatch: got %d, want %d", loaded.ConcurrentRepolls, original.ConcurrentRepolls)
	}
	if loaded.OptimalProcessing != original.OptimalProcessing {
		t.Errorf("OptimalProcessing mismatch: got %d, want %d", loaded.OptimalProcessing, original.OptimalProcessing)
	}
	if loaded.MaxOutstandingItems != original.MaxOutstandingItems {
		t.Errorf("MaxOutstandingItems mismatch: got %d, want %d", loaded.MaxOutstandingItems, original.MaxOutstandingItems)
	}
	if loaded.MaxItemProcessingTime != original.MaxItemProcessingTime {
		t.Errorf("MaxItemProcessingTime mismatch: got %v, want %v", loaded.MaxItemProcessingTime, original.MaxItemProcessingTime)
	}
}