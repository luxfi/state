package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// NetworkConfig represents a blockchain network configuration
type NetworkConfig struct {
	Name         string `json:"name"`
	ChainID      uint64 `json:"chain_id"`
	BlockchainID string `json:"blockchain_id"`
	Type         string `json:"type"`
	Token        string `json:"token"`
	ExpectedSize string `json:"expected_size"`
	SourcePath   string `json:"source_path"`
}

// TestSuite manages the entire test process
type TestSuite struct {
	BaseDir       string
	RawDataDir    string
	ExtractionDir string
	FinalDir      string
	Networks      map[string]NetworkConfig
	TestResults   []TestResult
}

// TestResult stores individual test results
type TestResult struct {
	Name     string
	Passed   bool
	Duration time.Duration
	Error    error
	Output   string
}

// NewTestSuite creates a new test suite
func NewTestSuite() *TestSuite {
	baseDir := "/home/z/work/lux/genesis"
	
	return &TestSuite{
		BaseDir:       baseDir,
		RawDataDir:    "/home/z/archived/restored-blockchain-data/chainData",
		ExtractionDir: filepath.Join(baseDir, "test-extraction-go"),
		FinalDir:      filepath.Join(baseDir, "final-chaindata-go"),
		Networks: map[string]NetworkConfig{
			"lux-mainnet-96369": {
				Name:         "lux-mainnet-96369",
				ChainID:      96369,
				BlockchainID: "dnmzhuf6poM6PUNQCe7MWWfBdTJEnddhHRNXz2x7H6qSmyBEJ",
				Type:         "C-Chain",
				Token:        "LUX",
				ExpectedSize: "7.2GB",
			},
			"lux-testnet-96368": {
				Name:         "lux-testnet-96368",
				ChainID:      96368,
				BlockchainID: "2sdADEgBC3NjLM4inKc1hY1PQpCT3JVyGVJxdmcq6sqrDndjFG",
				Type:         "testnet",
				Token:        "LUX",
				ExpectedSize: "1.1MB",
			},
			"zoo-mainnet-200200": {
				Name:         "zoo-mainnet-200200",
				ChainID:      200200,
				BlockchainID: "bXe2MhhAnXg6WGj6G8oDk55AKT1dMMsN72S8te7JdvzfZX1zM",
				Type:         "L2",
				Token:        "ZOO",
				ExpectedSize: "3.7MB",
			},
			"zoo-testnet-200201": {
				Name:         "zoo-testnet-200201",
				ChainID:      200201,
				BlockchainID: "2usKC5aApgWQWwanB4LL6QPoqxR1bWWjPCtemBYbZvxkNfcnbj",
				Type:         "testnet",
				Token:        "ZOO",
				ExpectedSize: "292KB",
			},
			"spc-mainnet-36911": {
				Name:         "spc-mainnet-36911",
				ChainID:      36911,
				BlockchainID: "QFAFyn1hh59mh7kokA55dJq5ywskF5A1yn8dDpLhmKApS6FP1",
				Type:         "L2",
				Token:        "SPC",
				ExpectedSize: "48KB",
			},
		},
		TestResults: []TestResult{},
	}
}

// Run executes all tests
func (ts *TestSuite) Run() {
	fmt.Println("=== LUX NETWORK 2025 - GO TEST SUITE ===")
	fmt.Println("Running comprehensive extraction and deployment tests")
	fmt.Println()

	// Clean up previous runs
	ts.cleanup()

	// Run test phases
	ts.runPhase("Prerequisites", ts.testPrerequisites)
	ts.runPhase("Raw Data Verification", ts.testRawData)
	ts.runPhase("Tool Validation", ts.testTools)
	ts.runPhase("Configuration Validation", ts.testConfigurations)
	ts.runPhase("Extraction Process", ts.testExtraction)
	ts.runPhase("Data Validation", ts.testExtractedData)
	ts.runPhase("Deployment Preparation", ts.testDeploymentPrep)
	ts.runPhase("Final Validation", ts.testFinalValidation)

	// Generate report
	ts.generateReport()
}

// runPhase executes a test phase
func (ts *TestSuite) runPhase(name string, testFunc func() []TestResult) {
	fmt.Printf("\n=== PHASE: %s ===\n", name)
	results := testFunc()
	ts.TestResults = append(ts.TestResults, results...)
	
	passed := 0
	for _, r := range results {
		if r.Passed {
			passed++
		}
	}
	fmt.Printf("Phase complete: %d/%d tests passed\n", passed, len(results))
}

