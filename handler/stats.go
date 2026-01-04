package handler

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
)

// ProcessingStats holds statistics collected during file processing
type ProcessingStats struct {
	// Timing
	StartTime time.Time
	EndTime   time.Time

	// Counts
	TotalFiles     int
	ProcessedFiles int
	PhotoCount     int
	VideoCount     int
	RawCount       int
	GroupsCreated  int

	// RAW organization
	PairedRaw int
	OrphanRaw int

	// Disk
	TotalBytes int64

	// Cleanup
	EmptyDirsRemoved []string          // List of empty directories removed
	EmptyDirsFailed  map[string]string // Map of failed directory removals (path -> error message)

	// Duplicates (v2.8.0+)
	DuplicatesDetected map[string]string // Map of detected duplicates (duplicate path -> original path)
	DuplicatesSkipped  int               // Number of duplicates skipped

	// Issues
	ModTimeFallbackCount int // Files that fell back to ModTime
	Errors               []*PicsplitError
}

// AddError adds an error to the statistics
// If the error is a PicsplitError, it's added to the Errors slice
// If it's a generic error, it's wrapped as a PicsplitError with ErrTypeIO
func (s *ProcessingStats) AddError(err error) {
	if err == nil {
		return
	}

	var perr *PicsplitError
	if pErr, ok := err.(*PicsplitError); ok {
		perr = pErr
	} else {
		// Wrap generic errors as PicsplitError
		perr = &PicsplitError{
			Type: ErrTypeIO,
			Op:   "unknown",
			Err:  err,
		}
	}

	s.Errors = append(s.Errors, perr)
}

// HasCriticalErrors returns true if any critical errors were encountered
func (s *ProcessingStats) HasCriticalErrors() bool {
	for _, err := range s.Errors {
		if err.IsCritical() {
			return true
		}
	}
	return false
}

// Duration returns the total processing duration
func (s *ProcessingStats) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}
	return s.EndTime.Sub(s.StartTime)
}

// Throughput returns the processing speed in MB/s
func (s *ProcessingStats) Throughput() float64 {
	seconds := s.Duration().Seconds()
	if seconds == 0 {
		return 0
	}
	megabytes := float64(s.TotalBytes) / 1024 / 1024
	return megabytes / seconds
}

// SuccessRate returns the percentage of successfully processed files
func (s *ProcessingStats) SuccessRate() float64 {
	if s.TotalFiles == 0 {
		return 0
	}
	return float64(s.ProcessedFiles) / float64(s.TotalFiles) * 100
}

