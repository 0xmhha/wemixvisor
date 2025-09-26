package download

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wemix/wemixvisor/internal/config"
	"github.com/wemix/wemixvisor/pkg/logger"
	"go.uber.org/zap"
)

// Downloader manages binary downloads and verification
type Downloader struct {
	cfg    *config.Config
	logger *logger.Logger
	client *http.Client
}

// NewDownloader creates a new downloader instance
func NewDownloader(cfg *config.Config, logger *logger.Logger) *Downloader {
	return &Downloader{
		cfg:    cfg,
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large binaries
		},
	}
}

// DownloadBinary downloads a binary from the specified URL
func (d *Downloader) DownloadBinary(url, destPath string) error {
	d.logger.Info("downloading binary",
		zap.String("url", url),
		zap.String("destination", destPath))

	// Create temporary file
	tempFile := destPath + ".tmp"

	// Download to temporary file
	if err := d.downloadFile(url, tempFile); err != nil {
		// Clean up temporary file on error
		os.Remove(tempFile)
		return fmt.Errorf("download failed: %w", err)
	}

	// Move temporary file to final destination
	if err := os.Rename(tempFile, destPath); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to move downloaded file: %w", err)
	}

	// Make binary executable
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	d.logger.Info("binary downloaded successfully",
		zap.String("path", destPath))

	return nil
}

// DownloadAndVerify downloads a binary and verifies its checksum
func (d *Downloader) DownloadAndVerify(url, destPath, checksumURL string) error {
	// Download checksum file
	checksumData, err := d.fetchChecksum(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to fetch checksum: %w", err)
	}

	// Download binary
	if err := d.DownloadBinary(url, destPath); err != nil {
		return err
	}

	// Verify checksum
	if err := d.verifyChecksum(destPath, checksumData); err != nil {
		// Remove downloaded file if verification fails
		os.Remove(destPath)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	return nil
}

// downloadFile performs the actual file download with retry logic
func (d *Downloader) downloadFile(url, destPath string) error {
	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			d.logger.Info("retrying download",
				zap.Int("attempt", attempt+1),
				zap.Int("max_retries", maxRetries))
			time.Sleep(time.Second * time.Duration(attempt*2))
		}

		err := d.attemptDownload(url, destPath)
		if err == nil {
			return nil
		}
		lastErr = err
		d.logger.Warn("download attempt failed",
			zap.Error(err),
			zap.Int("attempt", attempt+1))
	}

	return fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

// attemptDownload performs a single download attempt
func (d *Downloader) attemptDownload(url, destPath string) error {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Perform the request
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Create destination file
	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create progress reporter
	progressReader := &progressReader{
		reader: resp.Body,
		total:  resp.ContentLength,
		logger: d.logger,
	}

	// Copy with progress reporting
	_, err = io.Copy(file, progressReader)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// fetchChecksum downloads and parses checksum file
func (d *Downloader) fetchChecksum(url string) (string, error) {
	resp, err := d.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksum: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read checksum data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read checksum: %w", err)
	}

	// Parse checksum (format: "checksum filename")
	checksumStr := strings.TrimSpace(string(data))
	parts := strings.Fields(checksumStr)
	if len(parts) > 0 {
		return parts[0], nil
	}

	return checksumStr, nil
}

// verifyChecksum verifies the downloaded file's checksum
func (d *Downloader) verifyChecksum(filePath, expectedChecksum string) error {
	// Determine hash algorithm based on checksum length
	var h hash.Hash
	switch len(expectedChecksum) {
	case 64: // SHA256
		h = sha256.New()
	case 128: // SHA512
		h = sha512.New()
	default:
		return fmt.Errorf("unsupported checksum length: %d", len(expectedChecksum))
	}

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Calculate checksum
	if _, err := io.Copy(h, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(h.Sum(nil))

	// Compare checksums
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s",
			expectedChecksum, actualChecksum)
	}

	d.logger.Info("checksum verified successfully",
		zap.String("algorithm", checksumAlgorithm(h)),
		zap.String("checksum", actualChecksum))

	return nil
}

