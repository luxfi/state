package scanner

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"
	"strings"
)

// ExportTokenBurnsToCSV exports token burns to CSV file
func ExportTokenBurnsToCSV(burns []TokenBurn, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"TxHash", "BlockNumber", "Timestamp", "From", "To",
		"Amount", "TokenAddress", "LogIndex",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Sort by timestamp
	sort.Slice(burns, func(i, j int) bool {
		return burns[i].Timestamp.Before(burns[j].Timestamp)
	})

	// Write data
	for _, burn := range burns {
		record := []string{
			burn.TxHash,
			strconv.FormatUint(burn.BlockNumber, 10),
			burn.Timestamp.Format("2006-01-02 15:04:05"),
			burn.From,
			burn.To,
			burn.Amount,
			burn.TokenAddr,
			strconv.FormatUint(uint64(burn.LogIndex), 10),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportTokenTransfersToCSV exports token transfers to CSV file
func ExportTokenTransfersToCSV(transfers []TokenTransfer, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"TxHash", "BlockNumber", "Timestamp", "From", "To",
		"Amount", "TokenAddress", "LogIndex",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Sort by timestamp
	sort.Slice(transfers, func(i, j int) bool {
		return transfers[i].Timestamp.Before(transfers[j].Timestamp)
	})

	// Write data
	for _, transfer := range transfers {
		record := []string{
			transfer.TxHash,
			strconv.FormatUint(transfer.BlockNumber, 10),
			transfer.Timestamp.Format("2006-01-02 15:04:05"),
			transfer.From,
			transfer.To,
			transfer.Amount,
			transfer.TokenAddr,
			strconv.FormatUint(uint64(transfer.LogIndex), 10),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportNFTHoldersToCSV exports NFT holders to CSV file
func ExportNFTHoldersToCSV(holders []NFTHolder, filename string, metadata map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Address", "TokenCount"}
	if len(holders) > 0 && len(holders[0].TokenIDs) > 0 {
		header = append(header, "TokenIDs")
	}
	// Add metadata columns if provided
	for key := range metadata {
		header = append(header, key)
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Sort by token count (descending) then by address
	sort.Slice(holders, func(i, j int) bool {
		if holders[i].TokenCount != holders[j].TokenCount {
			return holders[i].TokenCount > holders[j].TokenCount
		}
		return holders[i].Address < holders[j].Address
	})

	// Write data
	for _, holder := range holders {
		record := []string{
			holder.Address,
			strconv.Itoa(holder.TokenCount),
		}
		if len(holder.TokenIDs) > 0 {
			record = append(record, strings.Join(holder.TokenIDs, ";"))
		}
		// Add metadata values
		for key := range metadata {
			record = append(record, metadata[key])
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// ExportCrossChainBalancesToCSV exports cross-chain balances to CSV
func ExportCrossChainBalancesToCSV(balances map[string][]CrossChainBalance, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Address", "ChainID", "ChainName", "TokenAddress",
		"Balance", "BlockNumber",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Flatten and sort
	type record struct {
		addr    string
		balance CrossChainBalance
	}
	records := []record{}
	for addr, balanceList := range balances {
		for _, balance := range balanceList {
			records = append(records, record{addr, balance})
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].addr != records[j].addr {
			return records[i].addr < records[j].addr
		}
		return records[i].balance.ChainID < records[j].balance.ChainID
	})

	// Write data
	chainNames := map[int64]string{
		1:      "Ethereum",
		56:     "BSC",
		137:    "Polygon",
		96369:  "Lux Mainnet",
		200200: "Zoo Mainnet",
		36911:  "SPC Mainnet",
	}

	for _, r := range records {
		chainName := chainNames[r.balance.ChainID]
		if chainName == "" {
			chainName = fmt.Sprintf("Chain-%d", r.balance.ChainID)
		}

		row := []string{
			r.addr,
			strconv.FormatInt(r.balance.ChainID, 10),
			chainName,
			r.balance.TokenAddress,
			r.balance.Balance,
			strconv.FormatUint(r.balance.BlockNumber, 10),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ExportBurnSummaryToCSV exports aggregated burn data by address
func ExportBurnSummaryToCSV(burnsByAddress map[string]*big.Int, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Address", "TotalBurned", "TotalBurnedDecimal"}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Convert to slice for sorting
	type burnRecord struct {
		addr   string
		amount *big.Int
	}
	records := []burnRecord{}
	for addr, amount := range burnsByAddress {
		records = append(records, burnRecord{addr, amount})
	}

	// Sort by amount (descending)
	sort.Slice(records, func(i, j int) bool {
		return records[i].amount.Cmp(records[j].amount) > 0
	})

	// Write data
	decimals := big.NewInt(1e18) // Assuming 18 decimals
	for _, r := range records {
		// Calculate decimal representation
		decimalAmount := new(big.Float).SetInt(r.amount)
		decimalAmount.Quo(decimalAmount, new(big.Float).SetInt(decimals))

		row := []string{
			r.addr,
			r.amount.String(),
			decimalAmount.Text('f', 6), // 6 decimal places
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ExportToJSON exports any data structure to JSON file
func ExportToJSON(data interface{}, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// GenerateSummaryReport generates a text summary report
func GenerateSummaryReport(filename string, sections map[string]string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for title, content := range sections {
		fmt.Fprintf(file, "%s\n", title)
		fmt.Fprintf(file, "%s\n", strings.Repeat("=", len(title)))
		fmt.Fprintf(file, "%s\n\n", content)
	}

	return nil
}
