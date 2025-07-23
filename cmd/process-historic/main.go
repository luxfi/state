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
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

type HistoricData struct {
	// Chain allocations
	Lux7777Allocations    map[string]*big.Int
	Lux96369Allocations   map[string]*big.Int
	Zoo200200Allocations  map[string]*big.Int
	
	// Cross-chain allocations
	ZooBSCAllocations     map[string]*big.Int
	LuxETHAllocations     map[string]*big.Int
	
	// Aggregated data
	TotalAllocations      map[string]*AllocationSummary
}

type AllocationSummary struct {
	Address         string
	Lux7777        *big.Int
	Lux96369       *big.Int
	Zoo200200      *big.Int
	ZooBSC         *big.Int
	LuxETH         *big.Int
	TotalLUX       *big.Int
	TotalZOO       *big.Int
	Chains         []string
	Rationale      string
}

type GenesisOutput struct {
	Network        string
	PChainGenesis  interface{}
	CChainGenesis  interface{}
	XChainGenesis  interface{}
	Summary        *ProcessingSummary
}

type ProcessingSummary struct {
	Timestamp           time.Time
	TotalAccounts      int
	TotalLUXSupply     *big.Int
	TotalZOOSupply     *big.Int
	ChainBreakdown     map[string]*ChainSummary
	MigrationPlan      []MigrationStep
	ValidationResults  []ValidationResult
}

type ChainSummary struct {
	ChainID        int
	Name           string
	Accounts       int
	TotalSupply    *big.Int
	TopHolders     []TopHolder
}

type TopHolder struct {
	Address    string
	Balance    *big.Int
	Percentage float64
}

type MigrationStep struct {
	From      string
	To        string
	Amount    *big.Int
	Accounts  int
	Rationale string
}

type ValidationResult struct {
	Check   string
	Status  string
	Details string
}

var (
	outputDir      string
	generateCSV    bool
	validateOnly   bool
	
	rootCmd = &cobra.Command{
		Use:   "process-historic",
		Short: "Process historic blockchain data for genesis generation",
		Long:  `Processes data from Lux 7777, 96369, Zoo 200200, and cross-chain deployments`,
		RunE:  runProcess,
	}
)

