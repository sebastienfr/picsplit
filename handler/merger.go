package handler

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const (
	// Conflict resolution strategies
	conflictAsk       = "ask"       // Ask user for each conflict
	conflictRename    = "rename"    // Rename conflicting file
	conflictSkip      = "skip"      // Skip conflicting file
	conflictOverwrite = "overwrite" // Overwrite target file
	conflictQuit      = "quit"      // Abort merge operation
	conflictApplyAll  = "all"       // Apply choice to all remaining conflicts

	// Allowed subdirectory names in media folders
	allowedSubdirMov = "mov"
	allowedSubdirRaw = "raw"

	// File permissions (Unix octal notation)
	// permDirectory: 0755 = rwxr-xr-x
	// Binary representation: 111 101 101
	// - Owner (7 = 111 binary): read(4) + write(2) + execute(1) = full access
	// - Group (5 = 101 binary): read(4) + execute(1) = read and traverse directory
	// - Others (5 = 101 binary): read(4) + execute(1) = read and traverse directory
	// Used for creating directories throughout the handler package
	permDirectory = 0755
)

// MergeConfig contains configuration for merge operation
//
//nolint:govet // Field alignment is less important than logical grouping
type MergeConfig struct {
	SourceFolders []string // Source folders to merge (min 1)
	TargetFolder  string   // Destination folder
	Force         bool     // Force overwrite on conflicts
	DryRun        bool     // Simulation mode

	// Custom extensions (v2.5.0+)
	CustomPhotoExts []string // Additional photo extensions
	CustomVideoExts []string // Additional video extensions
	CustomRawExts   []string // Additional RAW extensions
}

// FileConflict represents a file conflict between source and target
type FileConflict struct {
	SourceInfo os.FileInfo
	TargetInfo os.FileInfo
	SourcePath string
	TargetPath string
}

// mergeStats tracks merge operation statistics
type mergeStats struct {
	filesProcessed   int
	filesMoved       int
	filesSkipped     int
	filesRenamed     int
	filesOverwritten int
	foldersDeleted   int
	conflicts        int
}

// isMediaFolderWithContext validates that a folder contains only media files and allowed subdirectories (mov/, raw/)
// This prevents merging non-media folders (like GPS location folders or arbitrary directories)
func isMediaFolderWithContext(folderPath string, ctx *executionContext) error {
	entries, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read folder %s: %w", folderPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Only allow allowedSubdirMov and allowedSubdirRaw subdirectories
			dirName := strings.ToLower(entry.Name())
			if dirName != allowedSubdirMov && dirName != allowedSubdirRaw {
				return fmt.Errorf("folder %s contains non-media subdirectory: %s (only '%s' and '%s' subdirectories are allowed)", folderPath, entry.Name(), allowedSubdirMov, allowedSubdirRaw)
			}

			// Recursively validate subdirectories
			subPath := filepath.Join(folderPath, entry.Name())
			if err := isMediaFolderWithContext(subPath, ctx); err != nil {
				return err
			}
		} else {
			// Check if file is a media file using context
			if !ctx.isMediaFile(entry.Name()) {
				return fmt.Errorf("folder %s contains non-media file: %s", folderPath, entry.Name())
			}
		}
	}

	return nil
}

// validateMergeFolders validates that folders can be merged
func validateMergeFolders(sources []string, target string, ctx *executionContext) error {
	// Check minimum arguments
	if len(sources) < 1 {
		return fmt.Errorf("merge requires at least 1 source folder")
	}

	// Check each source folder
	for _, source := range sources {
		// Check if folder exists
		info, err := os.Stat(source)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("source folder does not exist: %s", source)
			}
			return fmt.Errorf("cannot access source folder %s: %w", source, err)
		}

		// Check if it's a directory
		if !info.IsDir() {
			return fmt.Errorf("source is not a directory: %s", source)
		}

		// Validate that folder contains only media files and allowed subdirectories
		if err := isMediaFolderWithContext(source, ctx); err != nil {
			return fmt.Errorf("source folder is not a valid media folder: %w", err)
		}
	}

	// If target exists, verify it's a directory
	if info, err := os.Stat(target); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("target exists but is not a directory: %s", target)
		}
	}

	return nil
}

// collectFilesRecursive collects all files from a directory recursively
func collectFilesRecursive(rootDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, collect only files
		if !info.IsDir() {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to collect files from %s: %w", rootDir, err)
	}

	return files, nil
}

