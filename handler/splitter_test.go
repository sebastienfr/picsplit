package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ========================================
// Test Helpers
// ========================================

// createTestFile creates a test file with a specific modification time
func createTestFile(t *testing.T, dir, name string, modTime time.Time) string {
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

	return path
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

			got := isMovie(fi)
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

			got := isPicture(fi)
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

			got := isRaw(fi)
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

func TestFindOrCreateDatedFolder(t *testing.T) {
	t.Run("create new dated folder", func(t *testing.T) {
		tmpDir := t.TempDir()

		testTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.Local)
		testFile := createTestFile(t, tmpDir, "test.jpg", testTime)

		fi, err := os.Stat(testFile)
		if err != nil {
			t.Fatal(err)
		}

		// Initialize cache
		directories = make(map[string]string)

		delta := 1 * time.Hour
		folderName, err := findOrCreateDatedFolder(tmpDir, fi, delta, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedName := testTime.Round(delta).Format(dateFormatPattern)
		if folderName != expectedName {
			t.Errorf("got %q, want %q", folderName, expectedName)
		}

		// Verify folder was created
		fullPath := filepath.Join(tmpDir, folderName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Error("dated folder was not created")
		}
	})

	t.Run("reuse existing dated folder", func(t *testing.T) {
		tmpDir := t.TempDir()

		testTime := time.Date(2024, 2, 20, 10, 0, 0, 0, time.Local)
		delta := 1 * time.Hour
		expectedName := testTime.Round(delta).Format(dateFormatPattern)

		// Create dated folder manually
		datedPath := filepath.Join(tmpDir, expectedName)
		if err := os.Mkdir(datedPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Initialize cache
		directories = make(map[string]string)
		directories[expectedName] = expectedName

		// Create file with same date
		testFile := createTestFile(t, tmpDir, "test.jpg", testTime)
		fi, err := os.Stat(testFile)
		if err != nil {
			t.Fatal(err)
		}

		folderName, err := findOrCreateDatedFolder(tmpDir, fi, delta, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if folderName != expectedName {
			t.Errorf("got %q, want %q", folderName, expectedName)
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()

		testTime := time.Date(2024, 3, 10, 9, 0, 0, 0, time.Local)
		testFile := createTestFile(t, tmpDir, "test.jpg", testTime)

		fi, err := os.Stat(testFile)
		if err != nil {
			t.Fatal(err)
		}

		delta := 1 * time.Hour
		folderName, err := findOrCreateDatedFolder(tmpDir, fi, delta, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedName := testTime.Round(delta).Format(dateFormatPattern)
		if folderName != expectedName {
			t.Errorf("got %q, want %q", folderName, expectedName)
		}

		// In dry run, folder should NOT be created
		fullPath := filepath.Join(tmpDir, folderName)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("folder should not exist in dry run mode")
		}
	})

	t.Run("different delta values", func(t *testing.T) {
		tests := []struct {
			name     string
			time     time.Time
			delta    time.Duration
			expected string
		}{
			{
				"1 hour delta",
				time.Date(2024, 4, 5, 14, 45, 0, 0, time.Local),
				1 * time.Hour,
				time.Date(2024, 4, 5, 15, 0, 0, 0, time.Local).Format(dateFormatPattern),
			},
			{
				"30 minute delta",
				time.Date(2024, 4, 5, 14, 20, 0, 0, time.Local),
				30 * time.Minute,
				time.Date(2024, 4, 5, 14, 30, 0, 0, time.Local).Format(dateFormatPattern),
			},
			{
				"2 hour delta",
				time.Date(2024, 4, 5, 13, 30, 0, 0, time.Local),
				2 * time.Hour,
				time.Date(2024, 4, 5, 14, 0, 0, 0, time.Local).Format(dateFormatPattern),
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// Each subtest gets its own temp directory
				tmpDir := t.TempDir()

				// Initialize cache for each subtest
				directories = make(map[string]string)

				testFile := createTestFile(t, tmpDir, "test.jpg", tt.time)
				fi, err := os.Stat(testFile)
				if err != nil {
					t.Fatal(err)
				}

				folderName, err := findOrCreateDatedFolder(tmpDir, fi, tt.delta, false)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if folderName != tt.expected {
					t.Errorf("got %q, want %q", folderName, tt.expected)
				}
			})
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
// Tests for Directory Listing (75% coverage)
// ========================================

func TestListDirectories(t *testing.T) {
	t.Run("find valid dated directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create valid dated folders
		validDirs := []string{
			"2024 - 0115 - 1400",
			"2024 - 0116 - 0900",
			"2024 - 0120 - 1600",
		}

		for _, dir := range validDirs {
			dirPath := filepath.Join(tmpDir, dir)
			if err := os.Mkdir(dirPath, 0755); err != nil {
				t.Fatal(err)
			}
		}

		// Create invalid folders
		invalidDirs := []string{"random", "not-a-date", "raw", "TEST"}
		for _, dir := range invalidDirs {
			dirPath := filepath.Join(tmpDir, dir)
			if err := os.Mkdir(dirPath, 0755); err != nil {
				t.Fatal(err)
			}
		}

		// Initialize cache
		directories = make(map[string]string)

		// Execute listDirectories
		err := listDirectories(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify only valid directories are cached
		if len(directories) != len(validDirs) {
			t.Errorf("expected %d directories, got %d", len(validDirs), len(directories))
		}

		for _, dir := range validDirs {
			if _, ok := directories[dir]; !ok {
				t.Errorf("valid directory %q not found in cache", dir)
			}
		}

		// Verify invalid directories are NOT cached
		for _, dir := range invalidDirs {
			if _, ok := directories[dir]; ok {
				t.Errorf("invalid directory %q should not be in cache", dir)
			}
		}
	})

	t.Run("empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		directories = make(map[string]string)

		err := listDirectories(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(directories) != 0 {
			t.Errorf("expected empty cache, got %d entries", len(directories))
		}
	})

	t.Run("directory with only files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create some files (not directories)
		files := []string{"photo.jpg", "video.mov", "2024 - 0101 - 1000.txt"}
		for _, file := range files {
			path := filepath.Join(tmpDir, file)
			if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}

		directories = make(map[string]string)

		err := listDirectories(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(directories) != 0 {
			t.Errorf("expected no directories, got %d", len(directories))
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
			{"photo1.jpg", 0},                            // 11:30 -> rounds to 12:00
			{"photo2.jpg", 20 * time.Minute},             // 11:50 -> rounds to 12:00 (same folder)
			{"photo3.jpg", 2 * time.Hour},                // 13:30 -> rounds to 14:00 (new folder)
			{"video1.mov", 2*time.Hour + 15*time.Minute}, // 13:45 -> rounds to 14:00 (same as photo3)
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

		// Verify folders were created
		expectedFolders := []string{
			"2024 - 0115 - 1200",
			"2024 - 0115 - 1400",
		}

		for _, folder := range expectedFolders {
			folderPath := filepath.Join(tmpDir, folder)
			if _, err := os.Stat(folderPath); os.IsNotExist(err) {
				t.Errorf("expected folder %q not created", folder)
			}
		}

		// Verify files were moved
		movedFiles := []string{
			filepath.Join(tmpDir, "2024 - 0115 - 1200", "photo1.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1200", "photo2.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1400", "photo3.jpg"),
			filepath.Join(tmpDir, "2024 - 0115 - 1400", movFolderName, "video1.mov"),
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

		// Verify structure (14:30 rounds to 15:00 with 1h delta)
		expectedPaths := []string{
			filepath.Join(tmpDir, "2024 - 0201 - 1500", "photo.jpg"),
			filepath.Join(tmpDir, "2024 - 0201 - 1500", rawFolderName, "photo.nef"),
			filepath.Join(tmpDir, "2024 - 0201 - 1500", movFolderName, "video.mov"),
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

		// All files should be in the same folder without subfolders (11:30 rounds to 12:00)
		expectedPaths := []string{
			filepath.Join(tmpDir, "2024 - 0301 - 1200", "photo.jpg"),
			filepath.Join(tmpDir, "2024 - 0301 - 1200", "photo.cr2"),
			filepath.Join(tmpDir, "2024 - 0301 - 1200", "video.mp4"),
		}

		for _, path := range expectedPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected file %q not found", path)
			}
		}

		// Verify no raw/mov subfolders were created
		datedFolder := filepath.Join(tmpDir, "2024 - 0301 - 1200")
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

		// Verify expected structure
		// 08:35 → folder 0900, 09:35 → 1000 (>1h from 08:35), 10:35 → 1100 (>1h from 09:35), 11:35 → 1200 (>1h from 10:35), 11:44 → 1200 (<1h from 11:35)
		expectedFiles := map[string][]string{
			fmt.Sprintf("%d - 0216 - 0900", year): {"PHOTO_01.JPG"},
			fmt.Sprintf("%d - 0216 - 1000", year): {"PHOTO_02.JPG"},
			fmt.Sprintf("%d - 0216 - 1100", year): {"PHOTO_03.JPG"},
			fmt.Sprintf("%d - 0216 - 1200", year): {"PHOTO_04.JPG"},
		}

		for folder, files := range expectedFiles {
			folderPath := filepath.Join(tmpDir, folder)

			// Folder might not exist if no files were moved there
			if len(files) == 0 {
				continue
			}

			if _, err := os.Stat(folderPath); os.IsNotExist(err) {
				t.Errorf("expected folder %q not created", folder)
				continue
			}

			for _, file := range files {
				filePath := filepath.Join(folderPath, file)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("expected file %q not found in %q", file, folder)
				}
			}
		}

		// Verify RAW files are in raw subfolder
		rawPaths := []string{
			filepath.Join(tmpDir, fmt.Sprintf("%d - 0216 - 1100", year), rawFolderName, "PHOTO_03.CR2"),
			filepath.Join(tmpDir, fmt.Sprintf("%d - 0216 - 1200", year), rawFolderName, "PHOTO_04.NEF"),
		}

		for _, path := range rawPaths {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("expected RAW file %q not found", path)
			}
		}

		// Verify movie is in mov subfolder
		movPath := filepath.Join(tmpDir, fmt.Sprintf("%d - 0216 - 1200", year), movFolderName, "PHOTO_04.MOV")
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

		// Verify all files were processed (12:30 rounds to 13:00)
		datedFolder := "2024 - 0501 - 1400"

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

	if cfg.Delta != 1*time.Hour {
		t.Errorf("expected Delta 1h, got %v", cfg.Delta)
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
}
