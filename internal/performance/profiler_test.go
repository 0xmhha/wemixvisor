package performance

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wemix/wemixvisor/pkg/logger"
)

// setupTestProfiler creates a test profiler with temp directory
func setupTestProfiler(t *testing.T) (*Profiler, string) {
	tempDir := t.TempDir()
	profileDir := filepath.Join(tempDir, "profiles")
	testLogger := logger.NewTestLogger()

	profiler := NewProfiler(profileDir, testLogger)
	return profiler, profileDir
}

// TestNewProfiler tests profiler initialization
func TestNewProfiler(t *testing.T) {
	// Arrange
	testLogger := logger.NewTestLogger()
	profileDir := "/tmp/test-profiles"

	// Act
	profiler := NewProfiler(profileDir, testLogger)

	// Assert
	assert.NotNil(t, profiler)
	assert.Equal(t, profileDir, profiler.profileDir)
	assert.False(t, profiler.cpuProfiling)
	assert.True(t, profiler.profileEnabled)
	assert.Equal(t, profileDir, profiler.GetProfileDir())
}

// TestProfilerStartStop tests profiler lifecycle
func TestProfilerStartStop(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)

	// Act - Start
	err := profiler.Start()

	// Assert - Start
	require.NoError(t, err)
	assert.DirExists(t, profileDir)

	// Act - Stop
	err = profiler.Stop()

	// Assert - Stop
	assert.NoError(t, err)
}

// TestStartStopCPUProfile tests CPU profiling lifecycle
func TestStartStopCPUProfile(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Act - Start CPU profiling
	err = profiler.StartCPUProfile()

	// Assert - Started
	require.NoError(t, err)
	assert.True(t, profiler.IsCPUProfiling())

	// Simulate some CPU activity
	time.Sleep(100 * time.Millisecond)

	// Act - Stop CPU profiling
	err = profiler.StopCPUProfile()

	// Assert - Stopped
	require.NoError(t, err)
	assert.False(t, profiler.IsCPUProfiling())

	// Verify profile file was created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify file naming pattern
	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" && len(file.Name()) > 4 && file.Name()[:4] == "cpu_" {
			found = true
			break
		}
	}
	assert.True(t, found, "CPU profile file not found")
}

// TestCPUProfileAlreadyRunning tests error when starting CPU profile twice
func TestCPUProfileAlreadyRunning(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	err = profiler.StartCPUProfile()
	require.NoError(t, err)
	defer profiler.StopCPUProfile()

	// Act - Try to start again
	err = profiler.StartCPUProfile()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already in progress")
}

// TestStopCPUProfileNotRunning tests error when stopping CPU profile that's not running
func TestStopCPUProfileNotRunning(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Act - Try to stop without starting
	err = profiler.StopCPUProfile()

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in progress")
}

// TestWriteHeapProfile tests heap profile generation
func TestWriteHeapProfile(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Act
	err = profiler.WriteHeapProfile()

	// Assert
	require.NoError(t, err)

	// Verify profile file was created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify file naming pattern
	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" && len(file.Name()) > 5 && file.Name()[:5] == "heap_" {
			found = true
			break
		}
	}
	assert.True(t, found, "Heap profile file not found")
}

// TestWriteGoroutineProfile tests goroutine profile generation
func TestWriteGoroutineProfile(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Act
	err = profiler.WriteGoroutineProfile()

	// Assert
	require.NoError(t, err)

	// Verify profile file was created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify file naming pattern
	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" && len(file.Name()) > 10 && file.Name()[:10] == "goroutine_" {
			found = true
			break
		}
	}
	assert.True(t, found, "Goroutine profile file not found")
}

// TestWriteBlockProfile tests block profile generation
func TestWriteBlockProfile(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Enable block profiling
	runtime.SetBlockProfileRate(1)
	defer runtime.SetBlockProfileRate(0)

	// Act
	err = profiler.WriteBlockProfile()

	// Assert
	require.NoError(t, err)

	// Verify profile file was created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify file naming pattern
	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" && len(file.Name()) > 6 && file.Name()[:6] == "block_" {
			found = true
			break
		}
	}
	assert.True(t, found, "Block profile file not found")
}

