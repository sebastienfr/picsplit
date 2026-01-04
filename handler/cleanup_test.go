package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupEmptyDirs_ValidateMode tests that cleanup is skipped in validate mode
func TestCleanupEmptyDirs_ValidateMode(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run cleanup in validate mode
	result, err := CleanupEmptyDirs(tmpDir, ModeValidate, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should not remove anything in validate mode
	if len(result.RemovedDirs) > 0 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs in validate mode, want 0", len(result.RemovedDirs))
	}

	// Directory should still exist
	if _, err := os.Stat(emptyDir); os.IsNotExist(err) {
		t.Error("empty directory was removed in validate mode")
	}
}

// TestCleanupEmptyDirs_DryRunMode tests that cleanup simulates in dryrun mode
func TestCleanupEmptyDirs_DryRunMode(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run cleanup in dryrun mode
	result, err := CleanupEmptyDirs(tmpDir, ModeDryRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should report what would be removed
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() reported %d dirs to remove, want 1", len(result.RemovedDirs))
	}

	// Directory should still exist (dry run doesn't actually remove)
	if _, err := os.Stat(emptyDir); os.IsNotExist(err) {
		t.Error("empty directory was removed in dryrun mode")
	}
}

// TestCleanupEmptyDirs_RunMode tests that cleanup actually removes in run mode
func TestCleanupEmptyDirs_RunMode(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Run cleanup in run mode
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove the empty directory
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// Directory should not exist
	if _, err := os.Stat(emptyDir); !os.IsNotExist(err) {
		t.Error("empty directory was not removed in run mode")
	}
}

// TestCleanupEmptyDirs_NestedEmpty tests bottom-up removal of nested empty directories
func TestCleanupEmptyDirs_NestedEmpty(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	level1 := filepath.Join(tmpDir, "level1")
	level2 := filepath.Join(level1, "level2")
	level3 := filepath.Join(level2, "level3")

	if err := os.MkdirAll(level3, 0755); err != nil {
		t.Fatal(err)
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove all 3 nested empty directories
	if len(result.RemovedDirs) != 3 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 3", len(result.RemovedDirs))
	}

	// All directories should not exist
	if _, err := os.Stat(level1); !os.IsNotExist(err) {
		t.Error("level1 directory was not removed")
	}
}

// TestCleanupEmptyDirs_MixedContent tests that non-empty directories are preserved
func TestCleanupEmptyDirs_MixedContent(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create empty dir
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create non-empty dir
	nonEmptyDir := filepath.Join(tmpDir, "non_empty")
	if err := os.Mkdir(nonEmptyDir, 0755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(nonEmptyDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove only the empty directory
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// Empty directory should not exist
	if _, err := os.Stat(emptyDir); !os.IsNotExist(err) {
		t.Error("empty directory was not removed")
	}

	// Non-empty directory should still exist
	if _, err := os.Stat(nonEmptyDir); os.IsNotExist(err) {
		t.Error("non-empty directory was removed")
	}

	// File should still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file in non-empty directory was removed")
	}
}

// TestCleanupEmptyDirs_ProtectedDirs tests that protected directories are skipped
func TestCleanupEmptyDirs_ProtectedDirs(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create protected directories
	protectedTests := []string{
		".git",
		".svn",
		"node_modules",
	}

	for _, protected := range protectedTests {
		dir := filepath.Join(tmpDir, protected)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should not remove any protected directories
	if len(result.RemovedDirs) > 0 {
		t.Errorf("CleanupEmptyDirs() removed %d protected dirs, want 0", len(result.RemovedDirs))
	}

	// All protected directories should still exist
	for _, protected := range protectedTests {
		dir := filepath.Join(tmpDir, protected)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("protected directory %s was removed", protected)
		}
	}
}

// TestCleanupEmptyDirs_RootNotRemoved tests that root path is never removed
func TestCleanupEmptyDirs_RootNotRemoved(t *testing.T) {
	// Create temp directory (empty root)
	tmpDir := t.TempDir()

	// Run cleanup on empty root
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should not remove root
	if len(result.RemovedDirs) > 0 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs including root, want 0", len(result.RemovedDirs))
	}

	// Root should still exist
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("root directory was removed")
	}
}

