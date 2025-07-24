package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewDenamespaceCommand returns a stub namespace command
func NewDenamespaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "namespace",
		Short: "Denamespace DB (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archaeology namespace")
			return nil
		},
	}
	return cmd
}
