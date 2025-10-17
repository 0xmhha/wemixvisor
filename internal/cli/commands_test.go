package cli

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// MockManager implements a mock node manager for testing
type MockManager struct {
	state      node.NodeState
	pid        int
	version    string
	startErr   error
	stopErr    error
	restartErr error
	nodeArgs   []string
}

func (m *MockManager) Start(args []string) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.state = node.StateRunning
	m.pid = 1234
	m.nodeArgs = args
	return nil
}

func (m *MockManager) Stop() error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.state = node.StateStopped
	m.pid = 0
	return nil
}

func (m *MockManager) Restart() error {
	if m.restartErr != nil {
		return m.restartErr
	}
	m.state = node.StateRunning
	m.pid = 5678
	return nil
}

func (m *MockManager) GetState() node.NodeState {
	return m.state
}

func (m *MockManager) GetStatus() *node.Status {
	return &node.Status{
		State:        m.state,
		StateString:  m.state.String(),
		PID:          m.pid,
		Version:      m.version,
		Network:      "testnet",
		Binary:       "/path/to/binary",
		RestartCount: 0,
		StartTime:    time.Now(),
		Uptime:       time.Hour,
	}
}

func (m *MockManager) GetVersion() string {
	return m.version
}

func (m *MockManager) GetPID() int {
	return m.pid
}

func (m *MockManager) SetNodeArgs(args []string) {
	m.nodeArgs = args
}

func (m *MockManager) Wait() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (m *MockManager) IsHealthy() bool {
	return m.state == node.StateRunning
}

func (m *MockManager) Close() error {
	return nil
}

func TestCommandHandler_Execute(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		mockState node.NodeState
		wantErr   bool
	}{
		{
			name:      "valid start command",
			args:      []string{"start", "--datadir", "/data"},
			mockState: node.StateStopped,
			wantErr:   false,
		},
		{
			name:      "start when already running",
			args:      []string{"start"},
			mockState: node.StateRunning,
			wantErr:   true,
		},
		{
			name:      "valid stop command",
			args:      []string{"stop"},
			mockState: node.StateRunning,
			wantErr:   false,
		},
		{
			name:      "stop when not running",
			args:      []string{"stop"},
			mockState: node.StateStopped,
			wantErr:   true,
		},
		{
			name:      "valid restart command",
			args:      []string{"restart"},
			mockState: node.StateRunning,
			wantErr:   false,
		},
		{
			name:      "status command",
			args:      []string{"status"},
			mockState: node.StateRunning,
			wantErr:   false,
		},
		{
			name:      "version command",
			args:      []string{"version"},
			mockState: node.StateRunning,
			wantErr:   false,
		},
		{
			name:    "invalid command",
			args:    []string{"invalid"},
			wantErr: true,
		},
		{
			name:    "no command",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test logger
			testLogger := &logger.Logger{
				Logger: zap.NewNop(),
			}

			// Create test config
			cfg := config.DefaultConfig()
			cfg.Daemon = true // Run in daemon mode to avoid waiting for signals

			// Create mock manager
			mockManager := &MockManager{
				state:   tt.mockState,
				version: "v0.4.0",
			}

			// Create handler with mock
			handler := &CommandHandler{
				config:  cfg,
				logger:  testLogger,
				manager: mockManager,
				parser:  NewParser(),
			}

			// Execute command
			err := handler.Execute(tt.args)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandHandler_HandleStatus_JSONOutput(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config with JSON output
	cfg := config.DefaultConfig()
	cfg.JSONOutput = true

	// Create mock manager
	mockManager := &MockManager{
		state:   node.StateRunning,
		pid:     1234,
		version: "v0.4.0",
	}

	// Create handler
	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: mockManager,
		parser:  NewParser(),
	}

	// Execute status command
	err := handler.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Get status for validation
	status := mockManager.GetStatus()

	// Verify JSON marshaling works
	_, err = json.MarshalIndent(status, "", "  ")
	if err != nil {
		t.Errorf("Failed to marshal status to JSON: %v", err)
	}
}

func TestCommandHandler_ApplyOptions(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config
	cfg := config.DefaultConfig()
	originalHome := cfg.Home

	// Create handler
	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: &MockManager{},
		parser:  NewParser(),
	}

	// Test applying options
	handler.applyOptions(map[string]string{
		"--home":    "/custom/home",
		"--network": "mainnet",
		"--debug":   "true",
		"--json":    "true",
		"--quiet":   "true",
	})

	// Verify options were applied
	if cfg.Home != "/custom/home" {
		t.Errorf("Home not updated: got %s, want /custom/home", cfg.Home)
	}
	if cfg.Network != "mainnet" {
		t.Errorf("Network not updated: got %s, want mainnet", cfg.Network)
	}
	if !cfg.Debug {
		t.Error("Debug not set to true")
	}
	if !cfg.JSONOutput {
		t.Error("JSONOutput not set to true")
	}
	if !cfg.Quiet {
		t.Error("Quiet not set to true")
	}

	// Restore original home for other tests
	cfg.Home = originalHome
}

func TestCommandHandler_RestartWithNewArgs(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config
	cfg := config.DefaultConfig()
	cfg.Daemon = true

	// Create mock manager
	mockManager := &MockManager{
		state: node.StateRunning,
	}

	// Create handler
	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: mockManager,
		parser:  NewParser(),
	}

	// Execute restart with new arguments
	err := handler.Execute([]string{"restart", "--datadir", "/new/data", "--port", "8545"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Verify new arguments were set
	expectedArgs := []string{"--datadir", "/new/data", "--port", "8545"}
	for i, arg := range expectedArgs {
		if i >= len(mockManager.nodeArgs) || mockManager.nodeArgs[i] != arg {
			t.Errorf("Node args not updated correctly: got %v, want %v", mockManager.nodeArgs, expectedArgs)
			break
		}
	}
}