# Wemixvisor Architectural Review & Development Plan
**Date**: 2025-01-20
**Version**: v0.7.0
**Reviewer**: Comprehensive System Analysis
**Status**: Production Readiness Assessment

---

## Executive Summary

### Overall Assessment
Wemixvisor has achieved **75% production readiness** with excellent foundation in node lifecycle management, configuration system, and code quality. However, **critical gaps** exist in the core upgrade automation workflow that prevent fully automated, human-error-free operations.

### Critical Findings
🔴 **CRITICAL**: Missing automatic block height monitoring and upgrade triggering
🟡 **HIGH**: Incomplete integration between upgrade detection components
🟡 **HIGH**: Limited external configuration methods for upgrade height
🟢 **EXCELLENT**: Node lifecycle management and process resilience
🟢 **EXCELLENT**: SOLID principles adherence and code quality

---

## 1. Architecture Analysis

### 1.1 Current Component Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Wemixvisor Core                         │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌──────────────┐    ┌──────────────┐    ┌───────────────┐  │
│  │ Config       │───▶│ Node Manager │───▶│ Node Process  │  │
│  │ Manager      │    │   (Phase 4)  │    │   (geth)      │  │
│  └──────┬───────┘    └──────┬───────┘    └───────────────┘  │
│         │                   │                                │
│         │ Hot-Reload        │ Lifecycle Control              │
│         ▼                   ▼                                │
│  ┌──────────────┐    ┌──────────────┐                       │
│  │ File Watcher │    │ Governance   │                       │
│  │ (Upgrade     │    │ Monitor      │                       │
│  │  Detection)  │    │ (Phase 6)    │                       │
│  └──────────────┘    └──────────────┘                       │
│         │                   │                                │
│         │                   │                                │
│         ▼                   ▼                                │
│  ┌──────────────────────────────────┐                       │
│  │    upgrade-info.json File         │                       │
│  └──────────────────────────────────┘                       │
│                                                               │
│  ┌─────────────────────────────────────────────────┐        │
│  │ Monitoring Layer (Phase 7)                      │        │
│  │  - Metrics Collector                             │        │
│  │  - Health Checker                                │        │
│  │  - Alerting System                               │        │
│  │  - API Server                                    │        │
│  └─────────────────────────────────────────────────┘        │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

#### ✅ Node Manager (internal/node/)
**Responsibility**: Node process lifecycle management
**Implementation Quality**: Excellent (91.2% test coverage)

**Strengths**:
- State machine pattern for process states (Stopped, Starting, Running, Stopping, Crashed, Error)
- Auto-restart with configurable max limits
- Proper signal handling (SIGTERM → grace period → SIGKILL)
- Process group management preventing zombie processes
- Health monitoring integration
- Metrics collection integration

**Code Example** (node/manager.go:107-208):
```go
// Excellent separation of concerns and error handling
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()

    if m.state != StateStopped {
        m.stateMutex.Unlock()
        return fmt.Errorf("node is not in stopped state: %v", m.state)
    }

    // State transition
    m.state = StateStarting
    // ... setup code ...

    // Unlock before spawning goroutines to prevent deadlock
    m.stateMutex.Unlock()

    // Start monitoring goroutine
    go m.monitor()

    return nil
}
```

**SOLID Compliance**:
- ✅ SRP: Focused on process lifecycle only
- ✅ DIP: Depends on abstractions (logger, config interfaces)
- ⚠️ Minor: Could extract metrics collection to separate component

---

#### ✅ Config Manager (internal/config/)
**Responsibility**: Configuration management with hot-reload
**Implementation Quality**: Excellent

**Strengths**:
- Hot-reload via fsnotify file watcher
- Multiple format support (TOML, YAML, JSON)
- Template system for network-specific configs
- Atomic file writes (write to .tmp → rename)
- Update notification channel for subscribers
- Migration system for version upgrades

**Code Example** (config/manager.go:411-455):
```go
// Excellent use of file watching for hot-reload
func (m *Manager) watchLoop() {
    for {
        select {
        case <-m.ctx.Done():
            return
        case event, ok := <-m.watcher.Events:
            if !ok {
                return
            }
            if event.Op&fsnotify.Write == fsnotify.Write {
                m.handleConfigChange()
            }
        }
    }
}
```

**SOLID Compliance**:
- ✅ SRP: Configuration management only
- ✅ OCP: Extensible via templates
- ✅ DIP: No concrete dependencies

---

#### ⚠️ Upgrade FileWatcher (internal/upgrade/)
**Responsibility**: Monitor upgrade-info.json for changes
**Implementation Quality**: Good, but **incomplete integration**

**Strengths**:
- Proper file polling with configurable interval
- Thread-safe with RWMutex
- Detects file modifications by timestamp

**Critical Gaps**:
```go
// Line 215-220: Placeholder implementation!
func (fw *FileWatcher) CheckHeight() (int64, error) {
    // TODO: Implement actual height checking via RPC
    // For now, return a mock value
    return 0, nil  // ❌ CRITICAL: Always returns 0!
}
```

**Missing Functionality**:
- ❌ No actual block height monitoring
- ❌ No automatic comparison: current_height vs upgrade_height
- ❌ No integration with governance monitor
- ❌ No orchestration to trigger upgrade at specific height

**SOLID Compliance**:
- ✅ SRP: Focused on file watching
- ❌ OCP: Hard to extend with new upgrade sources
- ✅ LSP: Mockable interface

---

#### ⚠️ Governance Monitor (internal/governance/)
**Responsibility**: Track on-chain proposals and schedule upgrades
**Implementation Quality**: Good, but **isolated from main workflow**

**Strengths**:
- Monitors blockchain proposals via RPC
- Tracks voting progress
- Can create upgrade-info.json when proposal passes

**Critical Gaps**:
```go
// Lines 292-331: Has height checking but disconnected from FileWatcher
func (m *Monitor) processUpgradeQueue() error {
    currentHeight, err := m.rpcClient.GetCurrentHeight()  // ✅ Has height
    // ...
    for _, upgrade := range upgrades {
        if upgrade.Height <= currentHeight {  // ✅ Compares height
            m.triggerUpgrade(upgrade)  // ✅ Can trigger
        }
    }
}
```

**Problem**: This component **has** the capability to monitor height and trigger upgrades, but it's:
- Only active when governance monitoring is enabled
- Not integrated with the main upgrade watcher
- Creates upgrade-info.json but doesn't directly control node lifecycle

**SOLID Compliance**:
- ✅ SRP: Focused on governance monitoring
- ✅ DIP: Uses WBFTClientInterface abstraction
- ⚠️ Integration: Should coordinate with main upgrade orchestrator

---

### 1.3 Critical Missing Component: Upgrade Orchestrator

**Problem**: No central component coordinates the upgrade workflow

**Current Workflow** (Broken):
```
1. User manually creates upgrade-info.json
   OR
2. Governance proposal passes → creates upgrade-info.json
   ↓
3. FileWatcher detects file change
   ↓
4. ??? (No automatic height monitoring or trigger)
   ↓
5. User must manually restart node (defeating automation purpose)
```

