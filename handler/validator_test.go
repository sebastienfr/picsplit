package handler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidationReport_Duration(t *testing.T) {
	t.Run("completed validation", func(t *testing.T) {
		start := time.Now()
		end := start.Add(5 * time.Second)
		report := &ValidationReport{
			StartTime: start,
			EndTime:   end,
		}

		duration := report.Duration()
		if duration != 5*time.Second {
			t.Errorf("expected duration 5s, got %v", duration)
		}
	})

	t.Run("in-progress validation", func(t *testing.T) {
		report := &ValidationReport{
			StartTime: time.Now().Add(-2 * time.Second),
			EndTime:   time.Time{}, // Zero value = in progress
		}

		duration := report.Duration()
		if duration < 2*time.Second || duration > 3*time.Second {
			t.Errorf("expected duration ~2s, got %v", duration)
		}
	})
}

func TestValidationReport_HasCriticalErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		report := &ValidationReport{
			Errors: []*PicsplitError{},
		}

		if report.HasCriticalErrors() {
			t.Error("expected no critical errors")
		}
	})

	t.Run("only non-critical errors", func(t *testing.T) {
		report := &ValidationReport{
			Errors: []*PicsplitError{
				{Type: ErrTypeEXIF, Op: "test"},
				{Type: ErrTypeGPS, Op: "test"},
			},
		}

		if report.HasCriticalErrors() {
			t.Error("expected no critical errors")
		}
	})

	t.Run("has critical errors", func(t *testing.T) {
		report := &ValidationReport{
			Errors: []*PicsplitError{
				{Type: ErrTypeEXIF, Op: "test"},
				{Type: ErrTypePermission, Op: "test"}, // Critical
				{Type: ErrTypeGPS, Op: "test"},
			},
		}

		if !report.HasCriticalErrors() {
			t.Error("expected critical errors")
		}
	})
}

func TestValidationReport_CriticalErrorCount(t *testing.T) {
	report := &ValidationReport{
		Errors: []*PicsplitError{
			{Type: ErrTypeEXIF, Op: "test"},       // Non-critical
			{Type: ErrTypePermission, Op: "test"}, // Critical
			{Type: ErrTypeIO, Op: "test"},         // Critical
			{Type: ErrTypeGPS, Op: "test"},        // Non-critical
			{Type: ErrTypeValidation, Op: "test"}, // Critical
		},
	}

	count := report.CriticalErrorCount()
	if count != 3 {
		t.Errorf("expected 3 critical errors, got %d", count)
	}
}

func TestValidationReport_Print(t *testing.T) {
	// This test just ensures Print() doesn't panic
	// We don't check output since it uses slog
	report := &ValidationReport{
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(2 * time.Second),
		TotalFiles: 100,
		PhotoCount: 60,
		VideoCount: 30,
		RawCount:   10,
		TotalBytes: 1024 * 1024 * 500, // 500 MB
		Errors: []*PicsplitError{
			{Type: ErrTypePermission, Op: "read", Path: "/test/file.jpg"},
		},
		Warnings: []string{"Test warning"},
	}

	// Should not panic
	report.Print()
}

