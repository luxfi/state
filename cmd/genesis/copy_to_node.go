package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newCopyCmd() *cobra.Command {
	var (
		chainID, vmID, nodeDir, evmDir, tipHash string
		tipHeight                               uint64
	)
	cmd := &cobra.Command{
		Use:   "copy-to-node",
		Short: "Move migrated EVM DB into node layout and write consensus markers",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if chainID == "" || vmID == "" || nodeDir == "" || evmDir == "" || tipHash == "" {
				return fmt.Errorf("all flags are required")
			}

			dst := filepath.Join(
				nodeDir, "db", "chains", chainID, "vm", vmID, "evm",
			)
			if err := os.RemoveAll(dst); err != nil {
				return err
			}
			if err := copyDir(evmDir, dst); err != nil { // copyDir already exists in migrate code
				return err
			}
			return writeMarkers(
				filepath.Join(nodeDir, "db", "chains", chainID),
				tipHash, tipHeight,
			)
		},
	}
	cmd.Flags().StringVar(&chainID, "chain-id", "", "C-Chain ID")
	cmd.Flags().StringVar(&vmID, "vm-id", "", "VM ID")
	cmd.Flags().StringVar(&nodeDir, "node-dir", "", "luxd --data-dir")
	cmd.Flags().StringVar(&evmDir, "evm-db", "", "source evm/pebbledb dir")
	cmd.Flags().Uint64Var(&tipHeight, "height", 0, "tip height")
	cmd.Flags().StringVar(&tipHash, "hash", "", "tip block hash (0x...)")

	_ = cmd.MarkFlagRequired("chain-id")
	_ = cmd.MarkFlagRequired("vm-id")
	_ = cmd.MarkFlagRequired("node-dir")
	_ = cmd.MarkFlagRequired("evm-db")
	_ = cmd.MarkFlagRequired("height")
	_ = cmd.MarkFlagRequired("hash")
	return cmd
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// Create directories
		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Copy files
		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func writeMarkers(chainDir, tipHash string, tipHeight uint64) error {
	// Open state database
	stateDBPath := filepath.Join(chainDir, "state")
	opts := &pebble.Options{}
	db, err := pebble.Open(stateDBPath, opts)
	if err != nil {
		return fmt.Errorf("failed to open state db: %w", err)
	}
	defer db.Close()

	// Prepare hash bytes
	hashHex := strings.TrimPrefix(tipHash, "0x")
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("failed to decode hash: %w", err)
	}

	// Write Height marker
	heightKey := []byte("Height")
	heightValue := make([]byte, 8)
	for i := 0; i < 8; i++ {
		heightValue[7-i] = byte(tipHeight >> (8 * i))
	}
	if err := db.Set(heightKey, heightValue, pebble.Sync); err != nil {
		return fmt.Errorf("failed to write Height: %w", err)
	}

	// Write LastAccepted marker
	lastAcceptedKey := []byte("LastAccepted")
	if err := db.Set(lastAcceptedKey, hashBytes, pebble.Sync); err != nil {
		return fmt.Errorf("failed to write LastAccepted: %w", err)
	}

	fmt.Printf("✅ Wrote consensus markers:\n")
	fmt.Printf("   Height: %d (0x%x)\n", tipHeight, tipHeight)
	fmt.Printf("   LastAccepted: %s\n", tipHash)
	
	return nil
}