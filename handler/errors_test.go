package handler

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPicsplitError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		expected string
	}{
		{
			name: "with underlying error",
			err: &PicsplitError{
				Type: ErrTypePermission,
				Op:   "read_file",
				Path: "/photos/IMG_001.jpg",
				Err:  errors.New("permission denied"),
			},
			expected: "[Permission] read_file: /photos/IMG_001.jpg - permission denied",
		},
		{
			name: "without underlying error",
			err: &PicsplitError{
				Type: ErrTypeValidation,
				Op:   "validate_extension",
				Path: "/photos/IMG_001.orf",
			},
			expected: "[Validation] validate_extension: /photos/IMG_001.orf",
		},
		{
			name: "IO error with disk full",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "move_file",
				Path: "/photos/IMG_002.jpg",
				Err:  errors.New("disk full"),
			},
			expected: "[IO] move_file: /photos/IMG_002.jpg - disk full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPicsplitError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("original error")
	err := &PicsplitError{
		Type: ErrTypeIO,
		Op:   "test_op",
		Path: "/test/path",
		Err:  underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test with no underlying error
	errNoUnderlying := &PicsplitError{
		Type: ErrTypeValidation,
		Op:   "test_op",
		Path: "/test/path",
	}

	if errNoUnderlying.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no underlying error exists")
	}
}

func TestPicsplitError_Suggestion_Permission(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		contains string // Use contains instead of exact match for cross-platform compatibility
	}{
		{
			name: "read file permission",
			err: &PicsplitError{
				Type: ErrTypePermission,
				Op:   "read_file",
				Path: "/photos/IMG_001.jpg",
			},
			contains: "chmod +r",
		},
		{
			name: "create folder permission",
			err: &PicsplitError{
				Type: ErrTypePermission,
				Op:   "create_folder",
				Path: "/photos/2024-01-01",
			},
			contains: "chmod +w",
		},
		{
			name: "generic permission",
			err: &PicsplitError{
				Type: ErrTypePermission,
				Op:   "other_op",
				Path: "/photos/test",
			},
			contains: "Check permissions on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Suggestion()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Suggestion() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestPicsplitError_Suggestion_Validation(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		expected string
	}{
		{
			name: "unknown extension",
			err: &PicsplitError{
				Type: ErrTypeValidation,
				Op:   "validate_extension",
				Path: "/photos/IMG_001.orf",
				Details: map[string]string{
					"extension": "orf",
				},
			},
			expected: "picsplit <path> --add-extension orf:raw",
		},
		{
			name: "generic validation error",
			err: &PicsplitError{
				Type: ErrTypeValidation,
				Op:   "validate_config",
				Path: "/config.json",
			},
			expected: "Check file format and configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Suggestion()
			if result != tt.expected {
				t.Errorf("Suggestion() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPicsplitError_Suggestion_IO(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		contains string
	}{
		{
			name: "disk full",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "move_file",
				Path: "/photos/IMG_001.jpg",
				Err:  errors.New("disk full"),
			},
			contains: "Free up disk space",
		},
		{
			name: "no space left",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "copy_file",
				Path: "/photos/IMG_002.jpg",
				Err:  errors.New("no space left on device"),
			},
			contains: "Free up disk space",
		},
		{
			name: "file not found",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "move_file",
				Path: "/photos/missing.jpg",
				Err:  errors.New("no such file or directory"),
			},
			contains: "Check that source path exists",
		},
		{
			name: "generic IO error",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "write_file",
				Path: "/photos/output.jpg",
				Err:  errors.New("unknown I/O error"),
			},
			contains: "Check filesystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Suggestion()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Suggestion() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestPicsplitError_Suggestion_EXIF(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		contains string
	}{
		{
			name: "no associated JPEG",
			err: &PicsplitError{
				Type: ErrTypeEXIF,
				Op:   "extract_metadata",
				Path: "/photos/IMG_001.orf",
				Err:  errors.New("No associated JPEG found"),
			},
			contains: "modification time as fallback",
		},
		{
			name: "corrupted EXIF",
			err: &PicsplitError{
				Type: ErrTypeEXIF,
				Op:   "extract_exif",
				Path: "/photos/IMG_002.jpg",
				Err:  errors.New("corrupted EXIF data"),
			},
			contains: "modification time as fallback",
		},
		{
			name: "generic EXIF error",
			err: &PicsplitError{
				Type: ErrTypeEXIF,
				Op:   "decode_exif",
				Path: "/photos/IMG_003.jpg",
				Err:  errors.New("failed to decode"),
			},
			contains: "modification time as fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Suggestion()
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Suggestion() = %q, want to contain %q", result, tt.contains)
			}
		})
	}
}

