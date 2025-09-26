package upgrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"github.com/wemix/wemixvisor/pkg/types"
)

func TestNewFileWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 100 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")

	watcher := NewFileWatcher(cfg, logger)
	if watcher == nil {
		t.Fatal("expected watcher to be non-nil")
	}

	if watcher.interval != cfg.PollInterval {
		t.Errorf("expected interval %v, got %v", cfg.PollInterval, watcher.interval)
	}

	expectedFile := filepath.Join(tmpDir, "data", "upgrade-info.json")
	if watcher.filename != expectedFile {
		t.Errorf("expected filename %s, got %s", expectedFile, watcher.filename)
	}
}

func TestFileWatcherStartStop(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 50 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")
	watcher := NewFileWatcher(cfg, logger)

	// Start watcher
	err := watcher.Start()
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Start again should error
	err = watcher.Start()
	if err == nil {
		t.Error("expected error when starting already started watcher")
	}

	// Stop watcher
	watcher.Stop()

	// Stop again should not panic
	watcher.Stop()
}

func TestFileWatcherCheckFile(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 50 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")
	watcher := NewFileWatcher(cfg, logger)

	// Test with non-existent file
	info, err := watcher.checkFile()
	if err != nil {
		t.Errorf("unexpected error for non-existent file: %v", err)
	}
	if info != nil {
		t.Error("expected nil info for non-existent file")
	}

	// Create empty file
	upgradeFile := filepath.Join(dataDir, "upgrade-info.json")
	err = os.WriteFile(upgradeFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("failed to write empty file: %v", err)
	}

	// Test with empty file
	info, err = watcher.checkFile()
	if err == nil {
		t.Error("expected error for empty file")
	}

	// Create valid upgrade file
	validContent := `{"name": "v2.0.0", "height": 1000000}`
	err = os.WriteFile(upgradeFile, []byte(validContent), 0644)
	if err != nil {
		t.Fatalf("failed to write valid file: %v", err)
	}

	// Force modification time update
	time.Sleep(10 * time.Millisecond)
	os.Chtimes(upgradeFile, time.Now(), time.Now())

	// Test with valid file
	info, err = watcher.checkFile()
	if err != nil {
		t.Errorf("unexpected error for valid file: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil info for valid file")
	}
	if info.Name != "v2.0.0" {
		t.Errorf("expected name v2.0.0, got %s", info.Name)
	}
	if info.Height != 1000000 {
		t.Errorf("expected height 1000000, got %d", info.Height)
	}
}

func TestFileWatcherCheckForUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 50 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")
	watcher := NewFileWatcher(cfg, logger)

	// Initially no update
	hasUpdate := watcher.checkForUpdate()
	if hasUpdate {
		t.Error("expected no update initially")
	}

	// Create upgrade file
	upgradeFile := filepath.Join(dataDir, "upgrade-info.json")
	content := `{"name": "v2.0.0", "height": 1000000}`
	err := os.WriteFile(upgradeFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write upgrade file: %v", err)
	}

	// Should detect update
	hasUpdate = watcher.checkForUpdate()
	if !hasUpdate {
		t.Error("expected update to be detected")
	}

	if !watcher.NeedsUpdate() {
		t.Error("expected needs update to be true")
	}

	// Check current upgrade
	current := watcher.GetCurrentUpgrade()
	if current == nil {
		t.Fatal("expected current upgrade to be non-nil")
	}
	if current.Name != "v2.0.0" {
		t.Errorf("expected upgrade name v2.0.0, got %s", current.Name)
	}

	// Clear update flag
	watcher.ClearUpdateFlag()
	if watcher.NeedsUpdate() {
		t.Error("expected needs update to be false after clear")
	}

	// No update if file hasn't changed
	hasUpdate = watcher.checkForUpdate()
	if hasUpdate {
		t.Error("expected no update when file hasn't changed")
	}
}

func TestFileWatcherMonitoring(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 50 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")
	watcher := NewFileWatcher(cfg, logger)

	// Start monitoring
	err := watcher.Start()
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer watcher.Stop()

	// Create upgrade file after monitoring starts
	time.Sleep(100 * time.Millisecond)

	upgradeFile := filepath.Join(dataDir, "upgrade-info.json")
	content := `{"name": "v3.0.0", "height": 2000000}`
	err = os.WriteFile(upgradeFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write upgrade file: %v", err)
	}

	// Wait for detection
	time.Sleep(150 * time.Millisecond)

	// Should have detected the update
	if !watcher.NeedsUpdate() {
		t.Error("expected update to be detected by monitor")
	}

	current := watcher.GetCurrentUpgrade()
	if current == nil || current.Name != "v3.0.0" {
		t.Error("expected v3.0.0 upgrade to be detected")
	}
}

func TestCreateUpgradeDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}

	upgradeName := "v4.0.0"
	err := CreateUpgradeDir(cfg, upgradeName)
	if err != nil {
		t.Fatalf("failed to create upgrade dir: %v", err)
	}

	// Check directory exists
	upgradeDir := cfg.UpgradeDir(upgradeName)
	binDir := filepath.Join(upgradeDir, "bin")

	info, err := os.Stat(binDir)
	if err != nil {
		t.Fatalf("upgrade bin directory does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("expected bin to be a directory")
	}

	// Create again should not error
	err = CreateUpgradeDir(cfg, upgradeName)
	if err != nil {
		t.Errorf("unexpected error creating existing upgrade dir: %v", err)
	}
}

func TestFileWatcherWaitForUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	cfg := &config.Config{
		Home:         tmpDir,
		PollInterval: 50 * time.Millisecond,
	}

	logger, _ := logger.New(false, true, "")
	watcher := NewFileWatcher(cfg, logger)

	// Start watcher
	err := watcher.Start()
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}

	// Create upgrade file in background
	go func() {
		time.Sleep(100 * time.Millisecond)
		upgradeFile := filepath.Join(dataDir, "upgrade-info.json")
		content := `{"name": "v5.0.0", "height": 3000000}`
		os.WriteFile(upgradeFile, []byte(content), 0644)
	}()

	// Wait for upgrade with timeout
	done := make(chan *types.UpgradeInfo)
	go func() {
		info := watcher.WaitForUpgrade()
		done <- info
	}()

	select {
	case info := <-done:
		if info == nil {
			t.Fatal("expected upgrade info to be non-nil")
		}
		if info.Name != "v5.0.0" {
			t.Errorf("expected upgrade name v5.0.0, got %s", info.Name)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for upgrade")
	}

	watcher.Stop()
}