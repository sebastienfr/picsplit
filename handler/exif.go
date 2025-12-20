package handler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/abema/go-mp4"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/sirupsen/logrus"
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
func ExtractMetadata(filePath string) (*FileMetadata, error) {
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

	// Déterminer le type de fichier
	if isPicture(info) {
		// Pour les fichiers RAW, chercher le JPG associé
		if isRaw(info) {
			jpegPath, err := findAssociatedJPEG(filePath)
			if err == nil {
				filePath = jpegPath
				logrus.Debugf("using associated JPEG %s for RAW file %s", jpegPath, info.Name())
			}
		}

		// Extraire EXIF
		dateTime, err := extractEXIFDate(filePath)
		if err == nil && isValidDateTime(dateTime) {
			metadata.DateTime = dateTime
			metadata.Source = DateSourceEXIF
			logrus.Debugf("extracted EXIF date for %s: %s", info.Name(), dateTime.Format(time.RFC3339))
		} else {
			logrus.Debugf("failed to extract EXIF date for %s: %v", info.Name(), err)
		}

		// Extraire GPS
		gps, err := extractGPS(filePath)
		if err == nil && gps != nil {
			metadata.GPS = gps
			logrus.Debugf("extracted GPS for %s: %.4f,%.4f", info.Name(), gps.Lat, gps.Lon)
		}
	} else if isMovie(info) {
		// Extraire métadonnées vidéo
		dateTime, err := extractVideoMetadata(filePath)
		if err == nil && isValidDateTime(dateTime) {
			metadata.DateTime = dateTime
			metadata.Source = DateSourceVideoMeta
			logrus.Debugf("extracted video metadata for %s: %s", info.Name(), dateTime.Format(time.RFC3339))
		} else {
			logrus.Debugf("failed to extract video metadata for %s: %v", info.Name(), err)
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

	var foundTime *time.Time

	// Parser le fichier MP4
	_, err = mp4.ReadBoxStructure(f, func(h *mp4.ReadHandle) (interface{}, error) {
		// Chercher le box mvhd (movie header) qui contient creation_time
		box, _, err := h.ReadPayload()
		if err != nil {
			return nil, err
		}

		// Si c'est un mvhd box
		if mvhd, ok := box.(*mp4.Mvhd); ok {
			// Convertir le timestamp MP4 (secondes depuis 1904-01-01) vers time.Time
			// MP4 epoch: 1904-01-01 00:00:00 UTC
			mp4Epoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
			creationTimestamp := mvhd.GetCreationTime()
			// Safe conversion with overflow check
			if creationTimestamp > uint64(1<<63-1) {
				return nil, fmt.Errorf("creation time overflow")
			}
			creationTime := mp4Epoch.Add(time.Duration(int64(creationTimestamp)) * time.Second)
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

// findAssociatedJPEG finds the corresponding JPEG file for a RAW file
// Ex: PHOTO_01.NEF → PHOTO_01.JPG or PHOTO_01.jpeg
func findAssociatedJPEG(rawPath string) (string, error) {
	dir := filepath.Dir(rawPath)
	baseName := strings.TrimSuffix(filepath.Base(rawPath), filepath.Ext(rawPath))

	// Essayer différentes extensions JPEG
	jpegExtensions := []string{".jpg", ".JPG", ".jpeg", ".JPEG"}

	for _, ext := range jpegExtensions {
		jpegPath := filepath.Join(dir, baseName+ext)
		if _, err := os.Stat(jpegPath); err == nil {
			return jpegPath, nil
		}
	}

	return "", fmt.Errorf("no associated JPEG found for %s", filepath.Base(rawPath))
}
