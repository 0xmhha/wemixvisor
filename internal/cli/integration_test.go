package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Integration tests for CLI functionality
func TestCLI_Integration_StartStopCycle(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config
	cfg := config.DefaultConfig()
	cfg.Daemon = true

	// Create mock manager
	mockManager := &MockManager{
		state:   node.StateStopped,
		version: "v0.4.0",
	}

	// Create handler
	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: mockManager,
		parser:  NewParser(),
	}

	// Test start with geth arguments
	err := handler.Execute([]string{"start", "--datadir", "/data", "--syncmode", "full", "--port", "30303"})
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Verify state
	if mockManager.state != node.StateRunning {
		t.Errorf("Expected state Running, got %v", mockManager.state)
	}

	// Verify arguments were passed
	expectedArgs := []string{"--datadir", "/data", "--syncmode", "full", "--port", "30303"}
	if len(mockManager.nodeArgs) != len(expectedArgs) {
		t.Errorf("Arguments mismatch: got %v, want %v", mockManager.nodeArgs, expectedArgs)
	}

	// Test status while running
	err = handler.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	// Test stop
	err = handler.Execute([]string{"stop"})
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify state
	if mockManager.state != node.StateStopped {
		t.Errorf("Expected state Stopped, got %v", mockManager.state)
	}
}

func TestCLI_Integration_MixedFlags(t *testing.T) {
	// Test parsing mixed wemixvisor and geth flags
	parser := NewParser()

	tests := []struct {
		name           string
		args           []string
		wantCommand    string
		wantWemixFlags map[string]string
		wantNodeArgs   []string
	}{
		{
			name:        "mixed flags with equals",
			args:        []string{"start", "--home=/custom", "--network=testnet", "--datadir=/data", "--port=8545"},
			wantCommand: "start",
			wantWemixFlags: map[string]string{
				"--home":    "/custom",
				"--network": "testnet",
			},
			wantNodeArgs: []string{"--datadir=/data", "--port=8545"},
		},
		{
			name:        "mixed flags with spaces",
			args:        []string{"start", "--home", "/custom", "--debug", "--datadir", "/data", "--syncmode", "full"},
			wantCommand: "start",
			wantWemixFlags: map[string]string{
				"--home":  "/custom",
				"--debug": "true",
			},
			wantNodeArgs: []string{"--datadir", "/data", "--syncmode", "full"},
		},
		{
			name:        "boolean and value flags",
			args:        []string{"start", "--json", "--quiet", "--home", "/test", "--verbosity", "5"},
			wantCommand: "start",
			wantWemixFlags: map[string]string{
				"--json":  "true",
				"--quiet": "true",
				"--home":  "/test",
			},
			wantNodeArgs: []string{"--verbosity", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.Command != tt.wantCommand {
				t.Errorf("Command = %v, want %v", parsed.Command, tt.wantCommand)
			}

			for key, wantVal := range tt.wantWemixFlags {
				if gotVal := parsed.WemixvisorOpts[key]; gotVal != wantVal {
					t.Errorf("WemixvisorOpts[%s] = %v, want %v", key, gotVal, wantVal)
				}
			}

			if len(parsed.NodeArgs) != len(tt.wantNodeArgs) {
				t.Errorf("NodeArgs = %v, want %v", parsed.NodeArgs, tt.wantNodeArgs)
			}
		})
	}
}

