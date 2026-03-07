package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

const (
	// dateFormatLength is the expected length of date-formatted folder names ("2006 - 0102 - 1504")
	dateFormatLength = len(dateFormatPattern) // 18

	// organizedFolderThreshold is the ratio of date-formatted subdirs required
	// to consider a folder as already organized by picsplit
	organizedFolderThreshold = 0.5
)

// isDateFormattedName checks if a folder name matches the date format pattern
func isDateFormattedName(name string) bool {
	if len(name) != dateFormatLength {
		return false
	}
	_, err := time.Parse(dateFormatPattern, name)
	return err == nil
}

// isSpecialFolder checks if a folder name is a picsplit-managed special folder
func isSpecialFolder(name string) bool {
	return name == movFolderName || name == rawFolderName || name == orphanFolderName
}

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

	if isDateFormattedName(folderName) {
		slog.Info("current folder is date-formatted - using orphan refresh mode", "folder", folderName)
		return true
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
		if isSpecialFolder(name) {
			slog.Debug("skipping special folder", "folder", name)
			continue
		}

		totalDirs++

		if isDateFormattedName(name) {
			organizedDirs++
			slog.Debug("found organized subfolder", "folder", name)
		}
	}

	isOrganized := totalDirs > 0 && float64(organizedDirs)/float64(totalDirs) > organizedFolderThreshold
	slog.Debug("organized folder check", "total_dirs", totalDirs, "organized_dirs", organizedDirs, "is_organized", isOrganized)

	return isOrganized
}

// refreshOrphanRAW scans organized folders and separates orphan RAW files
// This is used when picsplit is run on an already organized directory
func refreshOrphanRAW(cfg *Config, ctx *executionContext) error {
	slog.Info("detected organized folder structure - refreshing orphan RAW separation")

	stats := NewProcessingStats()
	defer finalizeProcessing(cfg, stats)

	// Check if we're IN a date-formatted folder (e.g., running picsplit inside "2024 - 1220 - 0900")
	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		absPath = cfg.BasePath
	}

	folderName := filepath.Base(absPath)
	if isDateFormattedName(folderName) {
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

		// Skip special folders and non-date folders
		if isSpecialFolder(subFolderName) || !isDateFormattedName(subFolderName) {
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
		if !isRawPaired(ctx, rawFilePath, folderPath, folderPath) {
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

			if cfg.Mode == ModeDryRun {
				slog.Info("[DRY RUN] would move orphan RAW", "source", rawFilePath, "dest", destPath)
			} else {
				slog.Info("moving orphan RAW", "source", rawFilePath, "dest", destPath)
				if err := os.Rename(rawFilePath, destPath); err != nil {
					stats.AddError(&PicsplitError{
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
