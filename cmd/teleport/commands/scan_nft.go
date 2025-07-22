package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewScanNFTCommand returns a stub scan-nft command
func NewScanNFTCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan-nft",
		Short: "Scan NFTs (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: teleport scan-nft")
			return nil
		},
	}
	return cmd
}
