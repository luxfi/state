package genesis

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

// SubnetConfig represents a subnet configuration
type SubnetConfig struct {
	SubnetID    string `json:"subnetID"`
	ChainID     string `json:"chainID"`
	VMID        string `json:"vmID"`
	GenesisFile string `json:"genesisFile"`
}

// SubnetDefinition defines a subnet's parameters
type SubnetDefinition struct {
	ChainID   string
	Name      string
	ConfigDir string
}

// GetSubnetDefinitions returns subnet definitions for a network
func GetSubnetDefinitions(network string) map[string]SubnetDefinition {
	subnets := map[string]map[string]SubnetDefinition{
		"mainnet": {
			"zoo": {
				ChainID:   "200200",
				Name:      "zoo-mainnet",
				ConfigDir: "chaindata/configs/zoo-mainnet-200200",
			},
			"spc": {
				ChainID:   "36911",
				Name:      "spc-mainnet",
				ConfigDir: "chaindata/configs/spc-mainnet-36911",
			},
		},
		"testnet": {
			"zoo": {
				ChainID:   "200201",
				Name:      "zoo-testnet",
				ConfigDir: "chaindata/configs/zoo-testnet-200201",
			},
		},
	}

	return subnets[network]
}

// BuildSubnetConfigs creates subnet configurations for a network
func BuildSubnetConfigs(network string) (map[string]*SubnetConfig, error) {
	definitions := GetSubnetDefinitions(network)
	configs := make(map[string]*SubnetConfig)

	for subnetName, def := range definitions {
		config := &SubnetConfig{
			SubnetID:    fmt.Sprintf("%s-subnet", def.Name),
			ChainID:     def.ChainID,
			VMID:        "evm",
			GenesisFile: fmt.Sprintf("%s-genesis.json", def.Name),
		}
		configs[subnetName] = config
	}

	return configs, nil
}

// LoadSubnetGenesis loads genesis data for a subnet
func LoadSubnetGenesis(configDir string) ([]byte, error) {
	genesisPath := filepath.Join(configDir, "genesis.json")
	return ioutil.ReadFile(genesisPath)
}

// SaveSubnetConfig saves a subnet configuration to file
func SaveSubnetConfig(config *SubnetConfig, outputPath string) error {
	data, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshal subnet config: %w", err)
	}

	return ioutil.WriteFile(outputPath, data, 0644)
}

// SaveSubnetGenesis saves subnet genesis data to file
func SaveSubnetGenesis(genesisData []byte, outputPath string) error {
	return ioutil.WriteFile(outputPath, genesisData, 0644)
}
