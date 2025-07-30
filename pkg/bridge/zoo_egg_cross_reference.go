package bridge

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	// ZOO per EGG NFT
	ZooPerEggNFT = 4200000

	// Key addresses
	EggPurchaseAddress = "0x28dad8427f127664365109c4a9406c8bc7844718"
	BurnAddress        = "0x000000000000000000000000000000000000dEaD"
)

// ZooEggPurchase represents a ZOO transfer for EGG NFT purchase
type ZooEggPurchase struct {
	TxHash       string    `json:"txHash"`
	BlockNumber  uint64    `json:"blockNumber"`
	Timestamp    time.Time `json:"timestamp"`
	From         string    `json:"from"`
	To           string    `json:"to"`
	Amount       string    `json:"amount"`
	ExpectedEggs int       `json:"expectedEggs"`
	ActualEggs   int       `json:"actualEggs,omitempty"`
	Matched      bool      `json:"matched"`
}

// ZooBurn represents a ZOO burn to dead address
type ZooBurn struct {
	TxHash           string    `json:"txHash"`
	BlockNumber      uint64    `json:"blockNumber"`
	Timestamp        time.Time `json:"timestamp"`
	From             string    `json:"from"`
	Amount           string    `json:"amount"`
	DeliveredMainnet bool      `json:"deliveredMainnet"`
	MainnetBalance   string    `json:"mainnetBalance,omitempty"`
}

// ZooEggCrossReference contains the complete cross-reference data
type ZooEggCrossReference struct {
	EggPurchases     []ZooEggPurchase  `json:"eggPurchases"`
	ZooBurns         []ZooBurn         `json:"zooBurns"`
	EggNFTHolders    map[string]int    `json:"eggNftHolders"`
	MainnetBalances  map[string]string `json:"mainnetBalances"`
	ValidationReport *ValidationReport `json:"validationReport"`
}

// ValidationReport contains validation results
type ValidationReport struct {
	TotalPurchases      int      `json:"totalPurchases"`
	TotalExpectedEggs   int      `json:"totalExpectedEggs"`
	TotalActualEggs     int      `json:"totalActualEggs"`
	MatchedPurchases    int      `json:"matchedPurchases"`
	MismatchedPurchases int      `json:"mismatchedPurchases"`
	UnexpectedHolders   []string `json:"unexpectedHolders"`
	TotalBurns          int      `json:"totalBurns"`
	TotalBurnedAmount   string   `json:"totalBurnedAmount"`
	DeliveredBurns      int      `json:"deliveredBurns"`
	UndeliveredBurns    int      `json:"undeliveredBurns"`
}

// ScanZooEggPurchases scans for all ZOO transfers to the EGG purchase address
func ScanZooEggPurchases(client *ethclient.Client, zooTokenAddress string, fromBlock, toBlock uint64) ([]ZooEggPurchase, error) {
	ctx := context.Background()

	// Parse ERC20 ABI for Transfer events
	contractABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	purchases := []ZooEggPurchase{}
	purchaseAddr := common.HexToAddress(EggPurchaseAddress)
	tokenAddr := common.HexToAddress(zooTokenAddress)

	// Get Transfer event signature
	transferEventSig := contractABI.Events["Transfer"].ID

	// Scan in chunks
	chunkSize := uint64(5000)
	for startBlock := fromBlock; startBlock <= toBlock; startBlock += chunkSize {
		endBlock := startBlock + chunkSize - 1
		if endBlock > toBlock {
			endBlock = toBlock
		}

		// Filter for transfers TO the purchase address
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{tokenAddr},
			Topics: [][]common.Hash{
				{transferEventSig},
				nil, // from (any)
				{common.BytesToHash(purchaseAddr.Bytes())}, // to (purchase address)
			},
		}

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}

		// Process transfers
		for _, vLog := range logs {
			var from, to common.Address
			var value *big.Int

			// Parse indexed topics
			if len(vLog.Topics) >= 3 {
				from = common.HexToAddress(vLog.Topics[1].Hex())
				to = common.HexToAddress(vLog.Topics[2].Hex())
			}

			// Parse value from data
			if len(vLog.Data) >= 32 {
				value = new(big.Int).SetBytes(vLog.Data)
			} else {
				continue
			}

			// Get block details for timestamp
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
			if err != nil {
				log.Printf("Warning: failed to get block %d: %v", vLog.BlockNumber, err)
			}

			// Calculate expected eggs
			zooAmount := new(big.Int).Set(value)
			zooPerEgg := new(big.Int).Mul(big.NewInt(ZooPerEggNFT), big.NewInt(1e18)) // Assuming 18 decimals
			expectedEggs := new(big.Int).Div(zooAmount, zooPerEgg).Int64()

			purchase := ZooEggPurchase{
				TxHash:       vLog.TxHash.Hex(),
				BlockNumber:  vLog.BlockNumber,
				Timestamp:    time.Unix(int64(block.Time()), 0),
				From:         from.Hex(),
				To:           to.Hex(),
				Amount:       value.String(),
				ExpectedEggs: int(expectedEggs),
			}

			purchases = append(purchases, purchase)
		}

		if (endBlock-startBlock)%50000 == 0 {
			log.Printf("Scanned purchases up to block %d/%d", endBlock, toBlock)
		}
	}

	return purchases, nil
}

