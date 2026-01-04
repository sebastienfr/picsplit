package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// DuplicateDetector detects duplicate files via SHA256 hash
type DuplicateDetector struct {
	hashes     map[string]string  // hash → first file path
	duplicates map[string]string  // duplicate path → original path
	sizeGroups map[int64][]string // size → file paths (pre-filtering)
	enabled    bool
}

// NewDuplicateDetector creates a new duplicate detector
func NewDuplicateDetector(enabled bool) *DuplicateDetector {
	return &DuplicateDetector{
		hashes:     make(map[string]string),
		duplicates: make(map[string]string),
		sizeGroups: make(map[int64][]string),
		enabled:    enabled,
	}
}

// AddFile adds a file to size pre-filtering
// This step is optional but improves performance
func (d *DuplicateDetector) AddFile(filePath string, size int64) {
	if !d.enabled {
		return
	}
	d.sizeGroups[size] = append(d.sizeGroups[size], filePath)
}

// Check verifies if the file is a duplicate
// Returns (isDuplicate, originalPath, error)
func (d *DuplicateDetector) Check(filePath string, size int64) (bool, string, error) {
	if !d.enabled {
		return false, "", nil
	}

	// Optimization: if only one file of this size, no duplicate possible
	if len(d.sizeGroups[size]) == 1 {
		slog.Debug("unique file size, skipping hash", "file", filePath, "size", size)
		return false, "", nil
	}

	// Calculate hash
	hash, err := sha256File(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to hash file: %w", err)
	}

	// Check if hash already seen
	if original, found := d.hashes[hash]; found {
		// Duplicate detected!
		d.duplicates[filePath] = original
		slog.Debug("duplicate detected", "file", filePath, "original", original, "hash", hash[:16])
		return true, original, nil
	}

	// First file with this hash
	d.hashes[hash] = filePath
	return false, "", nil
}

// GetDuplicates returns the map of detected duplicates
// map[duplicate_path]original_path
func (d *DuplicateDetector) GetDuplicates() map[string]string {
	return d.duplicates
}

// GetStats returns detector statistics
func (d *DuplicateDetector) GetStats() (totalFiles int, uniqueSizes int, potentialDuplicates int, confirmedDuplicates int) {
	totalFiles = 0
	uniqueSizes = 0
	potentialDuplicates = 0

	for _, files := range d.sizeGroups {
		totalFiles += len(files)
		if len(files) == 1 {
			uniqueSizes++
		} else {
			potentialDuplicates += len(files)
		}
	}

	confirmedDuplicates = len(d.duplicates)
	return
}

// sha256File calculates the SHA256 hash of a file
func sha256File(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
