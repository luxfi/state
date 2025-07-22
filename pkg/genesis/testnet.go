package genesis

import (
	"fmt"
)

// BuildTestnet creates the complete testnet genesis configuration
func BuildTestnet(cchainGenesisPath string) (*MainGenesis, error) {
	// Create builder
	builder, err := NewBuilder("testnet")
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	// Load C-Chain genesis
	if err := builder.LoadCChainGenesis(cchainGenesisPath); err != nil {
		return nil, fmt.Errorf("failed to load C-Chain genesis: %w", err)
	}

	// For testnet, we might use different validators or none at all
	// This can be customized based on requirements

	// Build genesis
	return builder.Build()
}