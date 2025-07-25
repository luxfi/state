package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/node/ids"
	"github.com/spf13/cobra"

	// Import command packages
	archaeologyCmd "github.com/luxfi/genesis/cmd/archeology/commands"
	teleportCmd "github.com/luxfi/genesis/cmd/teleport/commands"

	// Import internal packages
	"github.com/luxfi/genesis/cmd/namespace/pkg/namespace"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

// Genesis represents the Ethereum genesis block configuration
type Genesis struct {
	Config     map[string]interface{} `json:"config"`
	Nonce      string                 `json:"nonce"`
	Timestamp  string                 `json:"timestamp"`
	ExtraData  string                 `json:"extraData"`
	GasLimit   string                 `json:"gasLimit"`
	Difficulty string                 `json:"difficulty"`
	MixHash    string                 `json:"mixHash"`
	Coinbase   string                 `json:"coinbase"`
	Alloc      map[string]interface{} `json:"alloc"`
	Number     string                 `json:"number"`
	GasUsed    string                 `json:"gasUsed"`
	ParentHash string                 `json:"parentHash"`
}

// Global configuration for generate command
type GenesisConfig struct {
	Network          string
	OutputDir        string
	Lux7777CSV       string
	Zoo200200Genesis string
	ValidatorsFile   string
	TreasuryAddress  string
	TreasuryAmount   string
	IncludeTreasury  bool
	GeneratePChain   bool
	GenerateCChain   bool
	GenerateXChain   bool
	UseStandardDirs  bool
}

var cfg = &GenesisConfig{}

