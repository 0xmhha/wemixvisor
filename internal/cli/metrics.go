package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewMetricsCommand creates the metrics command
func NewMetricsCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Manage and view system metrics",
		Long: `Collect, view, and export system performance metrics.

Subcommands:
  collect   Start metrics collection
  show      Display current metrics
  export    Export metrics to Prometheus format

Examples:
  # Start metrics collection
  wemixvisor metrics collect

  # Show current metrics
  wemixvisor metrics show

  # Export metrics to Prometheus
  wemixvisor metrics export --port 9090`,
	}

	// Add subcommands
	cmd.AddCommand(newMetricsCollectCommand(cfg, log))
	cmd.AddCommand(newMetricsShowCommand(cfg, log))
	cmd.AddCommand(newMetricsExportCommand(cfg, log))

	return cmd
}

// newMetricsCollectCommand creates the metrics collect subcommand
func newMetricsCollectCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		interval            int
		enableSystemMetrics bool
		duration            int
	)

	cmd := &cobra.Command{
		Use:   "collect",
		Short: "Start collecting metrics",
		Long: `Start collecting system and application metrics.

Examples:
  # Collect metrics every 10 seconds
  wemixvisor metrics collect --interval 10

  # Collect for 60 seconds
  wemixvisor metrics collect --duration 60`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create collector config
			collectorConfig := &metrics.CollectorConfig{
				Enabled:             true,
				CollectionInterval:  time.Duration(interval) * time.Second,
				EnableSystemMetrics: enableSystemMetrics,
			}

			// Create and start collector
			collector := metrics.NewCollector(collectorConfig, log)
			if err := collector.Start(); err != nil {
				return fmt.Errorf("failed to start collector: %w", err)
			}
			defer collector.Stop()

			log.Info("Metrics collection started", "interval", interval, "duration", duration)

			// Collect for specified duration
			if duration > 0 {
				time.Sleep(time.Duration(duration) * time.Second)
				log.Info("Collection completed")

				// Show final snapshot
				snapshot := collector.GetSnapshot()
				if snapshot != nil {
					printMetricsSnapshot(snapshot)
				}
			} else {
				// Collect indefinitely
				select {}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&interval, "interval", 10, "Collection interval in seconds")
	cmd.Flags().BoolVar(&enableSystemMetrics, "system-metrics", true, "Enable system metrics")
	cmd.Flags().IntVar(&duration, "duration", 0, "Collection duration in seconds (0 = indefinite)")

	return cmd
}

// newMetricsShowCommand creates the metrics show subcommand
func newMetricsShowCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		jsonOutput bool
		watch      bool
		interval   int
	)

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current metrics",
		Long: `Display current system and application metrics.

Examples:
  # Show metrics once
  wemixvisor metrics show

  # Show metrics in JSON format
  wemixvisor metrics show --json

  # Watch metrics with auto-refresh
  wemixvisor metrics show --watch --interval 5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create collector
			collectorConfig := &metrics.CollectorConfig{
				Enabled:             true,
				CollectionInterval:  time.Duration(interval) * time.Second,
				EnableSystemMetrics: true,
			}

			collector := metrics.NewCollector(collectorConfig, log)
			if err := collector.Start(); err != nil {
				return fmt.Errorf("failed to start collector: %w", err)
			}
			defer collector.Stop()

			// Wait for first snapshot
			time.Sleep(time.Duration(interval) * time.Second)

			if watch {
				// Watch mode - refresh periodically
				ticker := time.NewTicker(time.Duration(interval) * time.Second)
				defer ticker.Stop()

				for {
					snapshot := collector.GetSnapshot()
					if snapshot != nil {
						// Clear screen
						fmt.Print("\033[H\033[2J")

						if jsonOutput {
							printMetricsJSON(snapshot)
						} else {
							printMetricsSnapshot(snapshot)
						}
					}
					<-ticker.C
				}
			} else {
				// Show once
				snapshot := collector.GetSnapshot()
				if snapshot == nil {
					return fmt.Errorf("no metrics available")
				}

				if jsonOutput {
					printMetricsJSON(snapshot)
				} else {
					printMetricsSnapshot(snapshot)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&watch, "watch", false, "Watch mode with auto-refresh")
	cmd.Flags().IntVar(&interval, "interval", 5, "Refresh interval in seconds")

	return cmd
}

// newMetricsExportCommand creates the metrics export subcommand
func newMetricsExportCommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export metrics in Prometheus format",
		Long: `Start Prometheus metrics exporter.