func TestPicsplitError_IsCritical(t *testing.T) {
	tests := []struct {
		name     string
		errType  ErrorType
		critical bool
	}{
		{"Permission is critical", ErrTypePermission, true},
		{"IO is critical", ErrTypeIO, true},
		{"Validation is critical", ErrTypeValidation, true},
		{"EXIF is not critical", ErrTypeEXIF, false},
		{"VideoMeta is not critical", ErrTypeVideoMeta, false},
		{"GPS is not critical", ErrTypeGPS, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &PicsplitError{
				Type: tt.errType,
				Op:   "test_op",
				Path: "/test/path",
			}
			result := err.IsCritical()
			if result != tt.critical {
				t.Errorf("IsCritical() = %v, want %v for type %s", result, tt.critical, tt.errType)
			}
		})
	}
}

func TestPicsplitError_Details(t *testing.T) {
	err := &PicsplitError{
		Type: ErrTypeValidation,
		Op:   "validate_extension",
		Path: "/photos/IMG_001.orf",
		Details: map[string]string{
			"extension":  "orf",
			"file_size":  "15MB",
			"expected":   "raw format",
			"suggestion": "add to custom extensions",
		},
	}

	// Verify details are accessible
	if ext := err.Details["extension"]; ext != "orf" {
		t.Errorf("Details[extension] = %q, want %q", ext, "orf")
	}
	if size := err.Details["file_size"]; size != "15MB" {
		t.Errorf("Details[file_size] = %q, want %q", size, "15MB")
	}
}

func TestPicsplitError_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name     string
		err      *PicsplitError
		wantErr  string
		wantSugg string
		wantCrit bool
	}{
		{
			name: "RAW file without JPEG (non-critical)",
			err: &PicsplitError{
				Type: ErrTypeEXIF,
				Op:   "extract_metadata",
				Path: "/photos/DSC_001.nef",
				Err:  fmt.Errorf("No associated JPEG/HEIC found"),
			},
			wantErr:  "[EXIF] extract_metadata: /photos/DSC_001.nef - No associated JPEG/HEIC found",
			wantSugg: "modification time as fallback",
			wantCrit: false,
		},
		{
			name: "Unknown extension (critical)",
			err: &PicsplitError{
				Type: ErrTypeValidation,
				Op:   "validate_extension",
				Path: "/photos/IMG_001.orf",
				Details: map[string]string{
					"extension": "orf",
				},
			},
			wantErr:  "[Validation] validate_extension: /photos/IMG_001.orf",
			wantSugg: "picsplit <path> --add-extension orf:raw",
			wantCrit: true,
		},
		{
			name: "Disk full during move (critical)",
			err: &PicsplitError{
				Type: ErrTypeIO,
				Op:   "move_file",
				Path: "/photos/IMG_002.jpg",
				Err:  errors.New("write failed: disk full"),
				Details: map[string]string{
					"dest":      "/output/2024-01-01/IMG_002.jpg",
					"file_size": "24MB",
				},
			},
			wantErr:  "[IO] move_file: /photos/IMG_002.jpg - write failed: disk full",
			wantSugg: "Free up disk space",
			wantCrit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Error()
			if got := tt.err.Error(); got != tt.wantErr {
				t.Errorf("Error() = %q, want %q", got, tt.wantErr)
			}

			// Test Suggestion()
			if got := tt.err.Suggestion(); !strings.Contains(got, tt.wantSugg) {
				t.Errorf("Suggestion() = %q, want to contain %q", got, tt.wantSugg)
			}

			// Test IsCritical()
			if got := tt.err.IsCritical(); got != tt.wantCrit {
				t.Errorf("IsCritical() = %v, want %v", got, tt.wantCrit)
			}
		})
	}
}
