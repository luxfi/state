package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/luxfi/geth"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/ethclient"
	"github.com/spf13/cobra"
)

// CachedBlock represents a cached block with its logs
type CachedBlock struct {
	Number uint64       `json:"number"`
	Hash   string       `json:"hash"`
	Logs   []CachedLog `json:"logs"`
	Cached time.Time    `json:"cached"`
}

// CachedLog represents a cached event log
type CachedLog struct {
	Address     string   `json:"address"`
	Topics      []string `json:"topics"`
	Data        string   `json:"data"`
	BlockNumber uint64   `json:"blockNumber"`
	TxHash      string   `json:"txHash"`
	TxIndex     uint     `json:"txIndex"`
	LogIndex    uint     `json:"logIndex"`
}

// BurnScanProgress tracks scanning progress
type BurnScanProgress struct {
	StartBlock      uint64    `json:"start_block"`
	EndBlock        uint64    `json:"end_block"`
	CurrentBlock    uint64    `json:"current_block"`
	LastUpdate      time.Time `json:"last_update"`
	ScannedRanges   [][]uint64 `json:"scanned_ranges"`
	TotalBurns      int       `json:"total_burns"`
	UniqueBurners   int       `json:"unique_burners"`
	TokenAddress    string    `json:"token_address"`
	BurnAddress     string    `json:"burn_address"`
}

