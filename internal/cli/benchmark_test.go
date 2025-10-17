package cli

import (
	"testing"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/internal/node"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// BenchmarkParser_Parse benchmarks argument parsing performance
func BenchmarkParser_Parse(b *testing.B) {
	parser := NewParser()
	args := []string{"start", "--datadir", "/tmp/data", "--port", "30303", "--debug"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_ParseComplexArgs benchmarks parsing complex arguments
func BenchmarkParser_ParseComplexArgs(b *testing.B) {
	parser := NewParser()
	args := []string{
		"start",
		"--home", "/custom/home",
		"--network", "testnet",
		"--debug",
		"--json",
		"--datadir", "/custom/data",
		"--port", "30303",
		"--syncmode", "full",
		"--cache", "1024",
		"--maxpeers", "50",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.Parse(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParser_ExtractGethCompatibleArgs benchmarks Geth args extraction
func BenchmarkParser_ExtractGethCompatibleArgs(b *testing.B) {
	parser := NewParser()
	args := []string{
		"start",
		"--home", "/custom/home",
		"--network", "testnet",
		"--debug",
		"--datadir", "/custom/data",
		"--port", "30303",
		"--syncmode", "full",
		"--cache", "1024",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parser.ExtractGethCompatibleArgs(args)
	}
}

// BenchmarkCommandHandler_Execute benchmarks command execution
func BenchmarkCommandHandler_Execute(b *testing.B) {
	cfg := config.DefaultConfig()
	logger, _ := logger.New(true, false, "")

	handler := NewCommandHandler(cfg, logger)
	handler.manager = &MockManager{
		state: node.StateStopped,
	}

	args := []string{"version"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := handler.Execute(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommandHandler_StatusCommand benchmarks status command performance
func BenchmarkCommandHandler_StatusCommand(b *testing.B) {
	cfg := config.DefaultConfig()
	logger, _ := logger.New(true, false, "")

	handler := NewCommandHandler(cfg, logger)
	handler.manager = &MockManager{
		state: node.StateRunning,
		pid:   1234,
	}

	args := []string{"status"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := handler.Execute(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommandHandler_StatusJSONCommand benchmarks JSON status command
func BenchmarkCommandHandler_StatusJSONCommand(b *testing.B) {
	cfg := config.DefaultConfig()
	cfg.JSONOutput = true
	logger, _ := logger.New(true, false, "")

	handler := NewCommandHandler(cfg, logger)
	handler.manager = &MockManager{
		state: node.StateRunning,
		pid:   1234,
	}

	args := []string{"status"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := handler.Execute(args)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidation benchmarks argument validation
func BenchmarkValidation(b *testing.B) {
	parser := NewParser()

	parsed := &ParsedArgs{
		Command:        "start",
		WemixvisorOpts: map[string]string{"--debug": "true"},
		NodeArgs:       []string{"--datadir", "/tmp"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := parser.validate(parsed)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBuildNodeArgs benchmarks node argument building
func BenchmarkBuildNodeArgs(b *testing.B) {
	parsed := &ParsedArgs{
		Command:        "start",
		WemixvisorOpts: map[string]string{"--debug": "true"},
		NodeArgs:       []string{"--syncmode", "full", "--cache", "1024"},
	}
	defaults := map[string]string{
		"--datadir": "/tmp/data",
		"--port":    "30303",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildNodeArgs(parsed, defaults)
	}
}

// BenchmarkIsValidCommand benchmarks command validation
func BenchmarkIsValidCommand(b *testing.B) {
	commands := []string{"start", "stop", "status", "restart", "version", "init", "run", "logs", "invalid"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cmd := range commands {
			_ = isValidCommand(cmd)
		}
	}
}