// ScanZooBurns scans for all ZOO burns to the dead address
func ScanZooBurns(client *ethclient.Client, zooTokenAddress string, fromBlock, toBlock uint64) ([]ZooBurn, error) {
	ctx := context.Background()

	// Parse ERC20 ABI
	contractABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse ABI: %w", err)
	}

	burns := []ZooBurn{}
	deadAddr := common.HexToAddress(BurnAddress)
	tokenAddr := common.HexToAddress(zooTokenAddress)

	// Get Transfer event signature
	transferEventSig := contractABI.Events["Transfer"].ID

	// Scan in chunks
	chunkSize := uint64(5000)
	for startBlock := fromBlock; startBlock <= toBlock; startBlock += chunkSize {
		endBlock := startBlock + chunkSize - 1
		if endBlock > toBlock {
			endBlock = toBlock
		}

		// Filter for transfers TO the dead address
		query := ethereum.FilterQuery{
			FromBlock: big.NewInt(int64(startBlock)),
			ToBlock:   big.NewInt(int64(endBlock)),
			Addresses: []common.Address{tokenAddr},
			Topics: [][]common.Hash{
				{transferEventSig},
				nil,                                    // from (any)
				{common.BytesToHash(deadAddr.Bytes())}, // to (dead address)
			},
		}

		logs, err := client.FilterLogs(ctx, query)
		if err != nil {
			log.Printf("Warning: failed to get logs for blocks %d-%d: %v", startBlock, endBlock, err)
			continue
		}

		// Process burns
		for _, vLog := range logs {
			var from common.Address
			var value *big.Int

			// Parse indexed topics
			if len(vLog.Topics) >= 3 {
				from = common.HexToAddress(vLog.Topics[1].Hex())
			}

			// Parse value
			if len(vLog.Data) >= 32 {
				value = new(big.Int).SetBytes(vLog.Data)
			} else {
				continue
			}

			// Get block details
			block, err := client.BlockByNumber(ctx, big.NewInt(int64(vLog.BlockNumber)))
			if err != nil {
				log.Printf("Warning: failed to get block %d: %v", vLog.BlockNumber, err)
			}

			burn := ZooBurn{
				TxHash:      vLog.TxHash.Hex(),
				BlockNumber: vLog.BlockNumber,
				Timestamp:   time.Unix(int64(block.Time()), 0),
				From:        from.Hex(),
				Amount:      value.String(),
			}

			burns = append(burns, burn)
		}

		if (endBlock-startBlock)%50000 == 0 {
			log.Printf("Scanned burns up to block %d/%d", endBlock, toBlock)
		}
	}

	return burns, nil
}

// CrossReferenceWithMainnet checks which burners have received ZOO on mainnet
func CrossReferenceWithMainnet(burns []ZooBurn, mainnetBalances map[string]string) {
	for i := range burns {
		burn := &burns[i]
		if balance, exists := mainnetBalances[strings.ToLower(burn.From)]; exists {
			burn.DeliveredMainnet = true
			burn.MainnetBalance = balance
		}
	}
}

