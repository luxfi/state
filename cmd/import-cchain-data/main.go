package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

// CChainGenesis represents the C-Chain genesis structure
type CChainGenesis struct {
	Config     ChainConfig               `json:"config"`
	Nonce      string                    `json:"nonce"`
	Timestamp  string                    `json:"timestamp"`
	ExtraData  string                    `json:"extraData"`
	GasLimit   string                    `json:"gasLimit"`
	Difficulty string                    `json:"difficulty"`
	MixHash    string                    `json:"mixHash"`
	Coinbase   string                    `json:"coinbase"`
	Alloc      map[string]AccountAlloc   `json:"alloc"`
	Number     string                    `json:"number"`
	GasUsed    string                    `json:"gasUsed"`
	ParentHash string                    `json:"parentHash"`
	BaseFee    string                    `json:"baseFeePerGas,omitempty"`
}

// ChainConfig represents the chain configuration
type ChainConfig struct {
	ChainID                *big.Int               `json:"chainId"`
	HomesteadBlock         *big.Int               `json:"homesteadBlock"`
	EIP150Block            *big.Int               `json:"eip150Block"`
	EIP150Hash             string                 `json:"eip150Hash"`
	EIP155Block            *big.Int               `json:"eip155Block"`
	EIP158Block            *big.Int               `json:"eip158Block"`
	ByzantiumBlock         *big.Int               `json:"byzantiumBlock"`
	ConstantinopleBlock    *big.Int               `json:"constantinopleBlock"`
	PetersburgBlock        *big.Int               `json:"petersburgBlock"`
	IstanbulBlock          *big.Int               `json:"istanbulBlock"`
	MuirGlacierBlock       *big.Int               `json:"muirGlacierBlock,omitempty"`
	BerlinBlock            *big.Int               `json:"berlinBlock,omitempty"`
	LondonBlock            *big.Int               `json:"londonBlock,omitempty"`
	FeeConfig              map[string]interface{} `json:"feeConfig"`
	AllowFeeRecipients     bool                   `json:"allowFeeRecipients"`
}

// AccountAlloc represents an account allocation
type AccountAlloc struct {
	Balance string                 `json:"balance"`
	Code    string                 `json:"code,omitempty"`
	Storage map[string]string      `json:"storage,omitempty"`
	Nonce   string                 `json:"nonce,omitempty"`
}

// ImportConfig holds all import configuration
type ImportConfig struct {
	OutputFile          string
	ChainID             int64
	Lux7777CSV          string
	Zoo200200CSV        string
	ExistingCChain      string
	BSCHolders          string
	ETHHolders          string
	TreasuryAddress     string
	ValidateOnly        bool
	GenerateReport      bool
}

// ImportReport tracks import statistics
type ImportReport struct {
	Timestamp         time.Time
	ChainID           int64
	TotalAccounts     int
	TotalSupply       *big.Int
	ImportedFrom      map[string]int
	LargestHolders    []HolderInfo
	ValidationErrors  []string
	DuplicateHandling map[string]string
}

type HolderInfo struct {
	Address    string
	Balance    *big.Int
	Source     string
	Percentage float64
}

var (
	cfg = &ImportConfig{}
	
	rootCmd = &cobra.Command{
		Use:   "import-cchain-data",
		Short: "Import historic blockchain data into C-Chain genesis",
		Long:  `Imports data from Lux 7777, Zoo 200200, and cross-chain deployments into a unified C-Chain genesis`,
		RunE:  runImport,
	}
)

