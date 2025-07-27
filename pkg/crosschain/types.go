package crosschain

import (
	"math/big"

	"github.com/luxfi/geth/common"
)

// TokenHolder represents a token holder
type TokenHolder struct {
	Address  common.Address `json:"address"`
	Balance  *big.Int       `json:"balance"`
	TokenIDs []int64        `json:"tokenIds,omitempty"` // For NFTs
	Note     string         `json:"note,omitempty"`
}

// BurnEvent represents a token burn event
type BurnEvent struct {
	From            common.Address `json:"from"`
	Amount          *big.Int       `json:"amount"`
	BlockNumber     uint64         `json:"blockNumber"`
	TransactionHash common.Hash    `json:"transactionHash"`
	Timestamp       uint64         `json:"timestamp,omitempty"`
}

// TransferEvent represents a token transfer
type TransferEvent struct {
	From            common.Address `json:"from"`
	To              common.Address `json:"to"`
	Value           *big.Int       `json:"value,omitempty"`
	TokenID         *big.Int       `json:"tokenId,omitempty"`
	BlockNumber     uint64         `json:"blockNumber"`
	TransactionHash common.Hash    `json:"transactionHash"`
}

// BlockInfo contains basic block information
type BlockInfo struct {
	Number     *big.Int `json:"number"`
	Hash       string   `json:"hash"`
	Timestamp  uint64   `json:"timestamp"`
	ParentHash string   `json:"parentHash"`
}

// ChainSnapshot represents a snapshot of chain data
type ChainSnapshot struct {
	ChainID         *big.Int       `json:"chainId"`
	BlockNumber     *big.Int       `json:"blockNumber"`
	Timestamp       uint64         `json:"timestamp"`
	TokenHolders    []TokenHolder  `json:"tokenHolders,omitempty"`
	BurnEvents      []BurnEvent    `json:"burnEvents,omitempty"`
	TransferEvents  []TransferEvent `json:"transferEvents,omitempty"`
}