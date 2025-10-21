# Phase 8: Core Upgrade Automation - TODO Checklist

**Branch**: `feature/phase8-upgrade-automation`
**Target Version**: v0.8.0
**Priority**: P0 - CRITICAL
**Estimated Duration**: 2-3 weeks

---

## Overview

Complete the missing core upgrade automation workflow by implementing:
1. Automatic blockchain height monitoring
2. Upgrade orchestration and triggering
3. CLI commands for upgrade management
4. API endpoints for external configuration
5. Comprehensive integration testing

**Success Criteria**:
- ✅ Automatic height monitoring via RPC
- ✅ Automatic upgrade execution at configured height
- ✅ CLI commands for scheduling/listing/canceling upgrades
- ✅ API endpoints for programmatic control
- ✅ 90%+ test coverage for all new components
- ✅ Zero manual intervention required for upgrades
- ✅ Rollback mechanism on upgrade failure

---

## Task 8.1: HeightMonitor Implementation (3 days, P0)

### 8.1.1: Design & Interface Definition

- [ ] **8.1.1.1**: Create directory structure
  ```bash
  mkdir -p internal/height
  touch internal/height/interfaces.go
  touch internal/height/monitor.go
  touch internal/height/monitor_test.go
  ```
  - [ ] Verify directory created
  - [ ] Verify files created with proper permissions

- [ ] **8.1.1.2**: Define HeightProvider interface
  - [ ] Create `internal/height/interfaces.go`
  - [ ] Define interface:
    ```go
    type HeightProvider interface {
        GetCurrentHeight() (int64, error)
    }
    ```
  - [ ] Add comprehensive documentation
  - [ ] Review for SOLID compliance (ISP, DIP)

- [ ] **8.1.1.3**: Design HeightMonitor struct
  - [ ] Define struct fields:
    - [ ] `provider HeightProvider`
    - [ ] `logger *logger.Logger`
    - [ ] `currentHeight int64`
    - [ ] `mu sync.RWMutex`
    - [ ] `pollInterval time.Duration`
    - [ ] `subscribers []chan<- int64`
    - [ ] `subMu sync.RWMutex`
    - [ ] `ctx context.Context`
    - [ ] `cancel context.CancelFunc`
    - [ ] `wg sync.WaitGroup`
  - [ ] Document each field's purpose
  - [ ] Review for proper encapsulation

---

### 8.1.2: Test-Driven Development (TDD)

#### Phase 1: Red (Write Failing Tests)

- [ ] **8.1.2.1**: Create mock HeightProvider
  ```go
  // internal/height/monitor_test.go
  type MockHeightProvider struct {
      height int64
      err    error
      calls  int
  }

  func (m *MockHeightProvider) GetCurrentHeight() (int64, error) {
      m.calls++
      return m.height, m.err
  }
  ```
  - [ ] Implement mock with call counting
  - [ ] Add methods to control mock behavior
  - [ ] Verify mock implements interface

- [ ] **8.1.2.2**: Test: NewHeightMonitor creation
  ```go
  func TestNewHeightMonitor(t *testing.T) {
      // Test normal creation
      // Test with nil provider (should panic or error)
      // Test with nil logger (should panic or error)
      // Test with zero interval (should use default)
  }
  ```
  - [ ] Write test cases
  - [ ] Run tests (should fail - Red phase)
  - [ ] Document expected behavior

- [ ] **8.1.2.3**: Test: Start and Stop lifecycle
  ```go
  func TestHeightMonitor_Lifecycle(t *testing.T) {
      // Test Start initializes goroutines
      // Test Stop cleans up properly
      // Test Stop waits for goroutines
      // Test double Start returns error
      // Test double Stop is safe
  }
  ```
  - [ ] Write test cases
  - [ ] Run tests (should fail - Red phase)

- [ ] **8.1.2.4**: Test: Height monitoring
  ```go
  func TestHeightMonitor_MonitorLoop(t *testing.T) {
      // Test height updates are detected
      // Test polling interval is respected
      // Test RPC errors are handled
      // Test context cancellation stops loop
  }
  ```
  - [ ] Write test cases with timing assertions
  - [ ] Use time.After for timeout protection
  - [ ] Run tests (should fail - Red phase)

- [ ] **8.1.2.5**: Test: Subscriber pattern
  ```go
  func TestHeightMonitor_Subscribe(t *testing.T) {
      // Test Subscribe returns channel
      // Test multiple subscribers receive updates
      // Test subscriber channel buffering
      // Test full channel doesn't block monitor
  }
  ```
  - [ ] Write test cases
  - [ ] Test concurrent subscriptions
  - [ ] Run tests (should fail - Red phase)

- [ ] **8.1.2.6**: Test: Concurrent access safety
  ```go
  func TestHeightMonitor_Concurrency(t *testing.T) {
      // Test GetCurrentHeight is thread-safe
      // Test Subscribe is thread-safe
      // Test notifySubscribers doesn't deadlock
  }
  ```
  - [ ] Use race detector: `go test -race`
  - [ ] Test with multiple goroutines
  - [ ] Run tests (should fail - Red phase)

#### Phase 2: Green (Implement Minimal Code)

- [ ] **8.1.2.7**: Implement NewHeightMonitor constructor
  ```go
  func NewHeightMonitor(provider HeightProvider, interval time.Duration, logger *logger.Logger) *HeightMonitor
  ```
  - [ ] Validate inputs
  - [ ] Initialize context
  - [ ] Set default interval if zero
  - [ ] Run tests (should pass some basic tests)