// generateUniqueName generates a unique filename to avoid conflicts
// Example: photo.jpg -> photo_1.jpg -> photo_2.jpg
func generateUniqueName(targetPath string) string {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)

	counter := 1
	for {
		newName := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		newPath := filepath.Join(dir, newName)

		// Check if this name is available
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}

		counter++
	}
}

// detectConflict checks if a file already exists at target path
func detectConflict(targetPath string) (*FileConflict, error) {
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No conflict
		}
		return nil, fmt.Errorf("cannot check target file %s: %w", targetPath, err)
	}

	return &FileConflict{
		TargetPath: targetPath,
		TargetInfo: targetInfo,
	}, nil
}

// askUserConflictResolution asks user how to resolve a file conflict
// Returns: (resolution, applyToAll, error)
func askUserConflictResolution(conflict *FileConflict) (string, bool, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println()
	fmt.Printf("⚠️  Conflict detected: %s\n", filepath.Base(conflict.TargetPath))
	fmt.Printf("   Source: %s (%d bytes)\n", conflict.SourcePath, conflict.SourceInfo.Size())
	fmt.Printf("   Target: %s (%d bytes)\n", conflict.TargetPath, conflict.TargetInfo.Size())
	fmt.Println()
	fmt.Println("Choose action:")
	fmt.Println("  [r] Rename source file (avoid conflict)")
	fmt.Println("  [s] Skip this file (keep target version)")
	fmt.Println("  [o] Overwrite (replace target with source)")
	fmt.Println("  [a] Apply to all remaining conflicts")
	fmt.Println("  [q] Quit merge operation")
	fmt.Print("Choice: ")

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", false, fmt.Errorf("failed to read user input: %w", err)
	}

	choice := strings.TrimSpace(strings.ToLower(input))

	switch choice {
	case "r":
		return conflictRename, false, nil
	case "s":
		return conflictSkip, false, nil
	case "o":
		return conflictOverwrite, false, nil
	case "a":
		// Ask what action to apply to all
		fmt.Print("Apply which action to all? [r/s/o]: ")
		actionInput, err := reader.ReadString('\n')
		if err != nil {
			return "", false, fmt.Errorf("failed to read action input: %w", err)
		}
		action := strings.TrimSpace(strings.ToLower(actionInput))

		switch action {
		case "r":
			return conflictRename, true, nil
		case "s":
			return conflictSkip, true, nil
		case "o":
			return conflictOverwrite, true, nil
		default:
			fmt.Println("Invalid action. Using 'skip' for all.")
			return conflictSkip, true, nil
		}
	case "q":
		return conflictQuit, false, nil
	default:
		fmt.Println("Invalid choice. Skipping this file.")
		return conflictSkip, false, nil
	}
}

