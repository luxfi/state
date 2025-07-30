package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
)

func main() {
	// Open EVM DB
	evmDB, err := pebble.Open("runtime/luxd-final/db/chains/X6CU5qgMJfzsTB9UWxj2ZY5hd68x41HfZ4m4hCBWbHuj1Ebc3/vm/mgj786NP7uDwBCcq6YwThhaN8FLyybkCa4zBWTQbNgmK6k9A6/evm", &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	defer evmDB.Close()

	// Get the hash of block 1082780
	blockNum := uint64(1082780)
	blockNumBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockNumBytes, blockNum)

	// Get canonical hash
	canonicalKey := append([]byte{0x68}, blockNumBytes...)
	canonicalKey = append(canonicalKey, 0x6e)

	var blockHash []byte
	if val, closer, err := evmDB.Get(canonicalKey); err == nil && len(val) == 32 {
		blockHash = make([]byte, 32)
		copy(blockHash, val)
		closer.Close()
		fmt.Printf("Block hash at height %d: %x\n", blockNum, blockHash)
	} else {
		log.Fatal("Could not find canonical hash for block 1082780")
	}

	// Get header to extract timestamp
	headerKey := append([]byte{0x68}, blockHash...)
	if header, closer, err := evmDB.Get(headerKey); err == nil {
		// Header is RLP encoded, timestamp is at offset 64-72 in the decoded header
		// For now, just print the length
		fmt.Printf("Header found, length: %d bytes\n", len(header))
		// In a typical Ethereum header, timestamp is the 11th field
		// Let's try to extract it (rough approximation)
		if len(header) > 100 {
			// Skip to approximately where timestamp should be
			// This is a rough estimate - proper RLP decoding would be better
			for i := 60; i < len(header)-8; i++ {
				timestamp := binary.BigEndian.Uint64(header[i : i+8])
				if timestamp > 1600000000 && timestamp < 1800000000 { // Reasonable Unix timestamp range
					fmt.Printf("Likely timestamp found: %d (Unix time)\n", timestamp)
					break
				}
			}
		}
		closer.Close()
	} else {
		fmt.Println("Could not find header for block")
	}
}
