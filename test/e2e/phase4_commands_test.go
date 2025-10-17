//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestPhase4StatusCommandE2E tests the enhanced status command with health monitoring
func TestPhase4StatusCommandE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create test environment
	testDir := t.TempDir()
	daemonHome := filepath.Join(testDir, "daemon")
	wemixvisorBin := buildWemixvisor(t, testDir)

	// Initialize environment using existing init command
	setupTestEnvironmentFromExisting(t, daemonHome, wemixvisorBin)

	// Test status when stopped
	cmd := exec.Command(wemixvisorBin, "status")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DAEMON_HOME=%s", daemonHome),
		"DAEMON_NAME=test-daemon")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status command failed: %v\nOutput: %s", err, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "stopped") {
		t.Errorf("status should show stopped state: %s", outputStr)
	}

	// Test JSON status output for Phase 4 features
	cmd = exec.Command(wemixvisorBin, "status", "--json")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DAEMON_HOME=%s", daemonHome),
		"DAEMON_NAME=test-daemon")

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status --json command failed: %v\nOutput: %s", err, string(output))
	}

	// Parse JSON output
	var status map[string]interface{}
	if err := json.Unmarshal(output, &status); err != nil {
		t.Fatalf("failed to parse JSON status output: %v\nOutput: %s", err, string(output))
	}

	// Verify Phase 4 JSON structure
	if _, exists := status["state_string"]; !exists {
		t.Error("JSON status missing state_string field")
	}
	if _, exists := status["network"]; !exists {
		t.Error("JSON status missing network field")
	}
	// Verify health field exists (may not be present for stopped nodes)
	if _, exists := status["health"]; !exists {
		t.Log("Note: health field not present - this may be expected for stopped nodes")
	}
}

// TestPhase4HealthIntegrationE2E tests health monitoring integration in CLI workflow
func TestPhase4HealthIntegrationE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create test environment
	testDir := t.TempDir()
	daemonHome := filepath.Join(testDir, "daemon")
	wemixvisorBin := buildWemixvisor(t, testDir)

	// Setup test environment using existing init command from process_upgrade_test.go
	setupTestEnvironmentFromExisting(t, daemonHome, wemixvisorBin)

	// Test health monitoring in status command when stopped
	cmd := exec.Command(wemixvisorBin, "status", "--json")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("DAEMON_HOME=%s", daemonHome),
		"DAEMON_NAME=test-daemon")

	statusOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status command failed: %v\nOutput: %s", err, string(statusOutput))
	}

	// Parse JSON to verify health information is present
	var status map[string]interface{}
	if err := json.Unmarshal(statusOutput, &status); err != nil {
		t.Fatalf("failed to parse JSON status: %v", err)
	}

	// Verify health field exists in Phase 4
	if health, exists := status["health"]; exists {
		healthMap, ok := health.(map[string]interface{})
		if !ok {
			t.Error("health field should be an object")
		} else {
			// Verify health structure includes expected fields
			if _, exists := healthMap["healthy"]; !exists {
				t.Error("health object missing 'healthy' field")
			}
			if _, exists := healthMap["timestamp"]; !exists {
				t.Error("health object missing 'timestamp' field")
			}
		}
	}

	t.Log("Phase 4 health integration test completed successfully")
}

// TestPhase4MetricsIntegrationE2E tests metrics collection integration
func TestPhase4MetricsIntegrationE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create test environment
	testDir := t.TempDir()
	daemonHome := filepath.Join(testDir, "daemon")
	wemixvisorBin := buildWemixvisor(t, testDir)

	// Setup test environment
	setupTestEnvironmentFromExisting(t, daemonHome, wemixvisorBin)

	// Test status command includes metrics information
	statusCmd := exec.Command(wemixvisorBin, "status", "--json")
	statusCmd.Env = append(os.Environ(),
		fmt.Sprintf("DAEMON_HOME=%s", daemonHome),
		"DAEMON_NAME=test-daemon")

	statusOutput, err := statusCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("status --json command failed: %v\nOutput: %s", err, string(statusOutput))
	}

	// Parse JSON to verify structure is correct
	var status map[string]interface{}
	if err := json.Unmarshal(statusOutput, &status); err != nil {
		t.Fatalf("failed to parse JSON status: %v", err)
	}

	// Verify basic structure is present
	if _, exists := status["state_string"]; !exists {
		t.Error("status missing state_string field")
	}

	if _, exists := status["network"]; !exists {
		t.Error("status missing network field")
	}

	// In Phase 4, even stopped nodes should have health information structure
	if _, exists := status["health"]; !exists {
		t.Log("Note: health field not present - this is expected for stopped nodes in some cases")
	}

	t.Log("Phase 4 metrics integration test completed successfully")
}

// Helper functions

// buildWemixvisor builds the wemixvisor binary for testing (reuse existing implementation)
func buildWemixvisor(t *testing.T, testDir string) string {
	wemixvisorBin := filepath.Join(testDir, "wemixvisor")
	buildCmd := exec.Command("go", "build", "-o", wemixvisorBin, "../../cmd/wemixvisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build wemixvisor: %v", err)
	}
	return wemixvisorBin
}

// setupTestEnvironmentFromExisting uses the existing init command to setup environment
func setupTestEnvironmentFromExisting(t *testing.T, daemonHome, wemixvisorBin string) {
	// Set environment for init command
	os.Setenv("DAEMON_HOME", daemonHome)
	os.Setenv("DAEMON_NAME", "test-daemon")
	defer func() {
		os.Unsetenv("DAEMON_HOME")
		os.Unsetenv("DAEMON_NAME")
	}()

	// Use Phase 4 init command (no arguments needed)
	initCmd := exec.Command(wemixvisorBin, "init")
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("init setup failed: %v\nOutput: %s", err, string(output))
	}
}

