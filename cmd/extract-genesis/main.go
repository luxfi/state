package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/cockroachdb/pebble"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	dbPath      = flag.String("db", "", "Path to the database directory")
	dbType      = flag.String("type", "auto", "Database type: leveldb, pebble, or auto")
	outputPath  = flag.String("output", "", "Output file path (default: stdout)")
	outputFormat = flag.String("format", "json", "Output format: json, raw, base64, or hex")
	prettyPrint = flag.Bool("pretty", true, "Pretty print JSON output")
	includeAlloc = flag.Bool("alloc", true, "Include account allocations in genesis")
)

func main() {
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintf(os.Stderr, "Error: -db flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	// Auto-detect database type if needed
	if *dbType == "auto" {
		*dbType = detectDatabaseType(*dbPath)
	}

	var genesisData []byte
	var err error

	switch *dbType {
	case "leveldb":
		genesisData, err = extractFromLevelDB(*dbPath)
	case "pebble":
		genesisData, err = extractFromPebbleDB(*dbPath)
	default:
		fmt.Fprintf(os.Stderr, "Error: Unknown database type: %s\n", *dbType)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting genesis: %v\n", err)
		os.Exit(1)
	}

	// Process and output the data
	if err := processAndOutput(genesisData); err != nil {
		fmt.Fprintf(os.Stderr, "Error processing output: %v\n", err)
		os.Exit(1)
	}
}

func detectDatabaseType(dbPath string) string {
	// Check for PebbleDB marker files
	if _, err := os.Stat(dbPath + "/CURRENT"); err == nil {
		if _, err := os.Stat(dbPath + "/OPTIONS-000003"); err == nil {
			return "pebble"
		}
	}
	
	// Default to LevelDB
	return "leveldb"
}

func extractFromPebbleDB(dbPath string) ([]byte, error) {
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("failed to open PebbleDB: %w", err)
	}
	defer db.Close()

	// Try to get the genesis key
	value, closer, err := db.Get([]byte("genesis"))
	if err != nil {
		return nil, fmt.Errorf("genesis key not found in database: %w", err)
	}
	defer closer.Close()

	// Copy the value since it's only valid until closer is called
	genesisData := make([]byte, len(value))
	copy(genesisData, value)

	return genesisData, nil
}

func extractFromLevelDB(dbPath string) ([]byte, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open LevelDB: %w", err)
	}
	defer db.Close()

	// Try to get the genesis key
	value, err := db.Get([]byte("genesis"), nil)
	if err != nil {
		return nil, fmt.Errorf("genesis key not found in database: %w", err)
	}

	return value, nil
}

func processAndOutput(genesisData []byte) error {
	// Check if it's a compressed blob or raw genesis
	if isCompressed(genesisData) {
		// For compressed data, handle based on format
		return outputCompressedGenesis(genesisData)
	}

	// Try to decode as RLP-encoded genesis
	var genesis core.Genesis
	if err := rlp.DecodeBytes(genesisData, &genesis); err == nil {
		return outputDecodedGenesis(&genesis)
	}

	// Try to decode as JSON
	if err := json.Unmarshal(genesisData, &genesis); err == nil {
		return outputDecodedGenesis(&genesis)
	}

	// If all else fails, output as raw
	return outputRawGenesis(genesisData)
}

func isCompressed(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	
	// Check for common compression magic bytes
	// ZSTD: 0x28, 0xB5, 0x2F, 0xFD
	if data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD {
		return true
	}
	
	// GZIP: 0x1F, 0x8B
	if data[0] == 0x1F && data[1] == 0x8B {
		return true
	}
	
	return false
}

func outputCompressedGenesis(data []byte) error {
	switch *outputFormat {
	case "raw":
		return outputRaw(data)
	case "base64":
		return outputBase64(data)
	case "hex":
		return outputHex(data)
	default:
		// For JSON format with compressed data, output metadata
		metadata := map[string]interface{}{
			"compressed": true,
			"size":       len(data),
			"format":     detectCompressionFormat(data),
			"data":       base64.StdEncoding.EncodeToString(data),
		}
		return outputJSON(metadata)
	}
}

func outputDecodedGenesis(genesis *core.Genesis) error {
	if *outputFormat != "json" {
		// For non-JSON formats, re-encode the genesis
		data, err := json.Marshal(genesis)
		if err != nil {
			return err
		}
		return outputRawGenesis(data)
	}

	// For JSON format, output the decoded genesis
	if !*includeAlloc {
		genesis.Alloc = nil
	}
	
	return outputJSON(genesis)
}

func outputRawGenesis(data []byte) error {
	switch *outputFormat {
	case "raw":
		return outputRaw(data)
	case "base64":
		return outputBase64(data)
	case "hex":
		return outputHex(data)
	default:
		// Try to parse as JSON for pretty printing
		var genesis map[string]interface{}
		if err := json.Unmarshal(data, &genesis); err == nil {
			return outputJSON(genesis)
		}
		// If not JSON, output as base64
		return outputBase64(data)
	}
}

func outputRaw(data []byte) error {
	if *outputPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	return os.WriteFile(*outputPath, data, 0644)
}

func outputBase64(data []byte) error {
	encoded := base64.StdEncoding.EncodeToString(data)
	if *outputPath == "" {
		fmt.Println(encoded)
		return nil
	}
	return os.WriteFile(*outputPath, []byte(encoded), 0644)
}

func outputHex(data []byte) error {
	encoded := hex.EncodeToString(data)
	if *outputPath == "" {
		fmt.Println(encoded)
		return nil
	}
	return os.WriteFile(*outputPath, []byte(encoded), 0644)
}

func outputJSON(v interface{}) error {
	var data []byte
	var err error
	
	if *prettyPrint {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	
	if err != nil {
		return err
	}
	
	if *outputPath == "" {
		fmt.Println(string(data))
		return nil
	}
	
	return os.WriteFile(*outputPath, data, 0644)
}

func detectCompressionFormat(data []byte) string {
	if len(data) < 4 {
		return "unknown"
	}
	
	// ZSTD magic bytes
	if data[0] == 0x28 && data[1] == 0xB5 && data[2] == 0x2F && data[3] == 0xFD {
		return "zstd"
	}
	
	// GZIP magic bytes
	if data[0] == 0x1F && data[1] == 0x8B {
		return "gzip"
	}
	
	return "unknown"
}