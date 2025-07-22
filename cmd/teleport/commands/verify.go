package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewVerifyCommand returns a stub verify command
func NewVerifyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "verify",
		Short: "Verify assets (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: teleport verify")
			return nil
		},
	}
	return cmd
}
