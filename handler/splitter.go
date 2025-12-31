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
	// Folder configuration
	movFolderName     = "mov"
	rawFolderName     = "raw"
	orphanFolderName  = "orphan"
	dateFormatPattern = "2006 - 0102 - 1504"
)

// fileGroup représente un groupe de fichiers détecté comme un événement
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

// collectMediaFilesWithMetadata récupère tous les fichiers médias avec leurs métadonnées EXIF/vidéo
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
			logrus.Warnf("failed to get info for %s: %v", entry.Name(), err)
			continue
		}

		// Use context to check if file is a media file
		if !ctx.isPhoto(info.Name()) && !ctx.isMovie(info.Name()) {
			logrus.Debugf("%s has unknown extension, skipping", info.Name())
			continue
		}

		filePath := filepath.Join(cfg.BasePath, info.Name())

		// Extraire métadonnées (EXIF/vidéo)
		var metadata *FileMetadata
		if cfg.UseEXIF {
			metadata, err = ExtractMetadata(ctx, filePath)
			if err != nil || metadata.Source == DateSourceModTime {
				logrus.Debugf("failed to extract metadata for %s, using ModTime", info.Name())
				exifFailCount++
			}
		} else {
			// Mode sans EXIF : utiliser directement ModTime
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

	// Mode strict : si au moins un fichier sans EXIF valide → fallback tous sur ModTime
	if cfg.UseEXIF && exifFailCount > 0 {
		logrus.Warnf("EXIF validation failed for %d/%d files, using file modification times for all files",
			exifFailCount, len(mediaFiles))

		for i := range mediaFiles {
			mediaFiles[i].DateTime = mediaFiles[i].FileInfo.ModTime()
			mediaFiles[i].Source = DateSourceModTime
			mediaFiles[i].GPS = nil
		}
	}

	return mediaFiles, nil
}

// sortFilesByDateTime trie les fichiers par date/heure croissante (EXIF ou ModTime)
func sortFilesByDateTime(files []FileMetadata) {
	sort.Slice(files, func(i, j int) bool {
		// Si les DateTime sont égaux, trier par nom (déterministe)
		if files[i].DateTime.Equal(files[j].DateTime) {
			return files[i].FileInfo.Name() < files[j].FileInfo.Name()
		}
		return files[i].DateTime.Before(files[j].DateTime)
	})
}

// groupFilesByGaps regroupe les fichiers par gaps temporels
// Un nouveau groupe démarre quand gap > delta
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
			// Gap acceptable, continuer le groupe
			currentGroup.files = append(currentGroup.files, files[i])
		} else {
			// Gap trop grand, finaliser groupe actuel
			currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
			groups = append(groups, currentGroup)

			// Démarrer nouveau groupe
			currentGroup = fileGroup{
				files:     []FileMetadata{files[i]},
				firstFile: files[i],
			}
		}
	}

	// Ajouter dernier groupe
	currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
	groups = append(groups, currentGroup)

	return groups
}

