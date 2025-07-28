package main

import (
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	
	"github.com/cockroachdb/pebble"
	_ "github.com/lib/pq"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: export-blocks-from-postgres <output-db>")
		fmt.Println()
		fmt.Println("This tool exports block mappings from PostgreSQL to PebbleDB")
		os.Exit(1)
	}

	outputPath := os.Args[1]

	// Connect to PostgreSQL
	connStr := "host=192.168.1.99 user=postgres dbname=explorer_luxnet sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Open output PebbleDB
	pdb, err := pebble.Open(outputPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open PebbleDB: %v", err)
	}
	defer pdb.Close()

	fmt.Println("=== Exporting Block Mappings from PostgreSQL ===")

	// Query all blocks
	rows, err := db.Query(`
		SELECT number, hash 
		FROM blocks 
		ORDER BY number ASC
	`)
	if err != nil {
		log.Fatalf("Failed to query blocks: %v", err)
	}
	defer rows.Close()

	batch := pdb.NewBatch()
	count := 0
	var highestNum int64
	var highestHash []byte

	for rows.Next() {
		var number int64
		var hash []byte
		
		if err := rows.Scan(&number, &hash); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}

		// Create number->hash mapping (evm + n + 8-byte number)
		numBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(numBytes, uint64(number))
		
		nKey := append([]byte("evmn"), numBytes...)
		if err := batch.Set(nKey, hash, pebble.Sync); err != nil {
			log.Fatalf("Failed to set n mapping: %v", err)
		}

		// Create hash->number mapping (evm + H + hash)
		HKey := append([]byte("evmH"), hash...)
		if err := batch.Set(HKey, numBytes, pebble.Sync); err != nil {
			log.Fatalf("Failed to set H mapping: %v", err)
		}

		count++
		highestNum = number
		highestHash = make([]byte, len(hash))
		copy(highestHash, hash)

		// Show progress
		if count%10000 == 0 {
			fmt.Printf("  Exported %d blocks (up to %d)...\n", count, number)
		}

		// Commit batch periodically
		if count%100000 == 0 {
			if err := batch.Commit(pebble.Sync); err != nil {
				log.Fatalf("Failed to commit batch: %v", err)
			}
			batch = pdb.NewBatch()
		}
	}

	// Final batch commit
	if err := batch.Commit(pebble.Sync); err != nil {
		log.Fatalf("Failed to commit final batch: %v", err)
	}

	// Set head pointers
	fmt.Printf("\nSetting head pointers to block %d (hash: %s)\n", highestNum, hex.EncodeToString(highestHash))
	
	// Set various head keys
	headKeys := [][]byte{
		[]byte("evmLastBlock"),
		[]byte("evmLastHeader"),
		[]byte("evmLastFast"),
		[]byte("evmh"), // Single letter head header
		[]byte("evmH"), // Single letter head block
		[]byte("evmF"), // Single letter head fast
	}

	for _, key := range headKeys {
		if err := pdb.Set(key, highestHash, pebble.Sync); err != nil {
			log.Printf("Failed to set head key %s: %v", string(key), err)
		}
	}

	fmt.Printf("\n=== Export Complete ===\n")
	fmt.Printf("Total blocks exported: %d\n", count)
	fmt.Printf("Highest block: %d\n", highestNum)
	fmt.Printf("Highest hash: %s\n", hex.EncodeToString(highestHash))
}