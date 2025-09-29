# Phase 4: 노드 라이프사이클 관리 상세 구현 계획

## 개요
Phase 4 (v0.4.0)는 wemixvisor가 실제 geth 노드를 완전히 관리할 수 있는 능력을 구현하는 핵심 단계입니다. 이 단계에서는 노드의 전체 생명주기를 관리하고, CLI 패스-스루를 통해 기존 geth 사용자들이 쉽게 전환할 수 있도록 합니다.

## 1. 주요 목표

### 1.1 핵심 기능
- ✅ 완전한 노드 라이프사이클 관리 (시작/중지/재시작)
- ✅ CLI 패스-스루로 100% geth 호환성
- ✅ 실시간 상태 모니터링 및 헬스 체크
- ✅ 자동 복구 메커니즘
- ✅ 구조화된 로깅 시스템

### 1.2 비기능적 목표
- **안정성**: 노드 장애 시 자동 복구
- **성능**: 오버헤드 최소화 (CPU <5%, 메모리 <100MB)
- **사용성**: 기존 geth 명령어와 완전 호환
- **테스트**: 85% 이상 코드 커버리지

## 2. 상세 구현 계획

### 2.1 Week 1: Enhanced Node Manager

#### 2.1.1 파일 구조
```
internal/
├── node/
│   ├── manager.go          # 핵심 노드 관리자
│   ├── manager_test.go     # 단위 테스트
│   ├── state.go            # 상태 관리
│   ├── state_test.go
│   ├── process.go          # 프로세스 제어
│   └── process_test.go
```