func TestCLI_Integration_LogsCommand(t *testing.T) {
	// Create temporary log file
	tmpFile, err := os.CreateTemp("", "test-logs-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test logs
	testLogs := []string{
		"2025-01-01 10:00:00 INFO Starting node",
		"2025-01-01 10:00:01 INFO Node initialized",
		"2025-01-01 10:00:02 INFO Syncing blockchain",
		"2025-01-01 10:00:03 WARN Low memory",
		"2025-01-01 10:00:04 ERROR Connection failed",
	}

	for _, log := range testLogs {
		fmt.Fprintln(tmpFile, log)
	}
	tmpFile.Close()

	// Test tailLogs
	t.Run("tail logs", func(t *testing.T) {
		// Capture output
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := tailLogs(tmpFile.Name(), 3)
		if err != nil {
			t.Errorf("tailLogs failed: %v", err)
		}

		w.Close()
		out, _ := io.ReadAll(r)
		os.Stdout = oldStdout

		output := string(out)
		// Should contain last 3 lines
		if !strings.Contains(output, "Syncing blockchain") {
			t.Error("Missing expected log line")
		}
		if !strings.Contains(output, "Low memory") {
			t.Error("Missing expected log line")
		}
		if !strings.Contains(output, "Connection failed") {
			t.Error("Missing expected log line")
		}
	})
}

func TestCLI_Integration_RunCommand(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config
	cfg := config.DefaultConfig()
	cfg.Daemon = false // Run in foreground

	// Create mock manager that immediately returns
	mockManager := &MockManager{
		state:   node.StateStopped,
		version: "v0.4.0",
	}

	// Create handler
	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: mockManager,
		parser:  NewParser(),
	}

	// Test run command
	t.Run("run command", func(t *testing.T) {
		// Check if run command is valid
		if !isValidCommand("run") {
			t.Skip("run command not yet fully implemented")
		}

		// Use goroutine to simulate signal after start
		go func() {
			time.Sleep(100 * time.Millisecond)
			// In real test, we would send signal, but here we just let it timeout
		}()

		// Since we can't easily test signal handling, just verify parsing
		parsed, err := handler.parser.Parse([]string{"run", "--datadir", "/data"})
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		if parsed.Command != "run" {
			t.Errorf("Expected command 'run', got %v", parsed.Command)
		}

		if len(parsed.NodeArgs) != 2 {
			t.Errorf("Expected 2 node args, got %d", len(parsed.NodeArgs))
		}
	})
}

func TestCLI_Integration_JSONStatus(t *testing.T) {
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

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute status command
	err := handler.Execute([]string{"status"})
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = oldStdout

	// Parse JSON output
	var status node.Status
	err = json.Unmarshal(out, &status)
	if err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify fields
	if status.PID != 1234 {
		t.Errorf("PID = %d, want 1234", status.PID)
	}
	if status.Version != "v0.4.0" {
		t.Errorf("Version = %s, want v0.4.0", status.Version)
	}
	if status.Network != "testnet" {
		t.Errorf("Network = %s, want testnet", status.Network)
	}
}

