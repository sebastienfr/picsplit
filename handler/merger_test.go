package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ========================================
// Helper Functions for Tests
// ========================================

// createTestFileInDir creates a test file with content in a specific directory
//
//nolint:unparam // Path return useful for debugging, even if not used in all tests
func createTestFileInDir(t *testing.T, dir, name, content string) string {
	t.Helper()

	path := filepath.Join(dir, name)

	// Create parent directories if needed
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		t.Fatalf("failed to create parent dir %s: %v", parentDir, err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create file %s: %v", path, err)
	}

	return path
}

// ========================================
// Tests for Helper Functions
// ========================================

func TestIsMediaFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		// Valid media files - photos
		{"JPEG lowercase", "photo.jpg", true},
		{"JPEG uppercase", "PHOTO.JPG", true},
		{"JPEG alternate", "image.jpeg", true},
		{"HEIC", "img.heic", true},
		{"WebP", "photo.webp", true},

		// Valid media files - videos
		{"MOV", "video.mov", true},
		{"MP4", "clip.mp4", true},
		{"AVI", "movie.avi", true},

		// Valid media files - raw
		{"NEF", "raw.nef", true},
		{"CR2", "canon.cr2", true},
		{"DNG", "adobe.dng", true},

		// Invalid files
		{"Text file", "readme.txt", false},
		{"Document", "notes.pdf", false},
		{"Config", "config.json", false},
		{"No extension", "file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isMediaFile(tt.filename)
			if result != tt.expected {
				t.Errorf("isMediaFile(%q) = %v, want %v", tt.filename, result, tt.expected)
			}
		})
	}
}

