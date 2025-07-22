package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	var (
		nodeCount = flag.Int("nodes", 5, "Number of nodes in the network")
		withData  = flag.Bool("with-data", false, "Import C-Chain data after starting")
		withL2s   = flag.Bool("with-l2s", false, "Deploy L2 subnets after starting")
		dataPath  = flag.String("data-path", "", "Path to PebbleDB data to import")
	)
	flag.Parse()

	// Get paths relative to this script
	luxdPath := filepath.Join("bin", "luxd")
	cliPath := filepath.Join("bin", "lux")
	
	// Check binaries exist
	if _, err := os.Stat(luxdPath); os.IsNotExist(err) {
		log.Fatalf("luxd not found at %s. Please run 'make install' first.", luxdPath)
	}
	if _, err := os.Stat(cliPath); os.IsNotExist(err) {
		log.Fatalf("lux not found at %s. Please run 'make install' first.", cliPath)
	}

	// Create network
	fmt.Printf("Starting %d-node local network...\n", *nodeCount)
	
	// Step 1: Create network configuration
	networkName := fmt.Sprintf("genesis-test-%d", time.Now().Unix())
	cmd := exec.Command(cliPath, "network", "create",
		"--network-name", networkName,
		"--node-count", fmt.Sprintf("%d", *nodeCount),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to create network: %v", err)
	}

	// Step 2: Start the network
	cmd = exec.Command(cliPath, "network", "start",
		"--network-name", networkName,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to start network: %v", err)
	}

	fmt.Println("Network started successfully!")

	// Wait for network to be healthy
	fmt.Println("Waiting for network to be healthy...")
	time.Sleep(30 * time.Second)

	// Step 3: Import C-Chain data if requested
	if *withData && *dataPath != "" {
		fmt.Printf("Importing C-Chain data from %s...\n", *dataPath)
		cmd = exec.Command(cliPath, "blockchain", "import", "c-chain",
			"--db-path", *dataPath,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			log.Printf("Failed to import C-Chain data: %v", err)
		}
	}

	// Step 4: Deploy L2 subnets if requested
	if *withL2s {
		fmt.Println("Deploying L2 subnets...")
		
		subnets := []struct {
			name    string
			chainID string
			genesis string
		}{
			{"zoo", "200200", "chaindata/configs/zoo-mainnet-200200/genesis.json"},
			{"spc", "36911", "chaindata/configs/spc-mainnet-36911/genesis.json"},
			{"hanzo", "36963", "chaindata/configs/hanzo-mainnet-36963/genesis.json"},
		}

		for _, subnet := range subnets {
			fmt.Printf("Creating %s subnet...\n", subnet.name)
			cmd = exec.Command(cliPath, "subnet", "create",
				"--subnet-name", subnet.name,
				"--chain-id", subnet.chainID,
				"--genesis-file", subnet.genesis,
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			
			if err := cmd.Run(); err != nil {
				log.Printf("Failed to create %s subnet: %v", subnet.name, err)
			}
		}
	}

	fmt.Println("\nNetwork is ready!")
	fmt.Printf("RPC endpoint: http://localhost:9650\n")
	fmt.Printf("To stop the network: %s network stop --network-name %s\n", cliPath, networkName)
}