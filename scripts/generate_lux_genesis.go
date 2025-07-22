package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/luxfi/node/ids"
	"github.com/luxfi/node/utils/formatting/address"
)

// Configuration for different networks - Updated to match actual network IDs
var networks = map[string]uint32{
	"mainnet": 96369, // LUX Mainnet
	"testnet": 96368, // LUX Testnet  
	"local":   12345, // LocalID for development
}

// Genesis address that gets initial LUX supply
const genesisETHAddr = "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714" // Treasury address from your docs

// Validator ETH addresses - 11 validators total
var validatorETHAddrs = []string{
	"0x1B475A4C983DfE4f32bbA4dE8DA8fd2c37f3A2A6",
	"0xEAbCC110fAcBfebabC66Ad6f9E7B67288e720B59",
	"0x8d5081153aE1cfb41f5c932fe0b6Beb7E159cF84",
	"0xf8f12D0592e6d1bFe92ee16CaBCC4a6F26dAAe23",
	"0xFb66808f708e1d4D7D43a8c75596e84f94e06806",
	"0x313CF291c069C58D6bd61B0D672673462B8951bD",
	"0xf7f52257a6143cE6BbD12A98eF2B0a3d0C648079",
	"0xCA92ad0C91bd8DE640B9dAFfEB338ac908725142",
	"0xB5B325df519eB58B7223d85aaeac8b56aB05f3d6",
	"0xcf5288bEe8d8F63511C389D5015185FDEDe30e54",
	"0x16204223fe4470f4B1F1dA19A368dC815736a3d7",
}

// Local network development addresses (from POA config)
var localValidatorETHAddrs = []string{
	"0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc", // POA test account
}

// Validator nodes for bootstrap
type ValidatorInfo struct {
	NodeID    string
	PublicKey string
	PoP       string // Proof of Possession
}

// These would need to be generated from actual validator keys
var validators = []ValidatorInfo{
	// Placeholder - these need to be replaced with actual validator info
	{
		NodeID:    "NodeID-111111111111111111111111111111111",
		PublicKey: "0x0000000000000000000000000000000000000000000000000000000000000000",
		PoP:       "0x0000000000000000000000000000000000000000000000000000000000000000",
	},
}

// Convert ETH address to Lux address (X-chain or P-chain)
func ethToLuxAddress(ethAddrHex string, chain string, networkID uint32) (string, error) {
	// Remove 0x prefix if present
	ethAddrHex = strings.TrimPrefix(ethAddrHex, "0x")

	// Decode hex to bytes
	ethAddrBytes, err := hex.DecodeString(ethAddrHex)
	if err != nil {
		return "", err
	}

	// Convert to ShortID
	ethAddr, err := ids.ToShortID(ethAddrBytes)
	if err != nil {
		return "", err
	}

	// Determine HRP based on network ID
	var hrp string
	switch networkID {
	case 96369: // LUX Mainnet
		hrp = "lux"
	case 96368: // LUX Testnet
		hrp = "test"
	case 12345: // LocalID
		hrp = "local"
	default:
		hrp = "custom"
	}

	// Format as Lux address (X-chain or P-chain)
	luxAddr, err := address.Format(chain, hrp, ethAddr.Bytes())
	if err != nil {
		return "", err
	}

	return luxAddr, nil
}

type Allocation struct {
	ETHAddr        string         `json:"ethAddr"`
	LUXAddr        string         `json:"luxAddr"`
	InitialAmount  string         `json:"initialAmount"` // Use string for big numbers
	UnlockSchedule []LockedAmount `json:"unlockSchedule"`
}

type LockedAmount struct {
	Amount   string `json:"amount"` // Use string for big numbers
	Locktime uint64 `json:"locktime"`
}

type Staker struct {
	NodeID        string      `json:"nodeID"`
	RewardAddress string      `json:"rewardAddress"`
	DelegationFee uint32      `json:"delegationFee"`
}

type Genesis struct {
	NetworkID                  uint32       `json:"networkID"`
	Allocations                []Allocation `json:"allocations"`
	StartTime                  uint64       `json:"startTime"`
	InitialStakeDuration       uint64       `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64       `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []string     `json:"initialStakedFunds"`
	InitialStakers             []Staker     `json:"initialStakers"`
	CChainGenesis              string       `json:"cChainGenesis"`
	Message                    string       `json:"message"`
}

