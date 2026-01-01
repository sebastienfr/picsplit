package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abema/go-mp4"
	"github.com/rwcarlsen/goexif/exif"
)

// DateSource indique l'origine de la date extraite
type DateSource int

const (
	// DateSourceModTime indique que la date provient du système de fichiers
	DateSourceModTime DateSource = iota
	// DateSourceEXIF indique que la date provient des métadonnées EXIF
	DateSourceEXIF
	// DateSourceVideoMeta indique que la date provient des métadonnées vidéo
	DateSourceVideoMeta
)

const (
	minValidYear  = 1990
	maxFutureDays = 1 // tolérance pour décalage d'horloge
)

const (
	dateSourceModTimeStr   = "ModTime"
	dateSourceEXIFStr      = "EXIF"
	dateSourceVideoMetaStr = "VideoMeta"
)

// String retourne une représentation textuelle de la source de date
func (ds DateSource) String() string {
	switch ds {
	case DateSourceEXIF:
		return dateSourceEXIFStr
	case DateSourceVideoMeta:
		return dateSourceVideoMetaStr
	default:
		return dateSourceModTimeStr
	}
}

// GPSCoord représente des coordonnées GPS
type GPSCoord struct {
	Lat float64
	Lon float64
}

// FileMetadata contient toutes les métadonnées extraites d'un fichier
type FileMetadata struct {
	FileInfo os.FileInfo
	DateTime time.Time
	GPS      *GPSCoord
	Source   DateSource
}

// ExtractMetadata extrait toutes les métadonnées d'un fichier (date et GPS si disponible)
// Utilise le contexte d'exécution pour respecter les extensions personnalisées
func ExtractMetadata(ctx *executionContext, filePath string) (*FileMetadata, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	metadata := &FileMetadata{
		FileInfo: info,
		DateTime: info.ModTime(),
		GPS:      nil,
		Source:   DateSourceModTime,
	}

	// Déterminer le type de fichier en utilisant le contexte
	if ctx.isPhoto(info.Name()) {
		// Pour les fichiers RAW, chercher le JPG associé
		if ctx.isRaw(info.Name()) {
			jpegPath, err := findAssociatedJPEG(filePath)
			if err == nil {
				filePath = jpegPath
				slog.Debug("using associated JPEG for RAW file", "jpeg", jpegPath, "raw", info.Name())
			}
		}

		// Extraire EXIF
		dateTime, err := extractEXIFDate(filePath)
		if err == nil && isValidDateTime(dateTime) {
			metadata.DateTime = dateTime
			metadata.Source = DateSourceEXIF
			slog.Debug("extracted EXIF date", "file", info.Name(), "date", dateTime.Format(time.RFC3339))
		} else {
			slog.Debug("failed to extract EXIF date", "file", info.Name(), "error", err)
		}

		// Extraire GPS
		gps, err := extractGPS(filePath)
		if err == nil && gps != nil {
			metadata.GPS = gps
			slog.Debug("extracted GPS coordinates", "file", info.Name(), "lat", gps.Lat, "lon", gps.Lon)
		}
	} else if ctx.isMovie(info.Name()) {
		// Extraire métadonnées vidéo
		dateTime, err := extractVideoMetadata(filePath)
		if err == nil && isValidDateTime(dateTime) {
			metadata.DateTime = dateTime
			metadata.Source = DateSourceVideoMeta
			slog.Debug("extracted video metadata", "file", info.Name(), "date", dateTime.Format(time.RFC3339))
		} else {
			slog.Debug("failed to extract video metadata", "file", info.Name(), "error", err)
		}
	}

	return metadata, nil
}

// extractEXIFDate extrait la date DateTimeOriginal d'une photo
func extractEXIFDate(filePath string) (time.Time, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to decode EXIF: %w", err)
	}

	// Chercher DateTimeOriginal (préféré) ou DateTime
	dateTime, err := x.DateTime()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get DateTime: %w", err)
	}

	return dateTime, nil
}

