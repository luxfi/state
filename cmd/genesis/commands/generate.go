package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewGenerateCommand returns a stub generate command
func NewGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate genesis (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: genesis generate")
			return nil
		},
	}
	return cmd
}
