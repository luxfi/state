package scanner_test

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/luxfi/genesis/pkg/scanner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data
var (
	testTokenAddress = "0x09e2b83fe5485a7c8beaa5dffd1d324a2b2d5c13"
	testNFTAddress   = "0x5bb68cf06289d54efde25155c88003be685356a8"
	testBurnAddress  = scanner.DeadAddress
	testFromBlock    = uint64(20000000)
	testToBlock      = uint64(20001000)
)

func TestTokenBurnScanner(t *testing.T) {
	t.Run("NewTokenBurnScanner", func(t *testing.T) {
		config := &scanner.TokenBurnScanConfig{
			RPC:          "https://bsc-dataseed.binance.org/",
			TokenAddress: testTokenAddress,
			BurnAddress:  testBurnAddress,
			FromBlock:    testFromBlock,
			ToBlock:      testToBlock,
		}

		scanner, err := scanner.NewTokenBurnScanner(config)
		require.NoError(t, err)
		require.NotNil(t, scanner)
		defer scanner.Close()
	})

	t.Run("FilterBurnsByAmount", func(t *testing.T) {
		// Create test burns
		burns := []scanner.TokenBurn{
			{
				From:   "0xaddr1",
				Amount: "1000000000000000000", // 1 token
			},
			{
				From:   "0xaddr2",
				Amount: "5000000000000000000", // 5 tokens
			},
			{
				From:   "0xaddr3",
				Amount: "500000000000000000", // 0.5 tokens
			},
		}

		// Filter by minimum 1 token
		minAmount := big.NewInt(1e18)
		filtered := scanner.FilterBurnsByAmount(burns, minAmount)

		assert.Equal(t, 2, len(filtered))
		assert.Equal(t, "0xaddr1", filtered[0].From)
		assert.Equal(t, "0xaddr2", filtered[1].From)
	})

	t.Run("GetUniqueBurners", func(t *testing.T) {
		burns := []scanner.TokenBurn{
			{From: "0xAddr1"},
			{From: "0xaddr1"}, // Same address, different case
			{From: "0xAddr2"},
			{From: "0xAddr3"},
			{From: "0xaddr2"}, // Same address, different case
		}

		unique := scanner.GetUniqueBurners(burns)
		assert.Equal(t, 3, len(unique))
	})
}

func TestTokenTransferScanner(t *testing.T) {
	t.Run("GetBalanceChanges", func(t *testing.T) {
		transfers := []scanner.TokenTransfer{
			{
				From:   "0xaddr1",
				To:     "0xaddr2",
				Amount: "1000000000000000000", // 1 token
			},
			{
				From:   "0xaddr2",
				To:     "0xaddr3",
				Amount: "500000000000000000", // 0.5 tokens
			},
			{
				From:   "0xaddr1",
				To:     "0xaddr3",
				Amount: "2000000000000000000", // 2 tokens
			},
		}

		balances := scanner.GetBalanceChanges(transfers)

		// addr1: -3 tokens
		addr1Balance := balances["0xaddr1"]
		expectedAddr1 := new(big.Int).Mul(big.NewInt(-3), big.NewInt(1e18))
		assert.Equal(t, 0, addr1Balance.Cmp(expectedAddr1))

		// addr2: +0.5 tokens (received 1, sent 0.5)
		addr2Balance := balances["0xaddr2"]
		expectedAddr2 := new(big.Int).Mul(big.NewInt(5), big.NewInt(1e17))
		assert.Equal(t, 0, addr2Balance.Cmp(expectedAddr2))

		// addr3: +2.5 tokens
		addr3Balance := balances["0xaddr3"]
		expectedAddr3 := new(big.Int).Mul(big.NewInt(25), big.NewInt(1e17))
		assert.Equal(t, 0, addr3Balance.Cmp(expectedAddr3))
	})
}

