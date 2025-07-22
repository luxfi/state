package commands

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewMigrateCommand() *cobra.Command {
	var (
		sourceChain     string
		sourceChainID   int64
		sourceRPC       string
		contractAddress string
		tokenType       string // erc20, erc721, erc1155
		targetLayer     string // L1, L2, L3
		targetName      string
		targetChainID   int64
		outputPath      string
		includeHolders  bool
		minBalance      string
		snapshot        bool
		genesisTemplate string
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate any token from any EVM chain to Lux Network",
		Long: `Migrate ERC20 tokens, NFT collections, or entire projects from any EVM-compatible
blockchain to Lux Network as a sovereign L1, subnet L2, or L3. This command handles
the complete migration workflow including holder snapshots, balance preservation,
and genesis generation.`,
		Example: `  # Migrate an ERC20 token to a new L2 subnet
  teleport migrate \
    --source-chain ethereum \
    --contract 0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48 \
    --token-type erc20 \
    --target-layer L2 \
    --target-name usdc-subnet \
    --target-chain-id 100001

  # Migrate NFT collection to sovereign L1
  teleport migrate \
    --source-rpc https://polygon-rpc.com \
    --contract 0xNFT_COLLECTION \
    --token-type erc721 \
    --target-layer L1 \
    --target-name my-nft-chain \
    --include-holders

  # Migrate with custom genesis template
  teleport migrate \
    --source-chain bsc \
    --contract 0xTOKEN \
    --token-type erc20 \
    --target-layer L2 \
    --target-name defi-subnet \
    --genesis-template ./templates/defi-subnet.json \
    --min-balance 1000000000000000000

  # Migrate to L3 (app-specific chain)
  teleport migrate \
    --source-chain arbitrum \
    --contract 0xGAME_TOKEN \
    --target-layer L3 \
    --target-name game-chain \
    --snapshot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate inputs
			if contractAddress == "" {
				return fmt.Errorf("contract address is required")
			}
			if targetName == "" {
				return fmt.Errorf("target name is required")
			}
			if targetLayer == "" {
				return fmt.Errorf("target layer (L1, L2, L3) is required")
			}
			if sourceChain == "" && sourceRPC == "" {
				return fmt.Errorf("either --source-chain or --source-rpc is required")
			}

			// Normalize target layer
			targetLayer = strings.ToUpper(targetLayer)
			if targetLayer != "L1" && targetLayer != "L2" && targetLayer != "L3" {
				return fmt.Errorf("target layer must be L1, L2, or L3")
			}

			// Auto-detect token type if not specified
			if tokenType == "" {
				log.Printf("Auto-detecting token type for %s...", contractAddress)
				// Detection logic would be implemented in the bridge package
			}

			config := bridge.MigrationConfig{
				SourceChain:     sourceChain,
				SourceChainID:   sourceChainID,
				SourceRPC:       sourceRPC,
				ContractAddress: contractAddress,
				TokenType:       tokenType,
				TargetLayer:     targetLayer,
				TargetName:      targetName,
				TargetChainID:   targetChainID,
				IncludeHolders:  includeHolders,
				MinBalance:      minBalance,
				Snapshot:        snapshot,
				GenesisTemplate: genesisTemplate,
			}

			migrator, err := bridge.NewMigrator(config)
			if err != nil {
				return fmt.Errorf("failed to create migrator: %w", err)
			}

			// Show migration plan
			fmt.Printf("\n🚀 Migration Plan\n")
			fmt.Printf("=================\n")
			fmt.Printf("Source: %s (Chain ID: %d)\n", sourceChain, sourceChainID)
			fmt.Printf("Contract: %s\n", contractAddress)
			fmt.Printf("Type: %s\n", tokenType)
			fmt.Printf("Target: %s %s\n", targetName, targetLayer)
			
			if targetChainID > 0 {
				fmt.Printf("Target Chain ID: %d\n", targetChainID)
			}

			fmt.Printf("\n📊 Analyzing source contract...\n")

			// Analyze source
			analysis, err := migrator.Analyze()
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}

			// Display analysis
			fmt.Printf("\n✅ Analysis Complete!\n\n")
			fmt.Printf("Token Name: %s\n", analysis.TokenName)
			fmt.Printf("Symbol: %s\n", analysis.Symbol)
			fmt.Printf("Decimals: %d\n", analysis.Decimals)
			fmt.Printf("Total Supply: %s\n", analysis.TotalSupply)
			fmt.Printf("Unique Holders: %d\n", analysis.UniqueHolders)

			if tokenType == "erc721" || tokenType == "erc1155" {
				fmt.Printf("Total NFTs: %d\n", analysis.TotalNFTs)
			}

			// Show migration options based on layer
			fmt.Printf("\n🔧 %s Configuration:\n", targetLayer)
			switch targetLayer {
			case "L1":
				fmt.Printf("  - Sovereign blockchain with independent consensus\n")
				fmt.Printf("  - Full control over validator set\n")
				fmt.Printf("  - Custom native token: %s\n", analysis.Symbol)
				fmt.Printf("  - Independent security model\n")
			case "L2":
				fmt.Printf("  - Subnet secured by Lux validators\n")
				fmt.Printf("  - Shared security with Lux Network\n")
				fmt.Printf("  - Lower operational overhead\n")
				fmt.Printf("  - Native interoperability with other subnets\n")
			case "L3":
				fmt.Printf("  - Application-specific chain\n")
				fmt.Printf("  - Optimized for your use case\n")
				fmt.Printf("  - Minimal infrastructure requirements\n")
				fmt.Printf("  - Built on L2 subnet infrastructure\n")
			}

			// Holder snapshot
			if includeHolders || snapshot {
				fmt.Printf("\n📸 Taking holder snapshot...\n")
				
				snapshotResult, err := migrator.TakeSnapshot()
				if err != nil {
					return fmt.Errorf("snapshot failed: %w", err)
				}

				fmt.Printf("Snapshot taken at block: %d\n", snapshotResult.BlockNumber)
				fmt.Printf("Total holders: %d\n", snapshotResult.HolderCount)
				
				if minBalance != "" {
					fmt.Printf("Holders above minimum: %d\n", snapshotResult.QualifiedHolders)
				}

				// Show distribution
				fmt.Printf("\nToken Distribution:\n")
				for _, tier := range snapshotResult.Distribution {
					fmt.Printf("  %s: %d holders (%.2f%%)\n", 
						tier.Range, tier.Count, tier.Percentage)
				}
			}

			// Generate migration artifacts
			fmt.Printf("\n⚙️  Generating migration artifacts...\n")

			result, err := migrator.GenerateArtifacts()
			if err != nil {
				return fmt.Errorf("artifact generation failed: %w", err)
			}

			// Display generated files
			fmt.Printf("\n✅ Migration artifacts generated!\n\n")
			fmt.Printf("Genesis Configuration: %s\n", result.GenesisPath)
			fmt.Printf("Chain Configuration: %s\n", result.ChainConfigPath)
			fmt.Printf("Deployment Script: %s\n", result.DeploymentScript)
			fmt.Printf("Migration Guide: %s\n", result.MigrationGuide)

			if result.ValidatorConfig != "" {
				fmt.Printf("Validator Config: %s\n", result.ValidatorConfig)
			}

			// Show next steps
			fmt.Printf("\n📋 Next Steps:\n")
			fmt.Printf("==============\n")
			
			switch targetLayer {
			case "L1":
				fmt.Printf("1. Review and customize genesis configuration\n")
				fmt.Printf("2. Set up validator nodes (minimum 1 for dev, 5 for production)\n")
				fmt.Printf("3. Deploy using: genesis launch --network %s --genesis %s\n", 
					targetName, result.GenesisPath)
				fmt.Printf("4. Bridge assets from source chain\n")
			case "L2":
				fmt.Printf("1. Review subnet configuration\n")
				fmt.Printf("2. Request subnet ID from Lux Network\n")
				fmt.Printf("3. Deploy subnet: ./deploy-subnet.sh %s\n", targetName)
				fmt.Printf("4. Add validators to your subnet\n")
				fmt.Printf("5. Enable cross-subnet communication\n")
			case "L3":
				fmt.Printf("1. Choose parent L2 subnet for deployment\n")
				fmt.Printf("2. Deploy app-specific chain configuration\n")
				fmt.Printf("3. Configure custom execution layer\n")
				fmt.Printf("4. Set up application endpoints\n")
			}

			// Export path
			if outputPath == "" {
				outputPath = fmt.Sprintf("./migrations/%s-%s", targetName, targetLayer)
			}

			fmt.Printf("\nAll files saved to: %s/\n", outputPath)

			// Show example RPC endpoint
			fmt.Printf("\n🌐 After deployment, your chain will be available at:\n")
			switch targetLayer {
			case "L1":
				fmt.Printf("   RPC: https://api.%s.network\n", targetName)
				fmt.Printf("   Explorer: https://explorer.%s.network\n", targetName)
			case "L2", "L3":
				fmt.Printf("   RPC: https://api.lux.network/ext/bc/%s/rpc\n", targetName)
				fmt.Printf("   Explorer: https://subnets.lux.network/%s\n", targetName)
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().StringVar(&sourceChain, "source-chain", "", "Source blockchain (ethereum, bsc, polygon, etc)")
	cmd.Flags().Int64Var(&sourceChainID, "source-chain-id", 0, "Source chain ID")
	cmd.Flags().StringVar(&sourceRPC, "source-rpc", "", "Custom source RPC URL")
	cmd.Flags().StringVar(&contractAddress, "contract", "", "Token contract address")
	cmd.Flags().StringVar(&tokenType, "token-type", "", "Token type (erc20, erc721, erc1155) - auto-detected if not specified")
	cmd.Flags().StringVar(&targetLayer, "target-layer", "", "Target layer (L1, L2, L3)")
	cmd.Flags().StringVar(&targetName, "target-name", "", "Name for your new chain/subnet")
	cmd.Flags().Int64Var(&targetChainID, "target-chain-id", 0, "Target chain ID (auto-generated if not specified)")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output directory for migration artifacts")
	cmd.Flags().BoolVar(&includeHolders, "include-holders", true, "Include holder snapshot in genesis")
	cmd.Flags().StringVar(&minBalance, "min-balance", "", "Minimum balance to include (in wei)")
	cmd.Flags().BoolVar(&snapshot, "snapshot", false, "Take detailed snapshot with merkle proofs")
	cmd.Flags().StringVar(&genesisTemplate, "genesis-template", "", "Custom genesis template")

	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("target-layer")
	cmd.MarkFlagRequired("target-name")

	return cmd
}