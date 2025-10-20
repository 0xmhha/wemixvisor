package orchestrator

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/height"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

// =============================================================================
// Mock Components
// =============================================================================

// MockNodeManager is a mock implementation of NodeManager for testing.
type MockNodeManager struct {
	mu            sync.Mutex
	state         node.NodeState
	startErr      error
	stopErr       error
	startCalls    int
	stopCalls     int
	startArgs     [][]string
	status        *node.Status
}

// NewMockNodeManager creates a new MockNodeManager with default values.
func NewMockNodeManager() *MockNodeManager {
	return &MockNodeManager{
		state: node.StateStopped,
		status: &node.Status{
			State:     node.StateStopped,
			PID:       0,
			StartTime: time.Now(),
		},
	}
}

// Start implements NodeManager interface.
func (m *MockNodeManager) Start(args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startCalls++
	m.startArgs = append(m.startArgs, args)

	if m.startErr != nil {
		return m.startErr
	}

	m.state = node.StateRunning
	m.status.State = node.StateRunning
	m.status.PID = 12345

	return nil
}

// Stop implements NodeManager interface.
func (m *MockNodeManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopCalls++

	if m.stopErr != nil {
		return m.stopErr
	}

	m.state = node.StateStopped
	m.status.State = node.StateStopped
	m.status.PID = 0

	return nil
}

// GetState implements NodeManager interface.
func (m *MockNodeManager) GetState() node.NodeState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// GetStatus implements NodeManager interface.
func (m *MockNodeManager) GetStatus() *node.Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// SetStartError sets the error to return from Start.
func (m *MockNodeManager) SetStartError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startErr = err
}

// SetStopError sets the error to return from Stop.
func (m *MockNodeManager) SetStopError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopErr = err
}

// GetStartCalls returns the number of times Start was called.
func (m *MockNodeManager) GetStartCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.startCalls
}

// GetStopCalls returns the number of times Stop was called.
func (m *MockNodeManager) GetStopCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stopCalls
}

// GetStartArgs returns all arguments passed to Start calls.
func (m *MockNodeManager) GetStartArgs() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to prevent race conditions
	result := make([][]string, len(m.startArgs))
	copy(result, m.startArgs)
	return result
}

// MockConfigManager is a mock implementation of ConfigManager for testing.
type MockConfigManager struct {
	mu     sync.Mutex
	config *config.Config
}

// NewMockConfigManager creates a new MockConfigManager with default config.
func NewMockConfigManager() *MockConfigManager {
	return &MockConfigManager{
		config: &config.Config{
			Home:        "/tmp/wemixvisor-test",
			Name:        "wemix",
			PollInterval: 1 * time.Second,
		},
	}
}

// GetConfig implements ConfigManager interface.
func (m *MockConfigManager) GetConfig() *config.Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.config
}

// SetConfig sets a new config.
func (m *MockConfigManager) SetConfig(cfg *config.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.config = cfg
}

// MockUpgradeWatcher is a mock implementation of UpgradeWatcher for testing.
type MockUpgradeWatcher struct {
	mu           sync.Mutex
	upgrade      *types.UpgradeInfo
	needsUpdate  bool
	clearCalls   int
}

// NewMockUpgradeWatcher creates a new MockUpgradeWatcher.
func NewMockUpgradeWatcher() *MockUpgradeWatcher {
	return &MockUpgradeWatcher{}
}

// GetCurrentUpgrade implements UpgradeWatcher interface.
func (m *MockUpgradeWatcher) GetCurrentUpgrade() *types.UpgradeInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.upgrade
}

// NeedsUpdate implements UpgradeWatcher interface.
func (m *MockUpgradeWatcher) NeedsUpdate() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.needsUpdate
}

// ClearUpdateFlag implements UpgradeWatcher interface.
func (m *MockUpgradeWatcher) ClearUpdateFlag() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.needsUpdate = false
	m.clearCalls++
}

