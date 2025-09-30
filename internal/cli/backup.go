package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/backup"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewBackupCommand creates the backup command
func NewBackupCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
		Long:  `Manage backup operations for the node data.`,
	}

	cmd.AddCommand(newBackupCreateCommand(cfg, logger))
	cmd.AddCommand(newBackupRestoreCommand(cfg, logger))
	cmd.AddCommand(newBackupListCommand(cfg, logger))
	cmd.AddCommand(newBackupCleanCommand(cfg, logger))

	return cmd
}

func newBackupCreateCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a backup",
		Long:  `Create a backup of the current node data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Creating backup...")

			// Create backup manager
			manager := backup.NewManager(cfg, logger)

			// Create backup
			backupPath, err := manager.CreateBackup("manual")
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}

			fmt.Printf("Backup created: %s\n", backupPath)
			return nil
		},
	}

	return cmd
}

func newBackupRestoreCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [backup-name]",
		Short: "Restore from a backup",
		Long:  `Restore node data from a backup.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupName := args[0]

			logger.Info("Restoring from backup: " + backupName)

			// Create backup manager
			manager := backup.NewManager(cfg, logger)

			// Restore backup
			if err := manager.RestoreBackup(backupName); err != nil {
				return fmt.Errorf("failed to restore backup: %w", err)
			}

			fmt.Printf("Backup restored: %s\n", backupName)
			return nil
		},
	}

	return cmd
}

func newBackupListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		Long:  `List all available backups.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Listing backups...")

			// Create backup manager
			manager := backup.NewManager(cfg, logger)

			// List backups
			backups, err := manager.ListBackups()
			if err != nil {
				return fmt.Errorf("failed to list backups: %w", err)
			}

			if len(backups) == 0 {
				fmt.Println("No backups found")
				return nil
			}

			fmt.Println("Available backups:")
			for _, backupName := range backups {
				fmt.Printf("  - %s\n", backupName)
			}

			return nil
		},
	}

	return cmd
}

func newBackupCleanCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
	var maxAgeDays int

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean old backups",
		Long:  `Clean old backups older than specified days.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger.Info("Cleaning old backups...")

			// Create backup manager
			manager := backup.NewManager(cfg, logger)

			// Clean old backups (convert days to duration)
			maxAge := time.Duration(maxAgeDays) * 24 * time.Hour
			err := manager.CleanOldBackups(maxAge)
			if err != nil {
				return fmt.Errorf("failed to clean backups: %w", err)
			}

			fmt.Println("Old backups cleaned successfully")
			return nil
		},
	}

	cmd.Flags().IntVar(&maxAgeDays, "max-age-days", 7, "Maximum age of backups to keep (in days)")

	return cmd
}