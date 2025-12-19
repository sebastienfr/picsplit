package handler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// Movie extensions
	extMOV = ".mov"
	extAVI = ".avi"
	extMP4 = ".mp4"

	// Picture extensions
	extJPG  = ".jpg"
	extJPEG = ".jpeg"

	// Raw picture extensions
	extNEF = ".nef"
	extNRW = ".nrw"
	extCRW = ".crw"
	extCR2 = ".cr2"
	extRW2 = ".rw2"

	// Modern image formats
	extHEIC = ".heic" // Apple HEIF
	extHEIF = ".heif" // Standard HEIF
	extWebP = ".webp" // Google WebP
	extAVIF = ".avif" // AV1 Image Format

	// Additional raw formats
	extDNG = ".dng" // Adobe Digital Negative
	extARW = ".arw" // Sony
	extORF = ".orf" // Olympus
	extRAF = ".raf" // Fujifilm

	// Folder configuration
	movFolderName     = "mov"
	rawFolderName     = "raw"
	dateFormatPattern = "2006 - 0102 - 1504"
)

var (
	// movieExtension the list of movie file extensions
	movieExtension = map[string]bool{
		extMOV: true,
		extAVI: true,
		extMP4: true,
	}

	// rawFileExtension the list of raw file extensions
	rawFileExtension = map[string]bool{
		extNEF: true,
		extNRW: true,
		extCRW: true,
		extCR2: true,
		extRW2: true,
		extDNG: true,
		extARW: true,
		extORF: true,
		extRAF: true,
	}

	// jpegExtension the list of JPEG and modern image file extensions
	jpegExtension = map[string]bool{
		extJPG:  true,
		extJPEG: true,
		extHEIC: true,
		extHEIF: true,
		extWebP: true,
		extAVIF: true,
	}

	// split directories cache
	directories map[string]string

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

	// Initialize directories cache
	directories = make(map[string]string)

	// List existing dated directories
	if err := listDirectories(cfg.BasePath); err != nil {
		return fmt.Errorf("failed to list directories: %w", err)
	}

	// Process files in the base directory
	if err := processFiles(cfg); err != nil {
		return fmt.Errorf("failed to process files: %w", err)
	}

	return nil
}

func listDirectories(basedir string) error {
	entries, err := os.ReadDir(basedir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", basedir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Try to parse folder name as a date
		_, err := time.Parse(dateFormatPattern, entry.Name())
		if err != nil {
			logrus.Debugf("ignoring non-date formatted folder: %s", entry.Name())
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logrus.Warnf("failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		logrus.Debugf("found folder: %s, date: %s", entry.Name(), info.ModTime().String())
		directories[entry.Name()] = entry.Name()
	}

	return nil
}

func processFiles(cfg *Config) error {
	entries, err := os.ReadDir(cfg.BasePath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			logrus.Warnf("failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		logrus.Debugf("processing file: %s, date: %s", info.Name(), info.ModTime().String())

		// Process pictures
		if isPicture(info) {
			if err := processPicture(cfg, info); err != nil {
				return fmt.Errorf("failed to process picture %s: %w", info.Name(), err)
			}
			continue
		}

		// Process movies
		if isMovie(info) {
			if err := processMovie(cfg, info); err != nil {
				return fmt.Errorf("failed to process movie %s: %w", info.Name(), err)
			}
			continue
		}

		logrus.Debugf("%s has unknown extension, skipping", info.Name())
	}

	return nil
}

// processPicture handles the processing of picture files
func processPicture(cfg *Config, fi os.FileInfo) error {
	logrus.Debugf("processing picture: %s, date: %s", fi.Name(), fi.ModTime().String())

	datedFolder, err := findOrCreateDatedFolder(cfg.BasePath, fi, cfg.Delta, cfg.DryRun)
	if err != nil {
		return err
	}

	destDir := datedFolder

	// Special handling for RAW files
	if isRaw(fi) && !cfg.NoMoveRaw {
		baseRawDir := filepath.Join(cfg.BasePath, datedFolder)
		rawDir, err := findOrCreateFolder(baseRawDir, rawFolderName, cfg.DryRun)
		if err != nil {
			return err
		}
		destDir = filepath.Join(datedFolder, rawDir)
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.DryRun)
}

// processMovie handles the processing of movie files
func processMovie(cfg *Config, fi os.FileInfo) error {
	logrus.Debugf("processing movie: %s, date: %s", fi.Name(), fi.ModTime().String())

	datedFolder, err := findOrCreateDatedFolder(cfg.BasePath, fi, cfg.Delta, cfg.DryRun)
	if err != nil {
		return err
	}

	destDir := datedFolder

	// Move to separate mov folder if needed
	if !cfg.NoMoveMovie {
		baseMovieDir := filepath.Join(cfg.BasePath, datedFolder)
		movieDir, err := findOrCreateFolder(baseMovieDir, movFolderName, cfg.DryRun)
		if err != nil {
			return err
		}
		destDir = filepath.Join(datedFolder, movieDir)
	}

	return moveFile(cfg.BasePath, fi.Name(), destDir, cfg.DryRun)
}

func findOrCreateDatedFolder(basedir string, file os.FileInfo, delta time.Duration, dryRun bool) (string, error) {
	roundedDate := file.ModTime().Round(delta).Format(dateFormatPattern)

	if dryRun {
		return roundedDate, nil
	}

	// Check cache first
	if folder, ok := directories[roundedDate]; ok {
		logrus.Debugf("found suitable folder: %s", roundedDate)
		return folder, nil
	}

	// Create new folder
	dirCreate := filepath.Join(basedir, roundedDate)
	logrus.Debugf("creating suitable folder: %s", roundedDate)

	if err := os.Mkdir(dirCreate, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dirCreate, err)
	}

	fi, err := os.Stat(dirCreate)
	if err != nil {
		return "", fmt.Errorf("failed to stat created directory: %w", err)
	}

	directories[roundedDate] = fi.Name()
	return fi.Name(), nil
}

func findOrCreateFolder(basedir, name string, dryRun bool) (string, error) {
	dirCreate := filepath.Join(basedir, name)

	logrus.Debugf("finding or creating folder: %s", dirCreate)

	if dryRun {
		return name, nil
	}

	fi, err := os.Stat(dirCreate)
	if err != nil {
		if os.IsNotExist(err) {
			// Create the folder
			if err := os.Mkdir(dirCreate, 0755); err != nil {
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
		logrus.Infof("[DRY RUN] would move file: %s -> %s", srcPath, dstPath)
		return nil
	}

	logrus.Infof("moving file: %s -> %s", srcPath, dstPath)

	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move %s to %s: %w", srcPath, dstPath, err)
	}

	return nil
}

func isMovie(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	return movieExtension[ext]
}

func isPicture(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	return jpegExtension[ext] || rawFileExtension[ext]
}

func isRaw(file os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	return rawFileExtension[ext]
}