func init() {
	rootCmd.Flags().StringVar(&outputDir, "output", "genesis-output", "Output directory")
	rootCmd.Flags().BoolVar(&generateCSV, "csv", true, "Generate CSV summary")
	rootCmd.Flags().BoolVar(&validateOnly, "validate", false, "Validate only, don't generate")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runProcess(cmd *cobra.Command, args []string) error {
	fmt.Println("Processing Historic Blockchain Data")
	fmt.Println("==================================")
	
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Load all historic data
	data, err := loadHistoricData()
	if err != nil {
		return fmt.Errorf("failed to load historic data: %w", err)
	}
	
	// Process and aggregate allocations
	fmt.Println("\nAggregating allocations across chains...")
	aggregated := aggregateAllocations(data)
	
	// Generate summary
	summary := generateSummary(aggregated, data)
	
	// Print summary
	printSummary(summary)
	
	// Generate CSV if requested
	if generateCSV {
		csvPath := filepath.Join(outputDir, "allocation-summary.csv")
		if err := generateAllocationCSV(aggregated, csvPath); err != nil {
			return fmt.Errorf("failed to generate CSV: %w", err)
		}
		fmt.Printf("\nCSV summary saved to: %s\n", csvPath)
	}
	
	// Validate allocations
	validationResults := validateAllocations(aggregated, data)
	printValidationResults(validationResults)
	
	if validateOnly {
		return nil
	}
	
	// Generate genesis files for each network
	fmt.Println("\nGenerating genesis configurations...")
	
	// Generate mainnet genesis
	mainnetOutput, err := generateNetworkGenesis("mainnet", aggregated, data)
	if err != nil {
		return fmt.Errorf("failed to generate mainnet genesis: %w", err)
	}
	
	// Save genesis files
	if err := saveGenesisOutput(mainnetOutput); err != nil {
		return fmt.Errorf("failed to save genesis output: %w", err)
	}
	
	fmt.Println("\nGenesis generation complete!")
	fmt.Printf("Output directory: %s\n", outputDir)
	
	return nil
}

func loadHistoricData() (*HistoricData, error) {
	data := &HistoricData{
		Lux7777Allocations:   make(map[string]*big.Int),
		Lux96369Allocations:  make(map[string]*big.Int),
		Zoo200200Allocations: make(map[string]*big.Int),
		ZooBSCAllocations:    make(map[string]*big.Int),
		LuxETHAllocations:    make(map[string]*big.Int),
		TotalAllocations:     make(map[string]*AllocationSummary),
	}
	
	// Load Lux 7777 airdrop data
	fmt.Println("Loading Lux 7777 airdrop data...")
	if err := loadLux7777Airdrop(data); err != nil {
		return nil, fmt.Errorf("failed to load Lux 7777 data: %w", err)
	}
	
	// Load Lux 96369 current state
	fmt.Println("Loading Lux 96369 current state...")
	if err := loadLux96369State(data); err != nil {
		return nil, fmt.Errorf("failed to load Lux 96369 data: %w", err)
	}
	
	// Load Zoo 200200 allocations
	fmt.Println("Loading Zoo 200200 allocations...")
	if err := loadZoo200200Allocations(data); err != nil {
		return nil, fmt.Errorf("failed to load Zoo 200200 data: %w", err)
	}
	
	// Load cross-chain data (if available)
	fmt.Println("Loading cross-chain data...")
	loadCrossChainData(data)
	
	return data, nil
}

func loadLux7777Airdrop(data *HistoricData) error {
	// Load the CSV file
	csvPath := "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet-no-treasury.csv"
	file, err := os.Open(csvPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	reader.Comment = '#'  // Skip comment lines
	reader.FieldsPerRecord = -1  // Variable number of fields
	
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 4 {
			continue
		}
		
		address := strings.ToLower(records[i][1])
		balanceStr := records[i][3] // Use wei amount
		
		balance, ok := new(big.Int).SetString(balanceStr, 10)
		if !ok {
			continue
		}
		
		data.Lux7777Allocations[address] = balance
	}
	
	fmt.Printf("  Loaded %d Lux 7777 allocations\n", len(data.Lux7777Allocations))
	return nil
}

func loadLux96369State(data *HistoricData) error {
	// Load from existing genesis if available
	genesisPath := "chaindata/configs/lux-mainnet-96369/genesis.json"
	if _, err := os.Stat(genesisPath); err == nil {
		// Parse existing genesis
		genesisData, err := ioutil.ReadFile(genesisPath)
		if err != nil {
			return err
		}
		
		var genesisConfig map[string]interface{}
		if err := json.Unmarshal(genesisData, &genesisConfig); err != nil {
			return err
		}
		
		// Extract allocations from C-Chain genesis
		if cchainStr, ok := genesisConfig["cChainGenesis"].(string); ok {
			var cchain map[string]interface{}
			if err := json.Unmarshal([]byte(cchainStr), &cchain); err == nil {
				if alloc, ok := cchain["alloc"].(map[string]interface{}); ok {
					for addr, allocData := range alloc {
						if allocMap, ok := allocData.(map[string]interface{}); ok {
							if balanceStr, ok := allocMap["balance"].(string); ok {
								balance, _ := new(big.Int).SetString(balanceStr, 10)
								data.Lux96369Allocations[strings.ToLower(addr)] = balance
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("  Loaded %d Lux 96369 allocations\n", len(data.Lux96369Allocations))
	return nil
}

func loadZoo200200Allocations(data *HistoricData) error {
	// Try to load Zoo genesis data
	zooGenesisPath := "chaindata/configs/zoo-mainnet-200200/genesis.json"
	if _, err := os.Stat(zooGenesisPath); err == nil {
		genesisData, err := ioutil.ReadFile(zooGenesisPath)
		if err != nil {
			return err
		}
		
		var genesisConfig map[string]interface{}
		if err := json.Unmarshal(genesisData, &genesisConfig); err != nil {
			return err
		}
		
		// Extract allocations
		if alloc, ok := genesisConfig["alloc"].(map[string]interface{}); ok {
			for addr, allocData := range alloc {
				if allocMap, ok := allocData.(map[string]interface{}); ok {
					if balanceStr, ok := allocMap["balance"].(string); ok {
						balance, _ := new(big.Int).SetString(balanceStr, 10)
						data.Zoo200200Allocations[strings.ToLower(addr)] = balance
					}
				}
			}
		}
	}
	
	// Also check for Zoo X-Chain allocations
	zooXChainPath := "exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.csv"
	if _, err := os.Stat(zooXChainPath); err == nil {
		file, err := os.Open(zooXChainPath)
		if err == nil {
			defer file.Close()
			reader := csv.NewReader(file)
			records, _ := reader.ReadAll()
			
			for i := 1; i < len(records); i++ {
				if len(records[i]) >= 3 {
					address := strings.ToLower(records[i][0])
					// Column 2 has zoo_amount
					if records[i][2] != "" {
						if amount, ok := new(big.Int).SetString(records[i][2], 10); ok {
							// Store as separate allocation or merge
							if existing, ok := data.Zoo200200Allocations[address]; ok && existing != nil {
								data.Zoo200200Allocations[address] = new(big.Int).Add(existing, amount)
							} else {
								data.Zoo200200Allocations[address] = new(big.Int).Set(amount)
							}
						}
					}
				}
			}
		}
	}
	
	fmt.Printf("  Loaded %d Zoo 200200 allocations\n", len(data.Zoo200200Allocations))
	return nil
}

func loadCrossChainData(data *HistoricData) {
	// This would load data from:
	// - Zoo on BSC (via bridge records or snapshot)
	// - Lux on ETH (via bridge records or snapshot)
	// For now, we'll create placeholder logic
	
	// Example: Add some test cross-chain data
	// In production, this would read from actual bridge records or chain snapshots
	
	fmt.Println("  Cross-chain data loading not yet implemented")
}

func aggregateAllocations(data *HistoricData) map[string]*AllocationSummary {
	aggregated := make(map[string]*AllocationSummary)
	
	// Helper function to ensure allocation exists
	getOrCreateAllocation := func(address string) *AllocationSummary {
		addr := strings.ToLower(address)
		if _, exists := aggregated[addr]; !exists {
			aggregated[addr] = &AllocationSummary{
				Address:   addr,
				Lux7777:   big.NewInt(0),
				Lux96369:  big.NewInt(0),
				Zoo200200: big.NewInt(0),
				ZooBSC:    big.NewInt(0),
				LuxETH:    big.NewInt(0),
				TotalLUX:  big.NewInt(0),
				TotalZOO:  big.NewInt(0),
				Chains:    []string{},
			}
		}
		return aggregated[addr]
	}
	
	// Process Lux 7777
	for addr, balance := range data.Lux7777Allocations {
		alloc := getOrCreateAllocation(addr)
		alloc.Lux7777 = balance
		alloc.TotalLUX = new(big.Int).Add(alloc.TotalLUX, balance)
		alloc.Chains = append(alloc.Chains, "Lux-7777")
		alloc.Rationale = "Original Lux Network holder"
	}
	
	// Process Lux 96369
	for addr, balance := range data.Lux96369Allocations {
		alloc := getOrCreateAllocation(addr)
		alloc.Lux96369 = balance
		// Don't double count if already in 7777
		if alloc.Lux7777.Cmp(big.NewInt(0)) == 0 {
			alloc.TotalLUX = new(big.Int).Add(alloc.TotalLUX, balance)
			alloc.Rationale = "Current Lux Network participant"
		}
		if !contains(alloc.Chains, "Lux-96369") {
			alloc.Chains = append(alloc.Chains, "Lux-96369")
		}
	}
	
	// Process Zoo 200200
	for addr, balance := range data.Zoo200200Allocations {
		alloc := getOrCreateAllocation(addr)
		alloc.Zoo200200 = balance
		alloc.TotalZOO = new(big.Int).Add(alloc.TotalZOO, balance)
		if !contains(alloc.Chains, "Zoo-200200") {
			alloc.Chains = append(alloc.Chains, "Zoo-200200")
		}
		if alloc.Rationale == "" {
			alloc.Rationale = "Zoo Network participant"
		}
	}
	
	// Process cross-chain data
	for addr, balance := range data.ZooBSCAllocations {
		alloc := getOrCreateAllocation(addr)
		alloc.ZooBSC = balance
		alloc.TotalZOO = new(big.Int).Add(alloc.TotalZOO, balance)
		if !contains(alloc.Chains, "Zoo-BSC") {
			alloc.Chains = append(alloc.Chains, "Zoo-BSC")
		}
	}
	
	for addr, balance := range data.LuxETHAllocations {
		alloc := getOrCreateAllocation(addr)
		alloc.LuxETH = balance
		alloc.TotalLUX = new(big.Int).Add(alloc.TotalLUX, balance)
		if !contains(alloc.Chains, "Lux-ETH") {
			alloc.Chains = append(alloc.Chains, "Lux-ETH")
		}
	}
	
	return aggregated
}

func generateSummary(aggregated map[string]*AllocationSummary, data *HistoricData) *ProcessingSummary {
	summary := &ProcessingSummary{
		Timestamp:      time.Now(),
		TotalAccounts:  len(aggregated),
		TotalLUXSupply: big.NewInt(0),
		TotalZOOSupply: big.NewInt(0),
		ChainBreakdown: make(map[string]*ChainSummary),
		MigrationPlan:  []MigrationStep{},
	}
	
	// Calculate totals
	for _, alloc := range aggregated {
		summary.TotalLUXSupply = new(big.Int).Add(summary.TotalLUXSupply, alloc.TotalLUX)
		summary.TotalZOOSupply = new(big.Int).Add(summary.TotalZOOSupply, alloc.TotalZOO)
	}
	
	// Generate chain breakdowns
	summary.ChainBreakdown["Lux-7777"] = generateChainSummary("Lux-7777", 7777, data.Lux7777Allocations)
	summary.ChainBreakdown["Lux-96369"] = generateChainSummary("Lux-96369", 96369, data.Lux96369Allocations)
	summary.ChainBreakdown["Zoo-200200"] = generateChainSummary("Zoo-200200", 200200, data.Zoo200200Allocations)
	
	// Define migration plan
	summary.MigrationPlan = []MigrationStep{
		{
			From:      "Lux-7777",
			To:        "Lux-96369-C-Chain",
			Amount:    summary.ChainBreakdown["Lux-7777"].TotalSupply,
			Accounts:  summary.ChainBreakdown["Lux-7777"].Accounts,
			Rationale: "Migrate all Lux 7777 balances to new C-Chain",
		},
		{
			From:      "Zoo-200200",
			To:        "Zoo-L2",
			Amount:    summary.ChainBreakdown["Zoo-200200"].TotalSupply,
			Accounts:  summary.ChainBreakdown["Zoo-200200"].Accounts,
			Rationale: "Maintain Zoo as L2 subnet on new network",
		},
	}
	
	return summary
}

func generateChainSummary(name string, chainID int, allocations map[string]*big.Int) *ChainSummary {
	summary := &ChainSummary{
		Name:        name,
		ChainID:     chainID,
		Accounts:    len(allocations),
		TotalSupply: big.NewInt(0),
		TopHolders:  []TopHolder{},
	}
	
	// Calculate total and find top holders
	type holder struct {
		address string
		balance *big.Int
	}
	
	holders := []holder{}
	for addr, balance := range allocations {
		summary.TotalSupply = new(big.Int).Add(summary.TotalSupply, balance)
		holders = append(holders, holder{addr, balance})
	}
	
	// Sort by balance (simple bubble sort for top 10)
	for i := 0; i < len(holders) && i < 10; i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].balance.Cmp(holders[i].balance) > 0 {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	
	// Add top holders
	for i := 0; i < len(holders) && i < 10; i++ {
		percentage := 0.0
		if summary.TotalSupply.Cmp(big.NewInt(0)) > 0 {
			balFloat := new(big.Float).SetInt(holders[i].balance)
			totalFloat := new(big.Float).SetInt(summary.TotalSupply)
			percentFloat := new(big.Float).Quo(balFloat, totalFloat)
			percentFloat.Mul(percentFloat, big.NewFloat(100))
			percentage, _ = percentFloat.Float64()
		}
		
		summary.TopHolders = append(summary.TopHolders, TopHolder{
			Address:    holders[i].address,
			Balance:    holders[i].balance,
			Percentage: percentage,
		})
	}
	
	return summary
}

func validateAllocations(aggregated map[string]*AllocationSummary, data *HistoricData) []ValidationResult {
	results := []ValidationResult{}
	
	// Check 1: Total supply conservation
	expectedLux7777 := new(big.Int)
	expectedLux7777.SetString("2000000000000000000000000000000", 10) // 2T LUX
	
	actualLux7777 := big.NewInt(0)
	for _, balance := range data.Lux7777Allocations {
		actualLux7777 = new(big.Int).Add(actualLux7777, balance)
	}
	
	if actualLux7777.Cmp(expectedLux7777) == 0 {
		results = append(results, ValidationResult{
			Check:   "Lux 7777 Total Supply",
			Status:  "PASS",
			Details: fmt.Sprintf("Expected: %s, Actual: %s", expectedLux7777.String(), actualLux7777.String()),
		})
	} else {
		results = append(results, ValidationResult{
			Check:   "Lux 7777 Total Supply",
			Status:  "FAIL",
			Details: fmt.Sprintf("Expected: %s, Actual: %s", expectedLux7777.String(), actualLux7777.String()),
		})
	}
	
	// Check 2: No negative balances
	negativeCount := 0
	for _, alloc := range aggregated {
		if alloc.TotalLUX.Sign() < 0 || alloc.TotalZOO.Sign() < 0 {
			negativeCount++
		}
	}
	
	if negativeCount == 0 {
		results = append(results, ValidationResult{
			Check:   "Negative Balances",
			Status:  "PASS",
			Details: "No negative balances found",
		})
	} else {
		results = append(results, ValidationResult{
			Check:   "Negative Balances",
			Status:  "FAIL",
			Details: fmt.Sprintf("%d accounts with negative balances", negativeCount),
		})
	}
	
	// Check 3: Address format validation
	invalidAddresses := 0
	for addr := range aggregated {
		if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
			invalidAddresses++
		}
	}
	
	if invalidAddresses == 0 {
		results = append(results, ValidationResult{
			Check:   "Address Format",
			Status:  "PASS",
			Details: "All addresses are valid Ethereum format",
		})
	} else {
		results = append(results, ValidationResult{
			Check:   "Address Format",
			Status:  "FAIL",
			Details: fmt.Sprintf("%d invalid addresses found", invalidAddresses),
		})
	}
	
	return results
}

func generateAllocationCSV(aggregated map[string]*AllocationSummary, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write header
	header := []string{
		"address",
		"lux_7777_wei",
		"lux_96369_wei", 
		"zoo_200200_wei",
		"zoo_bsc_wei",
		"lux_eth_wei",
		"total_lux_wei",
		"total_zoo_wei",
		"chains",
		"rationale",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	
	// Write data sorted by total LUX
	type entry struct {
		addr  string
		alloc *AllocationSummary
	}
	
	entries := []entry{}
	for addr, alloc := range aggregated {
		entries = append(entries, entry{addr, alloc})
	}
	
	// Sort by total LUX descending
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].alloc.TotalLUX.Cmp(entries[i].alloc.TotalLUX) > 0 {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Write records
	for _, e := range entries {
		record := []string{
			e.addr,
			e.alloc.Lux7777.String(),
			e.alloc.Lux96369.String(),
			e.alloc.Zoo200200.String(),
			e.alloc.ZooBSC.String(),
			e.alloc.LuxETH.String(),
			e.alloc.TotalLUX.String(),
			e.alloc.TotalZOO.String(),
			strings.Join(e.alloc.Chains, ";"),
			e.alloc.Rationale,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

func generateNetworkGenesis(network string, aggregated map[string]*AllocationSummary, data *HistoricData) (*GenesisOutput, error) {
	output := &GenesisOutput{
		Network: network,
	}
	
	// Create genesis builder
	builder, err := genesis.NewBuilder(network)
	if err != nil {
		return nil, err
	}
	
	// Add allocations based on aggregated data
	for addr, alloc := range aggregated {
		// For mainnet, we combine Lux 7777 and 96369 allocations
		totalLux := alloc.TotalLUX
		
		// Skip zero balances
		if totalLux.Sign() == 0 {
			continue
		}
		
		// Add to C-Chain
		if err := builder.AddAllocation(addr, totalLux); err != nil {
			log.Printf("Warning: failed to add allocation for %s: %v", addr, err)
		}
	}
	
	// Load validators if they exist
	validatorsFile := fmt.Sprintf("configs/%s-validators.json", network)
	if _, err := os.Stat(validatorsFile); err == nil {
		// Import validators using existing logic
		fmt.Printf("Loading validators from %s\n", validatorsFile)
		// This would use the existing validator loading logic
	}
	
	// Build genesis
	genesisData, err := builder.Build()
	if err != nil {
		return nil, err
	}
	
	output.CChainGenesis = genesisData
	output.Summary = generateNetworkSummary(network, aggregated)
	
	return output, nil
}

func generateNetworkSummary(network string, aggregated map[string]*AllocationSummary) *ProcessingSummary {
	// Generate network-specific summary
	summary := &ProcessingSummary{
		Timestamp:     time.Now(),
		TotalAccounts: 0,
		TotalLUXSupply: big.NewInt(0),
		TotalZOOSupply: big.NewInt(0),
	}
	
	for _, alloc := range aggregated {
		if alloc.TotalLUX.Sign() > 0 {
			summary.TotalAccounts++
			summary.TotalLUXSupply = new(big.Int).Add(summary.TotalLUXSupply, alloc.TotalLUX)
		}
		if alloc.TotalZOO.Sign() > 0 {
			summary.TotalZOOSupply = new(big.Int).Add(summary.TotalZOOSupply, alloc.TotalZOO)
		}
	}
	
	return summary
}

func saveGenesisOutput(output *GenesisOutput) error {
	// Create network directory
	networkDir := filepath.Join(outputDir, output.Network)
	if err := os.MkdirAll(networkDir, 0755); err != nil {
		return err
	}
	
	// Save C-Chain genesis
	cchainPath := filepath.Join(networkDir, "c-chain-genesis.json")
	cchainData, err := json.MarshalIndent(output.CChainGenesis, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(cchainPath, cchainData, 0644); err != nil {
		return err
	}
	
	// Save summary
	summaryPath := filepath.Join(networkDir, "genesis-summary.json")
	summaryData, err := json.MarshalIndent(output.Summary, "", "  ")
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(summaryPath, summaryData, 0644); err != nil {
		return err
	}
	
	fmt.Printf("Saved genesis files to %s\n", networkDir)
	return nil
}

func printSummary(summary *ProcessingSummary) {
	fmt.Println("\nProcessing Summary")
	fmt.Println("==================")
	fmt.Printf("Timestamp: %s\n", summary.Timestamp.Format(time.RFC3339))
	fmt.Printf("Total Accounts: %d\n", summary.TotalAccounts)
	fmt.Printf("Total LUX Supply: %s wei\n", summary.TotalLUXSupply.String())
	fmt.Printf("Total ZOO Supply: %s wei\n", summary.TotalZOOSupply.String())
	
	fmt.Println("\nChain Breakdown:")
	for name, chain := range summary.ChainBreakdown {
		fmt.Printf("\n%s (Chain ID: %d)\n", name, chain.ChainID)
		fmt.Printf("  Accounts: %d\n", chain.Accounts)
		fmt.Printf("  Total Supply: %s wei\n", chain.TotalSupply.String())
		fmt.Printf("  Top Holders:\n")
		for i, holder := range chain.TopHolders {
			if i >= 5 {
				break
			}
			fmt.Printf("    %d. %s: %.4f%%\n", i+1, holder.Address, holder.Percentage)
		}
	}
	
	fmt.Println("\nMigration Plan:")
	for _, step := range summary.MigrationPlan {
		fmt.Printf("  %s → %s\n", step.From, step.To)
		fmt.Printf("    Amount: %s wei\n", step.Amount.String())
		fmt.Printf("    Accounts: %d\n", step.Accounts)
		fmt.Printf("    Rationale: %s\n", step.Rationale)
	}
}

func printValidationResults(results []ValidationResult) {
	fmt.Println("\nValidation Results")
	fmt.Println("==================")
	
	passed := 0
	for _, result := range results {
		status := "✓"
		if result.Status != "PASS" {
			status = "✗"
		} else {
			passed++
		}
		fmt.Printf("%s %s: %s\n", status, result.Check, result.Details)
	}
	
	fmt.Printf("\nValidation Summary: %d/%d passed\n", passed, len(results))
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func formatLUX(wei *big.Int) string {
	lux := new(big.Float).SetInt(wei)
	lux = lux.Quo(lux, big.NewFloat(1e9))
	return fmt.Sprintf("%.9f LUX", lux)
}