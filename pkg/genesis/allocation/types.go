package allocation

import (
	"encoding/json"
	"fmt"
	"math/big"
	"time"
)

// Allocation represents an initial allocation of LUX tokens
type Allocation struct {
	ETHAddr        string          `json:"ethAddr"`
	LuxAddr        string          `json:"luxAddr"`
	InitialAmount  *big.Int        `json:"-"` // Use big.Int internally
	UnlockSchedule []LockedAmount  `json:"unlockSchedule"`
}

// MarshalJSON custom marshaller to handle big.Int
func (a Allocation) MarshalJSON() ([]byte, error) {
	type Alias Allocation
	return json.Marshal(&struct {
		InitialAmount string `json:"initialAmount"`
		*Alias
	}{
		InitialAmount: a.InitialAmount.String(),
		Alias:        (*Alias)(&a),
	})
}

// UnmarshalJSON custom unmarshaller to handle big.Int
func (a *Allocation) UnmarshalJSON(data []byte) error {
	type Alias Allocation
	aux := &struct {
		InitialAmount string `json:"initialAmount"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	a.InitialAmount = new(big.Int)
	if _, ok := a.InitialAmount.SetString(aux.InitialAmount, 10); !ok {
		return fmt.Errorf("invalid initial amount: %s", aux.InitialAmount)
	}
	
	return nil
}

// LockedAmount represents an amount of LUX locked until a specific time
type LockedAmount struct {
	Amount   *big.Int `json:"-"`
	Locktime uint64   `json:"locktime"`
}

// MarshalJSON custom marshaller for LockedAmount
func (l LockedAmount) MarshalJSON() ([]byte, error) {
	type Alias LockedAmount
	return json.Marshal(&struct {
		Amount string `json:"amount"`
		*Alias
	}{
		Amount: l.Amount.String(),
		Alias:  (*Alias)(&l),
	})
}

// UnmarshalJSON custom unmarshaller for LockedAmount
func (l *LockedAmount) UnmarshalJSON(data []byte) error {
	type Alias LockedAmount
	aux := &struct {
		Amount string `json:"amount"`
		*Alias
	}{
		Alias: (*Alias)(l),
	}
	
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	
	l.Amount = new(big.Int)
	if _, ok := l.Amount.SetString(aux.Amount, 10); !ok {
		return fmt.Errorf("invalid locked amount: %s", aux.Amount)
	}
	
	return nil
}

// UnlockScheduleConfig contains parameters for creating unlock schedules
type UnlockScheduleConfig struct {
	TotalAmount  *big.Int
	StartDate    time.Time
	Duration     time.Duration
	Periods      int
	CliffPeriods int // Number of periods before first unlock
}

// AllocationSet manages a collection of allocations
type AllocationSet struct {
	allocations map[string]*Allocation // Key is ETH address
	totalSupply *big.Int
}

// NewAllocationSet creates a new allocation set
func NewAllocationSet() *AllocationSet {
	return &AllocationSet{
		allocations: make(map[string]*Allocation),
		totalSupply: new(big.Int),
	}
}

// Add adds an allocation to the set
func (as *AllocationSet) Add(alloc *Allocation) error {
	if _, exists := as.allocations[alloc.ETHAddr]; exists {
		return fmt.Errorf("allocation already exists for address %s", alloc.ETHAddr)
	}
	
	as.allocations[alloc.ETHAddr] = alloc
	as.totalSupply.Add(as.totalSupply, alloc.InitialAmount)
	
	// Add locked amounts to total supply
	for _, locked := range alloc.UnlockSchedule {
		as.totalSupply.Add(as.totalSupply, locked.Amount)
	}
	
	return nil
}

// Get returns an allocation by ETH address
func (as *AllocationSet) Get(ethAddr string) (*Allocation, bool) {
	alloc, exists := as.allocations[ethAddr]
	return alloc, exists
}

// GetAll returns all allocations as a slice
func (as *AllocationSet) GetAll() []*Allocation {
	result := make([]*Allocation, 0, len(as.allocations))
	for _, alloc := range as.allocations {
		result = append(result, alloc)
	}
	return result
}

// TotalSupply returns the total supply across all allocations
func (as *AllocationSet) TotalSupply() *big.Int {
	return new(big.Int).Set(as.totalSupply)
}

// Count returns the number of allocations
func (as *AllocationSet) Count() int {
	return len(as.allocations)
}