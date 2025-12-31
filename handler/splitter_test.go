package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ========================================
// Test Helpers
// ========================================

// fileInfoToMetadata converts os.FileInfo to FileMetadata for tests (uses ModTime)
func fileInfoToMetadata(fi os.FileInfo) FileMetadata {
	return FileMetadata{
		FileInfo: fi,
		DateTime: fi.ModTime(),
		GPS:      nil,
		Source:   DateSourceModTime,
	}
}

// createTestFile creates a test file with a specific modification time
func createTestFile(t *testing.T, dir, name string, modTime time.Time) {
	t.Helper()

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create test file %s: %v", name, err)
	}
	defer f.Close()

	// Write some content
	if _, err := f.WriteString("test content"); err != nil {
		t.Fatalf("failed to write to test file: %v", err)
	}

	// Set modification time
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("failed to set file time: %v", err)
	}
}

// createTestDataset creates the test dataset similar to mktest.sh
func createTestDataset(t *testing.T, baseDir string) {
	t.Helper()

	// Use current year for dates (mktest.sh uses MMDDHHSS format)
	year := time.Now().Year()

	files := []struct {
		name    string
		modTime time.Time
	}{
		{"PHOTO_01.JPG", time.Date(year, 2, 16, 8, 35, 0, 0, time.Local)},
		{"PHOTO_02.JPG", time.Date(year, 2, 16, 9, 35, 0, 0, time.Local)},
		{"PHOTO_03.JPG", time.Date(year, 2, 16, 10, 35, 0, 0, time.Local)},
		{"PHOTO_03.CR2", time.Date(year, 2, 16, 10, 35, 0, 0, time.Local)},
		{"PHOTO_04.JPG", time.Date(year, 2, 16, 11, 35, 0, 0, time.Local)},
		{"PHOTO_04.NEF", time.Date(year, 2, 16, 11, 35, 0, 0, time.Local)},
		{"PHOTO_04.MOV", time.Date(year, 2, 16, 11, 44, 0, 0, time.Local)},
		{"PHOTO_04.test", time.Date(year, 2, 16, 16, 44, 0, 0, time.Local)},
	}

	// Create TEST directory
	testDir := filepath.Join(baseDir, "TEST")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("failed to create TEST directory: %v", err)
	}

	// Create all test files
	for _, f := range files {
		createTestFile(t, baseDir, f.name, f.modTime)
	}
}

// ========================================
// Tests for Pure Functions (100% coverage)
// ========================================

