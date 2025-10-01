package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Custom metrics for monitoring
var (
	// Counter for processed transactions
	transactionsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wemix_transactions_processed_total",
			Help: "Total number of processed transactions",
		},
		[]string{"status"},
	)

	// Gauge for active connections
	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "wemix_active_connections",
			Help: "Number of active connections",
		},
	)

	// Histogram for request duration
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wemix_request_duration_seconds",
			Help:    "Request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// Summary for block processing time
	blockProcessingTime = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:       "wemix_block_processing_seconds",
			Help:       "Block processing time in seconds",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
	)
)

func init() {
	// Register all custom metrics
	prometheus.MustRegister(transactionsProcessed)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(blockProcessingTime)
}

// MonitoringService simulates a service that generates metrics
type MonitoringService struct {
	stopCh chan struct{}
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		stopCh: make(chan struct{}),
	}
}

// Start starts the monitoring service
func (s *MonitoringService) Start() {
	fmt.Println("Starting monitoring service...")

	// Start metric generation
	go s.generateMetrics()

	// Start HTTP server for metrics endpoint
	go s.serveMetrics()
}

// Stop stops the monitoring service
func (s *MonitoringService) Stop() {
	close(s.stopCh)
	fmt.Println("Monitoring service stopped")
}

// generateMetrics simulates metric generation
func (s *MonitoringService) generateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	connections := 0.0

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			// Simulate transaction processing
			if time.Now().Unix()%2 == 0 {
				transactionsProcessed.WithLabelValues("success").Inc()
			} else {
				transactionsProcessed.WithLabelValues("failed").Inc()
			}

			// Simulate connection changes
			connections += (float64(time.Now().Unix()%10) - 5)
			if connections < 0 {
				connections = 0
			}
			activeConnections.Set(connections)

			// Simulate request duration
			duration := float64(time.Now().Unix()%10) / 10.0
			requestDuration.WithLabelValues("/api/v1/status").Observe(duration)

			// Simulate block processing
			processingTime := float64(time.Now().Unix()%5) / 2.0
			blockProcessingTime.Observe(processingTime)

			fmt.Printf("Metrics updated - Connections: %.0f, Processing: %.2fs\n",
				connections, processingTime)
		}
	}
}

// serveMetrics starts HTTP server for metrics
func (s *MonitoringService) serveMetrics() {
	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Metrics server listening on :9100")
	if err := http.ListenAndServe(":9100", nil); err != nil {
		log.Printf("Error starting metrics server: %v", err)
	}
}

// HealthChecker performs health checks
type HealthChecker struct {
	services map[string]ServiceHealth
}

// ServiceHealth represents health status of a service
type ServiceHealth struct {
	Name      string
	Healthy   bool
	LastCheck time.Time
	Message   string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		services: make(map[string]ServiceHealth),
	}
}

// CheckHealth checks health of all services
func (h *HealthChecker) CheckHealth() {
	// Check API service
	apiHealth := h.checkAPI()
	h.services["api"] = apiHealth

	// Check node service
	nodeHealth := h.checkNode()
	h.services["node"] = nodeHealth

	// Check database
	dbHealth := h.checkDatabase()
	h.services["database"] = dbHealth

	// Print health status
	fmt.Println("\n=== Health Check Results ===")
	for name, health := range h.services {
		status := "❌"
		if health.Healthy {
			status = "✅"
		}
		fmt.Printf("%s %s: %s (checked: %s)\n",
			status, name, health.Message, health.LastCheck.Format("15:04:05"))
	}
}

// checkAPI checks API service health
func (h *HealthChecker) checkAPI() ServiceHealth {
	health := ServiceHealth{
		Name:      "API",
		LastCheck: time.Now(),
	}

	// Simulate API health check
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		health.Healthy = false
		health.Message = fmt.Sprintf("Failed to connect: %v", err)
		return health
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		health.Healthy = true
		health.Message = "API is responding normally"
	} else {
		health.Healthy = false
		health.Message = fmt.Sprintf("API returned status %d", resp.StatusCode)
	}

	return health
}