func TestIsMediaFolder(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		wantError bool
		errorText string
	}{
		{
			name: "Valid folder with only photos",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo1.jpg", "data")
				createTestFileInDir(t, dir, "photo2.jpeg", "data")
				return dir
			},
			wantError: false,
		},
		{
			name: "Valid folder with photos and mov subdir",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				createTestFileInDir(t, dir, "mov/video.mov", "data")
				return dir
			},
			wantError: false,
		},
		{
			name: "Valid folder with photos, mov and raw subdirs",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				createTestFileInDir(t, dir, "mov/video.mp4", "data")
				createTestFileInDir(t, dir, "raw/image.nef", "data")
				return dir
			},
			wantError: false,
		},
		{
			name: "Invalid - contains non-media file",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				createTestFileInDir(t, dir, "readme.txt", "data")
				return dir
			},
			wantError: true,
			errorText: "non-media file",
		},
		{
			name: "Invalid - contains non-allowed subdir",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				if err := os.Mkdir(filepath.Join(dir, "thumbnails"), 0755); err != nil {
					t.Fatal(err)
				}
				return dir
			},
			wantError: true,
			errorText: "non-media subdirectory",
		},
		{
			name: "Invalid - GPS folder structure (would be rejected)",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				// Simulate GPS location folder with nested time folders
				gpsDir := filepath.Join(dir, "48.8566N-2.3522E")
				if err := os.Mkdir(gpsDir, 0755); err != nil {
					t.Fatal(err)
				}
				createTestFileInDir(t, gpsDir, "2025 - 0616 - 0945/photo.jpg", "data")
				return gpsDir
			},
			wantError: true,
			errorText: "non-media subdirectory",
		},
		{
			name: "Invalid - mov folder with non-media file",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				createTestFileInDir(t, dir, "mov/video.mov", "data")
				createTestFileInDir(t, dir, "mov/readme.txt", "text")
				return dir
			},
			wantError: true,
			errorText: "non-media file",
		},
		{
			name: "Invalid - raw folder with non-media file",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				createTestFileInDir(t, dir, "photo.jpg", "data")
				createTestFileInDir(t, dir, "raw/image.nef", "data")
				createTestFileInDir(t, dir, "raw/notes.doc", "text")
				return dir
			},
			wantError: true,
			errorText: "non-media file",
		},
		{
			name: "Empty folder",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			wantError: false, // Empty is technically valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folderPath := tt.setupFunc(t)
			err := isMediaFolder(folderPath)

			if tt.wantError {
				if err == nil {
					t.Errorf("isMediaFolder() expected error containing %q, got nil", tt.errorText)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("isMediaFolder() error = %v, want error containing %q", err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("isMediaFolder() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateUniqueName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	existingFile := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Generate unique name
	uniqueName := generateUniqueName(existingFile)

	// Should be photo_1.jpg
	expected := filepath.Join(tmpDir, "photo_1.jpg")
	if uniqueName != expected {
		t.Errorf("generateUniqueName() = %q, want %q", uniqueName, expected)
	}

	// Create photo_1.jpg and try again
	if err := os.WriteFile(expected, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	uniqueName2 := generateUniqueName(existingFile)
	expected2 := filepath.Join(tmpDir, "photo_2.jpg")
	if uniqueName2 != expected2 {
		t.Errorf("generateUniqueName() = %q, want %q", uniqueName2, expected2)
	}
}

func TestCollectFilesRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test structure with media files
	createTestFileInDir(t, tmpDir, "photo1.jpg", "content1")
	createTestFileInDir(t, tmpDir, "photo2.jpeg", "content2")
	createTestFileInDir(t, tmpDir, "mov/video.mov", "video")
	createTestFileInDir(t, tmpDir, "raw/photo.nef", "raw")

	files, err := collectFilesRecursive(tmpDir)
	if err != nil {
		t.Fatalf("collectFilesRecursive() error = %v", err)
	}

	// Should have 4 files
	if len(files) != 4 {
		t.Errorf("collectFilesRecursive() returned %d files, want 4", len(files))
	}

	// Verify all files are present (not checking order)
	expectedFiles := map[string]bool{
		"photo1.jpg":                      false,
		"photo2.jpeg":                     false,
		filepath.Join("mov", "video.mov"): false,
		filepath.Join("raw", "photo.nef"): false,
	}

	for _, file := range files {
		relPath, _ := filepath.Rel(tmpDir, file)
		if _, ok := expectedFiles[relPath]; ok {
			expectedFiles[relPath] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("expected file %q not found", file)
		}
	}
}

// ========================================
// Tests for Validation
// ========================================

func TestValidateMergeFolders_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid source folders with media files
	source1 := filepath.Join(tmpDir, "2025 - 0616 - 0945")
	source2 := filepath.Join(tmpDir, "2025 - 0616 - 1430")

	createTestFileInDir(t, source1, "photo1.jpg", "data")
	createTestFileInDir(t, source2, "photo2.jpeg", "data")

	target := filepath.Join(tmpDir, "merged")

	err := validateMergeFolders([]string{source1, source2}, target)
	if err != nil {
		t.Errorf("validateMergeFolders() unexpected error: %v", err)
	}
}

func TestValidateMergeFolders_NonMediaFolderRejected(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder with non-media content (e.g., GPS location folder with nested structure)
	gpsFolder := filepath.Join(tmpDir, "48.8566N-2.3522E")
	createTestFileInDir(t, gpsFolder, "2025 - 0616 - 0945/photo.jpg", "data")

	target := filepath.Join(tmpDir, "merged")

	err := validateMergeFolders([]string{gpsFolder}, target)
	if err == nil {
		t.Error("validateMergeFolders() should reject non-media folder (GPS folder with nested time folders)")
	}

	// Check error message
	expectedMsg := "not a valid media folder"
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error message should contain %q, got %q", expectedMsg, err.Error())
	}
}

func TestValidateMergeFolders_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	target := filepath.Join(tmpDir, "merged")

	err := validateMergeFolders([]string{nonExistent}, target)
	if err == nil {
		t.Error("validateMergeFolders() should error when source doesn't exist")
	}
}

func TestValidateMergeFolders_InsufficientArguments(t *testing.T) {
	err := validateMergeFolders([]string{}, "target")
	if err == nil {
		t.Error("validateMergeFolders() should error with no sources")
	}
}

func TestValidateMergeFolders_FolderWithNonMediaFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create folder with mix of media and non-media files
	source := filepath.Join(tmpDir, "mixed-content")
	createTestFileInDir(t, source, "photo.jpg", "data")
	createTestFileInDir(t, source, "readme.txt", "text content")

	target := filepath.Join(tmpDir, "merged")

	err := validateMergeFolders([]string{source}, target)
	if err == nil {
		t.Error("validateMergeFolders() should reject folder with non-media files")
	}

	expectedMsg := "non-media file"
	if err != nil && !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("error message should contain %q, got %q", expectedMsg, err.Error())
	}
}

// ========================================
// Tests for Basic Merge Operations
// ========================================

func TestMerge_BasicTwoFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source folders with media files
	source1 := filepath.Join(tmpDir, "folder1")
	source2 := filepath.Join(tmpDir, "folder2")

	createTestFileInDir(t, source1, "photo1.jpg", "content1")
	createTestFileInDir(t, source1, "photo2.jpg", "content2")
	createTestFileInDir(t, source2, "photo3.jpeg", "content3")

	target := filepath.Join(tmpDir, "merged")

	cfg := &MergeConfig{
		SourceFolders: []string{source1, source2},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify target contains all files
	expectedFiles := []string{"photo1.jpg", "photo2.jpg", "photo3.jpeg"}
	for _, file := range expectedFiles {
		path := filepath.Join(target, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q not found in target", file)
		}
	}

	// Verify source folders are deleted
	if _, err := os.Stat(source1); !os.IsNotExist(err) {
		t.Error("source1 should be deleted")
	}
	if _, err := os.Stat(source2); !os.IsNotExist(err) {
		t.Error("source2 should be deleted")
	}
}

