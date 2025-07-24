package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewExtractCommand returns a stub extract command
func NewExtractCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extract",
		Short: "Extract chain data (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archaeology extract")
			return nil
		},
	}
	return cmd
}