// checkNode checks node health
func (h *HealthChecker) checkNode() ServiceHealth {
	health := ServiceHealth{
		Name:      "Node",
		LastCheck: time.Now(),
	}

	// Simulate node health check
	// In real implementation, this would check actual node status
	if time.Now().Unix()%10 < 8 {
		health.Healthy = true
		health.Message = "Node is syncing normally"
	} else {
		health.Healthy = false
		health.Message = "Node is not syncing"
	}

	return health
}

// checkDatabase checks database health
func (h *HealthChecker) checkDatabase() ServiceHealth {
	health := ServiceHealth{
		Name:      "Database",
		LastCheck: time.Now(),
	}

	// Simulate database health check
	// In real implementation, this would ping the actual database
	health.Healthy = true
	health.Message = "Database connection is stable"

	return health
}

// AlertMonitor monitors for alert conditions
type AlertMonitor struct {
	thresholds map[string]float64
}

// NewAlertMonitor creates a new alert monitor
func NewAlertMonitor() *AlertMonitor {
	return &AlertMonitor{
		thresholds: map[string]float64{
			"cpu_usage":    80.0,
			"memory_usage": 90.0,
			"disk_usage":   85.0,
		},
	}
}

// CheckAlerts checks for alert conditions
func (a *AlertMonitor) CheckAlerts() {
	fmt.Println("\n=== Alert Check ===")

	// Simulate metric values
	cpuUsage := float64(time.Now().Unix()%100)
	memoryUsage := float64((time.Now().Unix()+20)%100)
	diskUsage := float64((time.Now().Unix()+40)%100)

	// Check CPU usage
	if cpuUsage > a.thresholds["cpu_usage"] {
		fmt.Printf("⚠️  ALERT: High CPU usage: %.1f%% (threshold: %.1f%%)\n",
			cpuUsage, a.thresholds["cpu_usage"])
	} else {
		fmt.Printf("✅ CPU usage normal: %.1f%%\n", cpuUsage)
	}

	// Check memory usage
	if memoryUsage > a.thresholds["memory_usage"] {
		fmt.Printf("⚠️  ALERT: High memory usage: %.1f%% (threshold: %.1f%%)\n",
			memoryUsage, a.thresholds["memory_usage"])
	} else {
		fmt.Printf("✅ Memory usage normal: %.1f%%\n", memoryUsage)
	}

	// Check disk usage
	if diskUsage > a.thresholds["disk_usage"] {
		fmt.Printf("⚠️  ALERT: Low disk space: %.1f%% used (threshold: %.1f%%)\n",
			diskUsage, a.thresholds["disk_usage"])
	} else {
		fmt.Printf("✅ Disk usage normal: %.1f%%\n", diskUsage)
	}
}

func main() {
	// Create monitoring service
	monitoringService := NewMonitoringService()
	monitoringService.Start()

	// Create health checker
	healthChecker := NewHealthChecker()

	// Create alert monitor
	alertMonitor := NewAlertMonitor()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Start monitoring loop
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Perform health checks
				healthChecker.CheckHealth()

				// Check for alerts
				alertMonitor.CheckAlerts()

				fmt.Println("\n" + strings.Repeat("-", 50))
			}
		}
	}()

	fmt.Println("Monitoring system started. Press Ctrl+C to stop.")
	fmt.Println("Metrics available at http://localhost:9100/metrics")

	// Wait for shutdown signal
	<-sigCh
	fmt.Println("\nShutting down...")

	// Stop monitoring
	cancel()
	monitoringService.Stop()

	fmt.Println("Monitoring system stopped.")
}

// Helper function
var strings = struct {
	Repeat func(s string, count int) string
}{
	Repeat: func(s string, count int) string {
		result := ""
		for i := 0; i < count; i++ {
			result += s
		}
		return result
	},
}