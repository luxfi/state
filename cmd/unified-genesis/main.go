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

	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/genesis/pkg/genesis/validator"
)

type UnifiedGenesisConfig struct {
	Network           string
	OutputDir         string
	Lux7777CSV       string
	Zoo200200Genesis  string
	ValidatorsFile    string
	TreasuryAddress   string
	IncludeTreasury   bool
	GeneratePChain    bool
	GenerateCChain    bool
	GenerateXChain    bool
}

type GenesisReport struct {
	Generated       time.Time                `json:"generated"`
	Network         string                   `json:"network"`
	ChainID         int                      `json:"chainId"`
	
	// Chain summaries
	PChain          *ChainReport             `json:"pChain"`
	CChain          *ChainReport             `json:"cChain"`
	XChain          *ChainReport             `json:"xChain"`
	
	// Migration details
	MigrationPlan   []MigrationRecord        `json:"migrationPlan"`
	
	// Validation
	Validations     []ValidationCheck        `json:"validations"`
	
	// Files generated
	FilesGenerated  []string                 `json:"filesGenerated"`
}

type ChainReport struct {
	Type           string                    `json:"type"`
	TotalAccounts  int                       `json:"totalAccounts"`
	TotalSupply    string                    `json:"totalSupply"`
	TopHolders     []HolderInfo              `json:"topHolders"`
	Validators     []ValidatorInfo           `json:"validators,omitempty"`
	
	// Chain-specific fields
	ContractCount  int                       `json:"contractCount,omitempty"`
	InitialStakers int                       `json:"initialStakers,omitempty"`
}

type HolderInfo struct {
	Rank       int    `json:"rank"`
	Address    string `json:"address"`
	Balance    string `json:"balance"`
	Percentage string `json:"percentage"`
	Source     string `json:"source"`
}

type ValidatorInfo struct {
	NodeID         string `json:"nodeId"`
	Weight         string `json:"weight"`
	DelegationFee  string `json:"delegationFee"`
}

type MigrationRecord struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Accounts    int    `json:"accounts"`
	Amount      string `json:"amount"`
	Rationale   string `json:"rationale"`
}

type ValidationCheck struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

var (
	cfg = &UnifiedGenesisConfig{}
	
	rootCmd = &cobra.Command{
		Use:   "unified-genesis",
		Short: "Generate unified genesis for all Lux chains",
		Long:  `Processes historic data and generates P-Chain, C-Chain, and X-Chain genesis files`,
		RunE:  runUnifiedGenesis,
	}
)

