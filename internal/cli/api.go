package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wemix/wemixvisor/internal/api"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/governance"
	"github.com/wemix/wemixvisor/internal/metrics"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// NewAPICommand creates the API server command
func NewAPICommand(cfg *config.Config, log *logger.Logger) *cobra.Command {
	var (
		port                int
		enableMetrics       bool
		enableGovernance    bool
		metricsInterval     int
		enableSystemMetrics bool
	)

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Start the API server",
		Long: `Start the RESTful API server with WebSocket support for real-time monitoring.

The API server provides:
- RESTful HTTP endpoints for status, metrics, and configuration
- WebSocket connections for real-time updates
- Prometheus metrics export
- Governance proposal monitoring
- System performance metrics

Examples:
  # Start API server on default port (8080)
  wemixvisor api start

  # Start with custom port
  wemixvisor api start --port 9090

  # Start with metrics collection
  wemixvisor api start --enable-metrics

  # Start with full monitoring
  wemixvisor api start --enable-metrics --enable-governance`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Create metrics collector if enabled
			var collector *metrics.Collector
			if enableMetrics {
				collectorConfig := &metrics.CollectorConfig{
					Enabled:             true,
					CollectionInterval:  time.Duration(metricsInterval) * time.Second,
					EnableSystemMetrics: enableSystemMetrics,
				}
				collector = metrics.NewCollector(collectorConfig, log)
				if err := collector.Start(); err != nil {
					return fmt.Errorf("failed to start metrics collector: %w", err)
				}
				defer collector.Stop()
				log.Info("Metrics collector started", "interval", metricsInterval)
			}

			// Create governance monitor if enabled
			var monitor *governance.Monitor
			if enableGovernance {
				monitor = governance.NewMonitor(cfg, log)
				if err := monitor.Start(); err != nil {
					return fmt.Errorf("failed to start governance monitor: %w", err)
				}
				defer monitor.Stop()
				log.Info("Governance monitor started")
			}

			// Update config with API port
			cfg.APIPort = port

			// Create and start API server
			server := api.NewServer(cfg, monitor, collector, log)
			if err := server.Start(); err != nil {
				return fmt.Errorf("failed to start API server: %w", err)
			}

			log.Info("API server started successfully", "port", port)
			log.Info("Access API at", "url", fmt.Sprintf("http://localhost:%d", port))
			log.Info("Health check", "url", fmt.Sprintf("http://localhost:%d/health", port))
			log.Info("API documentation", "url", fmt.Sprintf("http://localhost:%d/api/v1", port))
			if enableMetrics {
				log.Info("Metrics endpoint", "url", fmt.Sprintf("http://localhost:%d/api/v1/metrics", port))
			}

			// Wait for interrupt signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			log.Info("Shutting down API server...")
			if err := server.Stop(); err != nil {
				return fmt.Errorf("error stopping API server: %w", err)
			}

			log.Info("API server stopped successfully")
			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVar(&port, "port", 8080, "API server port")
	cmd.Flags().BoolVar(&enableMetrics, "enable-metrics", true, "Enable metrics collection")
	cmd.Flags().BoolVar(&enableGovernance, "enable-governance", true, "Enable governance monitoring")
	cmd.Flags().IntVar(&metricsInterval, "metrics-interval", 10, "Metrics collection interval in seconds")
	cmd.Flags().BoolVar(&enableSystemMetrics, "enable-system-metrics", true, "Enable system metrics collection")

	return cmd
}
