package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var (
	accountsCSV      = flag.String("accounts-csv", "", "Path to 7777 accounts CSV")
	minValidatorStake = flag.String("min-validator-stake", "1000000", "Minimum stake for validators (in LUX)")
	outputPath       = flag.String("output", "configs/xchain-genesis.json", "Output genesis file")
	luxPrimaryValidator = flag.String("lux-validator", "NodeID-9CkG9MBNavnw7EVSRsuFr7ws9gascDQy3", "Lux primary validator node ID")
)

type XChainGenesis struct {
	StartTime                  int64                  `json:"startTime"`
	Allocations                []Allocation           `json:"allocations"`
	InitialStakeDuration       int64                  `json:"initialStakeDuration"`
	InitialStakeDurationOffset int64                  `json:"initialStakeDurationOffset"`
	InitialStakers             []Staker               `json:"initialStakers"`
	CChainGenesis              string                 `json:"cChainGenesis"`
	Message                    string                 `json:"message"`
}

type Allocation struct {
	AvaxAddr       string   `json:"avaxAddr"`
	InitialAmount  uint64   `json:"initialAmount"`
	UnlockSchedule []Unlock `json:"unlockSchedule"`
}

type Unlock struct {
	Amount   uint64 `json:"amount"`
	Locktime int64  `json:"locktime"`
}

type Staker struct {
	NodeID        string `json:"nodeID"`
	RewardAddress string `json:"rewardAddress"`
	DelegationFee uint32 `json:"delegationFee"`
}

type Account struct {
	Address           string
	BalanceWei        *big.Int
	BalanceLux        float64
	ValidatorEligible bool
}

func main() {
	flag.Parse()

	if *accountsCSV == "" {
		log.Fatal("--accounts-csv is required")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(*outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Read accounts from CSV
	accounts, err := readAccountsCSV(*accountsCSV)
	if err != nil {
		log.Fatalf("Failed to read accounts CSV: %v", err)
	}

	// Parse minimum validator stake
	minStake, ok := new(big.Int).SetString(*minValidatorStake, 10)
	if !ok {
		log.Fatalf("Invalid min-validator-stake: %s", *minValidatorStake)
	}
	minStakeWei := new(big.Int).Mul(minStake, big.NewInt(1e18))

	// Create genesis structure
	genesis := XChainGenesis{
		StartTime:                  time.Now().Unix(),
		Allocations:                []Allocation{},
		InitialStakeDuration:       365 * 24 * 60 * 60, // 1 year
		InitialStakeDurationOffset: 90 * 24 * 60 * 60,  // 90 days
		InitialStakers:             []Staker{},
		CChainGenesis:              "configs/cchain-genesis.json",
		Message:                    "LUX Network Genesis - 7777 Account Migration",
	}

	// Add Lux as primary validator
	genesis.InitialStakers = append(genesis.InitialStakers, Staker{
		NodeID:        *luxPrimaryValidator,
		RewardAddress: "X-lux1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqv3qzan", // Lux treasury on X-Chain
		DelegationFee: 20000, // 2% fee (in basis points * 10)
	})

	// Process accounts
	validatorCount := 0
	delegatorCount := 0
	totalAllocated := new(big.Int)

	for _, account := range accounts {
		// Convert X-Chain address format (X-lux1...)
		// This is simplified - actual implementation needs proper bech32 encoding
		xchainAddr := fmt.Sprintf("X-lux1%s", account.Address[2:]) // Placeholder

		// Create allocation
		allocation := Allocation{
			AvaxAddr:      xchainAddr,
			InitialAmount: 0, // Will be set based on balance
			UnlockSchedule: []Unlock{},
		}

		// Convert balance to nLUX (nano LUX = 1e-9 LUX)
		balanceNano := new(big.Int).Div(account.BalanceWei, big.NewInt(1e9))
		allocation.InitialAmount = balanceNano.Uint64()

		// For large holders (validators), create unlock schedule
		if account.ValidatorEligible {
			// 10% immediate, 90% vested over 1 year
			immediateAmount := new(big.Int).Div(balanceNano, big.NewInt(10))
			vestedAmount := new(big.Int).Sub(balanceNano, immediateAmount)

			allocation.InitialAmount = immediateAmount.Uint64()
			
			// Create 4 quarterly unlocks
			quarterlyAmount := new(big.Int).Div(vestedAmount, big.NewInt(4))
			for i := 1; i <= 4; i++ {
				unlock := Unlock{
					Amount:   quarterlyAmount.Uint64(),
					Locktime: time.Now().Unix() + int64(i*90*24*60*60), // Every 90 days
				}
				allocation.UnlockSchedule = append(allocation.UnlockSchedule, unlock)
			}

			validatorCount++
		} else {
			// Smaller holders get immediate access
			delegatorCount++
		}

		genesis.Allocations = append(genesis.Allocations, allocation)
		totalAllocated.Add(totalAllocated, account.BalanceWei)
	}

	// Write genesis file
	file, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("Failed to create output file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(genesis); err != nil {
		log.Fatalf("Failed to encode genesis: %v", err)
	}

	// Print summary
	fmt.Printf("\nX-Chain Genesis Summary:\n")
	fmt.Printf("- Total accounts: %d\n", len(accounts))
	fmt.Printf("- Validator eligible: %d\n", validatorCount)
	fmt.Printf("- Delegators: %d\n", delegatorCount)
	totalLux := new(big.Float).Quo(
		new(big.Float).SetInt(totalAllocated),
		new(big.Float).SetInt(big.NewInt(1e18)),
	)
	fmt.Printf("- Total allocated: %.6f LUX\n", totalLux)
	fmt.Printf("- Output file: %s\n", *outputPath)
	fmt.Printf("\nNote: Accounts with ≥%s LUX are validator eligible\n", *minValidatorStake)
	fmt.Printf("      Validators receive 10%% immediate, 90%% vested over 1 year\n")
	fmt.Printf("      Delegators receive 100%% immediate access\n")
}

func readAccountsCSV(path string) ([]Account, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var accounts []Account
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) < 4 {
			continue
		}

		balanceWei, ok := new(big.Int).SetString(record[1], 10)
		if !ok {
			continue
		}

		balanceLux, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}

		validatorEligible := record[3] == "true"

		accounts = append(accounts, Account{
			Address:           record[0],
			BalanceWei:        balanceWei,
			BalanceLux:        balanceLux,
			ValidatorEligible: validatorEligible,
		})
	}

	return accounts, nil
}