// SetUpgrade sets a new upgrade and marks as needing update.
func (m *MockUpgradeWatcher) SetUpgrade(upgrade *types.UpgradeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.upgrade = upgrade
	m.needsUpdate = true
}

// GetClearCalls returns the number of times ClearUpdateFlag was called.
func (m *MockUpgradeWatcher) GetClearCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.clearCalls
}

// MockHeightProvider is a mock implementation for testing HeightMonitor.
type MockHeightProvider struct {
	mu     sync.Mutex
	height int64
	err    error
}

// NewMockHeightProvider creates a new MockHeightProvider.
func NewMockHeightProvider(height int64) *MockHeightProvider {
	return &MockHeightProvider{
		height: height,
	}
}

// GetCurrentHeight implements height.HeightProvider interface.
func (m *MockHeightProvider) GetCurrentHeight() (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.height, m.err
}

// SetHeight updates the mock height.
func (m *MockHeightProvider) SetHeight(height int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.height = height
}

// SetError sets the error to return.
func (m *MockHeightProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// Test helper function to create a test logger
func newTestLogger() *logger.Logger {
	log, err := logger.New(false, false, "")
	if err != nil {
		panic(err)
	}
	return log
}

// =============================================================================
// Test: Constructor
// =============================================================================

func TestNewUpgradeOrchestrator(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	// Act
	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Assert
	require.NotNil(t, orchestrator, "NewUpgradeOrchestrator should return non-nil orchestrator")
	assert.NotNil(t, orchestrator.nodeManager, "nodeManager should be set")
	assert.NotNil(t, orchestrator.configManager, "configManager should be set")
	assert.NotNil(t, orchestrator.heightMonitor, "heightMonitor should be set")
	assert.NotNil(t, orchestrator.upgradeWatcher, "upgradeWatcher should be set")
	assert.NotNil(t, orchestrator.logger, "logger should be set")
	assert.NotNil(t, orchestrator.ctx, "context should be initialized")
	assert.NotNil(t, orchestrator.cancel, "cancel function should be initialized")
	assert.False(t, orchestrator.upgrading, "initial upgrading should be false")
	assert.Nil(t, orchestrator.pendingUpgrade, "initial pendingUpgrade should be nil")
}

func TestNewUpgradeOrchestrator_WithNilDependencies(t *testing.T) {
	// Arrange
	log := newTestLogger()

	// Act
	orchestrator := NewUpgradeOrchestrator(nil, nil, nil, nil, log)

	// Assert
	require.NotNil(t, orchestrator, "NewUpgradeOrchestrator should return non-nil even with nil dependencies")
	// Note: The actual Start() should handle nil dependencies gracefully or panic with clear error
}

// =============================================================================
// Test: Lifecycle (Start/Stop)
// =============================================================================

func TestUpgradeOrchestrator_Start_Success(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Act
	err := orchestrator.Start()

	// Assert
	assert.NoError(t, err, "Start should succeed on first call")

	// Cleanup
	orchestrator.Stop()
}

func TestUpgradeOrchestrator_Start_AlreadyStarted(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Act
	err1 := orchestrator.Start()
	err2 := orchestrator.Start() // Second call should fail

	// Assert
	assert.NoError(t, err1, "first Start should succeed")
	assert.Error(t, err2, "second Start should return error")
	assert.Contains(t, err2.Error(), "already started", "error should indicate already started")

	// Cleanup
	orchestrator.Stop()
}

func TestUpgradeOrchestrator_Stop_Idempotent(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Act
	err := orchestrator.Start()
	require.NoError(t, err, "Start should succeed")

	// Stop multiple times
	orchestrator.Stop()
	orchestrator.Stop()
	orchestrator.Stop()

	// Assert - should not panic or cause issues
	// If we get here without panic, test passes
}

func TestUpgradeOrchestrator_Stop_WithoutStart(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Act & Assert - calling Stop without Start should not panic
	orchestrator.Stop()
}

// =============================================================================
// Test: Upgrade Scheduling
// =============================================================================

func TestUpgradeOrchestrator_ScheduleUpgrade_Success(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 2000,
	}

	// Act
	err := orchestrator.ScheduleUpgrade(upgrade)

	// Assert
	assert.NoError(t, err, "ScheduleUpgrade should succeed")

	status := orchestrator.GetStatus()
	assert.NotNil(t, status.PendingUpgrade, "pending upgrade should be set")
	assert.Equal(t, "v1.2.0", status.PendingUpgrade.Name, "upgrade name should match")
	assert.Equal(t, int64(2000), status.PendingUpgrade.Height, "upgrade height should match")
}

