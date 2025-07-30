package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func addLaunchSubcommands(parentCmd *cobra.Command) {

	// Launch L1 command (load chaindata into C-Chain)
	launchL1Cmd := &cobra.Command{
		Use:   "L1",
		Short: "Load chaindata into C-Chain",
		Long: `Launch luxd with L2 data loaded into the C-Chain.
		
This command:
1. Uses pre-migrated L2 data in C-Chain format
2. Sets up proper configuration for C-Chain
3. Launches luxd with network ID 96369
4. Monitors startup and reports RPC availability`,
		Args: cobra.NoArgs,
		RunE: runLaunchL1,
	}

	// Launch L2 command (load as L2 with lux primary network running)
	launchL2Cmd := &cobra.Command{
		Use:   "L2 [network-id]",
		Short: "Load as L2 with lux primary network running",
		Long: `Launch luxd with L2 subnet configuration.
		
This command:
1. Configures the L2 subnet
2. Launches with primary network active
3. Enables L2-specific features
4. Monitors startup and reports RPC availability`,
		Args: cobra.ExactArgs(1),
		RunE: runLaunchL2,
	}

	// Legacy cchain command for backward compatibility
	launchCChainCmd := &cobra.Command{
		Use:    "cchain [network-id]",
		Short:  "Launch luxd with subnet data imported as C-Chain",
		Hidden: true, // Hide from help but still available
		Args:   cobra.ExactArgs(1),
		RunE:   runLaunchCChain,
	}

	// Verify chain command
	verifyChainCmd := &cobra.Command{
		Use:   "verify [rpc-url]",
		Short: "Verify chain is running correctly",
		Long: `Verify that a launched chain is running correctly by checking:
- RPC endpoint availability
- Block height
- Chain ID
- Treasury balance (if applicable)`,
		Args: cobra.MaximumNArgs(1),
		RunE: runVerifyChain,
	}

	parentCmd.AddCommand(launchL1Cmd)
	parentCmd.AddCommand(launchL2Cmd)
	parentCmd.AddCommand(launchCChainCmd)
	parentCmd.AddCommand(verifyChainCmd)
}

func runLaunchL1(cmd *cobra.Command, args []string) error {
	// L1 always uses network ID 96369 for C-Chain
	networkID := "96369"

	fmt.Println("🚀 Launching L1 (C-Chain) with network ID", networkID)

	// Check if we already have migrated data
	migratedDataPath := filepath.Join(Paths.RuntimeDir, "lux-96369-migrated")
	if _, err := os.Stat(migratedDataPath); err == nil {
		fmt.Println("✅ Found existing migrated data at:", migratedDataPath)
		return launchWithExistingData(networkID, migratedDataPath)
	}

	// Otherwise use the standard flow
	return launchWithNetworkID(networkID)
}

func runLaunchL2(cmd *cobra.Command, args []string) error {
	networkID := args[0]

	fmt.Printf("🚀 Launching L2 with network ID %s\n", networkID)

	// L2 launch would involve subnet-specific configuration
	// For now, this is a placeholder that uses similar logic
	return launchWithNetworkID(networkID)
}

func runLaunchCChain(cmd *cobra.Command, args []string) error {
	networkID := args[0]
	return launchWithNetworkID(networkID)
}

