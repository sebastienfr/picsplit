package handler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// fileGroup représente un groupe de fichiers détecté comme un événement
type fileGroup struct {
	folderName string
	firstFile  os.FileInfo
	files      []os.FileInfo
}

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

	// Custom errors
	ErrNotDirectory = errors.New("path is not a directory")
	ErrInvalidDelta = errors.New("delta must be positive")
)

// collectMediaFiles récupère tous les fichiers médias (photos et vidéos)
func collectMediaFiles(basePath string) ([]os.FileInfo, error) {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var mediaFiles []os.FileInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			logrus.Warnf("failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		if isPicture(info) || isMovie(info) {
			mediaFiles = append(mediaFiles, info)
		} else {
			logrus.Debugf("%s has unknown extension, skipping", info.Name())
		}
	}

	return mediaFiles, nil
}

// sortFilesByModTime trie les fichiers par date de modification croissante
func sortFilesByModTime(files []os.FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		// Si les ModTime sont égaux, trier par nom (déterministe)
		if files[i].ModTime().Equal(files[j].ModTime()) {
			return files[i].Name() < files[j].Name()
		}
		return files[i].ModTime().Before(files[j].ModTime())
	})
}

// groupFilesByGaps regroupe les fichiers par gaps temporels
// Un nouveau groupe démarre quand gap > delta
func groupFilesByGaps(files []os.FileInfo, delta time.Duration) []fileGroup {
	if len(files) == 0 {
		return nil
	}

	var groups []fileGroup

	currentGroup := fileGroup{
		files:     []os.FileInfo{files[0]},
		firstFile: files[0],
	}

	for i := 1; i < len(files); i++ {
		gap := files[i].ModTime().Sub(files[i-1].ModTime())

		if gap <= delta {
			// Gap acceptable, continuer le groupe
			currentGroup.files = append(currentGroup.files, files[i])
		} else {
			// Gap trop grand, finaliser groupe actuel
			currentGroup.folderName = currentGroup.firstFile.ModTime().Format(dateFormatPattern)
			groups = append(groups, currentGroup)

			// Démarrer nouveau groupe
			currentGroup = fileGroup{
				files:     []os.FileInfo{files[i]},
				firstFile: files[i],
			}
		}
	}

	// Ajouter dernier groupe
	currentGroup.folderName = currentGroup.firstFile.ModTime().Format(dateFormatPattern)
	groups = append(groups, currentGroup)

	return groups
}

// processGroup traite tous les fichiers d'un groupe
func processGroup(cfg *Config, group fileGroup) error {
	// Créer dossier principal (si pas dry-run)
	if !cfg.DryRun {
		groupDir := filepath.Join(cfg.BasePath, group.folderName)
		if err := os.MkdirAll(groupDir, 0755); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", groupDir, err)
		}
	}

	// Traiter chaque fichier
	for _, file := range group.files {
		if isPicture(file) {
			if err := processPicture(cfg, file, group.folderName); err != nil {
				return err
			}
		} else if isMovie(file) {
			if err := processMovie(cfg, file, group.folderName); err != nil {
				return err
			}
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

	// 1. Collecter fichiers média
	mediaFiles, err := collectMediaFiles(cfg.BasePath)
	if err != nil {
		return fmt.Errorf("failed to collect media files: %w", err)
	}

	if len(mediaFiles) == 0 {
		logrus.Info("no media files found")
		return nil
	}

	logrus.Infof("found %d media files", len(mediaFiles))

	// 2. Trier chronologiquement
	sortFilesByModTime(mediaFiles)

	// 3. Grouper par gaps
	groups := groupFilesByGaps(mediaFiles, cfg.Delta)
	logrus.Infof("detected %d event groups (delta: %v)", len(groups), cfg.Delta)

	// 4. Traiter chaque groupe
	for i, group := range groups {
		logrus.Infof("[%d/%d] processing group %s (%d files)",
			i+1, len(groups), group.folderName, len(group.files))

		if err := processGroup(cfg, group); err != nil {
			return fmt.Errorf("failed to process group %s: %w", group.folderName, err)
		}
	}

	return nil
}

// processPicture handles the processing of picture files
func processPicture(cfg *Config, fi os.FileInfo, datedFolder string) error {
	logrus.Debugf("processing picture: %s → %s", fi.Name(), datedFolder)

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
func processMovie(cfg *Config, fi os.FileInfo, datedFolder string) error {
	logrus.Debugf("processing movie: %s → %s", fi.Name(), datedFolder)

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
