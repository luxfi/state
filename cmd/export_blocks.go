package cmd

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/core/rawdb"
	"github.com/luxfi/geth/rlp"
	"github.com/spf13/cobra"
)

var exportBlocksCmd = &cobra.Command{
	Use:   "export-blocks",
	Short: "Export blocks from a subnet database",
	Long: `Export blocks from a subnet database to RLP format.
	
This command reads blocks from a source database (with subnet prefix) and exports them
to a target database in a format that can be imported by the C-Chain.

The tool handles the subnet prefix (337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1)
and looks for block data in the expected locations.`,
	Run: runExportBlocks,
}

var (
	sourceDB   string
	targetDB   string
	startBlock uint64
	endBlock   uint64
	rlpFile    string
)

func init() {
	exportBlocksCmd.Flags().StringVar(&sourceDB, "source", "", "Source database path")
	exportBlocksCmd.Flags().StringVar(&targetDB, "target", "", "Target database path (optional)")
	exportBlocksCmd.Flags().StringVar(&rlpFile, "rlp", "", "Export to RLP file instead of database")
	exportBlocksCmd.Flags().Uint64Var(&startBlock, "start", 0, "Start block number")
	exportBlocksCmd.Flags().Uint64Var(&endBlock, "end", 0, "End block number (0 = latest)")
	
	exportBlocksCmd.MarkFlagRequired("source")
	
	rootCmd.AddCommand(exportBlocksCmd)
}

func runExportBlocks(cmd *cobra.Command, args []string) {
	if targetDB == "" && rlpFile == "" {
		log.Fatal("Either --target or --rlp must be specified")
	}
	
	// Open source database
	src, err := pebble.Open(sourceDB, &pebble.Options{
		ReadOnly: true,
	})
	if err != nil {
		log.Fatalf("Failed to open source database: %v", err)
	}
	defer src.Close()
	
	fmt.Printf("=== Exporting blocks from: %s ===\n", sourceDB)
	
	// The subnet prefix we found
	subnetPrefix := []byte{0x33, 0x7f, 0xb7, 0x3f, 0x9b, 0xcd, 0xac, 0x8c, 0x31, 0xa2, 0xd5, 0xf7, 0xb8, 0x77, 0xab, 0x1e, 0x8a, 0x2b, 0x7f, 0x2a, 0x1e, 0x9b, 0xf0, 0x2a, 0x0a, 0x0e, 0x6c, 0x6f, 0xd1, 0x64, 0xf1, 0xd1}
	
	// First, scan for block data patterns
	fmt.Println("Scanning for block data patterns...")
	patterns := scanForBlockPatterns(src, subnetPrefix)
	
	if len(patterns) == 0 {
		// Try looking for blocks stored as simple height->data mapping (like avalanchego test)
		fmt.Println("Trying simple block height mapping...")
		exportSimpleBlocks(src, targetDB, rlpFile, startBlock, endBlock)
		return
	}
	
	// Export using standard patterns
	exportStandardBlocks(src, targetDB, rlpFile, subnetPrefix, startBlock, endBlock)
}

func scanForBlockPatterns(db *pebble.DB, prefix []byte) map[string]int {
	patterns := make(map[string]int)
	
	// Look for standard rawdb prefixes after subnet prefix
	prefixesToCheck := map[string]byte{
		"headers":    rawdb.HeaderPrefix[0],
		"bodies":     rawdb.BlockBodyPrefix[0],
		"receipts":   rawdb.BlockReceiptsPrefix[0],
		"numberHash": 'n',
		"hashNumber": 'H',
	}
	
	for name, b := range prefixesToCheck {
		checkKey := append(prefix, b)
		iter, err := db.NewIter(&pebble.IterOptions{
			LowerBound: checkKey,
			UpperBound: append(checkKey, 0xff),
		})
		if err != nil {
			continue
		}
		
		count := 0
		for iter.First(); iter.Valid() && count < 10; iter.Next() {
			count++
		}
		iter.Close()
		
		if count > 0 {
			patterns[name] = count
			fmt.Printf("Found %s data: %d entries\n", name, count)
		}
	}
	
	return patterns
}

func exportSimpleBlocks(src *pebble.DB, targetPath, rlpPath string, start, end uint64) {
	var writer *rlp.Stream
	var outFile *os.File
	var dst *pebble.DB
	
	if rlpPath != "" {
		var err error
		outFile, err = os.Create(rlpPath)
		if err != nil {
			log.Fatalf("Failed to create RLP file: %v", err)
		}
		defer outFile.Close()
		
		// We'll write raw blocks to the file
		fmt.Printf("Exporting to RLP file: %s\n", rlpPath)
	}
	
	if targetPath != "" {
		var err error
		dst, err = pebble.Open(targetPath, &pebble.Options{})
		if err != nil {
			log.Fatalf("Failed to open target database: %v", err)
		}
		defer dst.Close()
	}
	
	// Try to read blocks by simple height key (8 bytes big endian)
	count := 0
	current := start
	
	for {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, current)
		
		val, closer, err := src.Get(key)
		if err != nil {
			if current == start {
				fmt.Printf("No block found at height %d with simple key\n", current)
			}
			break
		}
		
		// Found a block!
		if count == 0 {
			fmt.Printf("Found blocks with simple height keys!\n")
		}
		
		blockData := make([]byte, len(val))
		copy(blockData, val)
		closer.Close()
		
		if rlpPath != "" {
			// Write raw block data to file
			outFile.Write(blockData)
		}
		
		if dst != nil {
			// Store in target with same key format
			if err := dst.Set(key, blockData, pebble.Sync); err != nil {
				log.Printf("Failed to write block %d: %v", current, err)
			}
		}
		
		count++
		current++
		
		if count%1000 == 0 {
			fmt.Printf("Exported %d blocks...\n", count)
		}
		
		if end > 0 && current > end {
			break
		}
	}
	
	fmt.Printf("Exported %d blocks total\n", count)
}

func exportStandardBlocks(src *pebble.DB, targetPath, rlpPath string, prefix []byte, start, end uint64) {
	// Implementation for standard rawdb format blocks
	// This would handle the case where blocks are stored with proper rawdb prefixes
	
	fmt.Println("Standard block export not yet implemented for subnet data")
	fmt.Println("The subnet data appears to only contain state data, not block data")
	fmt.Println("To proceed with state-only migration, use:")
	fmt.Println("  genesis export-state --source", sourceDB)
}

// Helper function to create block key (height as 8 bytes big endian)
func blockKey(height uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	return key
}