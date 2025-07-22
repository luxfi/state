package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewLaunchCommand returns a stub launch command
func NewLaunchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch",
		Short: "Launch network (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis launch")
			return nil
		},
	}
	return cmd
}
