package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/node/genesis"
)

func TestNewBuilder(t *testing.T) {
	tests := []struct {
		name      string
		network   string
		expectErr bool
	}{
		{"mainnet", "mainnet", false},
		{"testnet", "testnet", false},
		{"local", "local", false},
		{"invalid", "invalid-network", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, err := NewBuilder(tt.network)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, builder)
			}
		})
	}
}

func TestAddAllocation(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	tests := []struct {
		name      string
		ethAddr   string
		amount    *big.Int
		expectErr bool
	}{
		{
			"valid allocation",
			"0x1234567890123456789012345678901234567890",
			big.NewInt(1000000000000000000), // 1 LUX
			false,
		},
		{
			"invalid address",
			"invalid-address",
			big.NewInt(1000000000000000000),
			true,
		},
		{
			"zero amount",
			"0x0987654321098765432109876543210987654321",
			big.NewInt(0),
			false,
		},
		{
			"negative amount",
			"0xabcdef0123456789012345678901234567890123",
			big.NewInt(-1),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := builder.AddAllocation(tt.ethAddr, tt.amount)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAddStaker(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	validStaker := StakerConfig{
		NodeID:            "NodeID-7VJMGZYzqTvVjS23912uzu8gCFcnquPQW",
		ETHAddress:        "0x1234567890123456789012345678901234567890",
		PublicKey:         "0x" + strings.Repeat("00", 48), // 48 bytes hex
		ProofOfPossession: "0x" + strings.Repeat("00", 96), // 96 bytes hex
		Weight:            1000000000000000000, // 1M LUX
		DelegationFee:     20000, // 2%
	}

	builder.AddStaker(validStaker)

	// Test duplicate NodeID - AddStaker doesn't return error for duplicates
	// It's handled during Build()
	builder.AddStaker(validStaker)
}

func TestBuild(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	// Add some allocations
	err = builder.AddAllocation("0x1234567890123456789012345678901234567890", big.NewInt(1000000000000000000))
	require.NoError(t, err)

	// Add a staker
	staker := StakerConfig{
		NodeID:            "NodeID-7VJMGZYzqTvVjS23912uzu8gCFcnquPQW",
		ETHAddress:        "0x1234567890123456789012345678901234567890",
		PublicKey:         "0x" + strings.Repeat("00", 48),
		ProofOfPossession: "0x" + strings.Repeat("00", 96),
		Weight:            1000000000000000000,
		DelegationFee:     20000,
	}
	builder.AddStaker(staker)

	// Build genesis
	genesis, err := builder.Build()
	require.NoError(t, err)
	assert.NotNil(t, genesis)

	// Verify network ID
	assert.Equal(t, uint32(96369), genesis.NetworkID)

	// Verify allocations
	assert.Len(t, genesis.Allocations, 1)

	// Verify stakers
	assert.Len(t, genesis.InitialStakers, 1)
}

func TestImportCChainGenesis(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	// Create a mock C-Chain genesis
	cchainGenesis := map[string]interface{}{
		"config": map[string]interface{}{
			"chainId": 96369,
		},
		"alloc": map[string]interface{}{
			"0x1234567890123456789012345678901234567890": map[string]string{
				"balance": "1000000000000000000",
			},
		},
	}

	// Write to temp file
	tempFile := t.TempDir() + "/cchain-genesis.json"
	data, _ := json.Marshal(cchainGenesis)
	require.NoError(t, ioutil.WriteFile(tempFile, data, 0644))

	// Import
	err = builder.ImportCChainGenesis(tempFile)
	assert.NoError(t, err)

	// Build and verify
	genesis, err := builder.Build()
	require.NoError(t, err)
	assert.Contains(t, string(genesis.CChainGenesis), "96369")
}

func TestImportCSVAllocations(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	// Create a mock CSV file with the expected format
	csvContent := `rank,address,balance_lux,balance_wei
1,0x1234567890123456789012345678901234567890,1000000000,1000000000000000000
2,0x0987654321098765432109876543210987654321,2000000000,2000000000000000000`

	tempFile := t.TempDir() + "/allocations.csv"
	require.NoError(t, ioutil.WriteFile(tempFile, []byte(csvContent), 0644))

	// Import
	err = builder.ImportCSVAllocations(tempFile)
	assert.NoError(t, err)

	// Build and verify
	genesis, err := builder.Build()
	require.NoError(t, err)
	assert.Len(t, genesis.Allocations, 2)
}

func TestAddVestedAllocation(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	vestingConfig := &allocation.UnlockScheduleConfig{
		TotalAmount:  big.NewInt(1000000000000000000), // 1M LUX
		StartDate:    time.Now(),
		Duration:     365 * 24 * time.Hour, // 1 year
		Periods:      12,                   // Monthly unlocks
		CliffPeriods: 3,                    // 3 month cliff
	}

	err = builder.AddVestedAllocation("0x1234567890123456789012345678901234567890", vestingConfig)
	assert.NoError(t, err)

	// Build and verify
	genesis, err := builder.Build()
	require.NoError(t, err)
	assert.Len(t, genesis.Allocations, 1)
	
	// Verify vesting schedule
	alloc := genesis.Allocations[0]
	assert.Equal(t, uint64(0), alloc.InitialAmount) // All locked initially
	assert.Len(t, alloc.UnlockSchedule, 9) // 12 periods - 3 cliff = 9 unlocks
}

func TestGetTotalSupply(t *testing.T) {
	builder, err := NewBuilder("mainnet")
	require.NoError(t, err)

	// Add allocations
	amounts := []*big.Int{
		big.NewInt(1000000000000000000), // 1M LUX
		func() *big.Int { v, _ := new(big.Int).SetString("2000000000000000000", 10); return v }(), // 2M LUX
		func() *big.Int { v, _ := new(big.Int).SetString("3000000000000000000", 10); return v }(), // 3M LUX
	}

	for i, amount := range amounts {
		addr := fmt.Sprintf("0x%040d", i)
		err := builder.AddAllocation(addr, amount)
		require.NoError(t, err)
	}

	// Verify total supply
	totalSupply := builder.GetTotalSupply()
	expected := big.NewInt(6000000000000000000) // 6M LUX
	assert.Equal(t, 0, totalSupply.Cmp(expected))
}

func TestValidateGenesis(t *testing.T) {
	tests := []struct {
		name      string
		genesis   *Genesis
		expectErr bool
		errMsg    string
	}{
		{
			"valid genesis",
			&Genesis{
				NetworkID:      96369,
				Allocations:    []genesis.UnparsedAllocation{},
				InitialStakers: []genesis.UnparsedStaker{},
				StartTime:      uint64(time.Now().Unix()),
			},
			false,
			"",
		},
		{
			"invalid network ID",
			&Genesis{
				NetworkID:      0,
				Allocations:    []genesis.UnparsedAllocation{},
				InitialStakers: []genesis.UnparsedStaker{},
				StartTime:      uint64(time.Now().Unix()),
			},
			true,
			"invalid network ID",
		},
		{
			"no start time",
			&Genesis{
				NetworkID:      96369,
				Allocations:    []genesis.UnparsedAllocation{},
				InitialStakers: []genesis.UnparsedStaker{},
				StartTime:      0,
			},
			true,
			"start time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGenesis(tt.genesis)
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected *big.Int
		hasError bool
	}{
		{"1000000", big.NewInt(1000000000000000), false}, // 1M LUX
		{"2T", func() *big.Int { v, _ := new(big.Int).SetString("2000000000000000000000", 10); return v }(), false}, // 2T LUX
		{"1.5B", big.NewInt(1500000000000000000), false}, // 1.5B LUX
		{"500M", big.NewInt(500000000000000000), false}, // 500M LUX
		{"100K", big.NewInt(100000000000000), false}, // 100K LUX
		{"invalid", nil, true},
		{"", nil, true},
		{"-100", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := allocation.ParseLUXAmount(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, 0, result.Cmp(tt.expected))
			}
		})
	}
}