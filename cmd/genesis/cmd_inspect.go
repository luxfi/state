package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect blockchain database contents",
		Long:  `Various inspection commands for analyzing blockchain databases`,
	}

	cmd.AddCommand(
		newInspectTipCmd(),
		newInspectBlocksCmd(),
		newInspectKeysCmd(),
		newInspectCanonicalCmd(),
		newInspectConsensusCmd(),
	)

	return cmd
}

func newInspectTipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tip <db-path>",
		Short: "Find the highest block in database",
		Args:  cobra.ExactArgs(1),
		RunE:  runInspectTip,
	}
}

func newInspectBlocksCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "blocks <db-path>",
		Short: "Inspect block headers and bodies",
		Args:  cobra.ExactArgs(1),
		RunE:  runInspectBlocks,
	}
}

func newInspectKeysCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "keys <db-path>",
		Short: "Analyze key patterns and prefixes",
		Args:  cobra.ExactArgs(1),
		RunE:  runInspectKeys,
	}
}

func newInspectCanonicalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "canonical <db-path>",
		Short: "Check canonical hash mappings",
		Args:  cobra.ExactArgs(1),
		RunE:  runInspectCanonical,
	}
}

func newInspectConsensusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "consensus <db-path>",
		Short: "Check consensus state markers",
		Args:  cobra.ExactArgs(1),
		RunE:  runInspectConsensus,
	}
}

func runInspectTip(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Finding chain tip...")
	
	// Check for evmh prefix (headers)
	maxBlock := uint64(0)
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("evmh"),
		UpperBound: []byte("evmi"),
	})
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) >= 12 {
			blockNum := binary.BigEndian.Uint64(key[4:12])
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
		}
	}

	// Also check raw header keys (0x68 prefix)
	iter2, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{0x68},
		UpperBound: []byte{0x69},
	})
	defer iter2.Close()

	for iter2.First(); iter2.Valid(); iter2.Next() {
		key := iter2.Key()
		if len(key) >= 9 {
			blockNum := binary.BigEndian.Uint64(key[1:9])
			if blockNum > maxBlock && blockNum < 100000000 { // Sanity check
				maxBlock = blockNum
			}
		}
	}

	fmt.Printf("Maximum block number: %d\n", maxBlock)
	
	// Check consensus Height
	if heightBytes, closer, err := db.Get([]byte("Height")); err == nil {
		height := binary.BigEndian.Uint64(heightBytes)
		fmt.Printf("Consensus Height: %d\n", height)
		closer.Close()
	}

	// Check LastAccepted
	if hash, closer, err := db.Get([]byte("LastAccepted")); err == nil {
		fmt.Printf("LastAccepted: 0x%s\n", hex.EncodeToString(hash))
		closer.Close()
	}

	return nil
}

func runInspectBlocks(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Sample some blocks
	fmt.Println("Sampling block data...")
	
	// Check evmh prefix
	fmt.Println("\nHeaders (evmh prefix):")
	sampleKeys(db, []byte("evmh"), 5)
	
	// Check 0x68 prefix
	fmt.Println("\nHeaders (0x68 prefix):")
	sampleKeys(db, []byte{0x68}, 5)
	
	// Check bodies
	fmt.Println("\nBodies (evmb prefix):")
	sampleKeys(db, []byte("evmb"), 5)
	
	return nil
}

func runInspectKeys(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Analyzing key patterns...")
	
	prefixCounts := make(map[string]int)
	totalKeys := 0
	
	iter, _ := db.NewIter(&pebble.IterOptions{})
	defer iter.Close()
	
	for iter.First(); iter.Valid() && totalKeys < 100000; iter.Next() {
		key := iter.Key()
		totalKeys++
		
		// Categorize by prefix
		if len(key) > 0 {
			// Check string prefixes
			if len(key) >= 3 {
				if key[0] == 'e' && key[1] == 'v' && key[2] == 'm' {
					prefixCounts[string(key[:4])]++
					continue
				}
			}
			// Single byte prefix
			prefixCounts[fmt.Sprintf("0x%02x", key[0])]++
		}
	}
	
	fmt.Printf("\nAnalyzed %d keys\n", totalKeys)
	fmt.Println("\nPrefix distribution:")
	for prefix, count := range prefixCounts {
		fmt.Printf("  %s: %d\n", prefix, count)
	}
	
	return nil
}

