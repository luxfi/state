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

// Config holds input parameters for archeology genesis command
type Config struct {
	NFTDataPath      string
	TokenDataPath    string
	AccountsDataPath string
	OutputPath       string
	ChainType        string
	AssetPrefix      string
}

// DeployerConfig holds subnet deployment parameters
type DeployerConfig struct {
	SubnetName  string
	GenesisPath string
}

// CreateResult is returned from subnet creation
type CreateResult struct {
	SubnetID      string
	TransactionID string
	BlockchainID  string
}

// DeployResult is returned after deploying a subnet
type DeployResult struct {
	SubnetID        string
	BlockchainID    string
	VMID            string
	ChainID         int
	NodeConfigPath  string
	ChainConfigPath string
}

// GeneratorConfig holds configuration for genesis file generation
type GeneratorConfig struct {
	NetworkName     string
	ChainID         int64
	ChainType       string
	DataPath        string
	ExternalPath    string
	TemplatePath    string
	OutputPath      string
	AssetPrefix     string
	IncludeTestData bool
}

// GeneratorResult contains details of a generated genesis file
type GeneratorResult struct {
   NetworkName     string
   ChainID         int64
   ChainType       string
   TotalAccounts   int
   TotalBalance    string
   Assets          []AssetInfo
   ExternalAssets  []struct {
       Type   string
       Source string
       Count  int
   }
   // Archeology genesis fields
   Timestamp        string
   TotalAssetTypes  int
   NFTCollections   map[string]struct {
       Count          int
       Holders        int
       StakingEnabled bool
       StakingPower   string
   }
   TokenAssets      map[string]struct {
       TotalSupply string
       Holders     int
   }
   AccountsMigrated  int
   ValidatorEligible int
   // OutputFile holds the written file path
   OutputFile       string
   // OutputPath and FileSize kept for CLI compatibility
   OutputPath       string
   FileSize         string
}

// ImporterConfig holds configuration for asset importing
type ImporterConfig struct {
	GenesisPath string
	OutputPath  string
}

// ImportResult contains result of importing external assets
type ImportResult struct {
	AssetsImported int
	AccountsAdded  int
	OutputPath     string
}

// LauncherConfig holds config for launching a network
type LauncherConfig struct {
	NetworkName string
	GenesisPath string
	RPCPort     int
	ChainID     int64
}

// LaunchResult contains details of a started network instance
type LaunchResult struct {
	ProcessID       int
	LogFile         string
	RPCEndpoint     string
	WSEndpoint      string
	MetricsEndpoint string
	NodeID          string
	NetworkID       int
	ChainID         int64
}

// NodeStatus reports runtime status of a node
type NodeStatus struct {
	Uptime       string
	BlockHeight  uint64
	PeerCount    int
	DatabaseSize string
}

// MergerConfig holds config for merging genesis files
type MergerConfig struct {
	InputFiles []string
	OutputPath string
}

// MergeResult contains result details of merging operation
type MergeResult struct {
	TotalAccounts     int
	TotalBalance      string
	AssetsMerged      int
	ConflictsResolved int
	Warnings          []string
}

// ValidatorConfig holds config for genesis validation
type ValidatorConfig struct {
	GenesisPath string
	ChainID     int64
	NetworkName string
}

// ValidatorResult contains result details of validation
type ValidatorResult struct {
	Status             string
	ChainID            int64
	NetworkName        string
	TotalAccounts      int
	TotalSupply        string
	ContractAccounts   int
	EOAAccounts        int
	ChecksPassed       int
	ChecksFailed       int
	ReadyForProduction bool
	AssetInfo          []AssetInfo
	Details            []CheckDetail
}

// AssetInfo holds basic asset statistics
type AssetInfo struct {
	Name        string
	Holders     int
	TotalSupply string
}

// CheckDetail provides detailed validation check results
type CheckDetail struct {
	Name    string
	Passed  bool
	Message string

}
