package genesis

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Config
	}{
		{
			name:   "Empty config gets defaults",
			config: Config{},
			want: Config{
				OutputPath:  "configs/x-chain-genesis-complete.json", // This is set after ChainType default
				ChainType:   "x-chain",
				AssetPrefix: "LUX",
			},
		},
		{
			name: "Custom config preserved",
			config: Config{
				OutputPath:  "custom.json",
				ChainType:   "p-chain",
				AssetPrefix: "ZOO",
			},
			want: Config{
				OutputPath:  "custom.json",
				ChainType:   "p-chain",
				AssetPrefix: "ZOO",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.config)
			if err != nil {
				t.Fatalf("NewGenerator() error = %v", err)
			}

			if gen.config.OutputPath != tt.want.OutputPath {
				t.Errorf("OutputPath = %v, want %v", gen.config.OutputPath, tt.want.OutputPath)
			}
			if gen.config.ChainType != tt.want.ChainType {
				t.Errorf("ChainType = %v, want %v", gen.config.ChainType, tt.want.ChainType)
			}
			if gen.config.AssetPrefix != tt.want.AssetPrefix {
				t.Errorf("AssetPrefix = %v, want %v", gen.config.AssetPrefix, tt.want.AssetPrefix)
			}
		})
	}
}

func TestGenerateGenesis(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create test CSV files
	nftCSV := filepath.Join(tmpDir, "nfts.csv")
	if err := ioutil.WriteFile(nftCSV, []byte(`address,asset_type,collection_type,balance_or_count,staking_power_wei,staking_power_token,chain_name,contract_address,project_name,last_activity_block,received_on_chain,token_ids
0x1234567890123456789012345678901234567890,NFT,Validator,1,1000000000000000000000000,1000000.000000,ethereum,0x31e0f919c67cedd2bc3e294340dc900735810311,lux,12345678,false,1
`), 0644); err != nil {
		t.Fatal(err)
	}

	accountsCSV := filepath.Join(tmpDir, "accounts.csv")
	if err := ioutil.WriteFile(accountsCSV, []byte(`address,balance_wei,balance_token,validator_eligible
0x9876543210987654321098765432109876543210,1000000000000000000000,1000.000000,false
`), 0644); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(tmpDir, "genesis.json")

	config := Config{
		NFTDataPath:      nftCSV,
		AccountsDataPath: accountsCSV,
		OutputPath:       outputPath,
		ChainType:        "x-chain",
		AssetPrefix:      "TEST",
	}

	gen, err := NewGenerator(config)
	if err != nil {
		t.Fatal(err)
	}

	result, err := gen.Generate()
	if err != nil {
		t.Fatal(err)
	}

	// Check result
	if result.ChainType != "x-chain" {
		t.Errorf("ChainType = %v, want x-chain", result.ChainType)
	}
	if result.AccountsMigrated != 1 {
		t.Errorf("AccountsMigrated = %v, want 1", result.AccountsMigrated)
	}
	if len(result.NFTCollections) == 0 {
		t.Error("No NFT collections found")
	}

	// Check output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Genesis file was not created")
	}
}

func TestConvertToXChainAddress(t *testing.T) {
	tests := []struct {
		name    string
		ethAddr string
		want    string
	}{
		{
			name:    "Normal address",
			ethAddr: "0x1234567890123456789012345678901234567890",
			want:    "X-lux112345678901234567890123456789012345678",
		},
		{
			name:    "Uppercase address",
			ethAddr: "0xABCDEF1234567890123456789012345678901234",
			want:    "X-lux1abcdef12345678901234567890123456789012", // Truncated to 38 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToXChainAddress(tt.ethAddr)
			if got != tt.want {
				t.Errorf("convertToXChainAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateAssetID(t *testing.T) {
	// Test that same seed produces same ID
	seed1 := "test_seed"
	id1 := generateAssetID(seed1)
	id2 := generateAssetID(seed1)
	
	if id1 != id2 {
		t.Error("Same seed should produce same asset ID")
	}

	// Test that different seeds produce different IDs
	seed2 := "different_seed"
	id3 := generateAssetID(seed2)
	
	if id1 == id3 {
		t.Error("Different seeds should produce different asset IDs")
	}

	// Test that IDs are valid hex
	if len(id1) != 64 { // SHA256 produces 32 bytes = 64 hex chars
		t.Errorf("Asset ID should be 64 hex chars, got %d", len(id1))
	}
}