func runInspectCanonical(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Checking canonical mappings...")
	
	// Check specific blocks
	blocks := []uint64{0, 1, 2, 100, 1000, 10000}
	
	for _, blockNum := range blocks {
		fmt.Printf("\nBlock %d:\n", blockNum)
		
		// Check evmn key
		blockBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(blockBytes, blockNum)
		evmnKey := append([]byte("evmn"), blockBytes...)
		
		if hash, closer, err := db.Get(evmnKey); err == nil {
			fmt.Printf("  evmn mapping: 0x%s\n", hex.EncodeToString(hash))
			closer.Close()
		} else {
			fmt.Printf("  evmn mapping: not found\n")
		}
		
		// Check 9-byte canonical key
		canonicalKey := append([]byte{0x68}, blockBytes...)
		canonicalKey = append(canonicalKey, 0x6e)
		
		if hash, closer, err := db.Get(canonicalKey); err == nil {
			fmt.Printf("  9-byte canonical: 0x%s\n", hex.EncodeToString(hash))
			closer.Close()
		} else {
			// Check 10-byte format
			canonicalKey10 := append([]byte{0x68}, blockBytes...)
			canonicalKey10 = append(canonicalKey10, 0x6e)
			
			if hash, closer, err := db.Get(canonicalKey10); err == nil {
				fmt.Printf("  10-byte canonical: 0x%s\n", hex.EncodeToString(hash))
				closer.Close()
			} else {
				fmt.Printf("  canonical: not found\n")
			}
		}
	}
	
	return nil
}

func runInspectConsensus(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	fmt.Println("Checking consensus state...")
	
	// Check Height
	if val, closer, err := db.Get([]byte("Height")); err == nil {
		height := binary.BigEndian.Uint64(val)
		fmt.Printf("Height: %d (0x%x)\n", height, height)
		closer.Close()
	} else {
		fmt.Println("Height: not found")
	}
	
	// Check LastAccepted
	if val, closer, err := db.Get([]byte("LastAccepted")); err == nil {
		fmt.Printf("LastAccepted: 0x%s\n", hex.EncodeToString(val))
		closer.Close()
	} else {
		fmt.Println("LastAccepted: not found")
	}
	
	// Check other consensus keys
	consensusKeys := []string{
		"lastAccepted",
		"consensus/accepted",
		"consensus/lastAccepted",
		"LastBlock",
		"LastHeader",
	}
	
	for _, key := range consensusKeys {
		if val, closer, err := db.Get([]byte(key)); err == nil {
			if len(val) == 8 {
				fmt.Printf("%s: %d\n", key, binary.BigEndian.Uint64(val))
			} else if len(val) == 32 {
				fmt.Printf("%s: 0x%s\n", key, hex.EncodeToString(val))
			} else {
				fmt.Printf("%s: %d bytes\n", key, len(val))
			}
			closer.Close()
		}
	}
	
	return nil
}

func sampleKeys(db *pebble.DB, prefix []byte, count int) {
	iter, _ := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: append(prefix, 0xff),
	})
	defer iter.Close()
	
	sampled := 0
	for iter.First(); iter.Valid() && sampled < count; iter.Next() {
		key := iter.Key()
		val := iter.Value()
		
		fmt.Printf("  Key: %x (len=%d), Value: %d bytes\n", key, len(key), len(val))
		sampled++
	}
	
	if sampled == 0 {
		fmt.Println("  No keys found with this prefix")
	}
}