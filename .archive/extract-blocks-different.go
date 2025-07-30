package main

import (
	"encoding/binary"
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
		fmt.Println("Usage: extract-blocks-different <source-db> <output-rlp>")
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

	// First, let's find all headers by looking for header-like keys
	headers := make(map[uint64]*types.Header)
	bodies := make(map[uint64]*types.Body)

	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()

	// Look for headers (keys with 0x48 at position 9)
	fmt.Println("Searching for headers...")
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		val := iter.Value()

		if len(key) > 9 && key[9] == 0x48 && len(val) > 100 { // 'H' prefix
			var header types.Header
			if err := rlp.DecodeBytes(val, &header); err == nil && header.Number != nil {
				blockNum := header.Number.Uint64()
				headers[blockNum] = &header
				if blockNum%1000 == 0 || blockNum < 10 {
					fmt.Printf("Found header for block %d (hash: %x)\n", blockNum, header.Hash())
				}
			}
		}
	}

	fmt.Printf("Found %d headers\n", len(headers))

	// Now look for bodies
	fmt.Println("\nSearching for bodies...")
	iter2, _ := db.NewIter(&pebble.IterOptions{})
	defer iter2.Close()

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		val := iter2.Value()

		if len(key) > 9 && key[9] == 0x62 && len(val) > 0 { // 'b' prefix
			var body types.Body
			if err := rlp.DecodeBytes(val, &body); err == nil {
				// Try to match with headers by hash
				for blockNum, header := range headers {
					// The key should contain the hash
					if len(key) >= 41 {
						hash := key[10:42] // Skip prefix, get 32 bytes
						headerHash := header.Hash()
						match := true
						for i := 0; i < 32; i++ {
							if hash[i] != headerHash[i] {
								match = false
								break
							}
						}
						if match {
							bodies[blockNum] = &body
							if blockNum%1000 == 0 || blockNum < 10 {
								fmt.Printf("Found body for block %d (txs: %d)\n", blockNum, len(body.Transactions))
							}
							break
						}
					}
				}
			}
		}
	}

	fmt.Printf("Found %d bodies\n", len(bodies))

	// Export blocks in order
	if len(headers) > 0 {
		// Get sorted block numbers
		blockNums := make([]uint64, 0, len(headers))
		for num := range headers {
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
		for _, blockNum := range blockNums {
			header := headers[blockNum]
			body, hasBody := bodies[blockNum]

			if !hasBody {
				// Create empty body for blocks without transactions
				body = &types.Body{
					Transactions: []*types.Transaction{},
					Uncles:       []*types.Header{},
				}
			}

			// Create block
			block := types.NewBlockWithHeader(header).WithBody(*body)

			// Write RLP to file
			if err := rlp.Encode(out, block); err != nil {
				fmt.Printf("Error encoding block %d: %v\n", blockNum, err)
				continue
			}

			exported++
			if blockNum%1000 == 0 {
				fmt.Printf("Exported %d blocks...\n", exported)
			}
		}

		fmt.Printf("\nExported %d blocks to %s\n", exported, outputFile)
	} else {
		fmt.Println("\nNo blocks found to export")
	}
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}