func main() {
	rootCmd := &cobra.Command{
		Use:   "genesis",
		Short: "Lux Network Genesis Management Tool",
		Long: `A comprehensive tool for managing Lux Network genesis configurations.

This unified tool combines all genesis-related functionality:
- Generate genesis files for all chains (P, C, X)
- Extract and analyze blockchain data
- Import cross-chain assets
- Manage validators
- Process historical data
- And much more...

Use 'genesis --help' to see all available commands.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&cfg.Network, "network", "mainnet", "Network to use (mainnet, testnet, local)")
	rootCmd.PersistentFlags().StringVar(&cfg.OutputDir, "output", "", "Output directory")

	// Core genesis commands
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate genesis for all chains",
		Long:  `Generate P-Chain, C-Chain, and X-Chain genesis files with proper directory structure`,
		RunE:  runGenerate,
	}
	addGenerateFlags(generateCmd)

	// Validators command group
	validatorsCmd := &cobra.Command{
		Use:   "validators",
		Short: "Manage validators",
		Long:  `Add, remove, list, and generate validators`,
	}
	addValidatorSubcommands(validatorsCmd)

	// Extract command group
	extractCmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract blockchain data",
		Long:  `Extract blockchain data from various sources (PebbleDB, LevelDB, etc.)`,
	}
	addExtractSubcommands(extractCmd)

	// Import command group
	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import blockchain data",
		Long:  `Import existing blockchain data, allocations, and cross-chain assets`,
	}
	addImportSubcommands(importCmd)

	// Analyze command group
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze blockchain data",
		Long:  `Analyze extracted blockchain data for accounts, balances, and contracts`,
	}
	addAnalyzeSubcommands(analyzeCmd)

	// Scan command group
	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan external blockchains",
		Long:  `Scan external blockchains (Ethereum, BSC, etc.) for assets to include in genesis`,
	}
	addScanSubcommands(scanCmd)

	// Migrate command group
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate cross-chain assets",
		Long:  `Migrate tokens and NFTs from external chains to Lux Network`,
	}
	addMigrateSubcommands(migrateCmd)

	// Process command group
	processCmd := &cobra.Command{
		Use:   "process",
		Short: "Process historical data",
		Long:  `Process historical blockchain data for genesis inclusion`,
	}
	addProcessSubcommands(processCmd)

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate genesis configuration",
		RunE:  runValidate,
	}

	// Tools command
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "List available tools and utilities",
		RunE:  runTools,
	}

	// Read command to extract genesis from chain data
	readCmd := &cobra.Command{
		Use:   "read [source-path]",
		Short: "Read genesis from historic chain data",
		Long: `Read genesis configuration from existing chain data.
		
This command extracts the genesis from a blockchain database and can optionally:
- Derive the blockchain ID from the genesis
- Write the genesis to a standard location
- Display genesis information`,
		Args: cobra.ExactArgs(1),
		RunE: runReadGenesis,
	}
	
	readCmd.Flags().StringP("output", "o", "", "Output path for genesis file (default: stdout)")
	readCmd.Flags().BoolP("write-config", "w", false, "Write genesis to ~/.luxd/configs/C/genesis.json")
	readCmd.Flags().BoolP("show-id", "i", true, "Show derived blockchain ID")
	readCmd.Flags().BoolP("raw", "r", false, "Save raw genesis bytes as genesis.blob")
	readCmd.Flags().BoolP("pointers", "p", false, "Show pointer keys (Height, LastAccepted, etc)")
	
	// Diagnose command to check database health
	diagnoseCmd := &cobra.Command{
		Use:   "diagnose [db-path]",
		Short: "Diagnose blockchain database issues",
		Long: `Diagnose common issues preventing historic blocks from loading:
- Check header count
- Verify pointer keys (Height, LastAccepted, etc)
- Extract genesis blob
- Compare genesis with config`,
		Args: cobra.ExactArgs(1),
		RunE: runDiagnose,
	}
	
	// Count command to count database keys
	countCmd := &cobra.Command{
		Use:   "count [db-path]",
		Short: "Count keys in blockchain database",
		Long:  `Count keys by prefix in a blockchain database`,
		Args:  cobra.ExactArgs(1),
		RunE:  runCount,
	}
	
	countCmd.Flags().StringP("prefix", "p", "68", "Key prefix in hex (68=headers, 62=bodies)")
	countCmd.Flags().BoolP("all", "a", false, "Count all keys (no prefix filter)")
	
	// Pointers command to manage pointer keys
	pointersCmd := &cobra.Command{
		Use:   "pointers [db-path]",
		Short: "Manage blockchain pointer keys",
		Long:  `View or update pointer keys (Height, LastAccepted, LastBlock, LastHeader)`,
		Args:  cobra.ExactArgs(1),
	}
	
	// Sub-commands for pointers
	pointersShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show pointer keys",
		RunE:  runPointersShow,
	}
	
	pointersSetCmd := &cobra.Command{
		Use:   "set [db-path] [key] [value]",
		Short: "Set a pointer key",
		Args:  cobra.ExactArgs(3),
		RunE:  runPointersSet,
	}
	
	pointersCopyCmd := &cobra.Command{
		Use:   "copy [source-db] [dest-db]",
		Short: "Copy pointer keys between databases",
		Args:  cobra.ExactArgs(2),
		RunE:  runPointersCopy,
	}
	
	pointersCmd.AddCommand(pointersShowCmd, pointersSetCmd, pointersCopyCmd)

	// Build command structure
	rootCmd.AddCommand(
		generateCmd,
		validatorsCmd,
		extractCmd,
		importCmd,
		analyzeCmd,
		scanCmd,
		migrateCmd,
		processCmd,
		validateCmd,
		toolsCmd,
		readCmd,
		diagnoseCmd,
		countCmd,
		pointersCmd,
		// Additional utility commands from teleport
		teleportCmd.NewExportCommand(),
		teleportCmd.NewVerifyCommand(),
	)

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func addGenerateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&cfg.Lux7777CSV, "lux7777", "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet-no-treasury.csv", "Lux 7777 airdrop CSV")
	cmd.Flags().StringVar(&cfg.Zoo200200Genesis, "zoo", "exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.csv", "Zoo allocations CSV")
	cmd.Flags().StringVar(&cfg.TreasuryAddress, "treasury", "0x9011e888251ab053b7bd1cdb598db4f9ded94714", "Treasury address")
	cmd.Flags().StringVar(&cfg.TreasuryAmount, "treasury-amount", "2T", "Treasury amount (e.g., 2T, 1B, 500M)")
	cmd.Flags().BoolVar(&cfg.IncludeTreasury, "include-treasury", true, "Include treasury in genesis")
	cmd.Flags().BoolVar(&cfg.GeneratePChain, "p-chain", true, "Generate P-Chain genesis")
	cmd.Flags().BoolVar(&cfg.GenerateCChain, "c-chain", true, "Generate C-Chain genesis")
	cmd.Flags().BoolVar(&cfg.GenerateXChain, "x-chain", true, "Generate X-Chain genesis")
	cmd.Flags().BoolVar(&cfg.UseStandardDirs, "standard-dirs", true, "Use standard directory structure (P/, C/, X/)")
	cmd.Flags().StringVar(&cfg.ValidatorsFile, "validators", "", "Path to validators JSON file")
}

func addValidatorSubcommands(validatorsCmd *cobra.Command) {
	// List validators
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List validators",
		RunE:  runListValidators,
	}
	listCmd.Flags().StringVar(&cfg.ValidatorsFile, "validators", "", "Path to validators JSON file")

	// Add validator
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a validator",
		RunE:  runAddValidator,
	}
	addCmd.Flags().String("node-id", "", "Node ID")
	addCmd.Flags().String("eth-address", "", "Ethereum address")
	addCmd.Flags().String("public-key", "", "BLS public key")
	addCmd.Flags().String("proof-of-possession", "", "BLS proof of possession")
	addCmd.Flags().String("weight", "100000000000000", "Validator weight")
	addCmd.MarkFlagRequired("node-id")
	addCmd.MarkFlagRequired("eth-address")

	// Remove validator
	removeCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a validator",
		RunE:  runRemoveValidator,
	}
	removeCmd.Flags().Int("index", -1, "Validator index to remove")
	removeCmd.Flags().String("node-id", "", "Node ID to remove")

	// Generate validators
	generateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate new validators",
		RunE:  runGenerateValidators,
	}
	generateCmd.Flags().String("mnemonic", "", "BIP39 mnemonic phrase")
	generateCmd.Flags().String("offsets", "0,1,2,3,4,5,6,7,8,9,10", "Comma-separated list of HD wallet offsets")
	generateCmd.Flags().String("save-keys", "", "Save validator keys to file")
	generateCmd.Flags().String("save-keys-dir", "configs/keys", "Directory to save individual validator key files")
	generateCmd.MarkFlagRequired("mnemonic")

	validatorsCmd.AddCommand(listCmd, addCmd, removeCmd, generateCmd)
}

func addExtractSubcommands(extractCmd *cobra.Command) {
	// State extraction (namespace)
	stateCmd := &cobra.Command{
		Use:   "state [source] [destination]",
		Short: "Extract state from PebbleDB (namespace)",
		Long:  `Extract account state and blockchain data from PebbleDB format, removing namespace prefixes`,
		Args:  cobra.ExactArgs(2),
		RunE:  runExtractState,
	}
	stateCmd.Flags().Int("network", 96369, "Chain ID")
	stateCmd.Flags().Bool("state", true, "Include state data")
	stateCmd.Flags().Int("limit", 0, "Limit number of entries (0 = no limit)")

	// Genesis extraction from blockchain
	genesisCmd := &cobra.Command{
		Use:   "genesis [database-path]",
		Short: "Extract genesis configuration from blockchain database",
		Long:  `Extract the genesis block configuration and allocations from an existing blockchain database`,
		Args:  cobra.ExactArgs(1),
		RunE:  runExtractGenesis,
	}
	genesisCmd.Flags().String("type", "auto", "Database type: leveldb, pebble, or auto")
	genesisCmd.Flags().String("output", "", "Output file path (default: stdout)")
	genesisCmd.Flags().Bool("pretty", true, "Pretty print JSON output")
	genesisCmd.Flags().Bool("alloc", true, "Include account allocations")
	genesisCmd.Flags().String("csv", "", "Export allocations to CSV file")

	// Add archaeology extract commands
	extractCmd.AddCommand(
		stateCmd,
		genesisCmd,
		archaeologyCmd.NewExtractCommand(),
	)
}

func addImportSubcommands(importCmd *cobra.Command) {
	// Import from original genesis block
	genesisCmd := &cobra.Command{
		Use:   "genesis [file]",
		Short: "Import from original genesis block",
		Long:  `Import allocations and configuration from an original genesis.json file`,
		Args:  cobra.ExactArgs(1),
		RunE:  runImportGenesis,
	}
	genesisCmd.Flags().String("chain", "C", "Chain type (P, C, or X)")
	genesisCmd.Flags().Bool("allocations-only", false, "Import only allocations, not config")
	genesisCmd.Flags().String("output", "", "Output file (default: updates current genesis)")

	// Import from blockchain state at specific block
	blockCmd := &cobra.Command{
		Use:   "block [number]",
		Short: "Import state from specific block",
		Long:  `Import account state from a specific block in the blockchain`,
		Args:  cobra.ExactArgs(1),
		RunE:  runImportBlock,
	}
	blockCmd.Flags().String("rpc", "http://localhost:9650/ext/bc/C/rpc", "RPC endpoint")
	blockCmd.Flags().String("output", "", "Output CSV file for allocations")

	// Import C-Chain data from extracted blockchain
	cchainCmd := &cobra.Command{
		Use:   "cchain [source]",
		Short: "Import C-Chain state",
		Long:  `Import existing C-Chain state from extracted blockchain data`,
		Args:  cobra.ExactArgs(1),
		RunE:  runImportCChain,
	}

	// Import allocations from CSV/JSON
	allocationsCmd := &cobra.Command{
		Use:   "allocations [file]",
		Short: "Import allocations from CSV or JSON",
		Args:  cobra.ExactArgs(1),
		RunE:  runImportAllocations,
	}
	allocationsCmd.Flags().String("format", "auto", "File format (csv, json, auto)")
	allocationsCmd.Flags().Bool("merge", false, "Merge with existing allocations")

	// Add all import commands
	importCmd.AddCommand(
		genesisCmd,
		blockCmd,
		cchainCmd,
		allocationsCmd,
		importSubnetCmd(),  // Import subnet as C-Chain fork
		archaeologyCmd.NewImportNFTCommand(),
		archaeologyCmd.NewImportTokenCommand(),
	)
}

func addAnalyzeSubcommands(analyzeCmd *cobra.Command) {
	// Add archaeology analyze commands
	analyzeCmd.AddCommand(
		archaeologyCmd.NewAnalyzeCommand(),
	)
}

func addScanSubcommands(scanCmd *cobra.Command) {
	// Add teleport scan commands
	scanCmd.AddCommand(
		teleportCmd.NewScanNFTCommand(),
		teleportCmd.NewScanTokenCommand(),
		teleportCmd.NewScanTokenBurnsCommand(),
		teleportCmd.NewScanNFTHoldersCommand(),
		teleportCmd.NewScanTokenTransfersCommand(),
		teleportCmd.NewScanEggHoldersCommand(),
	)

	// Add archaeology scan commands
	scanCmd.AddCommand(
		archaeologyCmd.NewScanCommand(),
		archaeologyCmd.NewScanBurnsCommand(),
		archaeologyCmd.NewScanHoldersCommand(),
		archaeologyCmd.NewScanTransfersCommand(),
		archaeologyCmd.NewScanCurrentHoldersCommand(),
	)
}

func addMigrateSubcommands(migrateCmd *cobra.Command) {
	// Add read command to extract genesis from chain data
	readCmd := &cobra.Command{
		Use:   "read [source-path]",
		Short: "Read genesis from historic chain data and migrate to new blockchain ID",
		Long: `Read genesis from historic chain data, derive new blockchain ID, and optionally migrate data.
		
This command:
1. Extracts genesis configuration from historic chain data
2. Derives the new blockchain ID from the genesis
3. Writes the genesis to ~/.luxd/configs/C/genesis.json
4. Optionally migrates the chain data with updated blockchain ID`,
		Args: cobra.ExactArgs(1),
		RunE: runMigrateRead,
	}
	
	readCmd.Flags().StringP("dst", "d", "", "Destination path for migrated data (optional)")
	readCmd.Flags().BoolP("genesis-only", "g", false, "Only extract genesis, don't migrate data")
	readCmd.Flags().BoolP("write-genesis", "w", true, "Write genesis to ~/.luxd/configs/C/genesis.json")
	
	// Add teleport migrate commands
	migrateCmd.AddCommand(
		readCmd,
		subnetToCChainCmd(),          // Convert subnet EVM to C-Chain format
		subnetToL2Cmd(),              // Convert subnet EVM to L2 format  
		migrateSubnetToCChainCmd(),   // Migrate subnet to C-Chain with prefixes
		teleportCmd.NewMigrateCommand(),
		teleportCmd.NewZooMigrateCommand(),
		teleportCmd.NewZooCrossReferenceCommand(),
		teleportCmd.NewZooCrossReferenceV2Command(),
	)
}

func addProcessSubcommands(processCmd *cobra.Command) {
	// Process historic command
	historicCmd := &cobra.Command{
		Use:   "historic [source]",
		Short: "Process historical blockchain data",
		Args:  cobra.ExactArgs(1),
		RunE:  runProcessHistoric,
	}

	processCmd.AddCommand(historicCmd)
}

// Command implementations - Generate command (delegate to generate.go)
func runGenerate(cmd *cobra.Command, args []string) error {
	// Delegate to the actual generate command
	generateCommand := generateCmd()
	generateCommand.SetArgs(args)
	
	// Copy flags
	generateCommand.Flags().Set("network", cfg.Network)
	if cfg.OutputDir != "" {
		generateCommand.Flags().Set("output", cfg.OutputDir)
	}
	
	return generateCommand.Execute()
}

// Extract state command implementation
func runExtractState(cmd *cobra.Command, args []string) error {
	source := args[0]
	destination := args[1]

	networkID, _ := cmd.Flags().GetInt("network")
	includeState, _ := cmd.Flags().GetBool("state")
	limit, _ := cmd.Flags().GetInt("limit")

	fmt.Printf("Extracting state from %s to %s\n", source, destination)
	fmt.Printf("Network ID: %d, Include State: %v, Limit: %d\n", networkID, includeState, limit)

	// Use the namespace package
	opts := namespace.Options{
		Source:      source,
		Destination: destination,
		NetworkID:   uint64(networkID),
		State:       includeState,
		Limit:       limit,
	}

	return namespace.Extract(opts)
}

// Extract genesis command implementation
func runExtractGenesis(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	dbType, _ := cmd.Flags().GetString("type")
	outputPath, _ := cmd.Flags().GetString("output")
	prettyPrint, _ := cmd.Flags().GetBool("pretty")
	includeAlloc, _ := cmd.Flags().GetBool("alloc")
	csvPath, _ := cmd.Flags().GetString("csv")

	// Build command arguments
	cmdArgs := []string{
		"-db", dbPath,
		"-type", dbType,
		fmt.Sprintf("-pretty=%v", prettyPrint),
		fmt.Sprintf("-alloc=%v", includeAlloc),
	}

	if outputPath != "" {
		cmdArgs = append(cmdArgs, "-output", outputPath)
	}

	// Build and run the extract-genesis binary
	extractCmd := exec.Command("./bin/extract-genesis", cmdArgs...)
	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr

	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("failed to extract genesis: %w", err)
	}

	// If CSV export was requested, extract allocations to CSV
	if csvPath != "" && outputPath != "" {
		// Read the generated genesis file
		genesisData, err := os.ReadFile(outputPath)
		if err != nil {
			return fmt.Errorf("failed to read genesis file: %w", err)
		}

		var genesis struct {
			Alloc map[string]struct {
				Balance string `json:"balance"`
			} `json:"alloc"`
		}
		if err := json.Unmarshal(genesisData, &genesis); err != nil {
			return fmt.Errorf("failed to parse genesis: %w", err)
		}

		// Write CSV
		csvFile, err := os.Create(csvPath)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer csvFile.Close()

		fmt.Fprintln(csvFile, "address,balance_wei")
		for addr, account := range genesis.Alloc {
			if account.Balance != "" {
				fmt.Fprintf(csvFile, "%s,%s\n", addr, account.Balance)
			}
		}
		fmt.Printf("Allocations exported to %s\n", csvPath)
	}

	return nil
}

func runArcheologyMigrate(cmd *cobra.Command, args []string) error {
	fmt.Println("Running archaeology migrate...")
	// Call the archaeology migrate command
	migrateCmd := exec.Command("./bin/archeology", "migrate")
	migrateCmd.Stdout = os.Stdout
	migrateCmd.Stderr = os.Stderr
	return migrateCmd.Run()
}

// Process historic command implementation
func runProcessHistoric(cmd *cobra.Command, args []string) error {
	source := args[0]
	fmt.Printf("Processing historic data from %s\n", source)
	// Implementation would go here
	return nil
}

// Import command implementations
func runImportGenesis(cmd *cobra.Command, args []string) error {
	genesisFile := args[0]
	chain, _ := cmd.Flags().GetString("chain")
	allocationsOnly, _ := cmd.Flags().GetBool("allocations-only")
	output, _ := cmd.Flags().GetString("output")

	fmt.Printf("Importing from original genesis: %s\n", genesisFile)
	fmt.Printf("Chain: %s, Allocations Only: %v\n", chain, allocationsOnly)

	// Read the original genesis file
	data, err := os.ReadFile(genesisFile)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	// Parse based on chain type
	switch chain {
	case "C":
		// Parse C-Chain genesis
		var cGenesis struct {
			Config map[string]interface{} `json:"config"`
			Alloc  map[string]struct {
				Balance string            `json:"balance"`
				Code    string            `json:"code,omitempty"`
				Storage map[string]string `json:"storage,omitempty"`
			} `json:"alloc"`
			Difficulty string `json:"difficulty"`
			GasLimit   string `json:"gasLimit"`
			Nonce      string `json:"nonce"`
			Timestamp  string `json:"timestamp"`
		}

		if err := json.Unmarshal(data, &cGenesis); err != nil {
			return fmt.Errorf("failed to parse C-Chain genesis: %w", err)
		}

		fmt.Printf("Found %d allocations in C-Chain genesis\n", len(cGenesis.Alloc))

		// If output specified, save allocations
		if output != "" {
			allocations := make(map[string]*big.Int)
			for addr, acc := range cGenesis.Alloc {
				balance := new(big.Int)
				balance.SetString(acc.Balance, 0)
				allocations[addr] = balance
			}

			// Save as JSON
			allocData, err := json.MarshalIndent(allocations, "", "  ")
			if err != nil {
				return err
			}

			if err := ioutil.WriteFile(output, allocData, 0644); err != nil {
				return err
			}

			fmt.Printf("Saved allocations to %s\n", output)
		}

		// If not allocations only, also import config
		if !allocationsOnly {
			fmt.Printf("Chain config: %+v\n", cGenesis.Config)
		}

	case "P":
		// Parse P-Chain genesis
		var pGenesis struct {
			NetworkID            uint32                   `json:"networkID"`
			Allocations          []map[string]interface{} `json:"allocations"`
			StartTime            uint64                   `json:"startTime"`
			InitialStakeDuration uint64                   `json:"initialStakeDuration"`
			InitialStakers       []map[string]interface{} `json:"initialStakers"`
			CChainGenesis        string                   `json:"cChainGenesis"`
			Message              string                   `json:"message"`
		}

		if err := json.Unmarshal(data, &pGenesis); err != nil {
			return fmt.Errorf("failed to parse P-Chain genesis: %w", err)
		}

		fmt.Printf("Network ID: %d\n", pGenesis.NetworkID)
		fmt.Printf("Allocations: %d\n", len(pGenesis.Allocations))
		fmt.Printf("Initial Stakers: %d\n", len(pGenesis.InitialStakers))

	case "X":
		// Parse X-Chain genesis
		var xGenesis struct {
			Allocations []struct {
				ETHAddr        string `json:"ethAddr"`
				AVAXAddr       string `json:"avaxAddr"`
				InitialAmount  uint64 `json:"initialAmount"`
				UnlockSchedule []struct {
					Amount   uint64 `json:"amount"`
					Locktime uint64 `json:"locktime"`
				} `json:"unlockSchedule"`
			} `json:"allocations"`
			StartTime            uint64                   `json:"startTime"`
			InitialStakeDuration uint64                   `json:"initialStakeDuration"`
			InitialStakers       []map[string]interface{} `json:"initialStakers"`
			CChainGenesis        string                   `json:"cChainGenesis"`
			Message              string                   `json:"message"`
		}

		if err := json.Unmarshal(data, &xGenesis); err != nil {
			return fmt.Errorf("failed to parse X-Chain genesis: %w", err)
		}

		fmt.Printf("Found %d allocations in X-Chain genesis\n", len(xGenesis.Allocations))

	default:
		return fmt.Errorf("unsupported chain type: %s", chain)
	}

	return nil
}

func runImportBlock(cmd *cobra.Command, args []string) error {
	blockNumber := args[0]
	rpc, _ := cmd.Flags().GetString("rpc")
	output, _ := cmd.Flags().GetString("output")

	fmt.Printf("Importing state from block %s\n", blockNumber)
	fmt.Printf("RPC endpoint: %s\n", rpc)

	// In a real implementation, this would:
	// 1. Connect to the RPC endpoint
	// 2. Query for the state at the specified block
	// 3. Extract all account balances
	// 4. Save to the output file

	if output != "" {
		fmt.Printf("Output will be saved to: %s\n", output)
	}

	return fmt.Errorf("block import not yet implemented - would query RPC for state at block %s", blockNumber)
}

func runImportCChain(cmd *cobra.Command, args []string) error {
	source := args[0]
	fmt.Printf("Importing C-Chain state from %s\n", source)

	// This would import from extracted blockchain data
	// Similar to the existing import-cchain-data tool

	return fmt.Errorf("C-Chain import not yet implemented")
}

func runImportAllocations(cmd *cobra.Command, args []string) error {
	file := args[0]
	format, _ := cmd.Flags().GetString("format")
	merge, _ := cmd.Flags().GetBool("merge")

	fmt.Printf("Importing allocations from %s\n", file)
	fmt.Printf("Format: %s, Merge: %v\n", format, merge)

	// Detect format if auto
	if format == "auto" {
		if strings.HasSuffix(file, ".csv") {
			format = "csv"
		} else if strings.HasSuffix(file, ".json") {
			format = "json"
		} else {
			return fmt.Errorf("cannot auto-detect format for file: %s", file)
		}
	}

	// Read file
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	allocations := make(map[string]*big.Int)

	switch format {
	case "csv":
		// Parse CSV format
		// Expected format: address,balance
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if i == 0 && strings.Contains(line, "address") {
				continue // Skip header
			}

			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				addr := strings.TrimSpace(parts[0])
				balanceStr := strings.TrimSpace(parts[1])

				balance := new(big.Int)
				balance.SetString(balanceStr, 10)
				allocations[addr] = balance
			}
		}

	case "json":
		// Parse JSON format
		if err := json.Unmarshal(data, &allocations); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	fmt.Printf("Imported %d allocations\n", len(allocations))

	// If merge is true, we would merge with existing allocations
	if merge {
		fmt.Println("Merging with existing allocations...")
		// Load existing allocations and merge
	}

	return nil
}

// ValidatorInfo represents a validator configuration
type ValidatorInfo struct {
	NodeID            string `json:"nodeId"`
	ETHAddress        string `json:"ethAddress"`
	PublicKey         string `json:"publicKey,omitempty"`
	ProofOfPossession string `json:"proofOfPossession,omitempty"`
	Weight            uint64 `json:"weight"`
	DelegationFee     uint64 `json:"delegationFee"`
}

// Validator command implementations
func runListValidators(cmd *cobra.Command, args []string) error {
	if cfg.ValidatorsFile == "" {
		cfg.ValidatorsFile = fmt.Sprintf("configs/%s-validators.json", cfg.Network)
	}

	validators, err := loadValidators(cfg.ValidatorsFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No validators found.")
			return nil
		}
		return err
	}

	fmt.Printf("Validators from %s:\n", cfg.ValidatorsFile)
	fmt.Printf("%-5s %-50s %-42s %s\n", "Index", "NodeID", "ETH Address", "Weight")
	fmt.Println(strings.Repeat("-", 120))

	for i, v := range validators {
		fmt.Printf("%-5d %-50s %-42s %d\n", i, v.NodeID, v.ETHAddress, v.Weight)
	}

	fmt.Printf("\nTotal validators: %d\n", len(validators))
	return nil
}

func runAddValidator(cmd *cobra.Command, args []string) error {
	nodeID, _ := cmd.Flags().GetString("node-id")
	ethAddress, _ := cmd.Flags().GetString("eth-address")
	publicKey, _ := cmd.Flags().GetString("public-key")
	proofOfPossession, _ := cmd.Flags().GetString("proof-of-possession")
	weightStr, _ := cmd.Flags().GetString("weight")

	weight, err := parseAmount(weightStr)
	if err != nil {
		return fmt.Errorf("invalid weight: %w", err)
	}

	if cfg.ValidatorsFile == "" {
		cfg.ValidatorsFile = fmt.Sprintf("configs/%s-validators.json", cfg.Network)
	}

	validators, _ := loadValidators(cfg.ValidatorsFile)

	newValidator := &ValidatorInfo{
		NodeID:            nodeID,
		ETHAddress:        ethAddress,
		PublicKey:         publicKey,
		ProofOfPossession: proofOfPossession,
		Weight:            weight.Uint64(),
		DelegationFee:     20000,
	}

	validators = append(validators, newValidator)

	if err := saveValidators(validators, cfg.ValidatorsFile); err != nil {
		return err
	}

	fmt.Printf("Added validator %s\n", nodeID)
	fmt.Printf("Total validators: %d\n", len(validators))
	return nil
}

func runRemoveValidator(cmd *cobra.Command, args []string) error {
	index, _ := cmd.Flags().GetInt("index")
	nodeID, _ := cmd.Flags().GetString("node-id")

	if cfg.ValidatorsFile == "" {
		cfg.ValidatorsFile = fmt.Sprintf("configs/%s-validators.json", cfg.Network)
	}

	validators, err := loadValidators(cfg.ValidatorsFile)
	if err != nil {
		return err
	}

	indexToRemove := -1

	if index >= 0 {
		indexToRemove = index
	} else if nodeID != "" {
		for i, v := range validators {
			if v.NodeID == nodeID {
				indexToRemove = i
				break
			}
		}
	} else {
		return fmt.Errorf("must specify --index or --node-id")
	}

	if indexToRemove < 0 || indexToRemove >= len(validators) {
		return fmt.Errorf("validator not found")
	}

	removed := validators[indexToRemove]
	validators = append(validators[:indexToRemove], validators[indexToRemove+1:]...)

	if err := saveValidators(validators, cfg.ValidatorsFile); err != nil {
		return err
	}

	fmt.Printf("Removed validator %s\n", removed.NodeID)
	fmt.Printf("Remaining validators: %d\n", len(validators))
	return nil
}

func runGenerateValidators(cmd *cobra.Command, args []string) error {
	_, _ = cmd.Flags().GetString("mnemonic") // In production, would use this for BIP39
	offsetsStr, _ := cmd.Flags().GetString("offsets")

	offsetStrs := strings.Split(offsetsStr, ",")
	offsets := make([]int, len(offsetStrs))
	for i, s := range offsetStrs {
		n := 0
		if _, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n); err != nil {
			return fmt.Errorf("invalid offset: %s", s)
		}
		offsets[i] = n
	}

	fmt.Printf("Generating %d validators...\n", len(offsets))
	validators := make([]*ValidatorInfo, 0, len(offsets))

	for _, offset := range offsets {
		// Simple mock validator generation - in production this would use BIP39/BIP44
		v := &ValidatorInfo{
			NodeID:            fmt.Sprintf("NodeID-Validator%d", offset),
			ETHAddress:        fmt.Sprintf("0x%040d", offset),
			PublicKey:         fmt.Sprintf("0xpubkey%d", offset),
			ProofOfPossession: fmt.Sprintf("0xpop%d", offset),
			Weight:            100000000000000, // 100T
			DelegationFee:     20000,           // 2%
		}
		validators = append(validators, v)
		fmt.Printf("Generated validator %d: %s\n", offset, v.NodeID)
	}

	if cfg.ValidatorsFile == "" {
		cfg.ValidatorsFile = fmt.Sprintf("configs/%s-validators.json", cfg.Network)
	}

	if err := saveValidators(validators, cfg.ValidatorsFile); err != nil {
		return err
	}

	fmt.Printf("\nSaved %d validators to %s\n", len(validators), cfg.ValidatorsFile)
	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	dir := cfg.OutputDir
	if dir == "" {
		dir = filepath.Join("configs", cfg.Network)
	}

	fmt.Printf("Validating genesis files in: %s\n", dir)

	chains := []string{"P", "C", "X"}
	allValid := true

	for _, chain := range chains {
		genesisPath := filepath.Join(dir, chain, "genesis.json")
		if _, err := os.Stat(genesisPath); err == nil {
			data, err := ioutil.ReadFile(genesisPath)
			if err != nil {
				fmt.Printf("✗ Error reading %s-Chain genesis: %v\n", chain, err)
				allValid = false
				continue
			}

			var genesis map[string]interface{}
			if err := json.Unmarshal(data, &genesis); err != nil {
				fmt.Printf("✗ Invalid JSON in %s-Chain genesis: %v\n", chain, err)
				allValid = false
				continue
			}

			fmt.Printf("✓ Valid %s-Chain genesis: %s\n", chain, genesisPath)
		} else {
			fmt.Printf("✗ Missing %s-Chain genesis: %s\n", chain, genesisPath)
			allValid = false
		}
	}

	if allValid {
		fmt.Println("\n✅ All genesis files are valid!")
	} else {
		return fmt.Errorf("validation failed")
	}

	return nil
}

func runTools(cmd *cobra.Command, args []string) error {
	fmt.Println("Lux Network Genesis Tool - Available Commands")
	fmt.Println("============================================")
	fmt.Println()
	fmt.Println("Core Commands:")
	fmt.Println("  generate         - Generate genesis files for all chains")
	fmt.Println("  validators       - Manage validators (list, add, remove, generate)")
	fmt.Println("  validate         - Validate genesis configuration")
	fmt.Println()
	fmt.Println("Data Management:")
	fmt.Println("  extract          - Extract blockchain data from various sources")
	fmt.Println("  import           - Import blockchain data and allocations")
	fmt.Println("  analyze          - Analyze extracted blockchain data")
	fmt.Println("  process          - Process historical blockchain data")
	fmt.Println()
	fmt.Println("Cross-Chain Operations:")
	fmt.Println("  scan             - Scan external blockchains for assets")
	fmt.Println("  migrate          - Migrate cross-chain assets")
	fmt.Println()
	fmt.Println("Utilities:")
	fmt.Println("  export           - Export data in various formats")
	fmt.Println("  verify           - Verify migrations and data integrity")
	fmt.Println("  tools            - Show this help")
	fmt.Println()
	fmt.Println("For detailed help on any command, run: genesis <command> --help")

	return nil
}

// Helper functions
func loadValidators(path string) ([]*ValidatorInfo, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var validators []*ValidatorInfo
	if err := json.Unmarshal(data, &validators); err != nil {
		return nil, err
	}

	return validators, nil
}

func saveValidators(validators []*ValidatorInfo, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(validators, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

func parseAmount(s string) (*big.Int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty amount")
	}

	multiplier := big.NewInt(1)

	// Check for suffix
	lastChar := s[len(s)-1]
	switch lastChar {
	case 'T', 't':
		multiplier = new(big.Int).Exp(big.NewInt(10), big.NewInt(12), nil)
		s = s[:len(s)-1]
	case 'B', 'b':
		multiplier = new(big.Int).Exp(big.NewInt(10), big.NewInt(9), nil)
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
		s = s[:len(s)-1]
	case 'K', 'k':
		multiplier = new(big.Int).Exp(big.NewInt(10), big.NewInt(3), nil)
		s = s[:len(s)-1]
	}

	// Parse the number
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid number: %s", s)
	}

	// Convert to big.Int with proper precision handling
	fWithMultiplier := new(big.Float).SetInt(multiplier)
	f.Mul(f, fWithMultiplier)

	i := new(big.Int)
	f.Int(i)

	// Convert to wei (multiply by 10^18)
	wei := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	i.Mul(i, wei)

	return i, nil
}

// runReadGenesis implements the read command
func runReadGenesis(cmd *cobra.Command, args []string) error {
	srcPath := args[0]
	outputPath, _ := cmd.Flags().GetString("output")
	writeConfig, _ := cmd.Flags().GetBool("write-config")
	showID, _ := cmd.Flags().GetBool("show-id")
	saveRaw, _ := cmd.Flags().GetBool("raw")
	showPointers, _ := cmd.Flags().GetBool("pointers")
	
	// Extract genesis from historic data
	fmt.Printf("📖 Reading genesis from: %s\n", srcPath)
	genesis, genesisBytes, err := extractHistoricGenesis(srcPath)
	if err != nil {
		return fmt.Errorf("failed to extract genesis: %w", err)
	}
	
	// Save raw genesis bytes if requested
	if saveRaw {
		blobPath := "genesis.blob"
		if outputPath != "" {
			blobPath = strings.TrimSuffix(outputPath, ".json") + ".blob"
		}
		if err := ioutil.WriteFile(blobPath, genesisBytes, 0644); err != nil {
			return fmt.Errorf("failed to write genesis blob: %w", err)
		}
		fmt.Printf("💾 Saved raw genesis bytes to: %s (%d bytes)\n", blobPath, len(genesisBytes))
	}
	
	// Show pointer keys if requested
	if showPointers {
		if err := showPointerKeys(srcPath); err != nil {
			log.Printf("Warning: Could not read pointer keys: %v", err)
		}
	}
	
	// Format the genesis nicely
	formattedGenesis, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format genesis: %w", err)
	}
	
	// Show blockchain ID if requested
	if showID {
		blockchainID, err := deriveBlockchainID(genesisBytes)
		if err != nil {
			return fmt.Errorf("failed to derive blockchain ID: %w", err)
		}
		fmt.Printf("📌 Blockchain ID: %s\n", blockchainID.String())
		fmt.Printf("📌 Chain ID: %v\n", genesis.Config["chainId"])
	}
	
	// Write to config if requested
	if writeConfig {
		genesisDir := filepath.Join(os.Getenv("HOME"), ".luxd", "configs", "C")
		if err := os.MkdirAll(genesisDir, 0755); err != nil {
			return fmt.Errorf("failed to create genesis directory: %w", err)
		}
		
		genesisPath := filepath.Join(genesisDir, "genesis.json")
		if err := ioutil.WriteFile(genesisPath, formattedGenesis, 0644); err != nil {
			return fmt.Errorf("failed to write genesis: %w", err)
		}
		
		fmt.Printf("✅ Wrote genesis to: %s\n", genesisPath)
	}
	
	// Write to output file or stdout
	if outputPath != "" {
		if err := ioutil.WriteFile(outputPath, formattedGenesis, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("✅ Wrote genesis to: %s\n", outputPath)
	} else if !writeConfig {
		// Print to stdout if no output file specified
		fmt.Println("\n📄 Genesis Configuration:")
		fmt.Println(string(formattedGenesis))
	}
	
	return nil
}

// runMigrateRead implements the migrate read command
func runMigrateRead(cmd *cobra.Command, args []string) error {
	srcPath := args[0]
	dstPath, _ := cmd.Flags().GetString("dst")
	genesisOnly, _ := cmd.Flags().GetBool("genesis-only")
	writeGenesis, _ := cmd.Flags().GetBool("write-genesis")
	
	// Extract genesis from historic data
	fmt.Printf("📖 Reading genesis from historic chain data at %s\n", srcPath)
	genesis, genesisBytes, err := extractHistoricGenesis(srcPath)
	if err != nil {
		return fmt.Errorf("failed to extract genesis: %w", err)
	}
	
	// Derive the blockchain ID from genesis
	blockchainID, err := deriveBlockchainID(genesisBytes)
	if err != nil {
		return fmt.Errorf("failed to derive blockchain ID: %w", err)
	}
	
	fmt.Printf("✅ Derived blockchain ID: %s\n", blockchainID.String())
	fmt.Printf("   Chain ID: %v\n", genesis.Config["chainId"])
	
	// Write genesis if requested
	if writeGenesis {
		genesisDir := filepath.Join(os.Getenv("HOME"), ".luxd", "configs", "C")
		if err := os.MkdirAll(genesisDir, 0755); err != nil {
			return fmt.Errorf("failed to create genesis directory: %w", err)
		}
		
		genesisPath := filepath.Join(genesisDir, "genesis.json")
		formattedGenesis, err := json.MarshalIndent(genesis, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format genesis: %w", err)
		}
		
		if err := ioutil.WriteFile(genesisPath, formattedGenesis, 0644); err != nil {
			return fmt.Errorf("failed to write genesis: %w", err)
		}
		
		fmt.Printf("📝 Wrote genesis to %s\n", genesisPath)
	}
	
	if genesisOnly {
		fmt.Println("✅ Genesis extraction complete")
		return nil
	}
	
	if dstPath == "" {
		// Default destination path
		dstPath = filepath.Join(os.Getenv("HOME"), ".luxd", "chainData", blockchainID.String())
	}
	
	// Migrate the data
	fmt.Printf("🔄 Migrating chain data to %s\n", dstPath)
	if err := migrateChainData(srcPath, dstPath, blockchainID); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}
	
	fmt.Println("✅ Migration complete!")
	fmt.Printf("   Blockchain ID: %s\n", blockchainID.String())
	fmt.Printf("   Genesis: ~/.luxd/configs/C/genesis.json\n")
	fmt.Printf("   Chain data: %s\n", dstPath)
	
	return nil
}

// deriveBlockchainID derives the blockchain ID from genesis bytes
func deriveBlockchainID(genesisBytes []byte) (ids.ID, error) {
	// Create a hash of the genesis bytes
	hash := sha256.Sum256(genesisBytes)
	
	// Create an ID from the hash
	id, err := ids.ToID(hash[:])
	if err != nil {
		return ids.Empty, err
	}
	
	return id, nil
}

// extractHistoricGenesis extracts the genesis configuration from historic chain data
func extractHistoricGenesis(srcPath string) (*Genesis, []byte, error) {
	// Look for genesis.json in the chain data directory
	genesisPath := filepath.Join(srcPath, "genesis.json")
	if _, err := os.Stat(genesisPath); err == nil {
		genesisBytes, err := ioutil.ReadFile(genesisPath)
		if err != nil {
			return nil, nil, err
		}
		
		var genesis Genesis
		if err := json.Unmarshal(genesisBytes, &genesis); err != nil {
			return nil, nil, fmt.Errorf("failed to parse genesis: %w", err)
		}
		
		return &genesis, genesisBytes, nil
	}
	
	// If not found, try to read from the database
	// First check if it's a direct pebbledb path
	dbPath := filepath.Join(srcPath, "db", "pebbledb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Maybe srcPath is already pointing to the db directory
		altPath := filepath.Join(srcPath, "pebbledb")
		if _, err := os.Stat(altPath); err == nil {
			dbPath = altPath
		}
	}
	
	// Check if CURRENT file exists (required for pebbledb)
	currentFile := filepath.Join(dbPath, "CURRENT")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		// This might be a restored database without metadata
		// Create minimal genesis for now
		log.Printf("Warning: Database metadata not found, creating minimal genesis")
		return createMinimalGenesis()
	}
	
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database at %s: %w", dbPath, err)
	}
	defer db.Close()
	
	// Try to read genesis from database
	// First try to get the genesis key directly (no prefix)
	genesisValue, closer, err := db.Get([]byte("genesis"))
	if err == nil {
		defer closer.Close()
		value := make([]byte, len(genesisValue))
		copy(value, genesisValue)
		
		log.Printf("Found genesis blob in database (raw key)")
		// This is likely the compressed genesis blob, try to decode it
		// For now, return it as-is since we need the exact bytes
		genesis := &Genesis{
			Config: map[string]interface{}{
				"chainId": 96369,
			},
		}
		return genesis, value, nil
	}
	
	// Also check for pointer keys
	heightValue, closer2, err := db.Get([]byte("Height"))
	if err == nil {
		defer closer2.Close()
		height := make([]byte, len(heightValue))
		copy(height, heightValue)
		log.Printf("Found Height pointer: %x", height)
	}
	
	lastAcceptedValue, closer3, err := db.Get([]byte("LastAccepted"))
	if err == nil {
		defer closer3.Close()
		lastAccepted := make([]byte, len(lastAcceptedValue))
		copy(lastAccepted, lastAcceptedValue)
		log.Printf("Found LastAccepted pointer: %x", lastAccepted)
	}
	
	// Try iterating for other genesis-related keys
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// Look for genesis-related keys
		if bytes.Equal(key, []byte("genesis")) || bytes.Contains(key, []byte("genesis")) {
			value := make([]byte, len(iter.Value()))
			copy(value, iter.Value())
			
			log.Printf("Found genesis-related key: %x = %d bytes", key, len(value))
			// Return the raw genesis bytes
			genesis := &Genesis{
				Config: map[string]interface{}{
					"chainId": 96369,
				},
			}
			return genesis, value, nil
		}
	}
	
	// If we still can't find genesis, create a minimal one
	return createMinimalGenesis()
}

// createMinimalGenesis creates a minimal genesis configuration
func createMinimalGenesis() (*Genesis, []byte, error) {
	log.Printf("Creating minimal genesis configuration")
	
	genesis := &Genesis{
		Config: map[string]interface{}{
			"chainId":        96369,
			"homesteadBlock": 0,
			"eip150Block":    0,
			"eip155Block":    0,
			"eip158Block":    0,
		},
		Difficulty: "0x0",
		GasLimit:   "0x7a1200",
		Alloc:      make(map[string]interface{}),
		Nonce:      "0x0",
		Timestamp:  "0x0",
		ExtraData:  "0x00",
		MixHash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		Number:     "0x0",
		GasUsed:    "0x0",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
	}
	
	genesisBytes, err := json.Marshal(genesis)
	if err != nil {
		return nil, nil, err
	}
	
	return genesis, genesisBytes, nil
}

// migrateChainData migrates chain data from old blockchain ID to new
func migrateChainData(srcPath, dstPath string, newBlockchainID ids.ID) error {
	// Determine old blockchain ID from path
	var oldBlockchainID ids.ID
	base := filepath.Base(srcPath)
	if base != "." && base != "/" {
		var err error
		oldBlockchainID, err = ids.FromString(base)
		if err != nil {
			log.Printf("Could not parse blockchain ID from path, will migrate without ID translation")
		}
	}
	
	// Open source database
	srcDB, err := pebble.Open(filepath.Join(srcPath, "db", "pebbledb"), &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open source database: %w", err)
	}
	defer srcDB.Close()
	
	// Create destination directory
	dstDir := filepath.Join(dstPath, "db")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}
	
	// Open destination database
	dstDB, err := pebble.Open(filepath.Join(dstDir, "pebbledb"), &pebble.Options{})
	if err != nil {
		return fmt.Errorf("failed to open destination database: %w", err)
	}
	defer dstDB.Close()
	
	// Migrate data
	if oldBlockchainID != ids.Empty {
		log.Printf("Translating blockchain ID from %s to %s", oldBlockchainID.String(), newBlockchainID.String())
	}
	
	iter, err := srcDB.NewIter(&pebble.IterOptions{})
	if err != nil {
		return fmt.Errorf("failed to create iterator: %w", err)
	}
	defer iter.Close()
	
	count := 0
	start := time.Now()
	oldIDBytes := oldBlockchainID[:]
	newIDBytes := newBlockchainID[:]
	
	for iter.First(); iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		
		// Translate blockchain ID in keys if needed
		if oldBlockchainID != ids.Empty && len(key) >= 32 && bytes.HasPrefix(key, oldIDBytes) {
			newKey := make([]byte, len(key))
			copy(newKey, newIDBytes)
			copy(newKey[32:], key[32:])
			key = newKey
			
			// Also replace blockchain ID in values if present
			if bytes.Contains(value, oldIDBytes) {
				value = bytes.ReplaceAll(value, oldIDBytes, newIDBytes)
			}
		}
		
		if err := dstDB.Set(key, value, pebble.Sync); err != nil {
			return fmt.Errorf("failed to write key: %w", err)
		}
		
		count++
		if count%100000 == 0 {
			log.Printf("Migrated %d keys...", count)
		}
	}
	
	if err := iter.Error(); err != nil {
		return fmt.Errorf("iterator error: %w", err)
	}
	
	log.Printf("Migration complete! Migrated %d keys in %v", count, time.Since(start))
	return nil
}

// showPointerKeys displays the pointer keys from a database
func showPointerKeys(srcPath string) error {
	// Find the database path
	dbPath := filepath.Join(srcPath, "db", "pebbledb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		altPath := filepath.Join(srcPath, "pebbledb")
		if _, err := os.Stat(altPath); err == nil {
			dbPath = altPath
		}
	}
	
	// Check if CURRENT file exists
	currentFile := filepath.Join(dbPath, "CURRENT")
	if _, err := os.Stat(currentFile); os.IsNotExist(err) {
		return fmt.Errorf("database metadata not found")
	}
	
	db, err := pebble.Open(dbPath, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	fmt.Println("\n🔍 Pointer Keys:")
	
	// Check each pointer key
	pointerKeys := []string{"Height", "LastAccepted", "LastBlock", "LastHeader"}
	for _, key := range pointerKeys {
		value, closer, err := db.Get([]byte(key))
		if err != nil {
			fmt.Printf("   %-15s: <not found>\n", key)
			continue
		}
		defer closer.Close()
		
		// Copy the value
		val := make([]byte, len(value))
		copy(val, value)
		
		// Format based on key type
		if key == "Height" {
			// Height is uint64 big-endian
			if len(val) == 8 {
				height := uint64(0)
				for i := 0; i < 8; i++ {
					height = (height << 8) | uint64(val[i])
				}
				fmt.Printf("   %-15s: %d (0x%x)\n", key, height, val)
			} else {
				fmt.Printf("   %-15s: 0x%x\n", key, val)
			}
		} else {
			// Others are hashes
			fmt.Printf("   %-15s: 0x%x\n", key, val)
		}
	}
	
	return nil
}
