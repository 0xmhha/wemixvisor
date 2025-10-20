# Wemixvisor

Process manager for automated binary upgrades of WBFT-based blockchain nodes.

## Overview

Wemixvisor is inspired by Cosmos SDK's Cosmovisor and adapted specifically for WBFT (Wemix Byzantine Fault Tolerance) consensus-based blockchain nodes. It automates the process of upgrading blockchain node binaries with minimal downtime and manual intervention.

## Features

### Phase 1: MVP (v0.1.0) - Complete
- Automatic binary upgrade detection
- Process lifecycle management
- File-based upgrade monitoring
- Symbolic link-based version switching
- Signal handling (SIGTERM, SIGINT, SIGQUIT)
- Configurable polling intervals
- Environment variable configuration

### Phase 2: Core Features (v0.2.0) - Complete
- Data backup before upgrades
- Pre-upgrade hooks and validation
- Graceful shutdown with timeout
- Backup restoration on failure
- Custom pre-upgrade scripts
- Enhanced error handling

### Phase 3: Advanced Features (v0.3.0) - Complete
- Automatic binary downloads with SHA256/SHA512 checksum verification
- Batch upgrade support with plan management
- WBFT consensus integration for coordinated upgrades
- Validator-specific upgrade coordination
- Height-based upgrade scheduling
- Progress reporting for downloads
- Retry mechanism with exponential backoff

### Phase 4: Node Lifecycle Management (v0.4.0) - Complete
- Enhanced node process lifecycle management
- Robust start/stop/restart operations with state machine
- Graceful shutdown with configurable timeout (SIGTERM → SIGKILL)
- Auto-restart mechanism with configurable max limits
- Process group management and zombie prevention
- Real-time health monitoring and PID tracking
- Binary version detection with multiple command patterns
- Thread-safe concurrent operations
- Comprehensive error handling and recovery

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/wemix/wemixvisor.git
cd wemixvisor

# Build the binary
make build

# Install to $GOPATH/bin
make install
```

## Quick Start

1. Set up environment variables:
```bash
export DAEMON_HOME=$HOME/.wemixd
export DAEMON_NAME=wemixd
```

2. Initialize wemixvisor with your genesis binary:
```bash
wemixvisor init /path/to/wemixd
```

3. Run your node under wemixvisor management:
```bash
wemixvisor run start
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DAEMON_HOME` | `$HOME/.wemixd` | Home directory for the daemon |
| `DAEMON_NAME` | `wemixd` | Name of the daemon binary |
| `DAEMON_RESTART_AFTER_UPGRADE` | `true` | Restart after upgrade |
| `DAEMON_RESTART_DELAY` | `0` | Delay before restart |
| `DAEMON_POLL_INTERVAL` | `300ms` | Interval for checking upgrades |
| `DAEMON_SHUTDOWN_GRACE` | `30s` | Grace period for shutdown |
| `DAEMON_DATA_BACKUP_DIR` | `$DAEMON_HOME/backups` | Backup directory |
| `UNSAFE_SKIP_BACKUP` | `false` | Skip backup creation |
| `DAEMON_PREUPGRADE_MAX_RETRIES` | `0` | Pre-upgrade script retry attempts |
| `COSMOVISOR_CUSTOM_PREUPGRADE` | - | Custom pre-upgrade script path |
| `DAEMON_RPC_ADDRESS` | `localhost:8545` | RPC address for WBFT node |
| `VALIDATOR_MODE` | `false` | Enable validator-specific features |
| `DAEMON_ALLOW_DOWNLOAD_BINARIES` | `false` | Allow automatic binary downloads |
| `UNSAFE_SKIP_CHECKSUM` | `false` | Skip checksum verification for downloads |
| `DAEMON_RESTART_ON_FAILURE` | `true` | Auto-restart on process failure |
| `DAEMON_MAX_RESTARTS` | `5` | Maximum auto-restart attempts |
| `DAEMON_HEALTH_CHECK_INTERVAL` | `30s` | Health check interval |
| `DAEMON_LOG_FILE` | - | Log file path for node output |

### Directory Structure

```
$DAEMON_HOME/
├── wemixvisor/
│   ├── current/           # Symlink to active version
│   ├── genesis/           # Initial binary
│   │   └── bin/
│   │       └── wemixd
│   ├── upgrades/          # Upgrade binaries
│   │   └── v2.0.0/
│   │       ├── bin/
│   │       │   └── wemixd
│   │       └── pre-upgrade  # Optional pre-upgrade script
│   └── plans/             # Batch upgrade plans (v0.3.0+)
│       └── q4-2025-20250926-140530.json
├── data/
│   ├── upgrade-info.json  # Upgrade trigger file
│   └── upgrades/          # Height-based upgrade info (v0.3.0+)
│       └── 1000000/
│           └── upgrade-info.json
└── backups/               # Data backups (v0.2.0+)
```

### Upgrade Info Format

Create `$DAEMON_HOME/data/upgrade-info.json` to trigger an upgrade:

```json
{
  "name": "v2.0.0",
  "height": 1000000,
  "info": {
    "binaries": {
      "linux/amd64": "https://github.com/wemix/releases/...",
      "darwin/arm64": "https://github.com/wemix/releases/..."
    }
  }
}
```

## CLI Commands (Phase 4)

Wemixvisor provides a comprehensive CLI for node lifecycle management:

### Basic Commands

```bash
# Initialize wemixvisor directory structure and configuration
wemixvisor init