func TestIsMovie(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"MOV uppercase", "video.MOV", true},
		{"mov lowercase", "video.mov", true},
		{"AVI uppercase", "video.AVI", true},
		{"avi lowercase", "video.avi", true},
		{"MP4 uppercase", "video.MP4", true},
		{"mp4 lowercase", "video.mp4", true},
		{"JPG file", "photo.jpg", false},
		{"No extension", "video", false},
		{"Multiple dots", "my.video.mov", true},
		{"Unknown extension", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)

			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			f.Close()

			fi, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}

			ctx := newDefaultExecutionContext()
			got := ctx.isMovie(fi.Name())
			if got != tt.want {
				t.Errorf("isMovie(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsPicture(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Standard formats
		{"JPG uppercase", "photo.JPG", true},
		{"jpg lowercase", "photo.jpg", true},
		{"JPEG uppercase", "photo.JPEG", true},
		{"jpeg lowercase", "photo.jpeg", true},

		// Raw formats
		{"NEF (Nikon raw)", "photo.NEF", true},
		{"CR2 (Canon raw)", "photo.CR2", true},
		{"CRW (Canon raw)", "photo.CRW", true},
		{"NRW (Nikon raw)", "photo.NRW", true},
		{"RW2 (Panasonic raw)", "photo.RW2", true},

		// Modern formats
		{"HEIC (Apple)", "photo.HEIC", true},
		{"HEIF (standard)", "photo.HEIF", true},
		{"WebP (Google)", "photo.WEBP", true},
		{"AVIF (AV1)", "photo.AVIF", true},

		// Additional raw formats
		{"DNG (Adobe)", "photo.DNG", true},
		{"ARW (Sony)", "photo.ARW", true},
		{"ORF (Olympus)", "photo.ORF", true},
		{"RAF (Fujifilm)", "photo.RAF", true},

		// Non-picture files
		{"MOV file", "video.mov", false},
		{"TXT file", "readme.txt", false},
		{"No extension", "photo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)

			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			f.Close()

			fi, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}

			ctx := newDefaultExecutionContext()
			got := ctx.isPhoto(fi.Name())
			if got != tt.want {
				t.Errorf("isPicture(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestIsRaw(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		// Traditional raw formats
		{"NEF Nikon", "photo.nef", true},
		{"NRW Nikon", "photo.nrw", true},
		{"CR2 Canon", "photo.cr2", true},
		{"CRW Canon", "photo.crw", true},
		{"RW2 Panasonic", "photo.rw2", true},

		// Modern raw formats
		{"DNG Adobe", "photo.dng", true},
		{"ARW Sony", "photo.arw", true},
		{"ORF Olympus", "photo.orf", true},
		{"RAF Fujifilm", "photo.raf", true},

		// Non-raw files
		{"JPG file", "photo.jpg", false},
		{"MOV file", "video.mov", false},
		{"HEIC file", "photo.heic", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			path := filepath.Join(tmpDir, tt.filename)

			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			f.Close()

			fi, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}

			ctx := newDefaultExecutionContext()
			got := ctx.isRaw(fi.Name())
			if got != tt.want {
				t.Errorf("isRaw(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// ========================================
// Tests for Folder Management (85-90% coverage)
// ========================================

func TestFindOrCreateFolder(t *testing.T) {
	t.Run("create new folder", func(t *testing.T) {
		tmpDir := t.TempDir()

		folderName, err := findOrCreateFolder(tmpDir, "raw", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if folderName != "raw" {
			t.Errorf("got %q, want %q", folderName, "raw")
		}

		// Verify folder was created
		fullPath := filepath.Join(tmpDir, "raw")
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("folder was not created")
		}
	})

	t.Run("folder already exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create folder manually
		rawPath := filepath.Join(tmpDir, "raw")
		if err := os.Mkdir(rawPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Should return existing folder
		folderName, err := findOrCreateFolder(tmpDir, "raw", false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if folderName != "raw" {
			t.Errorf("got %q, want %q", folderName, "raw")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		// In dry run, folder should NOT be created
		folderName, err := findOrCreateFolder(tmpDir, "mov", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if folderName != "mov" {
			t.Errorf("got %q, want %q", folderName, "mov")
		}

		// Verify folder was NOT created
		fullPath := filepath.Join(tmpDir, "mov")
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("folder should not exist in dry run mode")
		}
	})
}

func TestMoveFile(t *testing.T) {
	t.Run("move file successfully", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create source file
		srcFile := "test.jpg"
		srcPath := filepath.Join(tmpDir, srcFile)
		if err := os.WriteFile(srcPath, []byte("test content"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create destination folder
		destDir := "2024 - 0101 - 1000"
		destPath := filepath.Join(tmpDir, destDir)
		if err := os.Mkdir(destPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Move file
		err := moveFile(tmpDir, srcFile, destDir, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify source file no longer exists
		if _, err := os.Stat(srcPath); !os.IsNotExist(err) {
			t.Error("source file should not exist after move")
		}

		// Verify file exists at destination
		movedPath := filepath.Join(tmpDir, destDir, srcFile)
		if _, err := os.Stat(movedPath); os.IsNotExist(err) {
			t.Error("file was not moved to destination")
		}

		// Verify content is intact
		content, err := os.ReadFile(movedPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != "test content" {
			t.Errorf("file content mismatch: got %q, want %q", string(content), "test content")
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create source file
		srcFile := "test.mov"
		srcPath := filepath.Join(tmpDir, srcFile)
		if err := os.WriteFile(srcPath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		destDir := "2024 - 0101 - 1000"

		// In dry run, file should NOT be moved
		err := moveFile(tmpDir, srcFile, destDir, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Source file should still exist
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			t.Error("source file should still exist in dry run mode")
		}
	})
}

// ========================================
// Tests for New Gap-Based Algorithm (v3.0.0)
// ========================================

func TestCollectMediaFiles(t *testing.T) {
	t.Run("collect all media files", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 11, 30, 0, 0, time.Local)

		// Create media files
		mediaFiles := []string{"photo1.jpg", "photo2.NEF", "video.mov", "photo3.heic"}
		for _, file := range mediaFiles {
			createTestFile(t, tmpDir, file, baseTime)
		}

		// Create non-media files
		createTestFile(t, tmpDir, "readme.txt", baseTime)
		createTestFile(t, tmpDir, "data.csv", baseTime)

		// Create subdirectory (should be ignored)
		subDir := filepath.Join(tmpDir, "subfolder")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		cfg := &Config{BasePath: tmpDir, UseEXIF: false}
		files, err := collectMediaFilesWithMetadata(cfg, newDefaultExecutionContext())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != len(mediaFiles) {
			t.Errorf("expected %d media files, got %d", len(mediaFiles), len(files))
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &Config{BasePath: tmpDir, UseEXIF: false}
		files, err := collectMediaFilesWithMetadata(cfg, newDefaultExecutionContext())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 0 {
			t.Errorf("expected no files, got %d", len(files))
		}
	})

	t.Run("only non-media files", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 11, 30, 0, 0, time.Local)

		createTestFile(t, tmpDir, "readme.txt", baseTime)
		createTestFile(t, tmpDir, "data.json", baseTime)

		cfg := &Config{BasePath: tmpDir, UseEXIF: false}
		files, err := collectMediaFilesWithMetadata(cfg, newDefaultExecutionContext())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(files) != 0 {
			t.Errorf("expected no media files, got %d", len(files))
		}
	})
}

func TestSortFilesByDateTime(t *testing.T) {
	t.Run("sort files chronologically", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files with different timestamps (in non-chronological order)
		times := []struct {
			name string
			time time.Time
		}{
			{"file3.jpg", time.Date(2024, 1, 15, 14, 0, 0, 0, time.Local)},
			{"file1.jpg", time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)},
			{"file2.jpg", time.Date(2024, 1, 15, 12, 0, 0, 0, time.Local)},
		}

		var files []FileMetadata
		for _, f := range times {
			createTestFile(t, tmpDir, f.name, f.time)
			fi, _ := os.Stat(filepath.Join(tmpDir, f.name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)

		// Verify order
		expectedOrder := []string{"file1.jpg", "file2.jpg", "file3.jpg"}
		for i, expected := range expectedOrder {
			if files[i].FileInfo.Name() != expected {
				t.Errorf("position %d: expected %q, got %q", i, expected, files[i].FileInfo.Name())
			}
		}
	})

	t.Run("same timestamp sorts by name", func(t *testing.T) {
		tmpDir := t.TempDir()

		sameTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)
		names := []string{"charlie.jpg", "alice.jpg", "bob.jpg"}

		var files []FileMetadata
		for _, name := range names {
			createTestFile(t, tmpDir, name, sameTime)
			fi, _ := os.Stat(filepath.Join(tmpDir, name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)

		// Should be sorted alphabetically
		expectedOrder := []string{"alice.jpg", "bob.jpg", "charlie.jpg"}
		for i, expected := range expectedOrder {
			if files[i].FileInfo.Name() != expected {
				t.Errorf("position %d: expected %q, got %q", i, expected, files[i].FileInfo.Name())
			}
		}
	})
}

func TestGroupFilesByGaps(t *testing.T) {
	t.Run("single group continuous", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)

		var files []FileMetadata
		for i := 0; i < 5; i++ {
			fileTime := baseTime.Add(time.Duration(i*10) * time.Minute)
			name := fmt.Sprintf("photo%d.jpg", i)
			createTestFile(t, tmpDir, name, fileTime)
			fi, _ := os.Stat(filepath.Join(tmpDir, name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)
		groups := groupFilesByGaps(files, 30*time.Minute)

		if len(groups) != 1 {
			t.Errorf("expected 1 group, got %d", len(groups))
		}

		if len(groups[0].files) != 5 {
			t.Errorf("expected 5 files in group, got %d", len(groups[0].files))
		}

		expectedFolder := baseTime.Format(dateFormatPattern)
		if groups[0].folderName != expectedFolder {
			t.Errorf("expected folder %q, got %q", expectedFolder, groups[0].folderName)
		}
	})

	t.Run("two groups with large gap", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)

		// First group: 10:00, 10:10, 10:20
		// Gap of 2 hours
		// Second group: 12:30, 12:40

		times := []time.Duration{0, 10 * time.Minute, 20 * time.Minute, 2*time.Hour + 30*time.Minute, 2*time.Hour + 40*time.Minute}
		var files []FileMetadata
		for i, offset := range times {
			fileTime := baseTime.Add(offset)
			name := fmt.Sprintf("photo%d.jpg", i)
			createTestFile(t, tmpDir, name, fileTime)
			fi, _ := os.Stat(filepath.Join(tmpDir, name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)
		groups := groupFilesByGaps(files, 1*time.Hour)

		if len(groups) != 2 {
			t.Errorf("expected 2 groups, got %d", len(groups))
		}

		if len(groups[0].files) != 3 {
			t.Errorf("expected 3 files in first group, got %d", len(groups[0].files))
		}

		if len(groups[1].files) != 2 {
			t.Errorf("expected 2 files in second group, got %d", len(groups[1].files))
		}
	})

	t.Run("all files separated", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)

		// Each file is 2 hours apart (delta = 1h)
		var files []FileMetadata
		for i := 0; i < 3; i++ {
			fileTime := baseTime.Add(time.Duration(i*2) * time.Hour)
			name := fmt.Sprintf("photo%d.jpg", i)
			createTestFile(t, tmpDir, name, fileTime)
			fi, _ := os.Stat(filepath.Join(tmpDir, name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)
		groups := groupFilesByGaps(files, 1*time.Hour)

		if len(groups) != 3 {
			t.Errorf("expected 3 groups (each file separate), got %d", len(groups))
		}

		for i, group := range groups {
			if len(group.files) != 1 {
				t.Errorf("group %d: expected 1 file, got %d", i, len(group.files))
			}
		}
	})

	t.Run("gap exactly equal to delta", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)

		// Two files exactly 1h apart (delta = 1h)
		times := []time.Duration{0, 1 * time.Hour}
		var files []FileMetadata
		for i, offset := range times {
			fileTime := baseTime.Add(offset)
			name := fmt.Sprintf("photo%d.jpg", i)
			createTestFile(t, tmpDir, name, fileTime)
			fi, _ := os.Stat(filepath.Join(tmpDir, name))
			files = append(files, fileInfoToMetadata(fi))
		}

		sortFilesByDateTime(files)
		groups := groupFilesByGaps(files, 1*time.Hour)

		// With gap <= delta, should be same group
		if len(groups) != 1 {
			t.Errorf("expected 1 group (gap <= delta), got %d", len(groups))
		}
	})

	t.Run("single file", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.Local)

		createTestFile(t, tmpDir, "photo.jpg", baseTime)
		fi, _ := os.Stat(filepath.Join(tmpDir, "photo.jpg"))
		files := []FileMetadata{fileInfoToMetadata(fi)}

		groups := groupFilesByGaps(files, 1*time.Hour)

		if len(groups) != 1 {
			t.Errorf("expected 1 group, got %d", len(groups))
		}

		if len(groups[0].files) != 1 {
			t.Errorf("expected 1 file in group, got %d", len(groups[0].files))
		}
	})

	t.Run("empty input", func(t *testing.T) {
		groups := groupFilesByGaps([]FileMetadata{}, 1*time.Hour)

		if groups != nil {
			t.Errorf("expected nil for empty input, got %d groups", len(groups))
		}
	})
}

// ========================================
// Integration Tests (70-75% coverage)
// ========================================

func TestSplit_Integration(t *testing.T) {
	t.Run("split pictures by time delta", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create files with different dates
		// Note: Using local time because os.Chtimes converts to system time
		baseTime := time.Date(2024, 1, 15, 11, 30, 0, 0, time.Local)

		files := []struct {
			name   string
			offset time.Duration
		}{
			{"photo1.jpg", 0},                            // 11:30 (first file of group 1)
			{"photo2.jpg", 20 * time.Minute},             // 11:50 (gap: 20min <= 1h, same group)
			{"photo3.jpg", 2 * time.Hour},                // 13:30 (gap: 1h40 > 1h, new group, first file)
			{"video1.mov", 2*time.Hour + 15*time.Minute}, // 13:45 (gap: 15min <= 1h, same group as photo3)
		}

		for _, f := range files {
			fileTime := baseTime.Add(f.offset)
			createTestFile(t, tmpDir, f.name, fileTime)
		}

		// Execute Split
		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      false,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify folders were created (named after first file of each group)
		expectedFolders := []string{
			"2024 - 0115 - 1130", // photo1.jpg (11:30)
			"2024 - 0115 - 1330", // photo3.jpg (13:30)
		}

		for _, folder := range expectedFolders {
			folderPath := filepath.Join(tmpDir, folder)
			if _, err := os.Stat(folderPath); os.IsNotExist(err) {
				t.Errorf("expected folder %q not created", folder)
			}
		}

		// Verify files were moved
		movedFiles := []string{
			filepath.Join(tmpDir, "2024 - 0115 - 1130", "photo1.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1130", "photo2.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1330", "photo3.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1330", movFolderName, "video1.mov"),
		}

		for _, path := range movedFiles {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %q not found", path)
			}
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		baseTime := time.Date(2024, 1, 15, 11, 30, 0, 0, time.Local)
		createTestFile(t, tmpDir, "photo.jpg", baseTime)

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      true,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// In dry run, no folders should be created
		entries, err := os.ReadDir(tmpDir)
		if err != nil {
			t.Fatal(err)
		}

		// Should only have the original file
		if len(entries) != 1 {
			t.Errorf("expected 1 entry in dry run, got %d", len(entries))
		}
	})

	t.Run("with raw and movie separation", func(t *testing.T) {
		tmpDir := t.TempDir()

		baseTime := time.Date(2024, 2, 1, 14, 30, 0, 0, time.Local)

		createTestFile(t, tmpDir, "photo.jpg", baseTime)
		createTestFile(t, tmpDir, "photo.nef", baseTime)
		createTestFile(t, tmpDir, "video.mov", baseTime)

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      false,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify structure (folder named after first file: 14:30)
		expectedPaths := []string{
			filepath.Join(tmpDir, "2024 - 0201 - 1430", "photo.jpg"),
			filepath.Join(tmpDir, "2024 - 0201 - 1430", rawFolderName, "photo.nef"),
			filepath.Join(tmpDir, "2024 - 0201 - 1430", movFolderName, "video.mov"),
		}

		for _, path := range expectedPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %q not found", path)
			}
		}
	})

	t.Run("no move raw and movie", func(t *testing.T) {
		tmpDir := t.TempDir()

		baseTime := time.Date(2024, 3, 1, 11, 30, 0, 0, time.Local)

		createTestFile(t, tmpDir, "photo.jpg", baseTime)
		createTestFile(t, tmpDir, "photo.cr2", baseTime)
		createTestFile(t, tmpDir, "video.mp4", baseTime)

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: true,
			NoMoveRaw:   true,
			DryRun:      false,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// All files should be in the same folder without subfolders (named after first file: 11:30)
		expectedPaths := []string{
			filepath.Join(tmpDir, "2024 - 0301 - 1130", "photo.jpg"),
			filepath.Join(tmpDir, "2024 - 0301 - 1130", "photo.cr2"),
			filepath.Join(tmpDir, "2024 - 0301 - 1130", "video.mp4"),
		}

		for _, path := range expectedPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %q not found", path)
			}
		}

		// Verify no raw/mov subfolders were created
		datedFolder := filepath.Join(tmpDir, "2024 - 0301 - 1130")
		entries, err := os.ReadDir(datedFolder)
		if err != nil {
			t.Fatal(err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				t.Errorf("unexpected subfolder %q created", entry.Name())
			}
		}
	})

	t.Run("mktest.sh equivalent dataset", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create test dataset similar to mktest.sh
		createTestDataset(t, tmpDir)

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      false,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		year := time.Now().Year()

		// With gap-based detection and delta=1h:
		// 08:35 → first file of group
		// 09:35 → gap = 1h (<=delta), same group
		// 10:35 → gap = 1h (<=delta), same group
		// 11:35 → gap = 1h (<=delta), same group
		// 11:44 → gap = 9min (<=delta), same group
		// All media files in ONE group named after first file: "2025 - 0216 - 0835"

		groupFolder := fmt.Sprintf("%d - 0216 - 0835", year)

		// Verify main folder exists
		folderPath := filepath.Join(tmpDir, groupFolder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			t.Fatalf("expected folder %q not created", groupFolder)
		}

		// Verify JPEG files in main folder
		jpegFiles := []string{"PHOTO_01.JPG", "PHOTO_02.JPG", "PHOTO_03.JPG", "PHOTO_04.JPG"}
		for _, file := range jpegFiles {
			filePath := filepath.Join(folderPath, file)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Errorf("expected file %q not found in %q", file, groupFolder)
			}
		}

		// Verify RAW files are in raw subfolder
		rawPaths := []string{
			filepath.Join(tmpDir, groupFolder, rawFolderName, "PHOTO_03.CR2"),
			filepath.Join(tmpDir, groupFolder, rawFolderName, "PHOTO_04.NEF"),
		}

		for _, path := range rawPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected RAW file %q not found", path)
			}
		}

		// Verify movie is in mov subfolder
		movPath := filepath.Join(tmpDir, groupFolder, movFolderName, "PHOTO_04.MOV")
		if _, err := os.Stat(movPath); os.IsNotExist(err) {
			t.Error("expected movie file PHOTO_04.MOV not found in mov subfolder")
		}

		// Verify TEST directory is untouched
		testDir := filepath.Join(tmpDir, "TEST")
		if _, err := os.Stat(testDir); os.IsNotExist(err) {
			t.Error("TEST directory should still exist")
		}

		// Verify unknown extension file is still in root
		unknownFile := filepath.Join(tmpDir, "PHOTO_04.test")
		if _, err := os.Stat(unknownFile); os.IsNotExist(err) {
			t.Error("PHOTO_04.test should still be in root directory")
		}
	})

	t.Run("modern image formats", func(t *testing.T) {
		tmpDir := t.TempDir()

		baseTime := time.Date(2024, 5, 1, 13, 30, 0, 0, time.Local)

		// Test modern formats
		modernFiles := []string{
			"photo.heic", // Apple
			"photo.heif", // Standard HEIF
			"photo.webp", // Google WebP
			"photo.avif", // AV1 Image
			"photo.dng",  // Adobe DNG
			"photo.arw",  // Sony RAW
			"photo.orf",  // Olympus RAW
			"photo.raf",  // Fujifilm RAW
		}

		for _, file := range modernFiles {
			createTestFile(t, tmpDir, file, baseTime)
		}

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      false,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Folder named after first file (sorted alphabetically when ModTime equal): 13:30
		datedFolder := "2024 - 0501 - 1330"

		// Standard images should be in root
		standardImages := []string{"photo.heic", "photo.heif", "photo.webp", "photo.avif"}
		for _, file := range standardImages {
			path := filepath.Join(tmpDir, datedFolder, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected modern format file %q not found", file)
			}
		}

		// RAW files should be in raw subfolder
		rawFiles := []string{"photo.dng", "photo.arw", "photo.orf", "photo.raf"}
		for _, file := range rawFiles {
			path := filepath.Join(tmpDir, datedFolder, rawFolderName, file)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected modern RAW file %q not found in raw subfolder", file)
			}
		}
	})
}