The exporter provides a /metrics endpoint for Prometheus to scrape.

Examples:
  # Start exporter on default port (9090)
  wemixvisor metrics export

  # Start on custom port
  wemixvisor metrics export --port 9100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create collector
			collectorConfig := &metrics.CollectorConfig{
				Enabled:             true,
				CollectionInterval:  10 * time.Second,
				EnableSystemMetrics: true,
			}

			collector := metrics.NewCollector(collectorConfig, log)
			if err := collector.Start(); err != nil {
				return fmt.Errorf("failed to start collector: %w", err)
			}
			defer collector.Stop()

			// Create exporter
			exporter := metrics.NewExporter(collector, port, "/metrics", log)
			if err := exporter.Start(); err != nil {
				return fmt.Errorf("failed to start exporter: %w", err)
			}
			defer exporter.Stop()

			log.Info("Metrics exporter started", "port", port)
			log.Info("Prometheus endpoint", "url", fmt.Sprintf("http://localhost:%d/metrics", port))

			// Wait indefinitely
			select {}
		},
	}

	cmd.Flags().IntVar(&port, "port", 9090, "Exporter port")

	return cmd
}

// printMetricsSnapshot prints metrics in human-readable format
func printMetricsSnapshot(snapshot *metrics.MetricsSnapshot) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "=== System Metrics ===\n")
	fmt.Fprintf(w, "Timestamp:\t%s\n", snapshot.Timestamp.Format(time.RFC3339))

	if snapshot.System != nil {
		fmt.Fprintf(w, "\nCPU:\t%.2f%%\n", snapshot.System.CPUUsage)
		fmt.Fprintf(w, "Memory Usage:\t%.2f%%\n", snapshot.System.MemoryUsage)
		fmt.Fprintf(w, "Memory Total:\t%d MB\n", snapshot.System.MemoryTotal/(1024*1024))
		fmt.Fprintf(w, "Memory Available:\t%d MB\n", snapshot.System.MemoryAvailable/(1024*1024))
		fmt.Fprintf(w, "Disk Usage:\t%.2f%%\n", snapshot.System.DiskUsage)
		fmt.Fprintf(w, "Goroutines:\t%d\n", snapshot.System.Goroutines)
		fmt.Fprintf(w, "Uptime:\t%d seconds\n", snapshot.System.Uptime)
	}

	if snapshot.Application != nil {
		fmt.Fprintf(w, "\n=== Application Metrics ===\n")
		fmt.Fprintf(w, "Upgrade Total:\t%d\n", snapshot.Application.UpgradeTotal)
		fmt.Fprintf(w, "Upgrade Success:\t%d\n", snapshot.Application.UpgradeSuccess)
		fmt.Fprintf(w, "Upgrade Failed:\t%d\n", snapshot.Application.UpgradeFailed)
		fmt.Fprintf(w, "Upgrade Pending:\t%d\n", snapshot.Application.UpgradePending)
		fmt.Fprintf(w, "Process Restarts:\t%d\n", snapshot.Application.ProcessRestarts)
	}

	if snapshot.Governance != nil {
		fmt.Fprintf(w, "\n=== Governance Metrics ===\n")
		fmt.Fprintf(w, "Proposal Total:\t%d\n", snapshot.Governance.ProposalTotal)
		fmt.Fprintf(w, "Proposal Voting:\t%d\n", snapshot.Governance.ProposalVoting)
		fmt.Fprintf(w, "Proposal Passed:\t%d\n", snapshot.Governance.ProposalPassed)
		fmt.Fprintf(w, "Proposal Rejected:\t%d\n", snapshot.Governance.ProposalRejected)
		fmt.Fprintf(w, "Voting Turnout:\t%.2f%%\n", snapshot.Governance.VotingTurnout*100)
	}
}

// printMetricsJSON prints metrics in JSON format
func printMetricsJSON(snapshot *metrics.MetricsSnapshot) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling metrics: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
