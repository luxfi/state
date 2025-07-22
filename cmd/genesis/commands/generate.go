package commands

import (
	"fmt"
	"path/filepath"
	
	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

// NewGenerateCommand creates the generate command
func NewGenerateCommand() *cobra.Command {
	var (
		network       string
		outputDir     string
		validatorsFile string
		withXChain    bool
		withSubnets   bool
	)

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate complete genesis files for Lux Network",
		Long: `Generate genesis files for Lux Network including:
- Main genesis with P-Chain validators and C-Chain state
- X-Chain genesis with airdrops and NFTs
- L2 subnet configurations for Zoo and other chains`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(network, outputDir, validatorsFile, withXChain, withSubnets)
		},
	}

	cmd.Flags().StringVarP(&network, "network", "n", "mainnet", "Network to generate (mainnet or testnet)")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "output", "Output directory for genesis files")
	cmd.Flags().StringVarP(&validatorsFile, "validators", "v", "", "Validators configuration file")
	cmd.Flags().BoolVar(&withXChain, "with-xchain", true, "Generate X-Chain genesis")
	cmd.Flags().BoolVar(&withSubnets, "with-subnets", true, "Generate L2 subnet configurations")

	return cmd
}

func runGenerate(network, outputDir, validatorsFile string, withXChain, withSubnets bool) error {
	// Create output directory
	if err := genesis.CreateDirectory(outputDir); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Get default paths if not specified
	if validatorsFile == "" {
		validatorsFile = genesis.GetDefaultPath(network, "validators")
	}
	cchainPath := genesis.GetDefaultPath(network, "cchain")
	airdropPath := genesis.GetDefaultPath(network, "airdrop")

	// Generate main genesis
	var mainGenesis *genesis.MainGenesis
	var err error

	switch network {
	case "mainnet":
		mainGenesis, err = genesis.BuildMainnet(validatorsFile, cchainPath)
	case "testnet":
		mainGenesis, err = genesis.BuildTestnet(cchainPath)
	default:
		return fmt.Errorf("unknown network: %s", network)
	}

	if err != nil {
		return fmt.Errorf("failed to build %s genesis: %w", network, err)
	}

	// Save main genesis
	genesisPath := filepath.Join(outputDir, fmt.Sprintf("genesis-%s-%d.json", network, genesis.GetNetworkID(network)))
	if err := genesis.SaveJSON(mainGenesis, genesisPath); err != nil {
		return fmt.Errorf("failed to save main genesis: %w", err)
	}

	fmt.Printf("\n✅ %s Genesis created: %s\n", network, genesisPath)

	// Generate X-Chain if requested
	if withXChain {
		xchainGenesis, err := genesis.BuildXChainGenesis(network, airdropPath)
		if err != nil {
			return fmt.Errorf("failed to build X-Chain genesis: %w", err)
		}

		xchainPath := filepath.Join(outputDir, fmt.Sprintf("xchain-genesis-%s.json", network))
		if err := genesis.SaveJSON(xchainGenesis, xchainPath); err != nil {
			return fmt.Errorf("failed to save X-Chain genesis: %w", err)
		}

		fmt.Printf("✅ X-Chain genesis created: %s\n", xchainPath)
	}

	// Generate subnet configs if requested
	if withSubnets {
		subnetConfigs, err := genesis.BuildSubnetConfigs(network)
		if err != nil {
			return fmt.Errorf("failed to build subnet configs: %w", err)
		}

		// Save each subnet config and genesis
		for name, config := range subnetConfigs {
			// Save config
			configPath := filepath.Join(outputDir, fmt.Sprintf("%s-subnet.json", config.SubnetID))
			if err := genesis.SaveSubnetConfig(config, configPath); err != nil {
				return fmt.Errorf("failed to save %s subnet config: %w", name, err)
			}
			fmt.Printf("✅ %s subnet config: %s\n", name, configPath)

			// Copy genesis if it exists
			def := genesis.GetSubnetDefinitions(network)[name]
			if genesisData, err := genesis.LoadSubnetGenesis(def.ConfigDir); err == nil {
				genesisPath := filepath.Join(outputDir, config.GenesisFile)
				if err := genesis.SaveSubnetGenesis(genesisData, genesisPath); err != nil {
					return fmt.Errorf("failed to save %s genesis: %w", name, err)
				}
				fmt.Printf("✅ %s genesis: %s\n", name, genesisPath)
			}
		}
	}

	fmt.Println("\n✅ Genesis generation complete!")
	return nil
}

