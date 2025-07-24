package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewImportNFTCommand returns a stub import-nft command
func NewImportNFTCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-nft",
		Short: "Import NFTs (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archaeology import-nft")
			return nil
		},
	}
	return cmd
}
