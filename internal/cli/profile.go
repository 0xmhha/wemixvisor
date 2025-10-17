package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/performance"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewProfileCommand creates the profile command
func NewProfileCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Performance profiling tools",
		Long: `Collect and manage performance profiles for debugging and optimization.

Subcommands:
  cpu        Capture CPU profile
  heap       Capture heap memory profile
  goroutine  Capture goroutine profile
  all        Capture all profile types
  list       List saved profiles
  clean      Clean old profiles

Examples:
  # Capture CPU profile for 30 seconds
  wemixvisor profile cpu --duration 30

  # Capture heap profile
  wemixvisor profile heap

  # Capture all profile types
  wemixvisor profile all

  # List saved profiles
  wemixvisor profile list

  # Clean profiles older than 7 days
  wemixvisor profile clean --max-age 168h`,
	}

	// Add subcommands
	cmd.AddCommand(newProfileCPUCommand(cfg, log))
	cmd.AddCommand(newProfileHeapCommand(cfg, log))
	cmd.AddCommand(newProfileGoroutineCommand(cfg, log))
	cmd.AddCommand(newProfileAllCommand(cfg, log))
	cmd.AddCommand(newProfileListCommand(cfg, log))
	cmd.AddCommand(newProfileCleanCommand(cfg, log))

	return cmd
}

// newProfileCPUCommand creates the CPU profile subcommand
func newProfileCPUCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		duration   int
		profileDir string
	)

	cmd := &cobra.Command{
		Use:   "cpu",
		Short: "Capture CPU profile",
		Long: `Capture CPU profiling data for performance analysis.

The CPU profile shows where the program spends CPU time.

Examples:
  # Profile for 30 seconds
  wemixvisor profile cpu --duration 30

  # Save to custom directory
  wemixvisor profile cpu --output /tmp/profiles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// Start CPU profiling
			log.Info("Starting CPU profiling", "duration", duration)
			if err := profiler.StartCPUProfile(); err != nil {
				return fmt.Errorf("failed to start CPU profiling: %w", err)
			}

			// Profile for specified duration
			time.Sleep(time.Duration(duration) * time.Second)

			// Stop CPU profiling
			if err := profiler.StopCPUProfile(); err != nil {
				return fmt.Errorf("failed to stop CPU profiling: %w", err)
			}

			log.Info("CPU profiling completed", "profile_dir", profileDir)
			log.Info("Analyze with: go tool pprof", "file", filepath.Join(profileDir, "cpu_*.prof"))

			return nil
		},
	}

	cmd.Flags().IntVar(&duration, "duration", 30, "Profiling duration in seconds")
	cmd.Flags().StringVar(&profileDir, "output", "", "Output directory for profiles")

	return cmd
}

// newProfileHeapCommand creates the heap profile subcommand
func newProfileHeapCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var profileDir string

	cmd := &cobra.Command{
		Use:   "heap",
		Short: "Capture heap memory profile",
		Long: `Capture heap memory profiling data.

The heap profile shows memory allocations and usage.

Examples:
  # Capture heap snapshot
  wemixvisor profile heap

  # Save to custom directory
  wemixvisor profile heap --output /tmp/profiles`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// Capture heap profile
			log.Info("Capturing heap profile")
			if err := profiler.WriteHeapProfile(); err != nil {
				return fmt.Errorf("failed to write heap profile: %w", err)
			}

			log.Info("Heap profile captured", "profile_dir", profileDir)
			log.Info("Analyze with: go tool pprof", "file", filepath.Join(profileDir, "heap_*.prof"))

			return nil
		},
	}

	cmd.Flags().StringVar(&profileDir, "output", "", "Output directory for profiles")

	return cmd
}

// newProfileGoroutineCommand creates the goroutine profile subcommand
func newProfileGoroutineCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var profileDir string

	cmd := &cobra.Command{
		Use:   "goroutine",
		Short: "Capture goroutine profile",
		Long: `Capture goroutine profiling data.

The goroutine profile shows all running goroutines and their stack traces.

Examples:
  # Capture goroutine snapshot
  wemixvisor profile goroutine`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// Capture goroutine profile
			log.Info("Capturing goroutine profile")
			if err := profiler.WriteGoroutineProfile(); err != nil {
				return fmt.Errorf("failed to write goroutine profile: %w", err)
			}

			// Show current goroutine count
			count := profiler.GetGoroutineCount()
			log.Info("Goroutine profile captured", "goroutines", count, "profile_dir", profileDir)

			return nil
		},
	}

	cmd.Flags().StringVar(&profileDir, "output", "", "Output directory for profiles")

	return cmd
}

// newProfileAllCommand creates the all profiles subcommand
func newProfileAllCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var profileDir string

	cmd := &cobra.Command{
		Use:   "all",
		Short: "Capture all profile types",
		Long: `Capture all available profile types: heap, goroutine, block, and mutex.

Examples:
  # Capture all profiles
  wemixvisor profile all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// Capture all profiles
			log.Info("Capturing all profile types")
			if err := profiler.WriteAllProfiles(); err != nil {
				return fmt.Errorf("failed to write profiles: %w", err)
			}

			log.Info("All profiles captured", "profile_dir", profileDir)
			log.Info("Analyze with: go tool pprof <profile_file>")

			return nil
		},
	}

	cmd.Flags().StringVar(&profileDir, "output", "", "Output directory for profiles")

	return cmd
}