#### 2.1.2 Node Manager 구현
```go
// internal/node/manager.go
package node

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "sync"
    "time"

    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/pkg/logger"
)

// NodeState represents the current state of the node
type NodeState int

const (
    StateStopped NodeState = iota
    StateStarting
    StateRunning
    StateStopping
    StateUpgrading
    StateError
    StateCrashed
)

// Manager handles the lifecycle of a node process
type Manager struct {
    // Core components
    config      *config.Config
    logger      *logger.Logger

    // Process management
    cmd         *exec.Cmd
    process     *os.Process
    state       NodeState
    stateMutex  sync.RWMutex

    // CLI pass-through
    nodeArgs    []string
    nodeOptions map[string]string

    // Monitoring
    startTime   time.Time
    restartCount int
    maxRestarts  int

    // Channels for lifecycle management
    stopCh      chan struct{}
    restartCh   chan struct{}
    errorCh     chan error

    // Context for graceful shutdown
    ctx         context.Context
    cancel      context.CancelFunc
}

// NewManager creates a new node manager
func NewManager(cfg *config.Config, logger *logger.Logger) *Manager {
    ctx, cancel := context.WithCancel(context.Background())

    return &Manager{
        config:      cfg,
        logger:      logger,
        state:       StateStopped,
        nodeOptions: make(map[string]string),
        maxRestarts: cfg.MaxRestarts,
        stopCh:      make(chan struct{}),
        restartCh:   make(chan struct{}),
        errorCh:     make(chan error, 10),
        ctx:         ctx,
        cancel:      cancel,
    }
}

// Start starts the node with the given arguments
func (m *Manager) Start(args []string) error {
    m.stateMutex.Lock()
    defer m.stateMutex.Unlock()

    if m.state != StateStopped {
        return fmt.Errorf("node is not in stopped state: %v", m.state)
    }

    m.state = StateStarting
    m.nodeArgs = args

    // Build command
    cmdPath := m.config.CurrentBin()
    cmd := exec.CommandContext(m.ctx, cmdPath, args...)

    // Set environment variables
    cmd.Env = m.buildEnvironment()

    // Set working directory
    cmd.Dir = m.config.Home

    // Setup stdout/stderr
    cmd.Stdout = m.logger.Writer()
    cmd.Stderr = m.logger.Writer()

    // Start the process
    if err := cmd.Start(); err != nil {
        m.state = StateError
        return fmt.Errorf("failed to start node: %w", err)
    }

    m.cmd = cmd
    m.process = cmd.Process
    m.startTime = time.Now()
    m.state = StateRunning

    // Start monitoring goroutine
    go m.monitor()

    m.logger.Info("node started successfully",
        "pid", m.process.Pid,
        "args", args)

    return nil
}

// Stop stops the node gracefully
func (m *Manager) Stop() error {
    m.stateMutex.Lock()
    defer m.stateMutex.Unlock()

    if m.state != StateRunning {
        return fmt.Errorf("node is not running")
    }

    m.state = StateStopping

    // Send SIGTERM for graceful shutdown
    if err := m.process.Signal(os.Interrupt); err != nil {
        return fmt.Errorf("failed to send interrupt signal: %w", err)
    }

    // Wait for graceful shutdown or timeout
    done := make(chan error, 1)
    go func() {
        done <- m.cmd.Wait()
    }()

    select {
    case <-done:
        m.logger.Info("node stopped gracefully")
    case <-time.After(m.config.ShutdownGrace):
        m.logger.Warn("grace period exceeded, forcing shutdown")
        if err := m.process.Kill(); err != nil {
            return fmt.Errorf("failed to kill process: %w", err)
        }
    }

    m.state = StateStopped
    m.cmd = nil
    m.process = nil

    return nil
}

// Restart restarts the node with the same arguments
func (m *Manager) Restart() error {
    m.logger.Info("restarting node")

    // Save current args
    args := m.nodeArgs

    // Stop the node
    if err := m.Stop(); err != nil {
        return fmt.Errorf("failed to stop node: %w", err)
    }

    // Wait briefly
    time.Sleep(time.Second)

    // Start with same args
    if err := m.Start(args); err != nil {
        return fmt.Errorf("failed to start node: %w", err)
    }

    m.restartCount++
    return nil
}

// GetState returns the current node state
func (m *Manager) GetState() NodeState {
    m.stateMutex.RLock()
    defer m.stateMutex.RUnlock()
    return m.state
}

// GetStatus returns detailed status information
func (m *Manager) GetStatus() *Status {
    m.stateMutex.RLock()
    defer m.stateMutex.RUnlock()

    status := &Status{
        State:        m.state,
        StartTime:    m.startTime,
        RestartCount: m.restartCount,
    }

    if m.process != nil {
        status.PID = m.process.Pid
        status.Uptime = time.Since(m.startTime)
    }

    return status
}

// monitor monitors the node process and handles crashes
func (m *Manager) monitor() {
    if m.cmd == nil {
        return
    }

    // Wait for process to exit
    err := m.cmd.Wait()

    m.stateMutex.Lock()
    defer m.stateMutex.Unlock()

    // Check if this was an expected shutdown
    if m.state == StateStopping {
        return
    }

    // Process crashed unexpectedly
    m.state = StateCrashed
    m.logger.Error("node process crashed unexpectedly", "error", err)

    // Check if we should auto-restart
    if m.config.RestartOnFailure && m.restartCount < m.maxRestarts {
        m.logger.Info("attempting auto-restart",
            "attempt", m.restartCount+1,
            "max", m.maxRestarts)

        go func() {
            time.Sleep(time.Second * 5) // Wait before restart
            if err := m.Restart(); err != nil {
                m.logger.Error("auto-restart failed", "error", err)
                m.errorCh <- err
            }
        }()
    }
}

// buildEnvironment builds the environment variables for the node
func (m *Manager) buildEnvironment() []string {
    env := os.Environ()

    // Add custom environment variables
    env = append(env, fmt.Sprintf("WEMIX_HOME=%s", m.config.Home))
    env = append(env, fmt.Sprintf("WEMIX_NETWORK=%s", m.config.Network))

    return env
}
```

#### 2.1.3 State Management
```go
// internal/node/state.go
package node

import (
    "encoding/json"
    "time"
)

// Status represents the current status of the node
type Status struct {
    State        NodeState     `json:"state"`
    StateString  string        `json:"state_string"`
    PID          int           `json:"pid"`
    StartTime    time.Time     `json:"start_time"`
    Uptime       time.Duration `json:"uptime"`
    RestartCount int           `json:"restart_count"`
    Version      string        `json:"version"`
    Network      string        `json:"network"`
}

// String returns the string representation of NodeState
func (s NodeState) String() string {
    switch s {
    case StateStopped:
        return "stopped"
    case StateStarting:
        return "starting"
    case StateRunning:
        return "running"
    case StateStopping:
        return "stopping"
    case StateUpgrading:
        return "upgrading"
    case StateError:
        return "error"
    case StateCrashed:
        return "crashed"
    default:
        return "unknown"
    }
}

// MarshalJSON implements json.Marshaler
func (s Status) MarshalJSON() ([]byte, error) {
    type Alias Status
    return json.Marshal(&struct {
        *Alias
        StateString string `json:"state_string"`
        UptimeStr   string `json:"uptime_string"`
    }{
        Alias:       (*Alias)(&s),
        StateString: s.State.String(),
        UptimeStr:   s.Uptime.String(),
    })
}
```

