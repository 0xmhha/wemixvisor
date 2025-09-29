package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewManager(t *testing.T) {
	cfg := &config.Config{
		Home:           t.TempDir(),
		DataBackupPath: filepath.Join(t.TempDir(), "backups"),
	}
	logger, _ := logger.New(false, true, "")

	manager := NewManager(cfg, logger)
	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}
	if manager.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if manager.logger != logger {
		t.Error("expected logger to be set")
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:             tmpDir,
		DataBackupPath:   filepath.Join(tmpDir, "backups"),
		UnsafeSkipBackup: false,
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create data directory with test files
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)

	testFile := filepath.Join(dataDir, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	subDir := filepath.Join(dataDir, "subdir")
	os.MkdirAll(subDir, 0755)
	subFile := filepath.Join(subDir, "sub.txt")
	os.WriteFile(subFile, []byte("sub content"), 0644)

	// Create backup
	backupPath, err := manager.CreateBackup("test-backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}
	if backupPath == "" {
		t.Error("expected backup path to be non-empty")
	}

	// Verify backup file exists
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("backup file does not exist: %v", err)
	}

	// Verify backup file has expected extension
	if filepath.Ext(backupPath) != ".gz" {
		t.Errorf("expected .gz extension, got %s", filepath.Ext(backupPath))
	}
}

func TestCreateBackupSkip(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:             tmpDir,
		DataBackupPath:   filepath.Join(tmpDir, "backups"),
		UnsafeSkipBackup: true, // Skip backup
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create backup with skip flag
	backupPath, err := manager.CreateBackup("test-backup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if backupPath != "" {
		t.Error("expected empty backup path when skipping")
	}
}

func TestCreateBackupNoDataDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:             tmpDir,
		DataBackupPath:   filepath.Join(tmpDir, "backups"),
		UnsafeSkipBackup: false,
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create backup without data directory
	backupPath, err := manager.CreateBackup("test-backup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should succeed but path might be empty
	_ = backupPath
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:             tmpDir,
		DataBackupPath:   filepath.Join(tmpDir, "backups"),
		UnsafeSkipBackup: false,
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create initial data
	dataDir := filepath.Join(tmpDir, "data")
	os.MkdirAll(dataDir, 0755)
	testFile := filepath.Join(dataDir, "test.txt")
	os.WriteFile(testFile, []byte("original content"), 0644)

	// Create backup
	backupPath, err := manager.CreateBackup("test-backup")
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Modify data
	os.WriteFile(testFile, []byte("modified content"), 0644)

	// Restore backup
	err = manager.RestoreBackup(backupPath)
	if err != nil {
		t.Fatalf("failed to restore backup: %v", err)
	}

	// Verify content was restored
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}
	if string(content) != "original content" {
		t.Errorf("expected 'original content', got '%s'", string(content))
	}
}

func TestCleanOldBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	cfg := &config.Config{
		Home:           tmpDir,
		DataBackupPath: backupDir,
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create backup directory
	os.MkdirAll(backupDir, 0755)

	// Create old backup file
	oldBackup := filepath.Join(backupDir, "old-backup.tar.gz")
	os.WriteFile(oldBackup, []byte("old"), 0644)

	// Set modification time to 8 days ago
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	os.Chtimes(oldBackup, oldTime, oldTime)

	// Create recent backup file
	newBackup := filepath.Join(backupDir, "new-backup.tar.gz")
	os.WriteFile(newBackup, []byte("new"), 0644)

	// Clean old backups (older than 7 days)
	err := manager.CleanOldBackups(7 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("failed to clean old backups: %v", err)
	}

	// Verify old backup was removed
	if _, err := os.Stat(oldBackup); !os.IsNotExist(err) {
		t.Error("expected old backup to be removed")
	}

	// Verify new backup still exists
	if _, err := os.Stat(newBackup); err != nil {
		t.Error("expected new backup to still exist")
	}
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	cfg := &config.Config{
		Home:           tmpDir,
		DataBackupPath: backupDir,
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// Create backup directory with some files
	os.MkdirAll(backupDir, 0755)
	os.WriteFile(filepath.Join(backupDir, "backup1.tar.gz"), []byte("1"), 0644)
	os.WriteFile(filepath.Join(backupDir, "backup2.tar.gz"), []byte("2"), 0644)
	os.WriteFile(filepath.Join(backupDir, "not-backup.txt"), []byte("3"), 0644)
	os.MkdirAll(filepath.Join(backupDir, "subdir"), 0755)

	// List backups
	backups, err := manager.ListBackups()
	if err != nil {
		t.Fatalf("failed to list backups: %v", err)
	}

	// Should only list .gz files
	if len(backups) != 2 {
		t.Errorf("expected 2 backups, got %d", len(backups))
	}

	// Verify backup names
	expectedBackups := map[string]bool{
		"backup1.tar.gz": false,
		"backup2.tar.gz": false,
	}
	for _, backup := range backups {
		if _, ok := expectedBackups[backup]; ok {
			expectedBackups[backup] = true
		}
	}
	for name, found := range expectedBackups {
		if !found {
			t.Errorf("expected backup %s not found", name)
		}
	}
}

func TestListBackupsEmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:           tmpDir,
		DataBackupPath: filepath.Join(tmpDir, "backups"),
	}
	logger, _ := logger.New(false, true, "")
	manager := NewManager(cfg, logger)

	// List backups from non-existent directory
	backups, err := manager.ListBackups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("expected empty list, got %d backups", len(backups))
	}
}