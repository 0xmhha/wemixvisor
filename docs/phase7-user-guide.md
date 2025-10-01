# Phase 7 - Advanced Features User Guide

## Overview

Phase 7 introduces advanced monitoring, alerting, and performance optimization features to Wemixvisor, providing comprehensive observability and management capabilities for WBFT blockchain nodes.

## Features

### 1. Metrics Collection

Wemixvisor collects and exports metrics in Prometheus format for monitoring system health and performance.

#### Available Metrics

**System Metrics:**
- `wemixvisor_cpu_usage_percent` - CPU usage percentage
- `wemixvisor_memory_usage_percent` - Memory usage percentage
- `wemixvisor_disk_usage_percent` - Disk usage percentage
- `wemixvisor_network_rx_bytes_total` - Network bytes received
- `wemixvisor_network_tx_bytes_total` - Network bytes transmitted
- `wemixvisor_goroutines` - Number of active goroutines

**Application Metrics:**
- `wemixvisor_upgrades_total` - Total number of upgrades
- `wemixvisor_upgrades_success_total` - Successful upgrades
- `wemixvisor_upgrades_failed_total` - Failed upgrades
- `wemixvisor_uptime_seconds` - Process uptime
- `wemixvisor_restarts_total` - Total process restarts

**Node Metrics:**
- `wemixvisor_node_height` - Current blockchain height
- `wemixvisor_node_peers` - Number of connected peers
- `wemixvisor_node_syncing` - Node sync status (0=not syncing, 1=syncing)
- `wemixvisor_node_validator_status` - Validator status

**Governance Metrics:**
- `wemixvisor_proposals_total` - Total proposals
- `wemixvisor_proposals_voting` - Active voting proposals
- `wemixvisor_proposals_passed` - Passed proposals
- `wemixvisor_proposals_rejected` - Rejected proposals
- `wemixvisor_validators_active` - Active validators

### 2. API Server

RESTful API for managing and monitoring Wemixvisor.

#### Starting the API Server

```bash
# Enable API in configuration
cat > config.yaml <<EOF
api:
  enabled: true
  listen_address: "0.0.0.0:8080"
  enable_auth: true
  api_key: "your-secure-api-key"
  jwt_secret: "your-jwt-secret"
  rate_limit: 100
  cors_origins: ["http://localhost:3000"]
EOF

# Start wemixvisor with API enabled
wemixvisor start --config config.yaml
```

#### API Endpoints

**Status & Health:**
```bash
# Get system status
curl http://localhost:8080/api/v1/status

# Health check
curl http://localhost:8080/health
```

**Metrics:**
```bash
# Get current metrics
curl http://localhost:8080/api/v1/metrics

# Get Prometheus metrics
curl http://localhost:8080/metrics
```

**Upgrades:**
```bash
# List upgrades
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/v1/upgrades

# Get specific upgrade
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/v1/upgrades/upgrade-name

# Schedule upgrade
curl -X POST -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"name":"v1.2.0","height":1000000}' \
  http://localhost:8080/api/v1/upgrades
```

**Governance:**
```bash
# Get proposals
curl -H "X-API-Key: your-api-key" \
  http://localhost:8080/api/v1/governance/proposals

# Vote on proposal
curl -X POST -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"proposal_id":"1","vote":"yes"}' \
  http://localhost:8080/api/v1/governance/vote
```

### 3. Monitoring Dashboard

Wemixvisor includes Grafana dashboard templates for visualization.

#### Setting up Grafana

1. **Install Grafana:**
```bash
# macOS
brew install grafana

# Linux
sudo apt-get install grafana

# Start Grafana
grafana-server start
```

2. **Configure Prometheus Data Source:**
- Navigate to http://localhost:3000
- Go to Configuration → Data Sources
- Add Prometheus data source
- URL: http://localhost:9090
- Save & Test

3. **Import Dashboard:**
```bash
# Import the provided dashboard
cp dashboards/wemixvisor-overview.json /tmp/
```
- Go to Dashboards → Import
- Upload JSON file or paste content
- Select Prometheus data source
- Import

### 4. Alerting System

Configure alerts for critical events and thresholds.

#### Alert Configuration

```yaml
alerting:
  enabled: true
  evaluation_interval: 30s
  alert_retention: 24h

  channels:
    - type: slack
      name: ops-alerts
      enabled: true
      config:
        webhook_url: "https://hooks.slack.com/services/YOUR/WEBHOOK/URL"
        channel: "#alerts"

    - type: email
      name: email-alerts
      enabled: true
      config:
        smtp_host: "smtp.gmail.com"
        smtp_port: 587
        username: "alerts@example.com"
        password: "secure-password"
        from: "alerts@example.com"
        to: ["admin@example.com"]

    - type: discord
      name: discord-alerts
      enabled: true
      config:
        webhook_url: "https://discord.com/api/webhooks/YOUR/WEBHOOK"
```

#### Default Alert Rules

**System Alerts:**
- High CPU usage (>80% for 5 minutes)
- High memory usage (>90% for 5 minutes)
- Low disk space (<15% free)
- High goroutine count (>1000)

**Node Alerts:**
- Node not syncing (15 minutes)
- Low peer count (<5 peers)
- Validator offline
- Missed blocks

**Upgrade Alerts:**
- Upgrade scheduled
- Upgrade started
- Upgrade completed
- Upgrade failed

### 5. Performance Optimization

Advanced performance features for optimal operation.

#### Cache Configuration

```yaml
performance:
  enable_caching: true
  cache_size: 1000  # Maximum cache entries
  cache_ttl: 1h     # Cache time-to-live
```

#### Connection Pooling

