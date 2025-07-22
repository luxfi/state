package scanner

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	// TODO: Replace with github.com/luxfi/geth when available
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Config holds scanner configuration
type Config struct {
	Chain           string
	RPC             string
	ContractAddress string
	ContractType    string // "nft", "token", or "auto"
	OutputPath      string
	BlockRange      int64
	ProjectName     string
	CrossRefPath    string
}

// Scanner performs external asset scanning
type Scanner struct {
	config  Config
	client  *ethclient.Client
	project ProjectConfig
}

// Result contains scan results
type Result struct {
	Chain            string
	ContractAddress  string
	AssetType        string
	TotalHolders     int
	TotalNFTs        int
	TotalSupply      string
	NFTCollections   map[string]int
	CrossRefStats    *CrossRefStats
	OutputFile       string
}

// CrossRefStats contains cross-reference statistics
type CrossRefStats struct {
	AlreadyReceived int
	NotYetReceived  int
}

// New creates a new scanner
func New(config Config) (*Scanner, error) {
	// Get project config
	projectConfig, exists := projectConfigs[config.ProjectName]
	if !exists {
		return nil, fmt.Errorf("unknown project: %s", config.ProjectName)
	}

	// Set up RPC URL
	if config.RPC == "" {
		if defaultRPC, ok := chainRPCs[config.Chain]; ok {
			config.RPC = defaultRPC
			log.Printf("Using default RPC for %s", config.Chain)
		} else {
			return nil, fmt.Errorf("no RPC URL provided and no default for chain %s", config.Chain)
		}
	}

	// Connect to EVM chain
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Chain, err)
	}

	// Set default output path
	if config.OutputPath == "" {
		assetType := "assets"
		if config.ContractType != "auto" {
			assetType = config.ContractType + "s"
		}
		config.OutputPath = fmt.Sprintf("exports/%s-%s-%s.csv", config.ProjectName, assetType, config.Chain)
	}

	return &Scanner{
		config:  config,
		client:  client,
		project: projectConfig,
	}, nil
}

// Scan performs the asset scan
func (s *Scanner) Scan() (*Result, error) {
	ctx := context.Background()
	contractAddr := common.HexToAddress(s.config.ContractAddress)

	// Get current block
	currentBlock, err := s.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current block: %w", err)
	}

	log.Printf("Current block: %d", currentBlock)
	log.Printf("Scanning back %d blocks", s.config.BlockRange)

	// Detect contract type if auto
	isNFT := false
	if s.config.ContractType == "auto" {
		isNFT, err = s.detectContractType(contractAddr)
		if err != nil {
			return nil, fmt.Errorf("could not auto-detect contract type: %w", err)
		}
		log.Printf("Detected contract type: %s", map[bool]string{true: "NFT", false: "Token"}[isNFT])
	} else {
		isNFT = s.config.ContractType == "nft"
	}

	// Scan for holders
	var holders map[string]*AssetHolder
	if isNFT {
		holders, err = s.scanNFTHolders(contractAddr, currentBlock)
	} else {
		holders, err = s.scanTokenHolders(contractAddr, currentBlock)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan holders: %w", err)
	}

	// Cross-reference if requested
	var crossRefStats *CrossRefStats
	if s.config.CrossRefPath != "" {
		crossRefStats = s.crossReference(holders)
	}

	// Export to CSV
	if err := s.exportToCSV(holders); err != nil {
		return nil, fmt.Errorf("failed to export to CSV: %w", err)
	}

	// Build result
	result := &Result{
		Chain:           s.config.Chain,
		ContractAddress: s.config.ContractAddress,
		AssetType:       map[bool]string{true: "NFT", false: "Token"}[isNFT],
		TotalHolders:    len(holders),
		CrossRefStats:   crossRefStats,
		OutputFile:      s.config.OutputPath,
	}

	if isNFT {
		result.NFTCollections = make(map[string]int)
		totalNFTs := 0
		for _, holder := range holders {
			key := fmt.Sprintf("%s_%s", holder.ProjectName, holder.CollectionType)
			result.NFTCollections[key] += len(holder.TokenIDs)
			totalNFTs += len(holder.TokenIDs)
		}
		result.TotalNFTs = totalNFTs
	} else {
		// Calculate total supply for tokens
		total := new(big.Int)
		for _, holder := range holders {
			total.Add(total, holder.Balance)
		}
		result.TotalSupply = formatTokenAmount(total)
	}

	return result, nil
}

