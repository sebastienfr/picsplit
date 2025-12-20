package handler

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	// Extension validation constraints
	maxExtensionLength = 8 // Maximum characters in extension (without leading dot)
)

var (
	// Default extension maps (lowercase for case-insensitive matching)
	defaultMovieExtensions = map[string]bool{
		".mov": true,
		".avi": true,
		".mp4": true,
	}

	defaultRawExtensions = map[string]bool{
		".nef": true,
		".nrw": true,
		".crw": true,
		".cr2": true,
		".rw2": true,
		".dng": true,
		".arw": true,
		".orf": true,
		".raf": true,
	}

	defaultPhotoExtensions = map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".heic": true,
		".heif": true,
		".webp": true,
		".avif": true,
	}
)

// ValidateExtension validates that an extension is reasonable
// - Max 8 characters (without leading dot)
// - Only alphanumeric characters
// - No spaces or special characters
func ValidateExtension(ext string) error {
	// Remove leading dot if present
	ext = strings.TrimPrefix(ext, ".")

	if ext == "" {
		return fmt.Errorf("extension cannot be empty")
	}

	if len(ext) > maxExtensionLength {
		return fmt.Errorf("only alphanumeric characters allowed (max %d chars)", maxExtensionLength)
	}

	// Check only alphanumeric
	for _, r := range ext {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("only alphanumeric characters allowed (max %d chars)", maxExtensionLength)
		}
	}

	return nil
}

// buildExtensionMap creates a combined extension map from defaults + custom
// All extensions are normalized to lowercase for case-insensitive matching
func buildExtensionMap(defaults map[string]bool, custom []string) (map[string]bool, error) {
	result := make(map[string]bool, len(defaults)+len(custom))

	// Copy defaults (all lowercase already)
	for ext := range defaults {
		result[ext] = true
	}

	// Add custom extensions
	for _, ext := range custom {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}

		// Validate before adding
		if err := ValidateExtension(ext); err != nil {
			return nil, err
		}

		// Normalize: lowercase + ensure leading dot
		ext = strings.ToLower(ext)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}

		// Add to map (duplicates silently ignored)
		result[ext] = true
	}

	return result, nil
}

// executionContext holds runtime configuration including extension maps
// Built once per execution with custom extensions merged into defaults
type executionContext struct {
	movieExtensions map[string]bool
	rawExtensions   map[string]bool
	photoExtensions map[string]bool
}

// newExecutionContext creates a context with default + custom extensions
// Returns error if custom extensions are invalid
func newExecutionContext(cfg *Config) (*executionContext, error) {
	movieExts, err := buildExtensionMap(defaultMovieExtensions, cfg.CustomVideoExts)
	if err != nil {
		return nil, fmt.Errorf("invalid video extensions: %w", err)
	}

	rawExts, err := buildExtensionMap(defaultRawExtensions, cfg.CustomRawExts)
	if err != nil {
		return nil, fmt.Errorf("invalid RAW extensions: %w", err)
	}

	photoExts, err := buildExtensionMap(defaultPhotoExtensions, cfg.CustomPhotoExts)
	if err != nil {
		return nil, fmt.Errorf("invalid photo extensions: %w", err)
	}

	return &executionContext{
		movieExtensions: movieExts,
		rawExtensions:   rawExts,
		photoExtensions: photoExts,
	}, nil
}

// newDefaultExecutionContext creates a context with only default extensions (no custom)
// Useful for testing and backward compatibility
func newDefaultExecutionContext() *executionContext {
	return &executionContext{
		movieExtensions: defaultMovieExtensions,
		rawExtensions:   defaultRawExtensions,
		photoExtensions: defaultPhotoExtensions,
	}
}

// isMovie checks if filename is a video file (case-insensitive)
func (ctx *executionContext) isMovie(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ctx.movieExtensions[ext]
}

// isPhoto checks if filename is a photo or RAW file (case-insensitive)
func (ctx *executionContext) isPhoto(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ctx.photoExtensions[ext] || ctx.rawExtensions[ext]
}

// isRaw checks if filename is a RAW file (case-insensitive)
func (ctx *executionContext) isRaw(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ctx.rawExtensions[ext]
}

// isMediaFile checks if filename is any supported media type (case-insensitive)
func (ctx *executionContext) isMediaFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ctx.movieExtensions[ext] || ctx.rawExtensions[ext] || ctx.photoExtensions[ext]
}
