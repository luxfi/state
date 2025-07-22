package address

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/formatting/address"
)

// Converter handles address conversion between different formats
type Converter struct {
	hrp string // Human-readable part
}

// NewConverter creates a new address converter for a specific network
func NewConverter(hrp string) *Converter {
	return &Converter{hrp: hrp}
}

// ETHToLux converts an Ethereum address to a Lux address
func (c *Converter) ETHToLux(ethAddr string, chain string) (string, error) {
	// Remove 0x prefix if present
	ethAddr = strings.TrimPrefix(strings.ToLower(ethAddr), "0x")

	// Decode hex to bytes
	ethAddrBytes, err := hex.DecodeString(ethAddr)
	if err != nil {
		return "", fmt.Errorf("invalid hex address: %w", err)
	}

	// Convert to ShortID
	shortID, err := ids.ToShortID(ethAddrBytes)
	if err != nil {
		return "", fmt.Errorf("failed to convert to short ID: %w", err)
	}

	// Format as Lux address
	luxAddr, err := address.Format(chain, c.hrp, shortID.Bytes())
	if err != nil {
		return "", fmt.Errorf("failed to format address: %w", err)
	}

	return luxAddr, nil
}

// LuxToETH converts a Lux address to an Ethereum address
func (c *Converter) LuxToETH(luxAddr string) (string, error) {
	// Parse the Lux address
	_, _, addrBytes, err := address.Parse(luxAddr)
	if err != nil {
		return "", fmt.Errorf("failed to parse Lux address: %w", err)
	}

	// Convert to hex with 0x prefix
	return "0x" + hex.EncodeToString(addrBytes), nil
}

// ValidateETHAddress checks if an Ethereum address is valid
func ValidateETHAddress(addr string) error {
	addr = strings.TrimPrefix(strings.ToLower(addr), "0x")
	
	if len(addr) != 40 {
		return fmt.Errorf("invalid address length: expected 40 characters, got %d", len(addr))
	}

	_, err := hex.DecodeString(addr)
	if err != nil {
		return fmt.Errorf("invalid hex encoding: %w", err)
	}

	return nil
}

// ValidateLuxAddress checks if a Lux address is valid
func ValidateLuxAddress(addr string) error {
	_, _, _, err := address.Parse(addr)
	return err
}

// BatchConvert converts multiple Ethereum addresses to Lux addresses
func (c *Converter) BatchConvert(ethAddrs []string, chain string) (map[string]string, error) {
	result := make(map[string]string, len(ethAddrs))
	
	for _, ethAddr := range ethAddrs {
		luxAddr, err := c.ETHToLux(ethAddr, chain)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s: %w", ethAddr, err)
		}
		result[ethAddr] = luxAddr
	}
	
	return result, nil
}