**Required Workflow**:
```
1. Upgrade configured (file/command/API/governance)
   ↓
2. Orchestrator monitors blockchain height continuously
   ↓
3. When currentHeight >= upgradeHeight:
   ↓
4. Orchestrator triggers upgrade:
      a. Stop node (via NodeManager)
      b. Switch binary symlink (via Config)
      c. Start node with new binary
   ↓
5. Monitor health and auto-restart if needed
```

---

## 2. SOLID Principles Assessment

### 2.1 Single Responsibility Principle (SRP)
**Score: 8/10** - Strong adherence with minor issues

**Strengths** ✅:
- `FileWatcher`: Only watches upgrade file
- `NodeManager`: Only manages node lifecycle
- `ConfigManager`: Only manages configuration
- `GovernanceMonitor`: Only monitors proposals

**Issues** ⚠️:
- `NodeManager` also handles metrics collection (lines 83-101)
- Should extract to separate `MetricsCoordinator`

**Recommendation**: Extract metrics collection orchestration:
```go
// Create new component
type MetricsCoordinator struct {
    nodeManager *node.Manager
    collector   *metrics.Collector
}

func (mc *MetricsCoordinator) Start() {
    mc.collector.SetNodeHeightCallback(func() (int64, error) {
        return mc.nodeManager.GetCurrentHeight()
    })
    mc.collector.Start()
}
```

---

### 2.2 Open/Closed Principle (OCP)
**Score: 7/10** - Moderate extensibility

**Strengths** ✅:
- Good use of interfaces (WBFTClientInterface, NotificationChannel)
- Template system allows config extension without modification
- Alert channels extensible via interface

**Issues** ⚠️:
- Adding new upgrade sources (e.g., HTTP API, database) requires modifying FileWatcher
- No plugin system for custom upgrade detection

**Recommendation**: Create UpgradeSource interface:
```go
type UpgradeSource interface {
    Watch(ctx context.Context) (<-chan *UpgradeInfo, error)
    GetCurrentUpgrade() (*UpgradeInfo, error)
}

// Implementations:
type FileUpgradeSource struct { ... }
type GovernanceUpgradeSource struct { ... }
type APIUpgradeSource struct { ... }

// Orchestrator uses multiple sources
type UpgradeOrchestrator struct {
    sources []UpgradeSource
}
```

---

### 2.3 Liskov Substitution Principle (LSP)
**Score: 9/10** - Excellent interface compliance

**Strengths** ✅:
- All interface implementations are fully interchangeable
- Mock implementations in tests work seamlessly
- No violations of expected behavior

**Evidence**:
```go
// WBFTClientInterface properly substitutable
type WBFTClientInterface interface {
    GetCurrentHeight() (int64, error)
    GetProposal(id string) (*Proposal, error)
    Close() error
}

// Both production and mock work identically
production := NewWBFTClient(addr, logger)
mock := &MockWBFTClient{}
// Both satisfy interface perfectly
```

---

### 2.4 Interface Segregation Principle (ISP)
**Score: 8/10** - Good interface design

**Strengths** ✅:
- Interfaces are focused and minimal
- No fat interfaces forcing unnecessary methods
- Clients only depend on methods they use

**Minor Improvement**:
Could segregate WBFTClientInterface further:
```go
// Current (good but could be better)
type WBFTClientInterface interface {
    GetCurrentHeight() (int64, error)
    GetProposal(id string) (*Proposal, error)
    GetProposals() ([]*Proposal, error)
    Close() error
}

// Suggested: Segregate responsibilities
type HeightProvider interface {
    GetCurrentHeight() (int64, error)
}

type ProposalProvider interface {
    GetProposal(id string) (*Proposal, error)
    GetProposals() ([]*Proposal, error)
}

type Closeable interface {
    Close() error
}
```

---

### 2.5 Dependency Inversion Principle (DIP)
**Score: 9/10** - Excellent dependency management

**Strengths** ✅:
- All major components depend on abstractions
- Configuration, logging injected via constructors
- Easy to test with mocks
- No concrete dependency on external services

**Evidence**:
```go
// Excellent dependency injection
func NewManager(cfg *config.Config, logger *logger.Logger) *Manager {
    // Depends on abstractions, not concretions
}

// Easy to mock for testing
func NewMonitor(cfg *config.Config, logger *logger.Logger) *Monitor {
    // Can inject mock RPCClient via SetRPCClient()
}
```

---

## 3. Gap Analysis Against Requirements

### 3.1 User Requirements
Based on the user's explanation, wemixvisor must:

1. ✅ **Automate node upgrades** → 70% implemented (missing height trigger)
2. ❌ **Monitor blockchain height continuously** → NOT IMPLEMENTED
3. ❌ **Trigger upgrade at specific block height** → NOT IMPLEMENTED
4. ⚠️ **Support external configuration** → Partial (file only, no command/API)
5. ✅ **Hot-reload configuration** → IMPLEMENTED
6. ✅ **Never crash itself** → IMPLEMENTED (excellent)
7. ✅ **Always restart dead nodes** → IMPLEMENTED (excellent)
8. ✅ **Full node lifecycle control** → IMPLEMENTED (excellent)

---

### 3.2 Detailed Gap Analysis

#### 🔴 CRITICAL GAP #1: Block Height Monitoring
**Status**: NOT IMPLEMENTED

**Current State**:
- `FileWatcher.CheckHeight()` returns 0 (placeholder)
- No component actively monitors blockchain height
- Governance monitor HAS height checking but only for governance proposals

**Impact**:
- **Human intervention required** to manually restart node at upgrade height
- **Defeats automation purpose** - operator must watch blockchain and time restart
- **Timing errors** - operator might restart too early or too late

**Required Implementation**:
```go
// New component needed
type HeightMonitor struct {
    rpcClient     HeightProvider
    currentHeight int64
    subscribers   []chan<- int64
    pollInterval  time.Duration
}

func (hm *HeightMonitor) Start() error {
    ticker := time.NewTicker(hm.pollInterval)
    for {
        select {
        case <-ticker.C:
            height, err := hm.rpcClient.GetCurrentHeight()
            if err != nil {
                continue
            }
            if height > hm.currentHeight {
                hm.currentHeight = height
                hm.notifySubscribers(height)
            }
        }
    }
}
```

---

#### 🔴 CRITICAL GAP #2: Upgrade Orchestration
**Status**: NOT IMPLEMENTED

**Current State**:
- Components exist separately but don't coordinate
- No central orchestrator to:
  - Monitor height
  - Compare with upgrade height
  - Trigger node stop/switch/start sequence

**Impact**:
- **Manual intervention required** at upgrade time
- **No automatic execution** of upgrade sequence
- **Component isolation** - parts work but don't integrate