# Start the node (in background by default)
wemixvisor start [node-args...]

# Stop the running node
wemixvisor stop

# Restart the node with optional new arguments
wemixvisor restart [node-args...]

# Show detailed node status
wemixvisor status [--json]

# Display version information
wemixvisor version

# Run node in foreground
wemixvisor run [node-args...]
```

### Status and Health Monitoring

The `status` command provides comprehensive node health information:

```bash
# Human-readable status
wemixvisor status

# JSON output with health metrics
wemixvisor status --json
```

Sample JSON output:
```json
{
  "state": "running",
  "state_string": "running",
  "pid": 12345,
  "uptime": "2h30m45s",
  "restart_count": 0,
  "network": "mainnet",
  "binary": "/path/to/current/bin/wemixd",
  "version": "v1.2.3",
  "health": {
    "healthy": true,
    "timestamp": "2025-09-29T12:00:00Z",
    "checks": {
      "process": {"name": "process", "healthy": true, "error": ""},
      "rpc_endpoint": {"name": "rpc_endpoint", "healthy": true, "error": ""},
      "memory": {"name": "memory", "healthy": true, "error": ""}
    }
  }
}
```

### Health Monitoring Features

- **Process Health**: Monitors node process liveness
- **RPC Endpoint**: Checks JSON-RPC connectivity
- **Memory Usage**: Tracks memory consumption
- **Disk Space**: Validates available storage
- **Network Connectivity**: Verifies peer connections
- **Sync Status**: Monitors blockchain synchronization

### Metrics Collection

Wemixvisor automatically collects and provides metrics:

- Node uptime and restart count
- Memory usage in megabytes
- Health status and check results
- JSON and Prometheus export formats

## Development

### Building

```bash
# Build binary
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make lint
```

### Testing

#### Running All Tests

```bash
# Run all tests with verbose output
go test -v ./...

# Run tests with coverage report
go test -v -cover ./...

# Generate coverage report with HTML output
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run tests with race detection
go test -race ./...
```

#### Running Specific Test Categories

```bash
# Unit tests only (fast)
go test -short ./...

# Integration tests
go test -v ./internal/... ./pkg/...

# E2E tests (requires build tag)
go test -tags=e2e -v ./test/e2e

# Benchmark tests
go test -bench=. -benchmem ./...

# Specific package tests
go test -v ./internal/monitor
go test -v ./internal/metrics
go test -v ./internal/cli
```

#### Test Coverage by Package

```bash
# Check coverage for specific packages
go test -cover ./internal/monitor
go test -cover ./internal/metrics
go test -cover ./internal/cli
go test -cover ./internal/node

# Detailed coverage breakdown
go test -coverprofile=coverage.out -covermode=atomic ./...
go tool cover -func=coverage.out
```

#### Testing Phase 4 Features

```bash
# Test health monitoring
go test -v ./internal/monitor/...

# Test metrics collection
go test -v ./internal/metrics/...

# Test CLI commands
go test -v ./internal/cli/...

# Test node management
go test -v ./internal/node/...

# Run Phase 4 E2E tests
go test -tags=e2e -v ./test/e2e -run TestPhase4
```

#### Performance Testing

```bash
# Run all benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark tests
go test -bench=BenchmarkHealthChecker ./internal/monitor
go test -bench=BenchmarkMetricsCollector ./internal/metrics
go test -bench=BenchmarkParser ./internal/cli

