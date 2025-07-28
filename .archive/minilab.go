package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cockroachdb/pebble"
)

func main() {
	var (
		src     = flag.String("src", "$HOME/archived/restored-blockchain-data/chainData/dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ/db/pebbledb", "source database")
		workDir = flag.String("work", "", "working directory (temp if empty)")
		sstCount = flag.Int("sst", 8, "number of SST files to copy")
		skipLuxd = flag.Bool("skip-luxd", false, "skip launching luxd")
	)
	flag.Parse()

	// Create working directory
	if *workDir == "" {
		*workDir = filepath.Join(".tmp", fmt.Sprintf("minilab-%d", time.Now().Unix()))
	}
	
	fmt.Println("=== Mini-Lab Migration Test ===")
	fmt.Printf("Source: %s\n", *src)
	fmt.Printf("Work dir: %s\n", *workDir)
	fmt.Printf("SST count: %d\n", *sstCount)

	// Step 1: Copy SST files
	if err := copySSTs(*src, *workDir, *sstCount); err != nil {
		log.Fatalf("Failed to copy SSTs: %v", err)
	}

	// Step 2: Migrate keys
	evmDB := filepath.Join(*workDir, "evm", "pebbledb")
	if err := migrateKeys(filepath.Join(*workDir, "src", "pebbledb"), evmDB); err != nil {
		log.Fatalf("Failed to migrate keys: %v", err)
	}

	// Step 3: Find tip height
	tip := findTipHeight(evmDB)
	fmt.Printf("\n➜ Sample tip height = %d\n", tip)

	if tip == 0 {
		fmt.Println("❌ FAILED - No canonical mappings found")
		fmt.Println("Root cause: evmn keys missing or revision suffix not stripped")
		os.Exit(1)
	}

	// Step 4: Replay consensus
	stateDB := filepath.Join(*workDir, "state", "pebbledb")
	if err := replayConsensus(evmDB, stateDB, tip); err != nil {
		log.Fatalf("Failed to replay consensus: %v", err)
	}

	// Step 5: Launch luxd and verify
	if !*skipLuxd {
		if err := verifyWithLuxd(*workDir, tip); err != nil {
			log.Fatalf("Failed to verify with luxd: %v", err)
		}
	}

	fmt.Println("\n✅ SUCCESS - Mini-lab test passed!")
}

func copySSTs(src, workDir string, count int) error {
	fmt.Println("\n=== Step 1: Copying SST files ===")
	
	srcPebble := filepath.Join(workDir, "src", "pebbledb")
	if err := os.MkdirAll(srcPebble, 0755); err != nil {
		return err
	}

	// List SST files
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read source dir: %w", err)
	}

	sstFiles := []string{}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".sst") {
			sstFiles = append(sstFiles, entry.Name())
		}
	}

	if len(sstFiles) == 0 {
		return fmt.Errorf("no SST files found in %s", src)
	}

	// Sort and copy first N SSTs
	if count > len(sstFiles) {
		count = len(sstFiles)
	}

	fmt.Printf("Found %d SST files, copying first %d\n", len(sstFiles), count)
	
	for i := 0; i < count; i++ {
		src := filepath.Join(src, sstFiles[i])
		dst := filepath.Join(srcPebble, sstFiles[i])
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s: %w", sstFiles[i], err)
		}
	}

	// Copy OPTIONS and MANIFEST files
	for _, pattern := range []string{"OPTIONS*", "MANIFEST-*"} {
		matches, _ := filepath.Glob(filepath.Join(src, pattern))
		for _, match := range matches {
			dst := filepath.Join(srcPebble, filepath.Base(match))
			if err := copyFile(match, dst); err != nil {
				return fmt.Errorf("failed to copy %s: %w", match, err)
			}
		}
	}

	return nil
}

func migrateKeys(src, dst string) error {
	fmt.Println("\n=== Step 2: Migrating keys ===")
	
	cmd := exec.Command("bin/migrate_evm", "--src", src, "--dst", dst, "--verbose")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("migrate_evm failed: %w", err)
	}
	
	return nil
}

func findTipHeight(dbPath string) uint64 {
	fmt.Println("\n=== Step 3: Finding tip height ===")
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		log.Printf("Failed to open database: %v", err)
		return 0
	}
	defer db.Close()

	iter, err := db.NewIter(nil)
	if err != nil {
		log.Printf("Failed to create iterator: %v", err)
		return 0
	}
	defer iter.Close()

	max := uint64(0)
	prefix := append([]byte("evm"), 'n')
	
	for iter.SeekGE(prefix); iter.Valid() && len(iter.Key()) >= 12 && string(iter.Key()[:4]) == "evmn"; iter.Next() {
		n := binary.BigEndian.Uint64(iter.Key()[4:12])
		if n > max {
			max = n
		}
	}

	return max
}

func replayConsensus(evmDB, stateDB string, tip uint64) error {
	fmt.Printf("\n=== Step 4: Replaying consensus (tip=%d) ===\n", tip)
	
	cmd := exec.Command("bin/replay-consensus-pebble",
		"--evm", evmDB,
		"--state", stateDB,
		"--tip", fmt.Sprintf("%d", tip))
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Output: %s\n", output)
		return fmt.Errorf("replay-consensus-pebble failed: %w", err)
	}
	
	// Show last few lines of output
	lines := strings.Split(string(output), "\n")
	start := len(lines) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(lines); i++ {
		if lines[i] != "" {
			fmt.Println(lines[i])
		}
	}
	
	return nil
}

func verifyWithLuxd(workDir string, expectedTip uint64) error {
	fmt.Println("\n=== Step 5: Launching luxd ===")
	
	port := "9655"
	cmd := exec.Command("luxd",
		"--db-dir", workDir,
		"--network-id", "96369",
		"--staking-enabled=false",
		"--http-port", port,
		"--log-level", "info")
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start luxd: %w", err)
	}
	defer cmd.Process.Kill()

	fmt.Printf("➜ luxd PID = %d\n", cmd.Process.Pid)
	fmt.Println("Waiting for initialization...")
	time.Sleep(10 * time.Second)

	// Check eth_blockNumber
	fmt.Println("\n=== Step 6: Verifying RPC ===")
	rpcCmd := exec.Command("curl", "-s",
		"--data", `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`,
		fmt.Sprintf("http://127.0.0.1:%s/ext/bc/C/rpc", port))
	
	output, err := rpcCmd.Output()
	if err != nil {
		return fmt.Errorf("RPC call failed: %w", err)
	}

	fmt.Printf("RPC response: %s\n", output)
	
	// Parse response (simplified - in production use proper JSON parsing)
	if strings.Contains(string(output), fmt.Sprintf("0x%x", expectedTip)) {
		fmt.Printf("✅ Block height matches expected: %d\n", expectedTip)
		return nil
	}

	return fmt.Errorf("block height mismatch")
}

func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, input, 0644)
}