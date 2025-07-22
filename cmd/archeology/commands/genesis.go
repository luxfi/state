package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewGenesisCommand returns a stub X-Chain/P-Chain genesis command.
func NewGenesisCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genesis",
		Short: "Generate genesis (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology genesis")
			return nil
		},
	}
	return cmd
}
