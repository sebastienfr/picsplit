package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// fileGroup represents a group of files detected as an event
type fileGroup struct {
	folderName string
	firstFile  FileMetadata
	files      []FileMetadata
}

var (
	// Custom errors
	ErrNotDirectory = errors.New("path is not a directory")
	ErrInvalidDelta = errors.New("delta must be positive")
)

// isOrganizedFolder checks if we're running picsplit on an already organized folder
// Two cases:
// 1. The folder itself has a date-formatted name (e.g., "2024 - 1220 - 0900")
// 2. The folder contains subdirectories with date-formatted names
func isOrganizedFolder(basePath string) bool {
	// Resolve absolute path to get real folder name
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		slog.Debug("failed to get absolute path", "path", basePath, "error", err)
		absPath = basePath
	}

	// Case 1: Check if the folder name itself is date-formatted
	folderName := filepath.Base(absPath)
	slog.Debug("checking folder name", "folder", folderName, "path", absPath, "len", len(folderName))

	if len(folderName) >= 18 && len(folderName) <= 19 {
		slog.Debug("folder name length matches", "len", len(folderName))
		if parsed, err := time.Parse(dateFormatPattern, folderName); err == nil {
			slog.Info("current folder is date-formatted - using orphan refresh mode", "folder", folderName, "parsed", parsed)
			return true
		} else {
			slog.Debug("parse failed", "folder", folderName, "error", err)
		}
	}

	// Case 2: Check if folder contains date-formatted subdirectories
	entries, err := os.ReadDir(basePath)
	if err != nil {
		slog.Debug("failed to read directory for organized check", "path", basePath, "error", err)
		return false
	}

	var totalDirs int
	var organizedDirs int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip special folders - don't count them in totalDirs
		if name == movFolderName || name == rawFolderName || name == orphanFolderName {
			slog.Debug("skipping special folder", "folder", name)
			continue
		}

		totalDirs++

		// Check if name matches date format pattern (YYYY - MMDD - HHMM)
		// Format: "2006 - 0102 - 1504" has length 18
		if len(name) == 18 && name[4:7] == " - " && name[11:14] == " - " {
			if _, err := time.Parse(dateFormatPattern, name); err == nil {
				organizedDirs++
				slog.Debug("found organized subfolder", "folder", name)
			}
		}
	}

	isOrganized := totalDirs > 0 && float64(organizedDirs)/float64(totalDirs) > 0.5
	slog.Debug("organized folder check", "total_dirs", totalDirs, "organized_dirs", organizedDirs, "is_organized", isOrganized)

	return isOrganized
}

// collectMediaFilesWithMetadata retrieves all media files with their EXIF/video metadata
func collectMediaFilesWithMetadata(cfg *Config, ctx *executionContext) ([]FileMetadata, error) {
	entries, err := os.ReadDir(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var mediaFiles []FileMetadata
	var exifFailCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			slog.Warn("failed to get file info", "file", entry.Name(), "error", err)
			continue
		}

		// Use context to check if file is a media file
		if !ctx.isPhoto(info.Name()) && !ctx.isMovie(info.Name()) {
			slog.Debug("skipping file with unknown extension", "file", info.Name())
			continue
		}

		filePath := filepath.Join(cfg.BasePath, info.Name())

		// Extract metadata (EXIF/video)
		var metadata *FileMetadata
		if cfg.UseEXIF {
			metadata, err = ExtractMetadata(ctx, filePath)
			if err != nil || metadata.Source == DateSourceModTime {
				slog.Debug("failed to extract metadata, using ModTime", "file", info.Name())
				exifFailCount++
			}
		} else {
			// Mode without EXIF: use ModTime directly
			metadata = &FileMetadata{
				FileInfo: info,
				DateTime: info.ModTime(),
				GPS:      nil,
				Source:   DateSourceModTime,
			}
		}

		if metadata != nil {
			mediaFiles = append(mediaFiles, *metadata)
		}
	}

	// Selective fallback: files without EXIF use ModTime, others keep their extracted metadata
	// GPS coordinates are preserved even when DateTime falls back to ModTime
	if cfg.UseEXIF && exifFailCount > 0 {
		slog.Warn("files used ModTime fallback",
			"count", exifFailCount,
			"reason", "EXIF metadata unavailable or corrupted")
		// Each file keeps its individually extracted metadata (DateTime and GPS)
		// No global reset - this allows GPS clustering to work with mixed file sets
	}

	return mediaFiles, nil
}