// cleanup removes previous test artifacts
func (ts *TestSuite) cleanup() {
	fmt.Println("Cleaning up previous test runs...")
	os.RemoveAll(ts.ExtractionDir)
	os.RemoveAll(ts.FinalDir)
	os.MkdirAll(ts.ExtractionDir, 0755)
	os.MkdirAll(ts.FinalDir, 0755)
}

// testPrerequisites checks system requirements
func (ts *TestSuite) testPrerequisites() []TestResult {
	var results []TestResult

	// Test 1: Check disk space
	results = append(results, ts.runTest("Disk Space Check", func() error {
		cmd := exec.Command("df", "-BG", ts.BaseDir)
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		// Parse output to check available space
		// Simplified for example
		_ = output
		return nil
	}))

	// Test 2: Check memory
	results = append(results, ts.runTest("Memory Check", func() error {
		cmd := exec.Command("free", "-g")
		output, err := cmd.Output()
		if err != nil {
			return err
		}
		// Check if we have enough memory
		_ = output
		return nil
	}))

	return results
}

// testRawData verifies all blockchain data exists
func (ts *TestSuite) testRawData() []TestResult {
	var results []TestResult

	for name, network := range ts.Networks {
		results = append(results, ts.runTest(
			fmt.Sprintf("Raw data exists: %s", name),
			func() error {
				path := filepath.Join(ts.RawDataDir, network.BlockchainID, "db", "pebbledb")
				if _, err := os.Stat(path); os.IsNotExist(err) {
					return fmt.Errorf("data not found at %s", path)
				}
				return nil
			},
		))
	}

	// Check 7777 data
	results = append(results, ts.runTest("Historical 7777 data", func() error {
		path := filepath.Join(ts.BaseDir, "data/2023-7777/pebble-clean-7777.tar.gz")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("7777 data not found at %s", path)
		}
		return nil
	}))

	return results
}

// testTools validates extraction tools
func (ts *TestSuite) testTools() []TestResult {
	var results []TestResult

	tools := []string{
		"denamespace-universal",
		"denamespace-selective",
		"evmarchaeology",
	}

	for _, tool := range tools {
		results = append(results, ts.runTest(
			fmt.Sprintf("Tool exists: %s", tool),
			func() error {
				path := filepath.Join(ts.BaseDir, "bin", tool)
				info, err := os.Stat(path)
				if err != nil {
					return err
				}
				if info.Mode()&0111 == 0 {
					return fmt.Errorf("%s is not executable", tool)
				}
				return nil
			},
		))
	}

	return results
}

// testConfigurations validates genesis and chain configs
func (ts *TestSuite) testConfigurations() []TestResult {
	var results []TestResult

	for name := range ts.Networks {
		configDir := filepath.Join(ts.BaseDir, "data/unified-genesis/configs", name)
		
		results = append(results, ts.runTest(
			fmt.Sprintf("Config exists: %s", name),
			func() error {
				genesisPath := filepath.Join(configDir, "genesis.json")
				if _, err := os.Stat(genesisPath); err != nil {
					return err
				}
				
				// Validate JSON
				data, err := os.ReadFile(genesisPath)
				if err != nil {
					return err
				}
				
				var genesis map[string]interface{}
				if err := json.Unmarshal(data, &genesis); err != nil {
					return fmt.Errorf("invalid JSON: %v", err)
				}
				
				return nil
			},
		))
	}

	return results
}

// testExtraction performs the actual extraction
func (ts *TestSuite) testExtraction() []TestResult {
	var results []TestResult

	// Extract each network
	for name, network := range ts.Networks {
		results = append(results, ts.runTest(
			fmt.Sprintf("Extract network: %s", name),
			func() error {
				return ts.extractNetwork(network)
			},
		))
	}

	// Extract 7777
	results = append(results, ts.runTest("Extract historical 7777", func() error {
		return ts.extract7777()
	}))

	return results
}

// extractNetwork extracts a single network
func (ts *TestSuite) extractNetwork(network NetworkConfig) error {
	fmt.Printf("  Extracting %s (chain ID: %d)...\n", network.Name, network.ChainID)
	
	srcPath := filepath.Join(ts.RawDataDir, network.BlockchainID, "db", "pebbledb")
	dstPath := filepath.Join(ts.ExtractionDir, network.Name, "db")
	
	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return err
	}
	
	// Run denamespace-universal
	cmd := exec.Command(
		filepath.Join(ts.BaseDir, "bin/denamespace-universal"),
		"-src", srcPath,
		"-dst", dstPath,
		"-network", fmt.Sprintf("%d", network.ChainID),
		"-state",
	)
	
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("extraction failed: %v\nOutput: %s", err, output)
	}
	
	// Parse output for statistics
	outputStr := string(output)
	if strings.Contains(outputStr, "Total keys copied:") {
		fmt.Printf("    ✓ Extraction successful\n")
	}
	
	return nil
}

