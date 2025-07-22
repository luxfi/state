package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewScanCommand returns a stub scan command
func NewScanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan external assets (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: archeology scan")
			return nil
		},
	}
	return cmd
}
