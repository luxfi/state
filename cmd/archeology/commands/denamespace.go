package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewDenamespaceCommand returns a stub denamespace command
func NewDenamespaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "denamespace",
		Short: "Denamespace DB (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology denamespace")
			return nil
		},
	}
	return cmd
}