func init() {
	rootCmd.Flags().StringVar(&cfg.OutputFile, "output", "cchain-genesis-complete.json", "Output genesis file")
	rootCmd.Flags().Int64Var(&cfg.ChainID, "chain-id", 96369, "C-Chain ID")
	rootCmd.Flags().StringVar(&cfg.Lux7777CSV, "lux7777", "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet.csv", "Lux 7777 CSV file")
	rootCmd.Flags().StringVar(&cfg.Zoo200200CSV, "zoo", "exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.csv", "Zoo allocations CSV")
	rootCmd.Flags().StringVar(&cfg.ExistingCChain, "existing", "", "Existing C-Chain genesis to merge")
	rootCmd.Flags().StringVar(&cfg.BSCHolders, "bsc", "", "BSC holders data")
	rootCmd.Flags().StringVar(&cfg.ETHHolders, "eth", "", "ETH holders data")
	rootCmd.Flags().StringVar(&cfg.TreasuryAddress, "treasury", "0x9011e888251ab053b7bd1cdb598db4f9ded94714", "Treasury address")
	rootCmd.Flags().BoolVar(&cfg.ValidateOnly, "validate", false, "Validate only, don't generate")
	rootCmd.Flags().BoolVar(&cfg.GenerateReport, "report", true, "Generate import report")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	fmt.Println("C-Chain Historic Data Import")
	fmt.Println("============================")
	fmt.Printf("Chain ID: %d\n", cfg.ChainID)
	fmt.Printf("Output: %s\n", cfg.OutputFile)
	
	// Initialize genesis
	genesis := createBaseGenesis(cfg.ChainID)
	
	// Initialize report
	report := &ImportReport{
		Timestamp:         time.Now(),
		ChainID:           cfg.ChainID,
		TotalSupply:       big.NewInt(0),
		ImportedFrom:      make(map[string]int),
		ValidationErrors:  []string{},
		DuplicateHandling: make(map[string]string),
	}
	
	// Import existing C-Chain if provided
	if cfg.ExistingCChain != "" {
		fmt.Printf("\n1. Loading existing C-Chain genesis from %s...\n", cfg.ExistingCChain)
		if err := importExistingGenesis(genesis, cfg.ExistingCChain, report); err != nil {
			return fmt.Errorf("failed to import existing genesis: %w", err)
		}
	}
	
	// Import Lux 7777 data (including treasury)
	fmt.Printf("\n2. Importing Lux 7777 allocations...\n")
	if err := importLux7777Data(genesis, cfg.Lux7777CSV, report); err != nil {
		return fmt.Errorf("failed to import Lux 7777 data: %w", err)
	}
	
	// Import Zoo 200200 data
	if cfg.Zoo200200CSV != "" && fileExists(cfg.Zoo200200CSV) {
		fmt.Printf("\n3. Importing Zoo 200200 allocations...\n")
		if err := importZoo200200Data(genesis, cfg.Zoo200200CSV, report); err != nil {
			return fmt.Errorf("failed to import Zoo 200200 data: %w", err)
		}
	}
	
	// Import cross-chain data
	if cfg.BSCHolders != "" && fileExists(cfg.BSCHolders) {
		fmt.Printf("\n4. Importing BSC holder data...\n")
		if err := importCrossChainData(genesis, cfg.BSCHolders, "BSC", report); err != nil {
			return fmt.Errorf("failed to import BSC data: %w", err)
		}
	}
	
	if cfg.ETHHolders != "" && fileExists(cfg.ETHHolders) {
		fmt.Printf("\n5. Importing ETH holder data...\n")
		if err := importCrossChainData(genesis, cfg.ETHHolders, "ETH", report); err != nil {
			return fmt.Errorf("failed to import ETH data: %w", err)
		}
	}
	
	// Add precompiles
	fmt.Printf("\n6. Adding precompiled contracts...\n")
	addPrecompiledContracts(genesis)
	
	// Calculate statistics
	calculateStatistics(genesis, report)
	
	// Validate
	fmt.Printf("\n7. Validating genesis...\n")
	validateGenesis(genesis, report)
	
	if cfg.ValidateOnly {
		printReport(report)
		return nil
	}
	
	// Save genesis
	fmt.Printf("\n8. Saving genesis...\n")
	if err := saveGenesis(genesis, cfg.OutputFile); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}
	
	// Generate report
	if cfg.GenerateReport {
		reportFile := strings.TrimSuffix(cfg.OutputFile, ".json") + "-report.json"
		if err := saveReport(report, reportFile); err != nil {
			return fmt.Errorf("failed to save report: %w", err)
		}
		fmt.Printf("Report saved to: %s\n", reportFile)
	}
	
	// Print summary
	printReport(report)
	
	fmt.Printf("\n✅ C-Chain genesis generated successfully!\n")
	fmt.Printf("Output file: %s\n", cfg.OutputFile)
	
	return nil
}