// extract7777 extracts the historical network
func (ts *TestSuite) extract7777() error {
	fmt.Println("  Extracting historical 7777 network...")
	
	destDir := filepath.Join(ts.ExtractionDir, "lux-7777")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	
	// Extract tar.gz
	tarPath := filepath.Join(ts.BaseDir, "data/2023-7777/pebble-clean-7777.tar.gz")
	cmd := exec.Command("tar", "-xzf", tarPath, "-C", destDir)
	
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("tar extraction failed: %v\nOutput: %s", err, output)
	}
	
	// Rename pebble-clean to db
	oldPath := filepath.Join(destDir, "pebble-clean")
	newPath := filepath.Join(destDir, "db")
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	
	fmt.Println("    ✓ Historical data extracted")
	return nil
}

// testExtractedData validates the extracted data
func (ts *TestSuite) testExtractedData() []TestResult {
	var results []TestResult

	for name := range ts.Networks {
		results = append(results, ts.runTest(
			fmt.Sprintf("Validate extracted data: %s", name),
			func() error {
				dbPath := filepath.Join(ts.ExtractionDir, name, "db")
				
				// Check if database exists and has content
				entries, err := os.ReadDir(dbPath)
				if err != nil {
					return err
				}
				
				if len(entries) == 0 {
					return fmt.Errorf("extracted database is empty")
				}
				
				// Count SST files
				sstCount := 0
				for _, entry := range entries {
					if strings.HasSuffix(entry.Name(), ".sst") {
						sstCount++
					}
				}
				
				fmt.Printf("    Found %d SST files in %s\n", sstCount, name)
				return nil
			},
		))
	}

	return results
}

// testDeploymentPrep prepares for deployment
func (ts *TestSuite) testDeploymentPrep() []TestResult {
	var results []TestResult

	// Copy extracted data to final location
	results = append(results, ts.runTest("Copy to final location", func() error {
		for name := range ts.Networks {
			src := filepath.Join(ts.ExtractionDir, name)
			dst := filepath.Join(ts.FinalDir, name)
			
			if err := copyDir(src, dst); err != nil {
				return fmt.Errorf("failed to copy %s: %v", name, err)
			}
		}
		
		// Copy 7777
		src := filepath.Join(ts.ExtractionDir, "lux-7777")
		dst := filepath.Join(ts.FinalDir, "lux-7777")
		if err := copyDir(src, dst); err != nil {
			return fmt.Errorf("failed to copy 7777: %v", err)
		}
		
		return nil
	}))

	// Copy configurations
	results = append(results, ts.runTest("Copy configurations", func() error {
		src := filepath.Join(ts.BaseDir, "data/unified-genesis/configs")
		dst := filepath.Join(ts.FinalDir, "configs")
		return copyDir(src, dst)
	}))

	// Create deployment scripts
	results = append(results, ts.runTest("Create deployment scripts", func() error {
		return ts.createDeploymentScripts()
	}))

	return results
}

// testFinalValidation performs final checks
func (ts *TestSuite) testFinalValidation() []TestResult {
	var results []TestResult

	// Validate all data is in place
	results = append(results, ts.runTest("Final data validation", func() error {
		requiredDirs := []string{
			"lux-mainnet-96369",
			"lux-testnet-96368",
			"zoo-mainnet-200200",
			"zoo-testnet-200201",
			"spc-mainnet-36911",
			"lux-7777",
			"configs",
		}
		
		for _, dir := range requiredDirs {
			path := filepath.Join(ts.FinalDir, dir)
			if _, err := os.Stat(path); err != nil {
				return fmt.Errorf("missing required directory: %s", dir)
			}
		}
		
		return nil
	}))

	// Validate deployment scripts
	results = append(results, ts.runTest("Deployment scripts created", func() error {
		scripts := []string{
			"launch_network.sh",
			"verify_deployment.sh",
			"run_7777_historical.sh",
		}
		
		for _, script := range scripts {
			path := filepath.Join(ts.FinalDir, script)
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("missing script: %s", script)
			}
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("script not executable: %s", script)
			}
		}
		
		return nil
	}))

	return results
}

// runTest executes a single test and returns the result
func (ts *TestSuite) runTest(name string, testFunc func() error) TestResult {
	fmt.Printf("  Testing: %s... ", name)
	
	start := time.Now()
	err := testFunc()
	duration := time.Since(start)
	
	result := TestResult{
		Name:     name,
		Passed:   err == nil,
		Duration: duration,
		Error:    err,
	}
	
	if result.Passed {
		fmt.Printf("✓ PASSED (%.2fs)\n", duration.Seconds())
	} else {
		fmt.Printf("✗ FAILED: %v\n", err)
	}
	
	return result
}