func init() {
	rootCmd.Flags().StringVar(&cfg.Network, "network", "mainnet", "Network (mainnet/testnet)")
	rootCmd.Flags().StringVar(&cfg.OutputDir, "output", "unified-genesis-output", "Output directory")
	rootCmd.Flags().StringVar(&cfg.Lux7777CSV, "lux7777", "chaindata/lux-genesis-7777/7777-airdrop-96369-mainnet-no-treasury.csv", "Lux 7777 airdrop CSV")
	rootCmd.Flags().StringVar(&cfg.Zoo200200Genesis, "zoo", "exports/genesis-analysis-20250722-060502/zoo_xchain_genesis_allocations.csv", "Zoo allocations CSV")
	rootCmd.Flags().StringVar(&cfg.ValidatorsFile, "validators", "", "Validators JSON file")
	rootCmd.Flags().StringVar(&cfg.TreasuryAddress, "treasury", "0x9011e888251ab053b7bd1cdb598db4f9ded94714", "Treasury address")
	rootCmd.Flags().BoolVar(&cfg.IncludeTreasury, "include-treasury", true, "Include treasury in genesis")
	rootCmd.Flags().BoolVar(&cfg.GeneratePChain, "p-chain", true, "Generate P-Chain genesis")
	rootCmd.Flags().BoolVar(&cfg.GenerateCChain, "c-chain", true, "Generate C-Chain genesis")
	rootCmd.Flags().BoolVar(&cfg.GenerateXChain, "x-chain", true, "Generate X-Chain genesis")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runUnifiedGenesis(cmd *cobra.Command, args []string) error {
	fmt.Println("Unified Genesis Generation")
	fmt.Println("==========================")
	fmt.Printf("Network: %s\n", cfg.Network)
	fmt.Printf("Output: %s\n", cfg.OutputDir)
	
	// Create output directory
	if err := os.MkdirAll(cfg.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	// Initialize report
	report := &GenesisReport{
		Generated:      time.Now(),
		Network:        cfg.Network,
		ChainID:        getChainID(cfg.Network),
		MigrationPlan:  []MigrationRecord{},
		Validations:    []ValidationCheck{},
		FilesGenerated: []string{},
	}
	
	// Load all allocations
	fmt.Println("\n1. Loading Historic Data...")
	allocations, err := loadAllAllocations()
	if err != nil {
		return fmt.Errorf("failed to load allocations: %w", err)
	}
	
	// Load validators if specified
	var validators []*validator.ValidatorInfo
	if cfg.ValidatorsFile != "" {
		fmt.Printf("   Loading validators from %s...\n", cfg.ValidatorsFile)
		validators, err = loadValidators(cfg.ValidatorsFile)
		if err != nil {
			return fmt.Errorf("failed to load validators: %w", err)
		}
		fmt.Printf("   Loaded %d validators\n", len(validators))
	} else {
		// Use default validators file if exists
		defaultFile := fmt.Sprintf("configs/%s-validators.json", cfg.Network)
		if _, err := os.Stat(defaultFile); err == nil {
			cfg.ValidatorsFile = defaultFile
			validators, _ = loadValidators(defaultFile)
			fmt.Printf("   Using default validators from %s (%d validators)\n", defaultFile, len(validators))
		}
	}
	
	// Generate P-Chain genesis
	if cfg.GeneratePChain {
		fmt.Println("\n2. Generating P-Chain Genesis...")
		pChainReport, err := generatePChainGenesis(allocations, validators, report)
		if err != nil {
			return fmt.Errorf("failed to generate P-Chain genesis: %w", err)
		}
		report.PChain = pChainReport
	}
	
	// Generate C-Chain genesis
	if cfg.GenerateCChain {
		fmt.Println("\n3. Generating C-Chain Genesis...")
		cChainReport, err := generateCChainGenesis(allocations, report)
		if err != nil {
			return fmt.Errorf("failed to generate C-Chain genesis: %w", err)
		}
		report.CChain = cChainReport
	}
	
	// Generate X-Chain genesis
	if cfg.GenerateXChain {
		fmt.Println("\n4. Generating X-Chain Genesis...")
		xChainReport, err := generateXChainGenesis(allocations, report)
		if err != nil {
			return fmt.Errorf("failed to generate X-Chain genesis: %w", err)
		}
		report.XChain = xChainReport
	}
	
	// Define migration plan
	fmt.Println("\n5. Creating Migration Plan...")
	report.MigrationPlan = createMigrationPlan(allocations)
	
	// Run validations
	fmt.Println("\n6. Running Validations...")
	report.Validations = runValidations(allocations, report)
	
	// Save report
	reportPath := filepath.Join(cfg.OutputDir, "genesis-report.json")
	reportData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	if err := ioutil.WriteFile(reportPath, reportData, 0644); err != nil {
		return fmt.Errorf("failed to save report: %w", err)
	}
	report.FilesGenerated = append(report.FilesGenerated, reportPath)
	
	// Generate comprehensive CSV
	csvPath := filepath.Join(cfg.OutputDir, "unified-allocations.csv")
	if err := generateComprehensiveCSV(allocations, csvPath); err != nil {
		return fmt.Errorf("failed to generate CSV: %w", err)
	}
	report.FilesGenerated = append(report.FilesGenerated, csvPath)
	
	// Print summary
	printSummary(report)
	
	fmt.Printf("\n✅ Genesis generation complete!\n")
	fmt.Printf("Output directory: %s\n", cfg.OutputDir)
	
	return nil
}

func loadAllAllocations() (map[string]*CombinedAllocation, error) {
	allocations := make(map[string]*CombinedAllocation)
	
	// Load Lux 7777 allocations (delta only)
	if err := loadLux7777Allocations(allocations); err != nil {
		return nil, err
	}
	
	// Load Zoo allocations
	if err := loadZooAllocations(allocations); err != nil {
		return nil, err
	}
	
	// Load Lux NFTs from Ethereum
	if err := loadLuxNFTAllocations(allocations); err != nil {
		return nil, err
	}
	
	// Add treasury if needed
	if cfg.IncludeTreasury {
		treasury := &CombinedAllocation{
			Address:    strings.ToLower(cfg.TreasuryAddress),
			Lux7777:    big.NewInt(0),
			Zoo200200:  big.NewInt(0),
			CChainLux:  func() *big.Int { v, _ := new(big.Int).SetString("1994739905397278683064838288203", 10); return v }(),
			XChainLux:  big.NewInt(0),
			TotalLux:   func() *big.Int { v, _ := new(big.Int).SetString("1994739905397278683064838288203", 10); return v }(),
			TotalZoo:   big.NewInt(0),
			Sources:    []string{"treasury"},
			Rationale:  "Treasury allocation",
		}
		allocations[treasury.Address] = treasury
		fmt.Printf("   Added treasury allocation: %s LUX\n", formatLux(treasury.TotalLux))
	}
	
	return allocations, nil
}

type CombinedAllocation struct {
	Address    string
	Lux7777    *big.Int
	Zoo200200  *big.Int
	CChainLux  *big.Int
	XChainLux  *big.Int
	TotalLux   *big.Int
	TotalZoo   *big.Int
	Sources    []string
	Rationale  string
	NFTData    map[string]interface{} // For storing NFT metadata like eggs
}

func loadLux7777Allocations(allocations map[string]*CombinedAllocation) error {
	log.Println("Loading Lux 7777 delta allocations for X-Chain...")
	
	// Use the delta file that contains only addresses that didn't get tokens on 96369
	possiblePaths := []string{
		"chaindata/lux-genesis-7777/7777-delta-allocations-for-xchain.csv",
		"data/chaindata/lux-genesis-7777/7777-delta-allocations-for-xchain.csv",
	}
	
	var file *os.File
	var err error
	var csvPath string
	
	for _, path := range possiblePaths {
		file, err = os.Open(path)
		if err == nil {
			csvPath = path
			break
		}
	}
	
	if file == nil {
		log.Printf("Warning: Could not find Lux 7777 delta CSV file")
		return nil
	}
	defer file.Close()
	
	log.Printf("Reading Lux 7777 delta data from: %s", csvPath)
	
	reader := csv.NewReader(file)
	reader.Comment = '#'
	reader.FieldsPerRecord = -1
	
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	count := 0
	totalLux := big.NewInt(0)
	
	// Skip header: address,balance_lux,balance_wei
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 3 {
			continue
		}
		
		address := strings.ToLower(records[i][0])
		balanceWei := records[i][2] // wei is in third column
		
		amount, ok := new(big.Int).SetString(balanceWei, 10)
		if !ok {
			continue
		}
		
		alloc := getOrCreateAllocation(allocations, address)
		alloc.Lux7777 = amount
		alloc.XChainLux = new(big.Int).Add(alloc.XChainLux, amount)
		alloc.TotalLux = new(big.Int).Add(alloc.TotalLux, amount)
		alloc.Sources = append(alloc.Sources, "Lux-7777-Delta")
		if alloc.Rationale == "" {
			alloc.Rationale = "Lux 7777 holder missed in 96369 migration"
		}
		
		totalLux = new(big.Int).Add(totalLux, amount)
		count++
	}
	
	fmt.Printf("   Loaded %d Lux 7777 delta allocations (Total: %s LUX)\n", count, formatLux(totalLux))
	return nil
}

func loadZooAllocations(allocations map[string]*CombinedAllocation) error {
	file, err := os.Open(cfg.Zoo200200Genesis)
	if err != nil {
		// Zoo allocations are optional
		fmt.Println("   Zoo allocations file not found, skipping")
		return nil
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	count := 0
	totalZoo := big.NewInt(0)
	eggCount := 0
	
	// Skip header: address,eggs,zoo_amount,source
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 3 {
			continue
		}
		
		address := strings.ToLower(records[i][0])
		eggs := "0"
		if len(records[i]) > 1 {
			eggs = records[i][1]
		}
		zooAmount := records[i][2]
		
		// Validate address
		if !common.IsHexAddress(address) {
			fmt.Printf("   Warning: Invalid address in Zoo: %s\n", address)
			continue
		}
		
		// Zoo amounts are already in whole tokens (not wei)
		amount, ok := new(big.Int).SetString(zooAmount, 10)
		if !ok || amount.Sign() <= 0 {
			continue
		}
		
		// Convert to wei for X-Chain allocation
		amountWei := new(big.Int).Mul(amount, big.NewInt(1e18))
		
		alloc := getOrCreateAllocation(allocations, address)
		alloc.Zoo200200 = amount // Store original Zoo amount
		alloc.XChainLux = new(big.Int).Add(alloc.XChainLux, amountWei) // Zoo goes to X-Chain
		alloc.TotalZoo = new(big.Int).Add(alloc.TotalZoo, amount)
		alloc.TotalLux = new(big.Int).Add(alloc.TotalLux, amountWei)
		if !contains(alloc.Sources, "Zoo-200200") {
			alloc.Sources = append(alloc.Sources, "Zoo-200200")
		}
		
		// Track eggs/NFTs
		if eggs != "0" && eggs != "" {
			eggNum, _ := new(big.Int).SetString(eggs, 10)
			if eggNum.Sign() > 0 {
				eggCount += int(eggNum.Int64())
				// Initialize NFTData if needed
				if alloc.NFTData == nil {
					alloc.NFTData = make(map[string]interface{})
				}
				alloc.NFTData["eggs"] = eggNum.Int64()
				
				if alloc.Rationale == "" {
					alloc.Rationale = fmt.Sprintf("Zoo holder with %s Egg NFT(s)", eggs)
				} else {
					alloc.Rationale += fmt.Sprintf(", %s Egg NFT(s)", eggs)
				}
			}
		} else if alloc.Rationale == "" {
			alloc.Rationale = "Zoo Network holder"
		}
		
		totalZoo = new(big.Int).Add(totalZoo, amount)
		count++
	}
	
	fmt.Printf("   Loaded %d Zoo allocations (Total: %s ZOO, %d Egg NFTs)\n", count, totalZoo.String(), eggCount)
	return nil
}

func loadLuxNFTAllocations(allocations map[string]*CombinedAllocation) error {
	log.Println("Loading Lux NFT allocations from Ethereum...")
	
	// Try to find the Lux NFT holder data
	possiblePaths := []string{
		"exports/lux-nft-analysis-20250723-014805/lux_nft_holders.csv",
		"data/exports/lux-nft-analysis-20250723-014805/lux_nft_holders.csv",
	}
	
	var file *os.File
	var err error
	var csvPath string
	
	for _, path := range possiblePaths {
		file, err = os.Open(path)
		if err == nil {
			csvPath = path
			break
		}
	}
	
	if file == nil {
		log.Printf("Warning: Could not find Lux NFT holders CSV file")
		return nil
	}
	defer file.Close()
	
	log.Printf("Reading Lux NFT data from: %s", csvPath)
	
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}
	
	count := 0
	totalNFTs := 0
	
	// Skip header: owner,balance
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 2 {
			continue
		}
		
		address := strings.ToLower(records[i][0])
		nftCount := records[i][1]
		
		// Validate address
		if !common.IsHexAddress(address) {
			fmt.Printf("   Warning: Invalid address in Lux NFT: %s\n", address)
			continue
		}
		
		numNFTs, ok := new(big.Int).SetString(nftCount, 10)
		if !ok || numNFTs.Sign() <= 0 {
			continue
		}
		
		alloc := getOrCreateAllocation(allocations, address)
		
		// Initialize NFTData if needed
		if alloc.NFTData == nil {
			alloc.NFTData = make(map[string]interface{})
		}
		alloc.NFTData["luxNFTs"] = numNFTs.Int64()
		
		// Grant validator eligibility (all Lux NFT holders can be validators)
		// Using 1M LUX per NFT as validator power
		oneMillion, _ := new(big.Int).SetString("1000000000000000000000000", 10) // 1M LUX in wei
		validatorPower := new(big.Int).Mul(numNFTs, oneMillion)
		alloc.XChainLux = new(big.Int).Add(alloc.XChainLux, validatorPower)
		alloc.TotalLux = new(big.Int).Add(alloc.TotalLux, validatorPower)
		
		if !contains(alloc.Sources, "Lux-NFT-ETH") {
			alloc.Sources = append(alloc.Sources, "Lux-NFT-ETH")
		}
		
		if alloc.Rationale == "" {
			alloc.Rationale = fmt.Sprintf("Lux NFT holder from Ethereum (%s NFTs)", nftCount)
		} else {
			alloc.Rationale += fmt.Sprintf(", Lux NFT holder (%s NFTs)", nftCount)
		}
		
		totalNFTs += int(numNFTs.Int64())
		count++
	}
	
	fmt.Printf("   Loaded %d Lux NFT holders (Total: %d NFTs)\n", count, totalNFTs)
	return nil
}