# Run benchmarks with CPU profiling
go test -bench=. -cpuprofile=cpu.prof ./internal/monitor
go tool pprof cpu.prof

# Run benchmarks with memory profiling
go test -bench=. -memprofile=mem.prof ./internal/monitor
go tool pprof mem.prof
```

#### Test Environment Setup

```bash
# Set test environment variables
export DAEMON_HOME=$HOME/.wemixd-test
export DAEMON_NAME=wemixd
export DAEMON_HEALTH_CHECK_INTERVAL=1s
export DAEMON_METRICS_INTERVAL=5s

# Clean test artifacts
rm -rf $DAEMON_HOME
rm -f coverage.out coverage.html
rm -f *.prof
```

#### Continuous Integration Testing

```bash
# Run full CI test suite
make ci-test

# Or manually:
go fmt ./...
go vet ./...
golangci-lint run
go test -race -coverprofile=coverage.out ./...
go test -tags=e2e ./test/e2e
```

#### Troubleshooting Tests

If tests fail or hang:

1. **Check test timeouts**: Some tests may need longer timeouts
   ```bash
   go test -timeout 30s ./...
   ```

2. **Run tests sequentially**: Avoid parallel test conflicts
   ```bash
   go test -p 1 ./...
   ```

3. **Enable verbose logging**: See detailed test output
   ```bash
   go test -v -count=1 ./...
   ```

4. **Clean test cache**: Force fresh test runs
   ```bash
   go clean -testcache
   go test ./...
   ```

### Project Structure

```
wemixvisor/
├── cmd/              # CLI commands
├── internal/         # Private packages
│   ├── backup/       # Data backup functionality
│   ├── batch/        # Batch upgrade management
│   ├── commands/     # Command implementations
│   ├── config/       # Configuration management
│   ├── download/     # Automatic binary downloads
│   ├── hooks/        # Pre-upgrade hooks
│   ├── node/         # Node lifecycle management (Phase 4)
│   ├── process/      # Process management
│   ├── upgrade/      # Upgrade handling
│   └── wbft/         # WBFT consensus integration
├── pkg/              # Public packages
│   ├── logger/       # Logging utilities
│   └── types/        # Common types
├── docs/             # Documentation
└── test/             # Integration tests
```

## Documentation

### Implementation Guides
- [Phase 4: Node Lifecycle](./docs/phase4-detailed-implementation.md) - Detailed implementation guide
- [Phase 7: Advanced Features](./docs/phase7-advanced-features.md) - Metrics, API, optimization
- [Phase 7 User Guide](./docs/phase7-user-guide.md) - Using advanced features

### Feature Documentation
- [Metrics & Monitoring](./docs/metrics.md) - Metrics collection and Prometheus integration
- [Alerting System](./docs/alerting.md) - Alert rules and notification channels
- [Grafana Dashboards](./docs/grafana.md) - Dashboard setup and configuration
- [Testing Guide](./docs/testing.md) - Testing strategy and coverage

### API References
- [Governance API](./docs/governance-api.md) - Governance integration API
- [Governance Overview](./docs/governance.md) - Governance system overview

### Project Management
- [Roadmap](./docs/ROADMAP.md) - Development roadmap and milestones
- [Changes Log](./CHANGES.md) - Version history and release notes

## Development Status

- ✅ Phase 1: Basic process management (v0.1.0) - Complete
- ✅ Phase 2: Core features (v0.2.0) - Complete
- ✅ Phase 3: Advanced features & WBFT integration (v0.3.0) - Complete
- ✅ Phase 4: Node lifecycle management (v0.4.0) - Complete
  - Enhanced process lifecycle with state machine
  - Auto-restart with configurable limits
  - Health monitoring and version detection
  - 91.2% test coverage achieved
- ✅ Phase 5: Configuration management system (v0.5.0) - Complete
- ✅ Phase 6: Governance integration (v0.6.0) - Complete
- ✅ Phase 7: Advanced features & optimization (v0.7.0) - Complete
  - Metrics collection and Prometheus exporter
  - RESTful API server with WebSocket support
  - Alerting system with multiple notification channels
  - Performance profiling and optimization tools

## Contributing

Contributions are welcome! Please ensure your code follows Go best practices and includes appropriate tests.

## License

Apache License 2.0