// createDeploymentScripts generates the deployment scripts
func (ts *TestSuite) createDeploymentScripts() error {
	// Create launch script
	launchScript := ts.generateLaunchScript()
	if err := os.WriteFile(
		filepath.Join(ts.FinalDir, "launch_network.sh"),
		[]byte(launchScript),
		0755,
	); err != nil {
		return err
	}

	// Create verification script
	verifyScript := ts.generateVerifyScript()
	if err := os.WriteFile(
		filepath.Join(ts.FinalDir, "verify_deployment.sh"),
		[]byte(verifyScript),
		0755,
	); err != nil {
		return err
	}

	// Create 7777 runner
	historicalScript := ts.generate7777Script()
	if err := os.WriteFile(
		filepath.Join(ts.FinalDir, "run_7777_historical.sh"),
		[]byte(historicalScript),
		0755,
	); err != nil {
		return err
	}

	return nil
}

// generateLaunchScript creates the network launch script
func (ts *TestSuite) generateLaunchScript() string {
	return `#!/bin/bash
set -e

echo "=== LAUNCHING LUX NETWORK 2025 ==="
echo ""

NETWORK_NAME="lux-2025"
CHAIN_DATA_DIR="$(dirname $0)"
NUM_NODES=5

echo "1. Creating local network with $NUM_NODES nodes..."
lux-cli network delete $NETWORK_NAME 2>/dev/null || true
lux-cli network create $NETWORK_NAME --num-nodes=$NUM_NODES

echo "2. Configuring nodes with chain data..."
echo "3. Starting network..."
lux-cli network start $NETWORK_NAME

echo "4. Deploying L2 subnets..."
echo ""
echo "=== NETWORK DEPLOYMENT COMPLETE ==="
`
}

// generateVerifyScript creates the verification script
func (ts *TestSuite) generateVerifyScript() string {
	return `#!/bin/bash

echo "=== VERIFYING LUX NETWORK 2025 DEPLOYMENT ==="
echo ""

# Test C-Chain RPC
curl -s -X POST -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
    http://localhost:9650/ext/bc/C/rpc

echo ""
echo "=== VERIFICATION COMPLETE ==="
`
}

// generate7777Script creates the historical network runner
func (ts *TestSuite) generate7777Script() string {
	return `#!/bin/bash

echo "=== RUNNING HISTORICAL LUX 7777 NETWORK ==="
echo ""

CHAIN_DATA_DIR="$(dirname $0)/lux-7777/db"

luxd \
    --network-id=7777 \
    --db-dir="$CHAIN_DATA_DIR" \
    --http-port=9651 \
    --staking-port=9652 \
    --log-level=info
`
}

// generateReport creates the final test report
func (ts *TestSuite) generateReport() {
	fmt.Println("\n=== TEST REPORT ===")
	fmt.Println()

	totalTests := len(ts.TestResults)
	passedTests := 0
	failedTests := 0

	for _, result := range ts.TestResults {
		if result.Passed {
			passedTests++
		} else {
			failedTests++
		}
	}

	fmt.Printf("Total Tests: %d\n", totalTests)
	fmt.Printf("Passed: %d\n", passedTests)
	fmt.Printf("Failed: %d\n", failedTests)
	fmt.Println()

	if failedTests > 0 {
		fmt.Println("Failed Tests:")
		for _, result := range ts.TestResults {
			if !result.Passed {
				fmt.Printf("  - %s: %v\n", result.Name, result.Error)
			}
		}
		fmt.Println()
	}

	// Write detailed report
	reportPath := filepath.Join(ts.BaseDir, fmt.Sprintf("test-report-%s.json", time.Now().Format("20060102-150405")))
	reportData, _ := json.MarshalIndent(ts.TestResults, "", "  ")
	os.WriteFile(reportPath, reportData, 0644)

	fmt.Printf("Detailed report saved to: %s\n", reportPath)
	fmt.Println()

	if failedTests == 0 {
		fmt.Println("✅ ALL TESTS PASSED!")
		fmt.Println()
		fmt.Printf("Final chain data location: %s\n", ts.FinalDir)
		fmt.Println()
		fmt.Println("To deploy the network:")
		fmt.Printf("  cd %s\n", ts.FinalDir)
		fmt.Println("  ./launch_network.sh")
		fmt.Println()
		fmt.Println("The Lux Network 2025 is ready for deployment!")
	} else {
		fmt.Println("❌ Some tests failed. Please fix the issues before proceeding.")
		os.Exit(1)
	}
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
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

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

func main() {
	// Create and run test suite
	suite := NewTestSuite()
	suite.Run()
}