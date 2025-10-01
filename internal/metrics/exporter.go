package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// Exporter handles the HTTP server for Prometheus metrics
type Exporter struct {
	collector *Collector
	logger    *logger.Logger
	server    *http.Server
	port      int
	path      string
}

// NewExporter creates a new Prometheus exporter
func NewExporter(collector *Collector, port int, path string, logger *logger.Logger) *Exporter {
	if path == "" {
		path = "/metrics"
	}
	if port == 0 {
		port = 9090
	}

	return &Exporter{
		collector: collector,
		logger:    logger,
		port:      port,
		path:      path,
	}
}

// Start starts the Prometheus HTTP server
func (e *Exporter) Start() error {
	mux := http.NewServeMux()

	// Metrics endpoint
	handler := promhttp.HandlerFor(
		e.collector.GetRegistry(),
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
			Timeout:           10 * time.Second,
			ErrorLog:          e.logger.StdLogger(),
		},
	)
	mux.Handle(e.path, handler)

	// Health check endpoint
	mux.HandleFunc("/health", e.healthHandler)

	// Ready check endpoint
	mux.HandleFunc("/ready", e.readyHandler)

	// Metrics info endpoint
	mux.HandleFunc("/api/metrics/info", e.metricsInfoHandler)

	e.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", e.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		e.logger.Info("Starting Prometheus exporter",
			"port", e.port,
			"path", e.path)

		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			e.logger.Error("Prometheus exporter error", "error", err.Error())
		}
	}()

	return nil
}

// Stop stops the Prometheus HTTP server
func (e *Exporter) Stop() error {
	if e.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.server.Shutdown(ctx); err != nil {
		e.logger.Error("Failed to shutdown Prometheus exporter gracefully", "error", err.Error())
		return err
	}

	e.logger.Info("Prometheus exporter stopped")
	return nil
}

// healthHandler handles health check requests
func (e *Exporter) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// readyHandler handles readiness check requests
func (e *Exporter) readyHandler(w http.ResponseWriter, r *http.Request) {
	snapshot := e.collector.GetSnapshot()
	if snapshot == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"not_ready","message":"No metrics collected yet"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

// metricsInfoHandler provides information about available metrics
func (e *Exporter) metricsInfoHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"exporter": map[string]interface{}{
			"version": "0.7.0",
			"port":    e.port,
			"path":    e.path,
		},
		"collector": map[string]interface{}{
			"enabled":         e.collector.config.Enabled,
			"interval":        e.collector.config.CollectionInterval.String(),
			"system_metrics":  e.collector.config.EnableSystemMetrics,
			"app_metrics":     e.collector.config.EnableAppMetrics,
			"gov_metrics":     e.collector.config.EnableGovMetrics,
			"perf_metrics":    e.collector.config.EnablePerfMetrics,
		},
		"metrics": map[string][]string{
			"system": {
				"wemixvisor_cpu_usage_percent",
				"wemixvisor_memory_usage_percent",
				"wemixvisor_disk_usage_percent",
				"wemixvisor_goroutines",
				"wemixvisor_network_rx_bytes_total",
				"wemixvisor_network_tx_bytes_total",
			},
			"application": {
				"wemixvisor_upgrades_total",
				"wemixvisor_upgrades_success_total",
				"wemixvisor_upgrades_failed_total",
				"wemixvisor_upgrades_pending",
				"wemixvisor_process_restarts_total",
				"wemixvisor_process_uptime_seconds",
				"wemixvisor_node_height",
				"wemixvisor_node_peers",
				"wemixvisor_node_syncing",
			},
			"governance": {
				"wemixvisor_proposals_total",
				"wemixvisor_proposals_voting",
				"wemixvisor_proposals_passed_total",
				"wemixvisor_proposals_rejected_total",
				"wemixvisor_voting_power",
				"wemixvisor_voting_turnout_percent",
				"wemixvisor_validators_active",
				"wemixvisor_validators_jailed",
			},
			"performance": {
				"wemixvisor_rpc_latency_milliseconds",
				"wemixvisor_api_latency_milliseconds",
				"wemixvisor_transactions_per_second",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	// Simplified JSON encoding for demo
	fmt.Fprintf(w, `{
		"exporter": {
			"version": "0.7.0",
			"port": %d,
			"path": "%s"
		},
		"collector": {
			"enabled": %v,
			"interval": "%s",
			"system_metrics": %v,
			"app_metrics": %v,
			"gov_metrics": %v,
			"perf_metrics": %v
		}
	}`, e.port, e.path,
		e.collector.config.Enabled,
		e.collector.config.CollectionInterval.String(),
		e.collector.config.EnableSystemMetrics,
		e.collector.config.EnableAppMetrics,
		e.collector.config.EnableGovMetrics,
		e.collector.config.EnablePerfMetrics)

	_ = info // Avoid unused variable warning
}

// GetURL returns the URL of the metrics endpoint
func (e *Exporter) GetURL() string {
	return fmt.Sprintf("http://localhost:%d%s", e.port, e.path)
}