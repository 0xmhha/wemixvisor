package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/height"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/internal/orchestrator"
	"github.com/wemix/wemixvisor/internal/upgrade"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

// TestPhase8_HeightMonitorIntegration tests HeightMonitor with mock provider
func TestPhase8_HeightMonitorIntegration(t *testing.T) {
	// Arrange
	mockProvider := &MockHeightProvider{heights: []int64{100, 200, 300}}
	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	monitor := height.NewHeightMonitor(mockProvider, 100*time.Millisecond, log)

	// Act
	require.NoError(t, monitor.Start())
	defer monitor.Stop()

	subscriber := monitor.Subscribe()
	var receivedHeights []int64

	// Collect heights with timeout
	timeout := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case height := <-subscriber:
			receivedHeights = append(receivedHeights, height)
		case <-timeout:
			t.Fatal("timeout waiting for heights")
		}
	}

	// Assert
	assert.Equal(t, []int64{100, 200, 300}, receivedHeights)
}

// TestPhase8_OrchestratorIntegration tests full orchestrator workflow
func TestPhase8_OrchestratorIntegration(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:               tmpDir,
		Name:               "wemixd",
		PollInterval:       100 * time.Millisecond,
		HeightPollInterval: 100 * time.Millisecond,
	}

	// Setup directories
	wemixvisorDir := filepath.Join(tmpDir, "wemixvisor")
	require.NoError(t, os.MkdirAll(wemixvisorDir, 0755))

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	// Create mock components
	mockNode := &MockNodeManager{
		state:  node.StateRunning,
		status: &node.Status{State: node.StateRunning, PID: 1234},
	}
	mockConfig := &MockConfigManager{cfg: cfg}
	mockProvider := &MockHeightProvider{heights: []int64{100, 200, 300, 1000000, 1000001}}

	// Create height monitor
	monitor := height.NewHeightMonitor(mockProvider, 100*time.Millisecond, log)
	require.NoError(t, monitor.Start())
	defer monitor.Stop()

	// Create file watcher
	fileWatcher := upgrade.NewFileWatcher(cfg, log)

	// Create orchestrator
	orch := orchestrator.NewUpgradeOrchestrator(
		mockNode,
		mockConfig,
		monitor,
		fileWatcher,
		log,
	)

	// Act
	require.NoError(t, orch.Start())
	defer orch.Stop()

	// Schedule upgrade
	upgradeInfo := &types.UpgradeInfo{
		Name:   "v1.2.0",
		Height: 1000000,
		Info:   map[string]interface{}{"description": "Test upgrade"},
	}

	require.NoError(t, orch.ScheduleUpgrade(upgradeInfo))

	// Verify scheduled
	status := orch.GetStatus()
	assert.NotNil(t, status.PendingUpgrade)
	assert.Equal(t, "v1.2.0", status.PendingUpgrade.Name)
	assert.Equal(t, int64(1000000), status.PendingUpgrade.Height)

	// Wait for upgrade execution (height 1000000 will trigger)
	time.Sleep(1 * time.Second)

	// Verify upgrade completed
	status = orch.GetStatus()
	assert.Nil(t, status.PendingUpgrade)
	assert.True(t, mockNode.stopCalled)
	assert.True(t, mockNode.startCalled)
}

