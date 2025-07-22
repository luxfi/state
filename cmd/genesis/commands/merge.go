package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewMergeCommand returns a stub merge command
func NewMergeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge genesis (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis merge")
			return nil
		},
	}
	return cmd
}
