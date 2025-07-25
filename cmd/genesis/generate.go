package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// generateCmd creates the genesis files for all chains
func generateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate genesis files for P, C, and X chains",
		Long: `Generate genesis files for all chains with proper configuration.
		
This command creates:
- P-Chain genesis with validators
- C-Chain genesis with allocations and EVM config
- X-Chain genesis with initial supply

Output structure:
  configs/{network}/P/genesis.json
  configs/{network}/C/genesis.json
  configs/{network}/X/genesis.json`,
		RunE: runGenerate,
	}
	
	cmd.Flags().String("network", "mainnet", "Network to generate for (mainnet/testnet)")
	cmd.Flags().String("output", "", "Output directory (default: configs/{network})")
	cmd.Flags().Bool("with-allocations", true, "Include account allocations")
	
	return cmd
}

func runGenerate(cmd *cobra.Command, args []string) error {
	network, _ := cmd.Flags().GetString("network")
	outputDir, _ := cmd.Flags().GetString("output")
	withAllocations, _ := cmd.Flags().GetBool("with-allocations")
	
	if outputDir == "" {
		outputDir = filepath.Join("configs", network)
	}
	
	fmt.Printf("🔧 Generating genesis files for %s\n", network)
	fmt.Printf("   Output directory: %s\n", outputDir)
	
	// Create directories
	for _, chain := range []string{"P", "C", "X"} {
		chainDir := filepath.Join(outputDir, chain)
		if err := os.MkdirAll(chainDir, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", chain, err)
		}
	}
	
	// Generate P-Chain genesis
	if err := generatePChainGenesis(filepath.Join(outputDir, "P", "genesis.json"), network); err != nil {
		return fmt.Errorf("failed to generate P-Chain genesis: %w", err)
	}
	
	// Generate C-Chain genesis
	if err := generateCChainGenesis(filepath.Join(outputDir, "C", "genesis.json"), network, withAllocations); err != nil {
		return fmt.Errorf("failed to generate C-Chain genesis: %w", err)
	}
	
	// Generate X-Chain genesis
	if err := generateXChainGenesis(filepath.Join(outputDir, "X", "genesis.json"), network); err != nil {
		return fmt.Errorf("failed to generate X-Chain genesis: %w", err)
	}
	
	fmt.Println("✅ Genesis generation complete!")
	fmt.Printf("   Files created in: %s\n", outputDir)
	
	return nil
}

func generatePChainGenesis(outputPath string, network string) error {
	fmt.Println("📝 Generating P-Chain genesis...")
	
	genesis := map[string]interface{}{
		"networkID": getNetworkID(network),
		"allocations": []interface{}{},
		"startTime": time.Now().Unix(),
		"initialStakeDuration": 31536000, // 1 year
		"initialStakeDurationOffset": 5400,
		"initialStakedFunds": []interface{}{},
		"initialStakers": []interface{}{},
		"cChainGenesis": "",
		"message": "Lux Network Genesis",
	}
	
	// Add validators if mainnet
	if network == "mainnet" {
		// Add initial validators here
		validators := []map[string]interface{}{
			{
				"nodeID": "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
				"startTime": time.Now().Unix(),
				"endTime": time.Now().Add(365 * 24 * time.Hour).Unix(),
				"weight": 100000000000000, // 100k LUX
			},
		}
		genesis["initialStakers"] = validators
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(outputPath, data, 0644)
}

func generateCChainGenesis(outputPath string, network string, withAllocations bool) error {
	fmt.Println("📝 Generating C-Chain genesis...")
	
	chainID := getChainID(network)
	
	genesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId": chainID,
			"homesteadBlock": 0,
			"daoForkBlock": 0,
			"daoForkSupport": true,
			"eip150Block": 0,
			"eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"eip155Block": 0,
			"eip158Block": 0,
			"byzantiumBlock": 0,
			"constantinopleBlock": 0,
			"petersburgBlock": 0,
			"istanbulBlock": 0,
			"muirGlacierBlock": 0,
			"apricotPhase1BlockTimestamp": 1607144400, // Dec 5, 2020
			"apricotPhase2BlockTimestamp": 1607144400,
			"apricotPhase3BlockTimestamp": 1607144400,
			"apricotPhase4BlockTimestamp": 1607144400,
			"apricotPhase5BlockTimestamp": 1607144400,
			"apricotPhasePre6BlockTimestamp": 1607144400,
			"apricotPhase6BlockTimestamp": 1607144400,
			"apricotPhasePost6BlockTimestamp": 1607144400,
			"banffBlockTimestamp": 1607144400,
			"cortinaBlockTimestamp": 1607144400,
			"durangoBlockTimestamp": 1607144400,
			"etnaBlockTimestamp": 1607144400,
			"cancunTime": 1607144400,
		},
		"nonce": "0x0",
		"timestamp": "0x0",
		"extraData": "0x00",
		"gasLimit": "0x7A1200", // 8M gas
		"difficulty": "0x0",
		"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"alloc": map[string]interface{}{},
		"number": "0x0",
		"gasUsed": "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	
	// Add allocations
	if withAllocations {
		// Treasury address
		genesis["alloc"].(map[string]interface{})["0x9011E888251AB053B7bD1cdB598Db4f9DEd94714"] = map[string]interface{}{
			"balance": "2000000000000000000000000000000000", // 2 trillion
		}
		
		// Test account for development
		if network == "testnet" {
			genesis["alloc"].(map[string]interface{})["0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"] = map[string]interface{}{
				"balance": "100000000000000000000000", // 100k for testing
			}
		}
		
		// Load existing allocations if available
		allocFile := fmt.Sprintf("data/allocations_%s.json", network)
		if data, err := ioutil.ReadFile(allocFile); err == nil {
			var allocations map[string]interface{}
			if err := json.Unmarshal(data, &allocations); err == nil {
				for addr, alloc := range allocations {
					genesis["alloc"].(map[string]interface{})[addr] = alloc
				}
			}
		}
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(outputPath, data, 0644)
}

func generateXChainGenesis(outputPath string, network string) error {
	fmt.Println("📝 Generating X-Chain genesis...")
	
	genesis := map[string]interface{}{
		"networkID": getNetworkID(network),
		"allocations": []interface{}{
			{
				"ethAddr": "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714",
				"luxAddr": "X-lux1jqg73pqj52pqhae63k4vs6cf7lkwj3csy6kzw",
				"initialAmount": 360000000000000000,
				"unlockSchedule": []interface{}{
					{
						"amount": 10000000000000,
						"locktime": time.Now().Unix(),
					},
				},
			},
		},
		"startTime": time.Now().Unix(),
		"initialStakeDuration": 31536000,
		"initialStakeDurationOffset": 5400,
		"initialStakedFunds": []string{
			"X-lux1jqg73pqj52pqhae63k4vs6cf7lkwj3csy6kzw",
		},
		"initialStakers": []interface{}{},
		"cChainGenesis": "",
		"message": "Lux Network X-Chain Genesis",
	}
	
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(outputPath, data, 0644)
}

func getNetworkID(network string) int {
	switch network {
	case "mainnet":
		return 96369
	case "testnet":
		return 96368
	default:
		return 96369
	}
}

func getChainID(network string) int {
	switch network {
	case "mainnet":
		return 96369
	case "testnet":
		return 96368
	default:
		return 96369
	}
}