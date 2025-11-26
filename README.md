# Wemixvisor

Process manager for automated binary upgrades of WBFT-based blockchain nodes.

## Overview

Wemixvisor is inspired by Cosmos SDK's Cosmovisor and adapted specifically for WBFT (Wemix Byzantine Fault Tolerance) consensus-based blockchain nodes. It automates the process of upgrading blockchain node binaries with minimal downtime and manual intervention.

## Features

- **Automatic Upgrade Detection** - Monitors for upgrade triggers and executes seamlessly
- **Height-Based Upgrade Scheduling** - Schedule upgrades at specific blockchain heights
- **Zero Manual Intervention** - Automatic execution when blockchain reaches target height
- **Automatic Binary Downloads** - Download binaries with SHA256/SHA512 checksum verification
- **Data Backup & Rollback** - Automatic backup before upgrades with rollback on failure
- **Process Lifecycle Management** - Robust start/stop/restart with auto-restart capability
- **Health Monitoring** - Real-time health checks and metrics collection
- **Graceful Shutdown** - Configurable timeout with SIGTERM → SIGKILL escalation
- **WBFT Consensus Integration** - Coordinated upgrades across validator network

## Installation

### From Source

```bash
git clone https://github.com/wemix/wemixvisor.git
cd wemixvisor
make build
make install
```

## Development

### Requirements

- Go 1.23 or higher
- Make
- (Optional) golangci-lint for linting
- (Optional) Docker for container builds

### Build Commands

```bash
# Build for current platform
make build

# Build for all platforms (Linux, macOS, Windows)
make build-all

# Build for specific platform
make build-linux
make build-darwin
make build-windows

# Install to $GOPATH/bin
make install

# Clean build artifacts
make clean
```

### Test Commands

```bash
# Run unit tests (default)
make test

# Run unit tests with race detector
make test-unit

# Run integration tests
make test-integration

# Run end-to-end tests
make test-e2e

# Run all tests
make test-all

# Run tests with verbose output
make test-verbose
```

### Code Coverage

```bash
# Generate coverage report
make coverage

# Generate HTML coverage report
make coverage-html

# Show function-level coverage
make coverage-func
```

### Code Quality

```bash
# Format code
make fmt

# Run go vet
make vet

# Run linter (requires golangci-lint)
make lint

# Run all quality checks
make check
```

### Dependencies

```bash
# Download dependencies
make deps

# Update dependencies
make deps-update

# Tidy dependencies
make deps-tidy
```

### Docker

```bash
# Build Docker image
make docker-build

# Push Docker image
make docker-push
```

### Release

```bash
# Prepare release (clean, check, test, build all platforms)
make release

# Generate release notes
make release-notes
```

### Useful Commands

```bash
# Show all available make targets
make help

# Show version and build information
make version
make info

# Build and run help (development mode)
make dev
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

## Upgrade Management

### Schedule an Upgrade

Schedule an upgrade to execute automatically at a specific block height:

```bash
# Basic upgrade scheduling
wemixvisor upgrade schedule v1.2.0 1000000

# With additional metadata
wemixvisor upgrade schedule v1.2.0 1000000 \
  --checksum abc123... \
  --info "Major protocol upgrade"
```

### Check Upgrade Status

```bash
wemixvisor upgrade status
```

### Cancel Scheduled Upgrade

```bash
wemixvisor upgrade cancel

# Skip confirmation
wemixvisor upgrade cancel --force
```

### How It Works

1. **Height Monitoring** - Continuously monitors blockchain height via RPC
2. **Automatic Trigger** - Executes upgrade when blockchain reaches scheduled height
3. **Safe Execution** - Node stops gracefully → binary switches → node restarts
4. **Rollback** - Automatic rollback to previous binary if upgrade fails

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
| `UNSAFE_SKIP_CHECKSUM` | `false` | Skip checksum verification |
| `DAEMON_RESTART_ON_FAILURE` | `true` | Auto-restart on process failure |
| `DAEMON_MAX_RESTARTS` | `5` | Maximum auto-restart attempts |
| `DAEMON_HEALTH_CHECK_INTERVAL` | `30s` | Health check interval |
| `DAEMON_LOG_FILE` | - | Log file path for node output |
| `DAEMON_UPGRADE_ENABLED` | `true` | Enable automatic upgrade monitoring |
| `DAEMON_HEIGHT_POLL_INTERVAL` | `5s` | Blockchain height polling interval |

### Directory Structure

```
$DAEMON_HOME/
├── wemixvisor/
│   ├── current/           # Symlink to active version
│   ├── genesis/           # Initial binary
│   │   └── bin/
│   │       └── wemixd
│   └── upgrades/          # Upgrade binaries
│       └── v2.0.0/
│           ├── bin/
│           │   └── wemixd
│           └── pre-upgrade  # Optional pre-upgrade script
├── data/
│   └── upgrade-info.json  # Upgrade trigger file
└── backups/               # Data backups
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

## CLI Commands

### Node Management

```bash
# Initialize directory structure
wemixvisor init

# Start the node (background)
wemixvisor start [node-args...]

# Stop the running node
wemixvisor stop

# Restart the node
wemixvisor restart [node-args...]

# Run node in foreground
wemixvisor run [node-args...]
```

### Status & Monitoring

```bash
# Human-readable status
wemixvisor status

# JSON output
wemixvisor status --json

# Version information
wemixvisor version
```

### Sample Status Output

```json
{
  "state": "running",
  "pid": 12345,
  "uptime": "2h30m45s",
  "restart_count": 0,
  "binary": "/path/to/current/bin/wemixd",
  "version": "v1.2.3",
  "health": {
    "healthy": true,
    "checks": {
      "process": {"healthy": true},
      "rpc_endpoint": {"healthy": true},
      "memory": {"healthy": true}
    }
  }
}
```

## Contributing

Contributions are welcome! Please ensure your code follows Go best practices and includes appropriate tests.

## License

Apache License 2.0
