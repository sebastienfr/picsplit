//go:build !windows
// +build !windows

package handler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestValidate_PermissionErrors tests validation with file permission errors
func TestValidate_PermissionErrors(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file with no read permissions
	restrictedFile := filepath.Join(tempDir, "restricted.jpg")
	if err := os.WriteFile(restrictedFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Remove read permissions
	if err := os.Chmod(restrictedFile, 0000); err != nil {
		t.Fatalf("failed to change permissions: %v", err)
	}
	defer os.Chmod(restrictedFile, 0644) // Restore for cleanup

	cfg := &Config{
		BasePath: tempDir,
		Delta:    30 * time.Minute,
	}

	report, err := Validate(cfg)
	if err != nil {
		t.Fatalf("Validate() failed: %v", err)
	}

	// Should have permission error
	hasPermissionError := false
	for _, err := range report.Errors {
		if err.Type == ErrTypePermission {
			hasPermissionError = true
			break
		}
	}

	if !hasPermissionError {
		t.Error("expected permission error")
	}

	// Should have critical errors
	if !report.HasCriticalErrors() {
		t.Error("expected critical errors")
	}
}
