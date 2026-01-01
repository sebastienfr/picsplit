package handler

import (
	"fmt"
	"log/slog"
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

	// Issues
	ModTimeFallbackCount int // Files that fell back to ModTime
	Errors               []*PicsplitError
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
