package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	
	"github.com/cockroachdb/pebble"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: import-blocks-from-csv <csv-file> <output-db>")
		os.Exit(1)
	}

	csvPath := os.Args[1]
	outputPath := os.Args[2]

	// Open CSV file
	file, err := os.Open(csvPath)
	if err != nil {
		log.Fatalf("Failed to open CSV: %v", err)
	}
	defer file.Close()

	// Open output PebbleDB
	pdb, err := pebble.Open(outputPath, &pebble.Options{})
	if err != nil {
		log.Fatalf("Failed to open PebbleDB: %v", err)
	}
	defer pdb.Close()

	fmt.Println("=== Importing Block Mappings from CSV ===")

	scanner := bufio.NewScanner(file)
	batch := pdb.NewBatch()
	count := 0
	var highestNum uint64
	var highestHash []byte

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}

		// Parse number
		number, err := strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			log.Printf("Failed to parse number %s: %v", parts[0], err)
			continue
		}

		// Parse hash (remove \x prefix if present)
		hashStr := strings.TrimPrefix(parts[1], "\\x")
		hash, err := hex.DecodeString(hashStr)
		if err != nil {
			log.Printf("Failed to decode hash %s: %v", parts[1], err)
			continue
		}

		// Create number->hash mapping (evm + n + 8-byte number)
		numBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(numBytes, number)
		
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
		if number > highestNum {
			highestNum = number
			highestHash = make([]byte, len(hash))
			copy(highestHash, hash)
		}

		// Show progress
		if count%10000 == 0 {
			fmt.Printf("  Imported %d blocks (up to %d)...\n", count, number)
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

	fmt.Printf("\n=== Import Complete ===\n")
	fmt.Printf("Total blocks imported: %d\n", count)
	fmt.Printf("Highest block: %d\n", highestNum)
	fmt.Printf("Highest hash: %s\n", hex.EncodeToString(highestHash))
}