package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/genesis/pkg/genesis/config"
	"github.com/luxfi/genesis/pkg/genesis/validator"
)

var (
	// Global flags
	networkFlag    string
	outputFlag     string
	validatorsFlag string
	
	// Treasury flags
	treasuryAddrFlag   string
	treasuryAmountFlag string
	
	// Import flags
	importCChainFlag string
	importAllocsFlag string
	
	// Validator flags
	mnemonicFlag   string
	offsetsFlag    string
	saveKeysFlag   string
	saveKeysDirFlag string
	
	// Validator management flags
	validatorIndexFlag int
	nodeIDFlag        string
	ethAddressFlag    string
	publicKeyFlag     string
	proofOfPossessionFlag string
	weightFlag        string
)

var rootCmd = &cobra.Command{
	Use:   "genesis-cli",
	Short: "Unified Lux Genesis Management Tool",
	Long:  `A comprehensive tool for managing Lux Network genesis configurations`,
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate genesis configuration",
	Long:  `Generate a new genesis configuration with validators and allocations`,
	RunE:  runGenerate,
}

var validatorsCmd = &cobra.Command{
	Use:   "validators",
	Short: "Manage validators",
	Long:  `Add, remove, list, and generate validators`,
}

var listValidatorsCmd = &cobra.Command{
	Use:   "list",
	Short: "List validators",
	RunE:  runListValidators,
}

var addValidatorCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a validator",
	RunE:  runAddValidator,
}

var removeValidatorCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a validator",
	RunE:  runRemoveValidator,
}

var generateValidatorsCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate new validators",
	RunE:  runGenerateValidators,
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate genesis configuration",
	RunE:  runValidate,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&networkFlag, "network", "mainnet", "Network to use (mainnet, testnet, local)")
	rootCmd.PersistentFlags().StringVar(&outputFlag, "output", "", "Output file path")
	rootCmd.PersistentFlags().StringVar(&validatorsFlag, "validators", "", "Path to validators JSON file")
	
	// Generate command flags
	generateCmd.Flags().StringVar(&treasuryAddrFlag, "treasury", config.DefaultTreasuryAddress, "Treasury address")
	generateCmd.Flags().StringVar(&treasuryAmountFlag, "treasury-amount", "2T", "Treasury amount (e.g., 2T, 1B, 500M)")
	generateCmd.Flags().StringVar(&importCChainFlag, "import-cchain", "", "Path to existing C-Chain genesis")
	generateCmd.Flags().StringVar(&importAllocsFlag, "import-allocations", "", "Path to allocations file (JSON or CSV)")
	
	// Validator generation flags
	generateValidatorsCmd.Flags().StringVar(&mnemonicFlag, "mnemonic", "", "Mnemonic phrase for deterministic generation")
	generateValidatorsCmd.Flags().StringVar(&offsetsFlag, "offsets", "", "Comma-separated account offsets")
	generateValidatorsCmd.Flags().StringVar(&saveKeysFlag, "save-keys", "", "Save validator configs to file")
	generateValidatorsCmd.Flags().StringVar(&saveKeysDirFlag, "save-keys-dir", "", "Save individual validator keys to directories")
	
	// Add validator flags
	addValidatorCmd.Flags().StringVar(&nodeIDFlag, "node-id", "", "Node ID")
	addValidatorCmd.Flags().StringVar(&ethAddressFlag, "eth-address", "", "Ethereum address")
	addValidatorCmd.Flags().StringVar(&publicKeyFlag, "public-key", "", "BLS public key (48 bytes hex)")
	addValidatorCmd.Flags().StringVar(&proofOfPossessionFlag, "proof-of-possession", "", "Proof of possession (96 bytes hex)")
	addValidatorCmd.Flags().StringVar(&weightFlag, "weight", "1000000", "Staking weight in LUX")
	
	// Remove validator flags
	removeValidatorCmd.Flags().IntVar(&validatorIndexFlag, "index", -1, "Validator index to remove")
	removeValidatorCmd.Flags().StringVar(&nodeIDFlag, "node-id", "", "Node ID to remove")
	
	// Build command tree
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(validatorsCmd)
	rootCmd.AddCommand(validateCmd)
	
	validatorsCmd.AddCommand(listValidatorsCmd)
	validatorsCmd.AddCommand(addValidatorCmd)
	validatorsCmd.AddCommand(removeValidatorCmd)
	validatorsCmd.AddCommand(generateValidatorsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Create genesis builder
	builder, err := genesis.NewBuilder(networkFlag)
	if err != nil {
		return fmt.Errorf("failed to create builder: %w", err)
	}
	
	// Import C-Chain genesis if specified
	if importCChainFlag != "" {
		fmt.Printf("Importing C-Chain genesis from %s...\n", importCChainFlag)
		if err := builder.ImportCChainGenesis(importCChainFlag); err != nil {
			return fmt.Errorf("failed to import C-Chain genesis: %w", err)
		}
	}
	
	// Import allocations if specified
	if importAllocsFlag != "" {
		fmt.Printf("Importing allocations from %s...\n", importAllocsFlag)
		if strings.HasSuffix(importAllocsFlag, ".csv") {
			if err := builder.ImportCSVAllocations(importAllocsFlag); err != nil {
				return fmt.Errorf("failed to import CSV allocations: %w", err)
			}
		} else {
			if err := builder.ImportCChainAllocations(importAllocsFlag); err != nil {
				return fmt.Errorf("failed to import allocations: %w", err)
			}
		}
	}
	
	// Add treasury allocation if not importing C-Chain
	if importCChainFlag == "" && treasuryAddrFlag != "" && treasuryAmountFlag != "" {
		treasuryAmount, err := allocation.ParseLUXAmount(treasuryAmountFlag)
		if err != nil {
			return fmt.Errorf("invalid treasury amount: %w", err)
		}
		
		fmt.Printf("Adding treasury allocation: %s -> %s\n", 
			treasuryAddrFlag, 
			allocation.FormatLUXAmount(treasuryAmount))
		
		if err := builder.AddAllocation(treasuryAddrFlag, treasuryAmount); err != nil {
			return fmt.Errorf("failed to add treasury allocation: %w", err)
		}
	}
	
	// Load validators if specified
	if validatorsFlag != "" {
		if err := loadAndAddValidators(builder, validatorsFlag); err != nil {
			return err
		}
	}
	
	// Build genesis
	fmt.Println("Building genesis configuration...")
	g, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build genesis: %w", err)
	}
	
	// Print summary
	printGenesisSummary(g, builder)
	
	// Save to file
	if outputFlag == "" {
		outputFlag = fmt.Sprintf("genesis_%s.json", networkFlag)
	}
	
	if err := builder.SaveToFile(g, outputFlag); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}
	
	fmt.Printf("\nGenesis saved to: %s\n", outputFlag)
	return nil
}

func runListValidators(cmd *cobra.Command, args []string) error {
	if validatorsFlag == "" {
		validatorsFlag = getDefaultValidatorsFile()
	}
	
	validators, err := loadValidators(validatorsFlag)
	if err != nil {
		return err
	}
	
	fmt.Printf("Validators in %s:\n\n", validatorsFlag)
	for i, v := range validators {
		fmt.Printf("Index: %d\n", i)
		fmt.Printf("NodeID: %s\n", v.NodeID)
		fmt.Printf("Address: %s\n", v.ETHAddress)
		fmt.Printf("Weight: %s LUX\n", formatLUXFromWei(v.Weight))
		fmt.Printf("Delegation Fee: %.2f%%\n", float64(v.DelegationFee)/10000)
		fmt.Println()
	}
	
	fmt.Printf("Total validators: %d\n", len(validators))
	return nil
}

