package rawdb

// Constants exported from github.com/ethereum/go-ethereum/core/rawdb/schema.go
// These are needed for migration tools but are not exported in the upstream

var (
	// Database key prefixes
	HeaderPrefix       = []byte("h") // headerPrefix + num (uint64 big endian) + hash -> header
	HeaderTDSuffix     = []byte("t") // headerPrefix + num (uint64 big endian) + hash + headerTDSuffix -> td
	HeaderHashSuffix   = []byte("n") // headerPrefix + num (uint64 big endian) + headerHashSuffix -> hash
	HeaderNumberPrefix = []byte("H") // headerNumberPrefix + hash -> num (uint64 big endian)
	BlockBodyPrefix    = []byte("b") // blockBodyPrefix + num (uint64 big endian) + hash -> block body
	BlockReceiptsPrefix = []byte("r") // blockReceiptsPrefix + num (uint64 big endian) + hash -> block receipts
	TxLookupPrefix     = []byte("l") // txLookupPrefix + hash -> transaction/receipt lookup metadata
	
	// Fixed keys
	HeadHeaderKey = []byte("LastHeader")
	HeadBlockKey  = []byte("LastBlock")
	HeadFastBlockKey = []byte("LastFast")
	
	// For compatibility
	HashPrefix = HeaderNumberPrefix // Alias for header number lookups
)