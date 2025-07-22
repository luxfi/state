package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewDeployCommand returns a stub deploy command
func NewDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy subnet (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis deploy")
			return nil
		},
	}
	return cmd
}
