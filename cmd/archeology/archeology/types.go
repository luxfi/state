// Package archeology provides tools for extracting and analyzing historical blockchain data
// from LevelDB and PebbleDB databases used by various EVM implementations.
package archeology

import (
    "encoding/binary"
    "fmt"
)

// KeyType represents different types of keys found in EVM databases
type KeyType byte

const (
    // Standard EVM key types
    KeyTypeHeader        KeyType = 'h' // 0x68 - Block headers
    KeyTypeHashToNumber  KeyType = 'H' // 0x48 - Block hash to number mapping
    KeyTypeBody          KeyType = 'b' // 0x62 - Block bodies (transactions)
    KeyTypeBodyAlt       KeyType = 'B' // 0x42 - Alternative block bodies
    KeyTypeReceipt       KeyType = 'r' // 0x72 - Transaction receipts
    KeyTypeNumberToHash  KeyType = 'n' // 0x6e - Block number to hash mapping
    KeyTypeTxLookup      KeyType = 'l' // 0x6c - Transaction lookup (tx hash to block)
    KeyTypeTransaction   KeyType = 't' // 0x74 - Raw transactions
    
    // State trie keys
    KeyTypeAccount       KeyType = 0x26 // Account data
    KeyTypeStorage       KeyType = 0xa3 // Contract storage
    KeyTypeCode          KeyType = 'c'  // 0x63 - Contract code
    KeyTypeState         KeyType = 's'  // 0x73 - State trie nodes
    KeyTypeStateObject   KeyType = 'o'  // 0x6f - State objects
    
    // Metadata keys
    KeyTypeMetadata      KeyType = 0xfd // Chain metadata
    KeyTypeLastValues    KeyType = 'l'  // 0x6c - Last accepted/finalized values
)

// KeyTypeInfo provides metadata about a key type
type KeyTypeInfo struct {
    Type        KeyType
    Name        string
    Description string
    IsState     bool // Whether this is state data vs blockchain data
}

// KnownKeyTypes maps key types to their information
var KnownKeyTypes = map[KeyType]KeyTypeInfo{
    KeyTypeHeader: {
        Type:        KeyTypeHeader,
        Name:        "headers",
        Description: "Block headers containing metadata",
        IsState:     false,
    },
    KeyTypeHashToNumber: {
        Type:        KeyTypeHashToNumber,
        Name:        "hash->number",
        Description: "Maps block hashes to block numbers",
        IsState:     false,
    },
    KeyTypeBody: {
        Type:        KeyTypeBody,
        Name:        "bodies",
        Description: "Block bodies containing transactions",
        IsState:     false,
    },
    KeyTypeReceipt: {
        Type:        KeyTypeReceipt,
        Name:        "receipts",
        Description: "Transaction receipts with execution results",
        IsState:     false,
    },
    KeyTypeAccount: {
        Type:        KeyTypeAccount,
        Name:        "accounts",
        Description: "Account balances and nonces",
        IsState:     true,
    },
    KeyTypeStorage: {
        Type:        KeyTypeStorage,
        Name:        "storage",
        Description: "Smart contract storage slots",
        IsState:     true,
    },
    KeyTypeCode: {
        Type:        KeyTypeCode,
        Name:        "code",
        Description: "Smart contract bytecode",
        IsState:     true,
    },
}

// ChainConfig represents configuration for a specific blockchain
type ChainConfig struct {
    NetworkID      string
    ChainID        uint64
    ChainIDHex     string // Hex representation of chain ID for namespace
    Name           string
    TokenSymbol    string
    BlockchainID   string // Optional: Avalanche-style blockchain ID
}

// KnownChains provides configurations for known networks
var KnownChains = map[string]ChainConfig{
    "lux-mainnet": {
        NetworkID:    "96369",
        ChainID:      96369,
        ChainIDHex:   "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1",
        Name:         "Lux Mainnet",
        TokenSymbol:  "LUX",
        BlockchainID: "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ",
    },
    "lux-testnet": {
        NetworkID:    "96368",
        ChainID:      96368,
        ChainIDHex:   "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1",
        Name:         "Lux Testnet",
        TokenSymbol:  "LUX",
        BlockchainID: "2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG",
    },
    "zoo-mainnet": {
        NetworkID:    "200200",
        ChainID:      200200,
        Name:         "Zoo Network",
        TokenSymbol:  "ZOO",
        BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM",
    },
    "spc-mainnet": {
        NetworkID:    "36911",
        ChainID:      36911,
        Name:         "Sparkle Pony Club",
        TokenSymbol:  "SPC",
        BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1",
    },
}

// ExtractionOptions configures what data to extract
type ExtractionOptions struct {
    IncludeHeaders      bool
    IncludeBodies       bool
    IncludeReceipts     bool
    IncludeState        bool
    IncludeTransactions bool
    IncludeMetadata     bool
    
    // Advanced options
    StartBlock      uint64
    EndBlock        uint64
    TargetAddresses []string // Only extract state for these addresses
}

// ExtractionResult contains statistics about an extraction
type ExtractionResult struct {
    TotalKeys       int
    KeysByType      map[KeyType]int
    HighestBlock    uint64
    LowestBlock     uint64
    Duration        float64
    ExtractedState  map[string]AccountState // Address -> State
}

// AccountState represents the state of an account
type AccountState struct {
    Address  string
    Balance  string // Hex string
    Nonce    uint64
    CodeHash string
    Storage  map[string]string // Storage slot -> value
}

// BlockInfo contains basic block information
type BlockInfo struct {
    Number     uint64
    Hash       string
    ParentHash string
    Timestamp  uint64
    TxCount    int
}

// ParseBlockNumber extracts block number from a value
func ParseBlockNumber(data []byte) (uint64, error) {
    if len(data) < 8 {
        return 0, fmt.Errorf("data too short for block number: %d bytes", len(data))
    }
    return binary.BigEndian.Uint64(data[:8]), nil
}

// GetKeyTypeName returns a human-readable name for a key type
func GetKeyTypeName(keyType byte) string {
    if info, ok := KnownKeyTypes[KeyType(keyType)]; ok {
        return info.Name
    }
    return fmt.Sprintf("unknown(0x%02x)", keyType)
}