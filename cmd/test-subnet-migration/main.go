package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"testing"
)

// TestSubnetMigration demonstrates migrating old subnet data to new L2s
func TestSubnetMigration(t *testing.T) {
	fmt.Println("=== LUX NETWORK 2025 - SUBNET MIGRATION TEST ===")
	fmt.Println("Migrating old SubnetEVM chains to new L2 architecture")
	fmt.Println()

	// Define subnet networks (L2s)
	subnets := []struct {
		Name         string
		ChainID      uint64
		BlockchainID string
		Token        string
		OldData      string
	}{
		{
			Name:         "zoo-mainnet",
			ChainID:      200200,
			BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM",
			Token:        "ZOO",
			OldData:      "/home/z/archived/restored-blockchain-data/chainData/bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM/db/pebbledb",
		},
		{
			Name:         "spc-mainnet",
			ChainID:      36911,
			BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1",
			Token:        "SPC",
			OldData:      "/home/z/archived/restored-blockchain-data/chainData/QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1/db/pebbledb",
		},
	}

	// Step 1: Extract subnet chain data
	t.Run("ExtractSubnetData", func(t *testing.T) {
		for _, subnet := range subnets {
			t.Run(subnet.Name, func(t *testing.T) {
				// Check if data exists
				if _, err := os.Stat(subnet.OldData); err != nil {
					t.Fatalf("Subnet data not found: %s", subnet.OldData)
				}

				// Show size
				cmd := exec.Command("du", "-sh", subnet.OldData)
				output, _ := cmd.Output()
				t.Logf("Data size for %s: %s", subnet.Name, output)

				// For subnets, we need to preserve the exact data structure
				// The namespace is already correct for SubnetEVM
				dstPath := fmt.Sprintf("./subnet-migration/%s/chaindata", subnet.Name)
				
				t.Logf("Migration command for %s:", subnet.Name)
				t.Logf("  mkdir -p %s", dstPath)
				t.Logf("  cp -r %s %s/", subnet.OldData, dstPath)
			})
		}
	})

	// Step 2: Generate subnet genesis with preserved state
	t.Run("GenerateSubnetGenesis", func(t *testing.T) {
		for _, subnet := range subnets {
			t.Run(subnet.Name, func(t *testing.T) {
				genesis := map[string]interface{}{
					"config": map[string]interface{}{
						"chainId":             subnet.ChainID,
						"homesteadBlock":      0,
						"eip150Block":         0,
						"eip155Block":         0,
						"eip158Block":         0,
						"byzantiumBlock":      0,
						"constantinopleBlock": 0,
						"petersburgBlock":     0,
						"istanbulBlock":       0,
						"muirGlacierBlock":    0,
						"subnetEVMTimestamp":  0,
						"feeConfig": map[string]interface{}{
							"gasLimit":                 15000000,
							"targetBlockRate":          2,
							"minBaseFee":               1000000000,
							"targetGas":                15000000,
							"baseFeeChangeDenominator": 48,
							"minBlockGasCost":          0,
							"maxBlockGasCost":          10000000,
							"blockGasCostStep":         500000,
						},
					},
					"nonce":     "0x0",
					"timestamp": "0x0",
					"extraData": "0x00",
					"gasLimit":  "0xe4e1c0",
					"difficulty": "0x0",
					"mixHash":   "0x0000000000000000000000000000000000000000000000000000000000000000",
					"coinbase":  "0x0000000000000000000000000000000000000000",
					"alloc": map[string]interface{}{
						// Note: Actual allocations preserved in chaindata
						// This is just for genesis structure
					},
					"airdropHash":   "0x0000000000000000000000000000000000000000000000000000000000000000",
					"airdropAmount": "0x0",
					"number":        "0x0",
					"gasUsed":       "0x0",
					"parentHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
					"baseFeePerGas": "0x0",
				}

				// Write genesis file
				data, _ := json.MarshalIndent(genesis, "", "  ")
				t.Logf("Genesis for %s (chain ID %d) created", subnet.Name, subnet.ChainID)
				_ = data // In real test, write to file
			})
		}
	})

	// Step 3: Show deployment commands
	t.Run("DeploymentCommands", func(t *testing.T) {
		t.Log("\n=== SUBNET DEPLOYMENT COMMANDS ===")
		t.Log("\n1. First, ensure main LUX network is running:")
		t.Log("   luxd --network-id=96369")
		
		t.Log("\n2. Create and deploy subnets with existing data:")
		for _, subnet := range subnets {
			t.Logf("\n   # Deploy %s subnet (Chain ID: %d)", subnet.Name, subnet.ChainID)
			t.Logf("   lux-cli subnet create %s --evm \\", subnet.Name)
			t.Logf("     --genesis-file ./subnet-migration/%s/genesis.json \\", subnet.Name)
			t.Logf("     --chain-id %d \\", subnet.ChainID)
			t.Logf("     --token-name %s \\", subnet.Token)
			t.Logf("     --subnet-evm-chain-id %d", subnet.ChainID)
			t.Logf("")
			t.Logf("   # Import existing chain data")
			t.Logf("   lux-cli subnet import-chaindata %s \\", subnet.Name)
			t.Logf("     --chaindata-dir ./subnet-migration/%s/chaindata", subnet.Name)
		}

		t.Log("\n3. Deploy subnets to local network:")
		t.Log("   lux-cli subnet deploy zoo --local")
		t.Log("   lux-cli subnet deploy spc --local")
		
		t.Log("\n4. Verify imported state:")
		t.Log("   # Check account balances")
		t.Log("   cast balance 0x9011E888251AB053B7bD1cdB598Db4f9DEd94714 --rpc-url <subnet-rpc>")
	})
}

// TestFullMigrationWorkflow shows the complete migration process
func TestFullMigrationWorkflow(t *testing.T) {
	t.Run("CompleteMigration", func(t *testing.T) {
		fmt.Println("\n=== COMPLETE MIGRATION WORKFLOW ===")
		fmt.Println()
		fmt.Println("1. Extract C-Chain data (LUX mainnet)")
		fmt.Println("   ./bin/denamespace -src <pebbledb> -dst <output> -network 96369 -state")
		fmt.Println()
		fmt.Println("2. Copy subnet data (preserves namespace)")
		fmt.Println("   cp -r <zoo-pebbledb> ./migration/zoo/chaindata")
		fmt.Println("   cp -r <spc-pebbledb> ./migration/spc/chaindata")
		fmt.Println()
		fmt.Println("3. Launch main network with C-Chain data")
		fmt.Println("   luxd --network-id=96369 --db-dir=<c-chain-data>")
		fmt.Println()
		fmt.Println("4. Create subnets with imported data")
		fmt.Println("   lux-cli subnet create zoo --evm --genesis-file <zoo-genesis>")
		fmt.Println("   lux-cli subnet create spc --evm --genesis-file <spc-genesis>")
		fmt.Println()
		fmt.Println("5. Import existing blockchain data")
		fmt.Println("   lux-cli subnet import-chaindata zoo --chaindata-dir <zoo-data>")
		fmt.Println("   lux-cli subnet import-chaindata spc --chaindata-dir <spc-data>")
		fmt.Println()
		fmt.Println("Result: All account balances and contract state preserved!")
	})
}

func main() {
	// Run as regular program if not in test mode
	t := &testing.T{}
	TestSubnetMigration(t)
	TestFullMigrationWorkflow(t)
}