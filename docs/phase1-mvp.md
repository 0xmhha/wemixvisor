# Phase 1: MVP Implementation

## Overview
Phase 1 implements the minimal viable product (MVP) for Wemixvisor, focusing on basic process management, file-based upgrade detection, and symbolic link management.

## Implemented Features

### 1. Core Process Management
- **Process Launcher**: Starts and manages the blockchain node process
- **Signal Handling**: Proper handling of SIGTERM, SIGINT, and SIGQUIT
- **Process Monitoring**: Continuous monitoring of process health
- **Automatic Restart**: Configurable restart after process exit

### 2. File-based Upgrade Detection
- **FileWatcher**: Monitors `upgrade-info.json` for changes
- **Polling Mechanism**: Configurable polling interval (default: 300ms)
- **Upgrade Info Parsing**: Validates upgrade name and height
- **Modification Detection**: Tracks file modification time

### 3. Symbolic Link Management
- **Current Link**: Points to the active binary version
- **Genesis Link**: Initial binary setup
- **Upgrade Links**: Manages links to upgrade directories
- **Atomic Switching**: Safe binary version switching

### 4. Configuration Management
- **Environment Variables**: Full support for DAEMON_* variables
- **Command Line Flags**: Override environment variables
- **Configuration Validation**: Ensures required settings are present
- **Default Values**: Sensible defaults for all settings

### 5. Directory Structure
```
$DAEMON_HOME/
├── wemixvisor/
│   ├── current/        # Symlink to active version
│   ├── genesis/        # Initial binary
│   │   └── bin/
│   │       └── wemixd
│   └── upgrades/       # Upgrade binaries
│       └── v2.0.0/
│           └── bin/
│               └── wemixd
├── data/
│   └── upgrade-info.json
└── backups/            # Data backups (Phase 2)
```

## Command Line Interface

### Initialize
```bash
wemixvisor init /path/to/wemixd
```
Sets up directory structure and copies genesis binary.

### Run
```bash
wemixvisor run start [flags]
```
Starts the managed process with upgrade monitoring.

### Version
```bash
wemixvisor version
```
Displays version information.

## Configuration Options

### Environment Variables
- `DAEMON_HOME`: Home directory for the daemon
- `DAEMON_NAME`: Binary name (default: wemixd)
- `DAEMON_ALLOW_DOWNLOAD_BINARIES`: Enable auto-download
- `DAEMON_RESTART_AFTER_UPGRADE`: Auto-restart after upgrade
- `DAEMON_SHUTDOWN_GRACE`: Grace period for shutdown
- `DAEMON_POLL_INTERVAL`: Upgrade check interval

### Upgrade Info Format
```json
{
  "name": "v2.0.0",
  "height": 1000000,
  "info": {
    "binaries": {
      "linux/amd64": "https://...",
      "darwin/arm64": "https://..."
    }
  }
}
```

## Architecture

### Component Interaction
```
┌─────────────────────────────────────┐
│         Process Manager             │
│                                     │
│  ┌─────────────────────────────┐    │
│  │     Process Launcher        │    │
│  └─────────────────────────────┘    │
│                 │                   │
│  ┌──────────────┴──────────────┐    │
│  │                             │    │
│  ▼                             ▼    │
│ ┌──────────────┐    ┌──────────────┐│
│ │File Watcher  │    │Signal Handler││
│ └──────────────┘    └──────────────┘│
│                                     │
│  ┌─────────────────────────────┐    │
│  │   Child Process (wemixd)    │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

### Upgrade Flow
1. FileWatcher detects `upgrade-info.json` change
2. Process Manager receives upgrade signal
3. Graceful shutdown of current process
4. Symbolic link updated to new version
5. New process started with upgraded binary

## Testing

### Manual Testing Steps
1. Initialize wemixvisor with a test binary
2. Start the process with `wemixvisor run`
3. Create an upgrade directory with new binary
4. Write upgrade-info.json with upgrade details
5. Verify automatic upgrade occurs

### Test Coverage Areas
- Configuration loading and validation
- Process lifecycle management
- Signal handling
- File watching and parsing
- Symbolic link operations

## Known Limitations (To be addressed in Phase 2+)
- No data backup before upgrade
- No pre-upgrade hooks
- No WBFT consensus integration
- No automatic binary download
- No batch upgrade support

## Next Steps (Phase 2)
- Implement data backup functionality
- Add pre-upgrade hook system
- Enhance graceful shutdown mechanism
- Add custom pre-upgrade scripts
- Improve error handling and recovery