func TestNFTHolderScanner(t *testing.T) {
	t.Run("FilterHoldersByMinTokens", func(t *testing.T) {
		holders := []scanner.NFTHolder{
			{Address: "0xaddr1", TokenCount: 1},
			{Address: "0xaddr2", TokenCount: 5},
			{Address: "0xaddr3", TokenCount: 10},
			{Address: "0xaddr4", TokenCount: 3},
		}

		filtered := scanner.FilterHoldersByMinTokens(holders, 5)
		assert.Equal(t, 2, len(filtered))
		assert.Equal(t, "0xaddr2", filtered[0].Address)
		assert.Equal(t, "0xaddr3", filtered[1].Address)
	})

	t.Run("GetHolderDistribution", func(t *testing.T) {
		holders := []scanner.NFTHolder{
			{Address: "0xaddr1", TokenCount: 1},
			{Address: "0xaddr2", TokenCount: 1},
			{Address: "0xaddr3", TokenCount: 5},
			{Address: "0xaddr4", TokenCount: 15},
			{Address: "0xaddr5", TokenCount: 25},
			{Address: "0xaddr6", TokenCount: 150},
		}

		distribution := scanner.GetHolderDistribution(holders)

		assert.Equal(t, 2, distribution["1 token"])
		assert.Equal(t, 1, distribution["2-5 tokens"])
		assert.Equal(t, 1, distribution["11-20 tokens"])
		assert.Equal(t, 1, distribution["21-50 tokens"])
		assert.Equal(t, 1, distribution["100+ tokens"])
	})
}

func TestExportFunctions(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ExportTokenBurnsToCSV", func(t *testing.T) {
		burns := []scanner.TokenBurn{
			{
				TxHash:      "0x123",
				BlockNumber: 12345,
				Timestamp:   time.Now(),
				From:        "0xaddr1",
				To:          scanner.DeadAddress,
				Amount:      "1000000000000000000",
				TokenAddr:   testTokenAddress,
				LogIndex:    0,
			},
		}

		csvPath := filepath.Join(tempDir, "burns.csv")
		err := scanner.ExportTokenBurnsToCSV(burns, csvPath)
		require.NoError(t, err)

		// Verify CSV was created
		_, err = os.Stat(csvPath)
		require.NoError(t, err)

		// Read and verify CSV content
		file, err := os.Open(csvPath)
		require.NoError(t, err)
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 2) // Header + 1 record
		assert.Equal(t, "TxHash", records[0][0])
		assert.Equal(t, "0x123", records[1][0])
	})

	t.Run("ExportNFTHoldersToCSV", func(t *testing.T) {
		holders := []scanner.NFTHolder{
			{
				Address:    "0xaddr1",
				TokenCount: 5,
				TokenIDs:   []string{"1", "2", "3", "4", "5"},
			},
		}

		metadata := map[string]string{
			"Contract": testNFTAddress,
		}

		csvPath := filepath.Join(tempDir, "holders.csv")
		err := scanner.ExportNFTHoldersToCSV(holders, csvPath, metadata)
		require.NoError(t, err)

		// Verify CSV was created
		_, err = os.Stat(csvPath)
		require.NoError(t, err)
	})

	t.Run("ExportToJSON", func(t *testing.T) {
		data := map[string]interface{}{
			"test":  "data",
			"count": 123,
			"items": []string{"a", "b", "c"},
		}

		jsonPath := filepath.Join(tempDir, "data.json")
		err := scanner.ExportToJSON(data, jsonPath)
		require.NoError(t, err)

		// Read and verify JSON
		content, err := os.ReadFile(jsonPath)
		require.NoError(t, err)

		var loaded map[string]interface{}
		err = json.Unmarshal(content, &loaded)
		require.NoError(t, err)
		assert.Equal(t, "data", loaded["test"])
		assert.Equal(t, float64(123), loaded["count"])
	})

	t.Run("ExportBurnSummaryToCSV", func(t *testing.T) {
		burnsByAddress := map[string]*big.Int{
			"0xaddr1": big.NewInt(1e18),
			"0xaddr2": big.NewInt(5e18),
			"0xaddr3": big.NewInt(2e18),
		}

		csvPath := filepath.Join(tempDir, "burn_summary.csv")
		err := scanner.ExportBurnSummaryToCSV(burnsByAddress, csvPath)
		require.NoError(t, err)

		// Verify CSV was created and has correct sorting
		file, err := os.Open(csvPath)
		require.NoError(t, err)
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 4) // Header + 3 records

		// Should be sorted by amount descending
		assert.Equal(t, "0xaddr2", records[1][0]) // 5 tokens
		assert.Equal(t, "0xaddr3", records[2][0]) // 2 tokens
		assert.Equal(t, "0xaddr1", records[3][0]) // 1 token
	})

	t.Run("GenerateSummaryReport", func(t *testing.T) {
		sections := map[string]string{
			"Summary": "This is a test summary",
			"Details": "These are test details",
			"Results": "These are test results",
		}

		reportPath := filepath.Join(tempDir, "report.txt")
		err := scanner.GenerateSummaryReport(reportPath, sections)
		require.NoError(t, err)

		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Summary")
		assert.Contains(t, contentStr, "This is a test summary")
		assert.Contains(t, contentStr, "Details")
		assert.Contains(t, contentStr, "Results")
	})
}

