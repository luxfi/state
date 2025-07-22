package main

import (
	"encoding/csv"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
)

var (
	dbPath          = flag.String("db-path", "", "Path to 7777 database")
	outputPath      = flag.String("output", "exports/7777-accounts.csv", "Output CSV file")
	excludeTreasury = flag.String("exclude-treasury", "", "Treasury address to exclude")
)

func main() {
	flag.Parse()

	if *dbPath == "" {
		log.Fatal("--db-path is required")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Open the database
	db, err := leveldb.New(*dbPath, 0, 0, "", false)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create CSV writer
	file, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"address", "balance_wei", "balance_lux", "validator_eligible"}); err != nil {
		log.Fatalf("Failed to write header: %v", err)
	}

	// Parse treasury address if provided
	var treasuryAddr common.Address
	if *excludeTreasury != "" {
		treasuryAddr = common.HexToAddress(*excludeTreasury)
	}

	// Account state prefix (0x26)
	accountPrefix := []byte{0x26}
	minValidatorStake := new(big.Int).Mul(big.NewInt(1000000), big.NewInt(1e18)) // 1M LUX

	exportedCount := 0
	treasurySkipped := false
	totalBalance := new(big.Int)

	// Iterate through all accounts
	iter := db.NewIterator(accountPrefix, nil)
	defer iter.Release()

	for iter.Next() {
		// Extract address from key
		key := iter.Key()
		if len(key) < 33 { // prefix(1) + hash(32)
			continue
		}

		// Get the account address hash
		addrHash := key[1:]
		
		// For this example, we'll use the hash as the address
		// In production, you'd need to maintain a reverse mapping
		addrStr := "0x" + hex.EncodeToString(addrHash[:20])
		addr := common.HexToAddress(addrStr)

		// Skip treasury if specified
		if *excludeTreasury != "" && addr == treasuryAddr {
			treasurySkipped = true
			continue
		}

		// Decode account data (simplified - actual format is RLP encoded)
		// This is a placeholder - you'd need proper RLP decoding
		value := iter.Value()
		if len(value) < 32 {
			continue
		}

		// Extract balance (this is simplified - actual implementation needs RLP)
		balance := new(big.Int).SetBytes(value[:32])
		if balance.Sign() == 0 {
			continue
		}

		// Calculate balance in LUX
		balanceLux := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			new(big.Float).SetInt(big.NewInt(1e18)),
		)

		// Check if eligible for validator
		validatorEligible := "false"
		if balance.Cmp(minValidatorStake) >= 0 {
			validatorEligible = "true"
		}

		// Write to CSV
		record := []string{
			addr.Hex(),
			balance.String(),
			fmt.Sprintf("%.6f", balanceLux),
			validatorEligible,
		}
		
		if err := writer.Write(record); err != nil {
			log.Printf("Failed to write record for %s: %v", addr.Hex(), err)
			continue
		}

		exportedCount++
		totalBalance.Add(totalBalance, balance)
	}

	if err := iter.Error(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	// Print summary
	fmt.Printf("\nExport Summary:\n")
	fmt.Printf("- Accounts exported: %d\n", exportedCount)
	if treasurySkipped {
		fmt.Printf("- Treasury excluded: %s\n", *excludeTreasury)
	}
	totalLux := new(big.Float).Quo(
		new(big.Float).SetInt(totalBalance),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	fmt.Printf("- Total balance: %.6f LUX\n", totalLux)
	fmt.Printf("- Output file: %s\n", *outputPath)
}