// TestWriteMutexProfile tests mutex profile generation
func TestWriteMutexProfile(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Enable mutex profiling
	runtime.SetMutexProfileFraction(1)
	defer runtime.SetMutexProfileFraction(0)

	// Act
	err = profiler.WriteMutexProfile()

	// Assert
	require.NoError(t, err)

	// Verify profile file was created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)

	// Verify file naming pattern
	found := false
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".prof" && len(file.Name()) > 6 && file.Name()[:6] == "mutex_" {
			found = true
			break
		}
	}
	assert.True(t, found, "Mutex profile file not found")
}

// TestWriteAllProfiles tests batch profile generation
func TestWriteAllProfiles(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Enable all profiling
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	defer func() {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(0)
	}()

	// Act
	err = profiler.WriteAllProfiles()

	// Assert
	require.NoError(t, err)

	// Verify multiple profile files were created
	files, err := os.ReadDir(profileDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 4, "Expected at least 4 profiles (heap, goroutine, block, mutex)")

	// Verify each profile type exists
	profileTypes := make(map[string]bool)
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".prof" {
			continue
		}
		switch {
		case len(file.Name()) > 5 && file.Name()[:5] == "heap_":
			profileTypes["heap"] = true
		case len(file.Name()) > 10 && file.Name()[:10] == "goroutine_":
			profileTypes["goroutine"] = true
		case len(file.Name()) > 6 && file.Name()[:6] == "block_":
			profileTypes["block"] = true
		case len(file.Name()) > 6 && file.Name()[:6] == "mutex_":
			profileTypes["mutex"] = true
		}
	}

	assert.True(t, profileTypes["heap"], "Heap profile not found")
	assert.True(t, profileTypes["goroutine"], "Goroutine profile not found")
	assert.True(t, profileTypes["block"], "Block profile not found")
	assert.True(t, profileTypes["mutex"], "Mutex profile not found")
}

// TestSetEnabledDisabled tests enabling/disabling profiler
func TestSetEnabledDisabled(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Assert - Initially enabled
	assert.True(t, profiler.IsEnabled())

	// Act - Disable
	profiler.SetEnabled(false)

	// Assert - Disabled
	assert.False(t, profiler.IsEnabled())

	// Act - Re-enable
	profiler.SetEnabled(true)

	// Assert - Enabled again
	assert.True(t, profiler.IsEnabled())
}

