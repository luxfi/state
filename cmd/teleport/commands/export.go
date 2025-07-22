package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewExportCommand returns a stub export command
func NewExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export NFTs/tokens (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: teleport export")
			return nil
		},
	}
	return cmd
}
