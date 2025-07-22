package archeology


// Network represents a known blockchain network
type Network struct {
	Name         string
	ChainID      int64
	BlockchainID string
}

// Prefix represents a database key prefix
type Prefix struct {
	Name        string
	Prefix      []byte
	Description string
}

// ExtractorConfig holds configuration for the extractor
type ExtractorConfig struct {
	SourcePath   string
	DestPath     string
	ChainID      int64
	NetworkName  string
	IncludeState bool
	Limit        int
	Verify       bool
}

// ExtractResult contains extraction results
type ExtractResult struct {
	ChainID      int64
	BlockCount   int
	AccountCount int
	StorageCount int
	OutputPath   string
}

// AnalyzerConfig holds configuration for the analyzer
type AnalyzerConfig struct {
	DatabasePath string
	AccountAddr  string
	BlockNumber  int64
	NetworkName  string
}

// AnalysisResult contains analysis results
type AnalysisResult struct {
	ChainID          int64
	LatestBlock      int64
	TotalAccounts    int
	ContractAccounts int
	TotalBalance     string
	GenesisBlock     *BlockInfo
	AccountInfo      *AccountInfo
	TopAccounts      []AccountBalance
}

// BlockInfo contains block information
type BlockInfo struct {
	Number    uint64
	Hash      string
	Timestamp uint64
}

// AccountInfo contains account details
type AccountInfo struct {
	Address      string
	Balance      string
	Nonce        uint64
	IsContract   bool
	CodeSize     int
	StorageCount int
}

// AccountBalance represents an account with balance
type AccountBalance struct {
	Address string
	Balance string
}

// DenamespacerConfig holds configuration for the denamespacer
type DenamespacerConfig struct {
	SourcePath   string
	DestPath     string
	ChainID      int64
	DryRun       bool
	ShowProgress bool
}

// DenamespacerResult contains denamespace operation results
type DenamespacerResult struct {
	KeysProcessed       int
	KeysWithNamespace   int
	KeysWithoutNamespace int
	Errors              int
}

// ValidatorConfig holds configuration for the validator
type ValidatorConfig struct {
	DatabasePath string
	CheckState   bool
	CheckBlocks  bool
	Verbose      bool
}

// ValidationResult contains validation results
type ValidationResult struct {
	Status              string
	BlocksValidated     int
	AccountsValidated   int
	Errors              []string
	Warnings            []string
	BlockchainIntegrity *BlockchainIntegrity
	StateIntegrity      *StateIntegrity
}

// BlockchainIntegrity contains blockchain integrity check results
type BlockchainIntegrity struct {
	Continuous     bool
	HashChainValid bool
	FirstBlock     int64
	LastBlock      int64
	MissingBlocks  []int64
}

// StateIntegrity contains state integrity check results
type StateIntegrity struct {
	StateRootValid     bool
	AccountHashesValid bool
	StorageHashesValid bool
}

// GetKnownNetworks returns known network configurations
func GetKnownNetworks() []Network {
	return []Network{
		{Name: "lux-mainnet", ChainID: 96369, BlockchainID: "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ"},
		{Name: "lux-testnet", ChainID: 96368, BlockchainID: "2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG"},
		{Name: "zoo-mainnet", ChainID: 200200, BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM"},
		{Name: "zoo-testnet", ChainID: 200201, BlockchainID: "2usKC5aApgWQWwanB4LL6QPoqxR1bWWjPCtemBYbZvxkNfcnbj"},
		{Name: "spc-mainnet", ChainID: 36911, BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1"},
		{Name: "spc-testnet", ChainID: 36912, BlockchainID: ""},
		{Name: "hanzo-mainnet", ChainID: 36963, BlockchainID: ""},
		{Name: "hanzo-testnet", ChainID: 36962, BlockchainID: ""},
		{Name: "lux-genesis-7777", ChainID: 7777, BlockchainID: ""},
	}
}

// GetKnownPrefixes returns known database key prefixes
func GetKnownPrefixes() []Prefix {
	return []Prefix{
		{Name: "headers", Prefix: []byte{0x68}, Description: "Block headers"},
		{Name: "hash-to-number", Prefix: []byte{0x48}, Description: "Block hash to number mapping"},
		{Name: "bodies", Prefix: []byte{0x62}, Description: "Block bodies"},
		{Name: "receipts", Prefix: []byte{0x72}, Description: "Transaction receipts"},
		{Name: "accounts", Prefix: []byte{0x26}, Description: "Account data"},
		{Name: "storage", Prefix: []byte{0xa3}, Description: "Contract storage"},
		{Name: "state", Prefix: []byte{0x73}, Description: "State trie nodes"},
		{Name: "code", Prefix: []byte{0x63}, Description: "Contract bytecode"},
		{Name: "logs", Prefix: []byte{0x6c}, Description: "Transaction logs"},
	}
}
