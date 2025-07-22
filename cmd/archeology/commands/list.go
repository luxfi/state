package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewListCommand returns a stub list command
func NewListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configs (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology list")
			return nil
		},
	}
	return cmd
}