// sortFilesByDateTime sorts files by ascending date/time (EXIF or ModTime)
func sortFilesByDateTime(files []FileMetadata) {
	sort.Slice(files, func(i, j int) bool {
		// If DateTime are equal, sort by name (deterministic)
		if files[i].DateTime.Equal(files[j].DateTime) {
			return files[i].FileInfo.Name() < files[j].FileInfo.Name()
		}
		return files[i].DateTime.Before(files[j].DateTime)
	})
}

// groupFilesByGaps groups files by time gaps
// A new group starts when gap > delta
func groupFilesByGaps(files []FileMetadata, delta time.Duration) []fileGroup {
	if len(files) == 0 {
		return nil
	}

	var groups []fileGroup

	currentGroup := fileGroup{
		files:     []FileMetadata{files[0]},
		firstFile: files[0],
	}

	for i := 1; i < len(files); i++ {
		gap := files[i].DateTime.Sub(files[i-1].DateTime)

		if gap <= delta {
			// Acceptable gap, continue the group
			currentGroup.files = append(currentGroup.files, files[i])
		} else {
			// Gap too large, finalize current group
			slog.Debug("gap exceeds delta, creating new group",
				"prev_file", files[i-1].FileInfo.Name(),
				"prev_time", files[i-1].DateTime,
				"curr_file", files[i].FileInfo.Name(),
				"curr_time", files[i].DateTime,
				"gap", gap,
				"delta", delta)
			currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
			groups = append(groups, currentGroup)

			// Start new group
			currentGroup = fileGroup{
				files:     []FileMetadata{files[i]},
				firstFile: files[i],
			}
		}
	}

	// Add last group
	currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
	groups = append(groups, currentGroup)

	return groups
}