### 2.2 Week 2: CLI Pass-Through System

#### 2.2.1 파일 구조
```
internal/
├── cli/
│   ├── passthrough.go      # CLI 패스-스루 시스템
│   ├── passthrough_test.go
│   ├── parser.go           # 인자 파싱
│   ├── parser_test.go
│   ├── validator.go        # 인자 검증
│   └── validator_test.go
```

#### 2.2.2 CLI Pass-Through 구현
```go
// internal/cli/passthrough.go
package cli

import (
    "fmt"
    "strings"

    "github.com/wemix/wemixvisor/internal/config"
)

// PassThrough handles CLI argument pass-through to the node
type PassThrough struct {
    parser    *ArgumentParser
    validator *ArgumentValidator
    builder   *CommandBuilder
}

// NewPassThrough creates a new CLI pass-through handler
func NewPassThrough() *PassThrough {
    return &PassThrough{
        parser:    NewArgumentParser(),
        validator: NewArgumentValidator(),
        builder:   NewCommandBuilder(),
    }
}

// ProcessArgs processes wemixvisor arguments and separates node arguments
func (p *PassThrough) ProcessArgs(args []string) (*ParsedArgs, error) {
    parsed := &ParsedArgs{
        WemixvisorArgs: make(map[string]string),
        NodeArgs:       []string{},
    }

    i := 0
    for i < len(args) {
        arg := args[i]

        // Check if this is a wemixvisor-specific flag
        if p.isWemixvisorFlag(arg) {
            // Process wemixvisor flag
            if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
                parsed.WemixvisorArgs[arg] = args[i+1]
                i += 2
            } else {
                parsed.WemixvisorArgs[arg] = "true"
                i++
            }
        } else {
            // Pass through to node
            parsed.NodeArgs = append(parsed.NodeArgs, arg)
            i++
        }
    }

    // Validate the parsed arguments
    if err := p.validator.Validate(parsed); err != nil {
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    return parsed, nil
}

// isWemixvisorFlag checks if a flag is wemixvisor-specific
func (p *PassThrough) isWemixvisorFlag(flag string) bool {
    wemixvisorFlags := []string{
        "--wemixvisor-config",
        "--auto-restart",
        "--max-restarts",
        "--health-check-interval",
    }

    for _, wf := range wemixvisorFlags {
        if strings.HasPrefix(flag, wf) {
            return true
        }
    }

    return false
}

// BuildNodeCommand builds the final node command with arguments
func (p *PassThrough) BuildNodeCommand(cfg *config.Config, parsed *ParsedArgs) []string {
    return p.builder.Build(cfg, parsed.NodeArgs)
}
```

