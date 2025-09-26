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

- [Phase 1 Documentation](./docs/phase1-mvp.md) - MVP implementation details
- [Phase 3 Documentation](./docs/phase3-advanced-features.md) - Advanced features guide
- [Changes Log](./CHANGES.md) - Version history

## Development Status

- ✅ Phase 1: Basic process management (v0.1.0) - Complete
- ✅ Phase 2: Core features (v0.2.0) - Complete
- ✅ Phase 3: Advanced features & WBFT integration (v0.3.0) - Complete

## Contributing

Please read [CLAUDE.md](./CLAUDE.md) for development guidelines.

## License

Apache License 2.0
