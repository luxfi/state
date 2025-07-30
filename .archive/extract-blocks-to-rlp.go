package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: extract-blocks-to-rlp <source-db> <output-rlp>")
		fmt.Println("This extracts blocks from subnet database to RLP format for geth import")
		os.Exit(1)
	}

	sourceDB := os.Args[1]
	outputFile := os.Args[2]

	// Open source database
	db, err := pebble.Open(sourceDB, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Printf("=== Extracting blocks from %s ===\n", sourceDB)

	// We need to find headers and bodies
	// Based on the key pattern analysis:
	// Headers: keys with length > 9 and key[9] == 0x48 ('H')
	// Bodies: keys with length > 9 and key[9] == 0x62 ('b')

	headers := make(map[string]*types.Header)
	bodies := make(map[string]*types.Body)
	blocksByNumber := make(map[uint64]string) // number -> hash

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	// First pass: collect headers
	fmt.Println("Pass 1: Collecting headers...")
	headerCount := 0
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		if len(key) > 9 && key[9] == 0x48 && len(val) > 100 { // 'H' prefix and reasonable size
			// The key format is: prefix(10) + hash(32)
			if len(key) >= 42 {
				hash := hex.EncodeToString(key[10:42])

				var header types.Header
				if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
					headers[hash] = &header
					blocksByNumber[header.Number.Uint64()] = hash
					headerCount++

					if headerCount <= 5 || headerCount%1000 == 0 {
						fmt.Printf("  Header %d: block %d, hash %s\n", headerCount, header.Number.Uint64(), hash[:16]+"...")
					}
				}
			}
		}
	}
	fmt.Printf("Found %d headers\n", len(headers))

	// Second pass: collect bodies
	fmt.Println("\nPass 2: Collecting bodies...")
	bodyCount := 0
	iter2, _ := db.NewIter(&pebble.IterOptions{})
	defer iter2.Close()

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		val := iter2.Value()

		if len(key) > 9 && key[9] == 0x62 && len(val) > 0 { // 'b' prefix
			// The key format is: prefix(10) + hash(32)
			if len(key) >= 42 {
				hash := hex.EncodeToString(key[10:42])

				var body types.Body
				if err := rlp.DecodeBytes(val, &body); err == nil {
					bodies[hash] = &body
					bodyCount++

					if bodyCount <= 5 || bodyCount%1000 == 0 {
						fmt.Printf("  Body %d: hash %s, txs %d\n", bodyCount, hash[:16]+"...", len(body.Transactions))
					}
				}
			}
		}
	}
	fmt.Printf("Found %d bodies\n", len(bodies))

	// Now combine and export blocks
	if len(headers) > 0 {
		// Get sorted block numbers
		blockNums := make([]uint64, 0, len(blocksByNumber))
		for num := range blocksByNumber {
			blockNums = append(blockNums, num)
		}
		sort.Slice(blockNums, func(i, j int) bool { return blockNums[i] < blockNums[j] })

		fmt.Printf("\nBlock range: %d to %d\n", blockNums[0], blockNums[len(blockNums)-1])

		// Create output file
		out, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer out.Close()

		exported := 0
		missingBodies := 0

		for _, blockNum := range blockNums {
			hash := blocksByNumber[blockNum]
			header := headers[hash]
			body, hasBody := bodies[hash]

			if !hasBody {
				// Create empty body
				body = &types.Body{
					Transactions: []*types.Transaction{},
					Uncles:       []*types.Header{},
				}
				missingBodies++
			}

			// Create block
			block := types.NewBlockWithHeader(header).WithBody(*body)

			// Write RLP to file
			if err := rlp.Encode(out, block); err != nil {
				fmt.Printf("Error encoding block %d: %v\n", blockNum, err)
				continue
			}

			exported++
			if exported%1000 == 0 {
				fmt.Printf("Exported %d blocks...\n", exported)
			}
		}

		fmt.Printf("\nExport complete!\n")
		fmt.Printf("Exported: %d blocks\n", exported)
		fmt.Printf("Missing bodies: %d (created empty bodies)\n", missingBodies)
		fmt.Printf("Output: %s\n", outputFile)
	} else {
		fmt.Println("\nNo blocks found to export")
	}
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
