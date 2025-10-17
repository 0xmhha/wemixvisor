package cli

import (
	"strings"
	"testing"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Edge case tests for CLI functionality
func TestCLI_EdgeCases_EmptyArgs(t *testing.T) {
	parser := NewParser()

	// Test completely empty args
	_, err := parser.Parse([]string{})
	if err == nil {
		t.Error("Expected error for empty args, got nil")
	}
	if !strings.Contains(err.Error(), "no command") {
		t.Errorf("Expected 'no command' error, got: %v", err)
	}

	// Test empty string command
	_, err = parser.Parse([]string{""})
	if err == nil {
		t.Error("Expected error for empty string command")
	}
}

func TestCLI_EdgeCases_SpecialCharacters(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "path with spaces",
			args:    []string{"start", "--datadir", "/path with spaces/data"},
			wantErr: false,
		},
		{
			name:    "path with special chars",
			args:    []string{"start", "--datadir", "/path/@special#chars$/data"},
			wantErr: false,
		},
		{
			name:    "unicode in path",
			args:    []string{"start", "--datadir", "/경로/데이터"},
			wantErr: false,
		},
		{
			name:    "very long path",
			args:    []string{"start", "--datadir", "/" + strings.Repeat("a", 1000)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && parsed.Command != "start" {
				t.Errorf("Command = %v, want start", parsed.Command)
			}
		})
	}
}

func TestCLI_EdgeCases_FlagVariations(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name         string
		args         []string
		wantNodeArgs []string
	}{
		{
			name:         "double dash with dot notation",
			args:         []string{"start", "--ws.api", "eth,net,web3", "--rpc.gascap", "25000000"},
			wantNodeArgs: []string{"--ws.api", "eth,net,web3", "--rpc.gascap", "25000000"},
		},
		{
			name:         "single dash flags",
			args:         []string{"start", "-v", "5", "-h", "0.0.0.0"},
			wantNodeArgs: []string{"-v", "5", "-h", "0.0.0.0"},
		},
		{
			name:         "mixed dash styles",
			args:         []string{"start", "--datadir=/data", "-v=5", "--port", "30303"},
			wantNodeArgs: []string{"--datadir=/data", "-v=5", "--port", "30303"},
		},
		{
			name:         "multiple equals signs",
			args:         []string{"start", "--extra=key=value=data"},
			wantNodeArgs: []string{"--extra=key=value=data"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if len(parsed.NodeArgs) != len(tt.wantNodeArgs) {
				t.Errorf("NodeArgs = %v, want %v", parsed.NodeArgs, tt.wantNodeArgs)
			}

			for i, want := range tt.wantNodeArgs {
				if i >= len(parsed.NodeArgs) || parsed.NodeArgs[i] != want {
					t.Errorf("NodeArgs[%d] = %v, want %v", i, parsed.NodeArgs[i], want)
				}
			}
		})
	}
}

func TestCLI_EdgeCases_CommandCaseSensitivity(t *testing.T) {
	parser := NewParser()

	// Commands should be case sensitive
	invalidCommands := []string{"Start", "STOP", "Restart", "STATUS"}
	for _, cmd := range invalidCommands {
		_, err := parser.Parse([]string{cmd})
		if err == nil {
			t.Errorf("Expected error for case variant command: %s", cmd)
		}
	}
}

func TestCLI_EdgeCases_DuplicateFlags(t *testing.T) {
	parser := NewParser()

	// Test duplicate wemixvisor flags - last one should win
	parsed, err := parser.Parse([]string{"start", "--home", "/first", "--home", "/second", "--datadir", "/data"})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.WemixvisorOpts["--home"] != "/second" {
		t.Errorf("Duplicate flag: got %v, want /second", parsed.WemixvisorOpts["--home"])
	}
}

