# Testing Guide

## Overview

This document describes the testing strategy and implementation for Wemixvisor, including unit tests and end-to-end (E2E) tests.

## Test Structure

```
wemixvisor/
├── internal/
│   ├── config/
│   │   └── config_test.go      # Config package unit tests
│   └── upgrade/
│       └── watcher_test.go     # File watcher unit tests
├── pkg/
│   ├── logger/
│   │   └── logger_test.go      # Logger unit tests
│   └── types/
│       └── upgrade_test.go     # Types unit tests
└── test/
    └── e2e/
        └── process_upgrade_test.go  # End-to-end tests
```

## Running Tests

### Unit Tests

Run all unit tests with coverage:
```bash
make test
```

Run specific package tests:
```bash
go test -v ./internal/config/
go test -v ./pkg/types/
```

### End-to-End Tests

Run E2E tests (requires more time and resources):
```bash
make test-e2e
```

### All Tests

Run both unit and E2E tests:
```bash
make test-all
```

### Coverage Report

Generate HTML coverage report:
```bash
make coverage
# Opens coverage.html in browser
```

## Test Coverage

### Phase 1 Coverage

| Package | Coverage | Description |
|---------|----------|-------------|
| `internal/config` | ✅ High | Configuration management and path operations |
| `internal/upgrade` | ✅ High | File watcher and upgrade detection |
| `internal/process` | ⚠️ Medium | Process management (requires mock processes) |
| `internal/commands` | ⚠️ Low | CLI commands (tested via E2E) |
| `pkg/types` | ✅ High | Type definitions and JSON parsing |
| `pkg/logger` | ✅ High | Logging functionality |

### Coverage Goals

- **Unit Test Coverage**: Target 80% or higher
- **Critical Path Coverage**: 100% for upgrade detection and binary switching
- **E2E Coverage**: Core upgrade scenarios

## Unit Tests

### Config Package Tests

Tests configuration management functionality:
- Default configuration values
- Environment variable handling
- Path generation (home, genesis, upgrades, etc.)
- Symbolic link operations
- Configuration validation

**Key Test Cases**:
- `TestDefaultConfig`: Verifies default values and env var override
- `TestConfigPaths`: Tests all path generation methods
- `TestSymLinkToGenesis`: Tests genesis symlink creation
- `TestSetCurrentUpgrade`: Tests upgrade symlink switching
- `TestValidate`: Tests configuration validation rules

### Types Package Tests

Tests type definitions and serialization:
- Upgrade info parsing from JSON files
- Binary info extraction
- Upgrade plan serialization
- Error handling for invalid data

**Key Test Cases**:
- `TestParseUpgradeInfoFile`: Tests parsing various JSON formats
- `TestWriteUpgradeInfoFile`: Tests writing upgrade info
- `TestParseBinaryInfo`: Tests binary URL and checksum extraction
- `TestUpgradePlan`: Tests plan marshaling/unmarshaling

### Logger Package Tests

Tests logging functionality:
- Logger creation with different configurations
- Log level handling
- Time format options
- Disabled logging mode

**Key Test Cases**:
- `TestNew`: Tests logger creation with various configs
- `TestLoggerMethods`: Tests all log methods (Info, Debug, Warn, Error)
- `TestLoggerWithDisabled`: Tests noop logger when disabled

### Upgrade Watcher Tests

Tests file monitoring and upgrade detection:
- File watcher initialization
- Upgrade file detection
- Modification time tracking
- Polling mechanism
- Upgrade state management

**Key Test Cases**:
- `TestFileWatcherStartStop`: Tests lifecycle management
- `TestFileWatcherCheckFile`: Tests file parsing and validation
- `TestFileWatcherCheckForUpdate`: Tests update detection logic
- `TestFileWatcherMonitoring`: Tests continuous monitoring
- `TestFileWatcherWaitForUpgrade`: Tests blocking wait for upgrades

## End-to-End Tests

### Process Upgrade Test

Complete upgrade workflow testing:
1. Build wemixvisor binary
2. Create mock daemon binaries (v1.0.0 and v2.0.0)
3. Initialize wemixvisor with genesis binary
4. Start process monitoring
5. Trigger upgrade via upgrade-info.json
6. Verify automatic binary switching
7. Confirm new version is running

### Init Command Test

Tests initialization workflow:
- Directory structure creation
- Genesis binary copying
- Symbolic link setup
- Permission handling

### Version Command Test

Tests version information display:
- Version number output
- Git commit hash
- Build date

## Test Utilities

### Mock Binaries

E2E tests use mock daemon binaries that:
- Print version information
- Run continuously until killed
- Simulate real daemon behavior

### Helper Functions

- `createMockDaemon`: Creates test binaries
- `copyFile`: File copying with permissions
- `waitForOutput`: Waits for expected output with timeout
- `contains`: String containment check

## Testing Best Practices

### 1. Test Isolation

- Use `t.TempDir()` for temporary test directories
- Clean up resources with `defer` statements
- Avoid modifying global state

### 2. Error Handling

- Check all errors in tests
- Use `t.Fatal()` for setup failures
- Use `t.Error()` for assertion failures

### 3. Test Data

- Use table-driven tests for multiple scenarios
- Keep test data close to tests
- Use meaningful test case names

### 4. Timing

- Use appropriate timeouts for async operations
- Avoid fixed sleep durations when possible
- Use polling with timeout patterns

### 5. Coverage

- Focus on critical paths first
- Test error conditions
- Include edge cases

## Continuous Integration

### GitHub Actions Workflow

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: make test
      - run: make test-e2e
```

## Future Improvements

### Phase 2 Testing
- Backup functionality tests
- Pre-upgrade hook tests
- Graceful shutdown tests
- Custom script execution tests

### Phase 3 Testing
- WBFT integration tests
- Network coordination tests
- Binary download tests
- Checksum verification tests

### Performance Testing
- Load testing with continuous upgrades
- Memory leak detection
- Resource usage monitoring
- Concurrent upgrade handling

### Integration Testing
- Testing with real WBFT nodes
- Multi-node upgrade coordination
- Network partition scenarios
- Rollback testing