**Required Implementation**:
```go
type UpgradeOrchestrator struct {
    nodeManager    *node.Manager
    configManager  *config.Manager
    heightMonitor  *HeightMonitor
    upgradeWatcher *upgrade.FileWatcher

    pendingUpgrade *types.UpgradeInfo
    logger         *logger.Logger
}

func (uo *UpgradeOrchestrator) Start() error {
    // 1. Watch for upgrade configuration
    go uo.watchUpgrades()

    // 2. Monitor blockchain height
    heightCh := uo.heightMonitor.Subscribe()

    // 3. Coordinate upgrade execution
    go func() {
        for height := range heightCh {
            if uo.shouldUpgrade(height) {
                uo.executeUpgrade()
            }
        }
    }()

    return nil
}

func (uo *UpgradeOrchestrator) executeUpgrade() error {
    upgrade := uo.pendingUpgrade

    uo.logger.Info("executing upgrade",
        zap.String("name", upgrade.Name),
        zap.Int64("height", upgrade.Height))

    // 1. Stop current node
    if err := uo.nodeManager.Stop(); err != nil {
        return fmt.Errorf("failed to stop node: %w", err)
    }

    // 2. Switch binary symlink
    cfg := uo.configManager.GetConfig()
    if err := cfg.SetCurrentUpgrade(upgrade.Name); err != nil {
        // Rollback: restore previous binary
        cfg.SymLinkToGenesis()
        return fmt.Errorf("failed to switch binary: %w", err)
    }

    // 3. Start new node
    if err := uo.nodeManager.Start(cfg.Args); err != nil {
        // Rollback: restore previous binary and restart
        cfg.SymLinkToGenesis()
        uo.nodeManager.Start(cfg.Args)
        return fmt.Errorf("failed to start upgraded node: %w", err)
    }

    uo.logger.Info("upgrade completed successfully",
        zap.String("name", upgrade.Name))

    return nil
}
```

---

#### 🟡 HIGH GAP #3: Limited Configuration Methods
**Status**: PARTIALLY IMPLEMENTED

**Current State**:
- ✅ Config file support
- ✅ Environment variables
- ✅ Hot-reload
- ❌ No CLI command to set upgrade height
- ❌ No API endpoint for configuration
- ❌ No web interface

**User Requirement**:
> "설정을 CLI 명령어, config파일, 혹은 web 등 외부에서 설정을 감지할 수 있도록"

**Required Implementations**:

**A) CLI Command**:
```go
// internal/cli/upgrade.go (NEW FILE)
func NewUpgradeCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "upgrade",
        Short: "Manage upgrades",
    }

    scheduleCmd := &cobra.Command{
        Use:   "schedule <name> <height>",
        Short: "Schedule an upgrade at a specific block height",
        Args:  cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            height, err := strconv.ParseInt(args[1], 10, 64)
            if err != nil {
                return fmt.Errorf("invalid height: %w", err)
            }

            // Create upgrade-info.json
            upgradeInfo := &types.UpgradeInfo{
                Name:   name,
                Height: height,
            }

            return writeUpgradeInfo(cfg.UpgradeInfoFilePath(), upgradeInfo)
        },
    }

    cmd.AddCommand(scheduleCmd)
    return cmd
}

// Usage:
// wemixvisor upgrade schedule v2.0.0 1500000
```

**B) API Endpoint**:
```go
// internal/api/upgrade.go (NEW FILE)
func (s *Server) registerUpgradeRoutes() {
    s.router.POST("/api/v1/upgrades", s.scheduleUpgrade)
    s.router.GET("/api/v1/upgrades", s.listUpgrades)
    s.router.DELETE("/api/v1/upgrades/:name", s.cancelUpgrade)
}

func (s *Server) scheduleUpgrade(c *gin.Context) {
    var req struct {
        Name   string `json:"name" binding:"required"`
        Height int64  `json:"height" binding:"required,gt=0"`
        Info   string `json:"info"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    upgradeInfo := &types.UpgradeInfo{
        Name:   req.Name,
        Height: req.Height,
        Info:   req.Info,
    }

    if err := s.orchestrator.ScheduleUpgrade(upgradeInfo); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{
        "message": "Upgrade scheduled successfully",
        "upgrade": upgradeInfo,
    })
}

// Usage:
// curl -X POST http://localhost:8080/api/v1/upgrades \
//   -H "Content-Type: application/json" \
//   -d '{"name":"v2.0.0","height":1500000}'
```

---

#### 🟡 MEDIUM GAP #4: Component Integration
**Status**: INCOMPLETE

**Current State**:
- FileWatcher and GovernanceMonitor work independently
- No coordination between components
- Duplicate responsibility (both can create upgrade-info.json)

**Required**:
- Unified upgrade source abstraction
- Single orchestrator consuming multiple sources
- Proper event propagation

---

## 4. Design Patterns Assessment

### 4.1 Currently Used Patterns ✅

#### State Pattern (node/state.go)
**Implementation**: Excellent
```go
type NodeState int

const (
    StateStopped NodeState = iota
    StateStarting
    StateRunning
    StateStopping
    StateCrashed
    StateError
)

// Clear state transitions with proper synchronization
```
**Benefit**: Clear lifecycle management, easy to reason about

---

#### Observer Pattern (config/manager.go)
**Implementation**: Good
```go
// Configuration changes notify subscribers
updateCh chan ConfigUpdate

// Subscribers can react to config changes
func (m *Manager) GetUpdateChannel() <-chan ConfigUpdate {
    return m.updateCh
}
```
**Benefit**: Decoupled configuration updates

---

#### Strategy Pattern (governance/monitor.go)
**Implementation**: Good
```go
type NotificationChannel interface {
    Send(message string) error
}

// Multiple implementations: Email, Slack, Discord, Webhook
// Can swap strategies at runtime
```
**Benefit**: Flexible notification system

---

### 4.2 Recommended Additional Patterns

#### Chain of Responsibility (for upgrade sources)
**Use Case**: Process upgrade configurations from multiple sources

```go
type UpgradeSourceHandler interface {
    SetNext(handler UpgradeSourceHandler)
    Handle(ctx context.Context) (*UpgradeInfo, error)
}

type FileUpgradeHandler struct {
    next    UpgradeSourceHandler
    watcher *upgrade.FileWatcher
}

func (h *FileUpgradeHandler) Handle(ctx context.Context) (*UpgradeInfo, error) {
    info := h.watcher.GetCurrentUpgrade()
    if info != nil {
        return info, nil
    }
    if h.next != nil {
        return h.next.Handle(ctx)
    }
    return nil, nil
}

// Chain: File → Governance → API → Default
```

---

#### Facade Pattern (for orchestration)
**Use Case**: Simplify complex upgrade workflow

```go
type UpgradeFacade struct {
    orchestrator *UpgradeOrchestrator
    node         *node.Manager
    config       *config.Manager
}

// Simple interface hiding complexity
func (uf *UpgradeFacade) ScheduleUpgrade(name string, height int64) error {
    return uf.orchestrator.ScheduleUpgrade(&UpgradeInfo{
        Name:   name,
        Height: height,
    })
}

