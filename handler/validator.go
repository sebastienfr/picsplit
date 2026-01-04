package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ValidationReport holds the results of a fast validation scan
type ValidationReport struct {
	StartTime  time.Time
	EndTime    time.Time
	TotalFiles int
	PhotoCount int
	VideoCount int
	RawCount   int
	TotalBytes int64
	Errors     []*PicsplitError
	Warnings   []string
}

// Duration returns the validation duration
func (r *ValidationReport) Duration() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// HasCriticalErrors returns true if any critical errors were detected
func (r *ValidationReport) HasCriticalErrors() bool {
	for _, err := range r.Errors {
		if err.IsCritical() {
			return true
		}
	}
	return false
}

// CriticalErrorCount returns the number of critical errors
func (r *ValidationReport) CriticalErrorCount() int {
	count := 0
	for _, err := range r.Errors {
		if err.IsCritical() {
			count++
		}
	}
	return count
}

// Print displays the validation report
func (r *ValidationReport) Print() {
	fmt.Println()
	slog.Info("=== Validation Summary ===")

	// Duration
	duration := r.Duration()
	slog.Info("analysis completed", "duration", duration.Round(time.Second))

	// Files to process
	slog.Info("files to process", "count", r.TotalFiles)

	if r.TotalFiles > 0 {
		// File breakdown
		photoPercent := float64(r.PhotoCount) / float64(r.TotalFiles) * 100
		videoPercent := float64(r.VideoCount) / float64(r.TotalFiles) * 100
		rawPercent := float64(r.RawCount) / float64(r.TotalFiles) * 100

		slog.Info("file breakdown",
			"photos", r.PhotoCount,
			"photos_pct", fmt.Sprintf("%.1f%%", photoPercent),
			"videos", r.VideoCount,
			"videos_pct", fmt.Sprintf("%.1f%%", videoPercent),
			"raw", r.RawCount,
			"raw_pct", fmt.Sprintf("%.1f%%", rawPercent))
	}

	// Estimated disk space
	slog.Info("estimated disk space", "size", FormatBytes(r.TotalBytes))

	// Critical errors
	criticalCount := r.CriticalErrorCount()
	if criticalCount > 0 {
		fmt.Println()
		slog.Error("critical issues detected", "count", criticalCount)
		for _, err := range r.Errors {
			if err.IsCritical() {
				slog.Error(err.Error(),
					"type", string(err.Type),
					"suggestion", err.Suggestion())
			}
		}
	}

	// Warnings
	if len(r.Warnings) > 0 {
		fmt.Println()
		slog.Warn("warnings detected", "count", len(r.Warnings))
		for _, warning := range r.Warnings {
			slog.Warn(warning)
		}
	}

	// Final recommendation
	fmt.Println()
	if criticalCount > 0 {
		slog.Warn("→ Fix critical issues before proceeding")
	} else {
		slog.Info("✓ No critical issues detected")
	}
	slog.Info("→ Run with --mode dryrun to simulate, or --mode run to execute")
}

// Validate performs a fast validation of the media files without extracting EXIF metadata
// This is much faster than a full scan as it only checks file types, sizes, and permissions
func Validate(cfg *Config) (*ValidationReport, error) {
	report := &ValidationReport{
		StartTime: time.Now(),
	}

	// Create execution context for extension checking
	ctx, err := newExecutionContext(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize extension context: %w", err)
	}

	// Fast scan without EXIF extraction
	entries, err := os.ReadDir(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var unknownExts = make(map[string]bool) // Track unknown extensions (deduplicated)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			report.Warnings = append(report.Warnings,
				fmt.Sprintf("Cannot stat file: %s", entry.Name()))
			continue
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))

		// Check if it's a known media file
		isMediaFile := false
		if ctx.isPhoto(info.Name()) {
			if ctx.isRaw(info.Name()) {
				report.RawCount++
			} else {
				report.PhotoCount++
			}
			report.TotalBytes += info.Size()
			isMediaFile = true
		} else if ctx.isMovie(info.Name()) {
			report.VideoCount++
			report.TotalBytes += info.Size()
			isMediaFile = true
		} else if ext != "" {
			// Unknown extension
			unknownExts[ext] = true
		}

		// Check permissions (basic read access)
		if isMediaFile {
			filePath := filepath.Join(cfg.BasePath, info.Name())
			file, err := os.Open(filePath)
			if err != nil {
				report.Errors = append(report.Errors, &PicsplitError{
					Type: ErrTypePermission,
					Op:   "read_file",
					Path: filePath,
					Err:  err,
				})
			} else {
				file.Close()
			}
		}
	}

	// Report unknown extensions as validation errors
	for ext := range unknownExts {
		report.Errors = append(report.Errors, &PicsplitError{
			Type: ErrTypeValidation,
			Op:   "validate_extension",
			Path: ext,
			Details: map[string]string{
				"extension": strings.TrimPrefix(ext, "."),
			},
		})
	}

	report.TotalFiles = report.PhotoCount + report.VideoCount + report.RawCount
	report.EndTime = time.Now()

	return report, nil
}