func generateGenesis(networkName string) error {
	networkID, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("unknown network: %s", networkName)
	}

	// Convert genesis ETH address to Lux X-chain address
	genesisLuxAddr, err := ethToLuxAddress(genesisETHAddr, "X", networkID)
	if err != nil {
		return fmt.Errorf("failed to convert genesis address: %v", err)
	}

	fmt.Printf("\nNetwork: %s (ID: %d)\n", networkName, networkID)
	fmt.Printf("Genesis ETH address: %s\n", genesisETHAddr)
	fmt.Printf("Genesis LUX X-chain address: %s\n", genesisLuxAddr)

	// Create C-Chain genesis with proper chain ID
	var chainID uint64
	switch networkID {
	case 96369: // LUX Mainnet
		chainID = 96369
	case 96368: // LUX Testnet
		chainID = 96368
	case 12345: // LocalID
		chainID = 12345
	default:
		chainID = uint64(networkID)
	}

	// C-Chain genesis configuration
	cChainGenesisObj := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId":             chainID,
			"homesteadBlock":      0,
			"eip150Block":         0,
			"eip150Hash":          "0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0",
			"eip155Block":         0,
			"eip158Block":         0,
			"byzantiumBlock":      0,
			"constantinopleBlock": 0,
			"petersburgBlock":     0,
			"istanbulBlock":       0,
			"muirGlacierBlock":    0,
			"berlinBlock":         0,
			"londonBlock":         0,
			"subnetEVMTimestamp":  0,
			"feeConfig": map[string]interface{}{
				"gasLimit":                 15000000,
				"minBaseFee":               25000000000,
				"targetGas":                15000000,
				"baseFeeChangeDenominator": 36,
				"minBlockGasCost":          0,
				"maxBlockGasCost":          1000000,
				"targetBlockRate":          2,
				"blockGasCostStep":         200000,
			},
			"allowFeeRecipients": false,
		},
		"alloc": map[string]interface{}{
			// For POA development, give initial balance to test account
			"0x8db97c7cece249c2b98bdc0226cc4c2a57bf52fc": map[string]interface{}{
				"balance": "0xd3c21bcecceda1000000", // 1,000,000 ETH in wei
			},
		},
		"nonce":      "0x0",
		"timestamp":  "0x0",
		"extraData":  "0x00",
		"gasLimit":   "0xE4E1C0", // 15,000,000
		"difficulty": "0x0",
		"mixHash":    "0x0000000000000000000000000000000000000000000000000000000000000000",
		"coinbase":   "0x0000000000000000000000000000000000000000",
		"number":     "0x0",
		"gasUsed":    "0x0",
		"parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
	}

	// Add allocations from your existing data if needed
	if networkName == "mainnet" {
		// TODO: Import allocations from your existing genesis data
		// This would read from genesis-lux/allocations_combined.json
		fmt.Println("Note: You should import existing allocations from genesis-lux/allocations_combined.json")
	}

	cChainGenesisBytes, err := json.Marshal(cChainGenesisObj)
	if err != nil {
		return fmt.Errorf("failed to marshal C-chain genesis: %v", err)
	}
	cChainGenesis := string(cChainGenesisBytes)

	// Choose validator addresses based on network
	validatorAddrs := validatorETHAddrs
	if networkName == "local" {
		validatorAddrs = localValidatorETHAddrs
	}

	// Create allocations using big numbers to handle 2T LUX properly
	// 2T LUX with 9 decimals = 2,000,000,000,000,000,000,000
	totalSupply := new(big.Int)
	totalSupply.SetString("2000000000000000000000", 10)

	// Reserve 1B LUX per validator for staking
	validatorStake := new(big.Int)
	validatorStake.SetString("1000000000000000000", 10) // 1B with 9 decimals

	stakingReserve := new(big.Int).Mul(validatorStake, big.NewInt(int64(len(validatorAddrs))))
	genesisAmount := new(big.Int).Sub(totalSupply, stakingReserve)

	// Create allocations
	allocations := []Allocation{
		{
			ETHAddr:        genesisETHAddr,
			LUXAddr:        genesisLuxAddr,
			InitialAmount:  genesisAmount.String(),
			UnlockSchedule: []LockedAmount{},
		},
	}

	// Add validator allocations
	stakedFunds := []string{}
	for i, ethAddr := range validatorAddrs {
		luxAddrX, err := ethToLuxAddress(ethAddr, "X", networkID)
		if err != nil {
			return fmt.Errorf("failed to convert validator address %s: %v", ethAddr, err)
		}

		// Simple allocation for validators
		allocations = append(allocations, Allocation{
			ETHAddr:        ethAddr,
			LUXAddr:        luxAddrX,
			InitialAmount:  validatorStake.String(),
			UnlockSchedule: []LockedAmount{},
		})

		stakedFunds = append(stakedFunds, luxAddrX)

		fmt.Printf("Validator %d: %s -> %s\n", i+1, ethAddr, luxAddrX)
	}

	// Create genesis structure
	genesis := Genesis{
		NetworkID:                  networkID,
		Allocations:                allocations,
		StartTime:                  uint64(time.Now().Unix()), // Current time for immediate start
		InitialStakeDuration:       31536000,                  // 365 days
		InitialStakeDurationOffset: 5400,                      // 90 minutes
		InitialStakedFunds:         stakedFunds,
		InitialStakers:             []Staker{}, // Empty for now - validators would be added here
		CChainGenesis:              cChainGenesis,
		Message:                    "Lux Network Genesis - Generated " + time.Now().Format("2006-01-02"),
	}

	// Marshal to JSON
	output, err := json.MarshalIndent(genesis, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal genesis: %v", err)
	}

	// Write to file
	filename := fmt.Sprintf("genesis_%s.json", networkName)
	if err := ioutil.WriteFile(filename, output, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Printf("\nGenerated %s\n", filename)
	fmt.Printf("Total supply: %s LUX\n", totalSupply.String())
	fmt.Printf("Genesis allocation: %s LUX\n", genesisAmount.String())
	fmt.Printf("Validator allocations: %s LUX total\n", stakingReserve.String())

	return nil
}

func main() {
	// Generate only for networks we want
	networksToGenerate := []string{"mainnet", "testnet", "local"}
	
	for _, network := range networksToGenerate {
		if err := generateGenesis(network); err != nil {
			log.Printf("Error generating genesis for %s: %v", network, err)
		}
	}

	fmt.Println("\nIMPORTANT NOTES:")
	fmt.Println("1. This generates a basic genesis structure")
	fmt.Println("2. For mainnet, you need to import existing allocations from genesis-lux/allocations_combined.json")
	fmt.Println("3. Validator node IDs, public keys, and PoP values need to be replaced with actual values")
	fmt.Println("4. The C-Chain genesis 'alloc' section should include migrated balances from your subnets")
}