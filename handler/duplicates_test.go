package handler

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewDuplicateDetector tests detector initialization
func TestNewDuplicateDetector(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled detector",
			enabled: true,
		},
		{
			name:    "disabled detector",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detector := NewDuplicateDetector(tt.enabled)
			if detector == nil {
				t.Fatal("NewDuplicateDetector() returned nil")
			}
			if detector.enabled != tt.enabled {
				t.Errorf("NewDuplicateDetector() enabled = %v, want %v", detector.enabled, tt.enabled)
			}
			if len(detector.hashes) != 0 {
				t.Errorf("NewDuplicateDetector() hashes not empty")
			}
			if len(detector.duplicates) != 0 {
				t.Errorf("NewDuplicateDetector() duplicates not empty")
			}
		})
	}
}

// TestDuplicateDetector_Disabled tests that disabled detector doesn't process
func TestDuplicateDetector_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(false)
	detector.AddFile(file1, 7)

	isDup, original, err := detector.Check(file1, 7)
	if err != nil {
		t.Errorf("Check() error = %v, want nil", err)
	}
	if isDup {
		t.Error("Check() isDup = true, want false for disabled detector")
	}
	if original != "" {
		t.Errorf("Check() original = %q, want empty", original)
	}
}

// TestDuplicateDetector_UniqueFiles tests detection with unique files
func TestDuplicateDetector_UniqueFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create 3 unique files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("content3"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)
	detector.AddFile(file1, 8)
	detector.AddFile(file2, 8)
	detector.AddFile(file3, 8)

	// Check all files - none should be duplicates
	for _, file := range []string{file1, file2, file3} {
		isDup, _, err := detector.Check(file, 8)
		if err != nil {
			t.Errorf("Check(%s) error = %v", filepath.Base(file), err)
		}
		if isDup {
			t.Errorf("Check(%s) isDup = true, want false for unique file", filepath.Base(file))
		}
	}

	if len(detector.GetDuplicates()) != 0 {
		t.Errorf("GetDuplicates() = %d, want 0", len(detector.GetDuplicates()))
	}
}

// TestDuplicateDetector_DuplicateFiles tests detection of actual duplicates
func TestDuplicateDetector_DuplicateFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create original file
	original := filepath.Join(tmpDir, "original.txt")
	content := []byte("this is the original content")
	if err := os.WriteFile(original, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Create exact duplicates
	dup1 := filepath.Join(tmpDir, "duplicate1.txt")
	dup2 := filepath.Join(tmpDir, "duplicate2.txt")
	if err := os.WriteFile(dup1, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dup2, content, 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)
	size := int64(len(content))
	detector.AddFile(original, size)
	detector.AddFile(dup1, size)
	detector.AddFile(dup2, size)

	// Check original - should not be duplicate
	isDup, _, err := detector.Check(original, size)
	if err != nil {
		t.Errorf("Check(original) error = %v", err)
	}
	if isDup {
		t.Error("Check(original) isDup = true, want false")
	}

	// Check first duplicate
	isDup, orig, err := detector.Check(dup1, size)
	if err != nil {
		t.Errorf("Check(dup1) error = %v", err)
	}
	if !isDup {
		t.Error("Check(dup1) isDup = false, want true")
	}
	if orig != original {
		t.Errorf("Check(dup1) original = %q, want %q", orig, original)
	}

	// Check second duplicate
	isDup, orig, err = detector.Check(dup2, size)
	if err != nil {
		t.Errorf("Check(dup2) error = %v", err)
	}
	if !isDup {
		t.Error("Check(dup2) isDup = false, want true")
	}
	if orig != original {
		t.Errorf("Check(dup2) original = %q, want %q", orig, original)
	}

	// Verify duplicates map
	dups := detector.GetDuplicates()
	if len(dups) != 2 {
		t.Errorf("GetDuplicates() = %d, want 2", len(dups))
	}
	if dups[dup1] != original {
		t.Errorf("GetDuplicates()[dup1] = %q, want %q", dups[dup1], original)
	}
	if dups[dup2] != original {
		t.Errorf("GetDuplicates()[dup2] = %q, want %q", dups[dup2], original)
	}
}

