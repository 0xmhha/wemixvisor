package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewConfigCommand creates the config command
func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage wemixvisor configuration",
		Long:  `Manage wemixvisor configuration including templates, validation, and migration.`,
	}

	cmd.AddCommand(NewConfigShowCommand())
	cmd.AddCommand(NewConfigSetCommand())
	cmd.AddCommand(NewConfigValidateCommand())
	cmd.AddCommand(NewConfigTemplateCommand())
	cmd.AddCommand(NewConfigMigrateCommand())

	return cmd
}

// NewConfigShowCommand creates the config show command
func NewConfigShowCommand() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  `Display the current wemixvisor configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path from environment or flag
			configPath := getConfigPath()

			// Create logger
			logger := logger.NewTestLogger()

			// Create config manager
			manager, err := config.NewManager(configPath, logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			defer manager.Stop()

			// Get current config
			cfg := manager.GetConfig()

			// Display based on format
			switch format {
			case "json":
				data, err := json.MarshalIndent(cfg, "", "  ")
				if err != nil {
					return fmt.Errorf("failed to marshal config: %w", err)
				}
				fmt.Println(string(data))
			default:
				// Simple text output
				fmt.Printf("Home: %s\n", cfg.Home)
				fmt.Printf("Name: %s\n", cfg.Name)
				fmt.Printf("Network ID: %d\n", cfg.NetworkID)
				fmt.Printf("Chain ID: %s\n", cfg.ChainID)
				fmt.Printf("RPC Port: %d\n", cfg.RPCPort)
				fmt.Printf("Validator Mode: %v\n", cfg.ValidatorMode)
				fmt.Printf("Max Restarts: %d\n", cfg.MaxRestarts)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format (text, json)")

	return cmd
}

// NewConfigSetCommand creates the config set command
func NewConfigSetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Long:  `Set a specific configuration value.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			// Get config path
			configPath := getConfigPath()

			// Create logger
			logger := logger.NewTestLogger()

			// Create config manager
			manager, err := config.NewManager(configPath, logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			defer manager.Stop()

			// Update config
			if err := manager.UpdateConfig(key, value); err != nil {
				return fmt.Errorf("failed to update config: %w", err)
			}

			fmt.Printf("Configuration updated: %s = %s\n", key, value)
			return nil
		},
	}

	return cmd
}

// NewConfigValidateCommand creates the config validate command
func NewConfigValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate current configuration",
		Long:  `Validate the current wemixvisor configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath := getConfigPath()

			// Create logger
			logger := logger.NewTestLogger()

			// Create config manager
			manager, err := config.NewManager(configPath, logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			defer manager.Stop()

			// Validate config
			if err := manager.Validate(); err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			fmt.Println("Configuration is valid")
			return nil
		},
	}

	return cmd
}

// NewConfigTemplateCommand creates the config template command
func NewConfigTemplateCommand() *cobra.Command {
	var list bool

	cmd := &cobra.Command{
		Use:   "template [template-name]",
		Short: "Apply or list configuration templates",
		Long:  `Apply a configuration template or list available templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath := getConfigPath()

			// Create logger
			logger := logger.NewTestLogger()

			// Create config manager
			manager, err := config.NewManager(configPath, logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			defer manager.Stop()

			if list {
				// List available templates
				tm := config.NewTemplateManager(logger)
				templates := tm.ListTemplates()

				fmt.Println("Available templates:")
				for _, tmpl := range templates {
					fmt.Printf("  - %s: %s\n", tmpl.Name, tmpl.Description)
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("template name required")
			}

			// Apply template
			templateName := args[0]
			if err := manager.ApplyTemplate(templateName); err != nil {
				return fmt.Errorf("failed to apply template: %w", err)
			}

			fmt.Printf("Template '%s' applied successfully\n", templateName)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&list, "list", "l", false, "List available templates")

	return cmd
}

// NewConfigMigrateCommand creates the config migrate command
func NewConfigMigrateCommand() *cobra.Command {
	var fromVersion, toVersion string
	var backup bool

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate configuration between versions",
		Long:  `Migrate configuration from one version to another.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get config path
			configPath := getConfigPath()

			// Create logger
			logger := logger.NewTestLogger()

			// Create config manager
			manager, err := config.NewManager(configPath, logger)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}
			defer manager.Stop()

			// Default to current version to 0.5.0
			if fromVersion == "" {
				cfg := manager.GetConfig()
				if cfg.ConfigVersion != "" {
					fromVersion = cfg.ConfigVersion
				} else {
					fromVersion = "0.4.0"
				}
			}
			if toVersion == "" {
				toVersion = "0.5.0"
			}

			// Create backup if requested
			if backup {
				migrator := config.NewMigrator(logger)
				backupPath := configPath + ".backup"
				if err := migrator.BackupConfig(manager.GetConfig(), backupPath); err != nil {
					return fmt.Errorf("failed to create backup: %w", err)
				}
				fmt.Printf("Backup created: %s\n", backupPath)
			}

			// Perform migration
			if err := manager.Migrate(fromVersion, toVersion); err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}

			fmt.Printf("Configuration migrated from %s to %s\n", fromVersion, toVersion)
			return nil
		},
	}

	cmd.Flags().StringVar(&fromVersion, "from", "", "Source version (auto-detect if not specified)")
	cmd.Flags().StringVar(&toVersion, "to", "0.5.0", "Target version")
	cmd.Flags().BoolVar(&backup, "backup", true, "Create backup before migration")

	return cmd
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	// This would normally check environment variables or flags
	// For now, return a default
	return "./wemixvisor.toml"
}