package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	
	"github.com/cockroachdb/pebble"
	"github.com/luxfi/geth/common"
	"github.com/luxfi/geth/core/types"
	"github.com/luxfi/geth/rlp"
)

// Helper to get the genesis binary path
func genesisBin() string {
	root, _ := filepath.Abs("../")
	return filepath.Join(root, "bin", "genesis")
}

// Helper to run genesis command
func genesis(args ...string) (string, error) {
	return run(genesisBin(), args...)
}

// Helper to run genesis command (must succeed)
func mustGenesis(args ...string) string {
	return mustRun(genesisBin(), args...)
}

// Helper to get absolute path to source file
func srcFile(name string) string {
	root, _ := filepath.Abs("../")
	return filepath.Join(root, name)
}

// Helper to run a command and return output
func mustRun(name string, args ...string) string {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		panic(fmt.Sprintf("Command failed: %s %v\nOutput: %s\nError: %v", 
			name, args, string(output), err))
	}
	return strings.TrimSpace(string(output))
}

// Helper to run a command and return output and error
func run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// Wait for RPC port to be available
func waitRPC(port string) error {
	for i := 0; i < 20; i++ {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("RPC port %s never came up", port)
}

// Build genesis tool if requested
func buildGenesisTool() error {
	if os.Getenv("GINKGO_BUILD_TOOLING") != "true" {
		// Check if binary already exists
		if _, err := os.Stat(genesisBin()); err == nil {
			return nil
		}
	}
	
	root, _ := filepath.Abs("../")
	cmd := exec.Command("go", "build", 
		"-o", genesisBin(),
		"./cmd/genesis")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to build genesis tool: %v\nOutput: %s", 
			err, string(output))
	}
	return nil
}
// createMiniTestDB creates a minimal test database with subnet namespace prefix
func createMiniTestDB(tmpDir string) string {
	dbPath := filepath.Join(tmpDir, "test-subnet-db")
	os.MkdirAll(dbPath, 0755)
	
	db, err := pebble.Open(dbPath, &pebble.Options{})
	if err != nil {
		panic(fmt.Sprintf("Failed to create test DB: %v", err))
	}
	defer db.Close()
	
	// Create namespace prefix for chain ID 96369 (LUX)
	namespace := make([]byte, 32)
	// Use the actual namespace format from the subnet
	// Chain hex: 337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1
	nsHex := "337fb73f9bcdac8c31a2d5f7b877ab1e8a2b7f2a1e9bf02a0a0e6c6fd164f1d1"
	nsBytes, _ := hex.DecodeString(nsHex)
	copy(namespace, nsBytes)
	
	// Create some test data with namespace prefix
	batch := db.NewBatch()
	
	// Add a test header (block 0)
	header := &types.Header{
		Number:     common.Big0,
		Time:       1000,
		Difficulty: common.Big1,
		GasLimit:   10000000,
	}
	headerRLP, _ := rlp.EncodeToBytes(header)
	headerHash := header.Hash()
	
	// Key format: namespace + 0x68 (header prefix) + block number
	headerKey := append(namespace, 0x68)
	headerKey = append(headerKey, encodeBlockNumber(0)...)
	headerKey = append(headerKey, headerHash[:]...)
	batch.Set(headerKey, headerRLP, nil)
	
	// Add hash->number mapping
	hashKey := append(namespace, 0x48) // 'H' prefix
	hashKey = append(hashKey, encodeBlockNumber(0)...)
	batch.Set(hashKey, headerHash[:], nil)
	
	// Add a test account
	accountKey := append(namespace, 0x26) // account prefix
	testAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	accountKey = append(accountKey, testAddr[:]...)
	accountValue := []byte{0x01, 0x02, 0x03} // dummy account data
	batch.Set(accountKey, accountValue, nil)
	
	// Commit the batch
	if err := batch.Commit(pebble.Sync); err != nil {
		panic(fmt.Sprintf("Failed to commit test data: %v", err))
	}
	
	return dbPath
}

func encodeBlockNumber(number uint64) []byte {
	enc := make([]byte, 8)
	binary.BigEndian.PutUint64(enc, number)
	return enc
}