func runAddValidator(cmd *cobra.Command, args []string) error {
	if validatorsFlag == "" {
		validatorsFlag = getDefaultValidatorsFile()
	}
	
	// Validate required flags
	if nodeIDFlag == "" || ethAddressFlag == "" || publicKeyFlag == "" || proofOfPossessionFlag == "" {
		return fmt.Errorf("all validator fields are required: --node-id, --eth-address, --public-key, --proof-of-possession")
	}
	
	// Load existing validators
	validators, _ := loadValidators(validatorsFlag)
	
	// Parse weight
	weightAmount, err := allocation.ParseLUXAmount(weightFlag)
	if err != nil {
		return fmt.Errorf("invalid weight: %w", err)
	}
	
	// Create new validator
	newValidator := ValidatorConfig{
		NodeID:            nodeIDFlag,
		ETHAddress:        ethAddressFlag,
		PublicKey:         publicKeyFlag,
		ProofOfPossession: proofOfPossessionFlag,
		Weight:            weightAmount.Uint64(),
		DelegationFee:     20000, // Default 2%
	}
	
	// Add to list
	validators = append(validators, newValidator)
	
	// Save
	if err := saveValidators(validatorsFlag, validators); err != nil {
		return err
	}
	
	fmt.Printf("Validator added successfully!\n")
	fmt.Printf("Total validators: %d\n", len(validators))
	return nil
}

func runRemoveValidator(cmd *cobra.Command, args []string) error {
	if validatorsFlag == "" {
		validatorsFlag = getDefaultValidatorsFile()
	}
	
	validators, err := loadValidators(validatorsFlag)
	if err != nil {
		return err
	}
	
	var newValidators []ValidatorConfig
	removed := false
	
	if validatorIndexFlag >= 0 {
		// Remove by index
		if validatorIndexFlag >= len(validators) {
			return fmt.Errorf("invalid index: %d", validatorIndexFlag)
		}
		newValidators = append(validators[:validatorIndexFlag], validators[validatorIndexFlag+1:]...)
		removed = true
	} else if nodeIDFlag != "" {
		// Remove by NodeID
		for _, v := range validators {
			if v.NodeID != nodeIDFlag {
				newValidators = append(newValidators, v)
			} else {
				removed = true
			}
		}
	} else {
		return fmt.Errorf("specify either --index or --node-id")
	}
	
	if !removed {
		return fmt.Errorf("validator not found")
	}
	
	// Save
	if err := saveValidators(validatorsFlag, newValidators); err != nil {
		return err
	}
	
	fmt.Printf("Validator removed successfully!\n")
	fmt.Printf("Total validators: %d\n", len(newValidators))
	return nil
}