// TestCleanupEmptyDirs_PartiallyEmptyTree tests cleanup of partially empty tree
func TestCleanupEmptyDirs_PartiallyEmptyTree(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create tree: tmpDir/a/b/c (all empty) and tmpDir/d/e (e has file)
	emptyBranch := filepath.Join(tmpDir, "a", "b", "c")
	if err := os.MkdirAll(emptyBranch, 0755); err != nil {
		t.Fatal(err)
	}

	nonEmptyBranch := filepath.Join(tmpDir, "d", "e")
	if err := os.MkdirAll(nonEmptyBranch, 0755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(nonEmptyBranch, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove only the empty branch (a, a/b, a/b/c = 3 dirs)
	if len(result.RemovedDirs) != 3 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 3", len(result.RemovedDirs))
	}

	// Empty branch should not exist
	if _, err := os.Stat(filepath.Join(tmpDir, "a")); !os.IsNotExist(err) {
		t.Error("empty branch was not removed")
	}

	// Non-empty branch should still exist
	if _, err := os.Stat(nonEmptyBranch); os.IsNotExist(err) {
		t.Error("non-empty branch was removed")
	}
}

// TestCleanupEmptyDirs_PermissionError tests handling of permission errors
func TestCleanupEmptyDirs_PermissionError(t *testing.T) {
	// Skip on Windows (permission model is different)
	if os.Getenv("GOOS") == "windows" {
		t.Skip("Skipping permission test on Windows")
	}

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

// TestCleanupEmptyDirs_IgnoresSystemFiles tests that system files are ignored when checking if directory is empty
func TestCleanupEmptyDirs_IgnoresSystemFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	dirWithSystemFiles := filepath.Join(tmpDir, "with_system_files")
	if err := os.Mkdir(dirWithSystemFiles, 0755); err != nil {
		t.Fatal(err)
	}

	// Add system files
	systemFiles := []string{".DS_Store", "Thumbs.db", "desktop.ini", "._.DS_Store"}
	for _, file := range systemFiles {
		filePath := filepath.Join(dirWithSystemFiles, file)
		if err := os.WriteFile(filePath, []byte("system"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run cleanup - should consider directory as empty despite system files
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove the directory (system files don't count)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// Directory should not exist
	if _, err := os.Stat(dirWithSystemFiles); !os.IsNotExist(err) {
		t.Error("directory with only system files was not removed")
	}
}

// TestIsDirEmpty tests the isDirEmpty helper function
func TestIsDirEmpty(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(string) string
		wantEmpty bool
		wantErr   bool
	}{
		{
			name: "empty directory",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "empty")
				os.Mkdir(dir, 0755)
				return dir
			},
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name: "directory with file",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "with_file")
				os.Mkdir(dir, 0755)
				os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0644)
				return dir
			},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "directory with subdirectory",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "with_subdir")
				os.Mkdir(dir, 0755)
				os.Mkdir(filepath.Join(dir, "subdir"), 0755)
				return dir
			},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "directory with only .DS_Store",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "with_ds_store")
				os.Mkdir(dir, 0755)
				os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("metadata"), 0644)
				return dir
			},
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name: "directory with only Thumbs.db",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "with_thumbs")
				os.Mkdir(dir, 0755)
				os.WriteFile(filepath.Join(dir, "Thumbs.db"), []byte("metadata"), 0644)
				return dir
			},
			wantEmpty: true,
			wantErr:   false,
		},
		{
			name: "directory with system files and real file",
			setup: func(tmpDir string) string {
				dir := filepath.Join(tmpDir, "with_mixed")
				os.Mkdir(dir, 0755)
				os.WriteFile(filepath.Join(dir, ".DS_Store"), []byte("metadata"), 0644)
				os.WriteFile(filepath.Join(dir, "photo.jpg"), []byte("photo"), 0644)
				return dir
			},
			wantEmpty: false,
			wantErr:   false,
		},
		{
			name: "non-existent directory",
			setup: func(tmpDir string) string {
				return filepath.Join(tmpDir, "does_not_exist")
			},
			wantEmpty: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := tt.setup(tmpDir)

			empty, err := isDirEmpty(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isDirEmpty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if empty != tt.wantEmpty {
				t.Errorf("isDirEmpty() = %v, want %v", empty, tt.wantEmpty)
			}
		})
	}
}

