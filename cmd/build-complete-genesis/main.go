package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/genesis/pkg/genesis/config"
	"github.com/luxfi/genesis/pkg/genesis/validator"
)

func main() {
	var (
		outputDir   = flag.String("output", "output", "Output directory for genesis files")
		networkName = flag.String("network", "mainnet", "Network name (mainnet or testnet)")
	)
	flag.Parse()

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Build mainnet or testnet
	if *networkName == "mainnet" {
		if err := buildMainnet(*outputDir); err != nil {
			log.Fatalf("Failed to build mainnet: %v", err)
		}
	} else {
		if err := buildTestnet(*outputDir); err != nil {
			log.Fatalf("Failed to build testnet: %v", err)
		}
	}
}

func buildMainnet(outputDir string) error {
	fmt.Println("Building Lux Mainnet Genesis...")

	// Create genesis builder
	builder, err := genesis.NewBuilder("mainnet")
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	// 1. Load C-Chain genesis from existing config
	cchainGenesisPath := "chaindata/configs/lux-mainnet-96369/genesis.json"
	fmt.Printf("Loading C-Chain genesis from %s...\n", cchainGenesisPath)
	
	cchainData, err := ioutil.ReadFile(cchainGenesisPath)
	if err != nil {
		return fmt.Errorf("failed to read C-Chain genesis: %w", err)
	}

	// Parse and extract allocations
	var cchainGenesis map[string]interface{}
	if err := json.Unmarshal(cchainData, &cchainGenesis); err != nil {
		return fmt.Errorf("failed to parse C-Chain genesis: %w", err)
	}

	// Set the C-Chain genesis directly
	builder.SetCChainGenesis(string(cchainData))

	// 2. Add validator allocations (locked for staking)
	fmt.Println("Adding validator allocations...")
	validatorConfig, err := ioutil.ReadFile("configs/mainnet-validators.json")
	if err != nil {
		return fmt.Errorf("failed to read validator config: %w", err)
	}

	var validators []validator.ValidatorInfo
	if err := json.Unmarshal(validatorConfig, &validators); err != nil {
		return fmt.Errorf("failed to parse validator config: %w", err)
	}

	// Add validators to P-Chain with locked allocations
	for i, v := range validators {
		// Add validator to staker set
		builder.AddStaker(genesis.StakerConfig{
			NodeID:            v.NodeID,
			ETHAddress:        v.ETHAddress,
			PublicKey:         v.PublicKey,
			ProofOfPossession: v.ProofOfPossession,
			Weight:            v.Weight,
			DelegationFee:     v.DelegationFee,
		})

		// Add locked allocation for validator (2M LUX each)
		stakingAmount := new(big.Int).SetUint64(2000000000000000) // 2M LUX in wei
		
		// Create vesting schedule starting Jan 1, 2020
		vestingConfig := &allocation.UnlockScheduleConfig{
			TotalAmount:  stakingAmount,
			StartDate:    time.Unix(1577836800, 0), // Jan 1, 2020 00:00:00 UTC
			Duration:     365 * 24 * time.Hour,     // 1 year
			Periods:      1,                        // Single unlock after 1 year
			CliffPeriods: 0,                        // No cliff
		}

		if err := builder.AddVestedAllocation(v.ETHAddress, vestingConfig); err != nil {
			return fmt.Errorf("failed to add validator %d allocation: %w", i, err)
		}
	}

	// 3. Load X-Chain airdrops from 7777 (excluding treasury)
	fmt.Println("Loading X-Chain airdrops from 7777...")
	airdropPath := "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet-no-treasury.csv"
	
	// For X-Chain, we need to create a separate genesis
	// This will be handled in a separate X-Chain genesis builder
	
	// 4. Build final genesis
	fmt.Println("Building final genesis...")
	mainGenesis, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build genesis: %w", err)
	}

	// Save main genesis
	genesisPath := filepath.Join(outputDir, "genesis-mainnet-96369.json")
	if err := builder.SaveToFile(mainGenesis, genesisPath); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}

	// Print summary
	fmt.Printf("\nMainnet Genesis Summary:\n")
	fmt.Printf("  Network ID: %d\n", mainGenesis.NetworkID)
	fmt.Printf("  Validators: %d\n", len(mainGenesis.InitialStakers))
	fmt.Printf("  Total P-Chain allocations: %d\n", len(mainGenesis.Allocations))
	fmt.Printf("  Total supply: %s LUX\n", formatLux(builder.GetTotalSupply()))
	fmt.Printf("  Output: %s\n", genesisPath)

	// Build X-Chain genesis separately
	if err := buildXChainGenesis(outputDir, "mainnet", airdropPath); err != nil {
		return fmt.Errorf("failed to build X-Chain genesis: %w", err)
	}

	// Prepare Zoo L2 configurations
	if err := prepareZooL2Configs(outputDir, "mainnet"); err != nil {
		return fmt.Errorf("failed to prepare Zoo L2 configs: %w", err)
	}

	return nil
}