func launchWithNetworkID(networkID string) error {
	// First, import the subnet data to C-Chain format
	fmt.Println("🔄 Importing Subnet EVM data as C-Chain...")

	sourceData := filepath.Join(Paths.ChaindataDir, fmt.Sprintf("lux-mainnet-%s", networkID), "db", "pebbledb")
	outputDir := filepath.Join(Paths.RuntimeDir, fmt.Sprintf("lux-%s-cchain", networkID))

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Run the import subnet command
	importCmd := exec.Command(os.Args[0], "import", "subnet", sourceData, outputDir)
	importCmd.Stdout = os.Stdout
	importCmd.Stderr = os.Stderr

	if err := importCmd.Run(); err != nil {
		return fmt.Errorf("import failed: %v", err)
	}

	fmt.Println("✅ Import complete!")

	// Now set up for launching luxd
	chainDataDir := outputDir
	chainConfigDir := Paths.ConfigsDir

	// Verify chain data exists - check both possible locations
	evmPath := filepath.Join(chainDataDir, "db", "pebbledb")
	if _, err := os.Stat(evmPath); os.IsNotExist(err) {
		// Try with evm subdirectory
		evmPath = filepath.Join(chainDataDir, "db", "evm", "pebbledb")
		if _, err := os.Stat(evmPath); os.IsNotExist(err) {
			return fmt.Errorf("chain data not found at %s or %s",
				filepath.Join(chainDataDir, "db", "pebbledb"),
				filepath.Join(chainDataDir, "db", "evm", "pebbledb"))
		}
	}

	// Use configured luxd path
	luxdPath := Paths.LuxdPath

	// Verify it exists
	if _, err := os.Stat(luxdPath); err != nil {
		// Try to find it in PATH
		if _, err := exec.LookPath("luxd"); err != nil {
			return fmt.Errorf("luxd binary not found at %s or in PATH", luxdPath)
		}
		luxdPath = "luxd" // Use from PATH
	}

	fmt.Printf("Launching luxd for network %s...\n", networkID)
	fmt.Printf("Chain data: %s\n", chainDataDir)
	fmt.Printf("Chain config: %s\n", chainConfigDir)

	// Build luxd command
	luxdCmd := exec.Command(luxdPath,
		"--network-id="+networkID,
		"--chain-config-dir="+chainConfigDir,
		"--chain-data-dir="+chainDataDir,
		"--dev", // Enables single-node mode with no sybil protection
		"--http-host=0.0.0.0",
		"--log-level=info",
	)

	// Set up output
	luxdCmd.Stdout = os.Stdout
	luxdCmd.Stderr = os.Stderr

	// Start luxd
	if err := luxdCmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %v", err)
	}

	fmt.Printf("Luxd started with PID %d\n", luxdCmd.Process.Pid)
	fmt.Println("Waiting for RPC to become available...")

	// Wait for RPC to be available
	rpcURL := "http://localhost:9650/ext/bc/C/rpc"
	if err := waitForRPC(rpcURL, 60*time.Second); err != nil {
		luxdCmd.Process.Kill()
		return fmt.Errorf("RPC failed to start: %v", err)
	}

	fmt.Println("\n✅ Luxd is running!")
	fmt.Printf("RPC endpoint: %s\n", rpcURL)
	fmt.Printf("Process ID: %d\n", luxdCmd.Process.Pid)
	fmt.Println("\nTo verify the chain, run:")
	fmt.Printf("  ./bin/genesis launch verify %s\n", rpcURL)
	fmt.Println("\nTo stop luxd:")
	fmt.Printf("  kill %d\n", luxdCmd.Process.Pid)

	// Wait for process to exit (keeps it running)
	return luxdCmd.Wait()
}