// NewScanBurnsCachedCommand creates the scan-burns-cached command
func NewScanBurnsCachedCommand() *cobra.Command {
	var (
		rpcURLs       []string
		tokenAddress  string
		burnAddress   string
		fromBlock     uint64
		toBlock       uint64
		cacheDir      string
		outputCSV     string
		outputJSON    string
		batchSize     uint64
		concurrent    int
		resumeFlag    bool
	)

	cmd := &cobra.Command{
		Use:   "scan-burns-cached",
		Short: "Scan token burns with local caching for resumability",
		Long: `Scans for token burns to dead addresses with local caching.
		
This command caches all fetched data locally, allowing you to:
- Resume interrupted scans
- Re-analyze data without re-fetching
- Handle RPC rate limits gracefully
- Build a local dataset for analysis`,
		Example: `  # Start a new scan with caching
  archaeology scan-burns-cached \
    --rpc https://bsc-dataseed.bnbchain.org \
    --rpc https://bsc-dataseed.nariox.org \
    --token 0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13 \
    --burn-address 0x000000000000000000000000000000000000dEaD \
    --cache-dir ./cache/zoo-burns \
    --from-block 14000000

  # Resume a previous scan
  archaeology scan-burns-cached \
    --cache-dir ./cache/zoo-burns \
    --resume`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Ensure cache directory exists
			if err := os.MkdirAll(cacheDir, 0755); err != nil {
				return fmt.Errorf("failed to create cache directory: %w", err)
			}

			progressFile := filepath.Join(cacheDir, "progress.json")
			var progress BurnScanProgress

			// Load or initialize progress
			if resumeFlag {
				data, err := os.ReadFile(progressFile)
				if err != nil {
					return fmt.Errorf("failed to load progress: %w", err)
				}
				if err := json.Unmarshal(data, &progress); err != nil {
					return fmt.Errorf("failed to parse progress: %w", err)
				}
				
				log.Printf("Resuming scan from block %d", progress.CurrentBlock)
				tokenAddress = progress.TokenAddress
				burnAddress = progress.BurnAddress
				fromBlock = progress.CurrentBlock
				toBlock = progress.EndBlock
			} else {
				// Initialize new scan
				if len(rpcURLs) == 0 {
					return fmt.Errorf("at least one RPC URL required")
				}

				// Get latest block if not specified
				if toBlock == 0 {
					client, err := ethclient.Dial(rpcURLs[0])
					if err != nil {
						return fmt.Errorf("failed to connect to RPC: %w", err)
					}
					header, err := client.HeaderByNumber(context.Background(), nil)
					if err != nil {
						return fmt.Errorf("failed to get latest block: %w", err)
					}
					toBlock = header.Number.Uint64()
				}

				progress = BurnScanProgress{
					StartBlock:   fromBlock,
					EndBlock:     toBlock,
					CurrentBlock: fromBlock,
					TokenAddress: tokenAddress,
					BurnAddress:  burnAddress,
					LastUpdate:   time.Now(),
				}
			}

			// Create RPC clients pool
			clients := make([]*ethclient.Client, 0, len(rpcURLs))
			for _, rpc := range rpcURLs {
				client, err := ethclient.Dial(rpc)
				if err != nil {
					log.Printf("Warning: failed to connect to %s: %v", rpc, err)
					continue
				}
				clients = append(clients, client)
				log.Printf("Connected to %s", rpc)
			}

			if len(clients) == 0 {
				return fmt.Errorf("failed to connect to any RPC endpoint")
			}

			// Prepare addresses
			tokenAddr := common.HexToAddress(tokenAddress)
			burnAddr := common.HexToAddress(burnAddress)

			// Transfer event topic
			transferTopic := common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

			// Create worker pool
			type workItem struct {
				fromBlock uint64
				toBlock   uint64
			}
			
			work := make(chan workItem, concurrent*2)
			results := make(chan *CachedBlock, concurrent*2)
			errors := make(chan error, concurrent)
			
			var wg sync.WaitGroup
			
			// Start workers
			for i := 0; i < concurrent; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					clientIdx := workerID % len(clients)
					
					for item := range work {
						// Check if already cached
						cacheFile := filepath.Join(cacheDir, fmt.Sprintf("blocks_%d_%d.json", item.fromBlock, item.toBlock))
						
						// Try to load from cache
						if data, err := os.ReadFile(cacheFile); err == nil {
							var cached []*CachedBlock
							if err := json.Unmarshal(data, &cached); err == nil {
								for _, block := range cached {
									results <- block
								}
								continue
							}
						}
						
						// Fetch from RPC
						client := clients[clientIdx]
						query := ethereum.FilterQuery{
							FromBlock: big.NewInt(int64(item.fromBlock)),
							ToBlock:   big.NewInt(int64(item.toBlock)),
							Addresses: []common.Address{tokenAddr},
							Topics: [][]common.Hash{
								{transferTopic},
								nil, // from (any)
								{common.BytesToHash(burnAddr.Bytes())}, // to (burn address)
							},
						}
						
						logs, err := client.FilterLogs(context.Background(), query)
						if err != nil {
							// Handle rate limits by retrying with smaller batch
							if strings.Contains(err.Error(), "limit") && item.toBlock > item.fromBlock {
								mid := (item.fromBlock + item.toBlock) / 2
								work <- workItem{item.fromBlock, mid}
								work <- workItem{mid + 1, item.toBlock}
								continue
							}
							errors <- fmt.Errorf("failed to get logs for blocks %d-%d: %w", item.fromBlock, item.toBlock, err)
							continue
						}
						
						// Group logs by block
						blockLogs := make(map[uint64][]CachedLog)
						for _, log := range logs {
							cached := CachedLog{
								Address:     log.Address.Hex(),
								Topics:      make([]string, len(log.Topics)),
								Data:        common.Bytes2Hex(log.Data),
								BlockNumber: log.BlockNumber,
								TxHash:      log.TxHash.Hex(),
								TxIndex:     log.TxIndex,
								LogIndex:    log.Index,
							}
							for i, topic := range log.Topics {
								cached.Topics[i] = topic.Hex()
							}
							blockLogs[log.BlockNumber] = append(blockLogs[log.BlockNumber], cached)
						}
						
						// Create cached blocks
						var blocks []*CachedBlock
						for blockNum, logs := range blockLogs {
							blocks = append(blocks, &CachedBlock{
								Number: blockNum,
								Logs:   logs,
								Cached: time.Now(),
							})
						}
						
						// Save to cache
						if len(blocks) > 0 || (item.toBlock-item.fromBlock) < 100 {
							// Save even empty results for small ranges to avoid re-querying
							cacheData, _ := json.Marshal(blocks)
							os.WriteFile(cacheFile, cacheData, 0644)
						}
						
						// Send results
						for _, block := range blocks {
							results <- block
						}
						
						if item.toBlock % 10000 == 0 {
							log.Printf("Worker %d: Scanned up to block %d", workerID, item.toBlock)
						}
					}
				}(i)
			}
			
			// Process results
			burnsByAddress := make(map[string]*big.Int)
			allBurns := []*CachedLog{}
			
			go func() {
				for result := range results {
					for _, log := range result.Logs {
						// Decode burn amount from data
						amount := new(big.Int).SetBytes(common.FromHex(log.Data))
						
						// Extract from address (remove padding)
						fromAddr := common.HexToAddress(log.Topics[1][26:]).Hex()
						
						if burnsByAddress[fromAddr] == nil {
							burnsByAddress[fromAddr] = new(big.Int)
						}
						burnsByAddress[fromAddr].Add(burnsByAddress[fromAddr], amount)
						
						allBurns = append(allBurns, &log)
					}
					
					// Update progress
					if result.Number > progress.CurrentBlock {
						progress.CurrentBlock = result.Number
						progress.TotalBurns = len(allBurns)
						progress.UniqueBurners = len(burnsByAddress)
					}
				}
			}()
			
			// Generate work items
			go func() {
				for block := progress.CurrentBlock; block <= progress.EndBlock; block += batchSize {
					endBlock := block + batchSize - 1
					if endBlock > progress.EndBlock {
						endBlock = progress.EndBlock
					}
					work <- workItem{block, endBlock}
				}
				close(work)
			}()
			
			// Wait for workers
			wg.Wait()
			close(results)
			close(errors)
			
			// Save final progress
			progress.LastUpdate = time.Now()
			progressData, _ := json.MarshalIndent(progress, "", "  ")
			os.WriteFile(progressFile, progressData, 0644)
			
			// Generate report
			log.Printf("\n=== Burn Scan Complete ===")
			log.Printf("Total burns found: %d", len(allBurns))
			log.Printf("Unique burners: %d", len(burnsByAddress))
			
			// Calculate total burned
			totalBurned := new(big.Int)
			for _, amount := range burnsByAddress {
				totalBurned.Add(totalBurned, amount)
			}
			
			// Save results
			if outputJSON != "" {
				results := map[string]interface{}{
					"scan_info": map[string]interface{}{
						"token_address": tokenAddress,
						"burn_address":  burnAddress,
						"from_block":    fromBlock,
						"to_block":      toBlock,
						"cached_at":     cacheDir,
					},
					"summary": map[string]interface{}{
						"total_burns":      len(allBurns),
						"unique_burners":   len(burnsByAddress),
						"total_burned_wei": totalBurned.String(),
						"total_burned_tokens": new(big.Float).Quo(
							new(big.Float).SetInt(totalBurned),
							new(big.Float).SetInt(big.NewInt(1e18)),
						),
					},
					"burns_by_address": burnsByAddress,
				}
				
				data, _ := json.MarshalIndent(results, "", "  ")
				os.WriteFile(outputJSON, data, 0644)
				log.Printf("Saved results to %s", outputJSON)
			}
			
			if outputCSV != "" {
				file, _ := os.Create(outputCSV)
				defer file.Close()
				
				fmt.Fprintf(file, "address,total_burned_wei,total_burned_tokens\n")
				
				// Sort by amount
				type burn struct {
					address string
					amount  *big.Int
				}
				var burns []burn
				for addr, amount := range burnsByAddress {
					burns = append(burns, burn{addr, amount})
				}
				sort.Slice(burns, func(i, j int) bool {
					return burns[i].amount.Cmp(burns[j].amount) > 0
				})
				
				for _, b := range burns {
					tokens := new(big.Float).Quo(
						new(big.Float).SetInt(b.amount),
						new(big.Float).SetInt(big.NewInt(1e18)),
					)
					fmt.Fprintf(file, "%s,%s,%s\n", b.address, b.amount.String(), tokens.Text('f', 6))
				}
				
				log.Printf("Saved CSV to %s", outputCSV)
			}
			
			// Show top burners
			fmt.Println("\nTop 10 Burners:")
			type burn struct {
				address string
				amount  *big.Int
			}
			var topBurns []burn
			for addr, amount := range burnsByAddress {
				topBurns = append(topBurns, burn{addr, new(big.Int).Set(amount)})
			}
			sort.Slice(topBurns, func(i, j int) bool {
				return topBurns[i].amount.Cmp(topBurns[j].amount) > 0
			})
			
			for i := 0; i < 10 && i < len(topBurns); i++ {
				tokens := new(big.Float).Quo(
					new(big.Float).SetInt(topBurns[i].amount),
					new(big.Float).SetInt(big.NewInt(1e18)),
				)
				fmt.Printf("%d. %s: %s tokens\n", i+1, topBurns[i].address, tokens.Text('f', 2))
			}
			
			return nil
		},
	}

	// Add flags
	cmd.Flags().StringSliceVar(&rpcURLs, "rpc", nil, "RPC endpoints (can specify multiple)")
	cmd.Flags().StringVar(&tokenAddress, "token", "", "Token contract address")
	cmd.Flags().StringVar(&burnAddress, "burn-address", "0x000000000000000000000000000000000000dEaD", "Burn address")
	cmd.Flags().Uint64Var(&fromBlock, "from-block", 0, "Start block")
	cmd.Flags().Uint64Var(&toBlock, "to-block", 0, "End block (0 = latest)")
	cmd.Flags().StringVar(&cacheDir, "cache-dir", "./cache", "Cache directory")
	cmd.Flags().StringVar(&outputCSV, "output", "", "Output CSV file")
	cmd.Flags().StringVar(&outputJSON, "output-json", "", "Output JSON file")
	cmd.Flags().Uint64Var(&batchSize, "batch-size", 1000, "Blocks per batch")
	cmd.Flags().IntVar(&concurrent, "concurrent", 5, "Concurrent workers")
	cmd.Flags().BoolVar(&resumeFlag, "resume", false, "Resume previous scan")

	return cmd
}