package genesis

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
)

// XChainGenesis represents X-Chain genesis configuration
type XChainGenesis struct {
	NetworkID             uint32         `json:"networkID"`
	Allocations          []interface{}   `json:"allocations"`
	StartTime            uint64          `json:"startTime"`
	InitialStakeDuration uint64          `json:"initialStakeDuration"`
	Message              string          `json:"message"`
}

// XChainAllocation represents an X-Chain allocation
type XChainAllocation struct {
	Address string   `json:"address"`
	Balance *big.Int `json:"balance"`
}

// BuildXChainGenesis creates X-Chain genesis for a network
func BuildXChainGenesis(network string, airdropCSVPath string) (*XChainGenesis, error) {
	networkID := GetNetworkID(network)
	
	genesis := &XChainGenesis{
		NetworkID:             networkID,
		Allocations:          make([]interface{}, 0),
		StartTime:            1577836800,  // Jan 1, 2020
		InitialStakeDuration: 31536000,    // 1 year
		Message:              fmt.Sprintf("Lux Network X-Chain Genesis - %s", network),
	}

	// Load airdrops if provided
	if airdropCSVPath != "" {
		allocations, err := LoadXChainAirdrops(airdropCSVPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load airdrops: %w", err)
		}
		
		// Convert to interface{} for JSON marshaling
		for _, alloc := range allocations {
			genesis.Allocations = append(genesis.Allocations, alloc)
		}
	}

	return genesis, nil
}

// LoadXChainAirdrops loads X-Chain allocations from CSV
func LoadXChainAirdrops(csvPath string) ([]XChainAllocation, error) {
	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comment = '#'  // Skip comment lines
	reader.FieldsPerRecord = -1  // Variable number of fields
	
	allocations := make([]XChainAllocation, 0)
	headerFound := false

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue  // Skip problematic lines
		}

		// Skip until we find the header
		if !headerFound {
			if len(record) > 0 && strings.HasPrefix(record[0], "rank") {
				headerFound = true
			}
			continue
		}

		// Expected format: rank,address,balance_lux,balance_wei,balance_hex,percentage
		if len(record) < 3 {
			continue
		}

		// Skip empty or invalid entries
		if record[0] == "" || record[1] == "" || record[2] == "" {
			continue
		}

		address := strings.TrimSpace(record[1])
		balanceLux := strings.TrimSpace(record[2])

		// Remove commas from balance
		balanceLux = strings.ReplaceAll(balanceLux, ",", "")

		// Parse LUX amount
		luxAmount := new(big.Float)
		if _, ok := luxAmount.SetString(balanceLux); !ok {
			continue  // Skip invalid amounts
		}

		// Convert to wei (9 decimals)
		weiAmount := new(big.Float).Mul(luxAmount, big.NewFloat(1e9))
		balance := new(big.Int)
		weiAmount.Int(balance)

		allocations = append(allocations, XChainAllocation{
			Address: address,
			Balance: balance,
		})
	}

	return allocations, nil
}

// GetNetworkID returns the network ID for a given network name
func GetNetworkID(network string) uint32 {
	switch network {
	case "mainnet":
		return 96369
	case "testnet":
		return 96368
	default:
		return 1 // default network ID
	}
}