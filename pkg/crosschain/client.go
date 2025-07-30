package crosschain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

// ChainConfig holds configuration for a blockchain
type ChainConfig struct {
	Name     string
	ChainID  *big.Int
	RPCURLs  []string
	CacheDir string
}

// Client provides access to cross-chain data with caching
type Client struct {
	config    *ChainConfig
	client    *ethclient.Client
	rpcClient *rpc.Client
	cacheDir  string
}

// NewClient creates a new cross-chain client
func NewClient(config *ChainConfig) (*Client, error) {
	// Ensure cache directory exists
	cacheDir := filepath.Join(config.CacheDir, config.Name)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	// Try RPC URLs until one works
	var client *ethclient.Client
	var rpcClient *rpc.Client
	var lastErr error

	for _, url := range config.RPCURLs {
		rpcClient, lastErr = rpc.Dial(url)
		if lastErr != nil {
			continue
		}
		client = ethclient.NewClient(rpcClient)

		// Test connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := client.ChainID(ctx)
		cancel()

		if err == nil {
			break
		}
		lastErr = err
	}

	if client == nil {
		return nil, fmt.Errorf("failed to connect to any RPC: %w", lastErr)
	}

	return &Client{
		config:    config,
		client:    client,
		rpcClient: rpcClient,
		cacheDir:  cacheDir,
	}, nil
}

// CacheKey generates a cache key for a request
func (c *Client) CacheKey(method string, params ...interface{}) string {
	// Create a unique key from method and params
	key := method
	for _, p := range params {
		key += fmt.Sprintf("_%v", p)
	}
	return key
}

// GetFromCache retrieves cached data
func (c *Client) GetFromCache(key string, result interface{}) (bool, error) {
	cachePath := filepath.Join(c.cacheDir, key+".json")

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return false, nil
	}

	// Read cache file
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return false, err
	}

	// Check cache age (24 hours)
	info, err := os.Stat(cachePath)
	if err != nil {
		return false, err
	}

	if time.Since(info.ModTime()) > 24*time.Hour {
		return false, nil // Cache expired
	}

	// Unmarshal data
	if err := json.Unmarshal(data, result); err != nil {
		return false, err
	}

	return true, nil
}

// SaveToCache saves data to cache
func (c *Client) SaveToCache(key string, data interface{}) error {
	cachePath := filepath.Join(c.cacheDir, key+".json")

	// Marshal data
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(cachePath, jsonData, 0644)
}

// GetLatestBlock returns the latest finalized block
func (c *Client) GetLatestBlock(ctx context.Context) (*big.Int, error) {
	// BSC finalizes after ~15 blocks, ETH after ~64
	safetyMargin := int64(15)
	if c.config.ChainID.Cmp(big.NewInt(1)) == 0 {
		safetyMargin = 64 // Ethereum
	}

	header, err := c.client.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}

	finalizedBlock := new(big.Int).Sub(header.Number, big.NewInt(safetyMargin))
	return finalizedBlock, nil
}

// GetTokenHolders retrieves token holders at a specific block
func (c *Client) GetTokenHolders(ctx context.Context, tokenAddress common.Address, blockNumber *big.Int) ([]TokenHolder, error) {
	// Check cache first
	cacheKey := c.CacheKey("token_holders", tokenAddress.Hex(), blockNumber.String())
	var holders []TokenHolder

	if found, _ := c.GetFromCache(cacheKey, &holders); found {
		return holders, nil
	}

	// This would require event log scanning
	// For now, return empty list with note
	holders = []TokenHolder{
		{
			Note: "Full holder scanning requires event log analysis",
		},
	}

	// Save to cache
	c.SaveToCache(cacheKey, holders)

	return holders, nil
}

// GetBurnEvents retrieves burn events for a token
func (c *Client) GetBurnEvents(ctx context.Context, tokenAddress common.Address, fromBlock, toBlock *big.Int) ([]BurnEvent, error) {
	// Check cache
	cacheKey := c.CacheKey("burn_events", tokenAddress.Hex(), fromBlock.String(), toBlock.String())
	var events []BurnEvent

	if found, _ := c.GetFromCache(cacheKey, &events); found {
		return events, nil
	}

	// Query would go here
	// For now, return empty with note
	events = []BurnEvent{}

	// Save to cache
	c.SaveToCache(cacheKey, events)

	return events, nil
}

// Close closes the client connection
func (c *Client) Close() {
	if c.rpcClient != nil {
		c.rpcClient.Close()
	}
}