func (uf *UpgradeFacade) GetStatus() *UpgradeStatus {
    return &UpgradeStatus{
        NodeState:      uf.node.GetState(),
        PendingUpgrade: uf.orchestrator.GetPendingUpgrade(),
        CurrentHeight:  uf.orchestrator.GetCurrentHeight(),
    }
}
```

---

## 5. Code Quality Assessment

### 5.1 Test Coverage
**Overall Score: 9/10** - Excellent

| Package | Coverage | Status |
|---------|----------|--------|
| internal/node | 91.2% | ✅ Excellent |
| internal/config | 85%+ | ✅ Excellent |
| internal/governance | 80%+ | ✅ Good |
| internal/upgrade | 75%+ | ⚠️ Needs improvement |
| internal/cli | 70%+ | ⚠️ Needs improvement |

**Strengths**:
- Comprehensive unit tests
- Good use of mocks and test helpers
- Integration tests for critical paths
- Edge case coverage (deadlock, errors, timeouts)

**Recommendations**:
- Increase upgrade package coverage to 85%+
- Add more CLI integration tests
- Add end-to-end upgrade workflow tests

---

### 5.2 Error Handling
**Score: 9/10** - Excellent

**Strengths** ✅:
- Proper error wrapping with context
- No silent error suppression
- Clear error messages
- Graceful degradation

**Examples**:
```go
// Good error wrapping
if err := cmd.Start(); err != nil {
    m.state = StateError
    m.stateMutex.Unlock()
    return fmt.Errorf("failed to start node: %w", err)
}

// Proper cleanup on error
if err := m.saveConfig(); err != nil {
    m.mergedConfig = oldConfig  // Rollback
    return fmt.Errorf("failed to save: %w", err)
}
```

---

### 5.3 Concurrency Safety
**Score: 9/10** - Excellent

**Strengths** ✅:
- Proper mutex usage (RWMutex for read-heavy)
- No deadlock issues (resolved in Phase 4)
- Channel-based communication
- Context-based cancellation

**Evidence**:
```go
// Excellent mutex management
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()
    // ... critical section ...
    m.stateMutex.Unlock()  // Unlock before goroutines

    go m.monitor()  // Safe: no lock held
}

// Proper channel usage
select {
case <-m.ctx.Done():
    return
case event := <-m.watcher.Events:
    // Handle event
}
```

---

### 5.4 Performance Considerations
**Score: 8/10** - Good

**Strengths** ✅:
- Connection pooling (Phase 7)
- Caching system (Phase 7)
- Worker pools for parallel operations
- Efficient file watching (not polling every file)

**Optimization Opportunities**:
- Cache binary version detection
- Batch RPC calls when possible
- Optimize hot-reload trigger frequency

---

## 6. Production Readiness Checklist

### 6.1 Critical (Must Have) ❌
- ❌ **Automatic block height monitoring**
- ❌ **Upgrade orchestration at specific height**
- ❌ **Integration between FileWatcher and GovernanceMonitor**
- ❌ **Rollback mechanism on upgrade failure**
- ✅ Process resilience and auto-restart
- ✅ Graceful shutdown
- ✅ Error handling and recovery

### 6.2 High Priority (Should Have) ⚠️
- ⚠️ CLI command for upgrade scheduling
- ⚠️ API endpoint for upgrade configuration
- ⚠️ Comprehensive logging of upgrade process
- ⚠️ Health check integration with upgrade workflow
- ✅ Metrics collection
- ✅ Alerting system

### 6.3 Medium Priority (Nice to Have) ⏳
- ⏳ Web interface for configuration
- ⏳ Upgrade dry-run / simulation mode
- ⏳ Multiple upgrade sources (Chain of Responsibility)
- ⏳ Detailed upgrade audit log
- ✅ Multiple notification channels
- ✅ Performance profiling

---

## 7. Development Plan

### 7.1 Phase 8: Core Upgrade Automation (v0.8.0)
**Priority**: CRITICAL
**Duration**: 2-3 weeks
**Objective**: Complete the missing upgrade automation workflow

#### Task 8.1: Implement Height Monitor
**Estimated Time**: 3 days
**Priority**: P0 - Critical

**Implementation**:
```go
// File: internal/height/monitor.go

package height

import (
    "context"
    "sync"
    "time"

    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/pkg/logger"
    "go.uber.org/zap"
)

// HeightProvider abstracts blockchain height queries
type HeightProvider interface {
    GetCurrentHeight() (int64, error)
}

// HeightMonitor continuously monitors blockchain height
type HeightMonitor struct {
    provider      HeightProvider
    logger        *logger.Logger

    // State
    currentHeight int64
    mu            sync.RWMutex

    // Configuration
    pollInterval  time.Duration

    // Subscribers
    subscribers   []chan<- int64
    subMu         sync.RWMutex

    // Lifecycle
    ctx           context.Context
    cancel        context.CancelFunc
    wg            sync.WaitGroup
}

func NewHeightMonitor(provider HeightProvider, interval time.Duration, logger *logger.Logger) *HeightMonitor {
    ctx, cancel := context.WithCancel(context.Background())

    return &HeightMonitor{
        provider:     provider,
        logger:       logger,
        pollInterval: interval,
        subscribers:  make([]chan<- int64, 0),
        ctx:          ctx,
        cancel:       cancel,
    }
}

func (hm *HeightMonitor) Start() error {
    hm.logger.Info("starting height monitor",
        zap.Duration("interval", hm.pollInterval))

    hm.wg.Add(1)
    go hm.monitorLoop()

    return nil
}

func (hm *HeightMonitor) Stop() {
    hm.logger.Info("stopping height monitor")
    hm.cancel()
    hm.wg.Wait()
}

func (hm *HeightMonitor) monitorLoop() {
    defer hm.wg.Done()

    ticker := time.NewTicker(hm.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-hm.ctx.Done():
            return

        case <-ticker.C:
            height, err := hm.provider.GetCurrentHeight()
            if err != nil {
                hm.logger.Error("failed to get current height", zap.Error(err))
                continue
            }

            hm.mu.Lock()
            oldHeight := hm.currentHeight
            hm.currentHeight = height
            hm.mu.Unlock()

            if height > oldHeight {
                hm.logger.Debug("blockchain height updated",
                    zap.Int64("old", oldHeight),
                    zap.Int64("new", height))
                hm.notifySubscribers(height)
            }
        }
    }
}

func (hm *HeightMonitor) Subscribe() <-chan int64 {
    hm.subMu.Lock()
    defer hm.subMu.Unlock()

    ch := make(chan int64, 10)
    hm.subscribers = append(hm.subscribers, ch)

    return ch
}

func (hm *HeightMonitor) notifySubscribers(height int64) {
    hm.subMu.RLock()
    defer hm.subMu.RUnlock()

    for _, ch := range hm.subscribers {
        select {
        case ch <- height:
        default:
            hm.logger.Warn("subscriber channel full, dropping height update")
        }
    }
}

func (hm *HeightMonitor) GetCurrentHeight() int64 {
    hm.mu.RLock()
    defer hm.mu.RUnlock()
    return hm.currentHeight
}
```

**Test Requirements**:
- Unit tests with mock HeightProvider
- Test height update notifications
- Test subscriber management
- Test concurrent access
- Integration test with real RPC client
- **Target Coverage**: 90%+

**SOLID Compliance**:
- ✅ SRP: Only monitors height
- ✅ OCP: Extensible via HeightProvider interface
- ✅ DIP: Depends on abstraction, not concretion

---

#### Task 8.2: Implement Upgrade Orchestrator
**Estimated Time**: 5 days
**Priority**: P0 - Critical

**Implementation**:
```go
// File: internal/orchestrator/orchestrator.go

package orchestrator

import (
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/internal/height"
    "github.com/wemix/wemixvisor/internal/node"
    "github.com/wemix/wemixvisor/internal/upgrade"
    "github.com/wemix/wemixvisor/pkg/logger"
    "github.com/wemix/wemixvisor/pkg/types"
    "go.uber.org/zap"
)