func TestValidate(t *testing.T) {
	t.Run("valid media files", func(t *testing.T) {
		// Create temp directory with test files
		tempDir := t.TempDir()

		// Create test files
		testFiles := []string{
			"photo1.jpg",
			"photo2.JPG",
			"video1.mp4",
			"video2.MOV",
			"raw1.nef",
			"raw2.CR2",
		}

		for _, name := range testFiles {
			filePath := filepath.Join(tempDir, name)
			if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		cfg := &Config{
			BasePath: tempDir,
			Delta:    30 * time.Minute,
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		// Check counts
		if report.TotalFiles != 6 {
			t.Errorf("expected 6 total files, got %d", report.TotalFiles)
		}

		if report.PhotoCount != 2 {
			t.Errorf("expected 2 photos, got %d", report.PhotoCount)
		}

		if report.VideoCount != 2 {
			t.Errorf("expected 2 videos, got %d", report.VideoCount)
		}

		if report.RawCount != 2 {
			t.Errorf("expected 2 RAW files, got %d", report.RawCount)
		}

		// Check timing
		if report.StartTime.IsZero() {
			t.Error("StartTime should be set")
		}

		if report.EndTime.IsZero() {
			t.Error("EndTime should be set")
		}

		// Should have no critical errors
		if report.HasCriticalErrors() {
			t.Errorf("expected no critical errors, got %d", report.CriticalErrorCount())
		}
	})

	t.Run("unknown extensions", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create files with unknown extensions
		unknownFiles := []string{
			"document.txt",
			"data.csv",
			"readme.md",
		}

		for _, name := range unknownFiles {
			filePath := filepath.Join(tempDir, name)
			if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		cfg := &Config{
			BasePath: tempDir,
			Delta:    30 * time.Minute,
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		// Should have validation errors for unknown extensions
		if len(report.Errors) != 3 {
			t.Errorf("expected 3 validation errors, got %d", len(report.Errors))
		}

		// All errors should be validation type
		for _, err := range report.Errors {
			if err.Type != ErrTypeValidation {
				t.Errorf("expected ErrTypeValidation, got %v", err.Type)
			}
		}
	})

	t.Run("permission errors", func(t *testing.T) {
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
	})

	t.Run("empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			BasePath: tempDir,
			Delta:    30 * time.Minute,
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		if report.TotalFiles != 0 {
			t.Errorf("expected 0 files, got %d", report.TotalFiles)
		}

		if report.TotalBytes != 0 {
			t.Errorf("expected 0 bytes, got %d", report.TotalBytes)
		}
	})

	t.Run("mixed valid and invalid files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create mix of valid and invalid files
		files := map[string]string{
			"photo.jpg":    "photo content",
			"video.mp4":    "video content",
			"document.pdf": "pdf content",
			"data.xml":     "xml content",
		}

		for name, content := range files {
			filePath := filepath.Join(tempDir, name)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
		}

		cfg := &Config{
			BasePath: tempDir,
			Delta:    30 * time.Minute,
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		// Should only count media files
		if report.TotalFiles != 2 {
			t.Errorf("expected 2 media files, got %d", report.TotalFiles)
		}

		// Should have 2 validation errors for unknown extensions
		if len(report.Errors) != 2 {
			t.Errorf("expected 2 validation errors, got %d", len(report.Errors))
		}
	})

	t.Run("custom extensions", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create file with custom extension
		customFile := filepath.Join(tempDir, "custom.xyz")
		if err := os.WriteFile(customFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cfg := &Config{
			BasePath:        tempDir,
			Delta:           30 * time.Minute,
			CustomPhotoExts: []string{"xyz"},
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		// Custom extension should be recognized as photo
		if report.PhotoCount != 1 {
			t.Errorf("expected 1 photo with custom extension, got %d", report.PhotoCount)
		}

		// Should have no validation errors
		if len(report.Errors) > 0 {
			t.Errorf("expected no errors for custom extension, got %d", len(report.Errors))
		}
	})

	t.Run("ignores subdirectories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a subdirectory with files
		subDir := filepath.Join(tempDir, "subdir")
		if err := os.Mkdir(subDir, 0755); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		subFile := filepath.Join(subDir, "photo.jpg")
		if err := os.WriteFile(subFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create file in subdirectory: %v", err)
		}

		// Create file in main directory
		mainFile := filepath.Join(tempDir, "main.jpg")
		if err := os.WriteFile(mainFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create main file: %v", err)
		}

		cfg := &Config{
			BasePath: tempDir,
			Delta:    30 * time.Minute,
		}

		report, err := Validate(cfg)
		if err != nil {
			t.Fatalf("Validate() failed: %v", err)
		}

		// Should only count file in main directory
		if report.TotalFiles != 1 {
			t.Errorf("expected 1 file (ignoring subdirectories), got %d", report.TotalFiles)
		}
	})

	t.Run("invalid config - invalid custom extension", func(t *testing.T) {
		tempDir := t.TempDir()

		cfg := &Config{
			BasePath:        tempDir,
			Delta:           30 * time.Minute,
			CustomPhotoExts: []string{"toolongextension"}, // Too long
		}

		_, err := Validate(cfg)
		if err == nil {
			t.Error("expected error for invalid custom extension")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		cfg := &Config{
			BasePath: "/non/existent/path",
			Delta:    30 * time.Minute,
		}

		_, err := Validate(cfg)
		if err == nil {
			t.Error("expected error for non-existent directory")
		}
	})
}
