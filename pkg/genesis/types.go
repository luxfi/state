package genesis

import (
	"github.com/luxfi/node/genesis"
	"github.com/luxfi/node/vms/platformvm/signer"
)

// MainGenesis represents the complete genesis configuration for all chains
type MainGenesis struct {
	NetworkID                  uint32                 `json:"networkID"`
	Allocations                []genesis.UnparsedAllocation `json:"allocations"`
	StartTime                  uint64                 `json:"startTime"`
	InitialStakeDuration       uint64                 `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64                 `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []string               `json:"initialStakedFunds"`
	InitialStakers             []genesis.UnparsedStaker    `json:"initialStakers"`
	CChainGenesis              string                 `json:"cChainGenesis"`
	Message                    string                 `json:"message"`
}

// Staker represents a validator in the initial staker set
type Staker struct {
	NodeID        string
	RewardAddress string
	DelegationFee uint32
	Weight        uint64
	Signer        *signer.ProofOfPossession
}

// StakerConfig contains configuration for creating stakers
type StakerConfig struct {
	NodeID        string
	ETHAddress    string
	PublicKey     string
	ProofOfPossession string
	Weight        uint64
	DelegationFee uint32
}