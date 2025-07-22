package bridge

import (
	"fmt"
	"math/big"
	"strings"
)

const (
	// Each EGG NFT represents 4,200,000 ZOO tokens
	ZooPerEgg = 4200000
)

// EggNFTHolder represents an EGG NFT holder with their holdings
type EggNFTHolder struct {
	Address      string `json:"address"`
	EggCount     int    `json:"eggCount"`
	ZooAmount    int64  `json:"zooAmount"`
	IsSpecial    bool   `json:"isSpecial,omitempty"`
	SpecialLabel string `json:"specialLabel,omitempty"`
}

// EggNFTSummary contains the complete EGG NFT analysis
type EggNFTSummary struct {
	TotalEggs       int             `json:"totalEggs"`
	TotalZooValue   int64           `json:"totalZooValue"`
	UniqueHolders   int             `json:"uniqueHolders"`
	Holders         []EggNFTHolder  `json:"holders"`
	SpecialHolders  map[string]string `json:"specialHolders"`
}

// AnalyzeEggNFTs analyzes EGG NFT holdings and calculates ZOO equivalents
func AnalyzeEggNFTs(nftResult *NFTScanResult) (*EggNFTSummary, error) {
	if nftResult == nil {
		return nil, fmt.Errorf("NFT scan result is nil")
	}

	// Build holder map
	holderMap := make(map[string]int)
	for _, nft := range nftResult.NFTs {
		addr := strings.ToLower(nft.Owner)
		holderMap[addr]++
	}

	// Special addresses
	specialAddresses := map[string]string{
		"0xffdb31285961d44d40c404566e9de9080b1abd50": "otc",
		"0xc06c7c6ec618de992d597d8e347669ea44ede2bc": "jules",
		"0x6762ff916de1b315da56f4fa7b78f39aa60d9f4c": "sean",
		"0x28dad8427f127664365109c4a9406c8bc7844718": "treasury/marketplace",
		"0x95a7b934860942e903c47d85041e263ea9167de8": "zach",
	}

	// Build holder list
	holders := []EggNFTHolder{}
	totalEggs := 0
	totalZoo := int64(0)

	for addr, count := range holderMap {
		zooAmount := int64(count * ZooPerEgg)
		holder := EggNFTHolder{
			Address:   addr,
			EggCount:  count,
			ZooAmount: zooAmount,
		}

		// Check if special address
		if label, isSpecial := specialAddresses[strings.ToLower(addr)]; isSpecial {
			holder.IsSpecial = true
			holder.SpecialLabel = label
		}

		holders = append(holders, holder)
		totalEggs += count
		totalZoo += zooAmount
	}

	summary := &EggNFTSummary{
		TotalEggs:      totalEggs,
		TotalZooValue:  totalZoo,
		UniqueHolders:  len(holders),
		Holders:        holders,
		SpecialHolders: specialAddresses,
	}

	return summary, nil
}

// CalculateZooAllocation calculates the Zoo token allocation for an EGG holder
func CalculateZooAllocation(eggCount int) *big.Int {
	// Each EGG = 4,200,000 ZOO tokens
	// Convert to wei (assuming 18 decimals)
	zooPerEgg := new(big.Int).SetInt64(ZooPerEgg)
	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	
	allocation := new(big.Int).Mul(zooPerEgg, decimals)
	allocation.Mul(allocation, big.NewInt(int64(eggCount)))
	
	return allocation
}

// GetEggDistribution analyzes the distribution of EGG holdings
func GetEggDistribution(holders []EggNFTHolder) map[string]int {
	distribution := map[string]int{
		"1 EGG":      0,
		"2-5 EGGs":   0,
		"6-10 EGGs":  0,
		"11-20 EGGs": 0,
		">20 EGGs":   0,
		"100+ EGGs":  0,
	}

	for _, holder := range holders {
		switch {
		case holder.EggCount == 1:
			distribution["1 EGG"]++
		case holder.EggCount >= 2 && holder.EggCount <= 5:
			distribution["2-5 EGGs"]++
		case holder.EggCount >= 6 && holder.EggCount <= 10:
			distribution["6-10 EGGs"]++
		case holder.EggCount >= 11 && holder.EggCount <= 20:
			distribution["11-20 EGGs"]++
		case holder.EggCount >= 100:
			distribution["100+ EGGs"]++
		case holder.EggCount > 20:
			distribution[">20 EGGs"]++
		}
	}

	return distribution
}

// ValidateEggHoldings validates EGG holdings against known data
func ValidateEggHoldings(actualHolders map[string]int, expectedHolders map[string]int) (matches int, mismatches []string) {
	matches = 0
	mismatches = []string{}

	for addr, expected := range expectedHolders {
		actual := actualHolders[strings.ToLower(addr)]
		if actual == expected {
			matches++
		} else {
			mismatches = append(mismatches, fmt.Sprintf("%s: expected %d, got %d", addr, expected, actual))
		}
	}

	// Check for unexpected holders
	for addr, count := range actualHolders {
		if _, exists := expectedHolders[addr]; !exists && count > 0 {
			mismatches = append(mismatches, fmt.Sprintf("%s: unexpected holder with %d EGGs", addr, count))
		}
	}

	return matches, mismatches
}

// FormatEggHolderReport formats a report of EGG holders
func FormatEggHolderReport(summary *EggNFTSummary) string {
	var report strings.Builder

	report.WriteString("EGG NFT Holdings Report\n")
	report.WriteString("======================\n\n")
	
	report.WriteString(fmt.Sprintf("Total EGGs: %d\n", summary.TotalEggs))
	report.WriteString(fmt.Sprintf("Total ZOO Value: %d\n", summary.TotalZooValue))
	report.WriteString(fmt.Sprintf("Unique Holders: %d\n\n", summary.UniqueHolders))

	report.WriteString("Top Holders:\n")
	report.WriteString("-----------\n")

	// Find top 20 holders
	topCount := 20
	if len(summary.Holders) < topCount {
		topCount = len(summary.Holders)
	}

	for i := 0; i < topCount; i++ {
		holder := summary.Holders[i]
		label := ""
		if holder.IsSpecial {
			label = fmt.Sprintf(" [%s]", holder.SpecialLabel)
		}
		report.WriteString(fmt.Sprintf("%2d. %s: %d EGGs (%d ZOO)%s\n", 
			i+1, holder.Address, holder.EggCount, holder.ZooAmount, label))
	}

	return report.String()
}