func getOrCreateAllocation(allocations map[string]*CombinedAllocation, address string) *CombinedAllocation {
	addr := strings.ToLower(address)
	if alloc, exists := allocations[addr]; exists {
		return alloc
	}
	
	alloc := &CombinedAllocation{
		Address:   addr,
		Lux7777:   big.NewInt(0),
		Zoo200200: big.NewInt(0),
		CChainLux: big.NewInt(0),
		XChainLux: big.NewInt(0),
		TotalLux:  big.NewInt(0),
		TotalZoo:  big.NewInt(0),
		Sources:   []string{},
		Rationale: "",
		NFTData:   nil, // Initialize as needed
	}
	allocations[addr] = alloc
	return alloc
}

func generatePChainGenesis(allocations map[string]*CombinedAllocation, validators []*validator.ValidatorInfo, report *GenesisReport) (*ChainReport, error) {
	// Create P-Chain genesis structure
	pChain := &ChainReport{
		Type:           "P-Chain",
		TotalAccounts:  0,
		TotalSupply:    "0",
		TopHolders:     []HolderInfo{},
		Validators:     []ValidatorInfo{},
		InitialStakers: len(validators),
	}
	
	// Create genesis builder
	builder, err := genesis.NewBuilder(cfg.Network)
	if err != nil {
		return nil, err
	}
	
	// Add validators and their allocations
	totalStake := big.NewInt(0)
	for _, v := range validators {
		// Add validator to P-Chain
		builder.AddStaker(genesis.StakerConfig{
			NodeID:            v.NodeID,
			ETHAddress:        v.ETHAddress,
			PublicKey:         v.PublicKey,
			ProofOfPossession: v.ProofOfPossession,
			Weight:            v.Weight,
			DelegationFee:     v.DelegationFee,
		})
		
		// Track validator info
		pChain.Validators = append(pChain.Validators, ValidatorInfo{
			NodeID:        v.NodeID,
			Weight:        fmt.Sprintf("%d", v.Weight),
			DelegationFee: fmt.Sprintf("%.2f%%", float64(v.DelegationFee)/10000),
		})
		
		totalStake = new(big.Int).Add(totalStake, new(big.Int).SetUint64(v.Weight))
	}
	
	pChain.TotalSupply = totalStake.String()
	
	// Build P-Chain genesis
	genesisData, err := builder.Build()
	if err != nil {
		return nil, err
	}
	
	// Save P-Chain genesis
	pChainPath := filepath.Join(cfg.OutputDir, "p-chain-genesis.json")
	pChainData, err := json.MarshalIndent(genesisData, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(pChainPath, pChainData, 0644); err != nil {
		return nil, err
	}
	
	report.FilesGenerated = append(report.FilesGenerated, pChainPath)
	fmt.Printf("   P-Chain: %d validators, Total stake: %s\n", len(validators), formatLux(totalStake))
	
	return pChain, nil
}

func generateCChainGenesis(allocations map[string]*CombinedAllocation, report *GenesisReport) (*ChainReport, error) {
	cChain := &ChainReport{
		Type:          "C-Chain",
		TotalAccounts: 0,
		TotalSupply:   "0",
		TopHolders:    []HolderInfo{},
	}
	
	// Create C-Chain genesis structure
	cChainGenesis := map[string]interface{}{
		"config": getCChainConfig(cfg.Network),
		"nonce": "0x0",
		"timestamp": "0x0",
		"extraData": "0x00",
		"gasLimit": "0x989680", // 10M gas
		"difficulty": "0x0",
		"mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase": "0x0000000000000000000000000000000000000000",
		"alloc": make(map[string]interface{}),
		"number": "0x0",
		"gasUsed": "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
		"baseFeePerGas": "0x0",
	}
	
	alloc := cChainGenesis["alloc"].(map[string]interface{})
	totalSupply := big.NewInt(0)
	
	// Process allocations
	type holder struct {
		address string
		balance *big.Int
	}
	holders := []holder{}
	
	for addr, allocation := range allocations {
		if allocation.CChainLux.Sign() > 0 {
			alloc[addr] = map[string]string{
				"balance": allocation.CChainLux.String(),
			}
			totalSupply = new(big.Int).Add(totalSupply, allocation.CChainLux)
			cChain.TotalAccounts++
			
			holders = append(holders, holder{addr, allocation.CChainLux})
		}
	}
	
	// Sort holders by balance for top holders
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].balance.Cmp(holders[i].balance) > 0 {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	
	// Add top 10 holders
	for i := 0; i < len(holders) && i < 10; i++ {
		percentage := new(big.Float).Quo(
			new(big.Float).SetInt(holders[i].balance),
			new(big.Float).SetInt(totalSupply),
		)
		percentage.Mul(percentage, big.NewFloat(100))
		pct, _ := percentage.Float64()
		
		cChain.TopHolders = append(cChain.TopHolders, HolderInfo{
			Rank:       i + 1,
			Address:    holders[i].address,
			Balance:    formatLux(holders[i].balance),
			Percentage: fmt.Sprintf("%.4f%%", pct),
			Source:     "C-Chain",
		})
	}
	
	cChain.TotalSupply = totalSupply.String()
	
	// Add precompiles
	addPrecompiles(alloc)
	
	// Save C-Chain genesis
	cChainPath := filepath.Join(cfg.OutputDir, "c-chain-genesis.json")
	cChainData, err := json.MarshalIndent(cChainGenesis, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(cChainPath, cChainData, 0644); err != nil {
		return nil, err
	}
	
	report.FilesGenerated = append(report.FilesGenerated, cChainPath)
	fmt.Printf("   C-Chain: %d accounts, Total supply: %s LUX\n", cChain.TotalAccounts, formatLux(totalSupply))
	
	return cChain, nil
}

func generateXChainGenesis(allocations map[string]*CombinedAllocation, report *GenesisReport) (*ChainReport, error) {
	xChain := &ChainReport{
		Type:          "X-Chain",
		TotalAccounts: 0,
		TotalSupply:   "0",
		TopHolders:    []HolderInfo{},
	}
	
	// Create multi-asset genesis with both LUX and ZOO assets
	// First, create the assets array
	assets := []map[string]interface{}{}
	
	// Add LUX asset
	luxHolders := []map[string]interface{}{}
	totalLuxSupply := big.NewInt(0)
	
	// Add ZOO asset
	zooHolders := []map[string]interface{}{}
	totalZooSupply := big.NewInt(0)
	
	// Process allocations
	type holder struct {
		address string
		balance *big.Int
	}
	holders := []holder{}
	
	for addr, allocation := range allocations {
		// Process LUX allocations
		if allocation.XChainLux.Sign() > 0 {
			luxHolder := map[string]interface{}{
				"address": addr,
				"amount": allocation.XChainLux.String(),
			}
			
			// Add NFT metadata if present
			if allocation.NFTData != nil && len(allocation.NFTData) > 0 {
				luxHolder["metadata"] = allocation.NFTData
			}
			
			luxHolders = append(luxHolders, luxHolder)
			totalLuxSupply = new(big.Int).Add(totalLuxSupply, allocation.XChainLux)
			holders = append(holders, holder{addr, allocation.XChainLux})
		}
		
		// Process ZOO allocations (separate asset)
		if allocation.Zoo200200.Sign() > 0 {
			// Convert ZOO to wei (18 decimals)
			zooWei := new(big.Int).Mul(allocation.Zoo200200, big.NewInt(1e18))
			zooHolder := map[string]interface{}{
				"address": addr,
				"amount": zooWei.String(),
			}
			
			// Add egg NFT data if present
			if allocation.NFTData != nil && allocation.NFTData["eggs"] != nil {
				zooHolder["eggs"] = allocation.NFTData["eggs"]
			}
			
			zooHolders = append(zooHolders, zooHolder)
			totalZooSupply = new(big.Int).Add(totalZooSupply, zooWei)
		}
		
		if allocation.XChainLux.Sign() > 0 || allocation.Zoo200200.Sign() > 0 {
			xChain.TotalAccounts++
		}
	}
	
	// Create LUX asset definition
	luxAsset := map[string]interface{}{
		"name": "Lux",
		"symbol": "LUX",
		"id": "LUX",
		"denomination": 9,
		"initialState": map[string]interface{}{
			"fixedCap": luxHolders,
		},
		"memo": []byte("Lux Network Native Token"),
	}
	assets = append(assets, luxAsset)
	
	// Create ZOO asset definition if there are ZOO holders
	if len(zooHolders) > 0 {
		zooAsset := map[string]interface{}{
			"name": "Zoo Token",
			"symbol": "ZOO",
			"id": "ZOO",
			"denomination": 18,
			"initialState": map[string]interface{}{
				"fixedCap": zooHolders,
			},
			"memo": []byte("Zoo Network Token"),
		}
		assets = append(assets, zooAsset)
		
		fmt.Printf("   Adding ZOO as separate asset with %d holders\n", len(zooHolders))
	}
	
	// Sort holders by balance for top holders
	for i := 0; i < len(holders); i++ {
		for j := i + 1; j < len(holders); j++ {
			if holders[j].balance.Cmp(holders[i].balance) > 0 {
				holders[i], holders[j] = holders[j], holders[i]
			}
		}
	}
	
	// Add top 10 holders
	for i := 0; i < len(holders) && i < 10; i++ {
		percentage := new(big.Float).Quo(
			new(big.Float).SetInt(holders[i].balance),
			new(big.Float).SetInt(totalLuxSupply),
		)
		percentage.Mul(percentage, big.NewFloat(100))
		pct, _ := percentage.Float64()
		
		xChain.TopHolders = append(xChain.TopHolders, HolderInfo{
			Rank:       i + 1,
			Address:    holders[i].address,
			Balance:    formatLux(holders[i].balance),
			Percentage: fmt.Sprintf("%.4f%%", pct),
			Source:     "X-Chain",
		})
	}
	
	xChain.TotalSupply = totalLuxSupply.String()
	
	// Create X-Chain genesis with multi-asset support
	xChainGenesis := map[string]interface{}{
		"networkID": getChainID(cfg.Network),
		"startTime": time.Now().Unix(),
		"assets": assets, // Multiple assets instead of single allocations
		"initialStakeDuration": 31536000, // 1 year in seconds
	}
	
	// Save X-Chain genesis
	xChainPath := filepath.Join(cfg.OutputDir, "x-chain-genesis.json")
	xChainData, err := json.MarshalIndent(xChainGenesis, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := ioutil.WriteFile(xChainPath, xChainData, 0644); err != nil {
		return nil, err
	}
	
	report.FilesGenerated = append(report.FilesGenerated, xChainPath)
	fmt.Printf("   X-Chain: %d accounts, LUX supply: %s, ZOO supply: %s\n", 
		xChain.TotalAccounts, formatLux(totalLuxSupply), formatLux(totalZooSupply))
	
	return xChain, nil
}

func getCChainConfig(network string) map[string]interface{} {
	chainID := getChainID(network)
	
	return map[string]interface{}{
		"chainId":             chainID,
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
		"feeConfig": map[string]interface{}{
			"gasLimit":        15000000,
			"targetBlockRate": 2,
			"minBaseFee":      25000000000,
			"targetGas":       15000000,
			"baseFeeChangeDenominator": 36,
			"minBlockGasCost": 0,
			"maxBlockGasCost": 1000000,
			"blockGasCostStep": 200000,
		},
		"allowFeeRecipients": true,
	}
}

func addPrecompiles(alloc map[string]interface{}) {
	// Add standard precompiles with admin permissions
	adminAddress := "0x1000000000000000000000000000000000000000"
	
	precompiles := map[string]string{
		"0x0000000000000000000000000000000000000400": "ContractDeployerAllowList",
		"0x0000000000000000000000000000000000000401": "FeeManager",
		"0x0000000000000000000000000000000000000402": "NativeMinter",
		"0x0000000000000000000000000000000000000403": "TxAllowList",
	}
	
	for addr, name := range precompiles {
		alloc[addr] = map[string]interface{}{
			"balance": "0x0",
			"storage": map[string]string{
				// Admin role for the admin address
				fmt.Sprintf("0x%064s", strings.TrimPrefix(adminAddress, "0x")): "0x0000000000000000000000000000000000000000000000000000000000000001",
			},
			"comment": name,
		}
	}
}

func createMigrationPlan(allocations map[string]*CombinedAllocation) []MigrationRecord {
	plan := []MigrationRecord{}
	
	// Count Lux 7777 migrations
	lux7777Count := 0
	lux7777Total := big.NewInt(0)
	for _, alloc := range allocations {
		if alloc.Lux7777.Sign() > 0 {
			lux7777Count++
			lux7777Total = new(big.Int).Add(lux7777Total, alloc.Lux7777)
		}
	}
	
	if lux7777Count > 0 {
		plan = append(plan, MigrationRecord{
			From:      "Lux-7777",
			To:        "Lux-96369-X-Chain",
			Type:      "Full Balance Migration",
			Accounts:  lux7777Count,
			Amount:    lux7777Total.String(),
			Rationale: "Migrate all Lux 7777 holders to new X-Chain with preserved balances",
		})
	}
	
	// Count Zoo migrations
	zooCount := 0
	zooTotal := big.NewInt(0)
	for _, alloc := range allocations {
		if alloc.Zoo200200.Sign() > 0 {
			zooCount++
			zooTotal = new(big.Int).Add(zooTotal, alloc.Zoo200200)
		}
	}
	
	if zooCount > 0 {
		plan = append(plan, MigrationRecord{
			From:      "Zoo-200200",
			To:        "Lux-96369-X-Chain",
			Type:      "Full Balance Migration with NFTs",
			Accounts:  zooCount,
			Amount:    zooTotal.String(),
			Rationale: "Migrate Zoo holders with Egg NFTs to X-Chain with preserved balances",
		})
	}
	
	return plan
}

func runValidations(allocations map[string]*CombinedAllocation, report *GenesisReport) []ValidationCheck {
	checks := []ValidationCheck{}
	
	// Check 1: Total supply conservation
	totalLux := big.NewInt(0)
	for _, alloc := range allocations {
		totalLux = new(big.Int).Add(totalLux, alloc.TotalLux)
	}
	
	expectedSupply := new(big.Int)
	expectedSupply.SetString("2000000000000000000000000000000", 10) // 2T LUX
	
	if totalLux.Cmp(expectedSupply) == 0 {
		checks = append(checks, ValidationCheck{
			Check:   "Total LUX Supply Conservation",
			Status:  "PASS",
			Details: fmt.Sprintf("Supply matches expected 2T LUX"),
		})
	} else {
		checks = append(checks, ValidationCheck{
			Check:   "Total LUX Supply Conservation",
			Status:  "WARNING",
			Details: fmt.Sprintf("Expected: %s, Actual: %s", expectedSupply.String(), totalLux.String()),
		})
	}
	
	// Check 2: No negative balances
	negativeCount := 0
	for _, alloc := range allocations {
		if alloc.TotalLux.Sign() < 0 || alloc.TotalZoo.Sign() < 0 {
			negativeCount++
		}
	}
	
	if negativeCount == 0 {
		checks = append(checks, ValidationCheck{
			Check:   "Balance Validation",
			Status:  "PASS",
			Details: "No negative balances found",
		})
	} else {
		checks = append(checks, ValidationCheck{
			Check:   "Balance Validation",
			Status:  "FAIL",
			Details: fmt.Sprintf("%d accounts with negative balances", negativeCount),
		})
	}
	
	// Check 3: Address validation
	invalidAddresses := 0
	for addr := range allocations {
		if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
			invalidAddresses++
		}
	}
	
	if invalidAddresses == 0 {
		checks = append(checks, ValidationCheck{
			Check:   "Address Format",
			Status:  "PASS",
			Details: "All addresses are valid Ethereum format",
		})
	} else {
		checks = append(checks, ValidationCheck{
			Check:   "Address Format",
			Status:  "FAIL",
			Details: fmt.Sprintf("%d invalid addresses", invalidAddresses),
		})
	}
	
	// Check 4: Validator stake validation
	if report.PChain != nil && len(report.PChain.Validators) > 0 {
		checks = append(checks, ValidationCheck{
			Check:   "Validator Configuration",
			Status:  "PASS",
			Details: fmt.Sprintf("%d validators configured with total stake %s", 
				len(report.PChain.Validators), report.PChain.TotalSupply),
		})
	}
	
	return checks
}

func generateComprehensiveCSV(allocations map[string]*CombinedAllocation, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Header
	header := []string{
		"address",
		"lux_7777_wei",
		"zoo_200200_wei",
		"c_chain_lux_wei",
		"x_chain_lux_wei",
		"total_lux_wei",
		"total_zoo_wei",
		"lux_human",
		"zoo_human",
		"sources",
		"rationale",
	}
	if err := writer.Write(header); err != nil {
		return err
	}
	
	// Sort by total LUX descending
	type entry struct {
		addr  string
		alloc *CombinedAllocation
	}
	
	entries := []entry{}
	for addr, alloc := range allocations {
		entries = append(entries, entry{addr, alloc})
	}
	
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].alloc.TotalLux.Cmp(entries[i].alloc.TotalLux) > 0 {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
	
	// Write records
	for _, e := range entries {
		record := []string{
			e.addr,
			e.alloc.Lux7777.String(),
			e.alloc.Zoo200200.String(),
			e.alloc.CChainLux.String(),
			e.alloc.XChainLux.String(),
			e.alloc.TotalLux.String(),
			e.alloc.TotalZoo.String(),
			formatLux(e.alloc.TotalLux),
			formatZoo(e.alloc.TotalZoo),
			strings.Join(e.alloc.Sources, ";"),
			e.alloc.Rationale,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

func loadValidators(validatorsFile string) ([]*validator.ValidatorInfo, error) {
	data, err := ioutil.ReadFile(validatorsFile)
	if err != nil {
		return nil, err
	}
	
	var validators []*validator.ValidatorInfo
	if err := json.Unmarshal(data, &validators); err != nil {
		return nil, err
	}
	
	return validators, nil
}

func printSummary(report *GenesisReport) {
	fmt.Println("\n========================================")
	fmt.Println("Genesis Generation Summary")
	fmt.Println("========================================")
	fmt.Printf("Network: %s (Chain ID: %d)\n", report.Network, report.ChainID)
	fmt.Printf("Generated: %s\n", report.Generated.Format(time.RFC3339))
	
	// P-Chain summary
	if report.PChain != nil {
		fmt.Printf("\nP-Chain:\n")
		fmt.Printf("  Validators: %d\n", len(report.PChain.Validators))
		fmt.Printf("  Total Stake: %s wei\n", report.PChain.TotalSupply)
	}
	
	// C-Chain summary
	if report.CChain != nil {
		fmt.Printf("\nC-Chain:\n")
		fmt.Printf("  Accounts: %d\n", report.CChain.TotalAccounts)
		fmt.Printf("  Total Supply: %s wei\n", report.CChain.TotalSupply)
		fmt.Printf("  Top Holders:\n")
		for i, holder := range report.CChain.TopHolders {
			if i >= 5 {
				break
			}
			fmt.Printf("    %d. %s: %s (%s)\n", 
				holder.Rank, holder.Address[:10]+"..."+holder.Address[len(holder.Address)-6:], 
				holder.Balance, holder.Percentage)
		}
	}
	
	// X-Chain summary
	if report.XChain != nil {
		fmt.Printf("\nX-Chain:\n")
		fmt.Printf("  Status: Genesis created\n")
	}
	
	// Migration plan
	fmt.Printf("\nMigration Plan:\n")
	for _, migration := range report.MigrationPlan {
		fmt.Printf("  %s → %s\n", migration.From, migration.To)
		fmt.Printf("    Type: %s\n", migration.Type)
		fmt.Printf("    Accounts: %d\n", migration.Accounts)
		fmt.Printf("    Rationale: %s\n", migration.Rationale)
	}
	
	// Validations
	fmt.Printf("\nValidations:\n")
	passCount := 0
	for _, check := range report.Validations {
		status := "✓"
		if check.Status != "PASS" {
			status = "✗"
		} else {
			passCount++
		}
		fmt.Printf("  %s %s: %s\n", status, check.Check, check.Details)
	}
	fmt.Printf("\nValidation Summary: %d/%d passed\n", passCount, len(report.Validations))
	
	// Files
	fmt.Printf("\nFiles Generated:\n")
	for _, file := range report.FilesGenerated {
		fmt.Printf("  - %s\n", file)
	}
}

func getChainID(network string) int {
	switch network {
	case "mainnet":
		return 96369
	case "testnet":
		return 96368
	default:
		return 96370 // local
	}
}

func formatLux(wei *big.Int) string {
	lux := new(big.Float).SetInt(wei)
	lux = lux.Quo(lux, big.NewFloat(1e9))
	return fmt.Sprintf("%.9f", lux)
}

func formatZoo(wei *big.Int) string {
	zoo := new(big.Float).SetInt(wei)
	zoo = zoo.Quo(zoo, big.NewFloat(1e18))
	return fmt.Sprintf("%.18f", zoo)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}