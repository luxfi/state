package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewAnalyzeCommand returns a stub analyze command
func NewAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze chain data (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archaeology analyze")
			return nil
		},
	}
	return cmd
}
