package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
)

const (
	// Folder configuration
	movFolderName        = "mov"
	rawFolderName        = "raw"
	orphanFolderName     = "orphan"
	duplicatesFolderName = "duplicates"
	dateFormatPattern    = "2006 - 0102 - 1504"
)

var (
	// Custom errors
	ErrNotDirectory = errors.New("path is not a directory")
	ErrInvalidDelta = errors.New("delta must be positive")
)

// Split is the main function that moves files to dated folders according to configuration
func Split(cfg *Config) error {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Handle execution modes
	switch cfg.Mode {
	case ModeValidate:
		// Fast validation without EXIF extraction
		report, err := Validate(cfg)
		if err != nil {
			return err
		}
		report.Print()
		if report.HasCriticalErrors() {
			return fmt.Errorf("validation found %d critical error(s)", report.CriticalErrorCount())
		}
		return nil

	case ModeDryRun, ModeRun:
		// Continue with split processing
		return splitInternal(cfg)

	default:
		return fmt.Errorf("unknown execution mode: %v", cfg.Mode)
	}
}

// finalizeProcessing handles cleanup of empty directories and prints the processing summary.
// Used as a deferred call in both splitInternal and refreshOrphanRAW.
func finalizeProcessing(cfg *Config, stats *ProcessingStats) {
	if cfg.CleanupEmptyDirs && cfg.Mode != ModeValidate {
		slog.Info("cleaning up empty directories", "path", cfg.BasePath)
		result, err := CleanupEmptyDirs(cfg.BasePath, cfg.Mode, cfg.Force, cfg.CleanupIgnore)
		if err != nil {
			slog.Warn("cleanup failed", "error", err)
		} else {
			stats.EmptyDirsRemoved = result.RemovedDirs
			for path, cleanupErr := range result.FailedDirs {
				stats.EmptyDirsFailed[path] = cleanupErr.Error()
			}
		}
	}

	stats.EndTime = time.Now()
	stats.PrintSummary(cfg.Mode == ModeDryRun)
}

