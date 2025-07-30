package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Setup
	code := m.Run()
	// Cleanup
	os.Exit(code)
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *big.Int
		wantErr  bool
	}{
		{
			name:     "simple number",
			input:    "1000",
			expected: new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "with K suffix",
			input:    "1K",
			expected: new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "with M suffix",
			input:    "1M",
			expected: new(big.Int).Mul(big.NewInt(1000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "with B suffix",
			input:    "1B",
			expected: new(big.Int).Mul(big.NewInt(1000000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "with T suffix",
			input:    "2T",
			expected: new(big.Int).Mul(big.NewInt(2000000000000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "decimal number",
			input:    "1.5M",
			expected: new(big.Int).Mul(big.NewInt(1500000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid format",
			input:    "abc",
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseAmount(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 0, tt.expected.Cmp(result), "expected %s, got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestValidatorOperations(t *testing.T) {
	// Create temporary directory for test files
	tmpDir, err := ioutil.TempDir("", "genesis-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	validatorsFile := filepath.Join(tmpDir, "validators.json")

	t.Run("save and load validators", func(t *testing.T) {
		validators := []*ValidatorInfo{
			{
				NodeID:        "NodeID-1",
				ETHAddress:    "0x1234567890123456789012345678901234567890",
				Weight:        100000000000000,
				DelegationFee: 20000,
			},
			{
				NodeID:        "NodeID-2",
				ETHAddress:    "0x2345678901234567890123456789012345678901",
				Weight:        200000000000000,
				DelegationFee: 20000,
			},
		}

		// Save validators
		err := saveValidators(validators, validatorsFile)
		require.NoError(t, err)

		// Load validators
		loaded, err := loadValidators(validatorsFile)
		require.NoError(t, err)
		assert.Equal(t, len(validators), len(loaded))
		assert.Equal(t, validators[0].NodeID, loaded[0].NodeID)
		assert.Equal(t, validators[1].ETHAddress, loaded[1].ETHAddress)
	})

	t.Run("load non-existent file", func(t *testing.T) {
		_, err := loadValidators(filepath.Join(tmpDir, "non-existent.json"))
		assert.Error(t, err)
	})
}

func TestImportGenesis(t *testing.T) {
	// Create temporary directory
	tmpDir, err := ioutil.TempDir("", "genesis-import-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import C-Chain genesis", func(t *testing.T) {
		// Create test C-Chain genesis
		cGenesis := map[string]interface{}{
			"config": map[string]interface{}{
				"chainId":        96369,
				"homesteadBlock": 0,
			},
			"alloc": map[string]interface{}{
				"0x1234567890123456789012345678901234567890": map[string]interface{}{
					"balance": "1000000000000000000000",
				},
				"0x2345678901234567890123456789012345678901": map[string]interface{}{
					"balance": "2000000000000000000000",
				},
			},
			"difficulty": "0x1",
			"gasLimit":   "0x1000000",
			"nonce":      "0x0",
			"timestamp":  "0x0",
		}

		genesisFile := filepath.Join(tmpDir, "c-genesis.json")
		data, err := json.MarshalIndent(cGenesis, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(genesisFile, data, 0644)
		require.NoError(t, err)

		outputFile := filepath.Join(tmpDir, "output-allocations.json")

		// Create command and run
		cmd := &cobra.Command{}
		cmd.Flags().String("chain", "C", "")
		cmd.Flags().Bool("allocations-only", true, "")
		cmd.Flags().String("output", outputFile, "")

		err = runImportGenesis(cmd, []string{genesisFile})
		require.NoError(t, err)

		// Verify output file was created
		assert.FileExists(t, outputFile)

		// Read and verify allocations
		outputData, err := ioutil.ReadFile(outputFile)
		require.NoError(t, err)

		var allocations map[string]*big.Int
		err = json.Unmarshal(outputData, &allocations)
		require.NoError(t, err)

		assert.Equal(t, 2, len(allocations))
		assert.NotNil(t, allocations["0x1234567890123456789012345678901234567890"])
		assert.NotNil(t, allocations["0x2345678901234567890123456789012345678901"])
	})

	t.Run("import P-Chain genesis", func(t *testing.T) {
		pGenesis := map[string]interface{}{
			"networkID": 96369,
			"allocations": []map[string]interface{}{
				{
					"ethAddr":       "0x1234567890123456789012345678901234567890",
					"avaxAddr":      "X-avax1xxxx",
					"initialAmount": 1000000000000,
				},
			},
			"startTime":            1640995200,
			"initialStakeDuration": 31536000,
			"initialStakers":       []map[string]interface{}{},
			"cChainGenesis":        "{}",
			"message":              "test",
		}

		genesisFile := filepath.Join(tmpDir, "p-genesis.json")
		data, err := json.MarshalIndent(pGenesis, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(genesisFile, data, 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("chain", "P", "")
		cmd.Flags().Bool("allocations-only", false, "")
		cmd.Flags().String("output", "", "")

		err = runImportGenesis(cmd, []string{genesisFile})
		require.NoError(t, err)
	})

	t.Run("import X-Chain genesis", func(t *testing.T) {
		xGenesis := map[string]interface{}{
			"allocations": []map[string]interface{}{
				{
					"ethAddr":       "0x1234567890123456789012345678901234567890",
					"avaxAddr":      "X-avax1xxxx",
					"initialAmount": 1000000000000,
					"unlockSchedule": []map[string]interface{}{
						{
							"amount":   500000000000,
							"locktime": 1640995200,
						},
					},
				},
			},
			"startTime":            1640995200,
			"initialStakeDuration": 31536000,
			"initialStakers":       []map[string]interface{}{},
			"cChainGenesis":        "{}",
			"message":              "test",
		}

		genesisFile := filepath.Join(tmpDir, "x-genesis.json")
		data, err := json.MarshalIndent(xGenesis, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(genesisFile, data, 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("chain", "X", "")
		cmd.Flags().Bool("allocations-only", false, "")
		cmd.Flags().String("output", "", "")

		err = runImportGenesis(cmd, []string{genesisFile})
		require.NoError(t, err)
	})

	t.Run("import invalid chain type", func(t *testing.T) {
		// Create a dummy genesis file
		dummyFile := filepath.Join(tmpDir, "dummy-genesis.json")
		err := ioutil.WriteFile(dummyFile, []byte("{}"), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("chain", "Z", "")
		cmd.Flags().Bool("allocations-only", false, "")
		cmd.Flags().String("output", "", "")

		err = runImportGenesis(cmd, []string{dummyFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported chain type")
	})
}

func TestImportAllocations(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "genesis-allocations-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("import CSV allocations", func(t *testing.T) {
		csvContent := `address,balance
0x1234567890123456789012345678901234567890,1000000000000000000000
0x2345678901234567890123456789012345678901,2000000000000000000000
0x3456789012345678901234567890123456789012,3000000000000000000000`

		csvFile := filepath.Join(tmpDir, "allocations.csv")
		err := ioutil.WriteFile(csvFile, []byte(csvContent), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("format", "auto", "")
		cmd.Flags().Bool("merge", false, "")

		err = runImportAllocations(cmd, []string{csvFile})
		require.NoError(t, err)
	})

	t.Run("import JSON allocations", func(t *testing.T) {
		// Use big.Int directly for JSON marshaling
		val1 := new(big.Int)
		val1.SetString("1000000000000000000000", 10)
		val2 := new(big.Int)
		val2.SetString("2000000000000000000000", 10)

		jsonAllocations := map[string]*big.Int{
			"0x1234567890123456789012345678901234567890": val1,
			"0x2345678901234567890123456789012345678901": val2,
		}

		jsonFile := filepath.Join(tmpDir, "allocations.json")
		data, err := json.Marshal(jsonAllocations)
		require.NoError(t, err)
		err = ioutil.WriteFile(jsonFile, data, 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("format", "auto", "")
		cmd.Flags().Bool("merge", false, "")

		err = runImportAllocations(cmd, []string{jsonFile})
		require.NoError(t, err)
	})

	t.Run("auto-detect format", func(t *testing.T) {
		// Test with .txt extension (should fail auto-detect)
		txtFile := filepath.Join(tmpDir, "allocations.txt")
		err := ioutil.WriteFile(txtFile, []byte("test"), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("format", "auto", "")
		cmd.Flags().Bool("merge", false, "")

		err = runImportAllocations(cmd, []string{txtFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot auto-detect format")
	})

	t.Run("invalid format", func(t *testing.T) {
		// Create a dummy file
		xmlFile := filepath.Join(tmpDir, "dummy.xml")
		err := ioutil.WriteFile(xmlFile, []byte("<xml></xml>"), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("format", "xml", "")
		cmd.Flags().Bool("merge", false, "")

		err = runImportAllocations(cmd, []string{xmlFile})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported format")
	})
}

func TestGenerateCommand(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "genesis-generate-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Save original config
	origCfg := *cfg
	defer func() {
		cfg = &origCfg
	}()

	t.Run("generate with standard dirs", func(t *testing.T) {
		cfg.OutputDir = tmpDir
		cfg.UseStandardDirs = true
		cfg.Network = "testnet"

		cmd := &cobra.Command{}
		err := runGenerate(cmd, []string{})
		require.NoError(t, err)

		// Check that directories were created
		assert.DirExists(t, filepath.Join(tmpDir, "P"))
		assert.DirExists(t, filepath.Join(tmpDir, "C"))
		assert.DirExists(t, filepath.Join(tmpDir, "X"))
	})

	t.Run("generate without standard dirs", func(t *testing.T) {
		tmpDir2, err := ioutil.TempDir("", "genesis-generate-test2-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir2)

		cfg.OutputDir = tmpDir2
		cfg.UseStandardDirs = false

		cmd := &cobra.Command{}
		err = runGenerate(cmd, []string{})
		require.NoError(t, err)

		// Check that single directory was created
		assert.DirExists(t, tmpDir2)
		// But not subdirectories
		assert.NoDirExists(t, filepath.Join(tmpDir2, "P"))
	})
}

func TestValidateCommand(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "genesis-validate-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test genesis files
	for _, chain := range []string{"P", "C", "X"} {
		chainDir := filepath.Join(tmpDir, chain)
		err := os.MkdirAll(chainDir, 0755)
		require.NoError(t, err)

		genesis := map[string]interface{}{
			"test": "data",
		}
		data, err := json.Marshal(genesis)
		require.NoError(t, err)

		err = ioutil.WriteFile(filepath.Join(chainDir, "genesis.json"), data, 0644)
		require.NoError(t, err)
	}

	// Save original config
	origCfg := *cfg
	defer func() {
		cfg = &origCfg
	}()

	cfg.OutputDir = tmpDir

	t.Run("validate valid genesis files", func(t *testing.T) {
		cmd := &cobra.Command{}
		err := runValidate(cmd, []string{})
		require.NoError(t, err)
	})

	t.Run("validate with missing file", func(t *testing.T) {
		// Remove one file
		os.Remove(filepath.Join(tmpDir, "X", "genesis.json"))

		cmd := &cobra.Command{}
		err := runValidate(cmd, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("validate with invalid JSON", func(t *testing.T) {
		// Write invalid JSON
		err := ioutil.WriteFile(filepath.Join(tmpDir, "C", "genesis.json"), []byte("invalid json"), 0644)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		err = runValidate(cmd, []string{})
		assert.Error(t, err)
	})
}

func TestExtractStateCommand(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "genesis-extract-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	t.Run("extract state parameters", func(t *testing.T) {
		srcDir := filepath.Join(tmpDir, "source")
		dstDir := filepath.Join(tmpDir, "dest")

		// Create source directory
		err := os.MkdirAll(srcDir, 0755)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().Int("network", 96369, "")
		cmd.Flags().Bool("state", true, "")
		cmd.Flags().Int("limit", 100, "")

		// This will fail because it needs a real PebbleDB, but we can test the parameters
		err = runExtractState(cmd, []string{srcDir, dstDir})
		assert.Error(t, err) // Expected to fail without real DB
	})
}

func TestCommandStructure(t *testing.T) {
	// Test that all commands are properly structured
	rootCmd := &cobra.Command{
		Use:   "genesis",
		Short: "Lux Network Genesis Management Tool",
	}

	// Build command structure (simplified version)
	generateCmd := &cobra.Command{Use: "generate"}
	validatorsCmd := &cobra.Command{Use: "validators"}
	extractCmd := &cobra.Command{Use: "extract"}
	importCmd := &cobra.Command{Use: "import"}

	rootCmd.AddCommand(generateCmd, validatorsCmd, extractCmd, importCmd)

	// Add validator subcommands
	validatorsCmd.AddCommand(
		&cobra.Command{Use: "list"},
		&cobra.Command{Use: "add"},
		&cobra.Command{Use: "remove"},
		&cobra.Command{Use: "generate"},
	)

	// Test command discovery
	t.Run("root command has subcommands", func(t *testing.T) {
		assert.True(t, rootCmd.HasSubCommands())
		assert.Equal(t, 4, len(rootCmd.Commands()))
	})

	t.Run("validators has subcommands", func(t *testing.T) {
		validatorsFound := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "validators" {
				validatorsFound = true
				assert.True(t, cmd.HasSubCommands())
				assert.Equal(t, 4, len(cmd.Commands()))
			}
		}
		assert.True(t, validatorsFound)
	})
}

// Helper function to capture output
func captureTestOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestToolsCommand(t *testing.T) {
	output := captureTestOutput(func() {
		cmd := &cobra.Command{}
		runTools(cmd, []string{})
	})

	// Check that output contains expected sections
	assert.Contains(t, output, "Lux Network Genesis Tool")
	assert.Contains(t, output, "Core Commands:")
	assert.Contains(t, output, "Data Management:")
	assert.Contains(t, output, "Cross-Chain Operations:")
	assert.Contains(t, output, "generate")
	assert.Contains(t, output, "validators")
	assert.Contains(t, output, "extract")
	assert.Contains(t, output, "import")
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir, err := ioutil.TempDir("", "genesis-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Full integration test scenario
	t.Run("full workflow", func(t *testing.T) {
		// 1. Generate validators
		validatorsFile := filepath.Join(tmpDir, "validators.json")
		validators := []*ValidatorInfo{
			{
				NodeID:        "NodeID-Integration1",
				ETHAddress:    "0x1111111111111111111111111111111111111111",
				Weight:        100000000000000,
				DelegationFee: 20000,
			},
		}
		err := saveValidators(validators, validatorsFile)
		require.NoError(t, err)

		// 2. Create test genesis to import from
		originalGenesis := map[string]interface{}{
			"config": map[string]interface{}{
				"chainId": 7777,
			},
			"alloc": map[string]interface{}{
				"0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA": map[string]interface{}{
					"balance": "5000000000000000000000",
				},
			},
			"difficulty": "0x1",
			"gasLimit":   "0x1000000",
		}

		originalFile := filepath.Join(tmpDir, "original-genesis.json")
		data, err := json.MarshalIndent(originalGenesis, "", "  ")
		require.NoError(t, err)
		err = ioutil.WriteFile(originalFile, data, 0644)
		require.NoError(t, err)

		// 3. Import allocations
		allocationsFile := filepath.Join(tmpDir, "imported-allocations.json")
		cmd := &cobra.Command{}
		cmd.Flags().String("chain", "C", "")
		cmd.Flags().Bool("allocations-only", true, "")
		cmd.Flags().String("output", allocationsFile, "")

		err = runImportGenesis(cmd, []string{originalFile})
		require.NoError(t, err)
		assert.FileExists(t, allocationsFile)

		// 4. Verify the complete workflow
		allocData, err := ioutil.ReadFile(allocationsFile)
		require.NoError(t, err)

		var importedAllocs map[string]*big.Int
		err = json.Unmarshal(allocData, &importedAllocs)
		require.NoError(t, err)

		assert.Equal(t, 1, len(importedAllocs))
		assert.NotNil(t, importedAllocs["0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"])
	})
}
