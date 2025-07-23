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

	"github.com/spf13/cobra"
)

var (
	cfg = struct {
		InputFile  string
		OutputDir  string
		Format     string
		MinBalance string
	}{
		OutputDir:  "imported-state",
		Format:     "json",
		MinBalance: "1000000000", // 1 Gwei minimum
	}
)

func init() {
	rootCmd.Flags().StringVar(&cfg.InputFile, "input", "", "Input file (CSV or JSON)")
	rootCmd.Flags().StringVar(&cfg.OutputDir, "output", cfg.OutputDir, "Output directory")
	rootCmd.Flags().StringVar(&cfg.Format, "format", cfg.Format, "Output format (json, csv, both)")
	rootCmd.Flags().StringVar(&cfg.MinBalance, "min-balance", cfg.MinBalance, "Minimum balance to include (wei)")
}

var rootCmd = &cobra.Command{
	Use:   "import-lux-mainnet",
	Short: "Import Lux mainnet 96369 allocations for C-Chain genesis",
	Long:  "Tool to import existing Lux mainnet allocations and prepare them for C-Chain genesis",
	RunE:  runImport,
}

type Allocation struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

func runImport(cmd *cobra.Command, args []string) error {
	// If no input file specified, look for known files
	if cfg.InputFile == "" {
		possibleFiles := []string{
			"configs/mainnet/c-chain-allocations.json",
			"configs/mainnet/allocations.csv",
			"chaindata/lux-mainnet-96369/allocations.json",
			"genesis-export-*/c-chain-allocations.json",
		}
		
		for _, pattern := range possibleFiles {
			matches, _ := filepath.Glob(pattern)
			if len(matches) > 0 {
				cfg.InputFile = matches[0]
				fmt.Printf("Found allocation file: %s\n", cfg.InputFile)
				break
			}
		}
		
		if cfg.InputFile == "" {
			// Create a placeholder with treasury only
			fmt.Println("No existing allocation file found, creating with treasury only")
			return createTreasuryOnly()
		}
	}

	// Parse minimum balance
	minBalance, ok := new(big.Int).SetString(cfg.MinBalance, 10)
	if !ok {
		return fmt.Errorf("invalid minimum balance: %s", cfg.MinBalance)
	}

	// Load allocations
	allocations, err := loadAllocations(cfg.InputFile)
	if err != nil {
		return fmt.Errorf("failed to load allocations: %w", err)
	}

	fmt.Printf("Loaded %d allocations\n", len(allocations))

	// Filter by minimum balance
	filtered := []Allocation{}
	totalBalance := big.NewInt(0)
	
	for _, alloc := range allocations {
		balance, ok := new(big.Int).SetString(alloc.Balance, 10)
		if !ok {
			log.Printf("Invalid balance for %s: %s", alloc.Address, alloc.Balance)
			continue
		}
		
		if balance.Cmp(minBalance) >= 0 {
			filtered = append(filtered, alloc)
			totalBalance.Add(totalBalance, balance)
		}
	}

	// Sort by balance
	sort.Slice(filtered, func(i, j int) bool {
		bi, _ := new(big.Int).SetString(filtered[i].Balance, 10)
		bj, _ := new(big.Int).SetString(filtered[j].Balance, 10)
		return bi.Cmp(bj) > 0
	})

	fmt.Printf("Filtered to %d allocations (total: %s wei)\n", len(filtered), totalBalance.String())

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save results
	timestamp := time.Now().Format("20060102-150405")
	
	// Save C-Chain genesis format
	cchainGenesis := createCChainGenesis(filtered)
	genesisPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("c-chain-genesis-96369-%s.json", timestamp))
	if err := saveJSON(genesisPath, cchainGenesis); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}
	fmt.Printf("Saved C-Chain genesis to: %s\n", genesisPath)

	// Save allocation list
	if cfg.Format == "json" || cfg.Format == "both" {
		jsonPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("allocations-96369-%s.json", timestamp))
		if err := saveJSON(jsonPath, map[string]interface{}{
			"source":        "lux-mainnet-96369",
			"totalAccounts": len(filtered),
			"totalBalance":  totalBalance.String(),
			"allocations":   filtered,
		}); err != nil {
			return fmt.Errorf("failed to save JSON: %w", err)
		}
		fmt.Printf("Saved allocations JSON to: %s\n", jsonPath)
	}

	if cfg.Format == "csv" || cfg.Format == "both" {
		csvPath := filepath.Join(cfg.OutputDir, fmt.Sprintf("allocations-96369-%s.csv", timestamp))
		if err := saveCSV(csvPath, filtered); err != nil {
			return fmt.Errorf("failed to save CSV: %w", err)
		}
		fmt.Printf("Saved allocations CSV to: %s\n", csvPath)
	}

	// Save summary
	summaryPath := filepath.Join(cfg.OutputDir, "import-summary.json")
	if err := saveJSON(summaryPath, map[string]interface{}{
		"timestamp":     time.Now().Format(time.RFC3339),
		"source":        cfg.InputFile,
		"totalAccounts": len(filtered),
		"totalBalance":  totalBalance.String(),
		"minBalance":    cfg.MinBalance,
		"topHolders":    getTopHolders(filtered, 20),
	}); err != nil {
		return fmt.Errorf("failed to save summary: %w", err)
	}

	return nil
}