// ========================================
// Tests for Config Validation
// ========================================

func TestConfig_Validate(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &Config{
			BasePath:    tmpDir,
			Delta:       1 * time.Hour,
			NoMoveMovie: false,
			NoMoveRaw:   false,
			DryRun:      false,
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("empty base path", func(t *testing.T) {
		cfg := &Config{
			BasePath: "",
			Delta:    1 * time.Hour,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for empty base path")
		}
	})

	t.Run("invalid delta", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &Config{
			BasePath: tmpDir,
			Delta:    0,
		}

		err := cfg.Validate()
		if err != ErrInvalidDelta {
			t.Errorf("expected ErrInvalidDelta, got %v", err)
		}

		cfg.Delta = -1 * time.Hour
		err = cfg.Validate()
		if err != ErrInvalidDelta {
			t.Errorf("expected ErrInvalidDelta for negative delta, got %v", err)
		}
	})

	t.Run("path does not exist", func(t *testing.T) {
		cfg := &Config{
			BasePath: "/nonexistent/path/12345",
			Delta:    1 * time.Hour,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("expected error for nonexistent path")
		}
	})

	t.Run("path is not a directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a file instead of directory
		filePath := filepath.Join(tmpDir, "file.txt")
		if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}

		cfg := &Config{
			BasePath: filePath,
			Delta:    1 * time.Hour,
		}

		err := cfg.Validate()
		if err != ErrNotDirectory {
			t.Errorf("expected ErrNotDirectory, got %v", err)
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	basePath := "/test/path"
	cfg := DefaultConfig(basePath)

	if cfg.BasePath != basePath {
		t.Errorf("expected BasePath %q, got %q", basePath, cfg.BasePath)
	}

	if cfg.Delta != 30*time.Minute {
		t.Errorf("expected Delta 30min, got %v", cfg.Delta)
	}

	if cfg.NoMoveMovie {
		t.Error("expected NoMoveMovie to be false")
	}

	if cfg.NoMoveRaw {
		t.Error("expected NoMoveRaw to be false")
	}

	if cfg.DryRun {
		t.Error("expected DryRun to be false")
	}

	if !cfg.UseEXIF {
		t.Error("expected UseEXIF to be true")
	}
}

// ========================================
// Tests for Split() edge cases
// ========================================

func TestSplit_NoMediaFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only non-media files
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: false,
		NoMoveRaw:   false,
		DryRun:      false,
		UseEXIF:     true,
		UseGPS:      false,
	}

	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() should not error with no media files, got: %v", err)
	}
}

