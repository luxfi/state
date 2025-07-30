package bridge

// Chain represents a supported blockchain
type Chain struct {
	Name       string
	ChainID    int64
	DefaultRPC string
	Type       string
}

// Project represents a known project configuration
type Project struct {
	Name           string
	Symbol         string
	NFTContracts   map[string]string
	TokenContracts map[string]string
	StakingPowers  map[string]string
}

// NFTScannerConfig holds configuration for NFT scanning
type NFTScannerConfig struct {
	Chain           string
	ChainID         int64
	RPC             string // Changed from RPCURL to RPC
	ContractAddress string
	ProjectName     string
	FromBlock       uint64
	ToBlock         uint64
	BatchSize       uint64
	IncludeMetadata bool
	CrossReference  string
	ValidatorNFT    bool // For NFTs that grant validator status
}

// NFTScanResult contains NFT scan results
type NFTScanResult struct {
	ContractAddress      string
	CollectionName       string
	Symbol               string
	TotalSupply          int
	UniqueHolders        int
	FromBlock            uint64
	ToBlock              uint64
	BlockScanned         uint64
	ChainID              uint64
	TypeDistribution     map[string]int
	TopHolders           []Holder
	TotalNFTs            int
	NFTs                 []ScannedNFT // Add NFTs field for export
	StakingInfo          *StakingInfo
	CrossReferenceResult *CrossReferenceResult
}

// ScannedNFT represents a scanned NFT
type ScannedNFT struct {
	TokenID      string `json:"tokenId"`
	Owner        string `json:"owner"`
	URI          string `json:"uri"`
	StakingPower string `json:"stakingPower,omitempty"`
}

// TokenScannerConfig holds configuration for token scanning
type TokenScannerConfig struct {
	Chain           string
	ChainID         int64
	RPC             string // Changed from RPCURL to RPC
	ContractAddress string
	ProjectName     string
	FromBlock       uint64
	ToBlock         uint64
	MinBalance      string
	IncludeZero     bool
	CrossReference  string
}

// TokenScanResult contains token scan results
type TokenScanResult struct {
	ContractAddress      string
	TokenName            string
	Symbol               string
	Decimals             int
	TotalSupply          string
	UniqueHolders        int
	FromBlock            uint64
	ToBlock              uint64
	Distribution         []DistributionTier
	TopHolders           []TokenHolder
	Holders              []TokenHolder // Add all holders for export
	CrossReferenceResult *CrossReferenceResult
	MigrationInfo        *MigrationInfo
}

// MigrationConfig holds configuration for token migration
type MigrationConfig struct {
	SourceChain     string
	SourceChainID   int64
	SourceRPC       string
	ContractAddress string
	TokenType       string
	TargetLayer     string
	TargetName      string
	TargetChainID   int64
	IncludeHolders  bool
	MinBalance      string
	Snapshot        bool
	GenesisTemplate string
}

// MigrationAnalysis contains migration analysis results
type MigrationAnalysis struct {
	TokenName     string
	Symbol        string
	Decimals      int
	TotalSupply   string
	UniqueHolders int
	TotalNFTs     int
}

// SnapshotResult contains snapshot results
type SnapshotResult struct {
	BlockNumber      uint64
	HolderCount      int
	QualifiedHolders int
	Distribution     []DistributionTier
}

// MigrationArtifacts contains generated migration files
type MigrationArtifacts struct {
	GenesisPath      string
	ChainConfigPath  string
	DeploymentScript string
	MigrationGuide   string
	ValidatorConfig  string
}

// ExporterConfig holds configuration for export
type ExporterConfig struct {
	InputFiles   []string
	OutputPath   string
	Format       string
	ChainType    string
	MergeMode    string
	IncludeProof bool
}

// ExportResult contains export results
type ExportResult struct {
	Format          string
	RecordsExported int
	TotalValue      string
	AssetsSummary   map[string]int
	ProofInfo       *ProofInfo
	OutputFiles     []string
}

// VerifierConfig holds configuration for verification
type VerifierConfig struct {
	ScanFile       string
	GenesisFile    string
	ChainData      string
	VerifyBalances bool
	VerifyHolders  bool
	VerifyMetadata bool
	CrossReference bool   // Cross-reference with existing chain data
	MinBalance     string // Minimum balance threshold
	MaxSupply      string // Maximum supply cap
}