// processGroup processes all files in a group
func processGroup(cfg *Config, ctx *executionContext, group fileGroup, stats *ProcessingStats, detector *DuplicateDetector) error {
	// Create main folder (unless dry-run)
	if cfg.Mode != ModeDryRun {
		groupDir := filepath.Join(cfg.BasePath, group.folderName)
		if err := os.MkdirAll(groupDir, permDirectory); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", groupDir, err)
		}
	}

	// Process each file
	for _, file := range group.files {
		fileName := file.FileInfo.Name()
		filePath := filepath.Join(cfg.BasePath, fileName)

		// Check if file is a duplicate
		if cfg.DetectDuplicates {
			isDup, original, err := detector.Check(filePath, file.FileInfo.Size())
			if err != nil {
				slog.Warn("failed to check duplicate", "file", fileName, "error", err)
				// Continue processing even if duplicate detection fails
			} else if isDup {
				// Duplicate detected
				stats.DuplicatesDetected[filePath] = original

				if cfg.SkipDuplicates {
					// Skip this file
					slog.Info("skipping duplicate", "file", fileName, "original", filepath.Base(original))
					stats.DuplicatesSkipped++
					continue
				} else if cfg.MoveDuplicates {
					// Move to duplicates/ folder
					duplicatesDir := filepath.Join(cfg.BasePath, duplicatesFolderName)
					if cfg.Mode != ModeDryRun {
						if err := os.MkdirAll(duplicatesDir, permDirectory); err != nil {
							slog.Error("failed to create duplicates folder", "error", err)
							if cfg.ContinueOnError {
								stats.AddError(fmt.Errorf("failed to create duplicates folder: %w", err))
								continue
							}
							return err
						}
					}

					if err := moveFile(cfg.BasePath, fileName, duplicatesFolderName, cfg.Mode == ModeDryRun); err != nil {
						slog.Error("failed to move duplicate", "file", fileName, "error", err)
						if cfg.ContinueOnError {
							stats.AddError(fmt.Errorf("failed to move duplicate %s: %w", fileName, err))
							continue
						}
						return err
					}
					if cfg.Mode == ModeDryRun {
						slog.Info("would move duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
					} else {
						slog.Info("moved duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
					}
					stats.DuplicatesSkipped++
					continue
				} else {
					// Just a warning, process anyway
					slog.Warn("duplicate detected (processing anyway)", "file", fileName, "original", filepath.Base(original))
				}
			}
		}

		// Process file normally
		if ctx.isPhoto(fileName) {
			if err := processPicture(cfg, ctx, file.FileInfo, group.folderName); err != nil {
				if cfg.ContinueOnError {
					stats.AddError(err)
					slog.Error("failed to process photo, continuing", "file", fileName, "error", err)
					continue
				}
				return err
			}
			stats.ProcessedFiles++
		} else if ctx.isMovie(fileName) {
			if err := processMovie(cfg, file.FileInfo, group.folderName); err != nil {
				if cfg.ContinueOnError {
					stats.AddError(err)
					slog.Error("failed to process video, continuing", "file", fileName, "error", err)
					continue
				}
				return err
			}
			stats.ProcessedFiles++
		}
	}

	return nil
}

// refreshOrphanRAW scans organized folders and separates orphan RAW files
// This is used when picsplit is run on an already organized directory
func refreshOrphanRAW(cfg *Config, ctx *executionContext) error {
	slog.Info("detected organized folder structure - refreshing orphan RAW separation")

	stats := &ProcessingStats{
		StartTime:        time.Now(),
		EmptyDirsFailed:  make(map[string]string),
		EmptyDirsRemoved: []string{},
	}
	defer func() {
		// Cleanup empty directories if requested (after all file operations)
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
	}()

	// Check if we're IN a date-formatted folder (e.g., running picsplit inside "2024 - 1220 - 0900")
	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		absPath = cfg.BasePath
	}

	folderName := filepath.Base(absPath)
	if len(folderName) >= 18 && len(folderName) <= 19 {
		if _, err := time.Parse(dateFormatPattern, folderName); err == nil {
			// We're inside a date-formatted folder, process it directly
			slog.Debug("processing current date-formatted folder", "folder", folderName)
			if err := processOrganizedFolder(cfg, ctx, stats, cfg.BasePath, folderName); err != nil {
				return err
			}
			// Fix stats before returning
			stats.TotalFiles = stats.RawCount
			stats.PhotoCount = 0
			stats.VideoCount = 0
			return nil
		}
	}

	// Otherwise, scan for date-formatted subfolders
	entries, err := os.ReadDir(cfg.BasePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Process each organized subfolder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		subFolderName := entry.Name()

		// Skip special folders
		if subFolderName == movFolderName || subFolderName == rawFolderName || subFolderName == orphanFolderName {
			continue
		}

		// Only process date-formatted folders
		if _, err := time.Parse(dateFormatPattern, subFolderName); err != nil {
			slog.Debug("skipping non-date folder", "folder", subFolderName)
			continue
		}

		folderPath := filepath.Join(cfg.BasePath, subFolderName)
		if err := processOrganizedFolder(cfg, ctx, stats, folderPath, subFolderName); err != nil {
			slog.Warn("failed to process folder", "folder", subFolderName, "error", err)
		}
	}

	// Fix stats for proper display in orphan RAW mode
	// In this mode, we only process RAW files
	stats.TotalFiles = stats.RawCount
	stats.PhotoCount = 0 // We don't process photos in orphan mode
	stats.VideoCount = 0 // We don't process videos in orphan mode

	return nil
}