func TestSplit_GPSMode_AllFilesWithGPS(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files without GPS coordinates
	baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)

	createTestFile(t, tmpDir, "photo1.jpg", baseTime)
	createTestFile(t, tmpDir, "photo2.jpg", baseTime.Add(10*time.Minute))
	createTestFile(t, tmpDir, "photo3.jpg", baseTime.Add(2*time.Hour))

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: false,
		NoMoveRaw:   false,
		DryRun:      false,
		UseEXIF:     false, // Use ModTime for simplicity
		UseGPS:      true,
		GPSRadius:   2000.0, // 2km
	}

	// Note: This test will process files but won't create GPS clusters
	// because we can't inject GPS coordinates without EXIF
	// When NO location clusters exist, files should be at root (no NoLocation folder)
	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() GPS mode error: %v", err)
	}

	// Verify NoLocation folder was NOT created (all files lack GPS = no segregation needed)
	noLocFolder := filepath.Join(tmpDir, GetNoLocationFolderName())
	if _, err := os.Stat(noLocFolder); !os.IsNotExist(err) {
		t.Error("NoLocation folder should NOT be created when ALL files lack GPS (no location clusters)")
	}

	// Verify time-based folders were created at root instead
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}

	foundTimeFolders := 0
	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "2024 - ") {
			foundTimeFolders++
		}
	}

	if foundTimeFolders == 0 {
		t.Error("Expected time-based folders at root when all files lack GPS")
	}
}

