package cchain

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// Genesis represents the C-Chain genesis configuration
type Genesis struct {
	Config     *ChainConfig           `json:"config"`
	Alloc      map[string]GenesisAccount `json:"alloc"`
	Nonce      string                 `json:"nonce"`
	Timestamp  string                 `json:"timestamp"`
	ExtraData  string                 `json:"extraData"`
	GasLimit   string                 `json:"gasLimit"`
	Difficulty string                 `json:"difficulty"`
	MixHash    string                 `json:"mixHash"`
	Coinbase   string                 `json:"coinbase"`
	Number     string                 `json:"number"`
	GasUsed    string                 `json:"gasUsed"`
	ParentHash string                 `json:"parentHash"`
}

// ChainConfig contains the C-Chain configuration parameters
type ChainConfig struct {
	ChainID             uint64      `json:"chainId"`
	HomesteadBlock      uint64      `json:"homesteadBlock"`
	EIP150Block         uint64      `json:"eip150Block"`
	EIP150Hash          string      `json:"eip150Hash"`
	EIP155Block         uint64      `json:"eip155Block"`
	EIP158Block         uint64      `json:"eip158Block"`
	ByzantiumBlock      uint64      `json:"byzantiumBlock"`
	ConstantinopleBlock uint64      `json:"constantinopleBlock"`
	PetersburgBlock     uint64      `json:"petersburgBlock"`
	IstanbulBlock       uint64      `json:"istanbulBlock"`
	MuirGlacierBlock    uint64      `json:"muirGlacierBlock"`
	BerlinBlock         uint64      `json:"berlinBlock"`
	LondonBlock         uint64      `json:"londonBlock"`
	SubnetEVMTimestamp  uint64      `json:"subnetEVMTimestamp"`
	FeeConfig           *FeeConfig  `json:"feeConfig"`
	AllowFeeRecipients  bool        `json:"allowFeeRecipients"`
}

// FeeConfig contains fee configuration for the C-Chain
type FeeConfig struct {
	GasLimit                 uint64 `json:"gasLimit"`
	MinBaseFee               uint64 `json:"minBaseFee"`
	TargetGas                uint64 `json:"targetGas"`
	BaseFeeChangeDenominator uint64 `json:"baseFeeChangeDenominator"`
	MinBlockGasCost          uint64 `json:"minBlockGasCost"`
	MaxBlockGasCost          uint64 `json:"maxBlockGasCost"`
	TargetBlockRate          uint64 `json:"targetBlockRate"`
	BlockGasCostStep         uint64 `json:"blockGasCostStep"`
}

// GenesisAccount represents an account in the genesis block
type GenesisAccount struct {
	Balance string            `json:"balance"`
	Code    string            `json:"code,omitempty"`
	Storage map[string]string `json:"storage,omitempty"`
	Nonce   string            `json:"nonce,omitempty"`
}

// Builder helps construct C-Chain genesis configurations
type Builder struct {
	chainID uint64
}

// NewBuilder creates a new C-Chain genesis builder
func NewBuilder(chainID uint64) *Builder {
	return &Builder{
		chainID: chainID,
	}
}

// Build creates a C-Chain genesis configuration
func (b *Builder) Build() *Genesis {
	return &Genesis{
		Config:     b.buildChainConfig(),
		Alloc:      make(map[string]GenesisAccount),
		Nonce:      "0x0",
		Timestamp:  "0x0",
		ExtraData:  "0x00",
		GasLimit:   "0xE4E1C0", // 15,000,000
		Difficulty: "0x0",
		MixHash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		Number:     "0x0",
		GasUsed:    "0x0",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
}

// buildChainConfig creates the chain configuration
func (b *Builder) buildChainConfig() *ChainConfig {
	return &ChainConfig{
		ChainID:             b.chainID,
		HomesteadBlock:      0,
		EIP150Block:         0,
		EIP150Hash:          "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
		EIP155Block:         0,
		EIP158Block:         0,
		ByzantiumBlock:      0,
		ConstantinopleBlock: 0,
		PetersburgBlock:     0,
		IstanbulBlock:       0,
		MuirGlacierBlock:    0,
		BerlinBlock:         0,
		LondonBlock:         0,
		SubnetEVMTimestamp:  0,
		FeeConfig:           b.buildFeeConfig(),
		AllowFeeRecipients:  false,
	}
}

// buildFeeConfig creates the fee configuration
func (b *Builder) buildFeeConfig() *FeeConfig {
	return &FeeConfig{
		GasLimit:                 15000000,
		MinBaseFee:               25000000000, // 25 Gwei
		TargetGas:                15000000,
		BaseFeeChangeDenominator: 36,
		MinBlockGasCost:          0,
		MaxBlockGasCost:          1000000,
		TargetBlockRate:          2,
		BlockGasCostStep:         200000,
	}
}

// AddAccount adds an account to the genesis allocation
func (b *Builder) AddAccount(address string, balance *big.Int) {
	// This method would be called on the Genesis object, not the builder
	// Keeping builder pattern clean
}

// AddAccountToGenesis adds an account to the genesis allocation
func AddAccountToGenesis(g *Genesis, address string, balance *big.Int) {
	// Convert balance to hex string
	balanceHex := fmt.Sprintf("0x%x", balance)
	
	g.Alloc[address] = GenesisAccount{
		Balance: balanceHex,
	}
}

// AddCodedAccount adds an account with code (contract) to genesis
func AddCodedAccount(g *Genesis, address string, balance *big.Int, code string, storage map[string]string) {
	balanceHex := fmt.Sprintf("0x%x", balance)
	
	g.Alloc[address] = GenesisAccount{
		Balance: balanceHex,
		Code:    code,
		Storage: storage,
	}
}

// ToJSON converts the genesis to JSON
func (g *Genesis) ToJSON() ([]byte, error) {
	return json.MarshalIndent(g, "", "\t")
}

// ImportAllocations imports allocations from a JSON file
func ImportAllocations(g *Genesis, allocationsJSON []byte) error {
	var allocations map[string]struct {
		Balance string `json:"balance"`
	}
	
	if err := json.Unmarshal(allocationsJSON, &allocations); err != nil {
		return fmt.Errorf("failed to unmarshal allocations: %w", err)
	}
	
	for address, account := range allocations {
		// Ensure address is lowercase and has 0x prefix
		if len(address) > 2 && address[:2] != "0x" {
			address = "0x" + address
		}
		address = strings.ToLower(address)
		
		g.Alloc[address] = GenesisAccount{
			Balance: account.Balance,
		}
	}
	
	return nil
}

// GetTotalSupply calculates the total supply in the C-Chain genesis
func (g *Genesis) GetTotalSupply() *big.Int {
	total := new(big.Int)
	
	for _, account := range g.Alloc {
		// Parse balance (remove 0x prefix)
		balance := new(big.Int)
		if len(account.Balance) > 2 && account.Balance[:2] == "0x" {
			balance.SetString(account.Balance[2:], 16)
		} else {
			balance.SetString(account.Balance, 10)
		}
		
		total.Add(total, balance)
	}
	
	return total
}