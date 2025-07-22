package commands

import (
	"fmt"
	"github.com/spf13/cobra"
)

// NewScanTokenCommand returns a stub scan-token command
func NewScanTokenCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scan-token",
		Short: "Scan tokens (stub)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("stub: teleport scan-token")
			return nil
		},
	}
	return cmd
}