func TestSplit_ValidationError(t *testing.T) {
	cfg := &Config{
		BasePath: "", // Empty path should fail validation
		Delta:    30 * time.Minute,
	}

	err := Split(cfg)
	if err == nil {
		t.Error("Split() should error on invalid configuration")
	}
}

func TestSplit_InvalidBasePath(t *testing.T) {
	cfg := &Config{
		BasePath: "/nonexistent/path/that/does/not/exist",
		Delta:    30 * time.Minute,
	}

	err := Split(cfg)
	if err == nil {
		t.Error("Split() should error when base path doesn't exist")
	}
}

func TestCollectMediaFilesWithMetadata_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &Config{
		BasePath: tmpDir,
		Delta:    30 * time.Minute,
		UseEXIF:  true,
	}

	files, err := collectMediaFilesWithMetadata(cfg, newDefaultExecutionContext())
	if err != nil {
		t.Fatalf("collectMediaFilesWithMetadata(, newDefaultExecutionContext()) error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty directory, got %d", len(files))
	}
}

func TestProcessGroup_DryRunMode(t *testing.T) {
	tmpDir := t.TempDir()

	baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)
	createTestFile(t, tmpDir, "photo1.jpg", baseTime)

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: false,
		NoMoveRaw:   false,
		DryRun:      true, // Dry run mode
		UseEXIF:     false,
	}

	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() dry-run error: %v", err)
	}

	// In dry-run, files should NOT be moved
	originalPath := filepath.Join(tmpDir, "photo1.jpg")
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		t.Error("file should not be moved in dry-run mode")
	}
}

