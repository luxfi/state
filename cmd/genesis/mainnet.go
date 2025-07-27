package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// mainnetCmd prepares the 2025 mainnet launch
func mainnetCmd() *cobra.Command {
	var (
		dataDir    string
		outputDir  string
		dbType     string
		nodeCount  int
		skipVerify bool
	)

	cmd := &cobra.Command{
		Use:   "mainnet",
		Short: "Prepare 2025 Lux Mainnet launch with BadgerDB",
		Long: `Prepare the 2025 Lux Mainnet launch by extracting state from historic 
PebbleDB data and configuring for BadgerDB backend with 21-node bootstrap network.

This command combines:
- State extraction from network 96369 (lux-mainnet)
- Genesis file preparation for P/C/X chains
- Bootstrap node configuration (21 nodes)
- Deployment package creation with BadgerDB config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return prepareMainnet(dataDir, outputDir, dbType, nodeCount, skipVerify)
		},
	}

	cmd.Flags().StringVar(&dataDir, "data", "chaindata/lux-mainnet-96369", "Historic chain data directory")
	cmd.Flags().StringVar(&outputDir, "output", "mainnet-2025", "Output directory for mainnet files")
	cmd.Flags().StringVar(&dbType, "db-type", "badgerdb", "Target database type")
	cmd.Flags().IntVar(&nodeCount, "nodes", 21, "Number of bootstrap nodes")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Skip verification steps")

	return cmd
}

func prepareMainnet(dataDir, outputDir, dbType string, nodeCount int, skipVerify bool) error {
	fmt.Println("🚀 Lux Network 2025 Mainnet Preparation")
	fmt.Println("=======================================")
	fmt.Printf("Data source: %s\n", dataDir)
	fmt.Printf("Output: %s\n", outputDir)
	fmt.Printf("Database: %s\n", dbType)
	fmt.Printf("Bootstrap nodes: %d\n", nodeCount)
	fmt.Println()

	// Step 1: Verify source data exists
	fmt.Println("📁 Step 1: Verifying source data...")
	pebbleDBPath := filepath.Join(dataDir, "db", "pebbledb")
	if _, err := os.Stat(pebbleDBPath); err != nil {
		return fmt.Errorf("source PebbleDB not found at %s: %w", pebbleDBPath, err)
	}
	fmt.Printf("✅ Found PebbleDB at: %s\n\n", pebbleDBPath)

	// Step 2: Extract state using existing extract command
	fmt.Println("📊 Step 2: Extracting blockchain state...")
	extractedDir := filepath.Join(outputDir, "extracted")
	
	// Use the existing extract state functionality
	extractStateCmd := &cobra.Command{}
	extractStateCmd.SetArgs([]string{pebbleDBPath, extractedDir})
	extractStateCmd.Flags().Int("network", 96369, "")
	extractStateCmd.Flags().Bool("state", true, "")
	
	if err := runExtractState(extractStateCmd, []string{pebbleDBPath, extractedDir}); err != nil {
		return fmt.Errorf("state extraction failed: %w", err)
	}
	fmt.Printf("✅ State extracted to: %s\n\n", extractedDir)

	// Step 3: Prepare genesis files
	fmt.Println("📄 Step 3: Preparing genesis files...")
	genesisDir := filepath.Join(outputDir, "genesis")
	
	// Use existing generate command
	generateCmd := &cobra.Command{}
	generateCmd.Flags().String("network", "mainnet", "")
	generateCmd.Flags().String("output", genesisDir, "")
	
	if err := runGenerate(generateCmd, []string{}); err != nil {
		return fmt.Errorf("genesis generation failed: %w", err)
	}
	fmt.Printf("✅ Genesis files created in: %s\n\n", genesisDir)

	// Step 4: Update bootstrap configuration
	fmt.Println("🌐 Step 4: Configuring bootstrap network...")
	bootstrapFile := filepath.Join(outputDir, "bootstrappers.json")
	
	if err := updateBootstrapNodes(bootstrapFile, nodeCount); err != nil {
		return fmt.Errorf("bootstrap configuration failed: %w", err)
	}
	fmt.Printf("✅ Bootstrap nodes configured: %s\n\n", bootstrapFile)

	// Step 5: Create deployment configuration
	fmt.Println("⚙️  Step 5: Creating deployment configuration...")
	configDir := filepath.Join(outputDir, "config")
	
	if err := createDeploymentConfig(configDir, dbType); err != nil {
		return fmt.Errorf("deployment config creation failed: %w", err)
	}
	fmt.Printf("✅ Deployment config created in: %s\n\n", configDir)

	// Step 6: Verification (optional)
	if !skipVerify {
		fmt.Println("✓ Step 6: Verifying configuration...")
		validateCmd := &cobra.Command{}
		validateCmd.Flags().String("network", "mainnet", "")
		
		if err := runValidate(validateCmd, []string{}); err != nil {
			fmt.Printf("⚠️  Warning: Validation failed: %v\n", err)
		} else {
			fmt.Println("✅ Configuration validated successfully")
		}
	}

	// Summary
	fmt.Println("\n✨ Mainnet preparation completed!")
	fmt.Printf("📂 Output directory: %s\n", outputDir)
	fmt.Println("\n📋 Directory contents:")
	fmt.Printf("  - %s/extracted/     # Extracted blockchain state\n", outputDir)
	fmt.Printf("  - %s/genesis/       # Genesis files (P/C/X chains)\n", outputDir)
	fmt.Printf("  - %s/config/        # Node configuration\n", outputDir)
	fmt.Printf("  - %s/bootstrappers.json  # Bootstrap nodes\n", outputDir)
	
	fmt.Println("\n🚀 Next steps:")
	fmt.Println("1. Review configuration in", outputDir)
	fmt.Println("2. Copy to node deployments")
	fmt.Println("3. Start nodes with:")
	fmt.Printf("   luxd --data-dir=%s/data --config-file=%s/config/node.json\n", outputDir, outputDir)

	return nil
}

func updateBootstrapNodes(outputFile string, nodeCount int) error {
	// Ensure we have the updated bootstrappers from the main genesis
	bootstrapSrc := "bootstrappers.json"
	
	// Check if source exists
	if _, err := os.Stat(bootstrapSrc); err == nil {
		// Copy existing bootstrappers
		data, err := os.ReadFile(bootstrapSrc)
		if err != nil {
			return err
		}
		
		// Ensure output directory exists
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return err
		}
		
		return os.WriteFile(outputFile, data, 0644)
	}
	
	// If no source, use the node genesis bootstrappers
	nodeSrc := filepath.Join("..", "..", "node", "genesis", "bootstrappers.json")
	if _, err := os.Stat(nodeSrc); err == nil {
		data, err := os.ReadFile(nodeSrc)
		if err != nil {
			return err
		}
		
		if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
			return err
		}
		
		return os.WriteFile(outputFile, data, 0644)
	}
	
	return fmt.Errorf("no bootstrappers.json found")
}

func createDeploymentConfig(configDir, dbType string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Node configuration
	nodeConfig := map[string]interface{}{
		"network-id":                     1,
		"db-type":                       dbType,
		"log-level":                     "info",
		"log-format":                    "json",
		"http-port":                     9650,
		"staking-port":                  9651,
		"staking-enabled":               true,
		"health-check-frequency":        "30s",
		"network-compression-type":      "zstd",
		"network-max-reconnect-delay":   "1m",
		"network-initial-reconnect-delay": "5s",
		"network-require-validator-to-connect": false,
	}

	// EVM configuration for BadgerDB
	evmConfig := map[string]interface{}{
		"database-type":              dbType,
		"log-level":                  "info",
		"rpc-gas-cap":                50000000,
		"rpc-tx-fee-cap":             100,
		"api-max-duration":           30000000000,
		"transaction-history":        0,
		"accepted-cache-size":        32,
		"use-standalone-database":    true,
		"http-body-limit":            33554432,
	}

	// Write configurations
	nodeFile := filepath.Join(configDir, "node.json")
	if err := writeJSON(nodeFile, nodeConfig); err != nil {
		return fmt.Errorf("failed to write node config: %w", err)
	}

	evmFile := filepath.Join(configDir, "evm.json")
	if err := writeJSON(evmFile, evmConfig); err != nil {
		return fmt.Errorf("failed to write EVM config: %w", err)
	}

	// Create launch script
	scriptContent := `#!/bin/bash
# Lux Network 2025 Mainnet Launch Script
# Database: BadgerDB

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CONFIG_DIR="$(dirname "$SCRIPT_DIR")/config"
DATA_DIR="$(dirname "$SCRIPT_DIR")/data"

echo "Starting Lux Network Node"
echo "========================"
echo "Network: Mainnet (ID: 1)"
echo "Database: BadgerDB"
echo "Config: $CONFIG_DIR/node.json"

exec luxd \
    --config-file="$CONFIG_DIR/node.json" \
    --data-dir="$DATA_DIR" \
    --plugin-dir="$DATA_DIR/plugins"
`
	
	scriptFile := filepath.Join(configDir, "..", "launch.sh")
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write launch script: %w", err)
	}

	return nil
}

func writeJSON(filename string, data interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}