func (s *Scanner) detectContractType(contractAddr common.Address) (bool, error) {
	// Try to call ERC721 totalSupply
	nftABI, _ := abi.JSON(strings.NewReader(erc721ABI))
	data, _ := nftABI.Pack("totalSupply")

	msg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: data,
	}

	_, err := s.client.CallContract(context.Background(), msg, nil)
	if err == nil {
		// Also check if it has ownerOf function
		data, _ = nftABI.Pack("ownerOf", big.NewInt(0))
		msg.Data = data
		_, err = s.client.CallContract(context.Background(), msg, nil)
		if err == nil || strings.Contains(err.Error(), "owner query for nonexistent token") {
			return true, nil // It's an NFT
		}
	}

	// Try ERC20 decimals function
	tokenABI, _ := abi.JSON(strings.NewReader(erc20ABI))
	data, _ = tokenABI.Pack("decimals")
	msg.Data = data

	_, err = s.client.CallContract(context.Background(), msg, nil)
	if err == nil {
		return false, nil // It's a token
	}

	return false, fmt.Errorf("could not determine contract type")
}

func (s *Scanner) exportToCSV(holders map[string]*AssetHolder) error {
	// Ensure output directory exists
	outputDir := filepath.Dir(s.config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create CSV file
	file, err := os.Create(s.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"address",
		"asset_type",
		"collection_type",
		"balance_or_count",
		"staking_power_wei",
		"staking_power_token",
		"chain_name",
		"contract_address",
		"project_name",
		"last_activity_block",
		"received_on_chain",
		"token_ids", // For NFTs
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write holder data
	for _, holder := range holders {
		balanceOrCount := ""
		tokenIDsStr := ""

		if holder.AssetType == "Token" {
			balanceOrCount = holder.Balance.String()
		} else {
			balanceOrCount = strconv.Itoa(len(holder.TokenIDs))
			// Join token IDs
			ids := make([]string, len(holder.TokenIDs))
			for i, id := range holder.TokenIDs {
				ids[i] = id.String()
			}
			tokenIDsStr = strings.Join(ids, ";")
		}

		stakingPowerToken := new(big.Float).Quo(
			new(big.Float).SetInt(holder.StakingPower),
			new(big.Float).SetInt(big.NewInt(1e18)),
		)

		record := []string{
			holder.Address.Hex(),
			holder.AssetType,
			holder.CollectionType,
			balanceOrCount,
			holder.StakingPower.String(),
			fmt.Sprintf("%.6f", stakingPowerToken),
			holder.ChainName,
			holder.ContractAddress,
			holder.ProjectName,
			strconv.FormatUint(holder.LastActivity, 10),
			strconv.FormatBool(holder.ReceivedOnChain),
			tokenIDsStr,
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	return nil
}

func (s *Scanner) crossReference(holders map[string]*AssetHolder) *CrossRefStats {
	// TODO: Implement actual cross-reference with chain data
	// For now, return placeholder stats
	stats := &CrossRefStats{
		AlreadyReceived: 0,
		NotYetReceived:  0,
	}

	for _, holder := range holders {
		if holder.ReceivedOnChain {
			stats.AlreadyReceived++
		} else {
			stats.NotYetReceived++
		}
	}

	return stats
}

// Helper functions
func formatTokenAmount(amount *big.Int) string {
	// Convert to token units (assuming 18 decimals)
	tokenAmount := new(big.Float).Quo(
		new(big.Float).SetInt(amount),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	return fmt.Sprintf("%.6f", tokenAmount)
}

// FormatStakingPower formats staking power for display
func FormatStakingPower(power *big.Int) string {
	if power.Sign() == 0 {
		return "0"
	}
	
	// Convert to token units
	tokens := new(big.Float).Quo(
		new(big.Float).SetInt(power),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	
	// Format with appropriate units
	tokensFloat, _ := tokens.Float64()
	if tokensFloat >= 1e6 {
		return fmt.Sprintf("%.1fM", tokensFloat/1e6)
	} else if tokensFloat >= 1e3 {
		return fmt.Sprintf("%.0fK", tokensFloat/1e3)
	}
	return fmt.Sprintf("%.0f", tokensFloat)
}