func buildTestnet(outputDir string) error {
	fmt.Println("Building Lux Testnet Genesis...")

	// Create genesis builder
	builder, err := genesis.NewBuilder("testnet")
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}

	// Load testnet C-Chain genesis
	cchainGenesisPath := "chaindata/configs/lux-testnet-96368/genesis.json"
	fmt.Printf("Loading C-Chain genesis from %s...\n", cchainGenesisPath)
	
	cchainData, err := ioutil.ReadFile(cchainGenesisPath)
	if err != nil {
		return fmt.Errorf("failed to read C-Chain genesis: %w", err)
	}

	builder.SetCChainGenesis(string(cchainData))

	// Use same validators but for testnet
	// ... similar to mainnet but with testnet configurations ...

	mainGenesis, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build genesis: %w", err)
	}

	genesisPath := filepath.Join(outputDir, "genesis-testnet-96368.json")
	if err := builder.SaveToFile(mainGenesis, genesisPath); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}

	fmt.Printf("\nTestnet Genesis created: %s\n", genesisPath)

	// Build testnet X-Chain and Zoo L2s
	if err := buildXChainGenesis(outputDir, "testnet", ""); err != nil {
		return fmt.Errorf("failed to build X-Chain genesis: %w", err)
	}

	if err := prepareZooL2Configs(outputDir, "testnet"); err != nil {
		return fmt.Errorf("failed to prepare Zoo L2 configs: %w", err)
	}

	return nil
}

func buildXChainGenesis(outputDir, network, airdropPath string) error {
	fmt.Printf("\nBuilding X-Chain genesis for %s...\n", network)

	// Get network configuration
	netConfig, err := config.GetNetwork(network)
	if err != nil {
		return fmt.Errorf("failed to get network config: %w", err)
	}

	// X-Chain genesis structure
	xchainGenesis := map[string]interface{}{
		"networkID": netConfig.ID,
		"allocations": []interface{}{},
		"startTime": 1577836800, // Jan 1, 2020
		"initialStakeDuration": 31536000, // 1 year
		"message": fmt.Sprintf("Lux Network X-Chain Genesis - %s", network),
	}

	// Load airdrops if provided
	if airdropPath != "" && fileExists(airdropPath) {
		fmt.Printf("Loading airdrops from %s...\n", airdropPath)
		// Parse CSV and add to allocations
		// This would need proper CSV parsing and X-Chain allocation format
	}

	// Add NFT allocations
	// Classic Lux NFTs configuration would go here

	// Save X-Chain genesis
	xchainPath := filepath.Join(outputDir, fmt.Sprintf("xchain-genesis-%s.json", network))
	data, err := json.MarshalIndent(xchainGenesis, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal X-Chain genesis: %w", err)
	}

	if err := ioutil.WriteFile(xchainPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write X-Chain genesis: %w", err)
	}

	fmt.Printf("X-Chain genesis created: %s\n", xchainPath)
	return nil
}

func prepareZooL2Configs(outputDir, network string) error {
	fmt.Printf("\nPreparing Zoo L2 configurations for %s...\n", network)

	// Define Zoo networks
	zooNetworks := map[string]struct {
		chainID   string
		name      string
		configDir string
	}{
		"mainnet": {
			chainID:   "200200",
			name:      "zoo-mainnet",
			configDir: "chaindata/configs/zoo-mainnet-200200",
		},
		"testnet": {
			chainID:   "200201", 
			name:      "zoo-testnet",
			configDir: "chaindata/configs/zoo-testnet-200201",
		},
	}

	zooConfig, ok := zooNetworks[network]
	if !ok {
		return fmt.Errorf("unknown network: %s", network)
	}

	// Load Zoo genesis
	zooGenesisPath := filepath.Join(zooConfig.configDir, "genesis.json")
	if fileExists(zooGenesisPath) {
		fmt.Printf("Loading Zoo genesis from %s...\n", zooGenesisPath)
		
		// Copy to output directory
		zooData, err := ioutil.ReadFile(zooGenesisPath)
		if err != nil {
			return fmt.Errorf("failed to read Zoo genesis: %w", err)
		}

		outputPath := filepath.Join(outputDir, fmt.Sprintf("zoo-%s-genesis.json", network))
		if err := ioutil.WriteFile(outputPath, zooData, 0644); err != nil {
			return fmt.Errorf("failed to write Zoo genesis: %w", err)
		}

		fmt.Printf("Zoo L2 genesis created: %s\n", outputPath)
	}

	// Create subnet configuration
	subnetConfig := map[string]interface{}{
		"subnetID": fmt.Sprintf("zoo-%s-subnet", network),
		"chainID": zooConfig.chainID,
		"vmID": "evm",
		"genesis": fmt.Sprintf("zoo-%s-genesis.json", network),
	}

	subnetPath := filepath.Join(outputDir, fmt.Sprintf("zoo-%s-subnet.json", network))
	data, err := json.MarshalIndent(subnetConfig, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal subnet config: %w", err)
	}

	if err := ioutil.WriteFile(subnetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write subnet config: %w", err)
	}

	fmt.Printf("Zoo L2 subnet config created: %s\n", subnetPath)
	return nil
}

func formatLux(wei *big.Int) string {
	if wei == nil {
		return "0"
	}
	lux := new(big.Float).SetInt(wei)
	lux.Quo(lux, big.NewFloat(1e9))
	return lux.Text('f', 2)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}