func TestMerge_MultipleFolders(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 4 source folders with media files
	sources := make([]string, 4)
	for i := 0; i < 4; i++ {
		sources[i] = filepath.Join(tmpDir, fmt.Sprintf("folder%d", i+1))
		createTestFileInDir(t, sources[i], fmt.Sprintf("photo%d.jpg", i+1), fmt.Sprintf("content%d", i+1))
	}

	target := filepath.Join(tmpDir, "merged")

	cfg := &MergeConfig{
		SourceFolders: sources,
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify all 4 files are in target
	for i := 1; i <= 4; i++ {
		path := filepath.Join(target, fmt.Sprintf("photo%d.jpg", i))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file photo%d.jpg not found", i)
		}
	}
}

func TestMerge_EmptySourceFolder(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty source folder
	source := filepath.Join(tmpDir, "empty")
	if err := os.MkdirAll(source, 0755); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "merged")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Source should still be deleted
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Error("empty source folder should be deleted")
	}
}

func TestMerge_TargetFolderExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source
	source := filepath.Join(tmpDir, "source")
	createTestFileInDir(t, source, "new.jpg", "new content")

	// Create target with existing file
	target := filepath.Join(tmpDir, "target")
	createTestFileInDir(t, target, "existing.jpg", "existing content")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Both files should exist
	if _, err := os.Stat(filepath.Join(target, "existing.jpg")); os.IsNotExist(err) {
		t.Error("existing file should remain")
	}
	if _, err := os.Stat(filepath.Join(target, "new.jpg")); os.IsNotExist(err) {
		t.Error("new file should be added")
	}
}

// ========================================
// Tests for Structure Preservation
// ========================================

func TestMerge_PreserveMovRawStructure(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")

	// Create files in different subdirectories
	createTestFileInDir(t, source, "photo.jpg", "photo")
	createTestFileInDir(t, source, "mov/video.mov", "video")
	createTestFileInDir(t, source, "raw/photo.nef", "raw")

	target := filepath.Join(tmpDir, "target")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify structure is preserved
	expectedPaths := []string{
		filepath.Join(target, "photo.jpg"),
		filepath.Join(target, "mov", "video.mov"),
		filepath.Join(target, "raw", "photo.nef"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %q not found", path)
		}
	}
}

// ========================================
// Tests for Conflict Handling
// ========================================

func TestMerge_NoConflict(t *testing.T) {
	tmpDir := t.TempDir()

	source1 := filepath.Join(tmpDir, "source1")
	source2 := filepath.Join(tmpDir, "source2")

	createTestFileInDir(t, source1, "photo1.jpg", "content1")
	createTestFileInDir(t, source2, "photo2.jpg", "content2")

	target := filepath.Join(tmpDir, "target")

	cfg := &MergeConfig{
		SourceFolders: []string{source1, source2},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Both files should exist without conflict
	if _, err := os.Stat(filepath.Join(target, "photo1.jpg")); os.IsNotExist(err) {
		t.Error("photo1.jpg not found")
	}
	if _, err := os.Stat(filepath.Join(target, "photo2.jpg")); os.IsNotExist(err) {
		t.Error("photo2.jpg not found")
	}
}

func TestMerge_ConflictWithForceFlag(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	target := filepath.Join(tmpDir, "target")

	// Create conflicting files
	createTestFileInDir(t, source, "conflict.jpg", "source content")
	createTestFileInDir(t, target, "conflict.jpg", "target content")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         true, // Force overwrite
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Target file should be overwritten with source content
	content, err := os.ReadFile(filepath.Join(target, "conflict.jpg"))
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "source content" {
		t.Errorf("file should be overwritten, got content: %q", string(content))
	}
}

// ========================================
// Tests for Dry-Run Mode
// ========================================

func TestMerge_DryRunMode(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	createTestFileInDir(t, source, "photo.jpg", "content")

	target := filepath.Join(tmpDir, "target")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        true, // Dry run mode
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Source should still exist (not moved)
	if _, err := os.Stat(filepath.Join(source, "photo.jpg")); os.IsNotExist(err) {
		t.Error("source file should still exist in dry-run mode")
	}

	// Target should not be created in dry-run
	// (Actually, target folder IS created even in dry-run, but files aren't moved)
	if _, err := os.Stat(filepath.Join(target, "photo.jpg")); !os.IsNotExist(err) {
		t.Error("file should not be moved in dry-run mode")
	}
}

func TestMerge_DryRunWithConflicts(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	target := filepath.Join(tmpDir, "target")

	// Create conflicting files
	createTestFileInDir(t, source, "conflict.jpg", "source")
	createTestFileInDir(t, target, "conflict.jpg", "target")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        true,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Both files should remain unchanged
	sourceContent, _ := os.ReadFile(filepath.Join(source, "conflict.jpg"))
	targetContent, _ := os.ReadFile(filepath.Join(target, "conflict.jpg"))

	if string(sourceContent) != "source" {
		t.Error("source file should be unchanged in dry-run")
	}
	if string(targetContent) != "target" {
		t.Error("target file should be unchanged in dry-run")
	}
}

// ========================================
// Tests for Error Cases
// ========================================

func TestMerge_ErrorTargetNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	createTestFileInDir(t, source, "photo.jpg", "content")

	// Create target as a file (not directory) to trigger validation error
	target := filepath.Join(tmpDir, "target")
	if err := os.WriteFile(target, []byte("blocking file"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err == nil {
		t.Error("Merge() should error when target exists as file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error should mention target not being a directory, got: %v", err)
	}
}

func TestMerge_ErrorMovingFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows (permissions work differently)")
	}
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root (can't test permission errors)")
	}

	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	createTestFileInDir(t, source, "photo.jpg", "content")

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
		DryRun:        false,
	}

	err := Merge(cfg)
	if err == nil {
		t.Error("Merge() should error when file move fails")
	}
}