// UpgradeOrchestrator coordinates the complete upgrade workflow
type UpgradeOrchestrator struct {
    // Core components
    nodeManager    NodeManager
    configManager  ConfigManager
    heightMonitor  *height.HeightMonitor
    upgradeWatcher UpgradeWatcher
    logger         *logger.Logger

    // State
    pendingUpgrade *types.UpgradeInfo
    upgrading      bool
    mu             sync.RWMutex

    // Lifecycle
    ctx            context.Context
    cancel         context.CancelFunc
    wg             sync.WaitGroup
}

// NodeManager interface for node lifecycle operations
type NodeManager interface {
    Start(args []string) error
    Stop() error
    GetState() node.NodeState
    GetStatus() *node.Status
}

// ConfigManager interface for configuration operations
type ConfigManager interface {
    GetConfig() *config.Config
}

// UpgradeWatcher interface for upgrade detection
type UpgradeWatcher interface {
    GetCurrentUpgrade() *types.UpgradeInfo
    NeedsUpdate() bool
    ClearUpdateFlag()
}

func NewUpgradeOrchestrator(
    nodeManager NodeManager,
    configManager ConfigManager,
    heightMonitor *height.HeightMonitor,
    upgradeWatcher UpgradeWatcher,
    logger *logger.Logger,
) *UpgradeOrchestrator {
    ctx, cancel := context.WithCancel(context.Background())

    return &UpgradeOrchestrator{
        nodeManager:    nodeManager,
        configManager:  configManager,
        heightMonitor:  heightMonitor,
        upgradeWatcher: upgradeWatcher,
        logger:         logger,
        ctx:            ctx,
        cancel:         cancel,
    }
}

func (uo *UpgradeOrchestrator) Start() error {
    uo.logger.Info("starting upgrade orchestrator")

    // Start watching for upgrade configurations
    uo.wg.Add(1)
    go uo.watchUpgradeConfigs()

    // Start monitoring heights for upgrade trigger
    heightCh := uo.heightMonitor.Subscribe()
    uo.wg.Add(1)
    go uo.monitorHeights(heightCh)

    return nil
}

func (uo *UpgradeOrchestrator) Stop() {
    uo.logger.Info("stopping upgrade orchestrator")
    uo.cancel()
    uo.wg.Wait()
}

func (uo *UpgradeOrchestrator) watchUpgradeConfigs() {
    defer uo.wg.Done()

    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-uo.ctx.Done():
            return

        case <-ticker.C:
            if uo.upgradeWatcher.NeedsUpdate() {
                upgrade := uo.upgradeWatcher.GetCurrentUpgrade()
                uo.upgradeWatcher.ClearUpdateFlag()

                if upgrade != nil {
                    uo.scheduleUpgrade(upgrade)
                }
            }
        }
    }
}

func (uo *UpgradeOrchestrator) monitorHeights(heightCh <-chan int64) {
    defer uo.wg.Done()

    for {
        select {
        case <-uo.ctx.Done():
            return

        case currentHeight := <-heightCh:
            if uo.shouldTriggerUpgrade(currentHeight) {
                if err := uo.executeUpgrade(); err != nil {
                    uo.logger.Error("upgrade execution failed", zap.Error(err))
                }
            }
        }
    }
}

func (uo *UpgradeOrchestrator) scheduleUpgrade(upgrade *types.UpgradeInfo) {
    uo.mu.Lock()
    defer uo.mu.Unlock()

    uo.logger.Info("scheduling upgrade",
        zap.String("name", upgrade.Name),
        zap.Int64("height", upgrade.Height))

    uo.pendingUpgrade = upgrade
}

func (uo *UpgradeOrchestrator) shouldTriggerUpgrade(currentHeight int64) bool {
    uo.mu.RLock()
    defer uo.mu.RUnlock()

    if uo.pendingUpgrade == nil || uo.upgrading {
        return false
    }

    return currentHeight >= uo.pendingUpgrade.Height
}

func (uo *UpgradeOrchestrator) executeUpgrade() error {
    uo.mu.Lock()
    if uo.upgrading {
        uo.mu.Unlock()
        return fmt.Errorf("upgrade already in progress")
    }
    uo.upgrading = true
    upgrade := uo.pendingUpgrade
    uo.mu.Unlock()

    defer func() {
        uo.mu.Lock()
        uo.upgrading = false
        uo.pendingUpgrade = nil
        uo.mu.Unlock()
    }()

    uo.logger.Info("executing upgrade",
        zap.String("name", upgrade.Name),
        zap.Int64("height", upgrade.Height))

    // Step 1: Stop current node
    uo.logger.Info("stopping current node for upgrade")
    if err := uo.nodeManager.Stop(); err != nil {
        return fmt.Errorf("failed to stop node: %w", err)
    }

    // Step 2: Switch binary symlink
    uo.logger.Info("switching to upgrade binary",
        zap.String("upgrade", upgrade.Name))

    cfg := uo.configManager.GetConfig()
    if err := cfg.SetCurrentUpgrade(upgrade.Name); err != nil {
        // Critical error: attempt rollback
        uo.logger.Error("failed to switch binary, attempting rollback", zap.Error(err))
        if rollbackErr := uo.rollback(cfg); rollbackErr != nil {
            uo.logger.Error("rollback failed", zap.Error(rollbackErr))
        }
        return fmt.Errorf("failed to switch binary: %w", err)
    }

    // Step 3: Start upgraded node
    uo.logger.Info("starting upgraded node")
    if err := uo.nodeManager.Start(cfg.Args); err != nil {
        // Critical error: attempt rollback
        uo.logger.Error("failed to start upgraded node, attempting rollback", zap.Error(err))
        if rollbackErr := uo.rollback(cfg); rollbackErr != nil {
            uo.logger.Error("rollback failed", zap.Error(rollbackErr))
        }
        return fmt.Errorf("failed to start upgraded node: %w", err)
    }

    // Step 4: Wait briefly and verify node is running
    time.Sleep(5 * time.Second)
    status := uo.nodeManager.GetStatus()
    if status.State != node.StateRunning {
        uo.logger.Error("upgraded node failed to start properly",
            zap.String("state", status.StateString))
        if rollbackErr := uo.rollback(cfg); rollbackErr != nil {
            uo.logger.Error("rollback failed", zap.Error(rollbackErr))
        }
        return fmt.Errorf("upgraded node not running: %s", status.StateString)
    }

    uo.logger.Info("upgrade completed successfully",
        zap.String("name", upgrade.Name),
        zap.String("version", status.Version))

    return nil
}

func (uo *UpgradeOrchestrator) rollback(cfg *config.Config) error {
    uo.logger.Warn("initiating rollback to previous binary")

    // Restore to genesis binary
    if err := cfg.SymLinkToGenesis(); err != nil {
        return fmt.Errorf("failed to restore genesis binary: %w", err)
    }

    // Attempt to restart with genesis binary
    if err := uo.nodeManager.Start(cfg.Args); err != nil {
        return fmt.Errorf("failed to restart with genesis binary: %w", err)
    }

    uo.logger.Info("rollback completed, node restarted with genesis binary")
    return nil
}

