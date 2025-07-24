package commands

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

// ERC721 ABI for totalSupply and ownerOf
const erc721ABI = `[
	{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[{"name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"name":"","type":"address"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"name","outputs":[{"name":"","type":"string"}],"type":"function"}
]`

// CurrentHolderResult represents the result of scanning current NFT holders
type CurrentHolderResult struct {
	Timestamp       int64                      `json:"timestamp"`
	ContractAddress string                     `json:"contract_address"`
	ContractName    string                     `json:"contract_name"`
	TotalSupply     uint64                     `json:"total_supply"`
	UniqueHolders   int                        `json:"unique_holders"`
	BurnedTokens    int                        `json:"burned_tokens"`
	Holders         map[string]*HolderInfo     `json:"holders"`
	Summary         CurrentHolderSummary       `json:"summary"`
}

type HolderInfo struct {
	TokenCount        int      `json:"token_count"`
	TokenIDs          []uint64 `json:"token_ids"`
	ValidatorEligible bool     `json:"validator_eligible,omitempty"`
}

type CurrentHolderSummary struct {
	TotalNFTsHeld      int               `json:"total_nfts_held"`
	Distribution       map[string]int    `json:"distribution"`
	ValidatorEligible  []string          `json:"validator_eligible,omitempty"`
}

