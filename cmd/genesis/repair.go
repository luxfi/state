package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

func newRepairCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repair",
		Short: "Database repair utilities",
	}
	
	cmd.AddCommand(newDeleteSuffixCmd())
	
	return cmd
}

func newDeleteSuffixCmd() *cobra.Command {
	var prefix string
	
	cmd := &cobra.Command{
		Use:   "delete-suffix [db-path] [suffix]",
		Short: "Delete keys with specific suffix",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			dbPath := args[0]
			suffixHex := args[1]
			
			suffix, err := hex.DecodeString(suffixHex)
			if err != nil {
				return fmt.Errorf("invalid suffix hex: %w", err)
			}
			
			var prefixBytes []byte
			if prefix != "" {
				prefixBytes, err = hex.DecodeString(prefix)
				if err != nil {
					return fmt.Errorf("invalid prefix hex: %w", err)
				}
			}
			
			opts := &pebble.Options{}
			db, err := pebble.Open(dbPath, opts)
			if err != nil {
				return fmt.Errorf("failed to open database: %w", err)
			}
			defer db.Close()
			
			// Count and delete keys with suffix
			var count int
			iter, err := db.NewIter(&pebble.IterOptions{})
			if err != nil {
				return fmt.Errorf("failed to create iterator: %w", err)
			}
			defer iter.Close()
			
			for iter.First(); iter.Valid(); iter.Next() {
				key := iter.Key()
				
				// Check prefix if specified
				if len(prefixBytes) > 0 && len(key) >= len(prefixBytes) {
					hasPrefix := true
					for i := 0; i < len(prefixBytes); i++ {
						if key[i] != prefixBytes[i] {
							hasPrefix = false
							break
						}
					}
					if !hasPrefix {
						continue
					}
				}
				
				// Check suffix
				if len(key) >= len(suffix) {
					hasSuffix := true
					for i := 0; i < len(suffix); i++ {
						if key[len(key)-len(suffix)+i] != suffix[i] {
							hasSuffix = false
							break
						}
					}
					
					if hasSuffix {
						keyCopy := make([]byte, len(key))
						copy(keyCopy, key)
						if err := db.Delete(keyCopy, pebble.Sync); err != nil {
							log.Printf("Failed to delete key %x: %v", keyCopy, err)
						} else {
							count++
							if count%1000 == 0 {
								fmt.Printf("Deleted %d keys...\r", count)
							}
						}
					}
				}
			}
			
			fmt.Printf("\n✅ Deleted %d keys with suffix 0x%s\n", count, suffixHex)
			if prefix != "" {
				fmt.Printf("   (filtered by prefix 0x%s)\n", prefix)
			}
			
			return nil
		},
	}
	
	cmd.Flags().StringVar(&prefix, "prefix", "", "Optional prefix filter (hex)")
	
	return cmd
}