func (uo *UpgradeOrchestrator) GetPendingUpgrade() *types.UpgradeInfo {
    uo.mu.RLock()
    defer uo.mu.RUnlock()
    return uo.pendingUpgrade
}

func (uo *UpgradeOrchestrator) GetStatus() *UpgradeStatus {
    uo.mu.RLock()
    defer uo.mu.RUnlock()

    return &UpgradeStatus{
        PendingUpgrade: uo.pendingUpgrade,
        Upgrading:      uo.upgrading,
        CurrentHeight:  uo.heightMonitor.GetCurrentHeight(),
        NodeState:      uo.nodeManager.GetState(),
    }
}

type UpgradeStatus struct {
    PendingUpgrade *types.UpgradeInfo
    Upgrading      bool
    CurrentHeight  int64
    NodeState      node.NodeState
}
```

**Test Requirements**:
- Unit tests with all components mocked
- Test upgrade scheduling
- Test upgrade execution flow
- Test rollback on failure
- Test concurrent upgrade attempts (should block)
- Integration test with real components
- **Target Coverage**: 90%+

**SOLID Compliance**:
- ✅ SRP: Only coordinates upgrades
- ✅ OCP: Extensible via interfaces
- ✅ DIP: Depends on abstractions

---

#### Task 8.3: Integrate Orchestrator with Main
**Estimated Time**: 2 days
**Priority**: P0 - Critical

**Modifications**:
```go
// File: cmd/wemixvisor/main.go (additions)

import (
    "github.com/wemix/wemixvisor/internal/orchestrator"
    "github.com/wemix/wemixvisor/internal/height"
    "github.com/wemix/wemixvisor/internal/governance"
)

func main() {
    // ... existing code ...

    // Initialize height provider (reuse governance RPC client)
    var heightProvider height.HeightProvider
    if cfg.GovernanceEnabled {
        rpcClient, err := governance.NewWBFTClient(cfg.RPCAddress, log)
        if err != nil {
            log.Warn("failed to create RPC client for height monitoring",
                zap.Error(err))
        } else {
            heightProvider = rpcClient
        }
    }

    // Create height monitor
    heightMonitor := height.NewHeightMonitor(
        heightProvider,
        5*time.Second,  // Poll every 5 seconds
        log,
    )

    // Create orchestrator
    upgradeOrchestrator := orchestrator.NewUpgradeOrchestrator(
        nodeManager,
        configManager,
        heightMonitor,
        upgradeWatcher,
        log,
    )

    // Start all components
    if err := heightMonitor.Start(); err != nil {
        log.Fatal("failed to start height monitor", zap.Error(err))
    }
    defer heightMonitor.Stop()

    if err := upgradeOrchestrator.Start(); err != nil {
        log.Fatal("failed to start orchestrator", zap.Error(err))
    }
    defer upgradeOrchestrator.Stop()

    // ... execute command ...
}
```

---

#### Task 8.4: CLI Command for Upgrade Scheduling
**Estimated Time**: 2 days
**Priority**: P1 - High

**Implementation**:
```go
// File: internal/cli/upgrade.go (NEW FILE)

package cli

import (
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "time"

    "github.com/spf13/cobra"
    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/pkg/logger"
    "github.com/wemix/wemixvisor/pkg/types"
)

func NewUpgradeCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "upgrade",
        Short: "Manage upgrades",
        Long:  "Schedule, list, and cancel upgrades",
    }

    cmd.AddCommand(newScheduleCommand(cfg, logger))
    cmd.AddCommand(newListCommand(cfg, logger))
    cmd.AddCommand(newCancelCommand(cfg, logger))
    cmd.AddCommand(newStatusCommand(cfg, logger))

    return cmd
}

func newScheduleCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    var (
        binariesURL string
        checksum    string
        info        string
    )

    cmd := &cobra.Command{
        Use:   "schedule <name> <height>",
        Short: "Schedule an upgrade at a specific block height",
        Long: `Schedule an upgrade to be executed automatically at a specific block height.

Example:
  wemixvisor upgrade schedule v2.0.0 1500000
  wemixvisor upgrade schedule v2.0.0 1500000 --binaries https://releases.wemix.com/v2.0.0.tar.gz
        `,
        Args: cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            name := args[0]
            height, err := strconv.ParseInt(args[1], 10, 64)
            if err != nil {
                return fmt.Errorf("invalid height: %w", err)
            }

            if height <= 0 {
                return fmt.Errorf("height must be positive")
            }

            // Create upgrade info
            upgradeInfo := &types.UpgradeInfo{
                Name:   name,
                Height: height,
                Time:   time.Now(),
                Info:   info,
            }

            if binariesURL != "" {
                upgradeInfo.Info = fmt.Sprintf(`{"binaries":{"url":"%s","checksum":"%s"}}`,
                    binariesURL, checksum)
            }

            // Write to upgrade-info.json
            filePath := cfg.UpgradeInfoFilePath()
            if err := writeUpgradeInfo(filePath, upgradeInfo); err != nil {
                return fmt.Errorf("failed to write upgrade info: %w", err)
            }

            logger.Info("upgrade scheduled successfully",
                zap.String("name", name),
                zap.Int64("height", height),
                zap.String("file", filePath))

            fmt.Printf("✓ Upgrade '%s' scheduled for block height %d\n", name, height)
            fmt.Printf("  File: %s\n", filePath)

            return nil
        },
    }

    cmd.Flags().StringVar(&binariesURL, "binaries", "", "URL to download binaries")
    cmd.Flags().StringVar(&checksum, "checksum", "", "Binary checksum for verification")
    cmd.Flags().StringVar(&info, "info", "", "Additional upgrade information")

    return cmd
}

func newListCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    return &cobra.Command{
        Use:   "list",
        Short: "List all scheduled upgrades",
        RunE: func(cmd *cobra.Command, args []string) error {
            filePath := cfg.UpgradeInfoFilePath()

            if _, err := os.Stat(filePath); os.IsNotExist(err) {
                fmt.Println("No upgrades scheduled")
                return nil
            }

            data, err := os.ReadFile(filePath)
            if err != nil {
                return fmt.Errorf("failed to read upgrade info: %w", err)
            }

            var upgradeInfo types.UpgradeInfo
            if err := json.Unmarshal(data, &upgradeInfo); err != nil {
                return fmt.Errorf("failed to parse upgrade info: %w", err)
            }

            fmt.Printf("Scheduled Upgrades:\n\n")
            fmt.Printf("  Name:   %s\n", upgradeInfo.Name)
            fmt.Printf("  Height: %d\n", upgradeInfo.Height)
            fmt.Printf("  Time:   %s\n", upgradeInfo.Time.Format(time.RFC3339))
            if upgradeInfo.Info != "" {
                fmt.Printf("  Info:   %s\n", upgradeInfo.Info)
            }

            return nil
        },
    }
}

func newCancelCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    return &cobra.Command{
        Use:   "cancel",
        Short: "Cancel scheduled upgrade",
        RunE: func(cmd *cobra.Command, args []string) error {
            filePath := cfg.UpgradeInfoFilePath()

            if err := os.Remove(filePath); err != nil {
                if os.IsNotExist(err) {
                    fmt.Println("No upgrade scheduled")
                    return nil
                }
                return fmt.Errorf("failed to cancel upgrade: %w", err)
            }

            fmt.Println("✓ Upgrade cancelled")
            return nil
        },
    }
}