#### 2.2.3 Argument Parser
```go
// internal/cli/parser.go
package cli

import (
    "strings"
)

// ParsedArgs represents parsed command-line arguments
type ParsedArgs struct {
    WemixvisorArgs map[string]string
    NodeArgs       []string
}

// ArgumentParser parses command-line arguments
type ArgumentParser struct {
    // Known geth flags for validation
    knownGethFlags map[string]bool
}

// NewArgumentParser creates a new argument parser
func NewArgumentParser() *ArgumentParser {
    return &ArgumentParser{
        knownGethFlags: initKnownGethFlags(),
    }
}

// initKnownGethFlags initializes the list of known geth flags
func initKnownGethFlags() map[string]bool {
    flags := []string{
        "--datadir", "--keystore", "--networkid", "--syncmode",
        "--gcmode", "--snapshot", "--txpool.locals", "--txpool.nolocals",
        "--txpool.journal", "--txpool.rejournal", "--txpool.pricelimit",
        "--txpool.pricebump", "--txpool.accountslots", "--txpool.globalslots",
        "--txpool.accountqueue", "--txpool.globalqueue", "--txpool.lifetime",
        "--cache", "--cache.database", "--cache.trie", "--cache.gc",
        "--cache.snapshot", "--cache.noprefetch", "--cache.preimages",
        "--port", "--bootnodes", "--maxpeers", "--maxpendpeers",
        "--nat", "--netrestrict", "--nodekey", "--nodekeyhex",
        "--http", "--http.addr", "--http.port", "--http.api",
        "--http.corsdomain", "--http.vhosts", "--ws", "--ws.addr",
        "--ws.port", "--ws.api", "--ws.origins", "--graphql",
        "--graphql.corsdomain", "--graphql.vhosts", "--authrpc.addr",
        "--authrpc.port", "--authrpc.vhosts", "--authrpc.jwtsecret",
        "--metrics", "--metrics.expensive", "--metrics.addr", "--metrics.port",
        "--miner.etherbase", "--miner.extradata", "--miner.gasprice",
        "--miner.gaslimit", "--miner.gastarget", "--miner.recommit",
    }

    m := make(map[string]bool)
    for _, flag := range flags {
        m[flag] = true
    }
    return m
}

// IsGethFlag checks if a flag is a known geth flag
func (p *ArgumentParser) IsGethFlag(flag string) bool {
    // Remove value if present (e.g., "--port=30303" -> "--port")
    if idx := strings.Index(flag, "="); idx != -1 {
        flag = flag[:idx]
    }

    return p.knownGethFlags[flag]
}

// Parse parses a command line argument
func (p *ArgumentParser) Parse(arg string) (key, value string, isFlag bool) {
    if !strings.HasPrefix(arg, "--") {
        return "", arg, false
    }

    isFlag = true

    // Check for key=value format
    if idx := strings.Index(arg, "="); idx != -1 {
        key = arg[:idx]
        value = arg[idx+1:]
    } else {
        key = arg
        value = ""
    }

    return key, value, isFlag
}
```

### 2.3 Week 3: Health Monitoring & CLI Commands

#### 2.3.1 Health Checker 구현
```go
// internal/monitor/health.go
package monitor

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/pkg/logger"
)

// HealthChecker monitors the health of the node
type HealthChecker struct {
    config        *config.Config
    logger        *logger.Logger
    httpClient    *http.Client
    rpcURL        string
    checkInterval time.Duration
    checks        []HealthCheck
}

// HealthCheck represents a single health check
type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(cfg *config.Config, logger *logger.Logger) *HealthChecker {
    return &HealthChecker{
        config:        cfg,
        logger:        logger,
        httpClient:    &http.Client{Timeout: 5 * time.Second},
        rpcURL:        fmt.Sprintf("http://localhost:%d", cfg.RPCPort),
        checkInterval: cfg.HealthCheckInterval,
        checks: []HealthCheck{
            &RPCHealthCheck{url: fmt.Sprintf("http://localhost:%d", cfg.RPCPort)},
            &PeerCountCheck{minPeers: 1},
            &SyncingCheck{},
        },
    }
}

// Start starts the health monitoring
func (h *HealthChecker) Start(ctx context.Context) <-chan HealthStatus {
    statusCh := make(chan HealthStatus, 1)

    go func() {
        defer close(statusCh)

        ticker := time.NewTicker(h.checkInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                status := h.performChecks(ctx)
                select {
                case statusCh <- status:
                case <-ctx.Done():
                    return
                }
            }
        }
    }()

    return statusCh
}

// performChecks performs all health checks
func (h *HealthChecker) performChecks(ctx context.Context) HealthStatus {
    status := HealthStatus{
        Healthy:   true,
        Timestamp: time.Now(),
        Checks:    make(map[string]CheckResult),
    }

    for _, check := range h.checks {
        result := CheckResult{
            Name: check.Name(),
        }

        if err := check.Check(ctx); err != nil {
            result.Healthy = false
            result.Error = err.Error()
            status.Healthy = false
        } else {
            result.Healthy = true
        }

        status.Checks[check.Name()] = result
    }

    return status
}

// RPCHealthCheck checks if the RPC endpoint is responsive
type RPCHealthCheck struct {
    url string
}

func (c *RPCHealthCheck) Name() string {
    return "rpc_endpoint"
}

func (c *RPCHealthCheck) Check(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, "POST", c.url, nil)
    if err != nil {
        return err
    }

    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("RPC endpoint unreachable: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("RPC endpoint returned status %d", resp.StatusCode)
    }

    return nil
}
```