func TestSplit_MixedMediaTypes(t *testing.T) {
	tmpDir := t.TempDir()

	baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)

	// Create mix of photos, videos, and RAW files
	createTestFile(t, tmpDir, "photo.jpg", baseTime)
	createTestFile(t, tmpDir, "video.mov", baseTime.Add(5*time.Minute))
	createTestFile(t, tmpDir, "raw.nef", baseTime.Add(10*time.Minute))
	createTestFile(t, tmpDir, "image.heic", baseTime.Add(15*time.Minute))

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: false, // Movies should go to mov/
		NoMoveRaw:   false, // RAW should go to raw/
		DryRun:      false,
		UseEXIF:     false, // Use ModTime
	}

	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() error: %v", err)
	}

	// All files should be in one group (within 30min delta)
	// Find the dated folder
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var datedFolder string
	for _, entry := range entries {
		if entry.IsDir() {
			datedFolder = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	if datedFolder == "" {
		t.Fatal("no dated folder created")
	}

	// Verify structure
	photoPath := filepath.Join(datedFolder, "photo.jpg")
	heicPath := filepath.Join(datedFolder, "image.heic")
	movPath := filepath.Join(datedFolder, "mov", "video.mov")
	rawPath := filepath.Join(datedFolder, "raw", "raw.nef")

	for _, path := range []string{photoPath, heicPath, movPath, rawPath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file not found: %s", path)
		}
	}
}

func TestSplit_NoMoveMovieAndRaw(t *testing.T) {
	tmpDir := t.TempDir()

	baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)

	createTestFile(t, tmpDir, "photo.jpg", baseTime)
	createTestFile(t, tmpDir, "video.mov", baseTime.Add(5*time.Minute))
	createTestFile(t, tmpDir, "raw.nef", baseTime.Add(10*time.Minute))

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: true, // Keep movies in root
		NoMoveRaw:   true, // Keep RAW in root
		DryRun:      false,
		UseEXIF:     false,
	}

	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() error: %v", err)
	}

	// Find the dated folder
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var datedFolder string
	for _, entry := range entries {
		if entry.IsDir() {
			datedFolder = filepath.Join(tmpDir, entry.Name())
			break
		}
	}

	if datedFolder == "" {
		t.Fatal("no dated folder created")
	}

	// Verify all files are in root of dated folder (not in mov/ or raw/)
	photoPath := filepath.Join(datedFolder, "photo.jpg")
	movPath := filepath.Join(datedFolder, "video.mov") // Not in mov/
	rawPath := filepath.Join(datedFolder, "raw.nef")   // Not in raw/

	for _, path := range []string{photoPath, movPath, rawPath} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file in root of dated folder, not found: %s", path)
		}
	}

	// Verify mov/ and raw/ folders were NOT created
	movDir := filepath.Join(datedFolder, "mov")
	rawDir := filepath.Join(datedFolder, "raw")

	if _, err := os.Stat(movDir); !os.IsNotExist(err) {
		t.Error("mov/ folder should not be created when NoMoveMovie is true")
	}
	if _, err := os.Stat(rawDir); !os.IsNotExist(err) {
		t.Error("raw/ folder should not be created when NoMoveRaw is true")
	}
}

func TestSplit_MultipleGroups(t *testing.T) {
	tmpDir := t.TempDir()

	baseTime := time.Date(2024, 6, 15, 10, 0, 0, 0, time.Local)

	// Create files in 3 distinct time groups (separated by > 30 minutes)
	createTestFile(t, tmpDir, "photo1.jpg", baseTime)
	createTestFile(t, tmpDir, "photo2.jpg", baseTime.Add(10*time.Minute)) // Same group

	createTestFile(t, tmpDir, "photo3.jpg", baseTime.Add(1*time.Hour))    // New group (60min gap)
	createTestFile(t, tmpDir, "photo4.jpg", baseTime.Add(65*time.Minute)) // Same group

	createTestFile(t, tmpDir, "photo5.jpg", baseTime.Add(3*time.Hour)) // New group (2h gap)

	cfg := &Config{
		BasePath:    tmpDir,
		Delta:       30 * time.Minute,
		NoMoveMovie: false,
		NoMoveRaw:   false,
		DryRun:      false,
		UseEXIF:     false,
	}

	err := Split(cfg)
	if err != nil {
		t.Fatalf("Split() error: %v", err)
	}

	// Count dated folders
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	folderCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			folderCount++
		}
	}

	if folderCount != 3 {
		t.Errorf("expected 3 dated folders (3 groups), got %d", folderCount)
	}
}