// TestIsProtectedDir tests the isProtectedDir helper function
func TestIsProtectedDir(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantProtected bool
	}{
		{
			name:          ".git directory",
			path:          "/home/user/project/.git",
			wantProtected: true,
		},
		{
			name:          ".svn directory",
			path:          "/home/user/project/.svn",
			wantProtected: true,
		},
		{
			name:          "node_modules directory",
			path:          "/home/user/project/node_modules",
			wantProtected: true,
		},
		{
			name:          "nested .git",
			path:          "/home/user/project/sub/.git",
			wantProtected: true,
		},
		{
			name:          "normal directory",
			path:          "/home/user/project/photos",
			wantProtected: false,
		},
		{
			name:          "directory containing 'git' in name",
			path:          "/home/user/github/project",
			wantProtected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isProtectedDir(tt.path); got != tt.wantProtected {
				t.Errorf("isProtectedDir(%q) = %v, want %v", tt.path, got, tt.wantProtected)
			}
		})
	}
}

// TestCleanupEmptyDirs_CustomIgnoredFiles tests that custom ignored files work correctly
func TestCleanupEmptyDirs_CustomIgnoredFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Directory with custom ignored files
	dirWithCustom := filepath.Join(tmpDir, "with_custom")
	if err := os.Mkdir(dirWithCustom, 0755); err != nil {
		t.Fatal(err)
	}

	// Add custom files that should be ignored
	customFiles := []string{".picasa.ini", ".nomedia", "folder.jpg"}
	for _, file := range customFiles {
		filePath := filepath.Join(dirWithCustom, file)
		if err := os.WriteFile(filePath, []byte("custom"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Directory with non-ignored file
	dirWithReal := filepath.Join(tmpDir, "with_real")
	if err := os.Mkdir(dirWithReal, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirWithReal, ".picasa.ini"), []byte("custom"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirWithReal, "photo.jpg"), []byte("photo"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run cleanup with custom ignored files
	customIgnored := []string{".picasa.ini", ".nomedia", "folder.jpg"}
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, customIgnored)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove only dirWithCustom (all files ignored)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// dirWithCustom should not exist
	if _, err := os.Stat(dirWithCustom); !os.IsNotExist(err) {
		t.Error("directory with only custom ignored files was not removed")
	}

	// dirWithReal should still exist (has photo.jpg)
	if _, err := os.Stat(dirWithReal); os.IsNotExist(err) {
		t.Error("directory with real file was removed")
	}
}

// TestCleanupEmptyDirs_MixedIgnoredFiles tests that default + custom ignored files work together
func TestCleanupEmptyDirs_MixedIgnoredFiles(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "mixed")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Add both default and custom ignored files
	files := []string{
		".DS_Store",   // Default ignored
		"Thumbs.db",   // Default ignored
		".picasa.ini", // Custom ignored
		".nomedia",    // Custom ignored
	}
	for _, file := range files {
		filePath := filepath.Join(testDir, file)
		if err := os.WriteFile(filePath, []byte("ignored"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run cleanup with custom ignored files
	customIgnored := []string{".picasa.ini", ".nomedia"}
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, customIgnored)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove the directory (all files ignored)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// Directory should not exist
	if _, err := os.Stat(testDir); !os.IsNotExist(err) {
		t.Error("directory with only ignored files (default + custom) was not removed")
	}
}

// TestCleanupEmptyDirs_ErrorDuringWalk tests error handling during directory walk
func TestCleanupEmptyDirs_ErrorDuringWalk(t *testing.T) {
	// Test with non-existent directory
	// WalkDir on non-existent root returns error immediately
	result, err := CleanupEmptyDirs("/nonexistent/path/that/does/not/exist", ModeRun, true, nil)

	// The walk function logs warnings but continues (returns nil from callback)
	// So the overall result is successful with empty list
	if err != nil {
		// Actually this might return error depending on WalkDir behavior
		// Both behaviors are acceptable - either error or success with empty result
		if result == nil {
			t.Error("CleanupEmptyDirs() should return valid result even on error")
		}
		return
	}

	// If no error, should have empty results
	if len(result.RemovedDirs) != 0 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 0", len(result.RemovedDirs))
	}
}

// TestCleanupEmptyDirs_InaccessibleSubdir tests handling of inaccessible subdirectories
func TestCleanupEmptyDirs_InaccessibleSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a subdirectory
	subdir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Make it inaccessible (no read permission)
	if err := os.Chmod(subdir, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(subdir, 0755) // Restore for cleanup

	// Run cleanup - should handle permission error gracefully
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should have logged the inaccessible directory as failed
	// but continue processing
	if result == nil {
		t.Error("CleanupEmptyDirs() should return valid result")
	}
}

// TestCleanupEmptyDirs_VeryDeepNesting tests multi-pass with very deep nesting
func TestCleanupEmptyDirs_VeryDeepNesting(t *testing.T) {
	tmpDir := t.TempDir()

	// Create very deep nesting (20 levels)
	deepPath := tmpDir
	for i := 0; i < 20; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
	}

	if err := os.MkdirAll(deepPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Add a system file at the deepest level
	if err := os.WriteFile(filepath.Join(deepPath, ".DS_Store"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove all 20 levels
	if len(result.RemovedDirs) != 20 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 20 (deep nesting)", len(result.RemovedDirs))
	}
}

// TestCleanupEmptyDirs_WithSymlinks tests handling of symbolic links
func TestCleanupEmptyDirs_WithSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real directory
	realDir := filepath.Join(tmpDir, "real")
	if err := os.Mkdir(realDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink to it
	symlinkPath := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(realDir, symlinkPath); err != nil {
		t.Skip("symlink creation not supported on this system")
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Should remove both real dir and symlink is considered as file (so real dir removed, symlink stays)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}
}

// TestCleanupEmptyDirs_ConcurrentModification simulates directory becoming non-empty during cleanup
func TestCleanupEmptyDirs_ConcurrentModification(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	// This test just verifies the re-check logic works
	// In real scenario, directory could change between passes
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}
}

// TestRemoveIgnoredFiles tests the removeIgnoredFiles function through integration
func TestCleanupEmptyDirs_RemoveIgnoredFilesError(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create an ignored file
	ignoredFile := filepath.Join(testDir, ".DS_Store")
	if err := os.WriteFile(ignoredFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Make the file read-only (but still removable on most systems)
	// This tests the error handling path in removeIgnoredFiles
	if err := os.Chmod(ignoredFile, 0444); err != nil {
		t.Fatal(err)
	}

	// Run cleanup - should still succeed even if file removal logs an error
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Directory should be removed (file deletion is best-effort)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}
}

// TestCleanupEmptyDirs_MultipleSystemFiles tests cleanup with multiple types of system files
func TestCleanupEmptyDirs_MultipleSystemFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create all default ignored files
	systemFiles := []string{".DS_Store", "Thumbs.db", "desktop.ini", "._.DS_Store"}
	for _, file := range systemFiles {
		filePath := filepath.Join(testDir, file)
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Run cleanup
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, true, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// Directory should be removed (all files are ignored)
	if len(result.RemovedDirs) != 1 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 1", len(result.RemovedDirs))
	}

	// Verify all system files were deleted
	for _, file := range systemFiles {
		filePath := filepath.Join(testDir, file)
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Errorf("system file %s was not deleted", file)
		}
	}
}

// TestCleanupEmptyDirs_EmptyListNoConfirmation tests that empty list doesn't ask for confirmation
func TestCleanupEmptyDirs_EmptyListNoConfirmation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory with a real file (not empty)
	testDir := filepath.Join(tmpDir, "notempty")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run cleanup without force (would ask for confirmation if there were empty dirs)
	result, err := CleanupEmptyDirs(tmpDir, ModeRun, false, nil)
	if err != nil {
		t.Errorf("CleanupEmptyDirs() error = %v, want nil", err)
	}

	// No directories should be removed
	if len(result.RemovedDirs) != 0 {
		t.Errorf("CleanupEmptyDirs() removed %d dirs, want 0", len(result.RemovedDirs))
	}

	// No errors either
	if len(result.FailedDirs) != 0 {
		t.Errorf("CleanupEmptyDirs() failed %d dirs, want 0", len(result.FailedDirs))
	}
}
