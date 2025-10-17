#!/bin/bash
# Start wemixvisor with full monitoring enabled

set -e

# Configuration
CONFIG_FILE="${WEMIXVISOR_CONFIG:-/opt/wemixd/config/wemixvisor.toml}"
API_PORT="${WEMIXVISOR_API_PORT:-8080}"
METRICS_PORT="${WEMIXVISOR_METRICS_PORT:-9090}"

echo "Starting wemixvisor with monitoring..."
echo "Config: $CONFIG_FILE"
echo "API Port: $API_PORT"
echo "Metrics Port: $METRICS_PORT"

# Start API server in background
wemixvisor api \
  --port $API_PORT \
  --enable-metrics \
  --enable-governance \
  --metrics-interval 10 \
  --enable-system-metrics &

API_PID=$!
echo "API server started (PID: $API_PID)"

# Wait for API server to be ready
sleep 2

# Start Prometheus exporter in background
wemixvisor metrics export \
  --port $METRICS_PORT &

METRICS_PID=$!
echo "Metrics exporter started (PID: $METRICS_PID)"

# Display access URLs
echo ""
echo "✓ Wemixvisor monitoring is running!"
echo ""
echo "Access points:"
echo "  - API Health:    http://localhost:$API_PORT/health"
echo "  - API Status:    http://localhost:$API_PORT/api/v1/status"
echo "  - Metrics:       http://localhost:$API_PORT/api/v1/metrics"
echo "  - Prometheus:    http://localhost:$METRICS_PORT/metrics"
echo "  - WebSocket:     ws://localhost:$API_PORT/api/v1/ws"
echo ""
echo "PIDs:"
echo "  - API Server:    $API_PID"
echo "  - Metrics:       $METRICS_PID"
echo ""
echo "To stop: kill $API_PID $METRICS_PID"
echo ""

# Keep script running and wait for interruption
trap "echo 'Stopping...'; kill $API_PID $METRICS_PID 2>/dev/null; exit 0" INT TERM

wait
