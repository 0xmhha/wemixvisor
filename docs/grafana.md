# Grafana Dashboard Setup Guide

Complete guide to setting up Grafana dashboards for wemixvisor monitoring.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Data Source Configuration](#data-source-configuration)
- [Dashboard Import](#dashboard-import)
- [Dashboard Customization](#dashboard-customization)
- [Alert Configuration](#alert-configuration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

Wemixvisor provides pre-built Grafana dashboards for comprehensive monitoring:

- **System Overview**: Resource metrics, uptime, and system health
- **Upgrades**: Upgrade tracking, success rates, and backup status
- **Governance**: Proposal monitoring, voting metrics, and validator stats

### Architecture

```
┌─────────────────┐
│  Wemixvisor     │
│  Prometheus     │
│  Exporter       │
│  :9090/metrics  │
└────────┬────────┘
         │ HTTP Scrape
         v
┌─────────────────┐
│  Prometheus     │
│  Server         │
│  :9090          │
└────────┬────────┘
         │ PromQL Queries
         v
┌─────────────────┐
│  Grafana        │
│  Server         │
│  :3000          │
└─────────────────┘
```

## Prerequisites

### Required Components

1. **Wemixvisor** with metrics enabled
   ```toml
   [monitoring]
   enable_metrics = true
   enable_prometheus = true
   prometheus_port = 9090
   ```

2. **Prometheus** for metrics collection
3. **Grafana** for visualization

### Recommended Versions

- Wemixvisor: v0.7.0+
- Prometheus: v2.40.0+
- Grafana: v9.0.0+

## Installation

### Option 1: Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  wemixvisor:
    image: wemix/wemixvisor:latest
    ports:
      - "8080:8080"  # API server
      - "9090:9090"  # Prometheus exporter
    volumes:
      - ./config:/config
      - ./data:/data
    command: api --enable-metrics --enable-prometheus

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"  # Different port to avoid conflict
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=30d'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    depends_on:
      - prometheus

volumes:
  prometheus-data:
  grafana-data:
```

Create `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['wemixvisor:9090']
    scrape_interval: 10s
```

Start services:

```bash
docker-compose up -d
```

### Option 2: Manual Installation

#### Install Prometheus

**Linux**:
```bash
# Download
wget https://github.com/prometheus/prometheus/releases/download/v2.40.0/prometheus-2.40.0.linux-amd64.tar.gz

# Extract
tar xvf prometheus-2.40.0.linux-amd64.tar.gz
cd prometheus-2.40.0.linux-amd64

# Configure
cat > prometheus.yml <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['localhost:9090']
EOF

# Run
./prometheus --config.file=prometheus.yml
```

**macOS**:
```bash
# Using Homebrew
brew install prometheus

# Configure
cat > /usr/local/etc/prometheus.yml <<EOF
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'wemixvisor'
    static_configs:
      - targets: ['localhost:9090']
EOF

# Run
brew services start prometheus
```

#### Install Grafana

**Linux (Ubuntu/Debian)**:
```bash
# Add GPG key
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -

# Add repository
echo "deb https://packages.grafana.com/oss/deb stable main" | sudo tee /etc/apt/sources.list.d/grafana.list

# Install
sudo apt-get update
sudo apt-get install grafana

# Start
sudo systemctl start grafana-server
sudo systemctl enable grafana-server
```

**macOS**:
```bash
# Using Homebrew
brew install grafana

# Start
brew services start grafana
```

**Access Grafana**:
- URL: http://localhost:3000
- Default credentials: admin/admin
- Change password on first login

## Data Source Configuration

### Add Prometheus Data Source

1. **Navigate to Data Sources**:
   - Grafana UI → Configuration (⚙️) → Data Sources
   - Click "Add data source"
   - Select "Prometheus"

2. **Configure Connection**:
   ```
   Name: Prometheus
   URL: http://localhost:9091 (or your Prometheus address)
   Access: Server (default)
   ```

3. **Test Connection**:
   - Click "Save & Test"
   - Should see: "Data source is working"

### Configuration via API

```bash
# Create Prometheus data source
curl -X POST http://admin:admin@localhost:3000/api/datasources \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Prometheus",
    "type": "prometheus",
    "url": "http://localhost:9091",
    "access": "proxy",
    "isDefault": true
  }'
```

### Configuration via Provisioning

Create `grafana/provisioning/datasources/prometheus.yml`:

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

## Dashboard Import

### Import from UI

1. **Navigate to Import**:
   - Grafana UI → Dashboards (➕) → Import

2. **Upload Dashboard**:
   - Click "Upload JSON file"
   - Select dashboard file:
     - `examples/grafana/overview-dashboard.json`
     - `examples/grafana/upgrades-dashboard.json`
     - `examples/grafana/governance-dashboard.json`

3. **Configure Import**:
   - Select "Prometheus" as data source
   - Click "Import"

4. **Verify Dashboard**:
   - Dashboard should load with live data
   - Panels should show metrics

### Import via API

```bash
# Import System Overview dashboard
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @examples/grafana/overview-dashboard.json

# Import Upgrades dashboard
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @examples/grafana/upgrades-dashboard.json

# Import Governance dashboard
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @examples/grafana/governance-dashboard.json
```

### Import via Provisioning

Create `grafana/provisioning/dashboards/wemixvisor.yml`:

```yaml
apiVersion: 1

providers:
  - name: 'Wemixvisor'
    orgId: 1
    folder: 'Wemixvisor'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

Copy dashboard files:

```bash
cp examples/grafana/*.json grafana/provisioning/dashboards/
```

Restart Grafana:

```bash
# Docker
docker-compose restart grafana

# Systemd
sudo systemctl restart grafana-server
```

## Dashboard Customization

### System Overview Dashboard

**Available Panels**:
1. **Uptime** - System uptime in seconds
2. **CPU Usage** - CPU usage percentage gauge
3. **Memory Usage** - Memory usage percentage gauge
4. **Disk Usage** - Disk usage percentage gauge
5. **Resource Usage Over Time** - CPU/Memory/Disk trend lines
6. **Goroutines** - Active goroutine count
7. **Network Traffic** - RX/TX bytes
8. **Active Alerts** - Current alert table

**Customization Options**:

```json
{
  "refresh": "10s",  // Change refresh interval
  "time": {
    "from": "now-1h",  // Change time range
    "to": "now"
  }
}
```

**Example Modifications**:

Change CPU threshold colors:
```json
{
  "thresholds": {
    "steps": [
      {"color": "green", "value": null},
      {"color": "yellow", "value": 60},  // Changed from 70
      {"color": "red", "value": 80}      // Changed from 85
    ]
  }
}
```

Add custom panel:
```json
{
  "title": "Custom Metric",
  "targets": [{
    "expr": "wemixvisor_custom_metric",
    "refId": "A"
  }]
}
```

### Upgrades Dashboard

**Available Panels**:
1. **Total Upgrades** - Upgrade count stat
2. **Successful Upgrades** - Success count
3. **Failed Upgrades** - Failure count
4. **Success Rate** - Percentage gauge
5. **Upgrade History** - Success/failure bars
6. **Upgrade Duration** - Duration trend
7. **Backup Stats** - Backup success metrics
8. **Hook Execution** - Pre/post upgrade hooks

**Custom Queries**:

Average upgrade duration:
```promql
avg_over_time(wemixvisor_upgrade_duration_seconds[24h])
```

Upgrade success rate (7 days):
```promql
100 * increase(wemixvisor_upgrades_success[7d]) / increase(wemixvisor_upgrades_total[7d])
```

### Governance Dashboard

**Available Panels**:
1. **Total Proposals** - Proposal count
2. **Active Voting** - Proposals in voting period
3. **Passed/Rejected** - Outcome stats
4. **Active Proposals Table** - Detailed proposal list
5. **Proposal Status Distribution** - Pie chart
6. **Voting Distribution** - Yes/No/Abstain/Veto
7. **Voter Participation** - Participation rate
8. **Validator Stats** - Validator metrics

**Custom Queries**:

Proposal pass rate:
```promql
100 * wemixvisor_governance_proposals_passed / wemixvisor_governance_proposals_total
```

Average voting participation:
```promql
avg_over_time(wemixvisor_governance_voter_participation_percent[30d])
```

### Advanced Customization

**Variables**:

Add dashboard variable for node selection:
```json
{
  "templating": {
    "list": [{
      "name": "node",
      "type": "query",
      "datasource": "Prometheus",
      "query": "label_values(wemixvisor_system_uptime_seconds, instance)",
      "refresh": 1
    }]
  }
}
```

Use variable in queries:
```promql
wemixvisor_system_cpu_usage_percent{instance="$node"}
```

**Annotations**:

Add upgrade events as annotations:
```json
{
  "annotations": {
    "list": [{
      "datasource": "Prometheus",
      "enable": true,
      "expr": "changes(wemixvisor_upgrades_total[1m]) > 0",
      "name": "Upgrades",
      "tagKeys": "upgrade",
      "textFormat": "Upgrade completed",
      "titleFormat": "Upgrade Event"
    }]
  }
}
```

**Links**:

Add navigation links between dashboards:
```json
{
  "links": [
    {
      "title": "System Overview",
      "type": "dashboards",
      "uid": "wemixvisor-overview"
    },
    {
      "title": "Upgrades",
      "type": "dashboards",
      "uid": "wemixvisor-upgrades"
    }
  ]
}
```

## Alert Configuration

### Grafana Alerts

**Create Alert Rule**:

1. **Edit Panel** → Alert tab
2. **Configure Conditions**:
   ```
   WHEN avg() OF query(A, 5m, now) IS ABOVE 80
   ```

3. **Set Notification**:
   - Name: High CPU Usage
   - Evaluate every: 1m
   - For: 5m

**Example Alert Rules**:

High CPU Usage:
```json
{
  "alert": {
    "name": "High CPU Usage",
    "conditions": [{
      "evaluator": {
        "params": [80],
        "type": "gt"
      },
      "query": {
        "model": {
          "expr": "wemixvisor_system_cpu_usage_percent"
        }
      }
    }],
    "for": "5m",
    "frequency": "1m"
  }
}
```

Low Disk Space:
```json
{
  "alert": {
    "name": "Low Disk Space",
    "conditions": [{
      "evaluator": {
        "params": [90],
        "type": "gt"
      },
      "query": {
        "model": {
          "expr": "wemixvisor_system_disk_usage_percent"
        }
      }
    }],
    "for": "1m",
    "frequency": "1m"
  }
}
```

### Notification Channels

**Email**:
```json
{
  "name": "Email Alerts",
  "type": "email",
  "settings": {
    "addresses": "admin@example.com;oncall@example.com"
  }
}
```

**Slack**:
```json
{
  "name": "Slack Alerts",
  "type": "slack",
  "settings": {
    "url": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "recipient": "#alerts",
    "username": "Grafana"
  }
}
```

**PagerDuty**:
```json
{
  "name": "PagerDuty",
  "type": "pagerduty",
  "settings": {
    "integrationKey": "YOUR_INTEGRATION_KEY",
    "severity": "critical"
  }
}
```

## Best Practices

### Performance Optimization

**Query Optimization**:
- Use recording rules for expensive queries
- Limit time range for heavy queries
- Use appropriate step intervals

**Example Recording Rules** (`prometheus.yml`):
```yaml
groups:
  - name: wemixvisor
    interval: 30s
    rules:
      - record: job:wemixvisor_cpu_usage:avg
        expr: avg(wemixvisor_system_cpu_usage_percent)

      - record: job:wemixvisor_memory_usage:avg
        expr: avg(wemixvisor_system_memory_usage_percent)
```

**Dashboard Settings**:
```json
{
  "refresh": "10s",          // Reasonable refresh rate
  "time": {"from": "now-1h"}, // Appropriate time range
  "timezone": "browser"       // Use browser timezone
}
```

### Layout Best Practices

**Panel Organization**:
1. **Top Row**: Key metrics (stats/gauges)
2. **Middle Rows**: Trends (graphs/time series)
3. **Bottom Rows**: Details (tables/lists)

**Panel Sizing**:
- Stats: 4-6 grid units wide
- Gauges: 4-6 grid units wide
- Graphs: 12-24 grid units wide
- Tables: 12-24 grid units wide

**Color Schemes**:
- **Green**: Healthy/Normal (0-70%)
- **Yellow**: Warning (70-85%)
- **Red**: Critical (85-100%)

### Monitoring Coverage

**Essential Dashboards**:
1. System Overview (always visible)
2. Upgrades (operational events)
3. Governance (blockchain-specific)

**Optional Dashboards**:
- Performance profiling
- API metrics
- Custom application metrics

### Access Control

**User Roles**:
```bash
# Create viewer user (read-only)
curl -X POST http://admin:admin@localhost:3000/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Viewer",
    "email": "viewer@example.com",
    "login": "viewer",
    "password": "secure-password",
    "role": "Viewer"
  }'

# Create editor user
curl -X POST http://admin:admin@localhost:3000/api/admin/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Editor",
    "email": "editor@example.com",
    "login": "editor",
    "password": "secure-password",
    "role": "Editor"
  }'
```

**Folder Permissions**:
1. Create "Wemixvisor" folder
2. Set permissions:
   - Viewer: View only
   - Editor: Edit dashboards
   - Admin: Full access

### Backup and Version Control

**Export Dashboards**:
```bash
# Export all dashboards
for uid in $(curl -s http://admin:admin@localhost:3000/api/search | jq -r '.[].uid'); do
  curl -s http://admin:admin@localhost:3000/api/dashboards/uid/$uid | \
    jq '.dashboard' > dashboard-$uid.json
done
```

**Version Control**:
```bash
# Store in Git
git add examples/grafana/*.json
git commit -m "Update Grafana dashboards"
```

**Automated Backup**:
```bash
#!/bin/bash
# backup-grafana.sh
BACKUP_DIR="/backups/grafana/$(date +%Y%m%d)"
mkdir -p "$BACKUP_DIR"

# Export dashboards
for uid in $(curl -s http://admin:admin@localhost:3000/api/search | jq -r '.[].uid'); do
  curl -s http://admin:admin@localhost:3000/api/dashboards/uid/$uid | \
    jq '.dashboard' > "$BACKUP_DIR/dashboard-$uid.json"
done

# Compress
tar czf "$BACKUP_DIR.tar.gz" "$BACKUP_DIR"
rm -rf "$BACKUP_DIR"
```

## Troubleshooting

### Dashboard Not Loading

**Check Prometheus Connection**:
```bash
# Test Prometheus API
curl http://localhost:9091/api/v1/query?query=up

# Check Grafana data source
curl http://admin:admin@localhost:3000/api/datasources
```

**Verify Metrics**:
```bash
# Check if wemixvisor metrics are available
curl http://localhost:9091/api/v1/query?query=wemixvisor_system_uptime_seconds
```

**Check Logs**:
```bash
# Grafana logs
tail -f /var/log/grafana/grafana.log

# Docker logs
docker-compose logs -f grafana
```

### No Data in Panels

**Verify Metric Collection**:
```bash
# Check wemixvisor metrics endpoint
curl http://localhost:9090/metrics | grep wemixvisor

# Verify Prometheus is scraping
curl http://localhost:9091/api/v1/targets
```

**Check Time Range**:
- Ensure time range includes data
- Try "Last 5 minutes" or "Last 15 minutes"

**Validate Queries**:
- Use Prometheus UI to test queries
- Check for typos in metric names
- Verify label filters

### Permission Issues

**Fix Data Source Permissions**:
```bash
# Grant access to data source
curl -X POST http://admin:admin@localhost:3000/api/datasources/1/permissions \
  -H "Content-Type: application/json" \
  -d '{
    "teamId": 0,
    "userId": 0,
    "permission": 1
  }'
```

**Reset Admin Password**:
```bash
# Using grafana-cli
grafana-cli admin reset-admin-password newpassword

# Docker
docker exec -it grafana grafana-cli admin reset-admin-password newpassword
```

### Performance Issues

**Reduce Query Load**:
- Increase refresh interval
- Limit time range
- Use recording rules

**Optimize Prometheus**:
```yaml
global:
  scrape_interval: 15s  # Increase if needed

scrape_configs:
  - job_name: 'wemixvisor'
    scrape_interval: 30s  # Less frequent scraping
```

**Enable Query Caching**:
```ini
# grafana.ini
[caching]
enabled = true
ttl = 60
```

## Production Deployment

### Recommended Configuration

**Docker Compose** (production):
```yaml
version: '3.8'

services:
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=90d'
      - '--storage.tsdb.retention.size=50GB'
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    volumes:
      - grafana-data:/var/lib/grafana
      - ./grafana/provisioning:/etc/grafana/provisioning
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_ADMIN_PASSWORD}
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_AUTH_ANONYMOUS_ENABLED=false
      - GF_LOG_LEVEL=info
    restart: unless-stopped

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/nginx/ssl
    depends_on:
      - grafana
    restart: unless-stopped

volumes:
  prometheus-data:
  grafana-data:
```

**Nginx Configuration** (reverse proxy with TLS):
```nginx
server {
    listen 443 ssl http2;
    server_name grafana.example.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;

    location / {
        proxy_pass http://grafana:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### High Availability

**Multiple Prometheus Instances**:
```yaml
scrape_configs:
  - job_name: 'wemixvisor-primary'
    static_configs:
      - targets: ['wemixvisor-1:9090']

  - job_name: 'wemixvisor-secondary'
    static_configs:
      - targets: ['wemixvisor-2:9090']
```

**Grafana Clustering**:
- Use external database (MySQL/PostgreSQL)
- Share storage for dashboards
- Load balance with Nginx

## See Also

- [Metrics Documentation](metrics.md)
- [Alerting Configuration](alerting.md)
- [API Documentation](api.md)
- [Configuration Reference](configuration.md)