// TestPhase8_FileWatcherIntegration tests UpgradeWatcher with real files
func TestPhase8_FileWatcherIntegration(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:               tmpDir,
		Name:               "wemixd",
		PollInterval:       100 * time.Millisecond,
		HeightPollInterval: 100 * time.Millisecond,
	}

	log, err := logger.New(false, false, "")
	require.NoError(t, err)

	fileWatcher := upgrade.NewFileWatcher(cfg, log)
	require.NoError(t, fileWatcher.Start())
	defer fileWatcher.Stop()

	// Act - No upgrade initially
	require.Nil(t, fileWatcher.GetCurrentUpgrade())
	require.False(t, fileWatcher.NeedsUpdate())

	// Create upgrade-info.json
	upgradeInfo := &types.UpgradeInfo{
		Name:   "v1.3.0",
		Height: 2000000,
		Info:   map[string]interface{}{"checksum": "abc123"},
	}

	upgradeInfoPath := cfg.UpgradeInfoFilePath()
	require.NoError(t, os.MkdirAll(filepath.Dir(upgradeInfoPath), 0755))
	require.NoError(t, types.WriteUpgradeInfoFile(upgradeInfoPath, upgradeInfo))

	// Wait for file watcher to detect changes (needs at least 2 poll intervals)
	time.Sleep(500 * time.Millisecond)

	// Assert
	current := fileWatcher.GetCurrentUpgrade()
	require.NotNil(t, current)
	assert.Equal(t, "v1.3.0", current.Name)
	assert.Equal(t, int64(2000000), current.Height)
	assert.True(t, fileWatcher.NeedsUpdate())

	// Clear update flag
	fileWatcher.ClearUpdateFlag()
	assert.False(t, fileWatcher.NeedsUpdate())
}

// MockHeightProvider for testing
type MockHeightProvider struct {
	heights []int64
	index   int
}

func (m *MockHeightProvider) GetCurrentHeight() (int64, error) {
	if m.index >= len(m.heights) {
		return m.heights[len(m.heights)-1], nil
	}
	height := m.heights[m.index]
	m.index++
	return height, nil
}

// MockNodeManager for testing
type MockNodeManager struct {
	state       node.NodeState
	status      *node.Status
	startCalled bool
	stopCalled  bool
	startErr    error
	stopErr     error
}

func (m *MockNodeManager) Start(args []string) error {
	m.startCalled = true
	if m.startErr != nil {
		return m.startErr
	}
	m.state = node.StateRunning
	return nil
}

func (m *MockNodeManager) Stop() error {
	m.stopCalled = true
	if m.stopErr != nil {
		return m.stopErr
	}
	m.state = node.StateStopped
	return nil
}

func (m *MockNodeManager) GetState() node.NodeState {
	return m.state
}

func (m *MockNodeManager) GetStatus() *node.Status {
	return m.status
}

// MockConfigManager for testing
type MockConfigManager struct {
	cfg *config.Config
}

func (m *MockConfigManager) GetConfig() *config.Config {
	return m.cfg
}

// Benchmark tests
func BenchmarkHeightMonitor_Polling(b *testing.B) {
	mockProvider := &MockHeightProvider{heights: make([]int64, b.N)}
	for i := 0; i < b.N; i++ {
		mockProvider.heights[i] = int64(i + 1)
	}

	log, _ := logger.New(false, false, "")
	monitor := height.NewHeightMonitor(mockProvider, 10*time.Millisecond, log)

	monitor.Start()
	defer monitor.Stop()

	subscriber := monitor.Subscribe()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		<-subscriber
	}
}

func BenchmarkOrchestrator_ScheduleUpgrade(b *testing.B) {
	cfg := &config.Config{Home: os.TempDir(), Name: "wemixd"}
	log, _ := logger.New(false, false, "")

	mockNode := &MockNodeManager{state: node.StateRunning}
	mockConfig := &MockConfigManager{cfg: cfg}
	mockProvider := &MockHeightProvider{heights: []int64{100}}
	monitor := height.NewHeightMonitor(mockProvider, 1*time.Second, log)
	fileWatcher := upgrade.NewFileWatcher(cfg, log)

	monitor.Start()
	defer monitor.Stop()

	orch := orchestrator.NewUpgradeOrchestrator(mockNode, mockConfig, monitor, fileWatcher, log)
	orch.Start()
	defer orch.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		upgradeInfo := &types.UpgradeInfo{
			Name:   fmt.Sprintf("v1.%d.0", i),
			Height: int64(1000000 + i),
		}
		orch.ScheduleUpgrade(upgradeInfo)
	}
}
