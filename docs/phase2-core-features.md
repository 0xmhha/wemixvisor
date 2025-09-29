# Phase 2: Core Features

## Overview

Phase 2 extends Wemixvisor with essential production features including data backup, pre-upgrade hooks, and graceful shutdown capabilities.

## Features Implemented

### 1. Backup Functionality

The backup system provides automatic data protection before upgrades.

#### Features
- **Automatic Backup**: Creates tar.gz archives of data directory before upgrades
- **Backup Management**: List, restore, and clean old backups
- **Safety Options**: Skip backups with `UnsafeSkipBackup` flag
- **Restore on Failure**: Automatic restore if upgrade fails

#### Configuration
```bash
export DAEMON_DATA_BACKUP_DIR="/path/to/backups"
export UNSAFE_SKIP_BACKUP=false  # Set to true to skip backups (not recommended)
```

#### Usage
```go
// Create backup
backupManager := backup.NewManager(cfg, logger)
backupPath, err := backupManager.CreateBackup("pre-upgrade-v2.0.0")

// Restore backup
err := backupManager.RestoreBackup(backupPath)

// Clean old backups (older than 7 days)
err := backupManager.CleanOldBackups(7 * 24 * time.Hour)

// List available backups
backups, err := backupManager.ListBackups()
```

### 2. Pre-Upgrade Hooks

Pre-upgrade hooks allow custom validation and preparation before upgrades.

#### Features
- **Custom Scripts**: Run user-defined scripts before upgrade
- **Standard Scripts**: Automatic detection of pre-upgrade scripts in upgrade directory
- **Validation**: Binary existence and executable checks
- **Retry Logic**: Configurable retry attempts for failed scripts
- **Environment Variables**: Pass upgrade info to scripts

#### Script Environment Variables
- `DAEMON_HOME`: Node home directory
- `DAEMON_NAME`: Binary name
- `UPGRADE_NAME`: Upgrade version name
- `UPGRADE_HEIGHT`: Block height for upgrade
- `UPGRADE_INFO`: Additional upgrade information (JSON)

#### Configuration
```bash
export DAEMON_PREUPGRADE_MAX_RETRIES=3
export COSMOVISOR_CUSTOM_PREUPGRADE="/path/to/custom-script.sh"
```

#### Script Locations
1. **Custom Script**: Specified by `COSMOVISOR_CUSTOM_PREUPGRADE`
2. **Standard Script**: `$DAEMON_HOME/wemixvisor/upgrades/<version>/pre-upgrade`

#### Example Pre-Upgrade Script
```bash
#!/bin/bash
echo "Preparing for upgrade $UPGRADE_NAME at height $UPGRADE_HEIGHT"

# Check disk space
REQUIRED_SPACE=10000000  # 10GB in KB
AVAILABLE_SPACE=$(df "$DAEMON_HOME" | awk 'NR==2 {print $4}')
if [ "$AVAILABLE_SPACE" -lt "$REQUIRED_SPACE" ]; then
    echo "Error: Insufficient disk space"
    exit 1
fi

# Verify network connectivity
if ! ping -c 1 google.com > /dev/null 2>&1; then
    echo "Warning: Network connectivity issue"
fi

echo "Pre-upgrade checks completed successfully"
exit 0
```

### 3. Graceful Shutdown

Enhanced process management with configurable shutdown grace periods.

#### Features
- **Graceful Termination**: SIGTERM with configurable timeout
- **Thread Dump**: SIGQUIT before force kill for debugging
- **Restart Delay**: Configurable delay between stop and start
- **Signal Handling**: Proper handling of SIGTERM, SIGINT, SIGQUIT

#### Configuration
```bash
export DAEMON_SHUTDOWN_GRACE=30s  # Grace period for shutdown
export DAEMON_RESTART_DELAY=5s    # Delay before restart
```

#### Shutdown Sequence
1. Send SIGTERM to process
2. Wait for grace period (default 30s)
3. If still running, send SIGQUIT for thread dump
4. Wait additional 5s
5. Force kill with SIGKILL if necessary

## Integration with Process Manager

The process manager integrates all Phase 2 features in the upgrade workflow:

```go
func (m *Manager) performUpgrade(info *types.UpgradeInfo) error {
    // Step 1: Validate upgrade
    if err := m.preHook.ValidateUpgrade(info); err != nil {
        return err
    }

    // Step 2: Create backup
    backupPath, err := m.backup.CreateBackup(fmt.Sprintf("pre-upgrade-%s", info.Name))
    if err != nil && !m.cfg.UnsafeSkipBackup {
        return err
    }

    // Step 3: Run pre-upgrade hook
    if err := m.preHook.Execute(info); err != nil {
        // Attempt restore on failure
        if backupPath != "" {
            m.backup.RestoreBackup(backupPath)
        }
        return err
    }

    // Step 4: Update symlink
    if err := m.cfg.SetCurrentUpgrade(info.Name); err != nil {
        // Attempt restore on failure
        if backupPath != "" {
            m.backup.RestoreBackup(backupPath)
        }
        return err
    }

    // Step 5: Clean old backups
    m.backup.CleanOldBackups(7 * 24 * time.Hour)

    return nil
}
```

## Testing

### Unit Tests

All components have comprehensive unit tests:

```bash
# Run backup tests
go test ./internal/backup/

# Run hooks tests
go test ./internal/hooks/

# Run all tests
make test
```

### Test Coverage

| Package | Coverage | Description |
|---------|----------|-------------|
| `internal/backup` | High | Backup and restore operations |
| `internal/hooks` | High | Pre-upgrade hook execution |
| `internal/process` | Enhanced | Graceful shutdown logic |

## Migration from Phase 1

Phase 2 is backward compatible with Phase 1. To enable new features:

1. **Update Configuration**: Set new environment variables as needed
2. **Add Pre-Upgrade Scripts**: Place scripts in upgrade directories
3. **Configure Backup Path**: Set `DAEMON_DATA_BACKUP_DIR`
4. **Test**: Run test upgrade in non-production environment

## Security Considerations

### Backup Security
- Backups contain sensitive data - secure storage location
- Set appropriate file permissions (0600) on backup files
- Consider encryption for backups in production

### Script Security
- Validate pre-upgrade scripts before deployment
- Use absolute paths for custom scripts
- Limit script permissions to necessary operations
- Review script environment variables for sensitive data

### Process Security
- Graceful shutdown prevents data corruption
- Backup restore provides rollback capability
- Validation prevents invalid upgrades

## Next Steps (Phase 3)

- WBFT consensus integration
- Batch upgrade support
- Automatic binary download with checksum verification
- Network-wide coordination for validators