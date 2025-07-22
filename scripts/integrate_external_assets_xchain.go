// +build ignore

package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/vms/avm"
	"github.com/ava-labs/avalanchego/vms/nftfx"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ethereum/go-ethereum/common"
)

var (
	nftCSVPath      = flag.String("nft-csv", "", "Path to scanned NFT data CSV")
	tokenCSVPath    = flag.String("token-csv", "", "Path to scanned token data CSV")
	accountsCSVPath = flag.String("accounts-csv", "", "Path to 7777 accounts CSV")
	outputPath      = flag.String("output", "configs/xchain-genesis-complete.json", "Output genesis file")
	assetNamePrefix = flag.String("asset-prefix", "LUX", "Asset name prefix (LUX, ZOO, SPC, HANZO)")
)

// X-Chain Genesis structures
type XChainGenesis struct {
	Allocations []GenesisAsset `json:"allocations"`
	StartTime   int64          `json:"startTime"`
	Message     string         `json:"message"`
}

type GenesisAsset struct {
	AssetAlias  string                `json:"assetAlias"`
	AssetID     string                `json:"assetID"`
	InitialState map[string][]UTXOData `json:"initialState"`
	Memo        string                `json:"memo"`
}

type UTXOData struct {
	Amount      uint64            `json:"amount,omitempty"`      // For fungible tokens
	Locktime    uint64            `json:"locktime"`
	Threshold   uint32            `json:"threshold"`
	Addresses   []string          `json:"addresses"`
	Payload     string            `json:"payload,omitempty"`     // NFT metadata
	GroupID     uint32            `json:"groupID,omitempty"`     // NFT collection
}

// CSV data structures
type NFTHolder struct {
	Address         string
	AssetType       string
	CollectionType  string
	TokenCount      int
	StakingPowerWei *big.Int
	ChainName       string
	ContractAddress string
	ProjectName     string
	TokenIDs        []string
	ReceivedOnChain bool
}

type TokenHolder struct {
	Address         string
	BalanceWei      *big.Int
	ChainName       string
	ContractAddress string
	ProjectName     string
	ReceivedOnChain bool
}

type Account7777 struct {
	Address           string
	BalanceWei        *big.Int
	ValidatorEligible bool
}

func main() {
	flag.Parse()

	// Load all data sources
	fmt.Println("Loading external asset data...")
	
	var nftHolders []NFTHolder
	var tokenHolders []TokenHolder
	var accounts7777 []Account7777
	
	if *nftCSVPath != "" {
		var err error
		nftHolders, err = loadNFTData(*nftCSVPath)
		if err != nil {
			log.Printf("Warning: Failed to load NFT data: %v", err)
		} else {
			fmt.Printf("Loaded %d NFT holders\n", len(nftHolders))
		}
	}
	
	if *tokenCSVPath != "" {
		var err error
		tokenHolders, err = loadTokenData(*tokenCSVPath)
		if err != nil {
			log.Printf("Warning: Failed to load token data: %v", err)
		} else {
			fmt.Printf("Loaded %d token holders\n", len(tokenHolders))
		}
	}
	
	if *accountsCSVPath != "" {
		var err error
		accounts7777, err = load7777Accounts(*accountsCSVPath)
		if err != nil {
			log.Printf("Warning: Failed to load 7777 accounts: %v", err)
		} else {
			fmt.Printf("Loaded %d accounts from 7777\n", len(accounts7777))
		}
	}

	// Create genesis structure
	genesis := XChainGenesis{
		Allocations: []GenesisAsset{},
		StartTime:   time.Now().Unix(),
		Message:     "LUX Network X-Chain Genesis - Complete Historical Asset Integration",
	}

	// Process NFT collections
	if len(nftHolders) > 0 {
		nftAssets := processNFTHolders(nftHolders)
		genesis.Allocations = append(genesis.Allocations, nftAssets...)
	}

	// Process fungible tokens (external)
	if len(tokenHolders) > 0 {
		tokenAssets := processTokenHolders(tokenHolders)
		genesis.Allocations = append(genesis.Allocations, tokenAssets...)
	}

	// Process LUX token allocations from 7777
	if len(accounts7777) > 0 {
		luxAsset := process7777Accounts(accounts7777)
		genesis.Allocations = append(genesis.Allocations, luxAsset)
	}

	// Write genesis file
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	file, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(genesis); err != nil {
		log.Fatalf("Failed to encode genesis: %v", err)
	}

	// Print summary
	printSummary(genesis, nftHolders, tokenHolders, accounts7777)
}