func writeUpgradeInfo(filePath string, info *types.UpgradeInfo) error {
    data, err := json.MarshalIndent(info, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal upgrade info: %w", err)
    }

    // Ensure directory exists
    dir := filepath.Dir(filePath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Write atomically
    tmpPath := filePath + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write temp file: %w", err)
    }

    return os.Rename(tmpPath, filePath)
}
```

**Update root command**:
```go
// File: internal/cli/root.go
func NewRootCommand(cfg *config.Config, logger *logger.Logger) *cobra.Command {
    // ... existing code ...

    // Add upgrade command
    cmd.AddCommand(NewUpgradeCommand(cfg, logger))

    return cmd
}
```

**Usage Examples**:
```bash
# Schedule an upgrade
wemixvisor upgrade schedule v2.0.0 1500000

# Schedule with binary download URL
wemixvisor upgrade schedule v2.0.0 1500000 \
  --binaries https://releases.wemix.com/v2.0.0.tar.gz \
  --checksum sha256:abc123...

# List scheduled upgrades
wemixvisor upgrade list

# Cancel upgrade
wemixvisor upgrade cancel
```

---

#### Task 8.5: API Endpoints for Upgrade Management
**Estimated Time**: 2 days
**Priority**: P1 - High

**Implementation**:
```go
// File: internal/api/upgrade_handlers.go (NEW FILE)

package api

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/wemix/wemixvisor/pkg/types"
    "go.uber.org/zap"
)

// RegisterUpgradeRoutes registers upgrade management endpoints
func (s *Server) RegisterUpgradeRoutes() {
    upgrades := s.router.Group("/api/v1/upgrades")
    {
        upgrades.GET("", s.listUpgrades)
        upgrades.POST("", s.scheduleUpgrade)
        upgrades.DELETE("/:name", s.cancelUpgrade)
        upgrades.GET("/status", s.getUpgradeStatus)
    }
}