func TestCollectMediaFilesWithMetadata_OnlyNonMedia(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only non-media files
	if err := os.WriteFile(filepath.Join(tmpDir, "doc.txt"), []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		BasePath: tmpDir,
		Delta:    30 * time.Minute,
		UseEXIF:  true,
	}

	files, err := collectMediaFilesWithMetadata(cfg, newDefaultExecutionContext())
	if err != nil {
		t.Fatalf("collectMediaFilesWithMetadata(, newDefaultExecutionContext()) error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 media files, got %d", len(files))
	}
}

// ========================================
// Tests for Orphan RAW Separation (v2.6.0)
// ========================================

func TestIsRawPaired(t *testing.T) {
	t.Run("RAW paired with JPEG same folder", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create RAW + JPEG pair
		createTestFile(t, tmpDir, "PHOTO_01.NEF", time.Now())
		createTestFile(t, tmpDir, "PHOTO_01.JPG", time.Now())

		rawPath := filepath.Join(tmpDir, "PHOTO_01.NEF")
		if !isRawPaired(rawPath, tmpDir, "") {
			t.Error("RAW should be paired with JPEG in same folder")
		}
	})

	t.Run("RAW paired with HEIC (iPhone)", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create RAW + HEIC pair (iPhone shoot RAW+HEIC)
		createTestFile(t, tmpDir, "IMG_1234.DNG", time.Now())
		createTestFile(t, tmpDir, "IMG_1234.HEIC", time.Now())

		rawPath := filepath.Join(tmpDir, "IMG_1234.DNG")
		if !isRawPaired(rawPath, tmpDir, "") {
			t.Error("RAW should be paired with HEIC")
		}
	})

	t.Run("RAW orphan (no JPEG/HEIC)", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create RAW without pair
		createTestFile(t, tmpDir, "PHOTO_02.NEF", time.Now())

		rawPath := filepath.Join(tmpDir, "PHOTO_02.NEF")
		if isRawPaired(rawPath, tmpDir, "") {
			t.Error("RAW should be orphan (no JPEG/HEIC)")
		}
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create RAW + lowercase jpeg
		createTestFile(t, tmpDir, "PHOTO_03.CR2", time.Now())
		createTestFile(t, tmpDir, "PHOTO_03.jpeg", time.Now())

		rawPath := filepath.Join(tmpDir, "PHOTO_03.CR2")
		if !isRawPaired(rawPath, tmpDir, "") {
			t.Error("RAW should be paired with .jpeg (case insensitive)")
		}
	})

	t.Run("RAW paired with JPEG already moved to destination", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create destination folder
		destFolder := filepath.Join(tmpDir, "2024 - 0701 - 1400")
		if err := os.Mkdir(destFolder, 0755); err != nil {
			t.Fatal(err)
		}

		// JPEG already moved to destination
		createTestFile(t, destFolder, "PHOTO_04.JPG", time.Now())

		// RAW still in source
		createTestFile(t, tmpDir, "PHOTO_04.NEF", time.Now())

		rawPath := filepath.Join(tmpDir, "PHOTO_04.NEF")
		if !isRawPaired(rawPath, tmpDir, destFolder) {
			t.Error("RAW should be paired with JPEG in destination folder")
		}
	})
}