func createBaseGenesis(chainID int64) *CChainGenesis {
	return &CChainGenesis{
		Config: ChainConfig{
			ChainID:             big.NewInt(chainID),
			HomesteadBlock:      big.NewInt(0),
			EIP150Block:         big.NewInt(0),
			EIP150Hash:          "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
			EIP155Block:         big.NewInt(0),
			EIP158Block:         big.NewInt(0),
			ByzantiumBlock:      big.NewInt(0),
			ConstantinopleBlock: big.NewInt(0),
			PetersburgBlock:     big.NewInt(0),
			IstanbulBlock:       big.NewInt(0),
			MuirGlacierBlock:    big.NewInt(0),
			BerlinBlock:         big.NewInt(0),
			LondonBlock:         big.NewInt(0),
			FeeConfig: map[string]interface{}{
				"gasLimit":                  15000000,
				"targetBlockRate":           2,
				"minBaseFee":                25000000000,
				"targetGas":                 15000000,
				"baseFeeChangeDenominator":  36,
				"minBlockGasCost":           0,
				"maxBlockGasCost":           1000000,
				"blockGasCostStep":          200000,
			},
			AllowFeeRecipients: true,
		},
		Nonce:      "0x0",
		Timestamp:  "0x0",
		ExtraData:  "0x00",
		GasLimit:   "0xe4e1c0", // 15M
		Difficulty: "0x0",
		MixHash:    "0x0000000000000000000000000000000000000000000000000000000000000000",
		Coinbase:   "0x0000000000000000000000000000000000000000",
		Alloc:      make(map[string]AccountAlloc),
		Number:     "0x0",
		GasUsed:    "0x0",
		ParentHash: "0x0000000000000000000000000000000000000000000000000000000000000000",
		BaseFee:    "0x5d21dba00", // 25 gwei
	}
}

func importExistingGenesis(genesis *CChainGenesis, filename string, report *ImportReport) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	
	var existing CChainGenesis
	if err := json.Unmarshal(data, &existing); err != nil {
		return err
	}
	
	// Merge allocations
	count := 0
	for addr, alloc := range existing.Alloc {
		genesis.Alloc[strings.ToLower(addr)] = alloc
		count++
	}
	
	report.ImportedFrom["existing"] = count
	fmt.Printf("  Imported %d existing accounts\n", count)
	
	return nil
}

func importLux7777Data(genesis *CChainGenesis, csvFile string, report *ImportReport) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	count := 0
	skipped := 0
	totalImported := big.NewInt(0)
	
	// Process records
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 4 {
			continue
		}
		
		address := strings.ToLower(records[i][1])
		balanceWei := records[i][3]
		
		// Validate address
		if !common.IsHexAddress(address) {
			report.ValidationErrors = append(report.ValidationErrors, 
				fmt.Sprintf("Invalid address in Lux 7777: %s", address))
			skipped++
			continue
		}
		
		balance, ok := new(big.Int).SetString(balanceWei, 10)
		if !ok || balance.Sign() <= 0 {
			skipped++
			continue
		}
		
		// Check for existing allocation
		if existing, exists := genesis.Alloc[address]; exists {
			existingBal, _ := new(big.Int).SetString(existing.Balance, 0)
			newBalance := new(big.Int).Add(existingBal, balance)
			genesis.Alloc[address] = AccountAlloc{
				Balance: fmt.Sprintf("0x%x", newBalance),
			}
			report.DuplicateHandling[address] = "merged"
		} else {
			genesis.Alloc[address] = AccountAlloc{
				Balance: fmt.Sprintf("0x%x", balance),
			}
		}
		
		totalImported.Add(totalImported, balance)
		count++
	}
	
	report.ImportedFrom["lux7777"] = count
	fmt.Printf("  Imported %d accounts from Lux 7777 (skipped %d)\n", count, skipped)
	fmt.Printf("  Total LUX imported: %s wei\n", totalImported.String())
	
	return nil
}