// newProfileListCommand creates the list profiles subcommand
func newProfileListCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var profileDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved profiles",
		Long: `List all saved profiling data files.

Examples:
  # List all profiles
  wemixvisor profile list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// List profiles
			profiles, err := profiler.ListProfiles()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}

			if len(profiles) == 0 {
				fmt.Println("No profiles found")
				return nil
			}

			// Print profiles in table format
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer w.Flush()

			fmt.Fprintf(w, "TYPE\tFILENAME\tSIZE\tTIMESTAMP\n")
			fmt.Fprintf(w, "----\t--------\t----\t---------\n")

			for _, profile := range profiles {
				sizeKB := profile.Size / 1024
				fmt.Fprintf(w, "%s\t%s\t%d KB\t%s\n",
					profile.Type,
					profile.Filename,
					sizeKB,
					profile.Timestamp.Format("2006-01-02 15:04:05"),
				)
			}

			fmt.Fprintf(w, "\nTotal: %d profiles\n", len(profiles))

			return nil
		},
	}

	cmd.Flags().StringVar(&profileDir, "output", "", "Profile directory")

	return cmd
}

// newProfileCleanCommand creates the clean profiles subcommand
func newProfileCleanCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		profileDir string
		maxAge     string
	)

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean old profiles",
		Long: `Remove profiles older than the specified age.

Examples:
  # Clean profiles older than 7 days
  wemixvisor profile clean --max-age 168h

  # Clean all profiles older than 24 hours
  wemixvisor profile clean --max-age 24h`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if profileDir == "" {
				profileDir = filepath.Join(cfg.Home, "profiles")
			}

			// Parse max age
			maxAgeDuration, err := time.ParseDuration(maxAge)
			if err != nil {
				return fmt.Errorf("invalid max-age: %w", err)
			}

			// Create profiler
			profiler := performance.NewProfiler(profileDir, log)
			if err := profiler.Start(); err != nil {
				return fmt.Errorf("failed to start profiler: %w", err)
			}

			// Clean old profiles
			log.Info("Cleaning old profiles", "max_age", maxAge)
			if err := profiler.CleanOldProfiles(maxAgeDuration); err != nil {
				return fmt.Errorf("failed to clean profiles: %w", err)
			}

			log.Info("Profile cleanup completed")

			return nil
		},
	}

	cmd.Flags().StringVar(&profileDir, "output", "", "Profile directory")
	cmd.Flags().StringVar(&maxAge, "max-age", "168h", "Maximum age of profiles to keep (e.g., 24h, 7d)")

	return cmd
}