func createTreasuryOnly() error {
	// Treasury address and balance
	treasury := Allocation{
		Address: "0x9011e888251ab053b7bd1cdb598db4f9ded94714",
		Balance: "1994739905397278683064838288203",
	}

	allocations := []Allocation{treasury}

	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save C-Chain genesis
	cchainGenesis := createCChainGenesis(allocations)
	genesisPath := filepath.Join(cfg.OutputDir, "c-chain-genesis-treasury-only.json")
	if err := saveJSON(genesisPath, cchainGenesis); err != nil {
		return fmt.Errorf("failed to save genesis: %w", err)
	}
	fmt.Printf("Created treasury-only C-Chain genesis: %s\n", genesisPath)

	return nil
}

func loadAllocations(filename string) ([]Allocation, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	
	if ext == ".csv" {
		return loadCSV(filename)
	} else if ext == ".json" {
		return loadJSON(filename)
	}
	
	return nil, fmt.Errorf("unsupported file format: %s", ext)
}

func loadCSV(filename string) ([]Allocation, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comment = '#'
	reader.FieldsPerRecord = -1

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	allocations := []Allocation{}
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		
		if len(record) < 2 {
			continue
		}
		
		allocations = append(allocations, Allocation{
			Address: strings.ToLower(record[0]),
			Balance: record[1],
		})
	}

	return allocations, nil
}

func loadJSON(filename string) ([]Allocation, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Try different JSON formats
	var allocations []Allocation
	
	// Try as array
	if err := json.Unmarshal(data, &allocations); err == nil {
		return allocations, nil
	}

	// Try as map
	var allocMap map[string]string
	if err := json.Unmarshal(data, &allocMap); err == nil {
		allocations = []Allocation{}
		for addr, balance := range allocMap {
			allocations = append(allocations, Allocation{
				Address: strings.ToLower(addr),
				Balance: balance,
			})
		}
		return allocations, nil
	}

	// Try as object with allocations field
	var obj struct {
		Allocations []Allocation `json:"allocations"`
		Alloc       map[string]struct {
			Balance string `json:"balance"`
		} `json:"alloc"`
	}
	
	if err := json.Unmarshal(data, &obj); err == nil {
		if len(obj.Allocations) > 0 {
			return obj.Allocations, nil
		}
		
		if len(obj.Alloc) > 0 {
			allocations = []Allocation{}
			for addr, data := range obj.Alloc {
				allocations = append(allocations, Allocation{
					Address: strings.ToLower(addr),
					Balance: data.Balance,
				})
			}
			return allocations, nil
		}
	}

	return nil, fmt.Errorf("unable to parse allocation file")
}

func createCChainGenesis(allocations []Allocation) map[string]interface{} {
	alloc := make(map[string]interface{})
	
	// Add allocations
	for _, allocation := range allocations {
		alloc[allocation.Address] = map[string]string{
			"balance": allocation.Balance,
		}
	}
	
	// Add precompiles
	precompiles := map[string]struct {
		balance string
		code    string
		storage map[string]string
	}{
		"0x0000000000000000000000000000000000000400": {
			balance: "0x0",
			storage: map[string]string{
				"0x0000000000000000000000001000000000000000000000000000000000000000": "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
		},
		"0x0000000000000000000000000000000000000401": {
			balance: "0x0",
			storage: map[string]string{
				"0x0000000000000000000000001000000000000000000000000000000000000000": "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
		},
		"0x0000000000000000000000000000000000000402": {
			balance: "0x0",
			storage: map[string]string{
				"0x0000000000000000000000001000000000000000000000000000000000000000": "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
		},
		"0x0000000000000000000000000000000000000403": {
			balance: "0x0",
			storage: map[string]string{
				"0x0000000000000000000000001000000000000000000000000000000000000000": "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
		},
	}
	
	for addr, data := range precompiles {
		entry := map[string]interface{}{
			"balance": data.balance,
		}
		if data.storage != nil {
			entry["storage"] = data.storage
		}
		alloc[addr] = entry
	}
	
	// Create genesis
	return map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             96369,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip150Hash":          "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
			"berlinBlock":         0,
			"londonBlock":         0,
			"allowFeeRecipients":  true,
			"feeConfig": map[string]interface{}{
				"gasLimit":                 15000000,
				"minBaseFee":               25000000000,
				"targetGas":                15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost":          0,
				"maxBlockGasCost":          1000000,
				"targetBlockRate":          2,
				"blockGasCostStep":         200000,
			},
		},
		"alloc":      alloc,
		"nonce":      "0x0",
		"timestamp":  "0x0",
		"extraData":  "0x00",
		"gasLimit":   "0x989680",
		"difficulty": "0x0",
		"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase":   "0x0000000000000000000000000000000000000000",
		"number":     "0x0",
		"gasUsed":    "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"baseFeePerGas": "0x0",
	}
}

func saveJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, jsonData, 0644)
}

func saveCSV(path string, allocations []Allocation) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"address", "balance_wei"}); err != nil {
		return err
	}

	// Write data
	for _, alloc := range allocations {
		if err := writer.Write([]string{alloc.Address, alloc.Balance}); err != nil {
			return err
		}
	}

	return nil
}

func getTopHolders(allocations []Allocation, limit int) []map[string]string {
	holders := []map[string]string{}
	
	for i := 0; i < len(allocations) && i < limit; i++ {
		balance, _ := new(big.Int).SetString(allocations[i].Balance, 10)
		
		// Convert to human readable
		lux := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			new(big.Float).SetInt(big.NewInt(1e9)),
		)
		
		holders = append(holders, map[string]string{
			"rank":       fmt.Sprintf("%d", i+1),
			"address":    allocations[i].Address,
			"balance":    allocations[i].Balance,
			"balanceLUX": fmt.Sprintf("%.9f", lux),
		})
	}
	
	return holders
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}