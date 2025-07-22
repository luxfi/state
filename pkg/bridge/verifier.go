package bridge

import (
	"fmt"
	"math/big"
)

// Verifier handles verification of imported assets
type Verifier struct {
	config VerifierConfig
}

// NewVerifier creates a new verifier
func NewVerifier(config VerifierConfig) (*Verifier, error) {
	return &Verifier{config: config}, nil
}

// VerifyNFTScan verifies NFT scan results
func (v *Verifier) VerifyNFTScan(result *NFTScanResult) (*VerificationResult, error) {
	if result == nil {
		return nil, fmt.Errorf("scan result is nil")
	}
	
	vr := &VerificationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}
	
	// Check for duplicate token IDs
	seen := make(map[string]bool)
	for _, nft := range result.NFTs {
		if seen[nft.TokenID] {
			vr.Errors = append(vr.Errors, fmt.Sprintf("duplicate token ID: %s", nft.TokenID))
			vr.Valid = false
		}
		seen[nft.TokenID] = true
	}
	
	// Check for valid addresses
	for _, nft := range result.NFTs {
		if !isValidAddress(nft.Owner) {
			vr.Errors = append(vr.Errors, fmt.Sprintf("invalid owner address for token %s: %s", nft.TokenID, nft.Owner))
			vr.Valid = false
		}
	}
	
	// Add summary
	vr.Summary = fmt.Sprintf("Verified %d NFTs from contract %s", len(result.NFTs), result.ContractAddress)
	
	return vr, nil
}

// VerifyTokenScan verifies token scan results
func (v *Verifier) VerifyTokenScan(result *TokenScanResult) (*VerificationResult, error) {
	if result == nil {
		return nil, fmt.Errorf("scan result is nil")
	}
	
	vr := &VerificationResult{
		Valid:    true,
		Warnings: []string{},
		Errors:   []string{},
	}
	
	// Parse minimum balance
	minBal := new(big.Int)
	minBal.SetString(v.config.MinBalance, 10)
	
	// Check total supply
	totalSupply := new(big.Int)
	totalSupply.SetString(result.TotalSupply, 10)
	
	// Verify balances sum to total supply
	sum := new(big.Int)
	validHolders := 0
	
	for _, holder := range result.Holders {
		bal := new(big.Int)
		if _, ok := bal.SetString(holder.Balance, 10); !ok {
			vr.Errors = append(vr.Errors, fmt.Sprintf("invalid balance for %s: %s", holder.Address, holder.Balance))
			vr.Valid = false
			continue
		}
		
		sum.Add(sum, bal)
		
		// Check minimum balance
		if bal.Cmp(minBal) >= 0 {
			validHolders++
		}
		
		// Check valid address
		if !isValidAddress(holder.Address) {
			vr.Errors = append(vr.Errors, fmt.Sprintf("invalid holder address: %s", holder.Address))
			vr.Valid = false
		}
	}
	
	// Check if sum matches total supply
	if sum.Cmp(totalSupply) != 0 {
		vr.Warnings = append(vr.Warnings, fmt.Sprintf("balance sum (%s) doesn't match total supply (%s)", sum.String(), totalSupply.String()))
	}
	
	// Add summary
	vr.Summary = fmt.Sprintf("Verified %d holders (%d above minimum) for %s (%s)", 
		len(result.Holders), validHolders, result.TokenName, result.Symbol)
	
	return vr, nil
}

// CrossReference cross-references with existing chain data
func (v *Verifier) CrossReference(scanResult interface{}, chainData interface{}) (*CrossReferenceResult, error) {
	// TODO: Implement actual cross-referencing logic
	// This is a stub implementation
	
	return &CrossReferenceResult{
		Matched:     900,
		NotFound:    50,
		Additional:  25,
		TotalSource: 975,
		TotalTarget: 925,
	}, nil
}


// isValidAddress checks if an address is valid
func isValidAddress(address string) bool {
	// Basic validation - 42 chars starting with 0x
	if len(address) != 42 || address[:2] != "0x" {
		return false
	}
	
	// Check hex characters
	for i := 2; i < len(address); i++ {
		c := address[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	
	return true
}