package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Manager handles backup operations
type Manager struct {
	cfg    *config.Config
	logger *logger.Logger
}

// NewManager creates a new backup manager
func NewManager(cfg *config.Config, logger *logger.Logger) *Manager {
	return &Manager{
		cfg:    cfg,
		logger: logger,
	}
}

// CreateBackup creates a backup of the data directory
func (m *Manager) CreateBackup(name string) (string, error) {
	if m.cfg.UnsafeSkipBackup {
		m.logger.Info("skipping backup as UnsafeSkipBackup is set")
		return "", nil
	}

	// Create backup directory if it doesn't exist
	backupDir := m.cfg.DataBackupPath
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s-%s.tar.gz", name, timestamp)
	backupPath := filepath.Join(backupDir, backupName)

	// Create the backup archive
	if err := m.createArchive(backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup archive: %w", err)
	}

	m.logger.Info("backup created successfully", zap.String("path", backupPath))
	return backupPath, nil
}

// createArchive creates a tar.gz archive of the data directory
func (m *Manager) createArchive(destPath string) error {
	dataDir := filepath.Join(m.cfg.Home, "data")

	// Check if data directory exists
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		m.logger.Warn("data directory does not exist, skipping backup", zap.String("path", dataDir))
		return nil
	}

	// Create the output file
	outFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Walk the data directory and add files to archive
	return filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the backup directory itself if it's inside data
		if filepath.HasPrefix(path, m.cfg.DataBackupPath) {
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update the name to be relative to data directory
		relPath, err := filepath.Rel(m.cfg.Home, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// If it's a file, write the content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})
}

// RestoreBackup restores a backup to the data directory
func (m *Manager) RestoreBackup(backupPath string) error {
	// Open the backup file
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzReader)

	// Extract files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct the full path
		targetPath := filepath.Join(m.cfg.Home, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// Create file
			if err := m.extractFile(tarReader, targetPath); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}
		}
	}

	m.logger.Info("backup restored successfully", zap.String("path", backupPath))
	return nil
}

// extractFile extracts a single file from the tar reader
func (m *Manager) extractFile(tarReader *tar.Reader, destPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create the file
	outFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// Copy content
	if _, err := io.Copy(outFile, tarReader); err != nil {
		return err
	}

	return nil
}

// CleanOldBackups removes backups older than the specified duration
func (m *Manager) CleanOldBackups(maxAge time.Duration) error {
	backupDir := m.cfg.DataBackupPath

	// Read directory
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	now := time.Now()
	removed := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if file is older than maxAge
		if now.Sub(info.ModTime()) > maxAge {
			path := filepath.Join(backupDir, entry.Name())
			if err := os.Remove(path); err != nil {
				m.logger.Warn("failed to remove old backup", zap.String("path", path), zap.Error(err))
			} else {
				removed++
				m.logger.Debug("removed old backup", zap.String("path", path))
			}
		}
	}

	if removed > 0 {
		m.logger.Info("cleaned old backups", zap.Int("removed", removed))
	}

	return nil
}

// ListBackups returns a list of available backups
func (m *Manager) ListBackups() ([]string, error) {
	backupDir := m.cfg.DataBackupPath

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".gz" {
			backups = append(backups, entry.Name())
		}
	}

	return backups, nil
}