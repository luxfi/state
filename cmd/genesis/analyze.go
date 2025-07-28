package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cockroachdb/pebble"
	"github.com/spf13/cobra"
)

// Analysis flags
var (
	analyzeLimit     int
	analyzeDetailed  bool
	analyzeSubnet    bool
	analyzeNetwork   int
	analyzeAccount   string
)

// NewAnalyzeCommand creates the analyze command with all subcommands
func NewAnalyzeCommand() *cobra.Command {
	analyzeCmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze blockchain data",
		Long: `Analyze various aspects of blockchain data.
		
Available analyses:
- keys: Analyze database key structures
- blocks: Analyze block data and heights
- subnet: Analyze subnet-specific data
- structure: Analyze overall data structure
- balance: Analyze account balances`,
	}

	// Add subcommands
	analyzeCmd.AddCommand(
		newAnalyzeKeysCmd(),
		newAnalyzeBlocksCmd(),
		newAnalyzeSubnetCmd(),
		newAnalyzeStructureCmd(),
		newAnalyzeBalanceCmd(),
	)

	return analyzeCmd
}

// newAnalyzeKeysCmd creates the keys analysis command
func newAnalyzeKeysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys <database-path>",
		Short: "Analyze database key structures",
		Long: `Analyze the structure and patterns of database keys.
		
This includes:
- Key prefixes and their meanings
- Key length distributions
- Namespace analysis
- EVMN key format detection`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyzeKeys,
	}

	cmd.Flags().IntVar(&analyzeLimit, "limit", 1000, "Maximum keys to analyze")
	cmd.Flags().BoolVar(&analyzeDetailed, "detailed", false, "Show detailed analysis")

	return cmd
}

// newAnalyzeBlocksCmd creates the blocks analysis command
func newAnalyzeBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blocks <database-path>",
		Short: "Analyze block data",
		Long: `Analyze blockchain block data.
		
This includes:
- Block height ranges
- Block size distribution
- Transaction counts
- Gas usage patterns`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyzeBlocks,
	}

	cmd.Flags().BoolVar(&analyzeSubnet, "subnet", false, "Analyze as subnet data")
	cmd.Flags().IntVar(&analyzeNetwork, "network", 96369, "Network ID for analysis")

	return cmd
}

// newAnalyzeSubnetCmd creates the subnet analysis command
func newAnalyzeSubnetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subnet <database-path>",
		Short: "Analyze subnet-specific data",
		Long: `Analyze data specific to subnet databases.
		
This includes:
- Subnet-specific key patterns
- Cross-subnet references
- Validator information
- Token allocations`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyzeSubnet,
	}

	return cmd
}

// newAnalyzeStructureCmd creates the structure analysis command
func newAnalyzeStructureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "structure <database-path>",
		Short: "Analyze overall data structure",
		Long: `Analyze the overall structure of the database.
		
This includes:
- Database organization
- Key-value relationships
- Data integrity checks
- Storage efficiency`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyzeStructure,
	}

	return cmd
}

// newAnalyzeBalanceCmd creates the balance analysis command
func newAnalyzeBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance <database-path>",
		Short: "Analyze account balances",
		Long: `Analyze account balances and token distributions.
		
This includes:
- Total supply calculation
- Top holders analysis
- Balance distribution
- Zero balance accounts`,
		Args: cobra.ExactArgs(1),
		RunE: runAnalyzeBalance,
	}

	cmd.Flags().StringVar(&analyzeAccount, "account", "", "Specific account to analyze")

	return cmd
}

// Command implementations

func runAnalyzeKeys(cmd *cobra.Command, args []string) error {
	dbPath := args[0]
	
	fmt.Printf("=== Analyzing Keys in %s ===\n", dbPath)
	fmt.Printf("Limit: %d keys\n", analyzeLimit)
	fmt.Printf("Detailed: %v\n", analyzeDetailed)
	fmt.Println()

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Track key patterns
	prefixCount := make(map[string]int)
	lengthCount := make(map[int]int)
	evmnCount := 0
	
	iter, err := db.NewIter(&pebble.IterOptions{})
	if err != nil {
		return err
	}
	defer iter.Close()

	count := 0
	for iter.First(); iter.Valid() && count < analyzeLimit; iter.Next() {
		key := iter.Key()
		count++

		// Analyze prefix
		prefix := getKeyPrefix(key)
		prefixCount[prefix]++

		// Analyze length
		lengthCount[len(key)]++

		// Check for EVMN pattern
		if strings.HasPrefix(string(key), "evmn") {
			evmnCount++
			if analyzeDetailed {
				fmt.Printf("EVMN key found: %x (len=%d)\n", key, len(key))
			}
		}
	}

	// Display results
	fmt.Printf("\nAnalyzed %d keys\n", count)
	
	fmt.Println("\nKey Prefixes:")
	displaySortedMap(prefixCount)

	fmt.Println("\nKey Lengths:")
	displaySortedIntMap(lengthCount)

	if evmnCount > 0 {
		fmt.Printf("\nEVMN Keys: %d (%.2f%%)\n", evmnCount, float64(evmnCount)/float64(count)*100)
	}

	return nil
}