```yaml
performance:
  enable_pooling: true
  max_connections: 100  # Maximum pool connections
  idle_timeout: 5m      # Idle connection timeout
```

#### Worker Pool

```yaml
performance:
  max_workers: 10  # Maximum concurrent workers
  task_queue_size: 1000  # Task queue capacity
```

#### Garbage Collection Tuning

```yaml
performance:
  enable_gc_tuning: true
  gc_percent: 100  # GC target percentage
```

## Usage Examples

### Complete Configuration Example

```yaml
# config.yaml
daemon_name: "wemixd"
daemon_home: "/home/wemix/.wemixd"
daemon_restart_after_upgrade: true
daemon_poll_interval: 300ms
unsafe_skip_backup: false

# API Configuration
api:
  enabled: true
  listen_address: "0.0.0.0:8080"
  enable_auth: true
  api_key: "change-me-to-secure-key"
  jwt_secret: "change-me-to-secure-secret"
  rate_limit: 100
  cors_origins: ["http://localhost:3000"]

# Metrics Configuration
metrics:
  enabled: true
  listen_address: "0.0.0.0:9100"
  collection_interval: 15s
  include_system: true
  include_application: true
  include_governance: true
  include_performance: true

# Alerting Configuration
alerting:
  enabled: true
  evaluation_interval: 30s
  alert_retention: 24h
  channels:
    - type: slack
      name: main-alerts
      enabled: true
      config:
        webhook_url: "${SLACK_WEBHOOK_URL}"
        channel: "#wemix-alerts"

# Performance Optimization
performance:
  enable_caching: true
  cache_size: 1000
  cache_ttl: 1h
  enable_pooling: true
  max_connections: 100
  max_workers: 10
  enable_gc_tuning: true
  gc_percent: 100
  enable_profiling: false
  profile_interval: 30s
```

### Docker Compose Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  wemixvisor:
    image: wemix/wemixvisor:latest
    container_name: wemixvisor
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/data
      - ./upgrades:/upgrades
    ports:
      - "8080:8080"  # API
      - "9100:9100"  # Metrics
    environment:
      - DAEMON_HOME=/data
      - SLACK_WEBHOOK_URL=${SLACK_WEBHOOK_URL}
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    ports:
      - "9090:9090"
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    volumes:
      - grafana_data:/var/lib/grafana
      - ./dashboards:/var/lib/grafana/dashboards
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    restart: unless-stopped

volumes:
  prometheus_data:
  grafana_data:
```

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['wemixvisor:9100']
        labels:
          instance: 'wemixvisor-main'
```

### Monitoring Script

```bash
#!/bin/bash
# monitor.sh

# Check wemixvisor status
check_status() {
    response=$(curl -s http://localhost:8080/api/v1/status)
    if [ $? -eq 0 ]; then
        echo "Status: OK"
        echo "$response" | jq .
    else
        echo "Status: Failed to connect"
        exit 1
    fi
}

# Check metrics
check_metrics() {
    metrics=$(curl -s http://localhost:9100/metrics | grep wemixvisor)
    echo "Current Metrics:"
    echo "$metrics" | grep -E "(cpu|memory|disk)_usage"
}

# Check alerts
check_alerts() {
    alerts=$(curl -s -H "X-API-Key: $API_KEY" \
        http://localhost:8080/api/v1/alerts)
    echo "Active Alerts:"
    echo "$alerts" | jq '.alerts[] | {name, level, message}'
}

# Main monitoring loop
while true; do
    clear
    echo "=== Wemixvisor Monitor ==="
    echo "$(date)"
    echo ""

    check_status
    echo ""

    check_metrics
    echo ""

    check_alerts

    sleep 10
done
```

## Best Practices

### 1. Security

- **Always use strong API keys** and JWT secrets
- **Enable HTTPS** in production environments
- **Restrict CORS origins** to trusted domains
- **Use rate limiting** to prevent abuse
- **Regularly rotate** credentials

### 2. Monitoring

- **Set up alerts** for critical metrics
- **Monitor trends** not just thresholds
- **Keep retention periods** appropriate
- **Test alert channels** regularly
- **Document runbooks** for alerts

### 3. Performance

- **Tune cache size** based on memory availability
- **Adjust worker pools** based on workload
- **Monitor GC metrics** and tune accordingly
- **Use connection pooling** for efficiency
- **Profile periodically** to identify bottlenecks

### 4. Maintenance

- **Regular backups** before upgrades
- **Test upgrades** in staging first
- **Monitor logs** for errors
- **Keep metrics history** for analysis
- **Update dependencies** regularly

## Troubleshooting

### Common Issues

**API Not Responding:**
```bash
# Check if API is enabled
grep "api:" config.yaml

# Check listening port
netstat -tlnp | grep 8080

# Check logs
tail -f wemixvisor.log | grep API
```

**Metrics Not Collecting:**
```bash
# Check metrics endpoint
curl http://localhost:9100/metrics

# Check Prometheus scraping
curl http://localhost:9090/api/v1/targets

# Verify metrics in Grafana
```

**Alerts Not Sending:**
```bash
# Test alert channel
curl -X POST http://localhost:8080/api/v1/alerts/test \
  -H "X-API-Key: your-api-key" \
  -H "Content-Type: application/json" \
  -d '{"channel":"slack"}'

# Check alert manager logs
grep "alert" wemixvisor.log
```

**High Memory Usage:**
```bash
# Check cache size
curl http://localhost:8080/api/v1/performance/cache/stats

# Reduce cache size in config
# Restart wemixvisor
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/wemix/wemixvisor/issues
- Documentation: https://docs.wemix.com/wemixvisor
- Community Discord: https://discord.gg/wemix