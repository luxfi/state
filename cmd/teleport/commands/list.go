package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/bridge"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List supported chains and configurations",
		Long:  `List supported blockchain networks, RPC endpoints, and migration options.`,
	}

	cmd.AddCommand(
		newListChainsCommand(),
		newListProjectsCommand(),
		newListLayersCommand(),
	)

	return cmd
}

func newListChainsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "chains",
		Short: "List supported blockchain networks",
		RunE: func(cmd *cobra.Command, args []string) error {
			chains := bridge.GetSupportedChains()

			fmt.Println("Supported Blockchain Networks:")
			fmt.Println("==============================")
			fmt.Printf("%-15s %-10s %-50s %s\n", "Name", "Chain ID", "Default RPC", "Type")
			fmt.Println("--------------------------------------------------------------------------------")

			for _, chain := range chains {
				fmt.Printf("%-15s %-10d %-50s %s\n", 
					chain.Name, chain.ChainID, chain.DefaultRPC, chain.Type)
			}

			fmt.Println("\nLocal/Development Chains:")
			fmt.Println("========================")
			fmt.Println("- local (7777)    - Local Lux chain on port 9650")
			fmt.Println("- lux-genesis-7777        - Legacy Lux chain")
			fmt.Println("- lux-mainnet     - Current Lux mainnet (96369)")

			fmt.Println("\nUse --chain flag with any of these network names")
			fmt.Println("or specify custom RPC with --rpc flag")

			return nil
		},
	}
}

func newListProjectsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "projects",
		Short: "List known project configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects := bridge.GetKnownProjects()

			fmt.Println("Known Project Configurations:")
			fmt.Println("=============================")
			fmt.Printf("%-10s %-15s %-15s %s\n", "Project", "Token Symbol", "NFT Collections", "Networks")
			fmt.Println("---------------------------------------------------------------")

			for _, project := range projects {
				nftCount := len(project.NFTContracts)
				networks := ""
				for chain := range project.TokenContracts {
					if networks != "" {
						networks += ", "
					}
					networks += chain
				}
				fmt.Printf("%-10s %-15s %-15d %s\n", 
					project.Name, project.Symbol, nftCount, networks)
			}

			fmt.Println("\nStaking Powers (for validator NFTs):")
			fmt.Println("===================================")
			for _, project := range projects {
				if len(project.StakingPowers) > 0 {
					fmt.Printf("\n%s:\n", project.Name)
					for nftType, power := range project.StakingPowers {
						fmt.Printf("  %s: %s\n", nftType, power)
					}
				}
			}

			return nil
		},
	}
}

func newListLayersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "layers",
		Short: "List deployment layer options",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Lux Network Deployment Layers:")
			fmt.Println("==============================")
			fmt.Println()

			fmt.Println("L1 - Sovereign Blockchain")
			fmt.Println("-------------------------")
			fmt.Println("✓ Independent consensus and validator set")
			fmt.Println("✓ Custom native token")
			fmt.Println("✓ Full control over network parameters")
			fmt.Println("✓ Highest level of sovereignty")
			fmt.Println("✗ Requires managing validators")
			fmt.Println("✗ Higher operational overhead")
			fmt.Println()

			fmt.Println("L2 - Subnet (Secured by Lux)")
			fmt.Println("-----------------------------")
			fmt.Println("✓ Shared security with Lux Network")
			fmt.Println("✓ Native cross-subnet communication")
			fmt.Println("✓ Lower operational overhead")
			fmt.Println("✓ Built-in interoperability")
			fmt.Println("✗ Dependent on Lux validator set")
			fmt.Println("✗ Less customization flexibility")
			fmt.Println()

			fmt.Println("L3 - Application-Specific Chain")
			fmt.Println("-------------------------------")
			fmt.Println("✓ Optimized for specific use case")
			fmt.Println("✓ Minimal infrastructure requirements")
			fmt.Println("✓ Built on L2 infrastructure")
			fmt.Println("✓ Fastest deployment")
			fmt.Println("✗ Most constrained customization")
			fmt.Println("✗ Dependent on parent L2")
			fmt.Println()

			fmt.Println("Recommended Usage:")
			fmt.Println("==================")
			fmt.Println("L1: Large projects needing full sovereignty (>$100M TVL)")
			fmt.Println("L2: DeFi protocols, NFT platforms, general dApps")
			fmt.Println("L3: Games, specific applications, experiments")

			return nil
		},
	}
}