func TestMerge_ValidationError(t *testing.T) {
	cfg := &MergeConfig{
		SourceFolders: []string{}, // No sources
		TargetFolder:  "/tmp/target",
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err == nil {
		t.Error("Merge() should error on validation failure")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error should mention validation, got: %v", err)
	}
}

func TestMerge_MultipleConflictsWithForce(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	target := filepath.Join(tmpDir, "target")

	// Create multiple conflicting files
	createTestFileInDir(t, source, "photo1.jpg", "source1")
	createTestFileInDir(t, source, "photo2.jpg", "source2")
	createTestFileInDir(t, source, "photo3.jpg", "source3")
	createTestFileInDir(t, target, "photo1.jpg", "target1")
	createTestFileInDir(t, target, "photo2.jpg", "target2")
	createTestFileInDir(t, target, "photo3.jpg", "target3")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         true, // Force overwrite all
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// All target files should be overwritten with source content
	for i := 1; i <= 3; i++ {
		content, err := os.ReadFile(filepath.Join(target, fmt.Sprintf("photo%d.jpg", i)))
		if err != nil {
			t.Fatal(err)
		}
		expected := fmt.Sprintf("source%d", i)
		if string(content) != expected {
			t.Errorf("photo%d.jpg should be overwritten, got content: %q, want: %q", i, string(content), expected)
		}
	}
}

func TestMerge_NestedStructurePreservation(t *testing.T) {
	tmpDir := t.TempDir()

	source := filepath.Join(tmpDir, "source")
	target := filepath.Join(tmpDir, "target")

	// Create nested structure with multiple levels
	createTestFileInDir(t, source, "photo1.jpg", "photo1")
	createTestFileInDir(t, source, "photo2.jpg", "photo2")
	createTestFileInDir(t, source, "mov/video1.mov", "video1")
	createTestFileInDir(t, source, "mov/video2.mp4", "video2")
	createTestFileInDir(t, source, "raw/image1.nef", "raw1")
	createTestFileInDir(t, source, "raw/image2.cr2", "raw2")

	cfg := &MergeConfig{
		SourceFolders: []string{source},
		TargetFolder:  target,
		Force:         false,
		DryRun:        false,
	}

	err := Merge(cfg)
	if err != nil {
		t.Fatalf("Merge() error = %v", err)
	}

	// Verify all files exist in correct locations
	expectedPaths := []string{
		filepath.Join(target, "photo1.jpg"),
		filepath.Join(target, "photo2.jpg"),
		filepath.Join(target, "mov", "video1.mov"),
		filepath.Join(target, "mov", "video2.mp4"),
		filepath.Join(target, "raw", "image1.nef"),
		filepath.Join(target, "raw", "image2.cr2"),
	}

	for _, path := range expectedPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", path)
		}
	}

	// Verify source is deleted
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Error("source folder should be deleted after merge")
	}
}
