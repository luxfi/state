package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"path/filepath"

	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/genesis/pkg/genesis/config"
)

// Command-line flags
var (
	networkFlag         = flag.String("network", "mainnet", "Network to generate genesis for (mainnet, testnet, local, zoo-mainnet, etc)")
	outputFlag          = flag.String("output", "", "Output file path (default: genesis_<network>.json)")
	importCChainFlag    = flag.String("import-cchain", "", "Path to existing C-Chain genesis to import")
	importAllocsFlag    = flag.String("import-allocations", "", "Path to C-Chain allocations JSON to import")
	treasuryAddrFlag    = flag.String("treasury", "0x9011E888251AB053B7bD1cdB598Db4f9DEd94714", "Treasury address")
	treasuryAmountFlag  = flag.String("treasury-amount", "2000000000000000000000", "Treasury amount in LUX (with 9 decimals)")
	validatorsFileFlag  = flag.String("validators", "", "Path to JSON file with validator configurations")
	dryRunFlag          = flag.Bool("dry-run", false, "Print genesis without saving to file")
	listNetworksFlag    = flag.Bool("list-networks", false, "List available networks")
)

// ValidatorConfig represents a validator in the config file
type ValidatorConfig struct {
	NodeID            string `json:"nodeID"`
	ETHAddress        string `json:"ethAddress"`
	PublicKey         string `json:"publicKey"`
	ProofOfPossession string `json:"proofOfPossession"`
	Weight            uint64 `json:"weight"`
	DelegationFee     uint32 `json:"delegationFee"`
}

func main() {
	flag.Parse()

	// List networks if requested
	if *listNetworksFlag {
		listNetworks()
		return
	}

	// Create genesis builder
	builder, err := genesis.NewBuilder(*networkFlag)
	if err != nil {
		log.Fatalf("Failed to create builder: %v", err)
	}

	// Import existing C-Chain genesis if specified
	if *importCChainFlag != "" {
		fmt.Printf("Importing C-Chain genesis from %s...\n", *importCChainFlag)
		if err := builder.ImportCChainGenesis(*importCChainFlag); err != nil {
			log.Fatalf("Failed to import C-Chain genesis: %v", err)
		}
	}

	// Import C-Chain allocations if specified
	if *importAllocsFlag != "" {
		fmt.Printf("Importing C-Chain allocations from %s...\n", *importAllocsFlag)
		if err := builder.ImportCChainAllocations(*importAllocsFlag); err != nil {
			log.Fatalf("Failed to import allocations: %v", err)
		}
	}

	// Add treasury allocation
	if *treasuryAddrFlag != "" && *treasuryAmountFlag != "" {
		treasuryAmount := new(big.Int)
		if _, ok := treasuryAmount.SetString(*treasuryAmountFlag, 10); !ok {
			log.Fatalf("Invalid treasury amount: %s", *treasuryAmountFlag)
		}

		fmt.Printf("Adding treasury allocation: %s -> %s LUX\n", 
			*treasuryAddrFlag, 
			allocation.FormatLUXAmount(treasuryAmount))

		if err := builder.AddAllocation(*treasuryAddrFlag, treasuryAmount); err != nil {
			log.Fatalf("Failed to add treasury allocation: %v", err)
		}
	}

	// Load validators from file if specified
	if *validatorsFileFlag != "" {
		validators, err := loadValidators(*validatorsFileFlag)
		if err != nil {
			log.Fatalf("Failed to load validators: %v", err)
		}

		fmt.Printf("Adding %d validators from %s...\n", len(validators), *validatorsFileFlag)
		for _, v := range validators {
			builder.AddStaker(genesis.StakerConfig{
				NodeID:            v.NodeID,
				ETHAddress:        v.ETHAddress,
				PublicKey:         v.PublicKey,
				ProofOfPossession: v.ProofOfPossession,
				Weight:            v.Weight,
				DelegationFee:     v.DelegationFee,
			})
		}
	}

	// Build genesis
	fmt.Println("Building genesis configuration...")
	g, err := builder.Build()
	if err != nil {
		log.Fatalf("Failed to build genesis: %v", err)
	}

	// Print summary
	fmt.Printf("\nGenesis Summary:\n")
	fmt.Printf("Network ID: %d\n", g.NetworkID)
	fmt.Printf("Allocations: %d\n", len(g.Allocations))
	fmt.Printf("Initial Stakers: %d\n", len(g.InitialStakers))
	fmt.Printf("Start Time: %d\n", g.StartTime)
	fmt.Printf("Total Supply: %s\n", allocation.FormatLUXAmount(builder.GetTotalSupply()))

	// Handle output
	if *dryRunFlag {
		// Print to stdout
		data, err := json.MarshalIndent(g, "", "\t")
		if err != nil {
			log.Fatalf("Failed to marshal genesis: %v", err)
		}
		fmt.Println("\nGenesis JSON:")
		fmt.Println(string(data))
	} else {
		// Save to file
		outputPath := *outputFlag
		if outputPath == "" {
			outputPath = fmt.Sprintf("genesis_%s.json", *networkFlag)
		}

		if err := builder.SaveToFile(g, outputPath); err != nil {
			log.Fatalf("Failed to save genesis: %v", err)
		}

		fmt.Printf("\nGenesis saved to: %s\n", outputPath)
	}
}

// loadValidators loads validator configurations from a JSON file
func loadValidators(path string) ([]ValidatorConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var validators []ValidatorConfig
	if err := json.Unmarshal(data, &validators); err != nil {
		return nil, err
	}

	return validators, nil
}

// listNetworks prints all available networks
func listNetworks() {
	fmt.Println("Available networks:")
	fmt.Println("\nL1 Networks (Primary):")
	for name, net := range config.Networks {
		if !net.IsL2 {
			fmt.Printf("  %-15s - %s (Chain ID: %d, Network ID: %d)\n", 
				name, net.Name, net.ChainID, net.ID)
		}
	}

	fmt.Println("\nL2 Networks (Subnets):")
	for name, net := range config.Networks {
		if net.IsL2 {
			fmt.Printf("  %-15s - %s (Chain ID: %d, Network ID: %d, Parent: %s)\n", 
				name, net.Name, net.ChainID, net.ID, net.ParentNetwork)
		}
	}
}

// Example usage function for documentation
func printUsageExamples() {
	examples := `
Examples:

# Generate mainnet genesis with default treasury
./genesis-builder -network mainnet

# Generate testnet genesis with custom output
./genesis-builder -network testnet -output testnet-genesis.json

# Import existing C-Chain genesis (for carrying forward chain data)
./genesis-builder -network mainnet \
  -import-cchain /path/to/existing/cchain-genesis.json

# Import C-Chain allocations from existing network
./genesis-builder -network mainnet \
  -import-allocations /path/to/allocations.json

# Generate L2 subnet genesis
./genesis-builder -network zoo-mainnet \
  -treasury 0x... \
  -treasury-amount 1000000000000000000

# Generate with validators from file
./genesis-builder -network mainnet \
  -validators validators.json

# Dry run to see output without saving
./genesis-builder -network mainnet -dry-run

# List all available networks
./genesis-builder -list-networks

Validator file format:
[
  {
    "nodeID": "NodeID-...",
    "ethAddress": "0x...",
    "publicKey": "0x...",
    "proofOfPossession": "0x...",
    "weight": 1000000000000000000,
    "delegationFee": 20000
  }
]
`
	fmt.Println(examples)
}

func init() {
	// Custom usage function
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Genesis builder for Lux Network and L2 subnets\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		printUsageExamples()
	}
}