func TestSplit_OrphanRawSeparation(t *testing.T) {
	t.Run("separate orphan RAW enabled (default)", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 7, 1, 14, 0, 0, 0, time.Local)

		// Create paired RAW+JPEG
		createTestFile(t, tmpDir, "PHOTO_01.JPG", baseTime)
		createTestFile(t, tmpDir, "PHOTO_01.NEF", baseTime)

		// Create orphan RAW (no JPEG)
		createTestFile(t, tmpDir, "PHOTO_02.NEF", baseTime)
		createTestFile(t, tmpDir, "PHOTO_03.CR2", baseTime)

		cfg := &Config{
			BasePath:          tmpDir,
			Delta:             1 * time.Hour,
			NoMoveMovie:       false,
			NoMoveRaw:         false,
			DryRun:            false,
			SeparateOrphanRaw: true, // Activé
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("Split() error: %v", err)
		}

		datedFolder := "2024 - 0701 - 1400"

		// Verify paired RAW is in raw/
		pairedRawPath := filepath.Join(tmpDir, datedFolder, "raw", "PHOTO_01.NEF")
		if _, err := os.Stat(pairedRawPath); os.IsNotExist(err) {
			t.Error("paired RAW should be in raw/ folder")
		}

		// Verify orphan RAW files are in orphan/
		orphan1Path := filepath.Join(tmpDir, datedFolder, "orphan", "PHOTO_02.NEF")
		orphan2Path := filepath.Join(tmpDir, datedFolder, "orphan", "PHOTO_03.CR2")

		if _, err := os.Stat(orphan1Path); os.IsNotExist(err) {
			t.Error("orphan RAW PHOTO_02.NEF should be in orphan/ folder")
		}
		if _, err := os.Stat(orphan2Path); os.IsNotExist(err) {
			t.Error("orphan RAW PHOTO_03.CR2 should be in orphan/ folder")
		}
	})

	t.Run("separate orphan RAW disabled", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 7, 1, 14, 0, 0, 0, time.Local)

		// Create paired + orphan RAW
		createTestFile(t, tmpDir, "PHOTO_01.JPG", baseTime)
		createTestFile(t, tmpDir, "PHOTO_01.NEF", baseTime)
		createTestFile(t, tmpDir, "PHOTO_02.NEF", baseTime) // orphan

		cfg := &Config{
			BasePath:          tmpDir,
			Delta:             1 * time.Hour,
			NoMoveMovie:       false,
			NoMoveRaw:         false,
			DryRun:            false,
			SeparateOrphanRaw: false, // Désactivé
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("Split() error: %v", err)
		}

		datedFolder := "2024 - 0701 - 1400"

		// Verify ALL RAW files are in raw/ (pas de séparation)
		pairedRawPath := filepath.Join(tmpDir, datedFolder, "raw", "PHOTO_01.NEF")
		orphanRawPath := filepath.Join(tmpDir, datedFolder, "raw", "PHOTO_02.NEF")

		if _, err := os.Stat(pairedRawPath); os.IsNotExist(err) {
			t.Error("paired RAW should be in raw/ folder")
		}
		if _, err := os.Stat(orphanRawPath); os.IsNotExist(err) {
			t.Error("orphan RAW should also be in raw/ when feature disabled")
		}

		// Verify orphan/ folder was NOT created
		orphanDir := filepath.Join(tmpDir, datedFolder, "orphan")
		if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
			t.Error("orphan/ folder should not exist when feature disabled")
		}
	})

	t.Run("HEIC pairing (iPhone RAW+HEIC)", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 7, 1, 14, 0, 0, 0, time.Local)

		// iPhone shoot RAW+HEIC (no JPEG)
		createTestFile(t, tmpDir, "IMG_5678.HEIC", baseTime)
		createTestFile(t, tmpDir, "IMG_5678.DNG", baseTime)

		// Orphan RAW
		createTestFile(t, tmpDir, "IMG_9999.DNG", baseTime)

		cfg := &Config{
			BasePath:          tmpDir,
			Delta:             1 * time.Hour,
			NoMoveMovie:       false,
			NoMoveRaw:         false,
			DryRun:            false,
			SeparateOrphanRaw: true,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("Split() error: %v", err)
		}

		datedFolder := "2024 - 0701 - 1400"

		// Verify HEIC-paired DNG is in raw/
		pairedPath := filepath.Join(tmpDir, datedFolder, "raw", "IMG_5678.DNG")
		if _, err := os.Stat(pairedPath); os.IsNotExist(err) {
			t.Error("HEIC-paired DNG should be in raw/ folder")
		}

		// Verify orphan DNG is in orphan/
		orphanPath := filepath.Join(tmpDir, datedFolder, "orphan", "IMG_9999.DNG")
		if _, err := os.Stat(orphanPath); os.IsNotExist(err) {
			t.Error("orphan DNG should be in orphan/ folder")
		}
	})

	t.Run("RAW processed before JPEG (alphabetical order)", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 7, 1, 14, 0, 0, 0, time.Local)

		// Create files where RAW comes before JPEG alphabetically
		// "A_" prefix ensures RAW is processed first
		createTestFile(t, tmpDir, "A_PHOTO.NEF", baseTime) // Processed FIRST
		createTestFile(t, tmpDir, "A_PHOTO.jpg", baseTime) // Processed SECOND

		cfg := &Config{
			BasePath:          tmpDir,
			Delta:             1 * time.Hour,
			NoMoveMovie:       false,
			NoMoveRaw:         false,
			DryRun:            false,
			SeparateOrphanRaw: true,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("Split() error: %v", err)
		}

		datedFolder := "2024 - 0701 - 1400"

		// Verify RAW is in raw/ (not orphan/) even though processed before JPEG
		pairedRawPath := filepath.Join(tmpDir, datedFolder, "raw", "A_PHOTO.NEF")
		if _, err := os.Stat(pairedRawPath); os.IsNotExist(err) {
			t.Error("RAW should be in raw/ folder even when processed before JPEG")
		}

		// Verify JPEG is in main folder
		jpegPath := filepath.Join(tmpDir, datedFolder, "A_PHOTO.jpg")
		if _, err := os.Stat(jpegPath); os.IsNotExist(err) {
			t.Error("JPEG should be in main folder")
		}

		// Verify orphan/ folder was NOT created
		orphanDir := filepath.Join(tmpDir, datedFolder, "orphan")
		if _, err := os.Stat(orphanDir); !os.IsNotExist(err) {
			t.Error("orphan/ folder should not exist when all RAW are paired")
		}
	})

	t.Run("dry run with orphan separation", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseTime := time.Date(2024, 7, 1, 14, 0, 0, 0, time.Local)

		createTestFile(t, tmpDir, "PHOTO_01.JPG", baseTime)
		createTestFile(t, tmpDir, "PHOTO_01.NEF", baseTime)
		createTestFile(t, tmpDir, "PHOTO_02.NEF", baseTime) // orphan

		cfg := &Config{
			BasePath:          tmpDir,
			Delta:             1 * time.Hour,
			NoMoveMovie:       false,
			NoMoveRaw:         false,
			DryRun:            true, // Dry run
			SeparateOrphanRaw: true,
		}

		err := Split(cfg)
		if err != nil {
			t.Fatalf("Split() error: %v", err)
		}

		// Verify NO files were moved
		if _, err := os.Stat(filepath.Join(tmpDir, "PHOTO_01.NEF")); os.IsNotExist(err) {
			t.Error("RAW should not be moved in dry run")
		}
		if _, err := os.Stat(filepath.Join(tmpDir, "PHOTO_02.NEF")); os.IsNotExist(err) {
			t.Error("orphan RAW should not be moved in dry run")
		}
	})
}