- [ ] **8.1.2.8**: Implement Start method
  ```go
  func (hm *HeightMonitor) Start() error
  ```
  - [ ] Launch monitoring goroutine
  - [ ] Add proper error handling
  - [ ] Prevent double start
  - [ ] Run tests (more tests should pass)

- [ ] **8.1.2.9**: Implement Stop method
  ```go
  func (hm *HeightMonitor) Stop()
  ```
  - [ ] Cancel context
  - [ ] Wait for goroutines via WaitGroup
  - [ ] Make idempotent (safe to call multiple times)
  - [ ] Run tests (more tests should pass)

- [ ] **8.1.2.10**: Implement monitorLoop
  ```go
  func (hm *HeightMonitor) monitorLoop()
  ```
  - [ ] Create ticker for polling
  - [ ] Call provider.GetCurrentHeight()
  - [ ] Update currentHeight if changed
  - [ ] Notify subscribers on height change
  - [ ] Handle errors gracefully (log, don't crash)
  - [ ] Respect context cancellation
  - [ ] Run tests (most tests should pass now)

- [ ] **8.1.2.11**: Implement Subscribe method
  ```go
  func (hm *HeightMonitor) Subscribe() <-chan int64
  ```
  - [ ] Create buffered channel (size 10)
  - [ ] Add to subscribers list with lock
  - [ ] Return read-only channel
  - [ ] Run tests

- [ ] **8.1.2.12**: Implement notifySubscribers
  ```go
  func (hm *HeightMonitor) notifySubscribers(height int64)
  ```
  - [ ] Lock subscribers list
  - [ ] Send to each subscriber with select/default
  - [ ] Log warning if channel full
  - [ ] Don't block on slow subscribers
  - [ ] Run tests

- [ ] **8.1.2.13**: Implement GetCurrentHeight
  ```go
  func (hm *HeightMonitor) GetCurrentHeight() int64
  ```
  - [ ] Use RLock for read access
  - [ ] Return current height
  - [ ] Run tests

- [ ] **8.1.2.14**: Run all tests
  ```bash
  go test -v ./internal/height/
  ```
  - [ ] All tests pass (Green phase achieved)
  - [ ] No race conditions: `go test -race ./internal/height/`
  - [ ] Check coverage: `go test -cover ./internal/height/`

#### Phase 3: Refactor

- [ ] **8.1.2.15**: Code review for SOLID principles
  - [ ] **SRP**: HeightMonitor only monitors height ✓
  - [ ] **OCP**: Extensible via HeightProvider interface ✓
  - [ ] **LSP**: Mock implements interface correctly ✓
  - [ ] **ISP**: Interface has minimal methods ✓
  - [ ] **DIP**: Depends on abstraction, not concretion ✓

- [ ] **8.1.2.16**: Improve code quality
  - [ ] Add comprehensive comments
  - [ ] Ensure proper error messages with context
  - [ ] Use constants for magic numbers (buffer size, etc.)
  - [ ] Extract complex logic to helper methods if needed

- [ ] **8.1.2.17**: Additional test coverage
  ```go
  func TestHeightMonitor_EdgeCases(t *testing.T) {
      // Test height going backwards (reorg scenario)
      // Test very large height values
      // Test rapid height changes
      // Test provider timeout/hanging
  }
  ```
  - [ ] Write edge case tests
  - [ ] Ensure all branches covered
  - [ ] Run coverage: `go test -coverprofile=coverage.out ./internal/height/`
  - [ ] View coverage: `go tool cover -html=coverage.out`

---

### 8.1.3: Integration Testing

- [ ] **8.1.3.1**: Create integration test with governance RPC client
  ```go
  // internal/height/integration_test.go
  func TestHeightMonitor_WithRealRPC(t *testing.T) {
      if testing.Short() {
          t.Skip("Skipping integration test")
      }
      // Test with actual WBFT RPC client
  }
  ```
  - [ ] Use build tag: `// +build integration`
  - [ ] Test with real RPC endpoint (testnet)
  - [ ] Verify height updates are received
  - [ ] Run: `go test -tags=integration -v ./internal/height/`

- [ ] **8.1.3.2**: Benchmark tests
  ```go
  func BenchmarkHeightMonitor_Subscribe(b *testing.B)
  func BenchmarkHeightMonitor_NotifySubscribers(b *testing.B)
  ```
  - [ ] Write benchmark tests
  - [ ] Run: `go test -bench=. ./internal/height/`
  - [ ] Ensure performance is acceptable (<1ms per operation)

---

### 8.1.4: Documentation

- [ ] **8.1.4.1**: Add package documentation
  - [ ] Create `internal/height/doc.go` with package overview
  - [ ] Document thread-safety guarantees
  - [ ] Provide usage examples

- [ ] **8.1.4.2**: Add godoc comments
  - [ ] Document all exported types
  - [ ] Document all exported functions
  - [ ] Include examples in comments
  - [ ] Run: `go doc -all ./internal/height/`

---

### 8.1.5: Commit and Review

- [ ] **8.1.5.1**: Run all checks
  ```bash
  make fmt
  make lint
  make vet
  go test -v -cover ./internal/height/
  go test -race ./internal/height/
  ```
  - [ ] Code formatted correctly
  - [ ] No linter warnings
  - [ ] No vet issues
  - [ ] All tests pass
  - [ ] No race conditions
  - [ ] Coverage ≥ 90%

- [ ] **8.1.5.2**: Commit changes
  ```bash
  git add internal/height/
  git commit -m "feat(height): implement HeightMonitor for blockchain height tracking"
  ```
  - [ ] Commit message follows convention
  - [ ] Include test coverage in message
  - [ ] Include SOLID principles applied

- [ ] **8.1.5.3**: Self code review
  - [ ] Review diff carefully
  - [ ] Check for any TODO comments
  - [ ] Verify no debug code left
  - [ ] Ensure consistent naming conventions

---

## Task 8.2: UpgradeOrchestrator Implementation (5 days, P0)

### 8.2.1: Design & Interface Definition

- [ ] **8.2.1.1**: Create directory structure
  ```bash
  mkdir -p internal/orchestrator
  touch internal/orchestrator/interfaces.go
  touch internal/orchestrator/orchestrator.go
  touch internal/orchestrator/orchestrator_test.go
  ```

- [ ] **8.2.1.2**: Define component interfaces
  - [ ] Create `NodeManager` interface:
    ```go
    type NodeManager interface {
        Start(args []string) error
        Stop() error
        GetState() node.NodeState
        GetStatus() *node.Status
    }
    ```
  - [ ] Create `ConfigManager` interface:
    ```go
    type ConfigManager interface {
        GetConfig() *config.Config
    }
    ```
  - [ ] Create `UpgradeWatcher` interface:
    ```go
    type UpgradeWatcher interface {
        GetCurrentUpgrade() *types.UpgradeInfo
        NeedsUpdate() bool
        ClearUpdateFlag()
    }
    ```
  - [ ] Review for proper abstraction

- [ ] **8.2.1.3**: Design UpgradeOrchestrator struct
  - [ ] Define fields:
    - [ ] `nodeManager NodeManager`
    - [ ] `configManager ConfigManager`
    - [ ] `heightMonitor *height.HeightMonitor`
    - [ ] `upgradeWatcher UpgradeWatcher`
    - [ ] `logger *logger.Logger`
    - [ ] `pendingUpgrade *types.UpgradeInfo`
    - [ ] `upgrading bool`
    - [ ] `mu sync.RWMutex`
    - [ ] `ctx context.Context`
    - [ ] `cancel context.CancelFunc`
    - [ ] `wg sync.WaitGroup`
  - [ ] Document responsibilities

- [ ] **8.2.1.4**: Design UpgradeStatus struct
  ```go
  type UpgradeStatus struct {
      PendingUpgrade *types.UpgradeInfo
      Upgrading      bool
      CurrentHeight  int64
      NodeState      node.NodeState
  }
  ```

---

### 8.2.2: Test-Driven Development (TDD)

#### Phase 1: Red (Write Failing Tests)

- [ ] **8.2.2.1**: Create mock components
  ```go
  type MockNodeManager struct {
      startErr error
      stopErr  error
      state    node.NodeState
  }

  type MockConfigManager struct {
      config *config.Config
  }

  type MockUpgradeWatcher struct {
      upgrade     *types.UpgradeInfo
      needsUpdate bool
  }
  ```
  - [ ] Implement all interface methods
  - [ ] Add call tracking
  - [ ] Add configurable behavior

- [ ] **8.2.2.2**: Test: Constructor
  ```go
  func TestNewUpgradeOrchestrator(t *testing.T) {
      // Test normal creation
      // Test with nil dependencies
      // Test context initialization
  }
  ```

- [ ] **8.2.2.3**: Test: Lifecycle (Start/Stop)
  ```go
  func TestUpgradeOrchestrator_Lifecycle(t *testing.T) {
      // Test Start initializes watchers
      // Test Stop cleans up properly
      // Test double Start/Stop
  }
  ```

- [ ] **8.2.2.4**: Test: Upgrade scheduling
  ```go
  func TestUpgradeOrchestrator_ScheduleUpgrade(t *testing.T) {
      // Test scheduling valid upgrade
      // Test replacing pending upgrade
      // Test concurrent scheduling
  }
  ```

- [ ] **8.2.2.5**: Test: Height-based triggering
  ```go
  func TestUpgradeOrchestrator_HeightTrigger(t *testing.T) {
      // Test upgrade triggers at exact height
      // Test upgrade doesn't trigger before height
      // Test upgrade triggers only once
  }
  ```

- [ ] **8.2.2.6**: Test: Upgrade execution flow
  ```go
  func TestUpgradeOrchestrator_ExecuteUpgrade(t *testing.T) {
      // Test complete upgrade flow
      // Test node stop called
      // Test binary switch called
      // Test node start called with correct args
  }
  ```

- [ ] **8.2.2.7**: Test: Rollback mechanism
  ```go
  func TestUpgradeOrchestrator_Rollback(t *testing.T) {
      // Test rollback on node stop failure
      // Test rollback on binary switch failure
      // Test rollback on node start failure
      // Test rollback restores genesis binary
      // Test rollback restarts node
  }
  ```

- [ ] **8.2.2.8**: Test: Concurrent upgrade prevention
  ```go
  func TestUpgradeOrchestrator_ConcurrentUpgrade(t *testing.T) {
      // Test second upgrade blocks during first
      // Test upgrading flag prevents concurrent execution
  }
  ```

- [ ] **8.2.2.9**: Test: Error handling
  ```go
  func TestUpgradeOrchestrator_ErrorHandling(t *testing.T) {
      // Test handles node manager errors
      // Test handles config errors
      // Test handles height monitor errors
  }
  ```

- [ ] **8.2.2.10**: Run all tests (should fail - Red phase)
  ```bash
  go test -v ./internal/orchestrator/
  ```

#### Phase 2: Green (Implement Code)

- [ ] **8.2.2.11**: Implement constructor
  ```go
  func NewUpgradeOrchestrator(...) *UpgradeOrchestrator
  ```
  - [ ] Validate inputs
  - [ ] Initialize context and WaitGroup
  - [ ] Run tests

- [ ] **8.2.2.12**: Implement Start method
  ```go
  func (uo *UpgradeOrchestrator) Start() error
  ```
  - [ ] Launch watchUpgradeConfigs goroutine
  - [ ] Subscribe to height monitor
  - [ ] Launch monitorHeights goroutine
  - [ ] Run tests

- [ ] **8.2.2.13**: Implement Stop method
  ```go
  func (uo *UpgradeOrchestrator) Stop()
  ```
  - [ ] Cancel context
  - [ ] Wait for goroutines
  - [ ] Run tests

- [ ] **8.2.2.14**: Implement watchUpgradeConfigs
  ```go
  func (uo *UpgradeOrchestrator) watchUpgradeConfigs()
  ```
  - [ ] Poll upgradeWatcher periodically
  - [ ] Call scheduleUpgrade when update detected
  - [ ] Run tests

- [ ] **8.2.2.15**: Implement scheduleUpgrade
  ```go
  func (uo *UpgradeOrchestrator) scheduleUpgrade(upgrade *types.UpgradeInfo)
  ```
  - [ ] Lock mutex
  - [ ] Update pendingUpgrade
  - [ ] Log scheduling
  - [ ] Run tests

- [ ] **8.2.2.16**: Implement monitorHeights
  ```go
  func (uo *UpgradeOrchestrator) monitorHeights(heightCh <-chan int64)
  ```
  - [ ] Receive height updates
  - [ ] Check shouldTriggerUpgrade
  - [ ] Call executeUpgrade when needed
  - [ ] Run tests

- [ ] **8.2.2.17**: Implement shouldTriggerUpgrade
  ```go
  func (uo *UpgradeOrchestrator) shouldTriggerUpgrade(currentHeight int64) bool
  ```
  - [ ] Check pendingUpgrade exists
  - [ ] Check not already upgrading
  - [ ] Compare height >= upgrade height
  - [ ] Run tests

- [ ] **8.2.2.18**: Implement executeUpgrade (critical!)
  ```go
  func (uo *UpgradeOrchestrator) executeUpgrade() error
  ```
  - [ ] Set upgrading flag
  - [ ] Log start
  - [ ] Stop node via nodeManager
  - [ ] Switch binary via config.SetCurrentUpgrade
  - [ ] Start node via nodeManager
  - [ ] Wait and verify node running
  - [ ] Call rollback on any failure
  - [ ] Clear upgrading flag
  - [ ] Run tests

- [ ] **8.2.2.19**: Implement rollback
  ```go
  func (uo *UpgradeOrchestrator) rollback(cfg *config.Config) error
  ```
  - [ ] Log rollback initiation
  - [ ] Restore genesis binary via SymLinkToGenesis
  - [ ] Restart node
  - [ ] Log result
  - [ ] Run tests

- [ ] **8.2.2.20**: Implement helper methods
  ```go
  func (uo *UpgradeOrchestrator) GetPendingUpgrade() *types.UpgradeInfo
  func (uo *UpgradeOrchestrator) GetStatus() *UpgradeStatus
  ```
  - [ ] Use appropriate locks
  - [ ] Run tests

- [ ] **8.2.2.21**: Run all tests (should pass - Green phase)
  ```bash
  go test -v ./internal/orchestrator/
  go test -race ./internal/orchestrator/
  ```

#### Phase 3: Refactor

- [ ] **8.2.2.22**: SOLID principles review
  - [ ] **SRP**: Orchestrator only coordinates, doesn't execute ✓
  - [ ] **OCP**: Extensible via interfaces ✓
  - [ ] **DIP**: Depends on abstractions ✓

- [ ] **8.2.2.23**: Improve error handling
  - [ ] Wrap errors with context
  - [ ] Add structured logging with zap fields
  - [ ] Ensure rollback is always attempted

- [ ] **8.2.2.24**: Add edge case tests
  ```go
  func TestUpgradeOrchestrator_EdgeCases(t *testing.T) {
      // Test upgrade with no genesis binary
      // Test upgrade with corrupted binary
      // Test upgrade with very large height
      // Test upgrade cancel scenario
  }
  ```

- [ ] **8.2.2.25**: Coverage verification
  ```bash
  go test -coverprofile=coverage.out ./internal/orchestrator/
  go tool cover -html=coverage.out
  ```
  - [ ] Verify ≥ 90% coverage
  - [ ] Identify and test uncovered branches

---

### 8.2.3: Integration Testing

- [ ] **8.2.3.1**: Create integration test with real components
  ```go
  // internal/orchestrator/integration_test.go
  func TestUpgradeOrchestrator_RealComponents(t *testing.T) {
      // Test with real NodeManager
      // Test with real ConfigManager
      // Test with mock HeightMonitor
  }
  ```

- [ ] **8.2.3.2**: Simulate complete upgrade flow
  - [ ] Setup test environment with binaries
  - [ ] Trigger upgrade via height
  - [ ] Verify binary switched
  - [ ] Verify node restarted

- [ ] **8.2.3.3**: Test rollback scenarios
  - [ ] Corrupt upgrade binary
  - [ ] Trigger upgrade
  - [ ] Verify rollback to genesis
  - [ ] Verify node running with genesis

---

### 8.2.4: Documentation

- [ ] **8.2.4.1**: Package documentation
  - [ ] Create `doc.go` with workflow description
  - [ ] Document upgrade sequence
  - [ ] Document rollback behavior

- [ ] **8.2.4.2**: Add usage examples
  ```go
  // Example: Basic usage
  // Example: Custom configuration
  // Example: Monitoring upgrade status
  ```

---

### 8.2.5: Commit and Review

- [ ] **8.2.5.1**: Run all checks
  ```bash
  make fmt
  make lint
  make vet
  go test -v -cover ./internal/orchestrator/
  go test -race ./internal/orchestrator/
  ```

- [ ] **8.2.5.2**: Commit changes
  ```bash
  git add internal/orchestrator/
  git commit -m "feat(orchestrator): implement UpgradeOrchestrator for automated upgrades"
  ```

---

## Task 8.3: Main Integration (2 days, P0)

### 8.3.1: Update Main Entry Point

- [ ] **8.3.1.1**: Modify `cmd/wemixvisor/main.go`
  - [ ] Import new packages:
    ```go
    import (
        "github.com/wemix/wemixvisor/internal/height"
        "github.com/wemix/wemixvisor/internal/orchestrator"
    )
    ```

- [ ] **8.3.1.2**: Initialize HeightProvider
  - [ ] Reuse governance RPC client if available
  - [ ] Create new client if governance disabled
  - [ ] Handle errors gracefully
  - [ ] Add configuration option for height monitoring

- [ ] **8.3.1.3**: Create HeightMonitor
  ```go
  heightMonitor := height.NewHeightMonitor(
      heightProvider,
      cfg.HeightPollInterval,
      log,
  )
  ```
  - [ ] Add HeightPollInterval to config
  - [ ] Default to 5 seconds if not set

- [ ] **8.3.1.4**: Create UpgradeOrchestrator
  ```go
  upgradeOrchestrator := orchestrator.NewUpgradeOrchestrator(
      nodeManager,
      configManager,
      heightMonitor,
      upgradeWatcher,
      log,
  )
  ```

- [ ] **8.3.1.5**: Start components in correct order
  - [ ] Start heightMonitor first
  - [ ] Start upgradeOrchestrator second
  - [ ] Add defer statements for cleanup
  - [ ] Handle startup errors

- [ ] **8.3.1.6**: Handle shutdown gracefully
  - [ ] Stop orchestrator first
  - [ ] Stop height monitor second
  - [ ] Ensure proper cleanup

---

### 8.3.2: Configuration Updates

- [ ] **8.3.2.1**: Add new config fields
  ```go
  // internal/config/config.go
  HeightPollInterval time.Duration `mapstructure:"height_poll_interval"`
  UpgradeEnabled     bool          `mapstructure:"upgrade_enabled"`
  ```

- [ ] **8.3.2.2**: Update DefaultConfig
  - [ ] Set HeightPollInterval default to 5s
  - [ ] Set UpgradeEnabled default to true

- [ ] **8.3.2.3**: Update environment variable parsing
  - [ ] Add DAEMON_HEIGHT_POLL_INTERVAL
  - [ ] Add DAEMON_UPGRADE_ENABLED

---

### 8.3.3: Testing

- [ ] **8.3.3.1**: Manual testing
  - [ ] Build: `make build`
  - [ ] Initialize: `./dist/bin/wemixvisor init`
  - [ ] Place mock node binary
  - [ ] Start with upgrade enabled
  - [ ] Verify height monitoring in logs
  - [ ] Schedule test upgrade
  - [ ] Verify upgrade triggers

- [ ] **8.3.3.2**: Integration test
  ```go
  // test/integration/main_test.go
  func TestMain_WithUpgradeOrchestrator(t *testing.T) {
      // Test full wemixvisor startup
      // Test components initialized correctly
  }
  ```

---

### 8.3.4: Commit

- [ ] **8.3.4.1**: Run checks
  ```bash
  make build
  make test
  ```

- [ ] **8.3.4.2**: Commit
  ```bash
  git add cmd/wemixvisor/main.go internal/config/
  git commit -m "feat(main): integrate HeightMonitor and UpgradeOrchestrator"
  ```

---

## Task 8.4: CLI Commands (2 days, P1)

### 8.4.1: Create Upgrade Command Package

- [ ] **8.4.1.1**: Create files
  ```bash
  touch internal/cli/upgrade.go
  touch internal/cli/upgrade_test.go
  ```

---

### 8.4.2: Implement Schedule Command

- [ ] **8.4.2.1**: Define command structure
  ```go
  func NewUpgradeCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command
  func newScheduleCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command
  ```

- [ ] **8.4.2.2**: Implement schedule logic
  - [ ] Parse name and height arguments
  - [ ] Validate height > 0
  - [ ] Support optional flags:
    - [ ] `--binaries <url>`
    - [ ] `--checksum <hash>`
    - [ ] `--info <text>`
  - [ ] Create UpgradeInfo struct
  - [ ] Write to upgrade-info.json atomically
  - [ ] Print confirmation message

- [ ] **8.4.2.3**: Write tests
  ```go
  func TestScheduleCommand(t *testing.T) {
      // Test normal schedule
      // Test invalid height
      // Test with binaries flag
      // Test file creation
  }
  ```

---

### 8.4.3: Implement List Command

- [ ] **8.4.3.1**: Implement list logic
  ```go
  func newListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command
  ```
  - [ ] Read upgrade-info.json
  - [ ] Parse JSON
  - [ ] Format output nicely
  - [ ] Handle file not exists

- [ ] **8.4.3.2**: Write tests
  ```go
  func TestListCommand(t *testing.T) {
      // Test with no upgrades
      // Test with scheduled upgrade
      // Test with invalid file
  }
  ```

---

### 8.4.4: Implement Cancel Command

- [ ] **8.4.4.1**: Implement cancel logic
  ```go
  func newCancelCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command
  ```
  - [ ] Delete upgrade-info.json
  - [ ] Handle file not exists gracefully
  - [ ] Print confirmation

- [ ] **8.4.4.2**: Write tests

---

### 8.4.5: Implement Status Command

- [ ] **8.4.5.1**: Implement status logic
  ```go
  func newStatusCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command
  ```
  - [ ] Show pending upgrade if any
  - [ ] Show current blockchain height
  - [ ] Show upgrade status (pending/in-progress/none)
  - [ ] Support JSON output flag

- [ ] **8.4.5.2**: Write tests

---

### 8.4.6: Integration

- [ ] **8.4.6.1**: Update root command
  ```go
  // internal/cli/root.go
  cmd.AddCommand(NewUpgradeCommand(cfg, logger))
  ```

- [ ] **8.4.6.2**: Update help text in main.go

---

### 8.4.7: Testing

- [ ] **8.4.7.1**: Manual CLI testing
  ```bash
  # Schedule
  ./dist/bin/wemixvisor upgrade schedule v2.0.0 1500000

  # List
  ./dist/bin/wemixvisor upgrade list

  # Status
  ./dist/bin/wemixvisor upgrade status

  # Cancel
  ./dist/bin/wemixvisor upgrade cancel
  ```

- [ ] **8.4.7.2**: Integration test
  ```go
  func TestCLI_UpgradeCommands(t *testing.T) {
      // Test schedule → list → cancel flow
  }
  ```

---

### 8.4.8: Commit

- [ ] **8.4.8.1**: Run checks
  ```bash
  make fmt
  make lint
  go test -v ./internal/cli/
  ```

- [ ] **8.4.8.2**: Commit
  ```bash
  git add internal/cli/upgrade.go internal/cli/upgrade_test.go internal/cli/root.go
  git commit -m "feat(cli): add upgrade management commands"
  ```

---

## Task 8.5: API Endpoints (2 days, P1)

### 8.5.1: Create API Handlers

- [ ] **8.5.1.1**: Create files
  ```bash
  touch internal/api/upgrade_handlers.go
  touch internal/api/upgrade_handlers_test.go
  ```

---

### 8.5.2: Implement POST /api/v1/upgrades

- [ ] **8.5.2.1**: Define request/response structures
  ```go
  type ScheduleUpgradeRequest struct {
      Name      string            `json:"name" binding:"required"`
      Height    int64             `json:"height" binding:"required,gt=0"`
      Info      string            `json:"info"`
      Binaries  map[string]string `json:"binaries"`
      Checksums map[string]string `json:"checksums"`
  }
  ```

- [ ] **8.5.2.2**: Implement handler
  ```go
  func (s *Server) scheduleUpgrade(c *gin.Context)
  ```
  - [ ] Bind JSON request
  - [ ] Validate input
  - [ ] Call orchestrator.ScheduleUpgrade
  - [ ] Return 200 OK or error

- [ ] **8.5.2.3**: Write tests
  ```go
  func TestAPI_ScheduleUpgrade(t *testing.T) {
      // Test valid request
      // Test invalid height
      // Test missing name
      // Test orchestrator error
  }
  ```

---

### 8.5.3: Implement GET /api/v1/upgrades

- [ ] **8.5.3.1**: Implement handler
  ```go
  func (s *Server) listUpgrades(c *gin.Context)
  ```
  - [ ] Get pending upgrade from orchestrator
  - [ ] Return JSON array
  - [ ] Include count

- [ ] **8.5.3.2**: Write tests

---

### 8.5.4: Implement DELETE /api/v1/upgrades/:name

- [ ] **8.5.4.1**: Implement handler
  ```go
  func (s *Server) cancelUpgrade(c *gin.Context)
  ```
  - [ ] Get name from path parameter
  - [ ] Call orchestrator.CancelUpgrade
  - [ ] Return success/error

- [ ] **8.5.4.2**: Write tests

---

### 8.5.5: Implement GET /api/v1/upgrades/status

- [ ] **8.5.5.1**: Implement handler
  ```go
  func (s *Server) getUpgradeStatus(c *gin.Context)
  ```
  - [ ] Get status from orchestrator
  - [ ] Return JSON with all fields
  - [ ] Include current height and node state

- [ ] **8.5.5.2**: Write tests

---

### 8.5.6: Register Routes

- [ ] **8.5.6.1**: Add to server initialization
  ```go
  func (s *Server) RegisterUpgradeRoutes()
  ```
  - [ ] Register all routes
  - [ ] Add to main RegisterRoutes call

---

### 8.5.7: Update Server Struct

- [ ] **8.5.7.1**: Add orchestrator reference
  ```go
  type Server struct {
      // ... existing fields
      orchestrator *orchestrator.UpgradeOrchestrator
  }
  ```

- [ ] **8.5.7.2**: Update constructor

---

### 8.5.8: Testing

- [ ] **8.5.8.1**: Manual API testing
  ```bash
  # Start API
  ./dist/bin/wemixvisor api --port 8080

  # Schedule
  curl -X POST http://localhost:8080/api/v1/upgrades \
    -H "Content-Type: application/json" \
    -d '{"name":"v2.0.0","height":1500000}'

  # List
  curl http://localhost:8080/api/v1/upgrades

  # Status
  curl http://localhost:8080/api/v1/upgrades/status

  # Cancel
  curl -X DELETE http://localhost:8080/api/v1/upgrades/v2.0.0
  ```

- [ ] **8.5.8.2**: Integration test
  ```go
  func TestAPI_UpgradeEndpoints(t *testing.T) {
      // Test complete flow via API
  }
  ```

---

### 8.5.9: Commit

- [ ] **8.5.9.1**: Run checks
  ```bash
  go test -v ./internal/api/
  ```

- [ ] **8.5.9.2**: Commit
  ```bash
  git add internal/api/upgrade_handlers.go internal/api/upgrade_handlers_test.go
  git commit -m "feat(api): add upgrade management endpoints"
  ```

---

## Task 8.6: Integration Testing (3 days, P0)

### 8.6.1: Setup E2E Test Environment

- [ ] **8.6.1.1**: Create test directory
  ```bash
  mkdir -p test/e2e
  touch test/e2e/upgrade_orchestration_test.go
  touch test/e2e/test_helpers.go
  ```

- [ ] **8.6.1.2**: Create test helper functions
  ```go
  type TestEnvironment struct {
      tmpDir      string
      wemixvisor  *exec.Cmd
      mockNode    *MockNode
      rpcServer   *MockRPCServer
      config      *config.Config
  }

  func setupTestEnvironment(t *testing.T) *TestEnvironment
  func (env *TestEnvironment) Cleanup()
  func (env *TestEnvironment) Start()
  func (env *TestEnvironment) ScheduleUpgrade(info *types.UpgradeInfo) error
  func (env *TestEnvironment) AdvanceBlockHeight(height int64)
  func (env *TestEnvironment) GetNodeStatus() *node.Status
  ```

---

### 8.6.2: Test Complete Upgrade Workflow

- [ ] **8.6.2.1**: Test normal upgrade flow
  ```go
  func TestUpgradeOrchestration_CompleteWorkflow(t *testing.T)
  ```
  - [ ] Setup test environment
  - [ ] Start wemixvisor
  - [ ] Schedule upgrade
  - [ ] Verify upgrade scheduled
  - [ ] Advance height below upgrade height
  - [ ] Verify no upgrade triggered
  - [ ] Advance to upgrade height
  - [ ] Wait for upgrade execution
  - [ ] Verify node running with new binary
  - [ ] Verify no pending upgrade

- [ ] **8.6.2.2**: Test rollback on failure
  ```go
  func TestUpgradeOrchestration_RollbackOnFailure(t *testing.T)
  ```
  - [ ] Setup with invalid upgrade binary
  - [ ] Trigger upgrade
  - [ ] Verify rollback occurred
  - [ ] Verify node running with genesis binary

- [ ] **8.6.2.3**: Test multiple upgrades
  ```go
  func TestUpgradeOrchestration_MultipleUpgrades(t *testing.T)
  ```
  - [ ] Schedule first upgrade
  - [ ] Execute first upgrade
  - [ ] Schedule second upgrade
  - [ ] Execute second upgrade
  - [ ] Verify chain of upgrades

---

### 8.6.3: Test CLI Integration

- [ ] **8.6.3.1**: Test CLI schedule and execute
  ```go
  func TestE2E_CLI_ScheduleAndExecute(t *testing.T)
  ```
  - [ ] Schedule via CLI
  - [ ] Verify via CLI list
  - [ ] Advance height
  - [ ] Verify upgrade executed
  - [ ] Verify via CLI status

- [ ] **8.6.3.2**: Test CLI cancel
  ```go
  func TestE2E_CLI_Cancel(t *testing.T)
  ```
  - [ ] Schedule via CLI
  - [ ] Cancel via CLI
  - [ ] Advance height
  - [ ] Verify no upgrade executed

---

### 8.6.4: Test API Integration

- [ ] **8.6.4.1**: Test API schedule and monitor
  ```go
  func TestE2E_API_ScheduleAndMonitor(t *testing.T)
  ```
  - [ ] Start API server
  - [ ] Schedule via API POST
  - [ ] Monitor via API GET
  - [ ] Advance height
  - [ ] Verify via API status

- [ ] **8.6.4.2**: Test WebSocket notifications
  ```go
  func TestE2E_API_WebSocketNotifications(t *testing.T)
  ```
  - [ ] Connect to WebSocket
  - [ ] Subscribe to upgrade events
  - [ ] Schedule upgrade
  - [ ] Receive upgrade scheduled event
  - [ ] Advance height
  - [ ] Receive upgrade triggered event
  - [ ] Receive upgrade completed event

---

### 8.6.5: Test Error Scenarios

- [ ] **8.6.5.1**: Test network disconnection
  ```go
  func TestE2E_NetworkDisconnection(t *testing.T)
  ```
  - [ ] Disconnect RPC during height monitoring
  - [ ] Verify graceful error handling
  - [ ] Reconnect
  - [ ] Verify monitoring resumes

- [ ] **8.6.5.2**: Test concurrent upgrade attempts
  ```go
  func TestE2E_ConcurrentUpgrades(t *testing.T)
  ```
  - [ ] Trigger upgrade manually
  - [ ] Attempt second upgrade while first in progress
  - [ ] Verify second is blocked

- [ ] **8.6.5.3**: Test node crash during upgrade
  ```go
  func TestE2E_NodeCrashDuringUpgrade(t *testing.T)
  ```
  - [ ] Trigger upgrade
  - [ ] Kill node during start
  - [ ] Verify rollback and recovery

---

### 8.6.6: Performance Testing

- [ ] **8.6.6.1**: Test height monitoring overhead
  ```go
  func TestE2E_HeightMonitoringOverhead(t *testing.T)
  ```
  - [ ] Measure CPU usage with monitoring
  - [ ] Verify < 5% overhead
  - [ ] Measure memory usage
  - [ ] Verify < 10MB additional

- [ ] **8.6.6.2**: Test upgrade execution time
  ```go
  func TestE2E_UpgradeExecutionTime(t *testing.T)
  ```
  - [ ] Measure time from trigger to completion
  - [ ] Verify < 30 seconds for normal case

---

### 8.6.7: Load Testing

- [ ] **8.6.7.1**: Test rapid height changes
  ```go
  func TestE2E_RapidHeightChanges(t *testing.T)
  ```
  - [ ] Simulate 1 block/second
  - [ ] Verify monitoring keeps up
  - [ ] No memory leaks

- [ ] **8.6.7.2**: Test long-running stability
  ```go
  func TestE2E_LongRunningStability(t *testing.T)
  ```
  - [ ] Run for 1 hour
  - [ ] Monitor for memory leaks
  - [ ] Monitor for goroutine leaks
  - [ ] Verify no crashes

---

### 8.6.8: Documentation

- [ ] **8.6.8.1**: Create E2E testing guide
  - [ ] Document how to run E2E tests
  - [ ] Document test environment setup
  - [ ] Document troubleshooting

- [ ] **8.6.8.2**: Update main README
  - [ ] Add E2E testing section
  - [ ] Document new CLI commands
  - [ ] Document new API endpoints

---

### 8.6.9: Commit

- [ ] **8.6.9.1**: Run all E2E tests
  ```bash
  go test -v -tags=e2e ./test/e2e/
  ```

- [ ] **8.6.9.2**: Commit
  ```bash
  git add test/e2e/
  git commit -m "test(e2e): comprehensive integration tests for upgrade automation"
  ```

---

## Final Phase: Review and Release

### Pre-Release Checklist

- [ ] **All Tasks Completed**
  - [ ] Task 8.1: HeightMonitor ✅
  - [ ] Task 8.2: UpgradeOrchestrator ✅
  - [ ] Task 8.3: Main Integration ✅
  - [ ] Task 8.4: CLI Commands ✅
  - [ ] Task 8.5: API Endpoints ✅
  - [ ] Task 8.6: Integration Testing ✅

- [ ] **Code Quality**
  - [ ] All tests pass: `make test`
  - [ ] No race conditions: `go test -race ./...`
  - [ ] Test coverage ≥ 90%: `make coverage`
  - [ ] No linter warnings: `make lint`
  - [ ] Code formatted: `make fmt`
  - [ ] No vet issues: `make vet`

- [ ] **SOLID Principles Verified**
  - [ ] Single Responsibility ✅
  - [ ] Open/Closed ✅
  - [ ] Liskov Substitution ✅
  - [ ] Interface Segregation ✅
  - [ ] Dependency Inversion ✅

- [ ] **Documentation**
  - [ ] All packages documented
  - [ ] README updated
  - [ ] API documentation complete
  - [ ] CLI help text updated
  - [ ] Examples provided

- [ ] **Performance**
  - [ ] Height monitoring overhead < 5%
  - [ ] Memory usage < 100MB additional
  - [ ] Upgrade execution < 30s
  - [ ] No memory leaks
  - [ ] No goroutine leaks

- [ ] **Manual Testing**
  - [ ] Test on testnet
  - [ ] Test complete upgrade flow
  - [ ] Test rollback scenario
  - [ ] Test CLI commands
  - [ ] Test API endpoints
  - [ ] Test error handling

---

### Release Preparation

- [ ] **Version Update**
  - [ ] Update version to v0.8.0
  - [ ] Update CHANGES.md
  - [ ] Update README.md

- [ ] **Create Release Branch**
  ```bash
  git checkout -b release/v0.8.0
  ```

- [ ] **Final Commit**
  ```bash
  git commit -m "chore(release): prepare v0.8.0 release"
  ```

- [ ] **Merge to Main**
  ```bash
  git checkout main
  git merge release/v0.8.0
  git tag v0.8.0
  git push origin main --tags
  ```

- [ ] **Merge to Dev**
  ```bash
  git checkout dev
  git merge main
  git push origin dev
  ```

---

## Success Criteria Verification

### Functional Requirements ✅
- [ ] Automatic block height monitoring implemented
- [ ] Automatic upgrade execution at configured height
- [ ] CLI command for upgrade scheduling works
- [ ] API endpoint for upgrade management works
- [ ] Rollback on upgrade failure works
- [ ] Multiple configuration sources (file, CLI, API)
- [ ] Hot-reload configuration support maintained
- [ ] Comprehensive logging and alerting

### Quality Requirements ✅
- [ ] Test coverage ≥ 90% for new components
- [ ] Zero critical bugs
- [ ] SOLID principles compliance verified
- [ ] Comprehensive documentation complete
- [ ] Performance overhead < 5%
- [ ] Memory usage < 100MB additional

### Operational Requirements ✅
- [ ] 99.9% uptime (node auto-restart maintained)
- [ ] Zero manual intervention for upgrades
- [ ] Complete audit trail in logs
- [ ] Production deployment guide updated

---

## Notes

**Development Guidelines**:
- Follow TDD: Red → Green → Refactor
- Commit frequently with clear messages
- Run tests before each commit
- Keep test coverage ≥ 90%
- Document as you code
- Review SOLID principles regularly

**Testing Strategy**:
- Unit tests for all functions
- Integration tests for component interaction
- E2E tests for complete workflows
- Performance tests for overhead
- Load tests for stability

**Documentation Requirements**:
- Package-level documentation
- Function-level comments
- Usage examples
- API documentation
- CLI help text

**Code Review Checklist**:
- SOLID principles followed
- Error handling comprehensive
- Concurrency safety verified
- Performance acceptable
- Documentation complete
- Tests comprehensive

---

**Total Estimated Time**: 17 days (2-3 weeks)
**Target Completion**: End of Week 3
**Production Readiness**: 95% after Phase 8 completion