#### 2.3.2 새로운 CLI 명령어 구현
```go
// cmd/wemixvisor/commands/start.go
package commands

import (
    "fmt"

    "github.com/spf13/cobra"
    "github.com/wemix/wemixvisor/internal/config"
    "github.com/wemix/wemixvisor/internal/node"
    "github.com/wemix/wemixvisor/internal/cli"
    "github.com/wemix/wemixvisor/pkg/logger"
)

// NewStartCommand creates the start command
func NewStartCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "start [flags] -- [node flags]",
        Short: "Start the node",
        Long:  `Start the node with the specified options. Any flags after -- are passed directly to the node.`,
        RunE:  runStart,
    }

    // Wemixvisor-specific flags
    cmd.Flags().Bool("auto-restart", true, "Automatically restart on failure")
    cmd.Flags().Int("max-restarts", 5, "Maximum number of automatic restarts")
    cmd.Flags().String("config", "", "Path to wemixvisor config file")

    return cmd
}

func runStart(cmd *cobra.Command, args []string) error {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        return fmt.Errorf("failed to load config: %w", err)
    }

    // Initialize logger
    logger, err := logger.New(cfg.Debug, cfg.ColorLogs, cfg.LogFile)
    if err != nil {
        return fmt.Errorf("failed to initialize logger: %w", err)
    }

    // Process CLI arguments
    passthrough := cli.NewPassThrough()
    parsed, err := passthrough.ProcessArgs(args)
    if err != nil {
        return fmt.Errorf("failed to process arguments: %w", err)
    }

    // Create node manager
    manager := node.NewManager(cfg, logger)

    // Build node command
    nodeArgs := passthrough.BuildNodeCommand(cfg, parsed)

    // Start the node
    if err := manager.Start(nodeArgs); err != nil {
        return fmt.Errorf("failed to start node: %w", err)
    }

    logger.Info("node started successfully")

    // Wait for shutdown signal
    waitForShutdown(manager)

    return nil
}

// NewStopCommand creates the stop command
func NewStopCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "stop",
        Short: "Stop the running node",
        RunE:  runStop,
    }
}

func runStop(cmd *cobra.Command, args []string) error {
    // Implementation similar to start
    // Read PID file, send stop signal, wait for graceful shutdown
    return nil
}

// NewStatusCommand creates the status command
func NewStatusCommand() *cobra.Command {
    return &cobra.Command{
        Use:   "status",
        Short: "Show node status",
        RunE:  runStatus,
    }
}

func runStatus(cmd *cobra.Command, args []string) error {
    // Connect to running wemixvisor, get status, display
    return nil
}

// NewLogsCommand creates the logs command
func NewLogsCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "logs",
        Short: "Show node logs",
        RunE:  runLogs,
    }

    cmd.Flags().BoolP("follow", "f", false, "Follow log output")
    cmd.Flags().IntP("lines", "n", 100, "Number of lines to show")

    return cmd
}
```

### 2.4 Week 4: Testing & Documentation

#### 2.4.1 단위 테스트 예시
```go
// internal/node/manager_test.go
package node

import (
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestManager_Start(t *testing.T) {
    cfg := &config.Config{
        Home: t.TempDir(),
        Name: "test-node",
    }

    logger := logger.NewTestLogger()
    manager := NewManager(cfg, logger)

    // Create mock binary
    mockBin := createMockBinary(t, cfg.Home)

    // Start the node
    err := manager.Start([]string{"--testnet"})
    require.NoError(t, err)

    // Check state
    assert.Equal(t, StateRunning, manager.GetState())

    // Check status
    status := manager.GetStatus()
    assert.NotZero(t, status.PID)
    assert.NotZero(t, status.StartTime)

    // Stop the node
    err = manager.Stop()
    require.NoError(t, err)

    assert.Equal(t, StateStopped, manager.GetState())
}

func TestManager_AutoRestart(t *testing.T) {
    cfg := &config.Config{
        Home:             t.TempDir(),
        Name:             "test-node",
        RestartOnFailure: true,
        MaxRestarts:      3,
    }

    logger := logger.NewTestLogger()
    manager := NewManager(cfg, logger)

    // Create mock binary that crashes after 1 second
    mockBin := createCrashingMockBinary(t, cfg.Home, time.Second)

    // Start the node
    err := manager.Start([]string{"--testnet"})
    require.NoError(t, err)

    // Wait for crash and auto-restart
    time.Sleep(3 * time.Second)

    // Check that it restarted
    assert.Equal(t, StateRunning, manager.GetState())
    assert.Equal(t, 1, manager.restartCount)
}
```