// TestProfilingWhenDisabled tests error when profiling is disabled
func TestProfilingWhenDisabled(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	profiler.SetEnabled(false)

	// Test CPU profiling
	t.Run("CPU profiling disabled", func(t *testing.T) {
		err := profiler.StartCPUProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	// Test heap profiling
	t.Run("Heap profiling disabled", func(t *testing.T) {
		err := profiler.WriteHeapProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	// Test goroutine profiling
	t.Run("Goroutine profiling disabled", func(t *testing.T) {
		err := profiler.WriteGoroutineProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	// Test block profiling
	t.Run("Block profiling disabled", func(t *testing.T) {
		err := profiler.WriteBlockProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})

	// Test mutex profiling
	t.Run("Mutex profiling disabled", func(t *testing.T) {
		err := profiler.WriteMutexProfile()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "disabled")
	})
}

// TestGetMemStats tests memory statistics
func TestGetMemStats(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)

	// Act
	stats := profiler.GetMemStats()

	// Assert
	assert.NotNil(t, stats)
	assert.Greater(t, stats.Alloc, uint64(0))
	assert.Greater(t, stats.TotalAlloc, uint64(0))
	assert.Greater(t, stats.Sys, uint64(0))
	assert.GreaterOrEqual(t, stats.NumGC, uint32(0))
}

// TestGetGoroutineCount tests goroutine count
func TestGetGoroutineCount(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)

	// Act
	count := profiler.GetGoroutineCount()

	// Assert
	assert.Greater(t, count, 0, "At least one goroutine (the test) should be running")
}

// TestListProfiles tests listing saved profiles
func TestListProfiles(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Create some profiles
	err = profiler.WriteHeapProfile()
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = profiler.WriteGoroutineProfile()
	require.NoError(t, err)

	// Act
	profiles, err := profiler.ListProfiles()

	// Assert
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(profiles), 2)

	// Verify profile information
	for _, profile := range profiles {
		assert.NotEmpty(t, profile.Type)
		assert.NotEmpty(t, profile.Filename)
		assert.NotZero(t, profile.Timestamp)
		assert.Greater(t, profile.Size, int64(0))
		assert.Contains(t, []string{"heap", "goroutine", "cpu", "block", "mutex"}, profile.Type)
	}

	// Verify non-profile files are ignored
	nonProfileFile := filepath.Join(profileDir, "notaprofile.txt")
	err = os.WriteFile(nonProfileFile, []byte("test"), 0644)
	require.NoError(t, err)

	profiles, err = profiler.ListProfiles()
	require.NoError(t, err)
	for _, profile := range profiles {
		assert.NotEqual(t, "notaprofile.txt", profile.Filename)
	}
}

// TestDeleteProfile tests deleting profiles
func TestDeleteProfile(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Create a profile
	err = profiler.WriteHeapProfile()
	require.NoError(t, err)

	// Get profile list
	profiles, err := profiler.ListProfiles()
	require.NoError(t, err)
	require.Greater(t, len(profiles), 0)

	filename := profiles[0].Filename

	// Act - Delete profile
	err = profiler.DeleteProfile(filename)

	// Assert
	require.NoError(t, err)

	// Verify profile was deleted
	profilesAfter, err := profiler.ListProfiles()
	require.NoError(t, err)
	assert.Equal(t, len(profiles)-1, len(profilesAfter))

	// Verify deleted profile is not in list
	for _, profile := range profilesAfter {
		assert.NotEqual(t, filename, profile.Filename)
	}
}

// TestDeleteProfileSecurityCheck tests path traversal protection
func TestDeleteProfileSecurityCheck(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	tests := []struct {
		name     string
		filename string
	}{
		{
			name:     "relative path traversal",
			filename: "../../../etc/passwd",
		},
		{
			name:     "complex path traversal",
			filename: "../../tmp/../etc/passwd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			err := profiler.DeleteProfile(tt.filename)

			// Assert - Should fail with security error or file not found
			assert.Error(t, err)
		})
	}
}

// TestCleanOldProfiles tests cleaning old profiles
func TestCleanOldProfiles(t *testing.T) {
	// Arrange
	profiler, profileDir := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	// Create some profiles
	err = profiler.WriteHeapProfile()
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)

	err = profiler.WriteGoroutineProfile()
	require.NoError(t, err)

	// Get initial profile count
	profilesBefore, err := profiler.ListProfiles()
	require.NoError(t, err)
	require.Greater(t, len(profilesBefore), 0)

	// Make one profile old by modifying its timestamp
	if len(profilesBefore) > 0 {
		oldFile := filepath.Join(profileDir, profilesBefore[0].Filename)
		oldTime := time.Now().Add(-2 * time.Hour)
		err = os.Chtimes(oldFile, oldTime, oldTime)
		require.NoError(t, err)
	}

	// Act - Clean profiles older than 1 hour
	err = profiler.CleanOldProfiles(1 * time.Hour)

	// Assert
	require.NoError(t, err)

	// Verify old profile was deleted
	profilesAfter, err := profiler.ListProfiles()
	require.NoError(t, err)
	assert.Less(t, len(profilesAfter), len(profilesBefore))
}

// TestProfilerStopWithCPUProfiling tests Stop() when CPU profiling is active
func TestProfilerStopWithCPUProfiling(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	err = profiler.StartCPUProfile()
	require.NoError(t, err)
	assert.True(t, profiler.IsCPUProfiling())

	// Act - Stop should also stop CPU profiling
	err = profiler.Stop()

	// Assert
	require.NoError(t, err)
	assert.False(t, profiler.IsCPUProfiling())
}

// TestConcurrentProfiling tests concurrent profile generation
func TestConcurrentProfiling(t *testing.T) {
	// Arrange
	profiler, _ := setupTestProfiler(t)
	err := profiler.Start()
	require.NoError(t, err)

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)
	defer func() {
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(0)
	}()

	// Act - Generate multiple profiles concurrently
	done := make(chan error, 4)

	go func() {
		done <- profiler.WriteHeapProfile()
	}()

	go func() {
		done <- profiler.WriteGoroutineProfile()
	}()

	go func() {
		done <- profiler.WriteBlockProfile()
	}()

	go func() {
		done <- profiler.WriteMutexProfile()
	}()

	// Assert - All should succeed
	for i := 0; i < 4; i++ {
		err := <-done
		assert.NoError(t, err)
	}

	// Verify all profiles were created
	profiles, err := profiler.ListProfiles()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(profiles), 4)
}