func TestUpgradeOrchestrator_ScheduleUpgrade_ReplacesPending(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	upgrade1 := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 2000,
	}

	upgrade2 := &types.UpgradeInfo{
		Name:   "v1.3.0",
		Height: 3000,
	}

	// Act
	err1 := orchestrator.ScheduleUpgrade(upgrade1)
	err2 := orchestrator.ScheduleUpgrade(upgrade2)

	// Assert
	assert.NoError(t, err1, "first ScheduleUpgrade should succeed")
	assert.NoError(t, err2, "second ScheduleUpgrade should succeed and replace first")

	status := orchestrator.GetStatus()
	assert.Equal(t, "v1.3.0", status.PendingUpgrade.Name, "second upgrade should replace first")
	assert.Equal(t, int64(3000), status.PendingUpgrade.Height, "height should be from second upgrade")
}

func TestUpgradeOrchestrator_ScheduleUpgrade_Concurrent(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 100*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	// Act - schedule upgrades concurrently
	var wg sync.WaitGroup
	const numConcurrent = 10

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			upgrade := &types.UpgradeInfo{
				Name:   "v1.2.0",
				Height: int64(2000 + index),
			}
			_ = orchestrator.ScheduleUpgrade(upgrade)
		}(i)
	}

	// Assert - should complete without deadlock or panic
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - all goroutines completed
	case <-time.After(2 * time.Second):
		t.Fatal("test timed out - possible deadlock in ScheduleUpgrade")
	}

	// One of the upgrades should be scheduled
	status := orchestrator.GetStatus()
	assert.NotNil(t, status.PendingUpgrade, "one upgrade should be scheduled")
}

// =============================================================================
// Test: Height-based Triggering
// =============================================================================

func TestUpgradeOrchestrator_TriggersAtExactHeight(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 50*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1500,
	}

	// Act
	err := orchestrator.ScheduleUpgrade(upgrade)
	require.NoError(t, err)

	err = heightMonitor.Start()
	require.NoError(t, err)
	defer heightMonitor.Stop()

	err = orchestrator.Start()
	require.NoError(t, err)
	defer orchestrator.Stop()

	// Increase height to trigger upgrade
	time.Sleep(100 * time.Millisecond)
	heightProvider.SetHeight(1500)

	// Wait for upgrade to trigger
	time.Sleep(300 * time.Millisecond)

	// Assert
	assert.GreaterOrEqual(t, nodeManager.GetStopCalls(), 1, "node should be stopped for upgrade")
	assert.GreaterOrEqual(t, nodeManager.GetStartCalls(), 1, "node should be restarted after upgrade")
}

func TestUpgradeOrchestrator_DoesNotTriggerBeforeHeight(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 50*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 2000,
	}

	// Act
	err := orchestrator.ScheduleUpgrade(upgrade)
	require.NoError(t, err)

	err = heightMonitor.Start()
	require.NoError(t, err)
	defer heightMonitor.Stop()

	err = orchestrator.Start()
	require.NoError(t, err)
	defer orchestrator.Stop()

	// Keep height below trigger
	time.Sleep(100 * time.Millisecond)
	heightProvider.SetHeight(1500)

	// Wait
	time.Sleep(200 * time.Millisecond)

	// Assert
	assert.Equal(t, 0, nodeManager.GetStopCalls(), "node should not be stopped before target height")
}