#### 2.4.2 통합 테스트
```go
// test/integration/node_lifecycle_test.go
package integration

import (
    "context"
    "testing"
    "time"
)

func TestFullNodeLifecycle(t *testing.T) {
    // Setup test environment
    env := setupTestEnvironment(t)
    defer env.Cleanup()

    // Initialize wemixvisor
    wv := env.InitWemixvisor()

    // Start node
    err := wv.Start([]string{
        "--datadir", env.DataDir,
        "--networkid", "1001",
        "--port", "30303",
        "--http",
        "--http.port", "8545",
    })
    require.NoError(t, err)

    // Wait for node to be ready
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    ready := wv.WaitForReady(ctx)
    assert.True(t, ready, "node did not become ready in time")

    // Check health
    health := wv.CheckHealth()
    assert.True(t, health.Healthy)

    // Restart node
    err = wv.Restart()
    require.NoError(t, err)

    // Check it's running again
    ready = wv.WaitForReady(ctx)
    assert.True(t, ready)

    // Stop node
    err = wv.Stop()
    require.NoError(t, err)

    // Verify it's stopped
    status := wv.GetStatus()
    assert.Equal(t, "stopped", status.State)
}
```

## 3. 구현 일정

### Week 1: Enhanced Node Manager
- [ ] Node Manager 구현 (manager.go)
- [ ] State 관리 구현 (state.go)
- [ ] Process 제어 구현 (process.go)
- [ ] 자동 재시작 메커니즘
- [ ] 단위 테스트 작성

### Week 2: CLI Pass-Through
- [ ] PassThrough 시스템 구현
- [ ] Argument Parser 구현
- [ ] Argument Validator 구현
- [ ] Command Builder 구현
- [ ] Geth 플래그 호환성 테스트

### Week 3: Health Monitoring & Commands
- [ ] Health Checker 구현
- [ ] RPC Health Check
- [ ] Peer Count Check
- [ ] Syncing Check
- [ ] CLI 명령어 구현 (start, stop, restart, status, logs)

### Week 4: Testing & Documentation
- [ ] 단위 테스트 완성 (85% 커버리지)
- [ ] 통합 테스트 작성
- [ ] E2E 테스트 작성
- [ ] 사용자 문서 작성
- [ ] API 문서 생성

## 4. 테스트 계획

### 4.1 단위 테스트 (85% 목표)
- Node Manager: 상태 전환, 에러 처리
- CLI PassThrough: 인자 파싱, 검증
- Health Checker: 각 health check 로직
- State Management: 상태 전환 유효성

### 4.2 통합 테스트
- 전체 노드 라이프사이클
- CLI 명령어 통합
- 자동 재시작 시나리오
- 업그레이드 호환성

### 4.3 E2E 테스트
- 실제 geth 바이너리와 테스트
- 네트워크 동기화 테스트
- RPC 상호작용 테스트
- 성능 벤치마크

## 5. 성공 기준

### 5.1 기능적 요구사항
- ✅ 노드 시작/중지/재시작 완벽 동작
- ✅ 기존 geth 명령어 100% 호환
- ✅ 자동 재시작 및 복구 동작
- ✅ 실시간 상태 모니터링

### 5.2 비기능적 요구사항
- ✅ CPU 오버헤드 < 5%
- ✅ 메모리 사용량 < 100MB
- ✅ 테스트 커버리지 >= 85%
- ✅ 모든 플랫폼 지원 (Linux/macOS, amd64/arm64)

## 6. 리스크 및 대응 방안

### 6.1 리스크
1. **Geth 버전 호환성**: 다양한 geth 버전의 CLI 차이
2. **프로세스 관리 복잡성**: OS별 프로세스 관리 차이
3. **성능 오버헤드**: 모니터링으로 인한 성능 저하
4. **테스트 환경**: 실제 블록체인 네트워크 테스트의 어려움

### 6.2 대응 방안
1. **호환성 매트릭스 관리**: 버전별 플래그 매핑 테이블
2. **OS 추상화 레이어**: OS별 구현 분리
3. **효율적 모니터링**: 샘플링 기반 모니터링
4. **Mock 네트워크**: 테스트용 경량 블록체인 구축

## 7. 다음 단계 준비

Phase 4 완료 후 Phase 5 (설정 관리 시스템)로 진행:
- 통합 설정 관리
- 네트워크별 템플릿
- 설정 검증 및 마이그레이션

Phase 4의 Node Manager와 CLI 시스템이 Phase 5의 기반이 되므로, 확장 가능한 구조로 설계하는 것이 중요합니다.