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
	"strconv"
	"strings"

	"github.com/luxfi/genesis/pkg/genesis"
	"github.com/luxfi/genesis/pkg/genesis/allocation"
	"github.com/luxfi/genesis/pkg/genesis/config"
	"github.com/luxfi/genesis/pkg/genesis/validator"
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
	
	// Validator key generation flags
	generateKeysFlag    = flag.Bool("generate-keys", false, "Generate validator keys")
	mnemonicFlag        = flag.String("mnemonic", "", "Mnemonic phrase for deterministic key generation")
	seedPhraseFlag      = flag.String("seed", "", "Alternative to mnemonic")
	privateKeysFlag     = flag.String("private-keys", "", "Comma-separated BLS private keys in hex")
	accountStartFlag    = flag.Int("account-start", 0, "Starting account number")
	accountCountFlag    = flag.Int("account-count", 1, "Number of accounts to generate")
	accountListFlag     = flag.String("accounts", "", "Comma-separated list of account numbers (e.g., 0,5,10)")
	offsetsFlag         = flag.String("offsets", "", "Comma-separated list of account offsets for mnemonic")
	luxdPathFlag        = flag.String("luxd-path", "../node/build/luxd", "Path to luxd binary")
	saveKeysFlag        = flag.String("save-keys", "", "Save generated validator configs to file")
	saveKeysDirFlag     = flag.String("save-keys-dir", "", "Save individual validator keys to separate directories")
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
	
	// Generate validator keys if requested
	if *generateKeysFlag {
		generateValidatorKeys()
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
	importedFromCSV := false
	if *importAllocsFlag != "" {
		fmt.Printf("Importing allocations from %s...\n", *importAllocsFlag)
		// Check if it's a CSV file
		if strings.HasSuffix(*importAllocsFlag, ".csv") {
			if err := builder.ImportCSVAllocations(*importAllocsFlag); err != nil {
				log.Fatalf("Failed to import CSV allocations: %v", err)
			}
			importedFromCSV = true
		} else {
			if err := builder.ImportCChainAllocations(*importAllocsFlag); err != nil {
				log.Fatalf("Failed to import allocations: %v", err)
			}
		}
	}

	// Add treasury allocation only if not importing from CSV (CSV already includes treasury)
	if !importedFromCSV && *treasuryAddrFlag != "" && *treasuryAmountFlag != "" {
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
		
		// For networks with validators, we need to add locked allocations for them
		networkCfg, err := config.GetNetwork(*networkFlag)
		if err != nil {
			log.Fatalf("Failed to get network config: %v", err)
		}
		
		for _, v := range validators {
			// Add staker
			builder.AddStaker(genesis.StakerConfig{
				NodeID:            v.NodeID,
				ETHAddress:        v.ETHAddress,
				PublicKey:         v.PublicKey,
				ProofOfPossession: v.ProofOfPossession,
				Weight:            v.Weight,
				DelegationFee:     v.DelegationFee,
			})
			
			// Add locked allocation for validator (staking amount)
			if v.Weight > 0 {
				stakingAmount := new(big.Int).SetUint64(v.Weight)
				vestingYears := int(networkCfg.InitialStakeDuration.Hours() / 24 / 365)
				if vestingYears < 1 {
					vestingYears = 1
				}
				
				err := builder.AddVestedAllocation(v.ETHAddress, &allocation.UnlockScheduleConfig{
					TotalAmount:  stakingAmount,
					StartDate:    networkCfg.StartTime,
					Duration:     networkCfg.InitialStakeDuration,
					Periods:      1, // Single unlock at end
					CliffPeriods: 0,
				})
				if err != nil {
					log.Fatalf("Failed to add validator allocation: %v", err)
				}
			}
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

// generateValidatorKeys generates validator keys
func generateValidatorKeys() {
	// Create key generator
	keygen := validator.NewKeyGenerator(*luxdPathFlag)
	
	// Parse private keys if provided
	var privateKeys []string
	if *privateKeysFlag != "" {
		parts := strings.Split(*privateKeysFlag, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				// Remove 0x prefix if present
				p = strings.TrimPrefix(p, "0x")
				privateKeys = append(privateKeys, p)
			}
		}
	}
	
	// Determine which accounts/offsets to use
	var accounts []int
	
	// Check for offsets flag (used with mnemonic)
	if *offsetsFlag != "" {
		parts := strings.Split(*offsetsFlag, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil {
				log.Fatalf("Invalid offset: %s", p)
			}
			accounts = append(accounts, n)
		}
	} else if *accountListFlag != "" {
		// Parse comma-separated account numbers
		parts := strings.Split(*accountListFlag, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			n, err := strconv.Atoi(p)
			if err != nil {
				log.Fatalf("Invalid account number: %s", p)
			}
			accounts = append(accounts, n)
		}
	} else if len(privateKeys) > 0 {
		// If using private keys, create sequential indices
		for i := 0; i < len(privateKeys); i++ {
			accounts = append(accounts, i)
		}
	} else {
		// Use sequential accounts
		for i := 0; i < *accountCountFlag; i++ {
			accounts = append(accounts, *accountStartFlag+i)
		}
	}
	
	// Validate we have accounts to generate
	if len(accounts) == 0 {
		log.Fatal("No accounts specified")
	}
	
	// Get mnemonic or seed phrase
	mnemonic := *mnemonicFlag
	if mnemonic == "" && *seedPhraseFlag != "" {
		mnemonic = *seedPhraseFlag
	}
	
	// Validate inputs
	if privateKeys != nil && len(privateKeys) > 0 {
		if len(privateKeys) != len(accounts) {
			log.Fatalf("Number of private keys (%d) must match number of accounts (%d)", len(privateKeys), len(accounts))
		}
		if mnemonic != "" {
			log.Fatal("Cannot use both private keys and mnemonic")
		}
	}
	
	// Create validator configurations
	validators := make([]*validator.ValidatorInfo, 0, len(accounts))
	
	fmt.Printf("Generating %d validator keys...\n", len(accounts))
	
	for idx, accountNum := range accounts {
		var keys *validator.ValidatorKeys
		var keysWithTLS *validator.ValidatorKeysWithTLS
		var err error
		
		if privateKeys != nil && idx < len(privateKeys) {
			// Use provided private key
			keysWithTLS, err = keygen.GenerateFromPrivateKey(privateKeys[idx])
			if err != nil {
				log.Fatalf("Failed to generate keys from private key %d: %v", idx+1, err)
			}
			keys = keysWithTLS.ValidatorKeys
		} else if mnemonic != "" {
			// Deterministic generation from mnemonic
			keysWithTLS, err = keygen.GenerateFromSeedWithTLS(mnemonic, accountNum)
			if err != nil {
				log.Fatalf("Failed to generate keys for account %d: %v", accountNum, err)
			}
			keys = keysWithTLS.ValidatorKeys
		} else {
			// Random generation
			keysWithTLS, err = keygen.GenerateCompatibleKeys()
			if err != nil {
				log.Fatalf("Failed to generate keys for validator %d: %v", idx+1, err)
			}
			keys = keysWithTLS.ValidatorKeys
		}
		
		// Use account number as part of ETH address derivation
		ethAddr := fmt.Sprintf("0x%040x", accountNum)
		
		validatorConfig := validator.GenerateValidatorConfig(
			keys,
			ethAddr,
			1000000000000000000, // 1B LUX default weight
			20000,               // 2% delegation fee
		)
		
		validators = append(validators, validatorConfig)
		
		fmt.Printf("\nValidator %d (Account %d):\n", idx+1, accountNum)
		fmt.Printf("  NodeID: %s\n", keys.NodeID)
		fmt.Printf("  Public Key: %s\n", keys.PublicKey)
		fmt.Printf("  Proof of Possession: %s\n", keys.ProofOfPossession)
		
		// Save individual keys if directory specified
		if *saveKeysDirFlag != "" {
			keyDir := filepath.Join(*saveKeysDirFlag, fmt.Sprintf("validator-%d", idx+1))
			if err := validator.SaveKeys(keys, keyDir); err != nil {
				log.Fatalf("Failed to save keys for validator %d: %v", idx+1, err)
			}
			
			// If we have TLS data, save it too
			if keysWithTLS != nil {
				if err := validator.SaveStakingFiles(keysWithTLS.TLSKeyBytes, keysWithTLS.TLSCertBytes, keyDir); err != nil {
					log.Fatalf("Failed to save staking files for validator %d: %v", idx+1, err)
				}
			}
		}
	}
	
	// Save validator configs to file if requested
	if *saveKeysFlag != "" {
		if err := validator.SaveValidatorConfigs(validators, *saveKeysFlag); err != nil {
			log.Fatalf("Failed to save validator configs: %v", err)
		}
		fmt.Printf("\nValidator configurations saved to: %s\n", *saveKeysFlag)
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

# Generate validator keys from mnemonic with sequential accounts
./genesis-builder -generate-keys \
  -mnemonic "your twelve word mnemonic phrase here" \
  -account-start 0 \
  -account-count 11 \
  -save-keys validators.json \
  -save-keys-dir validator-keys/

# Generate validator keys with specific account numbers
./genesis-builder -generate-keys \
  -mnemonic "your twelve word mnemonic phrase here" \
  -accounts "0,1,2,3,4,5,6,7,8,9,10" \
  -save-keys validators.json

# Generate random validator keys (no mnemonic)
./genesis-builder -generate-keys \
  -account-count 5 \
  -save-keys validators-random.json

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