package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

// NewUpgradeCommand creates the upgrade command with subcommands
func NewUpgradeCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Manage blockchain upgrades",
		Long:  `Schedule, monitor, and manage blockchain upgrades.`,
	}

	// Add subcommands
	cmd.AddCommand(newScheduleCommand(cfg, log))
	cmd.AddCommand(newUpgradeStatusCommand(cfg, log))
	cmd.AddCommand(newCancelCommand(cfg, log))

	return cmd
}

// newScheduleCommand creates the schedule subcommand
func newScheduleCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		binaries string
		checksum string
		info     string
	)

	cmd := &cobra.Command{
		Use:   "schedule <name> <height>",
		Short: "Schedule an upgrade at a specific block height",
		Long: `Schedule an upgrade to be executed at a specific block height.

The upgrade will be triggered automatically when the blockchain reaches
the specified height. The upgrade name should match the directory name
under the upgrades folder.

Examples:
  # Schedule upgrade "v1.2.0" at height 1000000
  wemixvisor upgrade schedule v1.2.0 1000000

  # Schedule with binary download URLs
  wemixvisor upgrade schedule v1.2.0 1000000 --binaries '{"linux/amd64":"https://..."}'

  # Schedule with checksum verification
  wemixvisor upgrade schedule v1.2.0 1000000 --checksum abc123...`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			heightStr := args[1]

			// Parse height
			height, err := strconv.ParseInt(heightStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid height '%s': must be a positive integer", heightStr)
			}

			if height <= 0 {
				return fmt.Errorf("height must be positive, got %d", height)
			}

			// Create upgrade info
			upgradeInfo := &types.UpgradeInfo{
				Name:   name,
				Height: height,
				Info:   make(map[string]interface{}),
			}

			// Add optional metadata
			if binaries != "" {
				var binMap map[string]string
				if err := json.Unmarshal([]byte(binaries), &binMap); err != nil {
					return fmt.Errorf("invalid binaries JSON: %w", err)
				}
				upgradeInfo.Info["binaries"] = binMap
			}

			if checksum != "" {
				upgradeInfo.Info["checksum"] = checksum
			}

			if info != "" {
				upgradeInfo.Info["description"] = info
			}

			// Write to upgrade-info.json
			upgradeInfoPath := cfg.UpgradeInfoFilePath()

			// Ensure data directory exists
			dataDir := filepath.Dir(upgradeInfoPath)
			if err := os.MkdirAll(dataDir, 0755); err != nil {
				return fmt.Errorf("failed to create data directory: %w", err)
			}

			if err := types.WriteUpgradeInfoFile(upgradeInfoPath, upgradeInfo); err != nil {
				return fmt.Errorf("failed to write upgrade info: %w", err)
			}

			log.Info("upgrade scheduled successfully",
				"name", name,
				"height", height,
				"file", upgradeInfoPath)

			// Print confirmation
			if cfg.JSONOutput {
				output, _ := json.MarshalIndent(upgradeInfo, "", "  ")
				fmt.Println(string(output))
			} else {
				fmt.Printf("✓ Upgrade scheduled successfully\n")
				fmt.Printf("  Name:   %s\n", name)
				fmt.Printf("  Height: %d\n", height)
				fmt.Printf("  File:   %s\n", upgradeInfoPath)
				fmt.Printf("\nThe upgrade will be triggered automatically when the blockchain reaches height %d.\n", height)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&binaries, "binaries", "", "Binary download URLs (JSON format)")
	cmd.Flags().StringVar(&checksum, "checksum", "", "Binary checksum for verification")
	cmd.Flags().StringVar(&info, "info", "", "Additional upgrade information")

	return cmd
}

// newUpgradeStatusCommand creates the status subcommand
func newUpgradeStatusCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current upgrade status",
		Long:  `Display the status of pending or active upgrades.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if upgrade-info.json exists
			upgradeInfoPath := cfg.UpgradeInfoFilePath()
			if _, err := os.Stat(upgradeInfoPath); os.IsNotExist(err) {
				if cfg.JSONOutput {
					output := map[string]interface{}{
						"status":  "no_upgrade",
						"message": "No upgrade scheduled",
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
				} else {
					fmt.Println("No upgrade scheduled")
				}
				return nil
			}

			// Read upgrade info
			upgradeInfo, err := types.ParseUpgradeInfoFile(upgradeInfoPath)
			if err != nil {
				return fmt.Errorf("failed to read upgrade info: %w", err)
			}

			if cfg.JSONOutput {
				output := map[string]interface{}{
					"status":  "scheduled",
					"upgrade": upgradeInfo,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("Upgrade Status: SCHEDULED\n\n")
				fmt.Printf("  Name:   %s\n", upgradeInfo.Name)
				fmt.Printf("  Height: %d\n", upgradeInfo.Height)

				if len(upgradeInfo.Info) > 0 {
					fmt.Printf("\nAdditional Info:\n")
					for key, value := range upgradeInfo.Info {
						fmt.Printf("  %s: %v\n", key, value)
					}
				}

				fmt.Printf("\nThe upgrade will trigger automatically when the blockchain reaches height %d.\n", upgradeInfo.Height)
			}

			return nil
		},
	}

	return cmd
}

// newCancelCommand creates the cancel subcommand
func newCancelCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a scheduled upgrade",
		Long:  `Cancel a scheduled upgrade by removing the upgrade-info.json file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			upgradeInfoPath := cfg.UpgradeInfoFilePath()

			// Check if upgrade exists
			if _, err := os.Stat(upgradeInfoPath); os.IsNotExist(err) {
				if cfg.JSONOutput {
					output := map[string]interface{}{
						"status":  "no_upgrade",
						"message": "No upgrade to cancel",
					}
					data, _ := json.MarshalIndent(output, "", "  ")
					fmt.Println(string(data))
				} else {
					fmt.Println("No upgrade to cancel")
				}
				return nil
			}

			// Read current upgrade info for confirmation
			upgradeInfo, err := types.ParseUpgradeInfoFile(upgradeInfoPath)
			if err != nil {
				return fmt.Errorf("failed to read upgrade info: %w", err)
			}

			// Confirm cancellation
			if !force && !cfg.Quiet {
				fmt.Printf("Cancel upgrade '%s' at height %d? (y/N): ", upgradeInfo.Name, upgradeInfo.Height)
				var response string
				fmt.Scanln(&response)
				if response != "y" && response != "Y" && response != "yes" {
					fmt.Println("Cancellation aborted")
					return nil
				}
			}

			// Remove upgrade-info.json
			if err := os.Remove(upgradeInfoPath); err != nil {
				return fmt.Errorf("failed to remove upgrade info: %w", err)
			}

			log.Info("upgrade cancelled",
				"name", upgradeInfo.Name,
				"height", upgradeInfo.Height)

			if cfg.JSONOutput {
				output := map[string]interface{}{
					"status":  "cancelled",
					"upgrade": upgradeInfo,
				}
				data, _ := json.MarshalIndent(output, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("✓ Upgrade cancelled successfully\n")
				fmt.Printf("  Name:   %s\n", upgradeInfo.Name)
				fmt.Printf("  Height: %d\n", upgradeInfo.Height)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompt")

	return cmd
}