func importZoo200200Data(genesis *CChainGenesis, csvFile string, report *ImportReport) error {
	file, err := os.Open(csvFile)
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	count := 0
	skipped := 0
	
	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 3 {
			continue
		}
		
		address := strings.ToLower(records[i][0])
		zooAmount := records[i][2]
		
		// Validate address
		if !common.IsHexAddress(address) {
			report.ValidationErrors = append(report.ValidationErrors,
				fmt.Sprintf("Invalid address in Zoo 200200: %s", address))
			skipped++
			continue
		}
		
		// Convert ZOO to wei (18 decimals)
		amount, ok := new(big.Int).SetString(zooAmount, 10)
		if !ok || amount.Sign() <= 0 {
			skipped++
			continue
		}
		
		// Convert to wei
		amountWei := new(big.Int).Mul(amount, big.NewInt(1e18))
		
		// Store as ZOO allocation (could be tracked separately)
		// For now, we'll add a comment to identify Zoo allocations
		genesis.Alloc[address] = AccountAlloc{
			Balance: fmt.Sprintf("0x%x", amountWei),
		}
		
		count++
	}
	
	report.ImportedFrom["zoo200200"] = count
	fmt.Printf("  Imported %d accounts from Zoo 200200 (skipped %d)\n", count, skipped)
	
	return nil
}

func importCrossChainData(genesis *CChainGenesis, dataFile string, source string, report *ImportReport) error {
	// This would import data from BSC or ETH holders
	// Implementation depends on the format of cross-chain data
	
	fmt.Printf("  Cross-chain import for %s not yet implemented\n", source)
	return nil
}

func addPrecompiledContracts(genesis *CChainGenesis) {
	// Admin address for precompiles
	adminAddress := "0x8db97C7cEcE249c2b98bDC0226Cc4C2A57BF52FC"
	
	precompiles := map[string]string{
		"0x0200000000000000000000000000000000000000": "ContractDeployerAllowList",
		"0x0200000000000000000000000000000000000001": "ContractNativeMinter",
		"0x0200000000000000000000000000000000000002": "TxAllowList",
		"0x0200000000000000000000000000000000000003": "FeeConfigManager",
		"0x0300000000000000000000000000000000000000": "RewardManager",
	}
	
	for addr, name := range precompiles {
		genesis.Alloc[addr] = AccountAlloc{
			Balance: "0x0",
			Storage: map[string]string{
				// Set admin role
				fmt.Sprintf("0x%064s", strings.TrimPrefix(adminAddress, "0x")): "0x0000000000000000000000000000000000000000000000000000000000000002",
			},
		}
		fmt.Printf("  Added precompile: %s (%s)\n", name, addr)
	}
}

func calculateStatistics(genesis *CChainGenesis, report *ImportReport) {
	totalSupply := big.NewInt(0)
	balances := make([]HolderInfo, 0, len(genesis.Alloc))
	
	// Calculate total supply and collect balances
	for addr, alloc := range genesis.Alloc {
		balance, _ := new(big.Int).SetString(alloc.Balance, 0)
		if balance.Sign() > 0 {
			totalSupply.Add(totalSupply, balance)
			balances = append(balances, HolderInfo{
				Address: addr,
				Balance: balance,
			})
		}
	}
	
	report.TotalAccounts = len(genesis.Alloc)
	report.TotalSupply = totalSupply
	
	// Sort by balance
	sort.Slice(balances, func(i, j int) bool {
		return balances[i].Balance.Cmp(balances[j].Balance) > 0
	})
	
	// Calculate percentages for top holders
	for i := 0; i < len(balances) && i < 20; i++ {
		percentage := new(big.Float).SetInt(balances[i].Balance)
		percentage.Quo(percentage, new(big.Float).SetInt(totalSupply))
		percentage.Mul(percentage, big.NewFloat(100))
		
		pct, _ := percentage.Float64()
		balances[i].Percentage = pct
		
		// Identify source
		if balances[i].Address == strings.ToLower(cfg.TreasuryAddress) {
			balances[i].Source = "Treasury"
		} else {
			balances[i].Source = "Historic Import"
		}
	}
	
	// Keep top 20
	if len(balances) > 20 {
		report.LargestHolders = balances[:20]
	} else {
		report.LargestHolders = balances
	}
}

