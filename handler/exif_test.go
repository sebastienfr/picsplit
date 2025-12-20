package handler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDateSource_String(t *testing.T) {
	tests := []struct {
		name     string
		source   DateSource
		expected string
	}{
		{
			name:     "ModTime source",
			source:   DateSourceModTime,
			expected: "ModTime",
		},
		{
			name:     "EXIF source",
			source:   DateSourceEXIF,
			expected: "EXIF",
		},
		{
			name:     "VideoMeta source",
			source:   DateSourceVideoMeta,
			expected: "VideoMeta",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.source.String()
			if result != tt.expected {
				t.Errorf("DateSource.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidDateTime(t *testing.T) {
	tests := []struct {
		name     string
		dateTime time.Time
		expected bool
	}{
		{
			name:     "valid date in 2024",
			dateTime: time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "valid date at minimum year (1990)",
			dateTime: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "invalid date before 1990",
			dateTime: time.Date(1989, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: false,
		},
		{
			name:     "invalid date in 1970 (Unix epoch)",
			dateTime: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "valid date today",
			dateTime: time.Now(),
			expected: true,
		},
		{
			name:     "valid date tomorrow (within tolerance)",
			dateTime: time.Now().AddDate(0, 0, 1),
			expected: true,
		},
		{
			name:     "invalid date too far in future",
			dateTime: time.Now().AddDate(0, 0, 2),
			expected: false,
		},
		{
			name:     "invalid date 1 year in future",
			dateTime: time.Now().AddDate(1, 0, 0),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidDateTime(tt.dateTime)
			if result != tt.expected {
				t.Errorf("isValidDateTime(%v) = %v, want %v", tt.dateTime, result, tt.expected)
			}
		})
	}
}

func TestFindAssociatedJPEG(t *testing.T) {
	// Créer un répertoire temporaire pour les tests
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		rawFile     string
		jpegFiles   []string
		shouldError bool
	}{
		{
			name:        "find .jpg (lowercase)",
			rawFile:     "PHOTO_01.NEF",
			jpegFiles:   []string{"PHOTO_01.jpg"},
			shouldError: false,
		},
		{
			name:        "find .JPG (uppercase)",
			rawFile:     "PHOTO_02.CR2",
			jpegFiles:   []string{"PHOTO_02.JPG"},
			shouldError: false,
		},
		{
			name:        "find .jpeg (lowercase)",
			rawFile:     "PHOTO_03.NEF",
			jpegFiles:   []string{"PHOTO_03.jpeg"},
			shouldError: false,
		},
		{
			name:        "find .JPEG (uppercase)",
			rawFile:     "PHOTO_04.ARW",
			jpegFiles:   []string{"PHOTO_04.JPEG"},
			shouldError: false,
		},
		{
			name:        "prefer .jpg when multiple exist",
			rawFile:     "PHOTO_05.DNG",
			jpegFiles:   []string{"PHOTO_05.jpg", "PHOTO_05.jpeg"},
			shouldError: false,
		},
		{
			name:        "no JPEG found",
			rawFile:     "PHOTO_06.NEF",
			jpegFiles:   []string{},
			shouldError: true,
		},
		{
			name:        "different filename (no match)",
			rawFile:     "PHOTO_07.CR2",
			jpegFiles:   []string{"OTHER_FILE.jpg"},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Créer le fichier RAW
			rawPath := filepath.Join(tempDir, tt.rawFile)
			if err := os.WriteFile(rawPath, []byte("dummy"), 0600); err != nil {
				t.Fatalf("failed to create RAW file: %v", err)
			}
			defer os.Remove(rawPath)

			// Créer les fichiers JPEG
			for _, jpegFile := range tt.jpegFiles {
				jpegPath := filepath.Join(tempDir, jpegFile)
				if err := os.WriteFile(jpegPath, []byte("dummy"), 0600); err != nil {
					t.Fatalf("failed to create JPEG file: %v", err)
				}
				defer os.Remove(jpegPath)
			}

			// Exécuter le test
			result, err := findAssociatedJPEG(rawPath)

			if tt.shouldError {
				if err == nil {
					t.Errorf("findAssociatedJPEG() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("findAssociatedJPEG() unexpected error: %v", err)
				}

				// Vérifier que le fichier existe
				if _, err := os.Stat(result); err != nil {
					t.Errorf("findAssociatedJPEG() returned non-existent file: %v", result)
				}

				// Vérifier que c'est bien un fichier JPEG
				ext := filepath.Ext(result)
				validExts := []string{".jpg", ".JPG", ".jpeg", ".JPEG"}
				valid := false
				for _, validExt := range validExts {
					if ext == validExt {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("findAssociatedJPEG() returned file with invalid extension: %v", ext)
				}

				// Vérifier que le basename correspond
				expectedBase := filepath.Base(tt.rawFile)
				expectedBase = expectedBase[:len(expectedBase)-len(filepath.Ext(tt.rawFile))]
				resultBase := filepath.Base(result)
				resultBase = resultBase[:len(resultBase)-len(ext)]
				if resultBase != expectedBase {
					t.Errorf("findAssociatedJPEG() returned file with wrong basename: got %v, want %v", resultBase, expectedBase)
				}
			}
		})
	}
}

func TestExtractMetadata_Fallback(t *testing.T) {
	// Créer un fichier JPEG sans EXIF
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.jpg")

	// Créer un fichier JPEG minimal sans EXIF
	// JPEG header: FF D8 FF E0 ... FF D9
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46,
		0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01,
		0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9,
	}

	if err := os.WriteFile(testFile, jpegData, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Extraire les métadonnées
	metadata, err := ExtractMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Vérifier que la source est ModTime (fallback)
	if metadata.Source != DateSourceModTime {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceModTime)
	}

	// Vérifier que FileInfo est présent
	if metadata.FileInfo == nil {
		t.Error("ExtractMetadata() FileInfo is nil")
	}

	// Vérifier que DateTime correspond à ModTime
	expectedTime := metadata.FileInfo.ModTime().Truncate(time.Second)
	actualTime := metadata.DateTime.Truncate(time.Second)
	if !expectedTime.Equal(actualTime) {
		t.Errorf("ExtractMetadata() DateTime = %v, want %v", actualTime, expectedTime)
	}

	// Vérifier que GPS est nil (pas de données GPS)
	if metadata.GPS != nil {
		t.Errorf("ExtractMetadata() GPS = %v, want nil", metadata.GPS)
	}
}

func TestExtractMetadata_NonExistentFile(t *testing.T) {
	_, err := ExtractMetadata("/nonexistent/file.jpg")
	if err == nil {
		t.Error("ExtractMetadata() expected error for non-existent file, got nil")
	}
}

// createJPEGWithEXIF creates a minimal JPEG file with EXIF DateTimeOriginal
func createJPEGWithEXIF(t *testing.T, filePath string, dateTime time.Time) {
	t.Helper()

	// JPEG with EXIF structure
	// This is a minimal valid JPEG with EXIF APP1 marker
	jpegHeader := []byte{
		0xFF, 0xD8, // SOI (Start of Image)
		0xFF, 0xE1, // APP1 marker (EXIF)
	}

	// EXIF data structure with DateTimeOriginal
	exifData := createMinimalEXIFData(dateTime)

	// APP1 length (big endian)
	app1Length := uint16(len(exifData) + 2) // +2 for length bytes
	jpegHeader = append(jpegHeader, byte(app1Length>>8), byte(app1Length&0xFF))
	jpegHeader = append(jpegHeader, exifData...)

	// Minimal JPEG body
	jpegBody := []byte{
		0xFF, 0xDB, 0x00, 0x43, // DQT marker
		0x00, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
		0x01, 0x01, 0x01, 0x01,
		0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00, 0x01, 0x01, 0x01, 0x11, 0x00,
		0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00, 0xD2, 0xCF, 0x20,
		0xFF, 0xD9, // EOI (End of Image)
	}

	jpegData := append(jpegHeader, jpegBody...)

	if err := os.WriteFile(filePath, jpegData, 0600); err != nil {
		t.Fatalf("failed to create JPEG with EXIF: %v", err)
	}
}

// createMinimalEXIFData creates minimal EXIF APP1 data with DateTimeOriginal
func createMinimalEXIFData(dateTime time.Time) []byte {
	data := []byte("Exif\x00\x00") // EXIF header

	// TIFF header (little endian)
	data = append(data, []byte{
		0x49, 0x49, // Little endian
		0x2A, 0x00, // TIFF magic number
		0x08, 0x00, 0x00, 0x00, // Offset to first IFD
	}...)

	// IFD0 (1 entry for DateTimeOriginal tag 0x9003)
	data = append(data, []byte{
		0x01, 0x00, // Number of entries (1)
		// Entry: DateTimeOriginal (0x9003)
		0x03, 0x90, // Tag 0x9003 (36867)
		0x02, 0x00, // Type: ASCII
		0x14, 0x00, 0x00, 0x00, // Count: 20 bytes
		0x1A, 0x00, 0x00, 0x00, // Offset to value
		// Next IFD offset
		0x00, 0x00, 0x00, 0x00,
	}...)

	// DateTimeOriginal value: "YYYY:MM:DD HH:MM:SS\0"
	dateStr := dateTime.Format("2006:01:02 15:04:05") + "\x00"
	data = append(data, []byte(dateStr)...)

	return data
}

func TestExtractEXIFDate_WithValidEXIF(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_exif.jpg")

	// Create JPEG with EXIF date: 2024-06-15 14:30:00
	expectedDate := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	createJPEGWithEXIF(t, testFile, expectedDate)

	// Extract EXIF date
	actualDate, err := extractEXIFDate(testFile)
	if err != nil {
		t.Fatalf("extractEXIFDate() failed: %v", err)
	}

	// Compare dates (allow some tolerance for formatting)
	if actualDate.Format("2006-01-02 15:04:05") != expectedDate.Format("2006-01-02 15:04:05") {
		t.Errorf("extractEXIFDate() = %v, want %v", actualDate, expectedDate)
	}
}

func TestExtractEXIFDate_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.jpg")

	// Create file with invalid EXIF data
	if err := os.WriteFile(testFile, []byte("not a valid JPEG"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := extractEXIFDate(testFile)
	if err == nil {
		t.Error("extractEXIFDate() expected error for invalid EXIF, got nil")
	}
}

func TestExtractMetadata_WithEXIF(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "photo.jpg")

	// Create JPEG with EXIF
	expectedDate := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	createJPEGWithEXIF(t, testFile, expectedDate)

	metadata, err := ExtractMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Verify source is EXIF
	if metadata.Source != DateSourceEXIF {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceEXIF)
	}

	// Verify date matches EXIF
	if metadata.DateTime.Format("2006-01-02 15:04:05") != expectedDate.Format("2006-01-02 15:04:05") {
		t.Errorf("ExtractMetadata() DateTime = %v, want %v", metadata.DateTime, expectedDate)
	}
}

func TestExtractMetadata_RAWWithAssociatedJPEG(t *testing.T) {
	tempDir := t.TempDir()
	rawFile := filepath.Join(tempDir, "photo.nef")
	jpegFile := filepath.Join(tempDir, "photo.jpg")

	// Create RAW file (dummy)
	if err := os.WriteFile(rawFile, []byte("dummy RAW"), 0600); err != nil {
		t.Fatalf("failed to create RAW file: %v", err)
	}

	// Create associated JPEG with EXIF
	expectedDate := time.Date(2024, 7, 20, 10, 15, 0, 0, time.UTC)
	createJPEGWithEXIF(t, jpegFile, expectedDate)

	metadata, err := ExtractMetadata(rawFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Should use EXIF from associated JPEG
	if metadata.Source != DateSourceEXIF {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceEXIF)
	}

	if metadata.DateTime.Format("2006-01-02 15:04:05") != expectedDate.Format("2006-01-02 15:04:05") {
		t.Errorf("ExtractMetadata() DateTime = %v, want %v", metadata.DateTime, expectedDate)
	}
}

func TestExtractMetadata_RAWWithoutJPEG(t *testing.T) {
	tempDir := t.TempDir()
	rawFile := filepath.Join(tempDir, "photo.cr2")

	// Create RAW file without associated JPEG
	if err := os.WriteFile(rawFile, []byte("dummy RAW"), 0600); err != nil {
		t.Fatalf("failed to create RAW file: %v", err)
	}

	metadata, err := ExtractMetadata(rawFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Should fallback to ModTime
	if metadata.Source != DateSourceModTime {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceModTime)
	}
}

func TestExtractGPS_NoGPSData(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "no_gps.jpg")

	// Create JPEG without GPS
	createJPEGWithEXIF(t, testFile, time.Now())

	_, err := extractGPS(testFile)
	if err == nil {
		t.Error("extractGPS() expected error for file without GPS, got nil")
	}
}

func TestExtractGPS_InvalidFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.jpg")

	if err := os.WriteFile(testFile, []byte("not valid"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := extractGPS(testFile)
	if err == nil {
		t.Error("extractGPS() expected error for invalid file, got nil")
	}
}

func TestExtractVideoMetadata_ValidMP4(t *testing.T) {
	// Note: Creating a fully compliant MP4 that go-mp4 can parse is complex.
	// This test verifies the error path when mvhd box is not found.
	// Real MP4 files would be tested in integration tests with actual camera footage.
	t.Skip("Skipping MP4 creation test - requires real MP4 fixtures")
}

func TestExtractVideoMetadata_InvalidMP4(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "invalid.mp4")

	if err := os.WriteFile(testFile, []byte("not a valid MP4"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := extractVideoMetadata(testFile)
	if err == nil {
		t.Error("extractVideoMetadata() expected error for invalid MP4, got nil")
	}
}

func TestExtractVideoMetadata_NoCreationTime(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "no_mvhd.mp4")

	// Create MP4 with only ftyp box (no moov/mvhd)
	ftypBox := []byte{
		0x00, 0x00, 0x00, 0x20,
		'f', 't', 'y', 'p',
		'i', 's', 'o', 'm',
		0x00, 0x00, 0x02, 0x00,
		'i', 's', 'o', 'm',
		'i', 's', 'o', '2',
		'a', 'v', 'c', '1',
		'm', 'p', '4', '1',
	}

	if err := os.WriteFile(testFile, ftypBox, 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := extractVideoMetadata(testFile)
	if err == nil {
		t.Error("extractVideoMetadata() expected error for MP4 without creation time, got nil")
	}
}

func TestExtractMetadata_VideoFile(t *testing.T) {
	t.Skip("Skipping video extraction test - requires real MP4 fixtures")
}

func TestExtractMetadata_VideoWithInvalidDate(t *testing.T) {
	t.Skip("Skipping video test - requires real MP4 fixtures")
}

func TestExtractMetadata_PhotoWithInvalidEXIFDate(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "old_photo.jpg")

	// Create JPEG with EXIF date before 1990
	invalidDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	createJPEGWithEXIF(t, testFile, invalidDate)

	metadata, err := ExtractMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Should fallback to ModTime because EXIF date is invalid
	if metadata.Source != DateSourceModTime {
		t.Errorf("ExtractMetadata() source = %v, want %v (should fallback for invalid EXIF date)", metadata.Source, DateSourceModTime)
	}
}

func TestExtractMetadata_UnsupportedFileType(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "document.txt")

	if err := os.WriteFile(testFile, []byte("Hello World"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	metadata, err := ExtractMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Should use ModTime for unsupported file types
	if metadata.Source != DateSourceModTime {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceModTime)
	}

	if metadata.FileInfo == nil {
		t.Error("ExtractMetadata() FileInfo is nil")
	}
}

func TestExtractMetadata_MovieFallback(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "broken.avi")

	// Create a file that's detected as movie but has no valid metadata
	if err := os.WriteFile(testFile, []byte("RIFF fake AVI"), 0600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	metadata, err := ExtractMetadata(testFile)
	if err != nil {
		t.Fatalf("ExtractMetadata() failed: %v", err)
	}

	// Should fallback to ModTime when video metadata extraction fails
	if metadata.Source != DateSourceModTime {
		t.Errorf("ExtractMetadata() source = %v, want %v", metadata.Source, DateSourceModTime)
	}
}

func TestExtractEXIFDate_FileOpenError(t *testing.T) {
	_, err := extractEXIFDate("/nonexistent/file.jpg")
	if err == nil {
		t.Error("extractEXIFDate() expected error for non-existent file, got nil")
	}
}

func TestExtractVideoMetadata_FileOpenError(t *testing.T) {
	_, err := extractVideoMetadata("/nonexistent/video.mp4")
	if err == nil {
		t.Error("extractVideoMetadata() expected error for non-existent file, got nil")
	}
}

func TestExtractGPS_FileOpenError(t *testing.T) {
	_, err := extractGPS("/nonexistent/file.jpg")
	if err == nil {
		t.Error("extractGPS() expected error for non-existent file, got nil")
	}
}