// NewScanCurrentHoldersCommand creates the scan-current-holders command
func NewScanCurrentHoldersCommand() *cobra.Command {
	var (
		rpcURLs      []string
		contract     string
		outputCSV    string
		outputJSON   string
		maxSupply    uint64
		concurrent   int
		projectName  string
	)

	cmd := &cobra.Command{
		Use:   "scan-current-holders",
		Short: "Scan current NFT holders without historical data",
		Long: `Efficiently scan current NFT holders by querying token ownership directly.
This is much faster than scanning historical blocks and gives you the current state.

Supports load balancing across multiple RPC endpoints.`,
		Example: `  # Scan EGG NFT holders on BSC
  archaeology scan-current-holders \
    --rpc https://bsc-dataseed.bnbchain.org \
    --rpc https://bsc-dataseed.nariox.org \
    --contract 0x5bb68cf06289d54efde25155c88003be685356a8 \
    --output holders.csv

  # Scan LUX NFT holders on Ethereum
  archaeology scan-current-holders \
    --rpc https://mainnet.infura.io/v3/YOUR_KEY \
    --contract 0x31e0f919c67cedd2bc3e294340dc900735810311 \
    --project lux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(rpcURLs) == 0 {
				return fmt.Errorf("at least one RPC URL required")
			}

			// Connect to first available RPC
			var client *ethclient.Client
			var err error
			for _, rpc := range rpcURLs {
				client, err = ethclient.Dial(rpc)
				if err == nil {
					log.Printf("Connected to %s", rpc)
					break
				}
			}
			if client == nil {
				return fmt.Errorf("failed to connect to any RPC endpoint")
			}

			// Parse contract address
			contractAddr := common.HexToAddress(contract)
			
			// Parse ABI
			parsedABI, err := abi.JSON(strings.NewReader(erc721ABI))
			if err != nil {
				return fmt.Errorf("failed to parse ABI: %w", err)
			}

			// Get contract name
			var contractName string
			nameData, err := client.CallContract(context.Background(), ethereum.CallMsg{
				To:   &contractAddr,
				Data: parsedABI.Methods["name"].ID,
			}, nil)
			if err == nil && len(nameData) > 0 {
				parsedABI.UnpackIntoInterface(&contractName, "name", nameData)
			}
			if contractName == "" {
				contractName = projectName
			}

			log.Printf("Scanning %s NFT holders at %s", contractName, contract)

			// Get total supply
			totalSupplyData, err := client.CallContract(context.Background(), ethereum.CallMsg{
				To:   &contractAddr,
				Data: parsedABI.Methods["totalSupply"].ID,
			}, nil)
			if err != nil {
				return fmt.Errorf("failed to get total supply: %w", err)
			}

			var totalSupply *big.Int
			err = parsedABI.UnpackIntoInterface(&totalSupply, "totalSupply", totalSupplyData)
			if err != nil || totalSupply == nil {
				// Use max supply if provided
				if maxSupply > 0 {
					totalSupply = big.NewInt(int64(maxSupply))
					log.Printf("Using max supply: %d", maxSupply)
				} else {
					return fmt.Errorf("failed to get total supply and no max-supply provided")
				}
			}

			log.Printf("Total supply: %s", totalSupply.String())

			// Scan holders concurrently
			holders := make(map[string][]uint64)
			holdersMux := sync.Mutex{}
			burned := []uint64{}
			burnedMux := sync.Mutex{}

			// Create work channel
			work := make(chan uint64, concurrent)
			var wg sync.WaitGroup

			// Start workers
			clients := make([]*ethclient.Client, len(rpcURLs))
			for i, rpc := range rpcURLs {
				c, err := ethclient.Dial(rpc)
				if err != nil {
					log.Printf("Warning: failed to create client for %s: %v", rpc, err)
					clients[i] = client // Use primary client as fallback
				} else {
					clients[i] = c
				}
			}

			for i := 0; i < concurrent; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					
					// Use round-robin client selection
					clientIdx := workerID % len(clients)
					workerClient := clients[clientIdx]

					for tokenID := range work {
						// Pack the ownerOf call
						input, err := parsedABI.Pack("ownerOf", big.NewInt(int64(tokenID)))
						if err != nil {
							continue
						}

						// Call ownerOf
						output, err := workerClient.CallContract(context.Background(), ethereum.CallMsg{
							To:   &contractAddr,
							Data: input,
						}, nil)

						if err != nil {
							// Token might be burned or doesn't exist
							if strings.Contains(err.Error(), "nonexistent") || strings.Contains(err.Error(), "invalid") {
								// Skip
							} else {
								burnedMux.Lock()
								burned = append(burned, tokenID)
								burnedMux.Unlock()
							}
							continue
						}

						// Unpack owner address
						var owner common.Address
						err = parsedABI.UnpackIntoInterface(&owner, "ownerOf", output)
						if err != nil {
							continue
						}

						// Add to holders
						holdersMux.Lock()
						holders[strings.ToLower(owner.Hex())] = append(holders[strings.ToLower(owner.Hex())], tokenID)
						holdersMux.Unlock()

						if tokenID%100 == 0 {
							log.Printf("Processed token %d", tokenID)
						}
					}
				}(i)
			}

			// Send work
			supply := totalSupply.Uint64()
			for i := uint64(0); i < supply; i++ {
				work <- i
			}
			close(work)

			// Wait for completion
			wg.Wait()

			// Prepare results
			result := CurrentHolderResult{
				Timestamp:       time.Now().Unix(),
				ContractAddress: contract,
				ContractName:    contractName,
				TotalSupply:     supply,
				UniqueHolders:   len(holders),
				BurnedTokens:    len(burned),
				Holders:         make(map[string]*HolderInfo),
				Summary: CurrentHolderSummary{
					TotalNFTsHeld:     int(supply) - len(burned),
					Distribution:      make(map[string]int),
					ValidatorEligible: []string{},
				},
			}

			// Process holders
			for addr, tokenIDs := range holders {
				sort.Slice(tokenIDs, func(i, j int) bool { return tokenIDs[i] < tokenIDs[j] })
				
				info := &HolderInfo{
					TokenCount: len(tokenIDs),
					TokenIDs:   tokenIDs,
				}

				// For LUX project, NFT holders are validator eligible
				if projectName == "lux" {
					info.ValidatorEligible = true
					result.Summary.ValidatorEligible = append(result.Summary.ValidatorEligible, addr)
				}

				result.Holders[addr] = info

				// Update distribution
				var key string
				switch {
				case info.TokenCount == 1:
					key = "1 NFT"
				case info.TokenCount <= 5:
					key = "2-5 NFTs"
				case info.TokenCount <= 10:
					key = "6-10 NFTs"
				default:
					key = "11+ NFTs"
				}
				result.Summary.Distribution[key]++
			}

			// Sort validator eligible addresses
			sort.Strings(result.Summary.ValidatorEligible)

			// Save JSON if requested
			if outputJSON != "" {
				data, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal JSON: %w", err)
				}
				if err := os.WriteFile(outputJSON, data, 0644); err != nil {
					return fmt.Errorf("failed to write JSON: %w", err)
				}
				log.Printf("Saved JSON to %s", outputJSON)
			}

			// Save CSV if requested
			if outputCSV != "" {
				file, err := os.Create(outputCSV)
				if err != nil {
					return fmt.Errorf("failed to create CSV: %w", err)
				}
				defer file.Close()

				writer := csv.NewWriter(file)
				defer writer.Flush()

				// Write header
				header := []string{"address", "token_count", "token_ids"}
				if projectName == "lux" {
					header = append(header, "validator_eligible")
				}
				writer.Write(header)

				// Write data
				for addr, info := range result.Holders {
					tokenIDStrs := make([]string, len(info.TokenIDs))
					for i, id := range info.TokenIDs {
						tokenIDStrs[i] = fmt.Sprintf("%d", id)
					}
					
					row := []string{
						addr,
						fmt.Sprintf("%d", info.TokenCount),
						strings.Join(tokenIDStrs, ";"),
					}
					if projectName == "lux" {
						row = append(row, "true")
					}
					writer.Write(row)
				}

				log.Printf("Saved CSV to %s", outputCSV)
			}

			// Print summary
			fmt.Printf("\n=== %s Holder Summary ===\n", contractName)
			fmt.Printf("Contract: %s\n", contract)
			fmt.Printf("Total Supply: %d\n", result.TotalSupply)
			fmt.Printf("Unique Holders: %d\n", result.UniqueHolders)
			fmt.Printf("Burned Tokens: %d\n", result.BurnedTokens)
			if len(result.Summary.ValidatorEligible) > 0 {
				fmt.Printf("Validator Eligible: %d addresses\n", len(result.Summary.ValidatorEligible))
			}

			fmt.Println("\nDistribution:")
			for key, count := range result.Summary.Distribution {
				fmt.Printf("  %s: %d holders\n", key, count)
			}

			// Show top holders
			type holderCount struct {
				address string
				count   int
			}
			var topHolders []holderCount
			for addr, info := range result.Holders {
				topHolders = append(topHolders, holderCount{addr, info.TokenCount})
			}
			sort.Slice(topHolders, func(i, j int) bool {
				return topHolders[i].count > topHolders[j].count
			})

			fmt.Println("\nTop 10 Holders:")
			for i := 0; i < 10 && i < len(topHolders); i++ {
				fmt.Printf("  %d. %s: %d NFTs\n", i+1, topHolders[i].address, topHolders[i].count)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().StringSliceVar(&rpcURLs, "rpc", nil, "RPC endpoints (can specify multiple for load balancing)")
	cmd.Flags().StringVar(&contract, "contract", "", "NFT contract address")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().Uint64Var(&maxSupply, "max-supply", 0, "Maximum supply to scan (if totalSupply fails)")
	cmd.Flags().IntVar(&concurrent, "concurrent", 10, "Number of concurrent workers")
	cmd.Flags().StringVar(&projectName, "project", "", "Project name (e.g., lux, zoo)")

	cmd.MarkFlagRequired("contract")

	return cmd
}