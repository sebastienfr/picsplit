package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
)

const partialHashSize int64 = 65536

// DuplicateDetector detects duplicate files via SHA256 hash
type DuplicateDetector struct {
	hashes        map[string]string   // full hash → first file path
	partialHashes map[string][]string // partial hash → list of file paths
	fullHashed    map[string]bool     // file path → true if full hash computed
	duplicates    map[string]string   // duplicate path → original path
	sizeGroups    map[int64][]string  // size → file paths (pre-filtering)
	enabled       bool
}

// NewDuplicateDetector creates a new duplicate detector
func NewDuplicateDetector(enabled bool) *DuplicateDetector {
	return &DuplicateDetector{
		hashes:        make(map[string]string),
		partialHashes: make(map[string][]string),
		fullHashed:    make(map[string]bool),
		duplicates:    make(map[string]string),
		sizeGroups:    make(map[int64][]string),
		enabled:       enabled,
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

	// Stage 1: partial hash to quickly eliminate non-duplicates
	partialHash, err := sha256FilePartial(filePath, partialHashSize)
	if err != nil {
		return false, "", fmt.Errorf("failed to hash file: %w", err)
	}

	candidates, partialFound := d.partialHashes[partialHash]
	if !partialFound {
		// First file with this partial hash — pre-compute full hash now
		// because the file may be moved/renamed before we need to compare it later
		d.partialHashes[partialHash] = []string{filePath}
		fullHash, fErr := sha256File(filePath)
		if fErr == nil {
			d.hashes[fullHash] = filePath
			d.fullHashed[filePath] = true
		}
		return false, "", nil
	}

	// Stage 2: partial hash collision, compute full hashes to confirm
	fullHash, err := sha256File(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to hash file: %w", err)
	}

	// Ensure full hashes are computed for previous candidates with same partial hash
	// (should already be done at first encounter, but handle edge cases)
	for _, candidate := range candidates {
		if d.fullHashed[candidate] {
			continue
		}
		candidateHash, cErr := sha256File(candidate)
		if cErr != nil {
			slog.Debug("failed to compute full hash for candidate", "file", candidate, "error", cErr)
			continue
		}
		d.hashes[candidateHash] = candidate
		d.fullHashed[candidate] = true
	}

	d.partialHashes[partialHash] = append(candidates, filePath)
	d.fullHashed[filePath] = true

	// Check if full hash already seen
	if original, found := d.hashes[fullHash]; found {
		d.duplicates[filePath] = original
		slog.Debug("duplicate detected", "file", filePath, "original", original, "hash", fullHash[:16])
		return true, original, nil
	}

	// First file with this full hash
	d.hashes[fullHash] = filePath
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

// sha256FilePartial calculates the SHA256 hash of the first maxBytes of a file
func sha256FilePartial(filePath string, maxBytes int64) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.CopyN(h, f, maxBytes); err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
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
