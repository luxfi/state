package main

import (
    "encoding/binary"
    "fmt"
    "log"
    "os"
    
    "github.com/cockroachdb/pebble"
    "github.com/luxfi/geth/core/types"
    "github.com/luxfi/geth/rlp"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Println("Usage: export-rlp <source-db> <output-file> [max-block]")
        fmt.Println("Example: export-rlp output/mainnet/C/chaindata-namespaced blocks.rlp 14552")
        os.Exit(1)
    }
    
    sourceDB := os.Args[1]
    outputFile := os.Args[2]
    maxBlock := uint64(14552) // Default to what we know we have
    
    if len(os.Args) > 3 {
        fmt.Sscanf(os.Args[3], "%d", &maxBlock)
    }
    
    // Open source database
    db, err := pebble.Open(sourceDB, &pebble.Options{
        ReadOnly: true,
    })
    if err != nil {
        log.Fatalf("Failed to open database: %v", err)
    }
    defer db.Close()
    
    // Create output file
    out, err := os.Create(outputFile)
    if err != nil {
        log.Fatalf("Failed to create output file: %v", err)
    }
    defer out.Close()
    
    fmt.Printf("Exporting blocks from %s to %s (max block: %d)\n", sourceDB, outputFile, maxBlock)
    
    exported := 0
    missing := 0
    
    // Export blocks in order
    for blockNum := uint64(0); blockNum <= maxBlock; blockNum++ {
        // Read block header
        headerKey := append([]byte{0x68}, encodeBlockNumber(blockNum)...) // 'h' + number
        headerKey = append(headerKey, []byte{0x6e}...) // + 'n'
        
        hashBytes, closer, err := db.Get(headerKey)
        if err != nil {
            if blockNum == 0 {
                fmt.Printf("Warning: Block %d header not found (canonical key: %x)\n", blockNum, headerKey)
            }
            missing++
            continue
        }
        closer.Close()
        
        // Get the actual header
        headerDataKey := append([]byte{0x48}, hashBytes...) // 'H' + hash
        headerData, closer, err := db.Get(headerDataKey)
        if err != nil {
            fmt.Printf("Warning: Block %d header data not found (key: %x)\n", blockNum, headerDataKey)
            missing++
            continue
        }
        closer.Close()
        
        // Get the body
        bodyKey := append([]byte{0x62}, hashBytes...) // 'b' + hash
        bodyData, closer, err := db.Get(bodyKey)
        if err != nil {
            fmt.Printf("Warning: Block %d body not found (key: %x)\n", blockNum, bodyKey)
            missing++
            continue
        }
        closer.Close()
        
        // Decode header
        var header types.Header
        if err := rlp.DecodeBytes(headerData, &header); err != nil {
            fmt.Printf("Error decoding header %d: %v\n", blockNum, err)
            continue
        }
        
        // Decode body
        var body types.Body
        if err := rlp.DecodeBytes(bodyData, &body); err != nil {
            fmt.Printf("Error decoding body %d: %v\n", blockNum, err)
            continue
        }
        
        // Create block
        block := types.NewBlockWithHeader(&header).WithBody(body)
        
        // Write RLP to file
        if err := rlp.Encode(out, block); err != nil {
            fmt.Printf("Error encoding block %d: %v\n", blockNum, err)
            continue
        }
        
        exported++
        
        if blockNum%1000 == 0 {
            fmt.Printf("Exported %d blocks...\n", blockNum)
        }
    }
    
    fmt.Printf("\nExport complete!\n")
    fmt.Printf("Exported: %d blocks\n", exported)
    fmt.Printf("Missing: %d blocks\n", missing)
    fmt.Printf("Output: %s\n", outputFile)
}

func encodeBlockNumber(number uint64) []byte {
    enc := make([]byte, 8)
    binary.BigEndian.PutUint64(enc, number)
    return enc
}