// splitInternal is the internal implementation of Split
func splitInternal(cfg *Config) error {
	// Create execution context with custom extensions
	ctx, err := newExecutionContext(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize extension context: %w", err)
	}

	// Check if we're in an already organized folder
	if cfg.SeparateOrphanRaw && isOrganizedFolder(cfg.BasePath) {
		slog.Info("detected organized folder - running orphan refresh mode")
		return refreshOrphanRAW(cfg, ctx)
	}

	// 1. Collect media files with metadata
	mediaFiles, err := collectMediaFilesWithMetadata(cfg, ctx)
	if err != nil {
		return fmt.Errorf("failed to collect media files: %w", err)
	}

	if len(mediaFiles) == 0 {
		slog.Info("no media files found")
		return nil
	}

	slog.Info("media files collected", "count", len(mediaFiles))

	// Initialize processing statistics
	stats := NewProcessingStats()
	stats.TotalFiles = len(mediaFiles)
	defer finalizeProcessing(cfg, stats)

	// Create duplicate detector if enabled
	detector := NewDuplicateDetector(cfg.DetectDuplicates)

	// Count file types and ModTime fallback
	// AND pre-fill duplicate detector by size
	for _, mf := range mediaFiles {
		fileName := mf.FileInfo.Name()
		if ctx.isPhoto(fileName) {
			if ctx.isRaw(fileName) {
				stats.RawCount++
			} else {
				stats.PhotoCount++
			}
		} else if ctx.isMovie(fileName) {
			stats.VideoCount++
		}

		// Track ModTime fallback
		if mf.Source == DateSourceModTime {
			stats.ModTimeFallbackCount++
		}

		// Track total bytes
		stats.TotalBytes += mf.FileInfo.Size()

		// Add to size pre-filtering (duplicates optimization)
		if cfg.DetectDuplicates {
			filePath := filepath.Join(cfg.BasePath, mf.FileInfo.Name())
			detector.AddFile(filePath, mf.FileInfo.Size())
		}
	}

	var groups []fileGroup

	// 2. GPS clustering mode or classic time-based mode
	if cfg.UseGPS {
		// Log GPS coverage analysis
		var filesWithGPSCount int
		for _, mf := range mediaFiles {
			if mf.GPS != nil {
				filesWithGPSCount++
			}
		}

		gpsPercentage := 0.0
		if len(mediaFiles) > 0 {
			gpsPercentage = float64(filesWithGPSCount) / float64(len(mediaFiles)) * 100
		}

		slog.Info("GPS coverage analysis",
			"files_with_gps", filesWithGPSCount,
			"total_files", len(mediaFiles),
			"coverage_pct", fmt.Sprintf("%.1f%%", gpsPercentage))

		// GPS clustering: location FIRST, then time within each location
		locationClusters, filesWithoutGPS := ClusterByLocation(mediaFiles, cfg.GPSRadius)

		slog.Info("GPS clustering completed",
			"location_clusters", len(locationClusters),
			"files_without_gps", len(filesWithoutGPS))

		// Pre-geocode all locations if geocoding is enabled
		// This shows progress and prepares location names before processing
		locationNames := make(map[int]string)
		geocodingSvc := NewGeocodingService()
		if cfg.GPSUseGeocoding && len(locationClusters) > 0 {
			geocodingBar := createProgressBar(len(locationClusters), "Geocoding locations", cfg.LogLevel, cfg.LogFormat)

			for i, cluster := range locationClusters {
				locationNames[i] = geocodingSvc.ReverseGeocode(cluster.Centroid.Lat, cluster.Centroid.Lon, cfg.GPSUseGeocoding)

				if geocodingBar != nil {
					_ = geocodingBar.Add(1)
				}
			}
		}

		// Now process each location cluster (geocoding already done)
		for i, cluster := range locationClusters {
			var locationName string
			if cfg.GPSUseGeocoding {
				locationName = locationNames[i]
			} else {
				locationName = FormatLocationName(cluster.Centroid)
			}

			slog.Debug("processing location cluster", "location", locationName, "files", len(cluster.Files))

			// Group by time within this location
			timeGroups := GroupLocationByTime(cluster, cfg.Delta)
			slog.Debug("location split into time groups", "location", locationName, "time_groups", len(timeGroups))

			// Create fileGroup for each time group
			for _, timeGroup := range timeGroups {
				if len(timeGroup) == 0 {
					continue
				}

				folderName := filepath.Join(locationName, timeGroup[0].DateTime.Format(dateFormatPattern))
				groups = append(groups, fileGroup{
					folderName: folderName,
					firstFile:  timeGroup[0],
					files:      timeGroup,
				})
			}
		}

		// Process files without GPS
		if len(filesWithoutGPS) > 0 {
			// Sort and group by time
			sortFilesByDateTime(filesWithoutGPS)
			noGPSGroups := groupFilesByGaps(filesWithoutGPS, cfg.Delta)

			// If location clusters exist, create "NoLocation" subfolder
			// Otherwise, put directly at root (no need for segregation)
			prefix := ""
			if len(locationClusters) > 0 {
				prefix = GetNoLocationFolderName()
				slog.Info("processing files without GPS in folder",
					"count", len(filesWithoutGPS),
					"folder", prefix)
			} else {
				slog.Info("processing files without GPS at root (no location clusters)",
					"count", len(filesWithoutGPS))
			}

			for _, noGPSGroup := range noGPSGroups {
				folderName := noGPSGroup.folderName
				if prefix != "" {
					folderName = filepath.Join(prefix, folderName)
				}
				groups = append(groups, fileGroup{
					folderName: folderName,
					firstFile:  noGPSGroup.firstFile,
					files:      noGPSGroup.files,
				})
			}
		}
	} else {
		// Classic time-based mode (backward compatible)
		// 2. Sort chronologically
		sortFilesByDateTime(mediaFiles)

		// 3. Group by gaps
		groups = groupFilesByGaps(mediaFiles, cfg.Delta)
	}

	slog.Info("event groups detected",
		"count", len(groups),
		"delta", cfg.Delta)

	// 3. Filter groups by MinGroupSize (v2.9.0+)
	// Groups below threshold will have files left at root instead of creating folder
	largeGroups := make([]fileGroup, 0, len(groups))
	smallGroups := make([]fileGroup, 0)

	for _, group := range groups {
		if len(group.files) >= cfg.MinGroupSize {
			largeGroups = append(largeGroups, group)
		} else {
			smallGroups = append(smallGroups, group)
			stats.SmallGroupsCount++
			stats.RootFilesCount += len(group.files)
		}
	}

	if stats.SmallGroupsCount > 0 {
		slog.Info("filtered small groups",
			"small_groups", stats.SmallGroupsCount,
			"files_at_root", stats.RootFilesCount,
			"threshold", cfg.MinGroupSize)
	}

	// Track groups created (only large groups create folders)
	stats.GroupsCreated = len(largeGroups)

	// 4. Process ALL groups (large + small) with combined progress bar
	totalGroups := len(largeGroups) + len(smallGroups)
	bar := createProgressBar(totalGroups, "Processing groups", cfg.LogLevel, cfg.LogFormat)

	// Process large groups (create folders)
	for i, group := range largeGroups {
		// Use Debug level when progress bar is active to avoid visual interference
		if bar != nil {
			slog.Debug("processing group",
				"current", i+1,
				"total", totalGroups,
				"folder", group.folderName,
				"files", len(group.files))
		} else {
			slog.Info("processing group",
				"current", i+1,
				"total", totalGroups,
				"folder", group.folderName,
				"files", len(group.files))
		}

		if err := processGroup(cfg, ctx, group, stats, detector); err != nil {
			// Track error at group level
			stats.AddError(&PicsplitError{
				Type:    ErrTypeIO,
				Op:      "process_group",
				Path:    group.folderName,
				Err:     err,
				Details: map[string]string{"file_count": fmt.Sprintf("%d", len(group.files))},
			})

			if !cfg.ContinueOnError {
				slog.Error("failed to process group, stopping", "folder", group.folderName, "error", err)
				return err
			}

			slog.Error("failed to process group, continuing", "folder", group.folderName, "error", err)
		}

		if bar != nil {
			_ = bar.Add(1)
		}
	}

	// 5. Process small groups (leave files at root)
	if len(smallGroups) > 0 {
		if bar == nil {
			slog.Info("processing small groups at root",
				"count", len(smallGroups),
				"total_files", stats.RootFilesCount)
		}

		for i, group := range smallGroups {
			// Extract destination root from folder name
			// For GPS mode: "Paris/2024-0615-1200" → root = "Paris"
			// For time mode: "2024-0615-1200" → root = "" (basePath root)
			destinationRoot := ""
			if cfg.UseGPS && filepath.Dir(group.folderName) != "." {
				destinationRoot = filepath.Dir(group.folderName)
			}

			if bar != nil {
				slog.Debug("processing small group at root",
					"current", len(largeGroups)+i+1,
					"total", totalGroups,
					"files", len(group.files))
			}

			// Process each file in small group
			for _, file := range group.files {
				fileName := file.FileInfo.Name()
				filePath := filepath.Join(cfg.BasePath, fileName)

				skip, err := handleDuplicate(cfg, detector, stats, filePath, fileName, file.FileInfo.Size())
				if err != nil {
					return err
				}
				if skip {
					continue
				}

				// Process file at root
				if ctx.isPhoto(fileName) || ctx.isMovie(fileName) {
					if err := processMediaFile(cfg, ctx, file.FileInfo, "", destinationRoot); err != nil {
						if cfg.ContinueOnError {
							stats.AddError(err)
							slog.Error("failed to process file at root, continuing", "file", fileName, "error", err)
							continue
						}
						return err
					}
					stats.ProcessedFiles++
				}
			}

			// Update progress bar after processing each small group
			if bar != nil {
				_ = bar.Add(1)
			}
		}
	}

	// Return error if critical errors occurred
	if stats.HasCriticalErrors() {
		return fmt.Errorf("processing completed with %d critical error(s)", len(stats.Errors))
	}

	return nil
}