func TestCLI_Integration_ErrorCases(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Create test config
	cfg := config.DefaultConfig()
	cfg.Daemon = true

	tests := []struct {
		name        string
		setupMock   func() *MockManager
		args        []string
		wantErr     bool
		errContains string
	}{
		{
			name: "start when already running",
			setupMock: func() *MockManager {
				return &MockManager{
					state: node.StateRunning,
				}
			},
			args:        []string{"start"},
			wantErr:     true,
			errContains: "already running",
		},
		{
			name: "stop when not running",
			setupMock: func() *MockManager {
				return &MockManager{
					state: node.StateStopped,
				}
			},
			args:        []string{"stop"},
			wantErr:     true,
			errContains: "not running",
		},
		{
			name: "start with error",
			setupMock: func() *MockManager {
				return &MockManager{
					state:    node.StateStopped,
					startErr: fmt.Errorf("binary not found"),
				}
			},
			args:        []string{"start"},
			wantErr:     true,
			errContains: "binary not found",
		},
		{
			name: "invalid command",
			setupMock: func() *MockManager {
				return &MockManager{}
			},
			args:        []string{"invalid-command"},
			wantErr:     true,
			errContains: "unknown command",
		},
		{
			name: "stop with node arguments",
			setupMock: func() *MockManager {
				return &MockManager{
					state: node.StateRunning,
				}
			},
			args:        []string{"stop", "--datadir", "/data"},
			wantErr:     true,
			errContains: "does not accept node arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := tt.setupMock()
			handler := &CommandHandler{
				config:  cfg,
				logger:  testLogger,
				manager: mockManager,
				parser:  NewParser(),
			}

			err := handler.Execute(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestCLI_Integration_GethCompatibility(t *testing.T) {
	parser := NewParser()

	// Test extraction of geth-compatible arguments
	tests := []struct {
		name     string
		args     []string
		wantGeth []string
	}{
		{
			name:     "all geth args",
			args:     []string{"--datadir", "/data", "--syncmode", "full", "--port", "30303", "--maxpeers", "50"},
			wantGeth: []string{"--datadir", "/data", "--syncmode", "full", "--port", "30303", "--maxpeers", "50"},
		},
		{
			name:     "mixed with wemixvisor flags",
			args:     []string{"start", "--home", "/home", "--datadir", "/data", "--debug", "--port", "8545"},
			wantGeth: []string{"--datadir", "/data", "--port", "8545"},
		},
		{
			name:     "complex geth args",
			args:     []string{"--ws", "--ws.addr", "0.0.0.0", "--ws.port", "8546", "--ws.api", "eth,net,web3"},
			wantGeth: []string{"--ws", "--ws.addr", "0.0.0.0", "--ws.port", "8546", "--ws.api", "eth,net,web3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gethArgs := parser.ExtractGethCompatibleArgs(tt.args)

			if len(gethArgs) != len(tt.wantGeth) {
				t.Errorf("ExtractGethCompatibleArgs() = %v, want %v", gethArgs, tt.wantGeth)
				return
			}

			for i, arg := range tt.wantGeth {
				if i >= len(gethArgs) || gethArgs[i] != arg {
					t.Errorf("Arg[%d] = %v, want %v", i, gethArgs[i], arg)
				}
			}
		})
	}
}

func TestCLI_Integration_BuildNodeArgs(t *testing.T) {
	tests := []struct {
		name     string
		parsed   *ParsedArgs
		defaults map[string]string
		want     []string
	}{
		{
			name: "merge with defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data"},
			},
			defaults: map[string]string{
				"--syncmode":  "full",
				"--port":      "30303",
				"--maxpeers":  "25",
				"--verbosity": "3",
			},
			want: []string{"--datadir", "/data", "--maxpeers", "25", "--port", "30303", "--syncmode", "full", "--verbosity", "3"},
		},
		{
			name: "override defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data", "--port", "8545", "--syncmode", "light"},
			},
			defaults: map[string]string{
				"--syncmode": "full",
				"--port":     "30303",
				"--maxpeers": "25",
			},
			want: []string{"--datadir", "/data", "--port", "8545", "--syncmode", "light", "--maxpeers", "25"},
		},
		{
			name: "boolean defaults",
			parsed: &ParsedArgs{
				NodeArgs: []string{"--datadir", "/data"},
			},
			defaults: map[string]string{
				"--ws":       "",
				"--metrics":  "",
				"--syncmode": "full",
			},
			want: []string{"--datadir", "/data", "--metrics", "--syncmode", "full", "--ws"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildNodeArgs(tt.parsed, tt.defaults)

			if len(result) != len(tt.want) {
				t.Errorf("BuildNodeArgs() = %v, want %v", result, tt.want)
				return
			}

			// Check all expected args are present (order may differ due to map iteration)
			for _, wantArg := range tt.want {
				found := false
				for _, gotArg := range result {
					if gotArg == wantArg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Missing expected arg %v in result %v", wantArg, result)
				}
			}
		})
	}
}

func TestCLI_Integration_ConfigApply(t *testing.T) {
	// Create test logger
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	// Test configuration application through options
	cfg := config.DefaultConfig()

	handler := &CommandHandler{
		config:  cfg,
		logger:  testLogger,
		manager: &MockManager{},
		parser:  NewParser(),
	}

	// Apply various options
	options := map[string]string{
		"--home":    "/test/home",
		"--name":    "test-node",
		"--network": "testnet",
		"--debug":   "true",
		"--json":    "true",
		"--quiet":   "true",
	}

	handler.applyOptions(options)

	// Verify all options were applied
	if cfg.Home != "/test/home" {
		t.Errorf("Home = %v, want /test/home", cfg.Home)
	}
	if cfg.Name != "test-node" {
		t.Errorf("Name = %v, want test-node", cfg.Name)
	}
	if cfg.Network != "testnet" {
		t.Errorf("Network = %v, want testnet", cfg.Network)
	}
	if !cfg.Debug {
		t.Error("Debug should be true")
	}
	if !cfg.JSONOutput {
		t.Error("JSONOutput should be true")
	}
	if !cfg.Quiet {
		t.Error("Quiet should be true")
	}
}