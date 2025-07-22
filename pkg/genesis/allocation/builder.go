package allocation

import (
	"fmt"
	"math/big"
	"time"
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
	
	// Calculate amount per period
	amountPerPeriod := new(big.Int).Div(config.TotalAmount, big.NewInt(int64(config.Periods)))
	periodDuration := config.Duration / time.Duration(config.Periods)

	// Start after cliff
	for i := config.CliffPeriods; i < config.Periods; i++ {
		unlockTime := config.StartDate.Add(periodDuration * time.Duration(i+1))
		
		amount := new(big.Int).Set(amountPerPeriod)
		// Add remainder to last period
		if i == config.Periods-1 {
			remainder := new(big.Int).Mod(config.TotalAmount, big.NewInt(int64(config.Periods)))
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
	OneLUX        = big.NewInt(1_000_000_000)
	ThousandLUX   = big.NewInt(1_000_000_000_000)
	MillionLUX    = big.NewInt(1_000_000_000_000_000)
	BillionLUX    = big.NewInt(1_000_000_000_000_000_000)
)

// ParseLUXAmount parses a LUX amount string (e.g., "1000000") and returns it with proper decimals
func ParseLUXAmount(amountStr string) (*big.Int, error) {
	amount := new(big.Int)
	if _, ok := amount.SetString(amountStr, 10); !ok {
		return nil, fmt.Errorf("invalid amount: %s", amountStr)
	}
	
	// Multiply by 10^9 for 9 decimals
	return amount.Mul(amount, OneLUX), nil
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