func TestCrossChainBalances(t *testing.T) {
	t.Run("ChainConfig", func(t *testing.T) {
		config := scanner.ChainConfig{
			Name:         "BSC",
			ChainID:      56,
			RPC:          "https://bsc-dataseed.binance.org/",
			TokenAddress: testTokenAddress,
		}

		assert.Equal(t, "BSC", config.Name)
		assert.Equal(t, int64(56), config.ChainID)
	})

	t.Run("ExportCrossChainBalancesToCSV", func(t *testing.T) {
		tempDir := t.TempDir()

		balances := map[string][]scanner.CrossChainBalance{
			"0xaddr1": {
				{
					Address:      "0xaddr1",
					Balance:      "1000000000000000000",
					ChainID:      56,
					TokenAddress: testTokenAddress,
					BlockNumber:  12345,
				},
				{
					Address:      "0xaddr1",
					Balance:      "2000000000000000000",
					ChainID:      200200,
					TokenAddress: testTokenAddress,
					BlockNumber:  54321,
				},
			},
		}

		csvPath := filepath.Join(tempDir, "cross_chain.csv")
		err := scanner.ExportCrossChainBalancesToCSV(balances, csvPath)
		require.NoError(t, err)

		// Verify CSV
		file, err := os.Open(csvPath)
		require.NoError(t, err)
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		require.NoError(t, err)
		require.Len(t, records, 3) // Header + 2 records

		// Check chain names
		assert.Equal(t, "BSC", records[1][2])
		assert.Equal(t, "Zoo Mainnet", records[2][2])
	})
}

// Integration test for the complete flow
func TestZooAnalysisIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	t.Run("CompleteZooAnalysisFlow", func(t *testing.T) {
		// This test would require a mock RPC or real connection
		// For now, we test the data structures and flow

		// Test burn aggregation
		burns := []scanner.TokenBurn{
			{From: "0xaddr1", Amount: "1000000000000000000"},
			{From: "0xaddr1", Amount: "2000000000000000000"},
			{From: "0xaddr2", Amount: "5000000000000000000"},
		}

		// Aggregate burns
		burnsByAddress := make(map[string]*big.Int)
		for _, burn := range burns {
			addr := burn.From
			amount := new(big.Int)
			amount.SetString(burn.Amount, 10)

			if existing, ok := burnsByAddress[addr]; ok {
				existing.Add(existing, amount)
			} else {
				burnsByAddress[addr] = amount
			}
		}

		// Verify aggregation
		assert.Equal(t, 2, len(burnsByAddress))

		addr1Total := burnsByAddress["0xaddr1"]
		expectedAddr1 := new(big.Int).Mul(big.NewInt(3), big.NewInt(1e18))
		assert.Equal(t, 0, addr1Total.Cmp(expectedAddr1))

		// Test ZOO per EGG calculation
		eggCount := 10
		zooPerEgg := 4200000
		totalZoo := eggCount * zooPerEgg
		assert.Equal(t, 42000000, totalZoo)
	})
}

// Benchmark tests
func BenchmarkFilterBurnsByAmount(b *testing.B) {
	// Create 1000 test burns
	burns := make([]scanner.TokenBurn, 1000)
	for i := range burns {
		burns[i] = scanner.TokenBurn{
			From:   fmt.Sprintf("0xaddr%d", i),
			Amount: fmt.Sprintf("%d000000000000000000", i+1),
		}
	}

	minAmount := new(big.Int).Mul(big.NewInt(500), big.NewInt(1e18))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.FilterBurnsByAmount(burns, minAmount)
	}
}

func BenchmarkGetBalanceChanges(b *testing.B) {
	// Create 1000 test transfers
	transfers := make([]scanner.TokenTransfer, 1000)
	for i := range transfers {
		transfers[i] = scanner.TokenTransfer{
			From:   fmt.Sprintf("0xaddr%d", i%100),
			To:     fmt.Sprintf("0xaddr%d", (i+1)%100),
			Amount: "1000000000000000000",
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scanner.GetBalanceChanges(transfers)
	}
}
