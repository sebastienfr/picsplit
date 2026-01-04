//go:build !windows

package handler

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupEmptyDirs_PermissionError tests handling of permission errors
func TestCleanupEmptyDirs_PermissionError(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Make directory non-removable (remove write permission from parent)
	if err := os.Chmod(tmpDir, 0555); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(tmpDir, 0755) // Restore permissions for cleanup

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should report failure but not crash
	if len(result.FailedDirs) == 0 {
		t.Error("CleanupEmptyDirs() should report permission errors")
	}

	// Directory should still exist
	if _, err := os.Stat(emptyDir); os.IsNotExist(err) {
		t.Error("directory was removed despite permission error")
	}
}
