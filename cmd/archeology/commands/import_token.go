package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewImportTokenCommand returns a stub import-token command
func NewImportTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-token",
		Short: "Import tokens (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology import-token")
			return nil
		},
	}
	return cmd
}
