package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/luxfi/genesis/pkg/archeology"
)

func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List various configurations and options",
		Long:  `List known networks, chain IDs, and other configuration options.`,
	}

	cmd.AddCommand(
		newListNetworksCommand(),
		newListPrefixesCommand(),
	)

	return cmd
}

func newListNetworksCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "networks",
		Short: "List known network configurations",
		RunE: func(cmd *cobra.Command, args []string) error {
			networks := archeology.GetKnownNetworks()

			fmt.Println("Known Network Configurations:")
			fmt.Println("=============================")
			fmt.Printf("%-20s %-10s %-50s\n", "Network", "Chain ID", "Blockchain ID")
			fmt.Println("---------------------------------------------------------------------")

			for _, net := range networks {
				fmt.Printf("%-20s %-10d %-50s\n", net.Name, net.ChainID, net.BlockchainID)
			}

			fmt.Println("\nUse --network flag with any of these network names")
			fmt.Println("or specify --chain-id directly")

			return nil
		},
	}
}

func newListPrefixesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prefixes",
		Short: "List known database key prefixes",
		RunE: func(cmd *cobra.Command, args []string) error {
			prefixes := archeology.GetKnownPrefixes()

			fmt.Println("Database Key Prefixes:")
			fmt.Println("=====================")
			fmt.Printf("%-20s %-10s %s\n", "Name", "Prefix", "Description")
			fmt.Println("---------------------------------------------------------------")

			for _, p := range prefixes {
				fmt.Printf("%-20s 0x%-8s %s\n", p.Name, fmt.Sprintf("%x", p.Prefix), p.Description)
			}

			return nil
		},
	}
}