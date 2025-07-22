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
	RPCURL          string
	ContractAddress string
	ProjectName     string
	FromBlock       uint64
	ToBlock         uint64
	BatchSize       uint64
	IncludeMetadata bool
	CrossReference  string
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
	TypeDistribution     map[string]int
	TopHolders           []Holder
	TotalNFTs            int
	StakingInfo          *StakingInfo
	CrossReferenceResult *CrossReferenceResult
}

// TokenScannerConfig holds configuration for token scanning
type TokenScannerConfig struct {
	Chain           string
	ChainID         int64
	RPCURL          string
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
	TokenName        string
	Symbol           string
	Decimals         int
	TotalSupply      string
	UniqueHolders    int
	TotalNFTs        int
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
}

// VerificationResult contains verification results
type VerificationResult struct {
	Status           string
	RecordsVerified  int
	ChecksPerformed  int
	ChecksPassed     int
	ChecksFailed     int
	Warnings         []string
	Discrepancies    []Discrepancy
	BalanceCheck     *BalanceCheckResult
	HolderCheck      *HolderCheckResult
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
	FoundOnChain     int
	NewAddresses     int
	MissingFromChain int
}

type DistributionTier struct {
	Range      string
	Count      int
	Percentage float64
}

type MigrationInfo struct {
	HoldersToMigrate  int
	BalanceToMigrate  string
	RecommendedLayer  string
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
				"bsc": "", // TODO: Add contract
			},
			StakingPowers: map[string]string{
				"Animal":  "1000000",
				"Habitat": "750000",
				"Item":    "250000",
			},
		},
	}
}