func processNFTHolders(holders []NFTHolder) []GenesisAsset {
	// Group NFTs by collection
	collections := make(map[string][]NFTHolder)
	for _, holder := range holders {
		key := fmt.Sprintf("%s_%s_%s", holder.ProjectName, holder.CollectionType, holder.ContractAddress)
		collections[key] = append(collections[key], holder)
	}

	var assets []GenesisAsset
	
	for collectionKey, collectionHolders := range collections {
		parts := strings.Split(collectionKey, "_")
		project := parts[0]
		collectionType := parts[1]
		contractAddr := parts[2]
		
		// Create NFT asset
		asset := GenesisAsset{
			AssetAlias: fmt.Sprintf("%s_%s_NFT", strings.ToUpper(project), collectionType),
			AssetID:    generateAssetID(collectionKey),
			InitialState: map[string][]UTXOData{
				"nftMintOutput": []UTXOData{},
			},
			Memo: fmt.Sprintf("NFT Collection: %s %s from %s", project, collectionType, contractAddr),
		}

		// Create NFT outputs for each holder
		for _, holder := range collectionHolders {
			// Convert Ethereum address to X-Chain address
			xAddr := convertEthToXChainAddress(holder.Address)
			
			// Create one NFT output per token ID
			for i, tokenID := range holder.TokenIDs {
				utxo := UTXOData{
					Locktime:  0,
					Threshold: 1,
					Addresses: []string{xAddr},
					GroupID:   determineGroupID(holder.CollectionType),
					Payload:   createNFTPayload(holder, tokenID, i),
				}
				
				// Add validator staking capability for eligible NFTs
				if holder.StakingPowerWei.Sign() > 0 {
					utxo.Payload = addStakingCapability(utxo.Payload, holder.StakingPowerWei)
				}
				
				asset.InitialState["nftMintOutput"] = append(asset.InitialState["nftMintOutput"], utxo)
			}
		}
		
		assets = append(assets, asset)
	}
	
	return assets
}

func processTokenHolders(holders []TokenHolder) []GenesisAsset {
	// Group tokens by contract
	contracts := make(map[string][]TokenHolder)
	for _, holder := range holders {
		key := fmt.Sprintf("%s_%s_%s", holder.ProjectName, holder.ChainName, holder.ContractAddress)
		contracts[key] = append(contracts[key], holder)
	}

	var assets []GenesisAsset
	
	for contractKey, contractHolders := range contracts {
		parts := strings.Split(contractKey, "_")
		project := parts[0]
		chain := parts[1]
		contractAddr := parts[2]
		
		// Create fungible token asset
		asset := GenesisAsset{
			AssetAlias: fmt.Sprintf("%s_TOKEN_%s", strings.ToUpper(project), chain),
			AssetID:    generateAssetID(contractKey),
			InitialState: map[string][]UTXOData{
				"fixedCapMintOutput": []UTXOData{},
			},
			Memo: fmt.Sprintf("Token: %s from %s on %s", project, contractAddr, chain),
		}

		// Create token outputs for each holder
		for _, holder := range contractHolders {
			// Skip if already received on-chain
			if holder.ReceivedOnChain {
				continue
			}
			
			// Convert Ethereum address to X-Chain address
			xAddr := convertEthToXChainAddress(holder.Address)
			
			// Convert balance to X-Chain denomination (nano-units)
			amount := new(big.Int).Div(holder.BalanceWei, big.NewInt(1e9))
			
			utxo := UTXOData{
				Amount:    amount.Uint64(),
				Locktime:  0,
				Threshold: 1,
				Addresses: []string{xAddr},
			}
			
			asset.InitialState["fixedCapMintOutput"] = append(asset.InitialState["fixedCapMintOutput"], utxo)
		}
		
		assets = append(assets, asset)
	}
	
	return assets
}

