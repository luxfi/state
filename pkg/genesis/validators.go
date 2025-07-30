package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// ValidatorInfo represents a validator's configuration
type ValidatorInfo struct {
	NodeID            string `json:"nodeID"`
	ETHAddress        string `json:"ethAddress"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
	Weight            uint64 `json:"weight"`
	DelegationFee     uint32 `json:"delegationFee"`
}

// LoadValidators loads validator configurations from a JSON file
func LoadValidators(path string) ([]ValidatorInfo, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read validators file: %w", err)
	}

	var validators []ValidatorInfo
	if err := json.Unmarshal(data, &validators); err != nil {
		return nil, fmt.Errorf("failed to parse validators: %w", err)
	}

	return validators, nil
}

// ValidateValidators checks that all validators have required fields
func ValidateValidators(validators []ValidatorInfo) error {
	for i, v := range validators {
		if v.NodeID == "" {
			return fmt.Errorf("validator %d missing nodeID", i)
		}
		if v.ETHAddress == "" {
			return fmt.Errorf("validator %d missing ethAddress", i)
		}
		if v.PublicKey == "" {
			return fmt.Errorf("validator %d missing publicKey", i)
		}
		if v.ProofOfPossession == "" {
			return fmt.Errorf("validator %d missing proofOfPossession", i)
		}
		if v.Weight == 0 {
			return fmt.Errorf("validator %d has zero weight", i)
		}
	}
	return nil
}
