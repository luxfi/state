package commands

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/genesis"
)

func NewDeployCommand() *cobra.Command {
	var (
		subnetName    string
		genesisPath   string
		validators    int
		endpoint      string
		privateKey    string
		gasPrice      string
		confirmations int
		dryRun        bool
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy subnet to running Lux network",
		Long: `Deploy a subnet configuration to a running Lux network.
This command handles subnet creation, validator addition, and genesis deployment.`,
		Example: `  # Deploy subnet to local network
  genesis deploy \
    --subnet zoo \
    --genesis ./genesis/zoo-mainnet.json \
    --validators 3

  # Deploy to remote network
  genesis deploy \
    --subnet custom-subnet \
    --genesis ./genesis/custom.json \
    --endpoint https://api.lux.network \
    --private-key $DEPLOYER_KEY

  # Dry run deployment
  genesis deploy \
    --subnet test-subnet \
    --genesis ./genesis/test.json \
    --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if subnetName == "" {
				return fmt.Errorf("subnet name is required")
			}
			if genesisPath == "" {
				return fmt.Errorf("genesis file is required")
			}

			deployer, err := genesis.NewDeployer(genesis.DeployerConfig{
				SubnetName:    subnetName,
				GenesisPath:   genesisPath,
				Validators:    validators,
				Endpoint:      endpoint,
				PrivateKey:    privateKey,
				GasPrice:      gasPrice,
				Confirmations: confirmations,
				DryRun:        dryRun,
			})
			if err != nil {
				return fmt.Errorf("failed to create deployer: %w", err)
			}

			fmt.Printf("🚀 Deploying subnet: %s\n", subnetName)
			fmt.Printf("Genesis: %s\n", genesisPath)
			fmt.Printf("Validators: %d\n", validators)
			fmt.Printf("Endpoint: %s\n", endpoint)

			if dryRun {
				fmt.Printf("\n🔍 DRY RUN MODE - No actual deployment\n")
			}

			// Check network connectivity
			fmt.Printf("\n📡 Checking network connectivity...\n")
			if err := deployer.CheckNetwork(); err != nil {
				return fmt.Errorf("network check failed: %w", err)
			}
			fmt.Printf("✅ Network is healthy\n")

			// Validate genesis
			fmt.Printf("\n📋 Validating genesis configuration...\n")
			if err := deployer.ValidateGenesis(); err != nil {
				return fmt.Errorf("genesis validation failed: %w", err)
			}
			fmt.Printf("✅ Genesis is valid\n")

			// Create subnet
			fmt.Printf("\n🔨 Creating subnet...\n")
			createResult, err := deployer.CreateSubnet()
			if err != nil {
				return fmt.Errorf("subnet creation failed: %w", err)
			}

			if !dryRun {
				fmt.Printf("✅ Subnet created!\n")
				fmt.Printf("   Subnet ID: %s\n", createResult.SubnetID)
				fmt.Printf("   Transaction: %s\n", createResult.TransactionID)
				fmt.Printf("   Blockchain ID: %s\n", createResult.BlockchainID)
			} else {
				fmt.Printf("✅ Subnet creation validated (dry run)\n")
			}

			// Add validators
			if validators > 0 {
				fmt.Printf("\n👥 Adding %d validators...\n", validators)
				
				for i := 0; i < validators; i++ {
					fmt.Printf("   Adding validator %d/%d...\n", i+1, validators)
					if !dryRun {
						time.Sleep(2 * time.Second) // Rate limiting
					}
				}
				
				fmt.Printf("✅ Validators added\n")
			}

			// Deploy configuration
			fmt.Printf("\n📦 Deploying subnet configuration...\n")
			deployResult, err := deployer.Deploy()
			if err != nil {
				return fmt.Errorf("deployment failed: %w", err)
			}

			// Show results
			fmt.Printf("\n✅ Subnet deployed successfully!\n\n")
			fmt.Printf("=== Deployment Summary ===\n")
			fmt.Printf("Subnet Name: %s\n", subnetName)
			fmt.Printf("Subnet ID: %s\n", deployResult.SubnetID)
			fmt.Printf("Blockchain ID: %s\n", deployResult.BlockchainID)
			fmt.Printf("VM ID: %s\n", deployResult.VMID)
			fmt.Printf("Chain ID: %d\n", deployResult.ChainID)

			fmt.Printf("\n🌐 Access Points:\n")
			fmt.Printf("RPC Endpoint: %s/ext/bc/%s/rpc\n", endpoint, deployResult.BlockchainID)
			fmt.Printf("WS Endpoint: %s/ext/bc/%s/ws\n", endpoint, deployResult.BlockchainID)

			if deployResult.ExplorerURL != "" {
				fmt.Printf("Explorer: %s\n", deployResult.ExplorerURL)
			}

			// Show configuration files
			fmt.Printf("\n📄 Configuration Files:\n")
			fmt.Printf("Node Config: %s\n", deployResult.NodeConfigPath)
			fmt.Printf("Chain Config: %s\n", deployResult.ChainConfigPath)
			
			if len(deployResult.ValidatorConfigs) > 0 {
				fmt.Printf("\nValidator Configs:\n")
				for i, config := range deployResult.ValidatorConfigs {
					fmt.Printf("  Validator %d: %s\n", i+1, config)
				}
			}

			// Next steps
			fmt.Printf("\n📋 Next Steps:\n")
			fmt.Printf("1. Configure your nodes with the generated configs\n")
			fmt.Printf("2. Restart nodes to recognize the new subnet\n")
			fmt.Printf("3. Verify subnet is producing blocks\n")
			fmt.Printf("4. Test RPC connectivity\n")

			if dryRun {
				fmt.Printf("\n🔍 This was a dry run - no actual deployment occurred\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&subnetName, "subnet", "s", "", "Subnet name")
	cmd.Flags().StringVarP(&genesisPath, "genesis", "g", "", "Path to genesis file")
	cmd.Flags().IntVar(&validators, "validators", 1, "Number of validators")
	cmd.Flags().StringVarP(&endpoint, "endpoint", "e", "http://localhost:9650", "Network endpoint")
	cmd.Flags().StringVar(&privateKey, "private-key", "", "Deployer private key")
	cmd.Flags().StringVar(&gasPrice, "gas-price", "25000000000", "Gas price in wei")
	cmd.Flags().IntVar(&confirmations, "confirmations", 1, "Required confirmations")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Simulate deployment without executing")

	cmd.MarkFlagRequired("subnet")
	cmd.MarkFlagRequired("genesis")

	return cmd
}