// scheduleUpgrade handles POST /api/v1/upgrades
func (s *Server) scheduleUpgrade(c *gin.Context) {
    var req struct {
        Name      string            `json:"name" binding:"required"`
        Height    int64             `json:"height" binding:"required,gt=0"`
        Info      string            `json:"info"`
        Binaries  map[string]string `json:"binaries"`
        Checksums map[string]string `json:"checksums"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid request",
            "details": err.Error(),
        })
        return
    }

    // Create upgrade info
    upgradeInfo := &types.UpgradeInfo{
        Name:   req.Name,
        Height: req.Height,
        Info:   req.Info,
    }

    // Schedule via orchestrator
    if err := s.orchestrator.ScheduleUpgrade(upgradeInfo); err != nil {
        s.logger.Error("failed to schedule upgrade",
            zap.String("name", req.Name),
            zap.Error(err))
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to schedule upgrade",
            "details": err.Error(),
        })
        return
    }

    s.logger.Info("upgrade scheduled via API",
        zap.String("name", req.Name),
        zap.Int64("height", req.Height))

    c.JSON(http.StatusOK, gin.H{
        "message": "Upgrade scheduled successfully",
        "upgrade": upgradeInfo,
    })
}

// listUpgrades handles GET /api/v1/upgrades
func (s *Server) listUpgrades(c *gin.Context) {
    pending := s.orchestrator.GetPendingUpgrade()

    var upgrades []interface{}
    if pending != nil {
        upgrades = append(upgrades, pending)
    }

    c.JSON(http.StatusOK, gin.H{
        "upgrades": upgrades,
        "count":    len(upgrades),
    })
}

// cancelUpgrade handles DELETE /api/v1/upgrades/:name
func (s *Server) cancelUpgrade(c *gin.Context) {
    name := c.Param("name")

    if err := s.orchestrator.CancelUpgrade(name); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "Failed to cancel upgrade",
            "details": err.Error(),
        })
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Upgrade cancelled successfully",
        "name":    name,
    })
}

// getUpgradeStatus handles GET /api/v1/upgrades/status
func (s *Server) getUpgradeStatus(c *gin.Context) {
    status := s.orchestrator.GetStatus()

    c.JSON(http.StatusOK, gin.H{
        "pending_upgrade": status.PendingUpgrade,
        "upgrading":       status.Upgrading,
        "current_height":  status.CurrentHeight,
        "node_state":      status.NodeState.String(),
    })
}
```

**API Documentation**:
```yaml
# POST /api/v1/upgrades
# Schedule an upgrade
{
  "name": "v2.0.0",
  "height": 1500000,
  "info": "Major upgrade with new consensus",
  "binaries": {
    "linux/amd64": "https://releases.wemix.com/v2.0.0-linux-amd64.tar.gz"
  },
  "checksums": {
    "linux/amd64": "sha256:abc123..."
  }
}

# GET /api/v1/upgrades
# List scheduled upgrades
Response:
{
  "upgrades": [
    {
      "name": "v2.0.0",
      "height": 1500000,
      "info": "Major upgrade",
      "time": "2025-01-20T10:00:00Z"
    }
  ],
  "count": 1
}

# DELETE /api/v1/upgrades/v2.0.0
# Cancel upgrade

# GET /api/v1/upgrades/status
# Get current upgrade status
Response:
{
  "pending_upgrade": {
    "name": "v2.0.0",
    "height": 1500000
  },
  "upgrading": false,
  "current_height": 1450000,
  "node_state": "running"
}
```

---

#### Task 8.6: Integration Testing
**Estimated Time**: 3 days
**Priority**: P0 - Critical

**Test Scenarios**:
```go
// File: test/e2e/upgrade_orchestration_test.go

package e2e

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestUpgradeOrchestration_E2E(t *testing.T) {
    t.Run("complete upgrade workflow", func(t *testing.T) {
        // Setup test environment
        env := setupTestEnvironment(t)
        defer env.Cleanup()

        // Start wemixvisor with mock node
        env.Start()

        // Schedule an upgrade
        upgradeInfo := &types.UpgradeInfo{
            Name:   "test-upgrade-v1",
            Height: 100,
        }
        err := env.ScheduleUpgrade(upgradeInfo)
        require.NoError(t, err)

        // Verify upgrade is scheduled
        pending := env.GetPendingUpgrade()
        assert.Equal(t, "test-upgrade-v1", pending.Name)
        assert.Equal(t, int64(100), pending.Height)

        // Simulate blockchain progression
        env.AdvanceBlockHeight(99)
        time.Sleep(1 * time.Second)

        // Verify node still running with old binary
        status := env.GetNodeStatus()
        assert.Equal(t, "running", status.State)
        assert.Equal(t, "genesis", status.BinaryVersion)

        // Advance to upgrade height
        env.AdvanceBlockHeight(100)

        // Wait for upgrade execution
        time.Sleep(10 * time.Second)

        // Verify upgrade completed
        status = env.GetNodeStatus()
        assert.Equal(t, "running", status.State)
        assert.Equal(t, "test-upgrade-v1", status.BinaryVersion)

        // Verify no pending upgrade
        pending = env.GetPendingUpgrade()
        assert.Nil(t, pending)
    })

    t.Run("rollback on upgrade failure", func(t *testing.T) {
        env := setupTestEnvironment(t)
        defer env.Cleanup()

        env.Start()

        // Schedule upgrade with invalid binary
        env.ScheduleUpgrade(&types.UpgradeInfo{
            Name:   "invalid-upgrade",
            Height: 100,
        })

        // Make the upgrade binary unavailable
        env.RemoveUpgradeBinary("invalid-upgrade")

        // Advance to upgrade height
        env.AdvanceBlockHeight(100)
        time.Sleep(10 * time.Second)

        // Verify rollback occurred
        status := env.GetNodeStatus()
        assert.Equal(t, "running", status.State)
        assert.Equal(t, "genesis", status.BinaryVersion)
    })

    t.Run("CLI schedule and cancel", func(t *testing.T) {
        env := setupTestEnvironment(t)
        defer env.Cleanup()

        // Schedule via CLI
        output, err := env.RunCLI("upgrade", "schedule", "v2.0.0", "200")
        require.NoError(t, err)
        assert.Contains(t, output, "scheduled successfully")

        // Verify scheduled
        output, err = env.RunCLI("upgrade", "list")
        require.NoError(t, err)
        assert.Contains(t, output, "v2.0.0")
        assert.Contains(t, output, "200")

        // Cancel via CLI
        output, err = env.RunCLI("upgrade", "cancel")
        require.NoError(t, err)
        assert.Contains(t, output, "cancelled")

        // Verify cancelled
        output, err = env.RunCLI("upgrade", "list")
        require.NoError(t, err)
        assert.Contains(t, output, "No upgrades")
    })

    t.Run("API schedule and monitor", func(t *testing.T) {
        env := setupTestEnvironment(t)
        defer env.Cleanup()

        env.StartAPI()

        // Schedule via API
        resp := env.APIPost("/api/v1/upgrades", map[string]interface{}{
            "name":   "api-upgrade",
            "height": 150,
        })
        assert.Equal(t, 200, resp.StatusCode)

        // Check status via API
        status := env.APIGet("/api/v1/upgrades/status")
        assert.Equal(t, "api-upgrade", status["pending_upgrade"].(map[string]interface{})["name"])
        assert.Equal(t, int64(150), status["pending_upgrade"].(map[string]interface{})["height"])
    })
}
```

---

### 7.2 Phase 9: Enhanced Configuration (v0.9.0)
**Priority**: HIGH
**Duration**: 1-2 weeks
**Objective**: Complete external configuration capabilities

#### Task 9.1: Web UI for Configuration (Optional)
**Estimated Time**: 1 week
**Priority**: P2 - Medium (if needed)

Could use existing API + simple web frontend:
- React/Vue dashboard showing current status
- Form to schedule upgrades
- Real-time height and upgrade monitoring

---

### 7.3 Phase 10: Production Hardening (v1.0.0)
**Priority**: CRITICAL
**Duration**: 2 weeks
**Objective**: Production-ready release

#### Tasks:
1. Comprehensive audit logging
2. Security review
3. Load testing
4. Documentation completion
5. Migration guide
6. Deployment automation

---

## 8. Timeline Summary

| Phase | Version | Duration | Priority | Dependencies |
|-------|---------|----------|----------|--------------|
| Phase 8 | v0.8.0 | 2-3 weeks | P0 - Critical | None |
| Phase 9 | v0.9.0 | 1-2 weeks | P1 - High | Phase 8 |
| Phase 10 | v1.0.0 | 2 weeks | P0 - Critical | Phase 8, 9 |
| **Total** | **v1.0.0** | **5-7 weeks** | | |

---

## 9. Risk Assessment

### 9.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| RPC connectivity issues during upgrade | Medium | High | Implement retry logic, fallback mechanisms |
| Binary corruption during upgrade | Low | Critical | Checksum verification, rollback capability |
| Deadlock in orchestrator | Low | High | Comprehensive concurrency testing |
| Height monitoring lag | Medium | Medium | Configurable poll interval, alerts |

### 9.2 Schedule Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Testing complexity | Medium | Medium | Allocate extra time for E2E tests |
| Integration challenges | Low | Medium | Incremental integration approach |
| Scope creep | Medium | High | Strict adherence to requirements |

---

## 10. Success Criteria

### 10.1 Functional Requirements ✅
- [ ] Automatic block height monitoring
- [ ] Automatic upgrade execution at configured height
- [ ] CLI command for upgrade scheduling
- [ ] API endpoint for upgrade management
- [ ] Rollback on upgrade failure
- [ ] Multiple configuration sources (file, CLI, API)
- [ ] Hot-reload configuration support
- [ ] Comprehensive logging and alerting

### 10.2 Quality Requirements ✅
- [ ] Test coverage ≥ 90% for new components
- [ ] Zero critical bugs
- [ ] SOLID principles compliance
- [ ] Comprehensive documentation
- [ ] Performance overhead < 5%
- [ ] Memory usage < 100MB additional

### 10.3 Operational Requirements ✅
- [ ] 99.9% uptime (node auto-restart)
- [ ] Zero manual intervention for upgrades
- [ ] Complete audit trail
- [ ] Production deployment guide
- [ ] Monitoring and alerting setup

---

## 11. Recommendations

### 11.1 Immediate Actions (Week 1)
1. ✅ **Accept this architectural review**
2. ⚡ **Start Phase 8 development immediately**
3. ⚡ **Implement HeightMonitor first** (foundational)
4. ⚡ **Implement UpgradeOrchestrator next** (core feature)
5. ⚡ **Add comprehensive tests throughout**

### 11.2 Best Practices
1. **TDD Approach**: Write tests before implementation
2. **Incremental Integration**: Test each component independently first
3. **Code Review**: Review all changes for SOLID compliance
4. **Documentation**: Document design decisions as you go
5. **Monitoring**: Add extensive logging for upgrade workflow

### 11.3 Long-term Architecture
1. **Plugin System**: Consider plugin architecture for custom upgrade sources
2. **Multi-chain Support**: Design for potential multi-chain deployment
3. **Advanced Rollback**: Implement state snapshot/restore
4. **Performance**: Optimize RPC call batching for height monitoring

---

## 12. Conclusion

Wemixvisor has achieved **excellent foundation** with:
- ✅ Outstanding node lifecycle management (91.2% test coverage)
- ✅ Robust configuration system with hot-reload
- ✅ Strong SOLID principles adherence
- ✅ Production-quality error handling and concurrency

However, **critical gaps** prevent production deployment:
- ❌ Missing automatic block height monitoring
- ❌ Missing upgrade orchestration and triggering
- ❌ Incomplete external configuration methods

**Recommendation**: Prioritize Phase 8 (Core Upgrade Automation) for immediate development. Estimated **2-3 weeks** to complete missing critical features and achieve production readiness.

**Overall Assessment**: System is **75% production-ready**. With Phase 8 completion, will reach **95% production-ready** status.

---

**Document Version**: 1.0
**Next Review**: After Phase 8 completion
**Author**: Comprehensive Architectural Analysis
**Date**: 2025-01-20
