package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	// Version is the current version of wemixvisor
	Version = "0.5.0"
	// GitCommit will be set by build flags
	GitCommit = "dev"
)

// NewVersionCommand creates the version command
func NewVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display the version information for wemixvisor.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("wemixvisor version: %s\n", Version)
			fmt.Printf("git commit: %s\n", GitCommit)
			fmt.Println("compatible with: WBFT consensus nodes")
		},
	}

	return cmd
}