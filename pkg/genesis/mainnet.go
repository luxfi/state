package genesis

import (
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/luxfi/genesis/pkg/genesis/allocation"
)

// BuildMainnet creates the complete mainnet genesis configuration
func BuildMainnet(validatorsPath, cchainGenesisPath string) (*MainGenesis, error) {
	// Create builder
	builder, err := NewBuilder("mainnet")
	if err != nil {
		return nil, fmt.Errorf("failed to create builder: %w", err)
	}

	// Load C-Chain genesis
	if err := builder.LoadCChainGenesis(cchainGenesisPath); err != nil {
		return nil, fmt.Errorf("failed to load C-Chain genesis: %w", err)
	}

	// Load and add validators
	validators, err := LoadValidators(validatorsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load validators: %w", err)
	}

	// Add validators with staking allocations
	if err := builder.AddValidatorsWithStaking(validators); err != nil {
		return nil, fmt.Errorf("failed to add validators: %w", err)
	}

	// Build genesis
	return builder.Build()
}

// LoadCChainGenesis loads C-Chain genesis from file
func (b *Builder) LoadCChainGenesis(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read C-Chain genesis: %w", err)
	}

	b.SetCChainGenesis(string(data))
	return nil
}

// AddValidatorsWithStaking adds validators with their staking allocations
func (b *Builder) AddValidatorsWithStaking(validators []ValidatorInfo) error {
	for i, v := range validators {
		// Add to staker set
		b.AddStaker(StakerConfig{
			NodeID:            v.NodeID,
			ETHAddress:        v.ETHAddress,
			PublicKey:         v.PublicKey,
			ProofOfPossession: v.ProofOfPossession,
			Weight:            v.Weight,
			DelegationFee:     v.DelegationFee,
		})

		// Add locked allocation for staking (2M LUX each)
		stakingAmount := new(big.Int).SetUint64(2000000000000000) // 2M LUX in wei

		vestingConfig := &allocation.UnlockScheduleConfig{
			TotalAmount:  stakingAmount,
			StartDate:    time.Unix(1577836800, 0), // Jan 1, 2020
			Duration:     365 * 24 * time.Hour,     // 1 year
			Periods:      1,
			CliffPeriods: 0,
		}

		if err := b.AddVestedAllocation(v.ETHAddress, vestingConfig); err != nil {
			return fmt.Errorf("failed to add validator %d allocation: %w", i, err)
		}
	}

	return nil
}