// Merge merges multiple source folders into a target folder
//
//nolint:gocyclo // Complex conflict handling logic, acceptable for this use case
func Merge(cfg *MergeConfig) error {
	// Create execution context with custom extensions
	tempCfg := &Config{
		CustomPhotoExts: cfg.CustomPhotoExts,
		CustomVideoExts: cfg.CustomVideoExts,
		CustomRawExts:   cfg.CustomRawExts,
	}

	ctx, err := newExecutionContext(tempCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize extension context: %w", err)
	}

	// Validate configuration
	if err := validateMergeFolders(cfg.SourceFolders, cfg.TargetFolder, ctx); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	stats := &mergeStats{}

	// Global conflict resolution mode (when "apply to all" is chosen)
	var globalResolution string
	applyToAll := false

	slog.Info("starting merge operation",
		"sources", cfg.SourceFolders,
		"target", cfg.TargetFolder)
	if cfg.Force {
		slog.Info("merge mode: FORCE", "auto_overwrite", true)
	}
	if cfg.DryRun {
		slog.Info("merge mode: DRY RUN", "simulation", true)
	}

	// Create target folder if it doesn't exist
	if !cfg.DryRun {
		if err := os.MkdirAll(cfg.TargetFolder, permDirectory); err != nil {
			return fmt.Errorf("failed to create target folder: %w", err)
		}
	} else {
		slog.Info("[DRY RUN] would create target folder", "folder", cfg.TargetFolder)
	}

	// Process each source folder
	for _, sourceFolder := range cfg.SourceFolders {
		slog.Info("processing source folder", "folder", sourceFolder)

		// Collect all files from source
		files, err := collectFilesRecursive(sourceFolder)
		if err != nil {
			return err
		}

		slog.Debug("files found in source", "count", len(files), "folder", sourceFolder)

		// Process each file
		for _, file := range files {
			stats.filesProcessed++

			// Calculate relative path
			relPath, err := filepath.Rel(sourceFolder, file)
			if err != nil {
				return fmt.Errorf("failed to calculate relative path: %w", err)
			}

			// Calculate target path
			targetPath := filepath.Join(cfg.TargetFolder, relPath)

			// Check for conflict
			conflict, err := detectConflict(targetPath)
			if err != nil {
				return err
			}

			var finalTargetPath string
			if conflict != nil {
				stats.conflicts++

				// Fill in source info
				sourceInfo, err := os.Stat(file)
				if err != nil {
					return fmt.Errorf("failed to stat source file %s: %w", file, err)
				}
				conflict.SourcePath = file
				conflict.SourceInfo = sourceInfo

				// Determine resolution strategy
				var resolution string
				if cfg.Force {
					resolution = conflictOverwrite
				} else if applyToAll {
					resolution = globalResolution
				} else {
					if cfg.DryRun {
						// In dry-run, simulate asking user
						slog.Warn("[DRY RUN] conflict detected (would ask user)", "file", filepath.Base(targetPath))
						resolution = conflictSkip // Default for dry-run
					} else {
						// Ask user
						var applyAll bool
						resolution, applyAll, err = askUserConflictResolution(conflict)
						if err != nil {
							return err
						}

						if applyAll {
							applyToAll = true
							globalResolution = resolution
							slog.Info("applying resolution to all remaining conflicts", "resolution", resolution)
						}
					}
				}

				// Handle quit
				if resolution == conflictQuit {
					return fmt.Errorf("merge canceled by user")
				}

				// Apply resolution
				switch resolution {
				case conflictRename:
					finalTargetPath = generateUniqueName(targetPath)
					stats.filesRenamed++
					slog.Info("renaming to avoid conflict", "file", filepath.Base(finalTargetPath))
				case conflictSkip:
					stats.filesSkipped++
					slog.Info("skipping file (keeping target)", "file", filepath.Base(file))
					continue // Skip this file
				case conflictOverwrite:
					finalTargetPath = targetPath
					stats.filesOverwritten++
					slog.Info("overwriting target", "file", filepath.Base(targetPath))
				}
			} else {
				finalTargetPath = targetPath
			}

			// Create parent directory
			targetDir := filepath.Dir(finalTargetPath)
			if !cfg.DryRun {
				if err := os.MkdirAll(targetDir, permDirectory); err != nil {
					return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
				}
			}

			// Move the file
			if cfg.DryRun {
				slog.Info("[DRY RUN] would move file", "source", file, "dest", finalTargetPath)
			} else {
				if err := os.Rename(file, finalTargetPath); err != nil {
					return fmt.Errorf("failed to move %s to %s: %w", file, finalTargetPath, err)
				}
				stats.filesMoved++
				slog.Debug("moved file", "source", file, "dest", finalTargetPath)
			}
		}

		// Cleanup source folder after processing all files
		if cfg.DryRun {
			slog.Info("[DRY RUN] would delete source folder", "folder", sourceFolder)
		} else {
			// Remove the folder (including empty subdirectories like mov/, raw/)
			if err := os.RemoveAll(sourceFolder); err != nil {
				slog.Warn("failed to remove source folder", "folder", sourceFolder, "error", err)
			} else {
				stats.foldersDeleted++
				slog.Info("deleted source folder", "folder", sourceFolder)
			}
		}
	}

	// Print summary
	fmt.Println()
	slog.Info("=== Merge Summary ===")
	slog.Info("merge statistics",
		"files_processed", stats.filesProcessed,
		"files_moved", stats.filesMoved)
	if stats.conflicts > 0 {
		slog.Info("conflicts detected",
			"total", stats.conflicts,
			"renamed", stats.filesRenamed,
			"skipped", stats.filesSkipped,
			"overwritten", stats.filesOverwritten)
	}
	slog.Info("cleanup completed",
		"folders_deleted", stats.foldersDeleted,
		"target_folder", cfg.TargetFolder)

	if cfg.DryRun {
		slog.Info("DRY RUN completed - no files were actually moved")
	}

	return nil
}
