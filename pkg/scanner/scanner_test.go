package scanner

import (
	"math/big"
	"strings"
	"testing"
)

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config with known project",
			config: Config{
				Chain:           "ethereum",
				ContractAddress: "0x1234567890123456789012345678901234567890",
				ProjectName:     "lux",
			},
			wantErr: false,
		},
		{
			name: "Unknown project",
			config: Config{
				Chain:           "ethereum",
				ContractAddress: "0x1234567890123456789012345678901234567890",
				ProjectName:     "unknown",
			},
			wantErr: true,
		},
		{
			name: "No RPC and unknown chain",
			config: Config{
				Chain:           "unknown-chain",
				ContractAddress: "0x1234567890123456789012345678901234567890",
				ProjectName:     "lux",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatStakingPower(t *testing.T) {
	tests := []struct {
		name  string
		power *big.Int
		want  string
	}{
		{
			name:  "Zero",
			power: big.NewInt(0),
			want:  "0",
		},
		{
			name:  "1 Million tokens",
			power: new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)),
			want:  "1.0M",
		},
		{
			name:  "500K tokens",
			power: new(big.Int).Mul(big.NewInt(500000), big.NewInt(1e18)),
			want:  "500K",
		},
		{
			name:  "100 tokens",
			power: new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18)),
			want:  "100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStakingPower(tt.power)
			if got != tt.want {
				t.Errorf("FormatStakingPower() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProjectConfigs(t *testing.T) {
	// Test that all projects have valid configurations
	for name, config := range projectConfigs {
		t.Run(name, func(t *testing.T) {
			// Check NFT contracts (skip empty TODO contracts)
			for chain, contract := range config.NFTContracts {
				if contract == "" && !strings.Contains(name, "zoo") {
					t.Errorf("Project %s has empty NFT contract for chain %s", name, chain)
				}
			}

			// Check token contracts (skip empty TODO contracts)
			for chain, contract := range config.TokenContracts {
				if contract == "" && !strings.Contains(name, "zoo") {
					t.Errorf("Project %s has empty token contract for chain %s", name, chain)
				}
			}

			// Check staking powers
			if len(config.StakingPowers) == 0 {
				t.Errorf("Project %s has no staking powers defined", name)
			}

			for nftType, power := range config.StakingPowers {
				if power == nil || power.Sign() < 0 {
					t.Errorf("Project %s has invalid staking power for %s", name, nftType)
				}
			}

			// Check type identifiers
			if len(config.TypeIdentifiers) == 0 {
				t.Errorf("Project %s has no type identifiers", name)
			}
		})
	}
}

func TestChainRPCs(t *testing.T) {
	// Test that all chain RPCs are valid URLs
	for chain, rpc := range chainRPCs {
		t.Run(chain, func(t *testing.T) {
			if rpc == "" {
				t.Errorf("Chain %s has empty RPC URL", chain)
			}
			
			// Basic URL validation
			if !strings.HasPrefix(rpc, "http://") && !strings.HasPrefix(rpc, "https://") {
				t.Errorf("Chain %s has invalid RPC URL: %s", chain, rpc)
			}
		})
	}
}