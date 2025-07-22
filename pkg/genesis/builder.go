package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/luxfi/node/genesis"
	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/vms/platformvm/signer"
	
	"github.com/luxfi/genesis/pkg/genesis/address"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/genesis/pkg/genesis/cchain"
	"github.com/luxfi/genesis/pkg/genesis/config"
)

// Builder orchestrates the creation of genesis configurations
type Builder struct {
	network       *config.NetworkConfig
	addressConv   *address.Converter
	allocBuilder  *allocation.Builder
	cchainBuilder *cchain.Builder
	allocations   *allocation.AllocationSet
	stakers       []StakerConfig
	cchainGenesis string // Stores the C-Chain genesis JSON
}

// NewBuilder creates a new genesis builder for a specific network
func NewBuilder(networkName string) (*Builder, error) {
	network, err := config.GetNetwork(networkName)
	if err != nil {
		return nil, err
	}

	addressConv := address.NewConverter(network.HRP)
	allocBuilder := allocation.NewBuilder(addressConv)
	cchainBuilder := cchain.NewBuilder(network.ChainID)

	return &Builder{
		network:       network,
		addressConv:   addressConv,
		allocBuilder:  allocBuilder,
		cchainBuilder: cchainBuilder,
		allocations:   allocation.NewAllocationSet(),
		stakers:       []StakerConfig{},
	}, nil
}

// AddAllocation adds a simple allocation
func (b *Builder) AddAllocation(ethAddr string, amount *big.Int) error {
	alloc, err := b.allocBuilder.CreateSimpleAllocation(ethAddr, amount)
	if err != nil {
		return err
	}
	return b.allocations.Add(alloc)
}

// AddVestedAllocation adds an allocation with vesting
func (b *Builder) AddVestedAllocation(ethAddr string, config *allocation.UnlockScheduleConfig) error {
	alloc, err := b.allocBuilder.CreateVestedAllocation(ethAddr, config)
	if err != nil {
		return err
	}
	return b.allocations.Add(alloc)
}

// AddStaker adds a validator to the initial staker set
func (b *Builder) AddStaker(config StakerConfig) {
	b.stakers = append(b.stakers, config)
}

// ImportCChainGenesis imports existing C-Chain genesis data
func (b *Builder) ImportCChainGenesis(genesisPath string) error {
	data, err := ioutil.ReadFile(genesisPath)
	if err != nil {
		return fmt.Errorf("failed to read C-Chain genesis: %w", err)
	}

	// Parse to validate JSON
	var cGenesis cchain.Genesis
	if err := json.Unmarshal(data, &cGenesis); err != nil {
		return fmt.Errorf("invalid C-Chain genesis: %w", err)
	}

	// Store for later use
	b.cchainGenesis = string(data)
	return nil
}

// ImportCChainAllocations imports allocations from C-Chain JSON
func (b *Builder) ImportCChainAllocations(allocPath string) error {
	data, err := ioutil.ReadFile(allocPath)
	if err != nil {
		return fmt.Errorf("failed to read allocations: %w", err)
	}

	cGenesis := b.cchainBuilder.Build()
	if err := cchain.ImportAllocations(cGenesis, data); err != nil {
		return err
	}

	// Store the updated genesis
	genesisJSON, err := cGenesis.ToJSON()
	if err != nil {
		return err
	}
	b.cchainGenesis = string(genesisJSON)
	
	return nil
}

// Build creates the final genesis configuration
func (b *Builder) Build() (*MainGenesis, error) {
	// Convert allocations to unparsed format
	unparsedAllocs := make([]genesis.UnparsedAllocation, 0)
	stakedFunds := make([]string, 0)

	for _, alloc := range b.allocations.GetAll() {
		// Convert locked amounts
		lockedAmounts := make([]genesis.UnparsedLockedAmount, len(alloc.UnlockSchedule))
		for i, locked := range alloc.UnlockSchedule {
			lockedAmounts[i] = genesis.UnparsedLockedAmount{
				Amount:   locked.Amount.Uint64(),
				Locktime: locked.Locktime,
			}
		}

		unparsedAllocs = append(unparsedAllocs, genesis.UnparsedAllocation{
			ETHAddr:        alloc.ETHAddr,
			LUXAddr:        alloc.LuxAddr,
			InitialAmount:  alloc.InitialAmount.Uint64(),
			UnlockSchedule: lockedAmounts,
		})

		// Track staked funds if has locked amounts
		if len(lockedAmounts) > 0 {
			stakedFunds = append(stakedFunds, alloc.LuxAddr)
		}
	}

	// Convert stakers to unparsed format
	unparsedStakers := make([]genesis.UnparsedStaker, 0)
	for _, staker := range b.stakers {
		// Convert P-chain address
		rewardAddr, err := b.addressConv.ETHToLux(staker.ETHAddress, "P")
		if err != nil {
			return nil, fmt.Errorf("failed to convert staker address: %w", err)
		}

		// Parse node ID
		nodeID, err := ids.NodeIDFromString(staker.NodeID)
		if err != nil {
			return nil, fmt.Errorf("invalid node ID %s: %w", staker.NodeID, err)
		}

		unparsedStaker := genesis.UnparsedStaker{
			NodeID:        nodeID,
			RewardAddress: rewardAddr,
			DelegationFee: staker.DelegationFee,
		}

		// Add signer info if provided
		if staker.PublicKey != "" && staker.ProofOfPossession != "" {
			unparsedStaker.Signer = &signer.ProofOfPossession{
				PublicKey:         staker.PublicKey,
				ProofOfPossession: staker.ProofOfPossession,
			}
		}

		unparsedStakers = append(unparsedStakers, unparsedStaker)
	}

	// Create C-Chain genesis if not imported
	if b.cchainGenesis == "" {
		cGenesis := b.cchainBuilder.Build()
		genesisJSON, err := cGenesis.ToJSON()
		if err != nil {
			return nil, fmt.Errorf("failed to create C-Chain genesis: %w", err)
		}
		b.cchainGenesis = string(genesisJSON)
	}

	return &MainGenesis{
		NetworkID:                  uint32(b.network.ID),
		Allocations:                unparsedAllocs,
		StartTime:                  uint64(b.network.StartTime.Unix()),
		InitialStakeDuration:       uint64(b.network.InitialStakeDuration.Seconds()),
		InitialStakeDurationOffset: 5400, // 90 minutes
		InitialStakedFunds:         stakedFunds,
		InitialStakers:             unparsedStakers,
		CChainGenesis:              b.cchainGenesis,
		Message:                    fmt.Sprintf("Lux Network Genesis - %s", b.network.Name),
	}, nil
}

// SaveToFile saves the genesis to a JSON file
func (b *Builder) SaveToFile(genesis *MainGenesis, filepath string) error {
	data, err := json.MarshalIndent(genesis, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %w", err)
	}

	if err := ioutil.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// SetCChainGenesis allows setting the C-Chain genesis directly
func (b *Builder) SetCChainGenesis(genesisJSON string) {
	b.cchainGenesis = genesisJSON
}

// GetTotalSupply returns the total supply across all allocations
func (b *Builder) GetTotalSupply() *big.Int {
	return b.allocations.TotalSupply()
}

// GetAllocationCount returns the number of allocations
func (b *Builder) GetAllocationCount() int {
	return b.allocations.Count()
}