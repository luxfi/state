package allocation

import (
	"fmt"
	"math/big"
	"strings"
	"time"
	"unicode"
)

// Builder helps construct allocations with various vesting schedules
type Builder struct {
	converter AddressConverter
}

// AddressConverter interface for address conversion
type AddressConverter interface {
	ETHToLux(ethAddr string, chain string) (string, error)
}

// NewBuilder creates a new allocation builder
func NewBuilder(converter AddressConverter) *Builder {
	return &Builder{
		converter: converter,
	}
}

// CreateSimpleAllocation creates a basic allocation with no vesting
func (b *Builder) CreateSimpleAllocation(ethAddr string, amount *big.Int) (*Allocation, error) {
	if amount.Sign() < 0 {
		return nil, fmt.Errorf("negative allocation amount not allowed")
	}

	luxAddr, err := b.converter.ETHToLux(ethAddr, "X")
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	return &Allocation{
		ETHAddr:        ethAddr,
		LuxAddr:        luxAddr,
		InitialAmount:  new(big.Int).Set(amount),
		UnlockSchedule: []LockedAmount{},
	}, nil
}

// CreateVestedAllocation creates an allocation with a vesting schedule
func (b *Builder) CreateVestedAllocation(ethAddr string, config *UnlockScheduleConfig) (*Allocation, error) {
	luxAddr, err := b.converter.ETHToLux(ethAddr, "X")
	if err != nil {
		return nil, fmt.Errorf("failed to convert address: %w", err)
	}

	schedule := b.createUnlockSchedule(config)

	return &Allocation{
		ETHAddr:        ethAddr,
		LuxAddr:        luxAddr,
		InitialAmount:  big.NewInt(0), // All funds are locked initially
		UnlockSchedule: schedule,
	}, nil
}

// CreateLinearVestingSchedule creates a linear unlock schedule
func (b *Builder) createUnlockSchedule(config *UnlockScheduleConfig) []LockedAmount {
	if config.Periods <= 0 {
		return []LockedAmount{}
	}

	schedule := make([]LockedAmount, 0, config.Periods-config.CliffPeriods)

	// Calculate amount per unlock period (not including cliff)
	unlockPeriods := config.Periods - config.CliffPeriods
	amountPerPeriod := new(big.Int).Div(config.TotalAmount, big.NewInt(int64(unlockPeriods)))
	periodDuration := config.Duration / time.Duration(config.Periods)

	// Start after cliff
	for i := config.CliffPeriods; i < config.Periods; i++ {
		unlockTime := config.StartDate.Add(periodDuration * time.Duration(i+1))

		amount := new(big.Int).Set(amountPerPeriod)
		// Add remainder to last period
		if i == config.Periods-1 {
			remainder := new(big.Int).Mod(config.TotalAmount, big.NewInt(int64(unlockPeriods)))
			amount.Add(amount, remainder)
		}

		schedule = append(schedule, LockedAmount{
			Amount:   amount,
			Locktime: uint64(unlockTime.Unix()),
		})
	}

	return schedule
}

// CreateStakingAllocation creates an allocation suitable for staking validators
func (b *Builder) CreateStakingAllocation(ethAddr string, stakeAmount *big.Int, vestingYears int) (*Allocation, error) {
	config := &UnlockScheduleConfig{
		TotalAmount:  stakeAmount,
		StartDate:    time.Now(),
		Duration:     time.Duration(vestingYears) * 365 * 24 * time.Hour,
		Periods:      vestingYears,
		CliffPeriods: 0, // No cliff for staking
	}

	return b.CreateVestedAllocation(ethAddr, config)
}

// Constants for LUX token amounts (with 9 decimals)
var (
	OneLUX      = big.NewInt(1_000_000_000)
	ThousandLUX = big.NewInt(1_000_000_000_000)
	MillionLUX  = big.NewInt(1_000_000_000_000_000)
	BillionLUX  = big.NewInt(1_000_000_000_000_000_000)
)

// ParseLUXAmount parses a LUX amount string with support for suffixes (T, B, M, K)
// Examples: "2T" = 2 trillion, "1.5B" = 1.5 billion, "1000000" = 1 million
func ParseLUXAmount(amountStr string) (*big.Int, error) {
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" {
		return nil, fmt.Errorf("empty amount")
	}

	// Check for suffix
	suffix := ""
	numStr := amountStr

	lastChar := amountStr[len(amountStr)-1]
	if !unicode.IsDigit(rune(lastChar)) {
		suffix = strings.ToUpper(string(lastChar))
		numStr = amountStr[:len(amountStr)-1]
	}

	// Parse the numeric part (supports decimals)
	parts := strings.Split(numStr, ".")

	// Parse integer part
	intPart := new(big.Int)
	if parts[0] != "" {
		if _, ok := intPart.SetString(parts[0], 10); !ok {
			return nil, fmt.Errorf("invalid number: %s", parts[0])
		}
		// Check for negative
		if intPart.Sign() < 0 {
			return nil, fmt.Errorf("negative amounts not allowed")
		}
	}

	// Handle decimal part if present
	decimalPlaces := 0
	if len(parts) > 1 {
		if len(parts) > 2 {
			return nil, fmt.Errorf("multiple decimal points")
		}
		decimalPlaces = len(parts[1])

		// Append decimal digits to integer
		if _, ok := intPart.SetString(parts[0]+parts[1], 10); !ok {
			return nil, fmt.Errorf("invalid decimal number")
		}
	}

	// Apply multiplier based on suffix
	multiplier := new(big.Int).Set(OneLUX) // Base unit with 9 decimals

	switch suffix {
	case "T": // Trillion
		multiplier.Mul(multiplier, big.NewInt(1_000_000_000_000))
	case "B": // Billion
		multiplier.Mul(multiplier, big.NewInt(1_000_000_000))
	case "M": // Million
		multiplier.Mul(multiplier, big.NewInt(1_000_000))
	case "K": // Thousand
		multiplier.Mul(multiplier, big.NewInt(1_000))
	case "":
		// No suffix, treat as LUX units
		multiplier.Set(OneLUX)
	default:
		return nil, fmt.Errorf("unknown suffix: %s", suffix)
	}

	// Adjust for decimal places
	for i := 0; i < decimalPlaces; i++ {
		multiplier.Div(multiplier, big.NewInt(10))
	}

	// Calculate final amount
	return intPart.Mul(intPart, multiplier), nil
}

// FormatLUXAmount formats a big.Int amount to human-readable LUX (dividing by 10^9)
func FormatLUXAmount(amount *big.Int) string {
	lux := new(big.Int).Div(amount, OneLUX)
	remainder := new(big.Int).Mod(amount, OneLUX)

	if remainder.Sign() == 0 {
		return fmt.Sprintf("%s LUX", lux.String())
	}

	// Show up to 9 decimal places
	return fmt.Sprintf("%s.%09d LUX", lux.String(), remainder.Int64())
}