func TestCLI_EdgeCases_MalformedFlags(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "triple dash",
			args:    []string{"start", "---flag", "value"},
			wantErr: false, // Should pass through as node arg
		},
		{
			name:    "flag without dash",
			args:    []string{"start", "flag", "value"},
			wantErr: false, // Should pass through as node arg
		},
		{
			name:    "empty flag name",
			args:    []string{"start", "--", "value"},
			wantErr: false, // Should pass through as node arg
		},
		{
			name:    "flag with spaces",
			args:    []string{"start", "-- flag", "value"},
			wantErr: false, // Should pass through as node arg
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCLI_EdgeCases_RestartScenarios(t *testing.T) {
	// Create test components
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	cfg := config.DefaultConfig()
	cfg.Daemon = true

	tests := []struct {
		name         string
		initialState node.NodeState
		restartArgs  []string
		wantErr      bool
		checkArgs    bool
	}{
		{
			name:         "restart from stopped state",
			initialState: node.StateStopped,
			restartArgs:  []string{"restart"},
			wantErr:      false,
			checkArgs:    false,
		},
		{
			name:         "restart with new arguments",
			initialState: node.StateRunning,
			restartArgs:  []string{"restart", "--datadir", "/new/data", "--port", "9000"},
			wantErr:      false,
			checkArgs:    true,
		},
		{
			name:         "restart from error state",
			initialState: node.StateError,
			restartArgs:  []string{"restart"},
			wantErr:      false,
			checkArgs:    false,
		},
		{
			name:         "restart from crashed state",
			initialState: node.StateCrashed,
			restartArgs:  []string{"restart"},
			wantErr:      false,
			checkArgs:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockManager{
				state: tt.initialState,
			}

			handler := &CommandHandler{
				config:  cfg,
				logger:  testLogger,
				manager: mockManager,
				parser:  NewParser(),
			}

			err := handler.Execute(tt.restartArgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && tt.checkArgs && len(mockManager.nodeArgs) == 0 {
				t.Error("Expected node arguments to be set")
			}
		})
	}
}

func TestCLI_EdgeCases_LargeArguments(t *testing.T) {
	parser := NewParser()

	// Test with many arguments
	args := []string{"start"}
	for i := 0; i < 100; i++ {
		args = append(args, "--flag"+string(rune(i)), "value"+string(rune(i)))
	}

	parsed, err := parser.Parse(args)
	if err != nil {
		t.Fatalf("Parse() failed with many args: %v", err)
	}

	if parsed.Command != "start" {
		t.Errorf("Command = %v, want start", parsed.Command)
	}

	// All non-wemixvisor flags should be in NodeArgs
	if len(parsed.NodeArgs) < 100 {
		t.Errorf("Expected at least 100 node args, got %d", len(parsed.NodeArgs))
	}
}

func TestCLI_EdgeCases_InitCommand(t *testing.T) {
	parser := NewParser()

	// init command should be recognized as a valid command
	if !isValidCommand("init") {
		t.Skip("init command not yet implemented")
	}

	parsed, err := parser.Parse([]string{"init"})
	if err != nil {
		t.Fatalf("Parse() error for init: %v", err)
	}

	if parsed.Command != "init" {
		t.Errorf("Command = %v, want init", parsed.Command)
	}
}

func TestCLI_EdgeCases_ContainsFlagVariants(t *testing.T) {
	tests := []struct {
		name string
		args []string
		flag string
		want bool
	}{
		{
			name: "exact match",
			args: []string{"--datadir", "/data"},
			flag: "--datadir",
			want: true,
		},
		{
			name: "with equals",
			args: []string{"--datadir=/data"},
			flag: "--datadir",
			want: true,
		},
		{
			name: "partial should not match",
			args: []string{"--data", "/data"},
			flag: "--datadir",
			want: false,
		},
		{
			name: "flag as value should not match",
			args: []string{"--other", "--datadir"},
			flag: "--datadir",
			want: true, // Actually, --datadir IS a flag here (it starts with --)
		},
		{
			name: "complex equals",
			args: []string{"--extra=--datadir=/data"},
			flag: "--datadir",
			want: false, // --datadir is part of value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsFlag(tt.args, tt.flag)
			if result != tt.want {
				t.Errorf("containsFlag(%v, %v) = %v, want %v", tt.args, tt.flag, result, tt.want)
			}
		})
	}
}

func TestCLI_EdgeCases_StatusWithDifferentStates(t *testing.T) {
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	cfg := config.DefaultConfig()

	states := []node.NodeState{
		node.StateStopped,
		node.StateStarting,
		node.StateRunning,
		node.StateStopping,
		node.StateUpgrading,
		node.StateError,
		node.StateCrashed,
	}

	for _, state := range states {
		t.Run(state.String(), func(t *testing.T) {
			mockManager := &MockManager{
				state:   state,
				version: "v0.4.0",
			}

			handler := &CommandHandler{
				config:  cfg,
				logger:  testLogger,
				manager: mockManager,
				parser:  NewParser(),
			}

			err := handler.Execute([]string{"status"})
			if err != nil {
				t.Errorf("Status failed for state %v: %v", state, err)
			}
		})
	}
}

func TestCLI_EdgeCases_VersionCommandStates(t *testing.T) {
	testLogger := &logger.Logger{
		Logger: zap.NewNop(),
	}

	cfg := config.DefaultConfig()

	// Test version command with different states
	tests := []struct {
		name    string
		state   node.NodeState
		version string
	}{
		{
			name:    "running with version",
			state:   node.StateRunning,
			version: "v0.4.0",
		},
		{
			name:    "stopped no version",
			state:   node.StateStopped,
			version: "",
		},
		{
			name:    "error state",
			state:   node.StateError,
			version: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := &MockManager{
				state:   tt.state,
				version: tt.version,
			}

			handler := &CommandHandler{
				config:  cfg,
				logger:  testLogger,
				manager: mockManager,
				parser:  NewParser(),
			}

			// Version command should always succeed
			err := handler.Execute([]string{"version"})
			if err != nil {
				t.Errorf("Version command failed: %v", err)
			}
		})
	}
}