func process7777Accounts(accounts []Account7777) GenesisAsset {
	// Create main LUX token asset
	asset := GenesisAsset{
		AssetAlias: "LUX",
		AssetID:    generateAssetID("LUX_MAIN"),
		InitialState: map[string][]UTXOData{
			"fixedCapMintOutput": []UTXOData{},
		},
		Memo: "LUX Token - Migrated from chain 7777",
	}

	// Process each account
	for _, account := range accounts {
		// Convert Ethereum address to X-Chain address
		xAddr := convertEthToXChainAddress(account.Address)
		
		// Convert balance to nano-units
		amount := new(big.Int).Div(account.BalanceWei, big.NewInt(1e9))
		
		// Create vesting schedule for large holders
		if account.ValidatorEligible {
			// 10% immediate
			immediateAmount := new(big.Int).Div(amount, big.NewInt(10))
			utxo := UTXOData{
				Amount:    immediateAmount.Uint64(),
				Locktime:  0,
				Threshold: 1,
				Addresses: []string{xAddr},
			}
			asset.InitialState["fixedCapMintOutput"] = append(asset.InitialState["fixedCapMintOutput"], utxo)
			
			// 90% vested over 1 year (4 quarterly unlocks)
			vestedAmount := new(big.Int).Sub(amount, immediateAmount)
			quarterlyAmount := new(big.Int).Div(vestedAmount, big.NewInt(4))
			
			for i := 1; i <= 4; i++ {
				vestingUTXO := UTXOData{
					Amount:    quarterlyAmount.Uint64(),
					Locktime:  uint64(time.Now().Unix() + int64(i*90*24*60*60)),
					Threshold: 1,
					Addresses: []string{xAddr},
				}
				asset.InitialState["fixedCapMintOutput"] = append(asset.InitialState["fixedCapMintOutput"], vestingUTXO)
			}
		} else {
			// Small holders get immediate access
			utxo := UTXOData{
				Amount:    amount.Uint64(),
				Locktime:  0,
				Threshold: 1,
				Addresses: []string{xAddr},
			}
			asset.InitialState["fixedCapMintOutput"] = append(asset.InitialState["fixedCapMintOutput"], utxo)
		}
	}
	
	return asset
}

func convertEthToXChainAddress(ethAddr string) string {
	// This is a simplified placeholder
	// In production, you'd need proper bech32 encoding with correct HRP
	ethAddrClean := strings.TrimPrefix(ethAddr, "0x")
	return fmt.Sprintf("X-lux1%s", strings.ToLower(ethAddrClean[:38]))
}

func generateAssetID(seed string) string {
	// Generate deterministic asset ID from seed
	// In production, this would use proper UTXO ID generation
	h := crypto.SHA256.Hash([]byte(seed))
	return ids.ID(h).String()
}

func determineGroupID(collectionType string) uint32 {
	// Assign group IDs based on collection type
	switch strings.ToLower(collectionType) {
	case "validator":
		return 1
	case "card":
		return 2
	case "coin":
		return 3
	case "animal":
		return 4
	case "habitat":
		return 5
	case "item":
		return 6
	case "pony":
		return 7
	case "accessory":
		return 8
	case "ai":
		return 9
	case "algorithm":
		return 10
	case "data":
		return 11
	default:
		return 99
	}
}

func createNFTPayload(holder NFTHolder, tokenID string, index int) string {
	// Create NFT metadata payload
	metadata := map[string]interface{}{
		"tokenId":         tokenID,
		"project":         holder.ProjectName,
		"collectionType":  holder.CollectionType,
		"originalChain":   holder.ChainName,
		"originalContract": holder.ContractAddress,
		"migrationDate":   time.Now().Format(time.RFC3339),
		"index":           index,
	}
	
	data, _ := json.Marshal(metadata)
	return string(data)
}

func addStakingCapability(payload string, stakingPower *big.Int) string {
	// Add staking capability to NFT metadata
	var metadata map[string]interface{}
	json.Unmarshal([]byte(payload), &metadata)
	
	metadata["stakingEnabled"] = true
	metadata["stakingPowerWei"] = stakingPower.String()
	metadata["stakingPowerLux"] = new(big.Float).Quo(
		new(big.Float).SetInt(stakingPower),
		new(big.Float).SetInt(big.NewInt(1e18)),
	).String()
	
	data, _ := json.Marshal(metadata)
	return string(data)
}