func launchWithExistingData(networkID string, chainDataDir string) error {
	// Use configured paths
	chainConfigDir := Paths.ConfigsDir

	// Verify chain data exists
	evmPath := filepath.Join(chainDataDir, "db", "evm", "pebbledb")
	if _, err := os.Stat(evmPath); os.IsNotExist(err) {
		// Try without evm subdirectory
		evmPath = filepath.Join(chainDataDir, "db", "pebbledb")
		if _, err := os.Stat(evmPath); os.IsNotExist(err) {
			return fmt.Errorf("chain data not found at %s", chainDataDir)
		}
	}

	// Use configured luxd path
	luxdPath := Paths.LuxdPath

	// Verify it exists
	if _, err := os.Stat(luxdPath); err != nil {
		// Try to find it in PATH
		if _, err := exec.LookPath("luxd"); err != nil {
			return fmt.Errorf("luxd binary not found at %s or in PATH", luxdPath)
		}
		luxdPath = "luxd" // Use from PATH
	}

	fmt.Printf("Launching luxd for network %s...\n", networkID)
	fmt.Printf("Chain data: %s\n", chainDataDir)
	fmt.Printf("Chain config: %s\n", chainConfigDir)
	fmt.Printf("Luxd binary: %s\n", luxdPath)

	// Build luxd command
	luxdCmd := exec.Command(luxdPath,
		"--network-id="+networkID,
		"--chain-config-dir="+chainConfigDir,
		"--chain-data-dir="+chainDataDir,
		"--dev", // Enables single-node mode with no sybil protection
		"--http-host=0.0.0.0",
		"--log-level=info",
	)

	// Set up output
	luxdCmd.Stdout = os.Stdout
	luxdCmd.Stderr = os.Stderr

	// Start luxd
	if err := luxdCmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %v", err)
	}

	fmt.Printf("Luxd started with PID %d\n", luxdCmd.Process.Pid)
	fmt.Println("Waiting for RPC to become available...")

	// Wait for RPC to be available
	rpcURL := "http://localhost:9650/ext/bc/C/rpc"
	if err := waitForRPC(rpcURL, 60*time.Second); err != nil {
		luxdCmd.Process.Kill()
		return fmt.Errorf("RPC failed to start: %v", err)
	}

	fmt.Println("\n✅ Luxd is running!")
	fmt.Printf("RPC endpoint: %s\n", rpcURL)
	fmt.Printf("Process ID: %d\n", luxdCmd.Process.Pid)
	fmt.Println("\nTo verify the chain, run:")
	fmt.Printf("  ./bin/genesis launch verify %s\n", rpcURL)
	fmt.Println("\nTo stop luxd:")
	fmt.Printf("  kill %d\n", luxdCmd.Process.Pid)

	// Wait for process to exit (keeps it running)
	return luxdCmd.Wait()
}

func runVerifyChain(cmd *cobra.Command, args []string) error {
	rpcURL := "http://localhost:9650/ext/bc/C/rpc"
	if len(args) > 0 {
		rpcURL = args[0]
	}

	fmt.Printf("Verifying chain at %s...\n\n", rpcURL)

	// Check block number
	blockNum, err := getRPCBlockNumber(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to get block number: %v", err)
	}
	fmt.Printf("✅ Current block height: %d\n", blockNum)

	// Check chain ID
	chainID, err := getRPCChainID(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to get chain ID: %v", err)
	}
	fmt.Printf("✅ Chain ID: %d\n", chainID)

	// Check treasury balance
	treasury := "0x9011e888251ab053b7bd1cdb598db4f9ded94714"
	balance, err := getRPCBalance(rpcURL, treasury)
	if err != nil {
		fmt.Printf("⚠️  Could not get treasury balance: %v\n", err)
	} else {
		fmt.Printf("✅ Treasury balance: %s wei\n", balance)
		// Check if > 1.9T (1.9e18)
		threshold := "1900000000000000000"
		if strings.Compare(balance, threshold) > 0 {
			fmt.Println("   ✓ Balance is above 1.9T threshold")
		} else {
			fmt.Println("   ✗ Balance is below 1.9T threshold")
		}
	}

	fmt.Println("\n✅ Chain verification complete!")
	return nil
}

func waitForRPC(rpcURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(rpcURL)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 || resp.StatusCode == 405 {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
		fmt.Print(".")
	}
	return fmt.Errorf("timeout waiting for RPC")
}

func getRPCBlockNumber(rpcURL string) (uint64, error) {
	payload := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
	resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// Convert hex to uint64
	blockNum, err := strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return blockNum, nil
}

func getRPCChainID(rpcURL string) (uint64, error) {
	payload := `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`
	resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	// Convert hex to uint64
	chainID, err := strconv.ParseUint(strings.TrimPrefix(result.Result, "0x"), 16, 64)
	if err != nil {
		return 0, err
	}

	return chainID, nil
}

func getRPCBalance(rpcURL, address string) (string, error) {
	payload := fmt.Sprintf(`{"jsonrpc":"2.0","method":"eth_getBalance","params":["%s","latest"],"id":1}`, address)
	resp, err := http.Post(rpcURL, "application/json", strings.NewReader(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Convert hex to decimal string
	balance := new(big.Int)
	balance.SetString(strings.TrimPrefix(result.Result, "0x"), 16)

	return balance.String(), nil
}