// processGroup traite tous les fichiers d'un groupe
func processGroup(cfg *Config, ctx *executionContext, group fileGroup) error {
	// Créer dossier principal (si pas dry-run)
	if !cfg.DryRun {
		groupDir := filepath.Join(cfg.BasePath, group.folderName)
		if err := os.MkdirAll(groupDir, permDirectory); err != nil {
			return fmt.Errorf("failed to create folder %s: %w", groupDir, err)
		}
	}

	// Traiter chaque fichier
	for _, file := range group.files {
		fileName := file.FileInfo.Name()
		if ctx.isPhoto(fileName) {
			if err := processPicture(cfg, ctx, file.FileInfo, group.folderName); err != nil {
				return err
			}
		} else if ctx.isMovie(fileName) {
			if err := processMovie(cfg, file.FileInfo, group.folderName); err != nil {
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

	// Create execution context with custom extensions
	ctx, err := newExecutionContext(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize extension context: %w", err)
	}

	// 1. Collecter fichiers média avec métadonnées
	mediaFiles, err := collectMediaFilesWithMetadata(cfg, ctx)
	if err != nil {
		return fmt.Errorf("failed to collect media files: %w", err)
	}

	if len(mediaFiles) == 0 {
		logrus.Info("no media files found")
		return nil
	}

	logrus.Infof("found %d media files", len(mediaFiles))

	var groups []fileGroup

	// 2. GPS clustering mode ou mode temporel classique
	if cfg.UseGPS {
		// GPS clustering: location FIRST, then time within each location
		locationClusters, filesWithoutGPS := ClusterByLocation(mediaFiles, cfg.GPSRadius)

		logrus.Infof("GPS clustering: %d location clusters, %d files without GPS",
			len(locationClusters), len(filesWithoutGPS))

		// Traiter chaque cluster de localisation
		for _, cluster := range locationClusters {
			locationName := FormatLocationName(cluster.Centroid)
			logrus.Debugf("processing location cluster: %s (%d files)", locationName, len(cluster.Files))

			// Grouper par temps dans cette localisation
			timeGroups := GroupLocationByTime(cluster, cfg.Delta)
			logrus.Debugf("location %s split into %d time groups", locationName, len(timeGroups))

			// Créer fileGroup pour chaque groupe temporel
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

		// Traiter fichiers sans GPS
		if len(filesWithoutGPS) > 0 {
			// Trier et grouper par temps
			sortFilesByDateTime(filesWithoutGPS)
			noGPSGroups := groupFilesByGaps(filesWithoutGPS, cfg.Delta)

			// Si des clusters de localisation existent, créer sous-dossier "NoLocation"
			// Sinon, mettre directement à la racine (pas de nécessité de ségrégation)
			if len(locationClusters) > 0 {
				logrus.Infof("processing %d files without GPS in '%s' folder", len(filesWithoutGPS), GetNoLocationFolderName())
				for _, noGPSGroup := range noGPSGroups {
					folderName := filepath.Join(GetNoLocationFolderName(), noGPSGroup.folderName)
					groups = append(groups, fileGroup{
						folderName: folderName,
						firstFile:  noGPSGroup.firstFile,
						files:      noGPSGroup.files,
					})
				}
			} else {
				logrus.Infof("processing %d files without GPS at root (no location clusters)", len(filesWithoutGPS))
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
		// Mode temporel classique (backward compatible)
		// 2. Trier chronologiquement
		sortFilesByDateTime(mediaFiles)

		// 3. Grouper par gaps
		groups = groupFilesByGaps(mediaFiles, cfg.Delta)
	}

	logrus.Infof("detected %d event groups (delta: %v)", len(groups), cfg.Delta)

	// 4. Traiter chaque groupe
	for i, group := range groups {
		logrus.Infof("[%d/%d] processing group %s (%d files)",
			i+1, len(groups), group.folderName, len(group.files))

		if err := processGroup(cfg, ctx, group); err != nil {
			return fmt.Errorf("failed to process group %s: %w", group.folderName, err)
		}
	}

	return nil
}

// processPicture handles the processing of picture files
func processPicture(cfg *Config, ctx *executionContext, fi os.FileInfo, datedFolder string) error {
	logrus.Debugf("processing picture: %s → %s", fi.Name(), datedFolder)

	destDir := datedFolder

	// Special handling for RAW files
	if ctx.isRaw(fi.Name()) && !cfg.NoMoveRaw {
		baseRawDir := filepath.Join(cfg.BasePath, datedFolder)

		// Déterminer si RAW va dans raw/ ou orphan/
		targetFolder := rawFolderName

		if cfg.SeparateOrphanRaw {
			// Vérifier si RAW a un JPEG/HEIC associé
			// Chercher dans la source (basePath) ET dans la destination (datedFolder)
			// car le JPEG peut avoir déjà été déplacé
			rawFilePath := filepath.Join(cfg.BasePath, fi.Name())
			destFolder := filepath.Join(cfg.BasePath, datedFolder)
			if !isRawPaired(rawFilePath, cfg.BasePath, destFolder) {
				targetFolder = orphanFolderName
				logrus.Debugf("orphan RAW (no JPEG/HEIC): %s → %s", fi.Name(), orphanFolderName)
			}
		}

		rawDir, err := findOrCreateFolder(baseRawDir, targetFolder, cfg.DryRun)
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
		logrus.Infof("[DRY RUN] would move file: %s -> %s", srcPath, dstPath)
		return nil
	}

	logrus.Infof("moving file: %s -> %s", srcPath, dstPath)

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

	// Extensions à chercher (JPEG et HEIC pour iPhone)
	photoExtensions := []string{".jpg", ".JPG", ".jpeg", ".JPEG", ".heic", ".HEIC"}

	// 1. Chercher dans le dossier source (basePath)
	for _, ext := range photoExtensions {
		photoPath := filepath.Join(basePath, baseName+ext)
		if _, err := os.Stat(photoPath); err == nil {
			logrus.Debugf("found paired photo in source: %s for RAW %s", photoPath, filepath.Base(rawPath))
			return true
		}
	}

	// 2. Chercher dans le dossier de destination (JPEG déjà déplacé)
	if destFolder != "" {
		for _, ext := range photoExtensions {
			photoPath := filepath.Join(destFolder, baseName+ext)
			if _, err := os.Stat(photoPath); err == nil {
				logrus.Debugf("found paired photo in destination: %s for RAW %s", photoPath, filepath.Base(rawPath))
				return true
			}
		}
	}

	return false // Orphelin
}
