package performance

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/wemix/wemixvisor/pkg/logger"
)

// Profiler manages CPU and memory profiling
type Profiler struct {
	logger         *logger.Logger
	profileDir     string
	cpuFile        *os.File
	cpuProfiling   bool
	profileEnabled bool
}

// ProfilerConfig represents profiler configuration
type ProfilerConfig struct {
	ProfileDir string        `json:"profile_dir"`
	Enabled    bool          `json:"enabled"`
	Interval   time.Duration `json:"interval"`
}

// NewProfiler creates a new profiler
func NewProfiler(profileDir string, logger *logger.Logger) *Profiler {
	return &Profiler{
		logger:         logger,
		profileDir:     profileDir,
		cpuProfiling:   false,
		profileEnabled: true,
	}
}

// Start starts the profiler
func (p *Profiler) Start() error {
	// Create profile directory if it doesn't exist
	if err := os.MkdirAll(p.profileDir, 0755); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	p.logger.Info("Profiler started", "profile_dir", p.profileDir)
	return nil
}

// Stop stops the profiler
func (p *Profiler) Stop() error {
	if p.cpuProfiling {
		if err := p.StopCPUProfile(); err != nil {
			return err
		}
	}

	p.logger.Info("Profiler stopped")
	return nil
}

// StartCPUProfile starts CPU profiling
func (p *Profiler) StartCPUProfile() error {
	if p.cpuProfiling {
		return fmt.Errorf("CPU profiling already in progress")
	}

	if !p.profileEnabled {
		return fmt.Errorf("profiling is disabled")
	}

	filename := filepath.Join(p.profileDir, fmt.Sprintf("cpu_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CPU profile file: %w", err)
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profiling: %w", err)
	}

	p.cpuFile = f
	p.cpuProfiling = true
	p.logger.Info("CPU profiling started", "file", filename)

	return nil
}

// StopCPUProfile stops CPU profiling
func (p *Profiler) StopCPUProfile() error {
	if !p.cpuProfiling {
		return fmt.Errorf("CPU profiling not in progress")
	}

	pprof.StopCPUProfile()
	if err := p.cpuFile.Close(); err != nil {
		p.logger.Error("Failed to close CPU profile file", "error", err.Error())
	}

	p.cpuProfiling = false
	p.cpuFile = nil
	p.logger.Info("CPU profiling stopped")

	return nil
}

// WriteHeapProfile writes a heap memory profile
func (p *Profiler) WriteHeapProfile() error {
	if !p.profileEnabled {
		return fmt.Errorf("profiling is disabled")
	}

	filename := filepath.Join(p.profileDir, fmt.Sprintf("heap_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create heap profile file: %w", err)
	}
	defer f.Close()

	runtime.GC() // Get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("failed to write heap profile: %w", err)
	}

	p.logger.Info("Heap profile written", "file", filename)
	return nil
}

// WriteGoroutineProfile writes a goroutine profile
func (p *Profiler) WriteGoroutineProfile() error {
	if !p.profileEnabled {
		return fmt.Errorf("profiling is disabled")
	}

	filename := filepath.Join(p.profileDir, fmt.Sprintf("goroutine_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create goroutine profile file: %w", err)
	}
	defer f.Close()

	profile := pprof.Lookup("goroutine")
	if profile == nil {
		return fmt.Errorf("goroutine profile not available")
	}

	if err := profile.WriteTo(f, 0); err != nil {
		return fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	p.logger.Info("Goroutine profile written", "file", filename)
	return nil
}

// WriteBlockProfile writes a block contention profile
func (p *Profiler) WriteBlockProfile() error {
	if !p.profileEnabled {
		return fmt.Errorf("profiling is disabled")
	}

	filename := filepath.Join(p.profileDir, fmt.Sprintf("block_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create block profile file: %w", err)
	}
	defer f.Close()

	profile := pprof.Lookup("block")
	if profile == nil {
		return fmt.Errorf("block profile not available")
	}

	if err := profile.WriteTo(f, 0); err != nil {
		return fmt.Errorf("failed to write block profile: %w", err)
	}

	p.logger.Info("Block profile written", "file", filename)
	return nil
}

// WriteMutexProfile writes a mutex contention profile
func (p *Profiler) WriteMutexProfile() error {
	if !p.profileEnabled {
		return fmt.Errorf("profiling is disabled")
	}

	filename := filepath.Join(p.profileDir, fmt.Sprintf("mutex_%s.prof", time.Now().Format("20060102_150405")))
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create mutex profile file: %w", err)
	}
	defer f.Close()

	profile := pprof.Lookup("mutex")
	if profile == nil {
		return fmt.Errorf("mutex profile not available")
	}

	if err := profile.WriteTo(f, 0); err != nil {
		return fmt.Errorf("failed to write mutex profile: %w", err)
	}

	p.logger.Info("Mutex profile written", "file", filename)
	return nil
}

