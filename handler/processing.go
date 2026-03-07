package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// handleDuplicate checks if a file is a duplicate and handles it according to config.
// Returns true if the file should be skipped (was a duplicate that was handled).
func handleDuplicate(cfg *Config, detector *DuplicateDetector, stats *ProcessingStats, filePath string, fileName string, fileSize int64) (skip bool, err error) {
	if !cfg.DetectDuplicates {
		return false, nil
	}

	isDup, original, checkErr := detector.Check(filePath, fileSize)
	if checkErr != nil {
		slog.Warn("failed to check duplicate", "file", fileName, "error", checkErr)
		return false, nil
	}

	if !isDup {
		return false, nil
	}

	stats.DuplicatesDetected[filePath] = original

	if cfg.SkipDuplicates {
		slog.Info("skipping duplicate", "file", fileName, "original", filepath.Base(original))
		stats.DuplicatesSkipped++
		return true, nil
	}

	if cfg.MoveDuplicates {
		duplicatesDir := filepath.Join(cfg.BasePath, duplicatesFolderName)
		if cfg.Mode != ModeDryRun {
			if mkErr := os.MkdirAll(duplicatesDir, permDirectory); mkErr != nil {
				slog.Error("failed to create duplicates folder", "error", mkErr)
				if cfg.ContinueOnError {
					stats.AddError(fmt.Errorf("failed to create duplicates folder: %w", mkErr))
					return true, nil
				}
				return false, mkErr
			}
		}

		if mvErr := moveFile(cfg.BasePath, fileName, duplicatesFolderName, cfg.Mode == ModeDryRun); mvErr != nil {
			slog.Error("failed to move duplicate", "file", fileName, "error", mvErr)
			if cfg.ContinueOnError {
				stats.AddError(fmt.Errorf("failed to move duplicate %s: %w", fileName, mvErr))
				return true, nil
			}
			return false, mvErr
		}
		if cfg.Mode == ModeDryRun {
			slog.Info("would move duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
		} else {
			slog.Info("moved duplicate", "file", fileName, "to", duplicatesFolderName+"/", "original", filepath.Base(original))
		}
		stats.DuplicatesSkipped++
		return true, nil
	}

	slog.Warn("duplicate detected (processing anyway)", "file", fileName, "original", filepath.Base(original))
	return false, nil
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

		skip, err := handleDuplicate(cfg, detector, stats, filePath, fileName, file.FileInfo.Size())
		if err != nil {
			return err
		}
		if skip {
			continue
		}

		// Process file normally
		if ctx.isPhoto(fileName) || ctx.isMovie(fileName) {
			if err := processMediaFile(cfg, ctx, file.FileInfo, group.folderName, ""); err != nil {
				if cfg.ContinueOnError {
					stats.AddError(err)
					slog.Error("failed to process file, continuing", "file", fileName, "error", err)
					continue
				}
				return err
			}
			stats.ProcessedFiles++
		}
	}

	return nil
}

func processMediaFile(cfg *Config, ctx *executionContext, fi os.FileInfo, datedFolder string, destinationRoot string) error {
	slog.Debug("processing media file", "file", fi.Name(), "dated_folder", datedFolder, "dest_root", destinationRoot)

	if datedFolder == "" && destinationRoot != "" && cfg.Mode != ModeDryRun {
		destRootPath := filepath.Join(cfg.BasePath, destinationRoot)
		if err := os.MkdirAll(destRootPath, permDirectory); err != nil {
			return fmt.Errorf("failed to create location folder %s: %w", destRootPath, err)
		}
	}

	destPrefix := datedFolder
	if datedFolder == "" {
		destPrefix = destinationRoot
	}

	destDir := destPrefix

	subfolderName, moveToSubfolder := mediaSubfolder(cfg, ctx, fi.Name())
	if moveToSubfolder {
		baseDir := filepath.Join(cfg.BasePath, destPrefix)

		if subfolderName == rawFolderName && cfg.SeparateOrphanRaw {
			rawFilePath := filepath.Join(cfg.BasePath, fi.Name())
			if !isRawPaired(ctx, rawFilePath, cfg.BasePath, baseDir) {
				subfolderName = orphanFolderName
				slog.Debug("orphan RAW file (no JPEG/HEIC)", "file", fi.Name(), "dest", orphanFolderName)
			}
		}

		subDir, err := findOrCreateFolder(baseDir, subfolderName, cfg.Mode == ModeDryRun)
		if err != nil {
			return err
		}
		if destPrefix != "" {
			destDir = filepath.Join(destPrefix, subDir)
		} else {
			destDir = subDir
		}
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.Mode == ModeDryRun)
}

func mediaSubfolder(cfg *Config, ctx *executionContext, fileName string) (string, bool) {
	if ctx.isRaw(fileName) && !cfg.NoMoveRaw {
		return rawFolderName, true
	}
	if ctx.isMovie(fileName) && !cfg.NoMoveMovie {
		return movFolderName, true
	}
	return "", false
}

func findOrCreateFolder(basedir, name string, dryRun bool) (string, error) {
	if dryRun {
		return name, nil
	}

	dirCreate := filepath.Join(basedir, name)
	if err := os.MkdirAll(dirCreate, permDirectory); err != nil {
		return "", fmt.Errorf("failed to create folder %s: %w", dirCreate, err)
	}

	return name, nil
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
func isRawPaired(ctx *executionContext, rawPath string, basePath string, destFolder string) bool {
	baseName := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))

	photoExtensions := ctx.photoSearchExtensions()

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

	return false // Orphan
}