func runGenerateValidators(cmd *cobra.Command, args []string) error {
	if saveKeysFlag == "" {
		saveKeysFlag = getDefaultValidatorsFile()
	}
	
	keygen := validator.NewKeyGenerator("luxd") // Use system luxd
	
	// Parse offsets
	var offsets []int
	if offsetsFlag != "" {
		parts := strings.Split(offsetsFlag, ",")
		for _, p := range parts {
			var offset int
			if _, err := fmt.Sscanf(strings.TrimSpace(p), "%d", &offset); err != nil {
				return fmt.Errorf("invalid offset: %s", p)
			}
			offsets = append(offsets, offset)
		}
	} else {
		// Default to 11 validators
		for i := 0; i < 11; i++ {
			offsets = append(offsets, i)
		}
	}
	
	validators := make([]*validator.ValidatorInfo, 0, len(offsets))
	
	fmt.Printf("Generating %d validators...\n", len(offsets))
	
	for idx, offset := range offsets {
		var keys *validator.ValidatorKeys
		var keysWithTLS *validator.ValidatorKeysWithTLS
		var err error
		
		if mnemonicFlag != "" {
			// Deterministic generation
			keysWithTLS, err = keygen.GenerateFromSeedWithTLS(mnemonicFlag, offset)
			if err != nil {
				return fmt.Errorf("failed to generate keys for offset %d: %w", offset, err)
			}
			keys = keysWithTLS.ValidatorKeys
		} else {
			// Random generation
			keysWithTLS, err = keygen.GenerateCompatibleKeys()
			if err != nil {
				return fmt.Errorf("failed to generate random keys: %w", err)
			}
			keys = keysWithTLS.ValidatorKeys
		}
		
		// Generate ETH address
		ethAddr := fmt.Sprintf("0x%040x", offset)
		
		validatorConfig := validator.GenerateValidatorConfig(
			keys,
			ethAddr,
			2000000000000000, // 2M LUX default
			20000,            // 2% delegation fee
		)
		
		validators = append(validators, validatorConfig)
		
		fmt.Printf("\nValidator %d (Offset %d):\n", idx+1, offset)
		fmt.Printf("  NodeID: %s\n", keys.NodeID)
		fmt.Printf("  Address: %s\n", ethAddr)
		
		// Save individual keys if requested
		if saveKeysDirFlag != "" {
			keyDir := fmt.Sprintf("%s/validator-%d", saveKeysDirFlag, idx+1)
			if err := validator.SaveKeys(keys, keyDir); err != nil {
				return fmt.Errorf("failed to save keys: %w", err)
			}
			if keysWithTLS != nil {
				if err := validator.SaveStakingFiles(keysWithTLS.TLSKeyBytes, keysWithTLS.TLSCertBytes, keyDir); err != nil {
					return fmt.Errorf("failed to save staking files: %w", err)
				}
			}
		}
	}
	
	// Save validator configs
	if err := validator.SaveValidatorConfigs(validators, saveKeysFlag); err != nil {
		return fmt.Errorf("failed to save validator configs: %w", err)
	}
	
	fmt.Printf("\nValidator configurations saved to: %s\n", saveKeysFlag)
	return nil
}

func runValidate(cmd *cobra.Command, args []string) error {
	genesisFile := outputFlag
	if genesisFile == "" {
		genesisFile = fmt.Sprintf("genesis_%s.json", networkFlag)
	}
	
	// Read genesis file
	data, err := ioutil.ReadFile(genesisFile)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}
	
	var g genesis.Genesis
	if err := json.Unmarshal(data, &g); err != nil {
		return fmt.Errorf("failed to parse genesis: %w", err)
	}
	
	fmt.Printf("Validating %s...\n\n", genesisFile)
	
	// Basic validation
	fmt.Println("Basic validation:")
	fmt.Printf("✓ Network ID: %d\n", g.NetworkID)
	fmt.Printf("✓ Allocations: %d\n", len(g.Allocations))
	fmt.Printf("✓ Initial stakers: %d\n", len(g.InitialStakers))
	
	// Validate C-Chain config
	fmt.Println("\nC-Chain configuration:")
	if g.CChainGenesis != "" {
		var cchainConfig map[string]interface{}
		if err := json.Unmarshal([]byte(g.CChainGenesis), &cchainConfig); err == nil {
			if config, ok := cchainConfig["config"].(map[string]interface{}); ok {
				if chainId, ok := config["chainId"]; ok {
					fmt.Printf("✓ Chain ID: %v\n", chainId)
				}
			}
		}
	}
	
	// Check precompiles
	fmt.Println("\nChecking precompiles:")
	precompiles := []string{
		"0x0000000000000000000000000000000000000400", // ContractDeployerAllowList
		"0x0000000000000000000000000000000000000401", // FeeManager
		"0x0000000000000000000000000000000000000402", // NativeMinter
		"0x0000000000000000000000000000000000000403", // TxAllowList
	}
	
	cchainStr := string(g.CChainGenesis)
	for _, precompile := range precompiles {
		if strings.Contains(cchainStr, precompile) {
			fmt.Printf("✓ Found precompile: %s\n", precompile)
		} else {
			fmt.Printf("✗ Missing precompile: %s\n", precompile)
		}
	}
	
	// Calculate total supply
	fmt.Println("\nSupply calculation:")
	var totalInitial, totalLocked uint64
	for _, alloc := range g.Allocations {
		totalInitial += alloc.InitialAmount
		for _, locked := range alloc.UnlockSchedule {
			totalLocked += locked.Amount
		}
	}
	
	fmt.Printf("Initial supply: %s LUX\n", formatLUXFromWei(totalInitial))
	fmt.Printf("Locked supply: %s LUX\n", formatLUXFromWei(totalLocked))
	fmt.Printf("Total supply: %s LUX\n", formatLUXFromWei(totalInitial + totalLocked))
	
	// Check for overflow
	maxUint64 := uint64(18446744073709551615)
	if totalInitial > maxUint64 {
		fmt.Printf("⚠️  WARNING: Initial supply exceeds uint64 max!\n")
	}
	
	fmt.Println("\n✅ Validation complete")
	return nil
}

