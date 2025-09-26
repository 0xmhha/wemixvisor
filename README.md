# Wemixvisor

Process manager for automated binary upgrades of WBFT-based blockchain nodes.

## Overview

Wemixvisor is inspired by Cosmos SDK's Cosmovisor and adapted specifically for WBFT (Wemix Byzantine Fault Tolerance) consensus-based blockchain nodes. It automates the process of upgrading blockchain node binaries with minimal downtime and manual intervention.

## Features

### Current (v0.1.0)
- ✅ Automatic binary upgrade detection
- ✅ Process lifecycle management
- ✅ File-based upgrade monitoring
- ✅ Symbolic link-based version switching
- ✅ Signal handling (SIGTERM, SIGINT, SIGQUIT)
- ✅ Configurable polling intervals
- ✅ Environment variable configuration

### Planned
- Data backup before upgrades (v0.2.0)
- Pre-upgrade hooks and validation (v0.2.0)
- Graceful shutdown with timeout (v0.2.0)
- WBFT consensus state monitoring (v0.3.0)
- Automatic binary downloads (v0.3.0)
- Batch upgrade support (v0.3.0)

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
| `DAEMON_POLL_INTERVAL` | `300ms` | Interval for checking upgrades |
| `DAEMON_SHUTDOWN_GRACE` | `0` | Grace period for shutdown |
| `DAEMON_RPC_ADDRESS` | `localhost:8545` | RPC address for WBFT node |

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
│           └── bin/
│               └── wemixd
├── data/
│   └── upgrade-info.json  # Upgrade trigger file
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
│   ├── commands/     # Command implementations
│   ├── config/       # Configuration management
│   ├── process/      # Process management
│   └── upgrade/      # Upgrade handling
├── pkg/              # Public packages
│   ├── logger/       # Logging utilities
│   └── types/        # Common types
├── docs/             # Documentation
└── test/             # Integration tests
```

## Documentation

- [Phase 1 Documentation](./docs/phase1-mvp.md) - MVP implementation details
- [Changes Log](./CHANGES.md) - Version history

## Development Status

- ✅ Phase 1: Basic process management (v0.1.0)
- 🚧 Phase 2: Core features (v0.2.0)
- 📋 Phase 3: WBFT integration (v0.3.0)

## Contributing

Please read [CLAUDE.md](./CLAUDE.md) for development guidelines.

## License

Apache License 2.0
