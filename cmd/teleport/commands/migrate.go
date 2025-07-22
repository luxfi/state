package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewMigrateCommand returns a stub migrate command
func NewMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate assets (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: teleport migrate")
			return nil
		},
	}
	return cmd
}