// processOrganizedFolder processes a single organized folder to separate orphan RAW files
func processOrganizedFolder(cfg *Config, ctx *executionContext, stats *ProcessingStats, folderPath, folderName string) error {
	slog.Debug("processing organized folder", "folder", folderName)

	// Check if there's a raw/ subfolder
	rawPath := filepath.Join(folderPath, rawFolderName)
	if _, err := os.Stat(rawPath); os.IsNotExist(err) {
		slog.Debug("no raw folder found", "folder", folderName)
		return nil
	}

	// Scan RAW files in raw/ subfolder
	rawEntries, err := os.ReadDir(rawPath)
	if err != nil {
		return fmt.Errorf("failed to read raw folder: %w", err)
	}

	for _, rawEntry := range rawEntries {
		if rawEntry.IsDir() {
			continue
		}

		rawFileName := rawEntry.Name()
		if !ctx.isRaw(rawFileName) {
			continue
		}

		stats.RawCount++
		stats.ProcessedFiles++ // Count all RAW files as processed (examined)
		rawFilePath := filepath.Join(rawPath, rawFileName)

		// Check if RAW has associated JPEG/HEIC in parent folder
		if !isRawPaired(rawFilePath, folderPath, folderPath) {
			// Orphan RAW - move to orphan/ folder
			stats.OrphanRaw++

			orphanPath := filepath.Join(folderPath, orphanFolderName)
			if cfg.Mode != ModeDryRun {
				if err := os.MkdirAll(orphanPath, permDirectory); err != nil {
					slog.Error("failed to create orphan folder", "folder", folderName, "error", err)
					stats.ProcessedFiles-- // Decrement if we couldn't process
					continue
				}
			}

			destPath := filepath.Join(orphanPath, rawFileName)
			slog.Info("moving orphan RAW", "from", rawFilePath, "to", destPath, "dryrun", cfg.Mode == ModeDryRun)

			if cfg.Mode != ModeDryRun {
				if err := os.Rename(rawFilePath, destPath); err != nil {
					stats.Errors = append(stats.Errors, &PicsplitError{
						Type: ErrTypeIO,
						Op:   "move_orphan",
						Path: rawFilePath,
						Err:  err,
					})
					slog.Error("failed to move orphan RAW", "file", rawFileName, "error", err)
					stats.ProcessedFiles-- // Decrement on error
				}
			}
		} else {
			// Paired RAW - keep in raw/
			stats.PairedRaw++
			slog.Debug("keeping paired RAW", "file", rawFileName)
		}
	}

	return nil
}

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
	stats := &ProcessingStats{
		StartTime:          time.Now(),
		TotalFiles:         len(mediaFiles),
		EmptyDirsFailed:    make(map[string]string),
		EmptyDirsRemoved:   []string{},
		DuplicatesDetected: make(map[string]string),
		DuplicatesSkipped:  0,
	}
	defer func() {
		// Cleanup empty directories if requested (after all file operations)
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
	}()

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

		// Process each location cluster
		for _, cluster := range locationClusters {
			locationName := FormatLocationName(cluster.Centroid)
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
			if len(locationClusters) > 0 {
				slog.Info("processing files without GPS in folder",
					"count", len(filesWithoutGPS),
					"folder", GetNoLocationFolderName())
				for _, noGPSGroup := range noGPSGroups {
					folderName := filepath.Join(GetNoLocationFolderName(), noGPSGroup.folderName)
					groups = append(groups, fileGroup{
						folderName: folderName,
						firstFile:  noGPSGroup.firstFile,
						files:      noGPSGroup.files,
					})
				}
			} else {
				slog.Info("processing files without GPS at root (no location clusters)",
					"count", len(filesWithoutGPS))
				for _, noGPSGroup := range noGPSGroups {
					groups = append(groups, fileGroup{
						folderName: noGPSGroup.folderName,
						firstFile:  noGPSGroup.firstFile,
						files:      noGPSGroup.files,
					})
				}
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

	// 4. Process large groups (create folders) with progress bar
	bar := createProgressBar(len(largeGroups), "Processing groups", cfg.LogLevel, cfg.LogFormat)

	for i, group := range largeGroups {
		slog.Info("processing group",
			"current", i+1,
			"total", len(largeGroups),
			"folder", group.folderName,
			"files", len(group.files))

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
		slog.Info("processing small groups at root",
			"count", len(smallGroups),
			"total_files", stats.RootFilesCount)

		for _, group := range smallGroups {
			// Extract destination root from folder name
			// For GPS mode: "Paris/2024-0615-1200" → root = "Paris"
			// For time mode: "2024-0615-1200" → root = "" (basePath root)
			destinationRoot := ""
			if cfg.UseGPS && filepath.Dir(group.folderName) != "." {
				destinationRoot = filepath.Dir(group.folderName)
			}

			// Process each file in small group
			for _, file := range group.files {
				fileName := file.FileInfo.Name()
				filePath := filepath.Join(cfg.BasePath, fileName)

				// Check duplicates (same logic as processGroup)
				if cfg.DetectDuplicates {
					isDup, original, err := detector.Check(filePath, file.FileInfo.Size())
					if err != nil {
						slog.Warn("failed to check duplicate", "file", fileName, "error", err)
					} else if isDup {
						stats.DuplicatesDetected[filePath] = original

						if cfg.SkipDuplicates {
							slog.Info("skipping duplicate", "file", fileName, "original", filepath.Base(original))
							stats.DuplicatesSkipped++
							continue
						} else if cfg.MoveDuplicates {
							duplicatesDir := filepath.Join(cfg.BasePath, duplicatesFolderName)
							if cfg.Mode != ModeDryRun {
								if err := os.MkdirAll(duplicatesDir, permDirectory); err != nil {
									slog.Error("failed to create duplicates folder", "error", err)
									if cfg.ContinueOnError {
										stats.AddError(fmt.Errorf("failed to create duplicates folder: %w", err))
										continue
									}
									return err
								}
							}

							if err := moveFile(cfg.BasePath, fileName, duplicatesFolderName, cfg.Mode == ModeDryRun); err != nil {
								slog.Error("failed to move duplicate", "file", fileName, "error", err)
								if cfg.ContinueOnError {
									stats.AddError(fmt.Errorf("failed to move duplicate %s: %w", fileName, err))
									continue
								}
								return err
							}
							if cfg.Mode == ModeDryRun {
								slog.Info("would move duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
							} else {
								slog.Info("moved duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
							}
							stats.DuplicatesSkipped++
							continue
						} else {
							slog.Warn("duplicate detected (processing anyway)", "file", fileName, "original", filepath.Base(original))
						}
					}
				}

				// Process file at root
				if ctx.isPhoto(fileName) {
					if err := processPictureAtRoot(cfg, ctx, file.FileInfo, destinationRoot); err != nil {
						if cfg.ContinueOnError {
							stats.AddError(err)
							slog.Error("failed to process photo at root, continuing", "file", fileName, "error", err)
							continue
						}
						return err
					}
					stats.ProcessedFiles++
				} else if ctx.isMovie(fileName) {
					if err := processMovieAtRoot(cfg, file.FileInfo, destinationRoot); err != nil {
						if cfg.ContinueOnError {
							stats.AddError(err)
							slog.Error("failed to process video at root, continuing", "file", fileName, "error", err)
							continue
						}
						return err
					}
					stats.ProcessedFiles++
				}
			}
		}
	}

	// Return error if critical errors occurred
	if stats.HasCriticalErrors() {
		return fmt.Errorf("processing completed with %d critical error(s)", len(stats.Errors))
	}

	return nil
}

// processPicture handles the processing of picture files
func processPicture(cfg *Config, ctx *executionContext, fi os.FileInfo, datedFolder string) error {
	slog.Debug("processing picture", "file", fi.Name(), "dest_folder", datedFolder)

	destDir := datedFolder

	// Special handling for RAW files
	if ctx.isRaw(fi.Name()) && !cfg.NoMoveRaw {
		baseRawDir := filepath.Join(cfg.BasePath, datedFolder)

		// Determine if RAW goes to raw/ or orphan/
		targetFolder := rawFolderName

		if cfg.SeparateOrphanRaw {
			// Check if RAW has associated JPEG/HEIC
			// Search in source (basePath) AND in destination (datedFolder)
			// because JPEG may have already been moved
			rawFilePath := filepath.Join(cfg.BasePath, fi.Name())
			destFolder := filepath.Join(cfg.BasePath, datedFolder)
			if !isRawPaired(rawFilePath, cfg.BasePath, destFolder) {
				targetFolder = orphanFolderName
				slog.Debug("orphan RAW file (no JPEG/HEIC)", "file", fi.Name(), "dest", orphanFolderName)
			}
		}

		rawDir, err := findOrCreateFolder(baseRawDir, targetFolder, cfg.Mode == ModeDryRun)
		if err != nil {
			return err
		}
		destDir = filepath.Join(datedFolder, rawDir)
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.Mode == ModeDryRun)
}

// processMovie handles the processing of movie files
func processMovie(cfg *Config, fi os.FileInfo, datedFolder string) error {
	slog.Debug("processing movie", "file", fi.Name(), "dest_folder", datedFolder)

	destDir := datedFolder

	// Move to separate mov folder if needed
	if !cfg.NoMoveMovie {
		baseMovieDir := filepath.Join(cfg.BasePath, datedFolder)
		movieDir, err := findOrCreateFolder(baseMovieDir, movFolderName, cfg.Mode == ModeDryRun)
		if err != nil {
			return err
		}
		destDir = filepath.Join(datedFolder, movieDir)
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.Mode == ModeDryRun)
}

func findOrCreateFolder(basedir, name string, dryRun bool) (string, error) {
	dirCreate := filepath.Join(basedir, name)

	slog.Debug("finding or creating folder", "path", dirCreate)

	if dryRun {
		return name, nil
	}

	fi, err := os.Stat(dirCreate)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the folder
			if err := os.Mkdir(dirCreate, permDirectory); err != nil {
				return "", fmt.Errorf("failed to create folder %s: %w", dirCreate, err)
			}

			fi, err = os.Stat(dirCreate)
			if err != nil {
				return "", fmt.Errorf("failed to stat created folder: %w", err)
			}

			return fi.Name(), nil
		}

		return "", fmt.Errorf("failed to stat folder %s: %w", dirCreate, err)
	}

	return fi.Name(), nil
}

func moveFile(basedir, src, dest string, dryRun bool) error {
	srcPath := filepath.Join(basedir, src)
	dstPath := filepath.Join(basedir, dest, src)

	if dryRun {
		slog.Info("[DRY RUN] would move file", "source", srcPath, "dest", dstPath)
		return nil
	}

	slog.Info("moving file", "source", srcPath, "dest", dstPath)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move %s to %s: %w", srcPath, dstPath, err)
	}

	return nil
}

