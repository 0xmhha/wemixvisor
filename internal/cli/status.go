package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewStatusCommand creates the status command
func NewStatusCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show node status",
		Long:  `Display the current status of the managed node process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Checking node status...")

			// Check if process is running
			pidFile := fmt.Sprintf("%s/wemixvisor.pid", cfg.Home)
			if _, err := os.Stat(pidFile); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Node is not running")
					return nil
				}
				return fmt.Errorf("failed to check status: %w", err)
			}

			// Read PID from file
			data, err := os.ReadFile(pidFile)
			if err != nil {
				return fmt.Errorf("failed to read PID file: %w", err)
			}

			fmt.Printf("Node is running (PID: %s)\n", string(data))
			fmt.Printf("Binary: %s\n", cfg.CurrentBin())
			fmt.Printf("Home: %s\n", cfg.Home)

			// Check for pending upgrades
			upgradeInfo := cfg.UpgradeInfoFilePath()
			if _, err := os.Stat(upgradeInfo); err == nil {
				fmt.Println("Upgrade pending: Yes")
			} else {
				fmt.Println("Upgrade pending: No")
			}

			return nil
		},
	}

	return cmd
}

// NewStopCommand creates the stop command
func NewStopCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the node",
		Long:  `Stop the managed node process gracefully.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Stopping node...")

			// Check if process is running
			pidFile := fmt.Sprintf("%s/wemixvisor.pid", cfg.Home)
			if _, err := os.Stat(pidFile); err != nil {
				if os.IsNotExist(err) {
					fmt.Println("Node is not running")
					return nil
				}
				return fmt.Errorf("failed to check status: %w", err)
			}

			// TODO: Implement actual stop logic
			fmt.Println("Node stop command is not yet implemented")
			return nil
		},
	}

	return cmd
}

// NewRestartCommand creates the restart command
func NewRestartCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the node",
		Long:  `Restart the managed node process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Restarting node...")

			// TODO: Implement actual restart logic
			fmt.Println("Node restart command is not yet implemented")
			return nil
		},
	}

	return cmd
}