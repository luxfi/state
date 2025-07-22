package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewImportCommand returns a stub import command
func NewImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import assets (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis import")
			return nil
		},
	}
	return cmd
}