// isRawPaired checks if a RAW file has an associated JPEG or HEIC
// Searches in the source directory and optionally in the destination folder
// (since JPEG may have already been moved during processing)
func isRawPaired(rawPath string, basePath string, destFolder string) bool {
	baseName := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))

	// Extensions to search for (JPEG and HEIC for iPhone)
	photoExtensions := []string{".jpg", ".JPG", ".jpeg", ".JPEG", ".heic", ".HEIC"}

	// 1. Search in source folder (basePath)
	for _, ext := range photoExtensions {
		photoPath := filepath.Join(basePath, baseName+ext)
		if _, err := os.Stat(photoPath); err == nil {
			slog.Debug("found paired photo in source", "photo", photoPath, "raw", filepath.Base(rawPath))
			return true
		}
	}

	// 2. Search in destination folder (JPEG already moved)
	if destFolder != "" {
		for _, ext := range photoExtensions {
			photoPath := filepath.Join(destFolder, baseName+ext)
			if _, err := os.Stat(photoPath); err == nil {
				slog.Debug("found paired photo in destination", "photo", photoPath, "raw", filepath.Base(rawPath))
				return true
			}
		}
	}

	return false // Orphelin
}

// processPictureAtRoot handles the processing of picture files for small groups (left at root)
func processPictureAtRoot(cfg *Config, ctx *executionContext, fi os.FileInfo, destinationRoot string) error {
	slog.Debug("processing picture at root", "file", fi.Name(), "dest_root", destinationRoot)

	destDir := destinationRoot

	// Special handling for RAW files
	if ctx.isRaw(fi.Name()) && !cfg.NoMoveRaw {
		baseRawDir := cfg.BasePath
		if destinationRoot != "" {
			baseRawDir = filepath.Join(cfg.BasePath, destinationRoot)
		}

		// Determine if RAW goes to raw/ or orphan/
		targetFolder := rawFolderName

		if cfg.SeparateOrphanRaw {
			rawFilePath := filepath.Join(cfg.BasePath, fi.Name())
			destFolder := baseRawDir
			if !isRawPaired(rawFilePath, cfg.BasePath, destFolder) {
				targetFolder = orphanFolderName
				slog.Debug("orphan RAW file (no JPEG/HEIC)", "file", fi.Name(), "dest", orphanFolderName)
			}
		}

		rawDir, err := findOrCreateFolder(baseRawDir, targetFolder, cfg.Mode == ModeDryRun)
		if err != nil {
			return err
		}
		if destinationRoot != "" {
			destDir = filepath.Join(destinationRoot, rawDir)
		} else {
			destDir = rawDir
		}
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.Mode == ModeDryRun)
}

// processMovieAtRoot handles the processing of movie files for small groups (left at root)
func processMovieAtRoot(cfg *Config, fi os.FileInfo, destinationRoot string) error {
	slog.Debug("processing movie at root", "file", fi.Name(), "dest_root", destinationRoot)

	destDir := destinationRoot

	// Move to separate mov folder if needed
	if !cfg.NoMoveMovie {
		baseMovieDir := cfg.BasePath
		if destinationRoot != "" {
			baseMovieDir = filepath.Join(cfg.BasePath, destinationRoot)
		}

		movieDir, err := findOrCreateFolder(baseMovieDir, movFolderName, cfg.Mode == ModeDryRun)
		if err != nil {
			return err
		}
		if destinationRoot != "" {
			destDir = filepath.Join(destinationRoot, movieDir)
		} else {
			destDir = movieDir
		}
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.Mode == ModeDryRun)
}
