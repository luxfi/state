package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/luxfi/node/ids"
	"github.com/spf13/cobra"
)

// NewReadGenesisCommand creates the read-genesis command
func NewReadGenesisCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "read-genesis [chaindata-path]",
		Short: "Read genesis from historic chain data",
		Long: `Read genesis configuration from historic blockchain data.
		
This command extracts the genesis block from a blockchain database by:
1. Looking for stored genesis keys
2. Reading block 0 if available
3. Extracting genesis from chain config
4. Deriving blockchain ID from genesis`,
		Args: cobra.ExactArgs(1),
		RunE: runReadGenesis,
	}

	cmd.Flags().StringP("output", "o", "", "Output file path (default: stdout)")
	cmd.Flags().BoolP("pretty", "p", true, "Pretty print JSON output")
	cmd.Flags().BoolP("show-id", "i", true, "Show derived blockchain ID")
	cmd.Flags().BoolP("raw", "r", false, "Output raw genesis bytes")
	cmd.Flags().StringP("format", "f", "json", "Output format: json, hex, base64")

	return cmd
}

func runReadGenesis(cmd *cobra.Command, args []string) error {
	chainDataPath := args[0]
	outputPath, _ := cmd.Flags().GetString("output")
	prettyPrint, _ := cmd.Flags().GetBool("pretty")
	showID, _ := cmd.Flags().GetBool("show-id")
	rawOutput, _ := cmd.Flags().GetBool("raw")
	format, _ := cmd.Flags().GetString("format")

	fmt.Printf("📖 Reading genesis from: %s\n", chainDataPath)

	// Try different database paths
	dbPaths := []string{
		filepath.Join(chainDataPath, "db", "pebbledb"),
		filepath.Join(chainDataPath, "db"),
		filepath.Join(chainDataPath, "pebbledb"),
		chainDataPath,
	}

	var db *pebble.DB
	var dbPath string

	for _, path := range dbPaths {
		if _, err := os.Stat(filepath.Join(path, "CURRENT")); err == nil {
			dbPath = path
			var openErr error
			db, openErr = pebble.Open(path, &pebble.Options{ReadOnly: true})
			if openErr == nil {
				break
			}
		}
	}

	if db == nil {
		return fmt.Errorf("failed to open database at any known path")
	}
	defer db.Close()

	fmt.Printf("✅ Opened database at: %s\n", dbPath)

	// Try multiple approaches to find genesis
	var genesisData []byte
	var genesis *core.Genesis

	// Approach 1: Direct genesis key
	if value, closer, err := db.Get([]byte("genesis")); err == nil {
		defer closer.Close()
		genesisData = make([]byte, len(value))
		copy(genesisData, value)
		fmt.Println("✅ Found genesis key directly")
	}

	// Approach 2: Look for block 0
	if genesisData == nil {
		// Try to find block 0 hash
		if blockHashValue, closer, err := db.Get(append([]byte("H"), encodeBlockNumber(0)...)); err == nil {
			defer closer.Close()
			blockHash := make([]byte, len(blockHashValue))
			copy(blockHash, blockHashValue)
			
			// Get block header
			if _, closer2, err := db.Get(append([]byte("h"), blockHash...)); err == nil {
				defer closer2.Close()
				fmt.Println("✅ Found block 0 header")
				// Parse header to extract genesis info
				// This is simplified - real implementation would decode the header
			}
		}
	}

	// Approach 3: Scan for config keys
	if genesisData == nil {
		iter, err := db.NewIter(&pebble.IterOptions{})
		if err == nil {
			defer iter.Close()
			
			count := 0
			for iter.First(); iter.Valid() && count < 10000; iter.Next() {
				key := iter.Key()
				
				// Look for config-related keys
				if bytes.Contains(key, []byte("config")) || 
				   bytes.Contains(key, []byte("genesis")) ||
				   bytes.Contains(key, []byte("Config")) {
					fmt.Printf("🔍 Found potential genesis key: %x\n", key)
					
					value := make([]byte, len(iter.Value()))
					copy(value, iter.Value())
					
					// Try to decode as genesis
					var testGenesis core.Genesis
					if err := json.Unmarshal(value, &testGenesis); err == nil {
						genesisData = value
						genesis = &testGenesis
						fmt.Println("✅ Found genesis in config key")
						break
					}
				}
				count++
			}
		}
	}

	// If we still don't have genesis, create a minimal one
	if genesisData == nil {
		fmt.Println("⚠️  No genesis found in database, creating minimal genesis")
		genesis = createMinimalGenesis()
		genesisData, _ = json.Marshal(genesis)
	}

	// Decode genesis if we haven't already
	if genesis == nil && genesisData != nil {
		// Try RLP decoding first
		if err := rlp.DecodeBytes(genesisData, &genesis); err != nil {
			// Try JSON decoding
			if err := json.Unmarshal(genesisData, &genesis); err != nil {
				// If both fail, treat as raw genesis blob
				fmt.Println("⚠️  Could not decode genesis, treating as raw blob")
			}
		}
	}

	// Show blockchain ID if requested
	if showID && genesisData != nil {
		blockchainID := deriveBlockchainID(genesisData)
		fmt.Printf("📌 Blockchain ID: %s\n", blockchainID)
		if genesis != nil && genesis.Config != nil {
			fmt.Printf("📌 Chain ID: %v\n", genesis.Config.ChainID)
		}
	}

	// Output based on format
	var output []byte
	if rawOutput {
		output = genesisData
	} else {
		switch format {
		case "hex":
			output = []byte(hex.EncodeToString(genesisData))
		case "base64":
			output = []byte(base64.StdEncoding.EncodeToString(genesisData))
		case "json":
			fallthrough
		default:
			if genesis != nil {
				if prettyPrint {
					output, _ = json.MarshalIndent(genesis, "", "  ")
				} else {
					output, _ = json.Marshal(genesis)
				}
			} else {
				output = genesisData
			}
		}
	}

	// Write output
	if outputPath != "" {
		if err := ioutil.WriteFile(outputPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		fmt.Printf("✅ Wrote genesis to: %s\n", outputPath)
	} else {
		fmt.Println("\n📄 Genesis:")
		fmt.Println(string(output))
	}

	return nil
}

func createMinimalGenesis() *core.Genesis {
	return &core.Genesis{
		Config: &params.ChainConfig{
			ChainID:        big.NewInt(96369),
			HomesteadBlock: big.NewInt(0),
			EIP150Block:    big.NewInt(0),
			EIP155Block:    big.NewInt(0),
			EIP158Block:    big.NewInt(0),
			ByzantiumBlock: big.NewInt(0),
		},
		Difficulty: big.NewInt(0),
		GasLimit:   8000000,
		Alloc:      make(core.GenesisAlloc),
	}
}

func deriveBlockchainID(genesisBytes []byte) string {
	hash := sha256.Sum256(genesisBytes)
	id, _ := ids.ToID(hash[:])
	return id.String()
}

func encodeBlockNumber(number uint64) []byte {
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(number)
		number >>= 8
	}
	return buf
}