// extractVideoMetadata extrait la date de création d'une vidéo MP4/MOV
func extractVideoMetadata(filePath string) (time.Time, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to open video: %w", err)
	}
	defer f.Close()

	// Get file ModTime for comparison
	fileInfo, err := f.Stat()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to stat video: %w", err)
	}
	modTime := fileInfo.ModTime()

	var foundTime *time.Time

	// Parser le fichier MP4
	_, err = mp4.ReadBoxStructure(f, func(h *mp4.ReadHandle) (interface{}, error) {
		// Expand container boxes (moov, trak, etc.) to read their children
		if h.BoxInfo.Type == mp4.BoxTypeMoov() || h.BoxInfo.Type == mp4.BoxTypeTrak() {
			// Expand to read child boxes
			return h.Expand()
		}

		// Chercher le box mvhd (movie header) qui contient creation_time
		box, _, err := h.ReadPayload()
		if err != nil {
			return nil, err
		}

		// Si c'est un mvhd box
		if mvhd, ok := box.(*mp4.Mvhd); ok {
			// Convertir le timestamp MP4 (secondes depuis 1904-01-01) vers time.Time
			// MP4 spec: timestamps should be UTC
			mp4Epoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
			creationTimestamp := mvhd.GetCreationTime()
			// Safe conversion with overflow check
			if creationTimestamp > uint64(1<<63-1) {
				return nil, fmt.Errorf("creation time overflow")
			}
			creationTimeUTC := mp4Epoch.Add(time.Duration(int64(creationTimestamp)) * time.Second)

			// Some cameras (Nikon, etc.) incorrectly store local time in the UTC field
			// Detect this by comparing wall clock times (HH:MM:SS) between MP4 UTC and ModTime local
			// If they match, camera stored local time as UTC (common bug)
			mp4Hour, mp4Min, mp4Sec := creationTimeUTC.Clock()
			modHour, modMin, modSec := modTime.Clock()

			// Calculate absolute differences
			hourDiff := mp4Hour - modHour
			if hourDiff < 0 {
				hourDiff = -hourDiff
			}
			minDiff := mp4Min - modMin
			if minDiff < 0 {
				minDiff = -minDiff
			}
			secDiff := mp4Sec - modSec
			if secDiff < 0 {
				secDiff = -secDiff
			}
			wallClockDiffSeconds := hourDiff*3600 + minDiff*60 + secDiff

			// If wall clock times are within 5 seconds, camera stored local as UTC
			var creationTime time.Time
			if wallClockDiffSeconds < 5 {
				// Camera stored local time as UTC, reinterpret by changing timezone
				// The time 21:45:03Z should be interpreted as 21:45:03 Local (not converted)
				year, month, day := creationTimeUTC.Date()
				hour, min, sec := creationTimeUTC.Clock()
				creationTime = time.Date(year, month, day, hour, min, sec, 0, time.Local)

				slog.Debug("MP4 timestamp appears to be local time (stored as UTC)",
					"file", filepath.Base(filePath),
					"mp4_utc_clock", fmt.Sprintf("%02d:%02d:%02d", mp4Hour, mp4Min, mp4Sec),
					"mod_local_clock", fmt.Sprintf("%02d:%02d:%02d", modHour, modMin, modSec),
					"corrected_time", creationTime,
					"wall_diff_sec", wallClockDiffSeconds)
			} else {
				// Proper UTC timestamp, keep as is
				slog.Debug("MP4 timestamp is proper UTC",
					"file", filepath.Base(filePath),
					"mp4_utc_clock", fmt.Sprintf("%02d:%02d:%02d", mp4Hour, mp4Min, mp4Sec),
					"mod_local_clock", fmt.Sprintf("%02d:%02d:%02d", modHour, modMin, modSec),
					"wall_diff_sec", wallClockDiffSeconds)
				creationTime = creationTimeUTC
			}

			foundTime = &creationTime
		}

		return nil, nil
	})

	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse MP4: %w", err)
	}

	if foundTime == nil {
		return time.Time{}, fmt.Errorf("no creation time found in video metadata")
	}

	return *foundTime, nil
}

// extractGPS extrait les coordonnées GPS de l'EXIF
func extractGPS(filePath string) (*GPSCoord, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	x, err := exif.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode EXIF: %w", err)
	}

	lat, lon, err := x.LatLong()
	if err != nil {
		return nil, fmt.Errorf("failed to get GPS coordinates: %w", err)
	}

	// Vérifier que les coordonnées ne sont pas nulles (valeur par défaut)
	if lat == 0 && lon == 0 {
		return nil, fmt.Errorf("GPS coordinates are zero")
	}

	return &GPSCoord{
		Lat: lat,
		Lon: lon,
	}, nil
}

// isValidDateTime vérifie que la date est cohérente
func isValidDateTime(t time.Time) bool {
	// Vérifier année minimum
	if t.Year() < minValidYear {
		return false
	}

	// Vérifier que ce n'est pas trop dans le futur
	maxFuture := time.Now().AddDate(0, 0, maxFutureDays)
	return !t.After(maxFuture)
}

// findAssociatedJPEG finds the corresponding JPEG or HEIC file for a RAW file
// Ex: PHOTO_01.NEF → PHOTO_01.JPG, PHOTO_01.jpeg, PHOTO_01.heic, or PHOTO_01.HEIC
func findAssociatedJPEG(rawPath string) (string, error) {
	dir := filepath.Dir(rawPath)
	baseName := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))

	// Essayer différentes extensions JPEG et HEIC (iPhone shoot RAW+HEIC)
	photoExtensions := []string{".jpg", ".JPG", ".jpeg", ".JPEG", ".heic", ".HEIC"}

	for _, ext := range photoExtensions {
		photoPath := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(photoPath); err == nil {
			return photoPath, nil
		}
	}

	return "", fmt.Errorf("no associated JPEG/HEIC found for %s", filepath.Base(rawPath))
}