// ValidateEggPurchases validates egg purchases against actual NFT holdings
func ValidateEggPurchases(purchases []ZooEggPurchase, eggHolders map[string]int) *ValidationReport {
	report := &ValidationReport{
		TotalPurchases: len(purchases),
	}

	// Group purchases by address
	purchasesByAddress := make(map[string]int)
	for i := range purchases {
		purchase := &purchases[i]
		addr := strings.ToLower(purchase.From)
		purchasesByAddress[addr] += purchase.ExpectedEggs
		report.TotalExpectedEggs += purchase.ExpectedEggs

		// Check if holder has EGGs
		if actualEggs, exists := eggHolders[addr]; exists {
			purchase.ActualEggs = actualEggs
			if actualEggs >= purchase.ExpectedEggs {
				purchase.Matched = true
				report.MatchedPurchases++
			} else {
				report.MismatchedPurchases++
			}
		} else {
			report.MismatchedPurchases++
		}
	}

	// Count total actual eggs
	for _, count := range eggHolders {
		report.TotalActualEggs += count
	}

	// Find unexpected holders (have EGGs but no purchase record)
	for addr, count := range eggHolders {
		if _, purchased := purchasesByAddress[addr]; !purchased && count > 0 {
			report.UnexpectedHolders = append(report.UnexpectedHolders, addr)
		}
	}

	return report
}

// ExportToCSV exports all data to CSV files
func ExportZooEggDataToCSV(data *ZooEggCrossReference, basePath string) error {
	// Export EGG purchases
	purchaseFile, err := os.Create(basePath + "_egg_purchases.csv")
	if err != nil {
		return err
	}
	defer purchaseFile.Close()

	purchaseWriter := csv.NewWriter(purchaseFile)
	defer purchaseWriter.Flush()

	// Write header
	purchaseWriter.Write([]string{
		"TxHash", "BlockNumber", "Timestamp", "From", "To",
		"ZooAmount", "ExpectedEggs", "ActualEggs", "Matched",
	})

	// Sort by timestamp
	sort.Slice(data.EggPurchases, func(i, j int) bool {
		return data.EggPurchases[i].Timestamp.Before(data.EggPurchases[j].Timestamp)
	})

	for _, p := range data.EggPurchases {
		purchaseWriter.Write([]string{
			p.TxHash,
			fmt.Sprintf("%d", p.BlockNumber),
			p.Timestamp.Format("2006-01-02 15:04:05"),
			p.From,
			p.To,
			p.Amount,
			fmt.Sprintf("%d", p.ExpectedEggs),
			fmt.Sprintf("%d", p.ActualEggs),
			fmt.Sprintf("%v", p.Matched),
		})
	}

	// Export burns
	burnFile, err := os.Create(basePath + "_zoo_burns.csv")
	if err != nil {
		return err
	}
	defer burnFile.Close()

	burnWriter := csv.NewWriter(burnFile)
	defer burnWriter.Flush()

	// Write header
	burnWriter.Write([]string{
		"TxHash", "BlockNumber", "Timestamp", "From",
		"BurnedAmount", "DeliveredMainnet", "MainnetBalance",
	})

	// Sort by timestamp
	sort.Slice(data.ZooBurns, func(i, j int) bool {
		return data.ZooBurns[i].Timestamp.Before(data.ZooBurns[j].Timestamp)
	})

	for _, b := range data.ZooBurns {
		burnWriter.Write([]string{
			b.TxHash,
			fmt.Sprintf("%d", b.BlockNumber),
			b.Timestamp.Format("2006-01-02 15:04:05"),
			b.From,
			b.Amount,
			fmt.Sprintf("%v", b.DeliveredMainnet),
			b.MainnetBalance,
		})
	}

	// Export EGG NFT holders
	holderFile, err := os.Create(basePath + "_egg_nft_holders.csv")
	if err != nil {
		return err
	}
	defer holderFile.Close()

	holderWriter := csv.NewWriter(holderFile)
	defer holderWriter.Flush()

	// Write header
	holderWriter.Write([]string{"Address", "EggCount", "ZooEquivalent"})

	// Sort holders
	type holder struct {
		addr  string
		count int
	}
	holders := []holder{}
	for addr, count := range data.EggNFTHolders {
		holders = append(holders, holder{addr, count})
	}
	sort.Slice(holders, func(i, j int) bool {
		if holders[i].count != holders[j].count {
			return holders[i].count > holders[j].count
		}
		return holders[i].addr < holders[j].addr
	})

	for _, h := range holders {
		zooEquiv := h.count * ZooPerEggNFT
		holderWriter.Write([]string{
			h.addr,
			fmt.Sprintf("%d", h.count),
			fmt.Sprintf("%d", zooEquiv),
		})
	}

	return nil
}
