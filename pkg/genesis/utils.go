package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
)

// FormatLux converts wei to LUX string representation
func FormatLux(wei *big.Int) string {
	if wei == nil {
		return "0"
	}
	lux := new(big.Float).SetInt(wei)
	lux.Quo(lux, big.NewFloat(1e9))
	return lux.Text('f', 2)
}

// SaveJSON saves data as formatted JSON to file
func SaveJSON(data interface{}, filepath string) error {
	jsonData, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return ioutil.WriteFile(filepath, jsonData, 0644)
}

// LoadJSON loads JSON data from file
func LoadJSON(filepath string, v interface{}) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateDirectory creates a directory if it doesn't exist
func CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// GetDefaultPath returns the default path for a resource
func GetDefaultPath(network, resource string) string {
	paths := map[string]map[string]string{
		"mainnet": {
			"validators": "configs/mainnet-validators.json",
			"cchain":     "chaindata/configs/lux-mainnet-96369/genesis.json",
			"airdrop":    "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet-no-treasury.csv",
		},
		"testnet": {
			"validators": "configs/testnet-validators.json",
			"cchain":     "chaindata/configs/lux-testnet-96368/genesis.json",
			"airdrop":    "",
		},
	}

	if networkPaths, ok := paths[network]; ok {
		if path, ok := networkPaths[resource]; ok {
			return path
		}
	}

	return ""
}