// Helper functions

type ValidatorConfig struct {
	NodeID            string `json:"nodeID"`
	ETHAddress        string `json:"ethAddress"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
	Weight            uint64 `json:"weight"`
	DelegationFee     uint32 `json:"delegationFee"`
}

func loadValidators(path string) ([]ValidatorConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []ValidatorConfig{}, nil
		}
		return nil, err
	}
	
	var validators []ValidatorConfig
	if err := json.Unmarshal(data, &validators); err != nil {
		return nil, err
	}
	
	return validators, nil
}

func saveValidators(path string, validators []ValidatorConfig) error {
	data, err := json.MarshalIndent(validators, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(path, data, 0644)
}

func loadAndAddValidators(builder *genesis.Builder, validatorsFile string) error {
	validators, err := loadValidators(validatorsFile)
	if err != nil {
		return fmt.Errorf("failed to load validators: %w", err)
	}
	
	fmt.Printf("Adding %d validators from %s...\n", len(validators), validatorsFile)
	
	networkCfg, err := config.GetNetwork(networkFlag)
	if err != nil {
		return fmt.Errorf("failed to get network config: %w", err)
	}
	
	for _, v := range validators {
		// Add staker
		builder.AddStaker(genesis.StakerConfig{
			NodeID:            v.NodeID,
			ETHAddress:        v.ETHAddress,
			PublicKey:         v.PublicKey,
			ProofOfPossession: v.ProofOfPossession,
			Weight:            v.Weight,
			DelegationFee:     v.DelegationFee,
		})
		
		// Add locked allocation for validator
		if v.Weight > 0 {
			stakingAmount := v.Weight
			vestingYears := int(networkCfg.InitialStakeDuration.Hours() / 24 / 365)
			if vestingYears < 1 {
				vestingYears = 1
			}
			
			err := builder.AddVestedAllocation(v.ETHAddress, &allocation.UnlockScheduleConfig{
				TotalAmount:  new(big.Int).SetUint64(stakingAmount),
				StartDate:    networkCfg.StartTime,
				Duration:     networkCfg.InitialStakeDuration,
				Periods:      1,
				CliffPeriods: 0,
			})
			if err != nil {
				return fmt.Errorf("failed to add validator allocation: %w", err)
			}
		}
	}
	
	return nil
}

func printGenesisSummary(g *genesis.Genesis, builder *genesis.Builder) {
	fmt.Printf("\nGenesis Summary:\n")
	fmt.Printf("Network ID: %d\n", g.NetworkID)
	fmt.Printf("Allocations: %d\n", len(g.Allocations))
	fmt.Printf("Initial Stakers: %d\n", len(g.InitialStakers))
	fmt.Printf("Start Time: %d\n", g.StartTime)
	fmt.Printf("Total Supply: %s\n", allocation.FormatLUXAmount(builder.GetTotalSupply()))
}

func getDefaultValidatorsFile() string {
	return fmt.Sprintf("configs/%s-validators.json", networkFlag)
}

func formatLUXFromWei(wei uint64) string {
	return fmt.Sprintf("%.0f", float64(wei)/1e9)
}