func validateGenesis(genesis *CChainGenesis, report *ImportReport) {
	// Check chain ID
	if genesis.Config.ChainID.Int64() != cfg.ChainID {
		report.ValidationErrors = append(report.ValidationErrors,
			fmt.Sprintf("Chain ID mismatch: expected %d, got %d", cfg.ChainID, genesis.Config.ChainID.Int64()))
	}
	
	// Check for zero addresses
	zeroCount := 0
	for addr, alloc := range genesis.Alloc {
		if addr == "0x0000000000000000000000000000000000000000" {
			report.ValidationErrors = append(report.ValidationErrors,
				"Allocation to zero address detected")
		}
		
		balance, _ := new(big.Int).SetString(alloc.Balance, 0)
		if balance.Sign() == 0 {
			zeroCount++
		}
	}
	
	if zeroCount > 0 {
		fmt.Printf("  Warning: %d accounts with zero balance\n", zeroCount)
	}
	
	// Verify treasury
	if treasury, exists := genesis.Alloc[strings.ToLower(cfg.TreasuryAddress)]; exists {
		balance, _ := new(big.Int).SetString(treasury.Balance, 0)
		fmt.Printf("  ✓ Treasury balance: %s wei\n", balance.String())
	} else {
		report.ValidationErrors = append(report.ValidationErrors,
			"Treasury address not found in allocations")
	}
	
	if len(report.ValidationErrors) == 0 {
		fmt.Printf("  ✓ All validations passed\n")
	} else {
		fmt.Printf("  ✗ %d validation errors found\n", len(report.ValidationErrors))
	}
}

func saveGenesis(genesis *CChainGenesis, filename string) error {
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(filename, data, 0644)
}

func saveReport(report *ImportReport, filename string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	
	return ioutil.WriteFile(filename, data, 0644)
}

func printReport(report *ImportReport) {
	fmt.Println("\n=====================================")
	fmt.Println("Import Report")
	fmt.Println("=====================================")
	fmt.Printf("Timestamp: %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Printf("Chain ID: %d\n", report.ChainID)
	fmt.Printf("Total Accounts: %d\n", report.TotalAccounts)
	fmt.Printf("Total Supply: %s wei\n", report.TotalSupply.String())
	
	// Convert to LUX
	luxSupply := new(big.Float).SetInt(report.TotalSupply)
	luxSupply.Quo(luxSupply, big.NewFloat(1e9))
	fmt.Printf("Total Supply: %.9f LUX\n", luxSupply)
	
	fmt.Println("\nImported From:")
	for source, count := range report.ImportedFrom {
		fmt.Printf("  - %s: %d accounts\n", source, count)
	}
	
	fmt.Println("\nTop 10 Holders:")
	for i, holder := range report.LargestHolders {
		if i >= 10 {
			break
		}
		
		// Format balance
		balLux := new(big.Float).SetInt(holder.Balance)
		balLux.Quo(balLux, big.NewFloat(1e9))
		
		fmt.Printf("  %2d. %s: %.9f LUX (%.4f%%) [%s]\n",
			i+1,
			holder.Address[:10]+"..."+holder.Address[len(holder.Address)-6:],
			balLux,
			holder.Percentage,
			holder.Source,
		)
	}
	
	if len(report.DuplicateHandling) > 0 {
		fmt.Printf("\nDuplicate Addresses Handled: %d\n", len(report.DuplicateHandling))
	}
	
	if len(report.ValidationErrors) > 0 {
		fmt.Println("\nValidation Errors:")
		for _, err := range report.ValidationErrors {
			fmt.Printf("  ✗ %s\n", err)
		}
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}