// checksumAlgorithm returns the name of the hash algorithm
func checksumAlgorithm(h hash.Hash) string {
	switch h.Size() {
	case sha256.Size:
		return "SHA256"
	case sha512.Size:
		return "SHA512"
	default:
		return "unknown"
	}
}

// progressReader reports download progress
type progressReader struct {
	reader     io.Reader
	total      int64
	downloaded int64
	lastReport time.Time
	logger     *logger.Logger
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.downloaded += int64(n)

	// Report progress every 5 seconds
	now := time.Now()
	if now.Sub(pr.lastReport) >= 5*time.Second && pr.total > 0 {
		percentage := float64(pr.downloaded) * 100 / float64(pr.total)
		pr.logger.Info("download progress",
			zap.Float64("percentage", percentage),
			zap.Int64("downloaded_bytes", pr.downloaded),
			zap.Int64("total_bytes", pr.total))
		pr.lastReport = now
	}

	return n, err
}

// GetBinaryURL constructs the download URL for a binary
func (d *Downloader) GetBinaryURL(upgradeName string) (string, error) {
	// Check if download URLs are configured
	if d.cfg.DownloadURLs == nil || len(d.cfg.DownloadURLs) == 0 {
		return "", fmt.Errorf("no download URLs configured")
	}

	// Look for specific upgrade URL
	if url, ok := d.cfg.DownloadURLs[upgradeName]; ok {
		return url, nil
	}

	// Look for default URL template
	if template, ok := d.cfg.DownloadURLs["default"]; ok {
		// Replace placeholders in template
		url := strings.ReplaceAll(template, "{name}", upgradeName)
		url = strings.ReplaceAll(url, "{version}", upgradeName)
		return url, nil
	}

	return "", fmt.Errorf("no download URL found for upgrade: %s", upgradeName)
}

// GetChecksumURL constructs the checksum URL for a binary
func (d *Downloader) GetChecksumURL(binaryURL string) string {
	// Common checksum file extensions
	checksumExtensions := []string{".sha256", ".sha512", ".checksum"}

	for _, ext := range checksumExtensions {
		checksumURL := binaryURL + ext
		// Try to fetch to see if it exists
		resp, err := d.client.Head(checksumURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return checksumURL
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Default to .sha256
	return binaryURL + ".sha256"
}

// EnsureUpgradeBinary ensures the upgrade binary exists, downloading if necessary
func (d *Downloader) EnsureUpgradeBinary(upgradeName string) error {
	// Check if binary already exists
	upgradeBin := d.cfg.UpgradeBin(upgradeName)
	if _, err := os.Stat(upgradeBin); err == nil {
		d.logger.Info("upgrade binary already exists",
			zap.String("path", upgradeBin))
		return nil
	}

	// Check if downloads are allowed
	if !d.cfg.AllowDownloadBinaries {
		return fmt.Errorf("binary not found and downloads disabled: %s", upgradeBin)
	}

	// Get download URL
	binaryURL, err := d.GetBinaryURL(upgradeName)
	if err != nil {
		return err
	}

	// Create upgrade directory if it doesn't exist
	upgradeDir := filepath.Dir(upgradeBin)
	if err := os.MkdirAll(upgradeDir, 0755); err != nil {
		return fmt.Errorf("failed to create upgrade directory: %w", err)
	}

	// Get checksum URL
	checksumURL := d.GetChecksumURL(binaryURL)

	// Download and verify
	d.logger.Info("downloading upgrade binary",
		zap.String("upgrade", upgradeName),
		zap.String("url", binaryURL))

	if err := d.DownloadAndVerify(binaryURL, upgradeBin, checksumURL); err != nil {
		// Try download without verification as fallback
		d.logger.Warn("checksum verification failed, attempting download without verification",
			zap.Error(err))

		if d.cfg.UnsafeSkipChecksum {
			return d.DownloadBinary(binaryURL, upgradeBin)
		}

		return err
	}

	return nil
}