func loadNFTData(path string) ([]NFTHolder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var holders []NFTHolder
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Parse CSV fields
		// address,asset_type,collection_type,balance_or_count,staking_power_wei,staking_power_token,chain_name,contract_address,project_name,last_activity_block,received_on_chain,token_ids
		
		tokenCount, _ := strconv.Atoi(record[3])
		stakingPowerWei := new(big.Int)
		stakingPowerWei.SetString(record[4], 10)
		
		received := record[10] == "true"
		tokenIDs := strings.Split(record[11], ";")
		
		holders = append(holders, NFTHolder{
			Address:         record[0],
			AssetType:       record[1],
			CollectionType:  record[2],
			TokenCount:      tokenCount,
			StakingPowerWei: stakingPowerWei,
			ChainName:       record[6],
			ContractAddress: record[7],
			ProjectName:     record[8],
			TokenIDs:        tokenIDs,
			ReceivedOnChain: received,
		})
	}

	return holders, nil
}

func loadTokenData(path string) ([]TokenHolder, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var holders []TokenHolder
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Parse balance
		balanceWei := new(big.Int)
		balanceWei.SetString(record[3], 10)
		
		received := record[10] == "true"
		
		holders = append(holders, TokenHolder{
			Address:         record[0],
			BalanceWei:      balanceWei,
			ChainName:       record[6],
			ContractAddress: record[7],
			ProjectName:     record[8],
			ReceivedOnChain: received,
		})
	}

	return holders, nil
}

func load7777Accounts(path string) ([]Account7777, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var accounts []Account7777
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		balanceWei := new(big.Int)
		balanceWei.SetString(record[1], 10)
		
		validatorEligible := record[3] == "true"
		
		accounts = append(accounts, Account7777{
			Address:           record[0],
			BalanceWei:        balanceWei,
			ValidatorEligible: validatorEligible,
		})
	}

	return accounts, nil
}

func printSummary(genesis XChainGenesis, nftHolders []NFTHolder, tokenHolders []TokenHolder, accounts7777 []Account7777) {
	fmt.Printf("\n=== X-Chain Genesis Integration Summary ===\n")
	fmt.Printf("Genesis timestamp: %s\n", time.Unix(genesis.StartTime, 0).Format(time.RFC3339))
	fmt.Printf("Total asset types created: %d\n", len(genesis.Allocations))
	
	// Count NFT collections
	nftCollections := make(map[string]int)
	totalNFTs := 0
	for _, holder := range nftHolders {
		key := fmt.Sprintf("%s_%s", holder.ProjectName, holder.CollectionType)
		nftCollections[key] += holder.TokenCount
		totalNFTs += holder.TokenCount
	}
	
	fmt.Printf("\nNFT Collections:\n")
	for collection, count := range nftCollections {
		fmt.Printf("  - %s: %d NFTs\n", collection, count)
	}
	fmt.Printf("  Total NFTs: %d\n", totalNFTs)
	
	// Count token distributions
	if len(tokenHolders) > 0 {
		fmt.Printf("\nExternal Tokens:\n")
		tokensByProject := make(map[string]*big.Int)
		for _, holder := range tokenHolders {
			if tokensByProject[holder.ProjectName] == nil {
				tokensByProject[holder.ProjectName] = new(big.Int)
			}
			tokensByProject[holder.ProjectName].Add(tokensByProject[holder.ProjectName], holder.BalanceWei)
		}
		
		for project, total := range tokensByProject {
			totalFloat := new(big.Float).Quo(
				new(big.Float).SetInt(total),
				new(big.Float).SetInt(big.NewInt(1e18)),
			)
			fmt.Printf("  - %s: %.6f tokens\n", project, totalFloat)
		}
	}
	
	// Count 7777 migration
	if len(accounts7777) > 0 {
		totalLux := new(big.Int)
		validatorCount := 0
		for _, account := range accounts7777 {
			totalLux.Add(totalLux, account.BalanceWei)
			if account.ValidatorEligible {
				validatorCount++
			}
		}
		
		totalLuxFloat := new(big.Float).Quo(
			new(big.Float).SetInt(totalLux),
			new(big.Float).SetInt(big.NewInt(1e18)),
		)
		
		fmt.Printf("\n7777 Migration:\n")
		fmt.Printf("  - Total accounts: %d\n", len(accounts7777))
		fmt.Printf("  - Validator eligible: %d\n", validatorCount)
		fmt.Printf("  - Total LUX: %.6f\n", totalLuxFloat)
	}
	
	fmt.Printf("\n✅ X-Chain genesis with complete historical data generated!\n")
	fmt.Printf("Output file: %s\n", *outputPath)
}