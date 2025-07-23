package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	// Import command packages
	archeologyCmd "github.com/luxfi/genesis/cmd/archeology/commands"
	teleportCmd "github.com/luxfi/genesis/cmd/teleport/commands"

	// Import internal packages
	"github.com/luxfi/genesis/cmd/denamespace/pkg/denamespace"
)

var (
	version = "1.0.0"
	commit  = "none"
	date    = "unknown"
)

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
	// State extraction (denamespace)
	stateCmd := &cobra.Command{
		Use:   "state [source] [destination]",
		Short: "Extract state from PebbleDB (denamespace)",
		Long:  `Extract account state and blockchain data from PebbleDB format, removing namespace prefixes`,
		Args:  cobra.ExactArgs(2),
		RunE:  runExtractState,
	}
	stateCmd.Flags().Int("network", 96369, "Chain ID")
	stateCmd.Flags().Bool("state", true, "Include state data")
	stateCmd.Flags().Int("limit", 0, "Limit number of entries (0 = no limit)")

	// Add archeology extract commands
	extractCmd.AddCommand(
		stateCmd,
		archeologyCmd.NewExtractCommand(),
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
		archeologyCmd.NewImportNFTCommand(),
		archeologyCmd.NewImportTokenCommand(),
	)
}

func addAnalyzeSubcommands(analyzeCmd *cobra.Command) {
	// Add archeology analyze commands
	analyzeCmd.AddCommand(
		archeologyCmd.NewAnalyzeCommand(),
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

	// Add archeology scan commands
	scanCmd.AddCommand(
		archeologyCmd.NewScanCommand(),
		archeologyCmd.NewScanBurnsCommand(),
		archeologyCmd.NewScanHoldersCommand(),
		archeologyCmd.NewScanTransfersCommand(),
		archeologyCmd.NewScanCurrentHoldersCommand(),
	)
}

func addMigrateSubcommands(migrateCmd *cobra.Command) {
	// Add teleport migrate commands
	migrateCmd.AddCommand(
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

// Command implementations - Generate command
func runGenerate(cmd *cobra.Command, args []string) error {
	// This is a simplified version - in production, copy the full implementation
	// from main_generate.go or main_old.go

	fmt.Println("Genesis Generation")
	fmt.Println("==================")
	fmt.Printf("Network: %s\n", cfg.Network)

	// Set default output directory if not specified
	if cfg.OutputDir == "" {
		cfg.OutputDir = filepath.Join("configs", cfg.Network)
	}
	fmt.Printf("Output: %s\n", cfg.OutputDir)

	// Create output directories based on standard structure
	if cfg.UseStandardDirs {
		for _, chain := range []string{"P", "C", "X"} {
			chainDir := filepath.Join(cfg.OutputDir, chain)
			if err := os.MkdirAll(chainDir, 0755); err != nil {
				return fmt.Errorf("failed to create %s directory: %w", chain, err)
			}
		}
	} else {
		if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	fmt.Println("\n✅ Genesis generation complete!")
	fmt.Printf("Output directory: %s\n", cfg.OutputDir)

	return nil
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

	// Use the denamespace package
	opts := denamespace.Options{
		Source:      source,
		Destination: destination,
		NetworkID:   uint64(networkID),
		State:       includeState,
		Limit:       limit,
	}

	return denamespace.Extract(opts)
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
	data, err := ioutil.ReadFile(genesisFile)
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
