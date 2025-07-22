package config

import (
	"fmt"
	"time"
)

// NetworkID represents a Lux network identifier
type NetworkID uint32

// Predefined network IDs
const (
	MainnetID NetworkID = 96369
	TestnetID NetworkID = 96368
	LocalID   NetworkID = 12345
)

// NetworkConfig contains configuration for a specific network
type NetworkConfig struct {
	ID                  NetworkID
	Name                string
	HRP                 string // Human-readable part for addresses
	ChainID             uint64 // C-Chain ID
	StartTime           time.Time
	InitialStakeDuration time.Duration
	MinValidatorStake   uint64
	MinDelegatorStake   uint64
	IsL2                bool      // Is this an L2 subnet?
	ParentNetwork       string    // Parent network name (for L2s)
}

// L2 Network IDs
const (
	ZooMainnetID NetworkID = 200200
	ZooTestnetID NetworkID = 200201
	SPCMainnetID NetworkID = 36911
	HanzoMainnetID NetworkID = 36963
	HanzoTestnetID NetworkID = 36962
)

// Networks contains predefined network configurations
var Networks = map[string]*NetworkConfig{
	// Primary L1 Networks
	"mainnet": {
		ID:                  MainnetID,
		Name:                "Lux Mainnet",
		HRP:                 "lux",
		ChainID:             96369,
		StartTime:           time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), // Jan 1, 2020
		InitialStakeDuration: 100 * 365 * 24 * time.Hour, // 100 years
		MinValidatorStake:   2000000000000000, // 2M LUX (with 9 decimals)
		MinDelegatorStake:   25000000000,      // 25 LUX (with 9 decimals)
		IsL2:                false,
	},
	"testnet": {
		ID:                  TestnetID,
		Name:                "Lux Testnet",
		HRP:                 "test",
		ChainID:             96368,
		StartTime:           time.Now(),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   1000000000,  // 1 LUX (with 9 decimals)
		MinDelegatorStake:   1000000000,  // 1 LUX (with 9 decimals)
		IsL2:                false,
	},
	"local": {
		ID:                  LocalID,
		Name:                "Local Network",
		HRP:                 "local",
		ChainID:             12345,
		StartTime:           time.Now(),
		InitialStakeDuration: 24 * time.Hour,
		MinValidatorStake:   1000000000,  // 1 LUX (with 9 decimals)
		MinDelegatorStake:   1000000000,  // 1 LUX (with 9 decimals)
		IsL2:                false,
	},
	
	// L2 Networks (Subnets)
	"zoo-mainnet": {
		ID:                  ZooMainnetID,
		Name:                "Zoo Mainnet L2",
		HRP:                 "zoo",
		ChainID:             200200,
		StartTime:           time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   2000000000000000, // 2M LUX (validators stake on parent)
		MinDelegatorStake:   25000000000,      // 25 LUX
		IsL2:                true,
		ParentNetwork:       "mainnet",
	},
	"zoo-testnet": {
		ID:                  ZooTestnetID,
		Name:                "Zoo Testnet L2",
		HRP:                 "zoo-test",
		ChainID:             200201,
		StartTime:           time.Now(),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   1000000000,  // 1 LUX
		MinDelegatorStake:   1000000000,  // 1 LUX
		IsL2:                true,
		ParentNetwork:       "testnet",
	},
	"spc-mainnet": {
		ID:                  SPCMainnetID,
		Name:                "SPC Mainnet L2",
		HRP:                 "spc",
		ChainID:             36911,
		StartTime:           time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   2000000000000000, // 2M LUX
		MinDelegatorStake:   25000000000,      // 25 LUX
		IsL2:                true,
		ParentNetwork:       "mainnet",
	},
	"hanzo-mainnet": {
		ID:                  HanzoMainnetID,
		Name:                "Hanzo Mainnet L2",
		HRP:                 "hanzo",
		ChainID:             36963,
		StartTime:           time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   2000000000000000, // 2M LUX
		MinDelegatorStake:   25000000000,      // 25 LUX
		IsL2:                true,
		ParentNetwork:       "mainnet",
	},
	"hanzo-testnet": {
		ID:                  HanzoTestnetID,
		Name:                "Hanzo Testnet L2",
		HRP:                 "hanzo-test",
		ChainID:             36962,
		StartTime:           time.Now(),
		InitialStakeDuration: 365 * 24 * time.Hour,
		MinValidatorStake:   1000000000,  // 1 LUX
		MinDelegatorStake:   1000000000,  // 1 LUX
		IsL2:                true,
		ParentNetwork:       "testnet",
	},
}

// GetNetwork returns the network configuration for a given name
func GetNetwork(name string) (*NetworkConfig, error) {
	config, ok := Networks[name]
	if !ok {
		return nil, fmt.Errorf("unknown network: %s", name)
	}
	return config, nil
}

// String returns the string representation of a NetworkID
func (n NetworkID) String() string {
	switch n {
	case MainnetID:
		return "mainnet"
	case TestnetID:
		return "testnet"
	case LocalID:
		return "local"
	default:
		return fmt.Sprintf("custom-%d", n)
	}
}