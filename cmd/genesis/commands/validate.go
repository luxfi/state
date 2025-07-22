package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewValidateCommand returns a stub validate command
func NewValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate genesis (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis validate")
			return nil
		},
	}
	return cmd
}