func runAnalyzeBlocks(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Analyzing Blocks in %s ===\n", dbPath)
	fmt.Printf("Network: %d\n", analyzeNetwork)
	fmt.Printf("Subnet mode: %v\n", analyzeSubnet)
	fmt.Println()

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Find block headers
	minBlock := uint64(^uint64(0))
	maxBlock := uint64(0)
	blockCount := 0

	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: []byte("h"),
		UpperBound: []byte("i"),
	})
	if err != nil {
		return err
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) == 41 && key[0] == 'h' { // Header key
			blockNum := decodeBlockNumber(key[1:])
			if blockNum < minBlock {
				minBlock = blockNum
			}
			if blockNum > maxBlock {
				maxBlock = blockNum
			}
			blockCount++
		}
	}

	if blockCount == 0 {
		fmt.Println("No blocks found in database")
		return nil
	}

	fmt.Printf("Block Range: %d - %d\n", minBlock, maxBlock)
	fmt.Printf("Total Blocks: %d\n", blockCount)
	fmt.Printf("Expected Blocks: %d\n", maxBlock-minBlock+1)
	
	if blockCount < int(maxBlock-minBlock+1) {
		fmt.Printf("Missing Blocks: %d\n", int(maxBlock-minBlock+1)-blockCount)
	}

	return nil
}

func runAnalyzeSubnet(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Analyzing Subnet Data in %s ===\n", dbPath)
	fmt.Println()

	// Subnet-specific analysis
	fmt.Println("Checking for subnet-specific patterns...")
	
	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Look for subnet-specific keys
	subnetPatterns := []string{
		"subnet:",
		"validator:",
		"delegation:",
		"staking:",
	}

	for _, pattern := range subnetPatterns {
		count := countKeysWithPrefix(db, []byte(pattern))
		if count > 0 {
			fmt.Printf("%s keys: %d\n", pattern, count)
		}
	}

	return nil
}

func runAnalyzeStructure(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Analyzing Data Structure in %s ===\n", dbPath)
	fmt.Println()

	db, err := pebble.Open(dbPath, &pebble.Options{ReadOnly: true})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Analyze overall structure
	metrics := db.Metrics()
	fmt.Printf("Database Metrics:\n")
	fmt.Printf("  Levels: %d\n", len(metrics.Levels))
	// Note: Total metrics are computed from individual levels
	var totalSize int64
	var totalFiles int64
	for _, level := range metrics.Levels {
		totalSize += level.Size
		totalFiles += level.NumFiles
	}
	fmt.Printf("  Total Size: %.2f MB\n", float64(totalSize)/1024/1024)
	fmt.Printf("  Table Count: %d\n", totalFiles)

	// Key categories
	categories := map[string]string{
		"h": "Headers",
		"b": "Bodies",
		"r": "Receipts",
		"H": "Hash->Number",
		"n": "Number->Hash",
		"t": "Transactions",
		"s": "State",
		"c": "Code",
		"evmn": "Canonical",
	}

	fmt.Println("\nKey Categories:")
	for prefix, name := range categories {
		count := countKeysWithPrefix(db, []byte(prefix))
		if count > 0 {
			fmt.Printf("  %s (%s): %d\n", name, prefix, count)
		}
	}

	return nil
}

func runAnalyzeBalance(cmd *cobra.Command, args []string) error {
	dbPath := args[0]

	fmt.Printf("=== Analyzing Balances in %s ===\n", dbPath)
	if analyzeAccount != "" {
		fmt.Printf("Account: %s\n", analyzeAccount)
	}
	fmt.Println()

	// TODO: Implement balance analysis
	fmt.Println("Balance analysis not yet implemented")

	return nil
}

// Helper functions

func getKeyPrefix(key []byte) string {
	if len(key) == 0 {
		return "<empty>"
	}
	
	// Check for text prefixes
	if len(key) >= 4 {
		prefix := string(key[:4])
		if strings.HasPrefix(prefix, "evmn") {
			return "evmn"
		}
	}
	
	// Return hex of first byte
	return fmt.Sprintf("%02x", key[0])
}

func displaySortedMap(m map[string]int) {
	// Sort keys
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Display sorted
	for _, k := range keys {
		fmt.Printf("  %s: %d\n", k, m[k])
	}
}

func displaySortedIntMap(m map[int]int) {
	// Sort keys
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Display sorted
	for _, k := range keys {
		fmt.Printf("  %d bytes: %d keys\n", k, m[k])
	}
}

func decodeBlockNumber(data []byte) uint64 {
	if len(data) != 8 {
		return 0
	}
	var num uint64
	for i := 0; i < 8; i++ {
		num = (num << 8) | uint64(data[i])
	}
	return num
}

func countKeysWithPrefix(db *pebble.DB, prefix []byte) int {
	count := 0
	iter, err := db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: incrementBytes(prefix),
	})
	if err != nil {
		return 0
	}
	defer iter.Close()

	for iter.First(); iter.Valid(); iter.Next() {
		count++
	}
	return count
}

func incrementBytes(b []byte) []byte {
	result := make([]byte, len(b))
	copy(result, b)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] < 255 {
			result[i]++
			break
		}
		result[i] = 0
	}
	return result
}