// VerificationResult contains verification results
type VerificationResult struct {
	Valid           bool                `json:"valid"`
	Summary         string              `json:"summary"`
	Warnings        []string            `json:"warnings"`
	Errors          []string            `json:"errors"`
	Status          string              `json:"status,omitempty"`
	RecordsVerified int                 `json:"recordsVerified,omitempty"`
	ChecksPerformed int                 `json:"checksPerformed,omitempty"`
	ChecksPassed    int                 `json:"checksPassed,omitempty"`
	ChecksFailed    int                 `json:"checksFailed,omitempty"`
	Discrepancies   []Discrepancy       `json:"discrepancies,omitempty"`
	BalanceCheck    *BalanceCheckResult `json:"balanceCheck,omitempty"`
	HolderCheck     *HolderCheckResult  `json:"holderCheck,omitempty"`
}

// Helper types

type Holder struct {
	Address string
	Count   int
}

type TokenHolder struct {
	Address          string
	Balance          string
	BalanceFormatted string
	Percentage       float64
}

type StakingInfo struct {
	ValidatorCount int
	TotalPower     string
}

type CrossReferenceResult struct {
	Matched          int      `json:"matched"`
	NotFound         int      `json:"notFound"`
	Additional       int      `json:"additional"`
	TotalSource      int      `json:"totalSource"`
	TotalTarget      int      `json:"totalTarget"`
	NotFoundIDs      []string `json:"notFoundIds,omitempty"`
	FoundOnChain     int      `json:"foundOnChain,omitempty"`
	NewAddresses     int      `json:"newAddresses,omitempty"`
	MissingFromChain int      `json:"missingFromChain,omitempty"`
}

type DistributionTier struct {
	Range      string
	Count      int
	Percentage float64
}

type MigrationInfo struct {
	HoldersToMigrate int
	BalanceToMigrate string
	RecommendedLayer string
}

type ProofInfo struct {
	RootHash        string
	TreeHeight      int
	ProofsGenerated int
}

type Discrepancy struct {
	Type        string
	Description string
}

type BalanceCheckResult struct {
	AccountsChecked int
	Matches         int
	Mismatches      int
	TotalDifference string
}

type HolderCheckResult struct {
	ExpectedHolders int
	FoundHolders    int
	Missing         int
	Extra           int
}

// GetSupportedChains returns supported blockchain networks
func GetSupportedChains() []Chain {
	return []Chain{
		{Name: "ethereum", ChainID: 1, DefaultRPC: "https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY", Type: "EVM"},
		{Name: "bsc", ChainID: 56, DefaultRPC: "https://bsc-dataseed.binance.org/", Type: "EVM"},
		{Name: "polygon", ChainID: 137, DefaultRPC: "https://polygon-rpc.com/", Type: "EVM"},
		{Name: "arbitrum", ChainID: 42161, DefaultRPC: "https://arb1.arbitrum.io/rpc", Type: "EVM"},
		{Name: "optimism", ChainID: 10, DefaultRPC: "https://mainnet.optimism.io", Type: "EVM"},
		{Name: "avalanche", ChainID: 43114, DefaultRPC: "https://api.avax.network/ext/bc/C/rpc", Type: "EVM"},
	}
}

// GetKnownProjects returns known project configurations
func GetKnownProjects() []Project {
	return []Project{
		{
			Name:   "lux",
			Symbol: "LUX",
			NFTContracts: map[string]string{
				"ethereum": "0x31e0f919c67cedd2bc3e294340dc900735810311",
			},
			StakingPowers: map[string]string{
				"Validator": "1000000",
				"Card":      "500000",
				"Coin":      "100000",
			},
		},
		{
			Name:   "zoo",
			Symbol: "ZOO",
			TokenContracts: map[string]string{
				"bsc": "0x0a6045b79151d0a54dbd5227082445750a023af2",
			},
			NFTContracts: map[string]string{
				"bsc": "0x5bb68cf06289d54efde25155c88003be685356a8", // EGG NFT
			},
			StakingPowers: map[string]string{
				"EGG":     "1000000",
				"Animal":  "750000",
				"Habitat": "500000",
			},
		},
	}
}
