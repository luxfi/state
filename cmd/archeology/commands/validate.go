package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewValidateCommand returns a stub validate command
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate extraction (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology validate")
			return nil
		},
	}
	return cmd
}
