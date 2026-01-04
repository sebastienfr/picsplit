//go:build !windows
// +build !windows

package handler

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateMerge_TargetScanError tests error handling when target directory has permission issues
func TestValidateMerge_TargetScanError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source
	source := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}
	photoPath := filepath.Join(source, "photo.jpg")
	if err := os.WriteFile(photoPath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create photo: %v", err)
	}

	// Create target with restricted permissions
	target := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatalf("failed to create target: %v", err)
	}

	// Create a subdirectory with no read permissions
	restrictedDir := filepath.Join(target, "restricted")
	if err := os.MkdirAll(restrictedDir, 0755); err != nil {
		t.Fatalf("failed to create restricted dir: %v", err)
	}
	if err := os.Chmod(restrictedDir, 0000); err != nil {
		t.Fatalf("failed to change permissions: %v", err)
	}
	defer os.Chmod(restrictedDir, 0755) // Restore for cleanup

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
	}

	// Should report error about scanning target
	err := validateMerge(cfg)
	if err == nil {
		t.Error("validateMerge() should fail when target cannot be scanned")
	}
}

// TestMerge_ErrorMovingFile tests error handling when file move fails due to permissions
func TestMerge_ErrorMovingFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can't test permission errors)")
	}

	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatalf("failed to create source: %v", err)
	}
	photoPath := filepath.Join(source, "photo.jpg")
	if err := os.WriteFile(photoPath, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create photo: %v", err)
	}

	target := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(target, 0755); err != nil {
		t.Fatal(err)
	}

	// Make target read-only to cause move error
	if err := os.Chmod(target, 0444); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chmod(target, 0755) // Cleanup, ignore error
	}()

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		Mode:          ModeRun,
	}

	err := Merge(cfg)
	if err == nil {
		t.Error("Merge() should error when file move fails")
	}
}
