package scanner

import (
	"math/big"

	// TODO: Replace with github.com/luxfi/geth when available
	"github.com/ethereum/go-ethereum/common"
)

// AssetHolder represents an NFT or token holder
type AssetHolder struct {
	Address         common.Address
	Balance         *big.Int   // For tokens
	TokenIDs        []*big.Int // For NFTs
	AssetType       string     // "NFT" or "Token"
	CollectionType  string     // NFT type (Validator, Card, etc.)
	StakingPower    *big.Int   // Staking power in wei
	ChainName       string
	ContractAddress string
	ProjectName     string
	LastActivity    uint64 // Block number
	ReceivedOnChain bool   // Cross-referenced
}

// ProjectConfig holds project-specific configurations
type ProjectConfig struct {
	TokenContracts  map[string]string   // chain -> contract
	NFTContracts    map[string]string   // chain -> contract
	StakingPowers   map[string]*big.Int // NFT type -> staking power
	TypeIdentifiers map[string][]string // NFT type -> keywords
}

// Default project configurations
var projectConfigs = map[string]ProjectConfig{
	"lux": {
		NFTContracts: map[string]string{
			"ethereum": "0x31e0f919c67cedd2bc3e294340dc900735810311",
		},
		StakingPowers: map[string]*big.Int{
			"Validator": new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M LUX
			"Card":      new(big.Int).Mul(big.NewInt(500000), big.NewInt(1e18)),  // 500K LUX
			"Coin":      new(big.Int).Mul(big.NewInt(100000), big.NewInt(1e18)),  // 100K LUX
			"Token":     big.NewInt(0), // Tokens don't have staking power
		},
		TypeIdentifiers: map[string][]string{
			"Validator": {"validator", "genesis", "founder"},
			"Card":      {"card", "legendary", "rare"},
			"Coin":      {"coin", "token", "lux"},
		},
	},
	"zoo": {
		TokenContracts: map[string]string{
			"bsc": "", // TODO: Add historic ZOO token contract
		},
		NFTContracts: map[string]string{
			"bsc": "", // TODO: Add ZOO NFTs if any
		},
		StakingPowers: map[string]*big.Int{
			"Animal":  new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M ZOO
			"Habitat": new(big.Int).Mul(big.NewInt(750000), big.NewInt(1e18)),  // 750K ZOO
			"Item":    new(big.Int).Mul(big.NewInt(250000), big.NewInt(1e18)),  // 250K ZOO
			"Token":   big.NewInt(0),
		},
		TypeIdentifiers: map[string][]string{
			"Animal":  {"animal", "creature", "beast"},
			"Habitat": {"habitat", "environment", "land"},
			"Item":    {"item", "tool", "resource"},
		},
	},
	"spc": {
		TokenContracts: map[string]string{},
		NFTContracts:   map[string]string{},
		StakingPowers: map[string]*big.Int{
			"Pony":      new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M SPC
			"Accessory": new(big.Int).Mul(big.NewInt(500000), big.NewInt(1e18)),  // 500K SPC
			"Token":     big.NewInt(0),
		},
		TypeIdentifiers: map[string][]string{
			"Pony":      {"pony", "sparkle", "unicorn"},
			"Accessory": {"accessory", "item", "gear"},
		},
	},
	"hanzo": {
		TokenContracts: map[string]string{},
		NFTContracts:   map[string]string{},
		StakingPowers: map[string]*big.Int{
			"AI":        new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)), // 1M AI
			"Algorithm": new(big.Int).Mul(big.NewInt(750000), big.NewInt(1e18)),  // 750K AI
			"Data":      new(big.Int).Mul(big.NewInt(500000), big.NewInt(1e18)),  // 500K AI
			"Token":     big.NewInt(0),
		},
		TypeIdentifiers: map[string][]string{
			"AI":        {"ai", "intelligence", "neural"},
			"Algorithm": {"algorithm", "compute", "process"},
			"Data":      {"data", "dataset", "training"},
		},
	},
}

// Common RPC endpoints
var chainRPCs = map[string]string{
	"ethereum":  "https://eth-mainnet.g.alchemy.com/v2/YOUR_API_KEY",
	"bsc":       "https://bsc-dataseed.binance.org/",
	"polygon":   "https://polygon-rpc.com/",
	"arbitrum":  "https://arb1.arbitrum.io/rpc",
	"optimism":  "https://mainnet.optimism.io",
	"avalanche": "https://api.avax.network/ext/bc/C/rpc",
}

// GetProjectConfigs returns all project configurations
func GetProjectConfigs() map[string]ProjectConfig {
	return projectConfigs
}

// GetChainRPCs returns default RPC endpoints
func GetChainRPCs() map[string]string {
	return chainRPCs
}

// ERC20 ABI for token functions
const erc20ABI = `[
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":false,"name":"value","type":"uint256"}],"name":"Transfer","type":"event"}
]`

// ERC721 ABI for NFT functions
const erc721ABI = `[
	{"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"type":"function"},
	{"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"inputs":[{"name":"tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"name":"","type":"string"}],"type":"function"},
	{"inputs":[{"name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"inputs":[{"name":"owner","type":"address"},{"name":"index","type":"uint256"}],"name":"tokenOfOwnerByIndex","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"anonymous":false,"inputs":[{"indexed":true,"name":"from","type":"address"},{"indexed":true,"name":"to","type":"address"},{"indexed":true,"name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"}
]`