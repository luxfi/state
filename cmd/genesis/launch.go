package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// launchCmd launches a multi-node network
func launchCmd() *cobra.Command {
	var (
		network   string
		nodes     int
		dataDir   string
		clean     bool
		detached  bool
	)

	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch a multi-node Lux network",
		Long: `Launch a multi-node Lux network with proper configuration.
		
This command:
- Generates validator keys for each node
- Creates genesis files with proper consensus parameters
- Configures each node with unique ports
- Launches all nodes with BadgerDB backend
- Monitors health status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchNetwork(network, nodes, dataDir, clean, detached)
		},
	}

	cmd.Flags().StringVar(&network, "network", "local", "Network type (mainnet/testnet/local)")
	cmd.Flags().IntVar(&nodes, "nodes", 5, "Number of nodes (21 for mainnet, 11 for testnet, 5 for local)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "Data directory (default: ~/.luxd/networks/{network})")
	cmd.Flags().BoolVar(&clean, "clean", false, "Clean existing data before launch")
	cmd.Flags().BoolVar(&detached, "detached", false, "Run nodes in background")

	return cmd
}

func launchNetwork(network string, nodes int, dataDir string, clean bool, detached bool) error {
	// Set defaults based on network
	if nodes == 5 {
		switch network {
		case "mainnet":
			nodes = 21
		case "testnet":
			nodes = 11
		case "local":
			nodes = 5
		}
	}

	// Set data directory
	if dataDir == "" {
		homeDir, _ := os.UserHomeDir()
		dataDir = filepath.Join(homeDir, ".luxd", "networks", network)
	}

	fmt.Printf("🚀 Launching %d-node %s network\n", nodes, network)
	fmt.Printf("   Data directory: %s\n", dataDir)

	// Clean if requested
	if clean && dirExists(dataDir) {
		fmt.Println("🧹 Cleaning existing data...")
		os.RemoveAll(dataDir)
	}

	// Create directories
	os.MkdirAll(dataDir, 0755)
	stakingDir := filepath.Join(dataDir, "staking")
	genesisDir := filepath.Join(dataDir, "genesis")
	os.MkdirAll(stakingDir, 0755)
	os.MkdirAll(genesisDir, 0755)

	// Step 1: Generate validator keys
	fmt.Printf("\n📝 Generating %d validator keys...\n", nodes)
	validators := make([]map[string]string, nodes)
	
	for i := 0; i < nodes; i++ {
		nodeDir := filepath.Join(stakingDir, fmt.Sprintf("node%02d", i+1))
		os.MkdirAll(nodeDir, 0755)
		
		// Generate staking keys using luxd
		nodeID, err := generateStakingKeys(nodeDir)
		if err != nil {
			return fmt.Errorf("failed to generate keys for node %d: %w", i+1, err)
		}
		
		validators[i] = map[string]string{
			"nodeID": nodeID,
			"host":   fmt.Sprintf("127.0.0.1:%d", 9651+i*10),
		}
		
		fmt.Printf("   Node %02d: %s\n", i+1, nodeID)
	}

	// Step 2: Generate genesis with proper parameters
	fmt.Println("\n📋 Generating genesis configuration...")
	if err := generateNetworkGenesis(genesisDir, network, validators); err != nil {
		return fmt.Errorf("failed to generate genesis: %w", err)
	}

	// Step 3: Create node configurations
	fmt.Println("\n⚙️  Creating node configurations...")
	bootstrapIPs := make([]string, 0, 5)
	bootstrapIDs := make([]string, 0, 5)
	
	// Use first 5 nodes as bootstrappers
	for i := 0; i < 5 && i < nodes; i++ {
		bootstrapIPs = append(bootstrapIPs, validators[i]["host"])
		bootstrapIDs = append(bootstrapIDs, validators[i]["nodeID"])
	}
	
	for i := 0; i < nodes; i++ {
		if err := createNodeConfig(dataDir, i+1, network, genesisDir, stakingDir, bootstrapIPs, bootstrapIDs); err != nil {
			return fmt.Errorf("failed to create config for node %d: %w", i+1, err)
		}
	}

	// Step 4: Launch nodes
	fmt.Println("\n🔥 Launching nodes...")
	pidsFile := filepath.Join(dataDir, "pids")
	pids := make([]string, 0, nodes)
	
	for i := 0; i < nodes; i++ {
		pid, err := launchNode(dataDir, i+1, detached)
		if err != nil {
			return fmt.Errorf("failed to launch node %d: %w", i+1, err)
		}
		pids = append(pids, fmt.Sprintf("%d", pid))
		fmt.Printf("   Node %02d launched (PID: %d)\n", i+1, pid)
		
		// Stagger launches
		time.Sleep(2 * time.Second)
	}
	
	// Save PIDs
	os.WriteFile(pidsFile, []byte(strings.Join(pids, "\n")), 0644)

	// Step 5: Check health
	fmt.Println("\n💓 Waiting for nodes to become healthy...")
	time.Sleep(10 * time.Second)
	
	healthy := 0
	for i := 0; i < nodes; i++ {
		if isNodeHealthy(9630 + i*10) {
			healthy++
		}
	}
	
	fmt.Printf("\n✅ Network launched! %d/%d nodes healthy\n", healthy, nodes)
	fmt.Printf("\nUseful commands:\n")
	fmt.Printf("  Status: genesis network status --data-dir %s\n", dataDir)
	fmt.Printf("  Stop:   genesis network stop --data-dir %s\n", dataDir)
	fmt.Printf("  Logs:   tail -f %s/node01/logs/main.log\n", dataDir)
	
	return nil
}

func generateStakingKeys(nodeDir string) (string, error) {
	// Use genesis tool's key generation
	cmd := exec.Command("go", "run", "-", nodeDir)
	cmd.Stdin = strings.NewReader(`
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "github.com/luxfi/node/staking"
)