func TestUpgradeOrchestrator_TriggersOnlyOnce(t *testing.T) {
	// Arrange
	nodeManager := NewMockNodeManager()
	configManager := NewMockConfigManager()
	heightProvider := NewMockHeightProvider(1000)
	heightMonitor := height.NewHeightMonitor(heightProvider, 30*time.Millisecond, newTestLogger())
	upgradeWatcher := NewMockUpgradeWatcher()
	log := newTestLogger()

	orchestrator := NewUpgradeOrchestrator(
		nodeManager,
		configManager,
		heightMonitor,
		upgradeWatcher,
		log,
	)

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1500,
	}

	// Act
	err := orchestrator.ScheduleUpgrade(upgrade)
	require.NoError(t, err)

	err = heightMonitor.Start()
	require.NoError(t, err)
	defer heightMonitor.Stop()

	err = orchestrator.Start()
	require.NoError(t, err)
	defer orchestrator.Stop()

	// Trigger upgrade
	time.Sleep(50 * time.Millisecond)
	heightProvider.SetHeight(1500)

	// Wait for upgrade
	time.Sleep(200 * time.Millisecond)

	// Continue with higher heights
	heightProvider.SetHeight(1600)
	time.Sleep(150 * time.Millisecond)

	heightProvider.SetHeight(1700)
	time.Sleep(150 * time.Millisecond)

	// Assert - upgrade should only trigger once
	stopCalls := nodeManager.GetStopCalls()
	startCalls := nodeManager.GetStartCalls()

	assert.LessOrEqual(t, stopCalls, 1, "node should be stopped at most once for this upgrade")
	assert.LessOrEqual(t, startCalls, 1, "node should be restarted at most once for this upgrade")
}

// =============================================================================
// Test: Validation
// =============================================================================

func TestValidateUpgrade_Success(t *testing.T) {
	// Arrange
	orchestrator := &UpgradeOrchestrator{}

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 2000,
	}
	currentHeight := int64(1000)

	// Act
	err := orchestrator.validateUpgrade(upgrade, currentHeight)

	// Assert
	assert.NoError(t, err, "valid upgrade should pass validation")
}

func TestValidateUpgrade_NilUpgrade(t *testing.T) {
	// Arrange
	orchestrator := &UpgradeOrchestrator{}
	currentHeight := int64(1000)

	// Act
	err := orchestrator.validateUpgrade(nil, currentHeight)

	// Assert
	assert.Error(t, err, "nil upgrade should fail validation")
	assert.Contains(t, err.Error(), "nil", "error should mention nil")
}

func TestValidateUpgrade_EmptyName(t *testing.T) {
	// Arrange
	orchestrator := &UpgradeOrchestrator{}

	upgrade := &types.UpgradeInfo{
		Name:   "",
		Height: 2000,
	}
	currentHeight := int64(1000)

	// Act
	err := orchestrator.validateUpgrade(upgrade, currentHeight)

	// Assert
	assert.Error(t, err, "empty name should fail validation")
	assert.Contains(t, err.Error(), "name", "error should mention name")
}

func TestValidateUpgrade_InvalidHeight(t *testing.T) {
	// Arrange
	orchestrator := &UpgradeOrchestrator{}

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 0,
	}
	currentHeight := int64(1000)

	// Act
	err := orchestrator.validateUpgrade(upgrade, currentHeight)

	// Assert
	assert.Error(t, err, "zero height should fail validation")
	assert.Contains(t, err.Error(), "positive", "error should mention positive")
}

func TestValidateUpgrade_HeightAlreadyPassed(t *testing.T) {
	// Arrange
	orchestrator := &UpgradeOrchestrator{}

	upgrade := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1000,
	}
	currentHeight := int64(1500)

	// Act
	err := orchestrator.validateUpgrade(upgrade, currentHeight)

	// Assert
	assert.Error(t, err, "past height should fail validation")
	assert.Contains(t, err.Error(), "exceeded", "error should mention exceeded")
}
