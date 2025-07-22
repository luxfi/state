package genesis

// X-Chain Genesis structures
type XChainGenesis struct {
	Allocations []GenesisAsset `json:"allocations"`
	StartTime   int64          `json:"startTime"`
	Message     string         `json:"message"`
}

type GenesisAsset struct {
	AssetAlias   string                `json:"assetAlias"`
	AssetID      string                `json:"assetID"`
	InitialState map[string][]UTXOData `json:"initialState"`
	Memo         string                `json:"memo"`
}

type UTXOData struct {
	Amount    uint64   `json:"amount,omitempty"`    // For fungible tokens
	Locktime  uint64   `json:"locktime"`
	Threshold uint32   `json:"threshold,omitempty"` // Default: 1
	Addresses []string `json:"addresses"`
	Payload   string   `json:"payload,omitempty"`   // NFT metadata
	GroupID   uint32   `json:"groupID,omitempty"`   // NFT collection
}

// P-Chain Genesis structures (placeholder)
type PChainGenesis struct {
	// TODO: Implement P-Chain genesis structure
}

// Data record types from CSV
type NFTRecord struct {
	Address         string
	AssetType       string
	CollectionType  string
	TokenCount      int
	StakingPowerWei string
	ChainName       string
	ContractAddress string
	ProjectName     string
	TokenIDs        []string
}

type TokenRecord struct {
	Address         string
	BalanceWei      string
	ChainName       string
	ContractAddress string
	ProjectName     string
}

type AccountRecord struct {
	Address           string
	BalanceWei        string
	ValidatorEligible bool
}