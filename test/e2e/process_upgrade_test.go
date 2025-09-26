//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestProcessUpgradeE2E tests the complete upgrade flow end-to-end
func TestProcessUpgradeE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// Create test environment
	testDir := t.TempDir()
	daemonHome := filepath.Join(testDir, "daemon")
	daemonName := "test-daemon"

	// Set environment variables
	os.Setenv("DAEMON_HOME", daemonHome)
	os.Setenv("DAEMON_NAME", daemonName)
	os.Setenv("DAEMON_POLL_INTERVAL", "100ms")
	defer func() {
		os.Unsetenv("DAEMON_HOME")
		os.Unsetenv("DAEMON_NAME")
		os.Unsetenv("DAEMON_POLL_INTERVAL")
	}()

	// Build wemixvisor binary
	wemixvisorBin := filepath.Join(testDir, "wemixvisor")
	buildCmd := exec.Command("go", "build", "-o", wemixvisorBin, "../../cmd/wemixvisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build wemixvisor: %v", err)
	}

	// Create mock daemon binaries
	createMockDaemon(t, testDir, "genesis", "1.0.0")
	createMockDaemon(t, testDir, "v2.0.0", "2.0.0")

	// Initialize wemixvisor
	genesisBin := filepath.Join(testDir, "genesis", daemonName)
	initCmd := exec.Command(wemixvisorBin, "init", genesisBin)
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to initialize wemixvisor: %v\nOutput: %s", err, output)
	}

	// Prepare v2.0.0 upgrade
	upgradeDir := filepath.Join(daemonHome, "wemixvisor", "upgrades", "v2.0.0", "bin")
	os.MkdirAll(upgradeDir, 0755)
	upgradeBin := filepath.Join(testDir, "v2.0.0", daemonName)
	copyFile(t, upgradeBin, filepath.Join(upgradeDir, daemonName))

	// Start wemixvisor in background
	runCmd := exec.Command(wemixvisorBin, "run", "start")
	stdout, err := runCmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}

	if err := runCmd.Start(); err != nil {
		t.Fatalf("failed to start wemixvisor: %v", err)
	}

	// Wait for initial version to start
	waitForOutput(t, stdout, "Version: 1.0.0", 5*time.Second)

	// Trigger upgrade by creating upgrade-info.json
	dataDir := filepath.Join(daemonHome, "data")
	os.MkdirAll(dataDir, 0755)
	upgradeInfo := map[string]interface{}{
		"name":   "v2.0.0",
		"height": 100,
	}
	upgradeData, _ := json.Marshal(upgradeInfo)
	upgradeFile := filepath.Join(dataDir, "upgrade-info.json")
	if err := os.WriteFile(upgradeFile, upgradeData, 0644); err != nil {
		t.Fatalf("failed to write upgrade file: %v", err)
	}

	// Wait for upgrade to occur
	waitForOutput(t, stdout, "Version: 2.0.0", 10*time.Second)

	// Stop the process
	if err := runCmd.Process.Kill(); err != nil {
		t.Errorf("failed to kill process: %v", err)
	}

	t.Log("E2E test completed successfully")
}

// TestInitCommandE2E tests the init command
func TestInitCommandE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	testDir := t.TempDir()
	daemonHome := filepath.Join(testDir, "daemon")
	daemonName := "test-daemon"

	os.Setenv("DAEMON_HOME", daemonHome)
	os.Setenv("DAEMON_NAME", daemonName)
	defer func() {
		os.Unsetenv("DAEMON_HOME")
		os.Unsetenv("DAEMON_NAME")
	}()

	// Build wemixvisor
	wemixvisorBin := filepath.Join(testDir, "wemixvisor")
	buildCmd := exec.Command("go", "build", "-o", wemixvisorBin, "../../cmd/wemixvisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build wemixvisor: %v", err)
	}

	// Create mock genesis binary
	createMockDaemon(t, testDir, "genesis", "1.0.0")
	genesisBin := filepath.Join(testDir, "genesis", daemonName)

	// Run init command
	initCmd := exec.Command(wemixvisorBin, "init", genesisBin)
	output, err := initCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("init command failed: %v\nOutput: %s", err, output)
	}

	// Verify directory structure
	dirs := []string{
		filepath.Join(daemonHome, "wemixvisor"),
		filepath.Join(daemonHome, "wemixvisor", "genesis"),
		filepath.Join(daemonHome, "wemixvisor", "genesis", "bin"),
		filepath.Join(daemonHome, "wemixvisor", "upgrades"),
		filepath.Join(daemonHome, "data"),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("expected directory %s to exist", dir)
		}
	}

	// Verify genesis binary was copied
	targetBin := filepath.Join(daemonHome, "wemixvisor", "genesis", "bin", daemonName)
	if _, err := os.Stat(targetBin); os.IsNotExist(err) {
		t.Error("genesis binary was not copied")
	}

	// Verify current symlink
	currentLink := filepath.Join(daemonHome, "wemixvisor", "current")
	target, err := os.Readlink(currentLink)
	if err != nil {
		t.Errorf("failed to read current symlink: %v", err)
	}
	if target != "genesis" {
		t.Errorf("expected current to link to genesis, got %s", target)
	}
}

// TestVersionCommandE2E tests the version command
func TestVersionCommandE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	testDir := t.TempDir()

	// Build wemixvisor
	wemixvisorBin := filepath.Join(testDir, "wemixvisor")
	buildCmd := exec.Command("go", "build", "-o", wemixvisorBin, "../../cmd/wemixvisor")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("failed to build wemixvisor: %v", err)
	}

	// Run version command
	versionCmd := exec.Command(wemixvisorBin, "version")
	output, err := versionCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version command failed: %v\nOutput: %s", err, output)
	}

	outputStr := string(output)
	if outputStr == "" {
		t.Error("expected version output to be non-empty")
	}

	// Check for version components
	expectedComponents := []string{
		"Wemixvisor",
		"Git Commit:",
		"Build Date:",
	}

	for _, component := range expectedComponents {
		if !contains(outputStr, component) {
			t.Errorf("expected output to contain '%s'", component)
		}
	}
}

// Helper functions

func createMockDaemon(t *testing.T, dir, name, version string) {
	t.Helper()

	daemonDir := filepath.Join(dir, name)
	os.MkdirAll(daemonDir, 0755)

	daemonName := "test-daemon"
	daemonPath := filepath.Join(daemonDir, daemonName)

	// Create a simple script that prints version and runs
	script := fmt.Sprintf(`#!/bin/bash
echo "Mock daemon running"
echo "Version: %s"
while true; do
    sleep 1
done
`, version)

	if err := os.WriteFile(daemonPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock daemon: %v", err)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	sourceFile, err := os.Open(src)
	if err != nil {
		t.Fatalf("failed to open source file: %v", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		t.Fatalf("failed to create dest file: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		t.Fatalf("failed to copy file: %v", err)
	}

	// Make executable
	if err := os.Chmod(dst, 0755); err != nil {
		t.Fatalf("failed to chmod file: %v", err)
	}
}

func waitForOutput(t *testing.T, reader io.Reader, expected string, timeout time.Duration) {
	t.Helper()

	done := make(chan bool)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := reader.Read(buf)
			if err != nil {
				return
			}
			output := string(buf[:n])
			t.Logf("Output: %s", output)
			if contains(output, expected) {
				done <- true
				return
			}
		}
	}()

	select {
	case <-done:
		t.Logf("Found expected output: %s", expected)
	case <-time.After(timeout):
		t.Fatalf("timeout waiting for output: %s", expected)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}