func main() {
    if len(os.Args) < 2 {
        panic("need output dir")
    }
    
    cert, key, err := staking.NewCertAndKeyBytes()
    if err != nil {
        panic(err)
    }
    
    nodeID, err := staking.CertToNodeID(cert)
    if err != nil {
        panic(err)
    }
    
    os.WriteFile(filepath.Join(os.Args[1], "staker.crt"), cert, 0600)
    os.WriteFile(filepath.Join(os.Args[1], "staker.key"), key, 0600)
    
    fmt.Print(nodeID.String())
}
`)
	
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(output)), nil
}

func generateNetworkGenesis(genesisDir string, network string, validators []map[string]string) error {
	// Create chain directories
	for _, chain := range []string{"P", "C", "X"} {
		chainDir := filepath.Join(genesisDir, chain)
		os.MkdirAll(chainDir, 0755)
	}
	
	// Get network ID
	networkID := uint32(1)
	switch network {
	case "mainnet":
		networkID = 96369
	case "testnet":
		networkID = 96368
	case "local":
		networkID = 12345
	}
	
	// Generate P-Chain genesis with validators
	pGenesis := map[string]interface{}{
		"networkID": networkID,
		"allocations": []interface{}{},
		"startTime": time.Now().Unix(),
		"initialStakeDuration": 31536000,
		"initialStakeDurationOffset": 5400,
		"initialStakedFunds": []string{},
		"initialStakers": []interface{}{},
		"cChainGenesis": "",
		"message": "Lux Network " + network,
	}
	
	// Add validators for mainnet/testnet
	if network != "local" {
		stakers := make([]interface{}, 0, len(validators))
		for _, val := range validators {
			stakers = append(stakers, map[string]interface{}{
				"nodeID": val["nodeID"],
				"rewardAddress": "X-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqz8hfvz",
				"delegationFee": 10000,
			})
		}
		pGenesis["initialStakers"] = stakers
	}
	
	// Write P-Chain genesis
	pData, _ := json.MarshalIndent(pGenesis, "", "  ")
	os.WriteFile(filepath.Join(genesisDir, "P", "genesis.json"), pData, 0644)
	
	// Generate C-Chain genesis
	cGenesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId": networkID,
			"homesteadBlock": 0,
			"eip150Block": 0,
			"eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
			"eip155Block": 0,
			"eip158Block": 0,
			"byzantiumBlock": 0,
			"constantinopleBlock": 0,
			"petersburgBlock": 0,
			"istanbulBlock": 0,
			"muirGlacierBlock": 0,
			"berlinBlock": 0,
			"londonBlock": 0,
		},
		"nonce": "0x0",
		"timestamp": "0x0",
		"extraData": "0x0",
		"gasLimit": "0x7A1200",
		"difficulty": "0x0",
		"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"alloc": map[string]interface{}{},
		"number": "0x0",
		"gasUsed": "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	
	// Add allocations for local network
	if network == "local" {
		cGenesis["alloc"] = map[string]interface{}{
			"0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC": map[string]string{
				"balance": "0x33b2e3c9fd0803ce8000000", // 1M LUX
			},
		}
	}
	
	cData, _ := json.MarshalIndent(cGenesis, "", "  ")
	os.WriteFile(filepath.Join(genesisDir, "C", "genesis.json"), cData, 0644)
	
	// Generate X-Chain genesis
	xGenesis := map[string]interface{}{
		"networkID": networkID,
		"allocations": []interface{}{},
		"startTime": time.Now().Unix(),
		"initialStakeDuration": 31536000,
		"initialStakeDurationOffset": 5400,
		"initialStakedFunds": []string{},
		"initialStakers": []interface{}{},
		"cChainGenesis": "",
		"message": "Lux Network " + network,
	}
	
	xData, _ := json.MarshalIndent(xGenesis, "", "  ")
	os.WriteFile(filepath.Join(genesisDir, "X", "genesis.json"), xData, 0644)
	
	return nil
}

func createNodeConfig(dataDir string, nodeNum int, network string, genesisDir string, stakingDir string, bootstrapIPs []string, bootstrapIDs []string) error {
	nodeDir := filepath.Join(dataDir, fmt.Sprintf("node%02d", nodeNum))
	os.MkdirAll(filepath.Join(nodeDir, "db"), 0755)
	os.MkdirAll(filepath.Join(nodeDir, "logs"), 0755)
	
	// Calculate ports
	httpPort := 9630 + (nodeNum-1)*10
	stakingPort := 9651 + (nodeNum-1)*10
	
	config := map[string]interface{}{
		"network-id":             network,
		"http-host":              "127.0.0.1",
		"http-port":              httpPort,
		"staking-port":           stakingPort,
		"db-dir":                 filepath.Join(nodeDir, "db"),
		"db-type":                "badgerdb",
		"log-dir":                filepath.Join(nodeDir, "logs"),
		"log-level":              "info",
		"chain-config-dir":       genesisDir,
		"staking-tls-cert-file":  filepath.Join(stakingDir, fmt.Sprintf("node%02d", nodeNum), "staker.crt"),
		"staking-tls-key-file":   filepath.Join(stakingDir, fmt.Sprintf("node%02d", nodeNum), "staker.key"),
		"bootstrap-ips":          strings.Join(bootstrapIPs, ","),
		"bootstrap-ids":          strings.Join(bootstrapIDs, ","),
	}
	
	data, _ := json.MarshalIndent(config, "", "  ")
	return os.WriteFile(filepath.Join(nodeDir, "config.json"), data, 0644)
}

func launchNode(dataDir string, nodeNum int, detached bool) (int, error) {
	nodeDir := filepath.Join(dataDir, fmt.Sprintf("node%02d", nodeNum))
	configFile := filepath.Join(nodeDir, "config.json")
	logFile := filepath.Join(nodeDir, "logs", "main.log")
	
	// Find luxd binary
	luxdPath := filepath.Join(os.Getenv("GOPATH"), "bin", "luxd")
	if !fileExists(luxdPath) {
		// Try local build
		luxdPath = filepath.Join("..", "..", "node", "build", "luxd")
		if !fileExists(luxdPath) {
			return 0, fmt.Errorf("luxd binary not found")
		}
	}
	
	cmd := exec.Command(luxdPath, "--config-file="+configFile)
	
	if detached {
		logFileHandle, _ := os.Create(logFile)
		cmd.Stdout = logFileHandle
		cmd.Stderr = logFileHandle
		
		if err := cmd.Start(); err != nil {
			return 0, err
		}
		return cmd.Process.Pid, nil
	} else {
		// Run in foreground
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return 0, cmd.Run()
	}
}

func isNodeHealthy(port int) bool {
	cmd := exec.Command("curl", "-s", "-X", "POST",
		"--data", `{"jsonrpc":"2.0","method":"health.health","params":{},"id":1}`,
		"-H", "content-type:application/json;",
		fmt.Sprintf("http://127.0.0.1:%d/ext/health", port))
	
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	
	return strings.Contains(string(output), `"healthy":true`)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}