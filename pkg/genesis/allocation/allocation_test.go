package allocation

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLUXAmount(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *big.Int
		hasError bool
	}{
		{
			"raw amount",
			"1000000",
			big.NewInt(1000000000000000), // 1M * 10^9
			false,
		},
		{
			"trillion suffix",
			"2T",
			new(big.Int).Mul(big.NewInt(2000000000000), OneLUX),
			false,
		},
		{
			"billion suffix",
			"1B",
			new(big.Int).Mul(big.NewInt(1000000000), OneLUX),
			false,
		},
		{
			"billion with decimal",
			"1.5B",
			new(big.Int).Mul(big.NewInt(1500000000), OneLUX),
			false,
		},
		{
			"million suffix",
			"500M",
			new(big.Int).Mul(big.NewInt(500000000), OneLUX),
			false,
		},
		{
			"thousand suffix",
			"100K",
			new(big.Int).Mul(big.NewInt(100000), OneLUX),
			false,
		},
		{
			"decimal without suffix",
			"123.456",
			big.NewInt(123456000000), // 123.456 * 10^9
			false,
		},
		{
			"empty string",
			"",
			nil,
			true,
		},
		{
			"invalid format",
			"abc",
			nil,
			true,
		},
		{
			"multiple decimals",
			"1.2.3",
			nil,
			true,
		},
		{
			"unknown suffix",
			"100X",
			nil,
			true,
		},
		{
			"negative amount",
			"-100",
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLUXAmount(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, 0, result.Cmp(tt.expected), 
					"Expected %s but got %s", tt.expected.String(), result.String())
			}
		})
	}
}

func TestFormatLUXAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   *big.Int
		expected string
	}{
		{
			"whole LUX",
			new(big.Int).Mul(big.NewInt(1000), OneLUX),
			"1000 LUX",
		},
		{
			"with decimals",
			big.NewInt(1234567890123),
			"1234.567890123 LUX",
		},
		{
			"zero",
			big.NewInt(0),
			"0 LUX",
		},
		{
			"small amount",
			big.NewInt(123),
			"0.000000123 LUX",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLUXAmount(tt.amount)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateSimpleAllocation(t *testing.T) {
	mockConverter := &mockAddressConverter{
		luxAddr: "X-lux1234567890",
	}
	
	builder := NewBuilder(mockConverter)
	
	ethAddr := "0x1234567890123456789012345678901234567890"
	amount := big.NewInt(1000000000000000000) // 1M LUX
	
	alloc, err := builder.CreateSimpleAllocation(ethAddr, amount)
	require.NoError(t, err)
	
	assert.Equal(t, ethAddr, alloc.ETHAddr)
	assert.Equal(t, "X-lux1234567890", alloc.LuxAddr)
	assert.Equal(t, 0, alloc.InitialAmount.Cmp(amount))
	assert.Empty(t, alloc.UnlockSchedule)
}

func TestCreateVestedAllocation(t *testing.T) {
	mockConverter := &mockAddressConverter{
		luxAddr: "X-lux1234567890",
	}
	
	builder := NewBuilder(mockConverter)
	
	ethAddr := "0x1234567890123456789012345678901234567890"
	// 12M LUX = 12000000 * 10^9
	totalAmount := new(big.Int)
	totalAmount.SetString("12000000000000000", 10)
	
	config := &UnlockScheduleConfig{
		TotalAmount:  totalAmount,
		StartDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Duration:     365 * 24 * time.Hour, // 1 year
		Periods:      12,                   // Monthly
		CliffPeriods: 3,                    // 3 month cliff
	}
	
	alloc, err := builder.CreateVestedAllocation(ethAddr, config)
	require.NoError(t, err)
	
	assert.Equal(t, ethAddr, alloc.ETHAddr)
	assert.Equal(t, "X-lux1234567890", alloc.LuxAddr)
	assert.Equal(t, int64(0), alloc.InitialAmount.Int64()) // All locked
	assert.Len(t, alloc.UnlockSchedule, 9) // 12 - 3 cliff = 9 unlocks
	
	// Verify unlock schedule
	totalUnlocked := big.NewInt(0)
	for _, unlock := range alloc.UnlockSchedule {
		totalUnlocked.Add(totalUnlocked, unlock.Amount)
	}
	// The total should match
	if totalUnlocked.Cmp(totalAmount) != 0 {
		t.Logf("Expected: %s, Got: %s", totalAmount.String(), totalUnlocked.String())
	}
	assert.Equal(t, 0, totalUnlocked.Cmp(totalAmount))
}

func TestCreateLinearVestingSchedule(t *testing.T) {
	mockConverter := &mockAddressConverter{
		luxAddr: "X-lux1234567890",
	}
	
	builder := NewBuilder(mockConverter)
	
	config := &UnlockScheduleConfig{
		TotalAmount:  big.NewInt(1000000000000000000), // 1M LUX
		StartDate:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Duration:     4 * 30 * 24 * time.Hour, // 4 months
		Periods:      4,
		CliffPeriods: 0,
	}
	
	schedule := builder.createUnlockSchedule(config)
	
	assert.Len(t, schedule, 4)
	
	// Verify each unlock is 250K LUX
	expectedPerPeriod := big.NewInt(250000000000000000)
	for i, unlock := range schedule {
		assert.Equal(t, 0, unlock.Amount.Cmp(expectedPerPeriod), 
			"Period %d amount mismatch", i)
		
		// Verify timing
		expectedTime := config.StartDate.Add(time.Duration(i+1) * 30 * 24 * time.Hour)
		assert.Equal(t, uint64(expectedTime.Unix()), unlock.Locktime)
	}
}

func TestCreateStakingAllocation(t *testing.T) {
	mockConverter := &mockAddressConverter{
		luxAddr: "X-lux1234567890",
	}
	
	builder := NewBuilder(mockConverter)
	
	ethAddr := "0x1234567890123456789012345678901234567890"
	stakeAmount := big.NewInt(2000000000000000000) // 2M LUX
	vestingYears := 2
	
	alloc, err := builder.CreateStakingAllocation(ethAddr, stakeAmount, vestingYears)
	require.NoError(t, err)
	
	assert.Equal(t, ethAddr, alloc.ETHAddr)
	assert.Equal(t, int64(0), alloc.InitialAmount.Int64())
	assert.Len(t, alloc.UnlockSchedule, vestingYears)
	
	// Verify total
	total := big.NewInt(0)
	for _, unlock := range alloc.UnlockSchedule {
		total.Add(total, unlock.Amount)
	}
	assert.Equal(t, 0, total.Cmp(stakeAmount))
}

func TestAllocationJSON(t *testing.T) {
	alloc := &Allocation{
		ETHAddr:       "0x1234567890123456789012345678901234567890",
		LuxAddr:       "X-lux1234567890",
		InitialAmount: big.NewInt(1000000000000000000),
		UnlockSchedule: []LockedAmount{
			{
				Amount:   big.NewInt(500000000000000000),
				Locktime: 1704067200, // 2024-01-01
			},
		},
	}
	
	// Test JSON marshaling
	data, err := json.Marshal(alloc)
	require.NoError(t, err)
	
	// Test JSON unmarshaling
	var decoded Allocation
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	
	assert.Equal(t, alloc.ETHAddr, decoded.ETHAddr)
	assert.Equal(t, alloc.LuxAddr, decoded.LuxAddr)
	assert.Equal(t, 0, alloc.InitialAmount.Cmp(decoded.InitialAmount))
	assert.Len(t, decoded.UnlockSchedule, 1)
	assert.Equal(t, 0, alloc.UnlockSchedule[0].Amount.Cmp(decoded.UnlockSchedule[0].Amount))
}

// Mock address converter for testing
type mockAddressConverter struct {
	luxAddr string
	err     error
}

func (m *mockAddressConverter) ETHToLux(ethAddr string, chain string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.luxAddr, nil
}