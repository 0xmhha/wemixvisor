package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
)

func TestNewDownloader(t *testing.T) {
	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")

	downloader := NewDownloader(cfg, logger)
	if downloader == nil {
		t.Fatal("expected downloader to be non-nil")
	}
	if downloader.cfg != cfg {
		t.Error("expected cfg to be set")
	}
	if downloader.logger != logger {
		t.Error("expected logger to be set")
	}
	if downloader.client == nil {
		t.Error("expected client to be non-nil")
	}
}

func TestDownloadBinary(t *testing.T) {
	// Create test server
	content := []byte("test binary content")
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.Write(content)
	}))
	defer server.Close()

	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Download binary
	destPath := filepath.Join(cfg.Home, "test-binary")
	err := downloader.DownloadBinary(server.URL, destPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists
	info, err := os.Stat(destPath)
	if err != nil {
		t.Fatalf("file not found: %v", err)
	}

	// Check file is executable
	if info.Mode()&0111 == 0 {
		t.Error("file is not executable")
	}

	// Verify content
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %s, want %s", string(data), string(content))
	}
}

func TestDownloadBinaryWithError(t *testing.T) {
	// Create test server that returns error
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Download binary
	destPath := filepath.Join(cfg.Home, "test-binary")
	err := downloader.DownloadBinary(server.URL, destPath)
	if err == nil {
		t.Fatal("expected error")
	}

	// Verify file doesn't exist
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("file should not exist")
	}
}

func TestDownloadAndVerify(t *testing.T) {
	// Create test content
	content := []byte("test binary content with checksum")
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	// Create test server
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Write(content)
		} else if r.URL.Path == "/checksum" {
			w.Write([]byte(checksum + " binary\n"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Download and verify
	destPath := filepath.Join(cfg.Home, "test-binary")
	err := downloader.DownloadAndVerify(
		server.URL+"/binary",
		destPath,
		server.URL+"/checksum",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %s, want %s", string(data), string(content))
	}
}

func TestDownloadAndVerifyChecksumMismatch(t *testing.T) {
	// Create test content
	content := []byte("test binary content")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	// Create test server
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Write(content)
		} else if r.URL.Path == "/checksum" {
			w.Write([]byte(wrongChecksum))
		}
	}))
	defer server.Close()

	cfg := &config.Config{
		Home: t.TempDir(),
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Download and verify
	destPath := filepath.Join(cfg.Home, "test-binary")
	err := downloader.DownloadAndVerify(
		server.URL+"/binary",
		destPath,
		server.URL+"/checksum",
	)
	if err == nil {
		t.Fatal("expected checksum mismatch error")
	}

	// Verify file was removed
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Error("file should not exist after checksum failure")
	}
}

func TestGetBinaryURL(t *testing.T) {
	tests := []struct {
		name         string
		downloadURLs map[string]string
		upgradeName  string
		expected     string
		expectError  bool
	}{
		{
			name: "specific URL",
			downloadURLs: map[string]string{
				"v2.0.0": "https://example.com/v2.0.0/binary",
			},
			upgradeName: "v2.0.0",
			expected:    "https://example.com/v2.0.0/binary",
			expectError: false,
		},
		{
			name: "template URL",
			downloadURLs: map[string]string{
				"default": "https://example.com/releases/{version}/binary",
			},
			upgradeName: "v2.0.0",
			expected:    "https://example.com/releases/v2.0.0/binary",
			expectError: false,
		},
		{
			name:         "no URLs configured",
			downloadURLs: nil,
			upgradeName:  "v2.0.0",
			expectError:  true,
		},
		{
			name: "no matching URL",
			downloadURLs: map[string]string{
				"v1.0.0": "https://example.com/v1.0.0/binary",
			},
			upgradeName: "v2.0.0",
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{
				Home:         t.TempDir(),
				Name:         "wemixd",
				DownloadURLs: tc.downloadURLs,
			}
			logger, _ := logger.New(false, true, "")
			downloader := NewDownloader(cfg, logger)

			url, err := downloader.GetBinaryURL(tc.upgradeName)
			if tc.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if url != tc.expected {
					t.Errorf("URL mismatch: got %s, want %s", url, tc.expected)
				}
			}
		})
	}
}

func TestEnsureUpgradeBinary(t *testing.T) {
	// Create test server
	content := []byte("test upgrade binary")
	hash := sha256.Sum256(content)
	checksum := hex.EncodeToString(hash[:])

	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2.0.0/wemixd" {
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			w.Write(content)
		} else if r.URL.Path == "/v2.0.0/wemixd.sha256" {
			w.Write([]byte(checksum))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:                  tmpDir,
		Name:                  "wemixd",
		AllowDownloadBinaries: true,
		DownloadURLs: map[string]string{
			"v2.0.0": server.URL + "/v2.0.0/wemixd",
		},
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Ensure binary
	err := downloader.EnsureUpgradeBinary("v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify binary exists
	upgradeBin := cfg.UpgradeBin("v2.0.0")
	data, err := os.ReadFile(upgradeBin)
	if err != nil {
		t.Fatalf("failed to read binary: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("content mismatch: got %s, want %s", string(data), string(content))
	}

	// Ensure again (should skip download)
	err = downloader.EnsureUpgradeBinary("v2.0.0")
	if err != nil {
		t.Fatalf("unexpected error on second ensure: %v", err)
	}
}

func TestEnsureUpgradeBinaryDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home:                  tmpDir,
		Name:                  "wemixd",
		AllowDownloadBinaries: false, // Downloads disabled
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Ensure binary
	err := downloader.EnsureUpgradeBinary("v2.0.0")
	if err == nil {
		t.Fatal("expected error when downloads disabled")
	}
}

func TestVerifyChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Home: tmpDir,
		Name: "wemixd",
	}
	logger, _ := logger.New(false, true, "")
	downloader := NewDownloader(cfg, logger)

	// Create test file
	content := []byte("test content for checksum")
	filePath := filepath.Join(tmpDir, "test-file")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test SHA256
	sha256Hash := sha256.Sum256(content)
	sha256Checksum := hex.EncodeToString(sha256Hash[:])
	err := downloader.verifyChecksum(filePath, sha256Checksum)
	if err != nil {
		t.Errorf("SHA256 verification failed: %v", err)
	}

	// Test wrong checksum
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
	err = downloader.verifyChecksum(filePath, wrongChecksum)
	if err == nil {
		t.Error("expected checksum mismatch error")
	}

	// Test unsupported checksum length
	shortChecksum := "00000000"
	err = downloader.verifyChecksum(filePath, shortChecksum)
	if err == nil {
		t.Error("expected unsupported checksum length error")
	}
}