// FormatBytes converts bytes to human-readable format (GB, MB, KB)
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// PrintSummary displays the processing summary
func (s *ProcessingStats) PrintSummary(dryRun bool) {
	fmt.Println()
	slog.Info("=== Processing Summary ===")

	// Duration
	duration := s.Duration()
	slog.Info("processing completed",
		"duration", fmt.Sprintf("%dm %ds", int(duration.Minutes()), int(duration.Seconds())%60))

	// Files processed
	slog.Info("files processed",
		"processed", s.ProcessedFiles,
		"total", s.TotalFiles,
		"success_rate", fmt.Sprintf("%.1f%%", s.SuccessRate()))

	// Breakdown by type
	if s.PhotoCount > 0 || s.VideoCount > 0 || s.RawCount > 0 {
		photoPercent := 0.0
		videoPercent := 0.0
		rawPercent := 0.0
		if s.TotalFiles > 0 {
			photoPercent = float64(s.PhotoCount) / float64(s.TotalFiles) * 100
			videoPercent = float64(s.VideoCount) / float64(s.TotalFiles) * 100
			rawPercent = float64(s.RawCount) / float64(s.TotalFiles) * 100
		}

		slog.Info("file breakdown",
			"photos", s.PhotoCount,
			"photos_pct", fmt.Sprintf("%.1f%%", photoPercent),
			"videos", s.VideoCount,
			"videos_pct", fmt.Sprintf("%.1f%%", videoPercent),
			"raw", s.RawCount,
			"raw_pct", fmt.Sprintf("%.1f%%", rawPercent))
	}

	// Groups created
	slog.Info("groups created", "count", s.GroupsCreated)

	// RAW organization
	if s.PairedRaw > 0 || s.OrphanRaw > 0 {
		slog.Info("RAW organization",
			"paired", s.PairedRaw,
			"orphan", s.OrphanRaw)
	}

	// Disk usage
	if s.TotalBytes > 0 {
		slog.Info("disk usage",
			"total", FormatBytes(s.TotalBytes),
			"throughput", fmt.Sprintf("%.1f MB/s", s.Throughput()))
	}

	// Separate critical errors from warnings
	var criticalErrors []*PicsplitError
	var warnings []*PicsplitError

	for _, err := range s.Errors {
		if err.IsCritical() {
			criticalErrors = append(criticalErrors, err)
		} else {
			warnings = append(warnings, err)
		}
	}

	// Display critical errors
	if len(criticalErrors) > 0 {
		fmt.Println()
		slog.Error("critical errors encountered", "count", len(criticalErrors))
		for _, err := range criticalErrors {
			slog.Error(err.Error(),
				"type", string(err.Type),
				"operation", err.Op,
				"path", err.Path,
				"suggestion", err.Suggestion())
		}
	}

	// Display warnings (non-critical errors)
	if len(warnings) > 0 {
		fmt.Println()
		slog.Warn("warnings detected", "count", len(warnings))
		for _, err := range warnings {
			slog.Warn(err.Error(),
				"type", string(err.Type),
				"operation", err.Op,
				"path", err.Path,
				"suggestion", err.Suggestion())
		}
	}

	// ModTime fallback summary
	if s.ModTimeFallbackCount > 0 {
		fmt.Println()
		slog.Warn("files used ModTime fallback",
			"count", s.ModTimeFallbackCount,
			"reason", "EXIF metadata unavailable or corrupted")
	}

	// Duplicates summary
	if len(s.DuplicatesDetected) > 0 || s.DuplicatesSkipped > 0 {
		fmt.Println()
		if s.DuplicatesSkipped > 0 {
			slog.Info("duplicates skipped",
				"count", s.DuplicatesSkipped)
			// Show first 10 duplicates
			count := 0
			for dup, original := range s.DuplicatesDetected {
				if count >= 10 {
					break
				}
				slog.Info("skipped duplicate",
					"file", filepath.Base(dup),
					"original", filepath.Base(original))
				count++
			}
			if len(s.DuplicatesDetected) > 10 {
				slog.Info("and more...", "additional", len(s.DuplicatesDetected)-10)
			}
		} else {
			// Duplicates detected but not skipped (warning mode)
			slog.Warn("duplicates detected (processed anyway)",
				"count", len(s.DuplicatesDetected))
			// Show first 10 duplicates
			count := 0
			for dup, original := range s.DuplicatesDetected {
				if count >= 10 {
					break
				}
				slog.Warn("duplicate detected",
					"file", filepath.Base(dup),
					"original", filepath.Base(original))
				count++
			}
			if len(s.DuplicatesDetected) > 10 {
				slog.Warn("and more...", "additional", len(s.DuplicatesDetected)-10)
			}
		}
	}

	// Cleanup summary
	if len(s.EmptyDirsRemoved) > 0 || len(s.EmptyDirsFailed) > 0 {
		fmt.Println()
		if len(s.EmptyDirsRemoved) > 0 {
			slog.Info("cleanup completed",
				"empty_dirs_removed", len(s.EmptyDirsRemoved))
			if len(s.EmptyDirsRemoved) <= 10 {
				for _, dir := range s.EmptyDirsRemoved {
					slog.Info("removed empty directory", "path", dir)
				}
			} else {
				slog.Info("showing first 10 removed directories")
				for i := 0; i < 10; i++ {
					slog.Info("removed empty directory", "path", s.EmptyDirsRemoved[i])
				}
				slog.Info("and more...", "additional", len(s.EmptyDirsRemoved)-10)
			}
		}
		if len(s.EmptyDirsFailed) > 0 {
			slog.Warn("cleanup warnings",
				"failed_removals", len(s.EmptyDirsFailed))
			for path, errMsg := range s.EmptyDirsFailed {
				slog.Warn("failed to remove directory",
					"path", path,
					"error", errMsg)
			}
		}
	}

	// Final status
	fmt.Println()
	if len(criticalErrors) > 0 {
		slog.Error("⚠ Operation completed with errors",
			"total_errors", len(criticalErrors),
			"files_failed", len(criticalErrors))
	} else if dryRun {
		slog.Info("DRY RUN completed - no files were actually moved")
	} else if s.ProcessedFiles == s.TotalFiles {
		slog.Info("✓ Operation completed successfully")
	} else {
		slog.Warn("⚠ Operation completed with some files skipped",
			"skipped", s.TotalFiles-s.ProcessedFiles)
	}
}
