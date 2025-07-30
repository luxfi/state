package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Exporter handles exporting scan results to various formats
type Exporter struct {
	outputPath string
}

// NewExporter creates a new exporter
func NewExporter(outputPath string) (*Exporter, error) {
	if outputPath == "" {
		outputPath = "./output"
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &Exporter{outputPath: outputPath}, nil
}

// ExportNFTScan exports NFT scan results
func (e *Exporter) ExportNFTScan(result *NFTScanResult, format string) (string, error) {
	if result == nil {
		return "", fmt.Errorf("scan result is nil")
	}

	filename := fmt.Sprintf("nft-scan-%s.%s", result.ContractAddress[:8], format)
	filepath := filepath.Join(e.outputPath, filename)

	switch format {
	case "json":
		return e.exportJSON(filepath, result)
	case "csv":
		return e.exportNFTCSV(filepath, result)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportTokenScan exports token scan results
func (e *Exporter) ExportTokenScan(result *TokenScanResult, format string) (string, error) {
	if result == nil {
		return "", fmt.Errorf("scan result is nil")
	}

	filename := fmt.Sprintf("token-scan-%s.%s", result.ContractAddress[:8], format)
	filepath := filepath.Join(e.outputPath, filename)

	switch format {
	case "json":
		return e.exportJSON(filepath, result)
	case "csv":
		return e.exportTokenCSV(filepath, result)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// ExportGenesisData exports genesis-ready data
func (e *Exporter) ExportGenesisData(data *GenesisData) (string, error) {
	if data == nil {
		return "", fmt.Errorf("genesis data is nil")
	}

	filename := fmt.Sprintf("genesis-%s.json", data.NetworkName)
	filepath := filepath.Join(e.outputPath, filename)

	return e.exportJSON(filepath, data)
}

// exportJSON exports data as JSON
func (e *Exporter) exportJSON(filepath string, data interface{}) (string, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return "", fmt.Errorf("failed to encode JSON: %w", err)
	}

	return filepath, nil
}

// exportNFTCSV exports NFT data as CSV
func (e *Exporter) exportNFTCSV(filepath string, result *NFTScanResult) (string, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "TokenID,Owner,URI,StakingPower")

	// Write data
	for _, nft := range result.NFTs {
		fmt.Fprintf(file, "%s,%s,%s,%s\n",
			nft.TokenID,
			nft.Owner,
			nft.URI,
			nft.StakingPower,
		)
	}

	return filepath, nil
}

// exportTokenCSV exports token data as CSV
func (e *Exporter) exportTokenCSV(filepath string, result *TokenScanResult) (string, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "Address,Balance,Percentage")

	// Write data
	for _, holder := range result.Holders {
		fmt.Fprintf(file, "%s,%s,%.4f\n",
			holder.Address,
			holder.Balance,
			holder.Percentage,
		)
	}

	return filepath, nil
}

// GenesisData represents genesis-ready data
type GenesisData struct {
	NetworkName string                 `json:"networkName"`
	ChainID     uint64                 `json:"chainID"`
	Allocations map[string]string      `json:"allocations"`
	NFTs        []GenesisNFT           `json:"nfts,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// GenesisNFT represents an NFT in genesis format
type GenesisNFT struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	StakingPower string `json:"stakingPower,omitempty"`
	Metadata     string `json:"metadata"`
}