// TestDuplicateDetector_SizeOptimization tests size-based pre-filtering
func TestDuplicateDetector_SizeOptimization(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files of different sizes
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	if err := os.WriteFile(file1, []byte("short"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("medium content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("this is a much longer content for testing"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)

	// Add files with their actual sizes
	info1, _ := os.Stat(file1)
	info2, _ := os.Stat(file2)
	info3, _ := os.Stat(file3)

	detector.AddFile(file1, info1.Size())
	detector.AddFile(file2, info2.Size())
	detector.AddFile(file3, info3.Size())

	// Check files - each has unique size, so no hashing should occur
	// (we can verify this indirectly by checking no duplicates are found)
	isDup1, _, err := detector.Check(file1, info1.Size())
	if err != nil {
		t.Errorf("Check(file1) error = %v", err)
	}
	if isDup1 {
		t.Error("Check(file1) isDup = true, want false")
	}

	isDup2, _, err := detector.Check(file2, info2.Size())
	if err != nil {
		t.Errorf("Check(file2) error = %v", err)
	}
	if isDup2 {
		t.Error("Check(file2) isDup = true, want false")
	}

	isDup3, _, err := detector.Check(file3, info3.Size())
	if err != nil {
		t.Errorf("Check(file3) error = %v", err)
	}
	if isDup3 {
		t.Error("Check(file3) isDup = true, want false")
	}
}

// TestDuplicateDetector_SameSizeDifferentContent tests files with same size but different content
func TestDuplicateDetector_SameSizeDifferentContent(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two files with same size but different content
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	content1 := []byte("hello")
	content2 := []byte("world")

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)
	size := int64(5)
	detector.AddFile(file1, size)
	detector.AddFile(file2, size)

	// Check both files - should not be duplicates (different hash)
	isDup1, _, err := detector.Check(file1, size)
	if err != nil {
		t.Errorf("Check(file1) error = %v", err)
	}
	if isDup1 {
		t.Error("Check(file1) isDup = true, want false")
	}

	isDup2, _, err := detector.Check(file2, size)
	if err != nil {
		t.Errorf("Check(file2) error = %v", err)
	}
	if isDup2 {
		t.Error("Check(file2) isDup = true, want false")
	}

	if len(detector.GetDuplicates()) != 0 {
		t.Errorf("GetDuplicates() = %d, want 0 (same size, different content)", len(detector.GetDuplicates()))
	}
}

// TestDuplicateDetector_NonExistentFile tests error handling for non-existent files
func TestDuplicateDetector_NonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	existing := filepath.Join(tmpDir, "existing.txt")

	// Create one existing file so size group has > 1 file (forces hashing)
	if err := os.WriteFile(existing, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)
	size := int64(100)
	detector.AddFile(nonExistent, size) // Will be in size group
	detector.AddFile(existing, size)    // Same size group

	// Try to check non-existent file (will try to hash)
	isDup, original, err := detector.Check(nonExistent, size)
	if err == nil {
		t.Error("Check() error = nil, want error for non-existent file")
	}
	if isDup {
		t.Error("Check() isDup = true, want false on error")
	}
	if original != "" {
		t.Errorf("Check() original = %q, want empty on error", original)
	}
}

// TestDuplicateDetector_GetStats tests statistics retrieval
func TestDuplicateDetector_GetStats(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt") // duplicate of file1
	file3 := filepath.Join(tmpDir, "file3.txt") // different size
	file4 := filepath.Join(tmpDir, "file4.txt") // different size

	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte("different"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file4, []byte("also different content"), 0644); err != nil {
		t.Fatal(err)
	}

	detector := NewDuplicateDetector(true)
	detector.AddFile(file1, 7)
	detector.AddFile(file2, 7)
	detector.AddFile(file3, 9)
	detector.AddFile(file4, 22)

	// Check files
	detector.Check(file1, 7)
	detector.Check(file2, 7)
	detector.Check(file3, 9)
	detector.Check(file4, 22)

	totalFiles, uniqueSizes, potentialDuplicates, confirmedDuplicates := detector.GetStats()

	if totalFiles != 4 {
		t.Errorf("GetStats() totalFiles = %d, want 4", totalFiles)
	}
	if uniqueSizes != 2 { // file3 and file4 have unique sizes
		t.Errorf("GetStats() uniqueSizes = %d, want 2", uniqueSizes)
	}
	if potentialDuplicates != 2 { // file1 and file2 have same size
		t.Errorf("GetStats() potentialDuplicates = %d, want 2", potentialDuplicates)
	}
	if confirmedDuplicates != 1 { // file2 is duplicate of file1
		t.Errorf("GetStats() confirmedDuplicates = %d, want 1", confirmedDuplicates)
	}
}

// TestSha256File tests the SHA256 hashing function
func TestSha256File(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Compute hash
	hash1, err := sha256File(testFile)
	if err != nil {
		t.Errorf("sha256File() error = %v, want nil", err)
	}
	if hash1 == "" {
		t.Error("sha256File() returned empty hash")
	}

	// Compute hash again - should be identical
	hash2, err := sha256File(testFile)
	if err != nil {
		t.Errorf("sha256File() error = %v, want nil", err)
	}
	if hash1 != hash2 {
		t.Errorf("sha256File() not deterministic: %s != %s", hash1, hash2)
	}

	// Create different file with same content
	testFile2 := filepath.Join(tmpDir, "test2.txt")
	if err := os.WriteFile(testFile2, content, 0644); err != nil {
		t.Fatal(err)
	}

	hash3, err := sha256File(testFile2)
	if err != nil {
		t.Errorf("sha256File() error = %v, want nil", err)
	}
	if hash1 != hash3 {
		t.Errorf("sha256File() different hash for same content: %s != %s", hash1, hash3)
	}

	// Create file with different content
	testFile3 := filepath.Join(tmpDir, "test3.txt")
	if err := os.WriteFile(testFile3, []byte("different"), 0644); err != nil {
		t.Fatal(err)
	}

	hash4, err := sha256File(testFile3)
	if err != nil {
		t.Errorf("sha256File() error = %v, want nil", err)
	}
	if hash1 == hash4 {
		t.Error("sha256File() same hash for different content")
	}
}

// TestSha256File_NonExistent tests error handling for non-existent files
func TestSha256File_NonExistent(t *testing.T) {
	_, err := sha256File("/nonexistent/file.txt")
	if err == nil {
		t.Error("sha256File() error = nil, want error for non-existent file")
	}
}

// TestSha256File_Directory tests error handling for directories
func TestSha256File_Directory(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := sha256File(tmpDir)
	if err == nil {
		t.Error("sha256File() error = nil, want error for directory")
	}
}