// WriteAllProfiles writes all available profiles
func (p *Profiler) WriteAllProfiles() error {
	var errs []error

	if err := p.WriteHeapProfile(); err != nil {
		errs = append(errs, err)
	}

	if err := p.WriteGoroutineProfile(); err != nil {
		errs = append(errs, err)
	}

	if err := p.WriteBlockProfile(); err != nil {
		errs = append(errs, err)
	}

	if err := p.WriteMutexProfile(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to write %d profiles: %v", len(errs), errs)
	}

	p.logger.Info("All profiles written")
	return nil
}

// GetProfileDir returns the profile directory
func (p *Profiler) GetProfileDir() string {
	return p.profileDir
}

// IsCPUProfiling returns true if CPU profiling is in progress
func (p *Profiler) IsCPUProfiling() bool {
	return p.cpuProfiling
}

// SetEnabled enables or disables profiling
func (p *Profiler) SetEnabled(enabled bool) {
	p.profileEnabled = enabled
	p.logger.Info("Profiling enabled status changed", "enabled", enabled)
}

// IsEnabled returns true if profiling is enabled
func (p *Profiler) IsEnabled() bool {
	return p.profileEnabled
}

// GetMemStats returns current memory statistics
func (p *Profiler) GetMemStats() *runtime.MemStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return &m
}

// GetGoroutineCount returns the current number of goroutines
func (p *Profiler) GetGoroutineCount() int {
	return runtime.NumGoroutine()
}

// ProfileInfo holds information about a saved profile
type ProfileInfo struct {
	Type      string    `json:"type"`
	Filename  string    `json:"filename"`
	Timestamp time.Time `json:"timestamp"`
	Size      int64     `json:"size"`
}

// ListProfiles returns a list of saved profiles
func (p *Profiler) ListProfiles() ([]*ProfileInfo, error) {
	files, err := os.ReadDir(p.profileDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile directory: %w", err)
	}

	var profiles []*ProfileInfo
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		// Parse profile type from filename
		var profileType string
		switch {
		case filepath.Ext(info.Name()) != ".prof":
			continue // Skip non-profile files
		case len(info.Name()) > 4 && info.Name()[:4] == "cpu_":
			profileType = "cpu"
		case len(info.Name()) > 5 && info.Name()[:5] == "heap_":
			profileType = "heap"
		case len(info.Name()) > 10 && info.Name()[:10] == "goroutine_":
			profileType = "goroutine"
		case len(info.Name()) > 6 && info.Name()[:6] == "block_":
			profileType = "block"
		case len(info.Name()) > 6 && info.Name()[:6] == "mutex_":
			profileType = "mutex"
		default:
			profileType = "unknown"
		}

		profiles = append(profiles, &ProfileInfo{
			Type:      profileType,
			Filename:  info.Name(),
			Timestamp: info.ModTime(),
			Size:      info.Size(),
		})
	}

	return profiles, nil
}

// DeleteProfile deletes a saved profile file
func (p *Profiler) DeleteProfile(filename string) error {
	path := filepath.Join(p.profileDir, filename)

	// Security check: ensure the file is within profile directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	absProfileDir, err := filepath.Abs(p.profileDir)
	if err != nil {
		return fmt.Errorf("invalid profile directory: %w", err)
	}

	if !filepath.HasPrefix(absPath, absProfileDir) {
		return fmt.Errorf("path outside profile directory")
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	p.logger.Info("Profile deleted", "file", filename)
	return nil
}

// CleanOldProfiles removes profiles older than the specified duration
func (p *Profiler) CleanOldProfiles(maxAge time.Duration) error {
	profiles, err := p.ListProfiles()
	if err != nil {
		return err
	}

	now := time.Now()
	deletedCount := 0

	for _, profile := range profiles {
		if now.Sub(profile.Timestamp) > maxAge {
			if err := p.DeleteProfile(profile.Filename); err != nil {
				p.logger.Warn("Failed to delete old profile", "file", profile.Filename, "error", err.Error())
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		p.logger.Info("Cleaned old profiles", "count", deletedCount)
	}

	return nil
}
