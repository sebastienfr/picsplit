package handler

import (
	"errors"
	"os"
	"time"
)

const (
	defaultGPSRadiusMeters = 2000.0 // Rayon par défaut pour le clustering GPS : 2km
)

// Config holds all configuration for the split operation
type Config struct {
	BasePath    string
	Delta       time.Duration
	NoMoveMovie bool
	NoMoveRaw   bool
	DryRun      bool
	UseEXIF     bool
	UseGPS      bool
	GPSRadius   float64 // Rayon en mètres pour le clustering GPS

	// Custom extensions (v2.5.0+)
	// These are ADDITIVE to the default extensions
	CustomPhotoExts []string // Additional photo extensions (e.g., ["png", "gif", "bmp"])
	CustomVideoExts []string // Additional video extensions (e.g., ["mkv", "mpeg", "wmv"])
	CustomRawExts   []string // Additional RAW extensions (e.g., ["rwx", "srw", "3fr"])

	// Orphan RAW separation (v2.6.0+)
	SeparateOrphanRaw bool // Separate unpaired RAW files (without JPEG/HEIC) to orphan/ folder
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BasePath == "" {
		return errors.New("base path cannot be empty")
	}

	if c.Delta <= 0 {
		return ErrInvalidDelta
	}

	if c.UseGPS && c.GPSRadius <= 0 {
		return errors.New("GPS radius must be positive when GPS clustering is enabled")
	}

	// Check if path exists and is a directory
	fi, err := os.Stat(c.BasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("path does not exist")
		}
		return err
	}

	if !fi.IsDir() {
		return ErrNotDirectory
	}

	return nil
}

// DefaultConfig returns a configuration with default values
func DefaultConfig(basePath string) *Config {
	return &Config{
		BasePath:          basePath,
		Delta:             30 * time.Minute,
		NoMoveMovie:       false,
		NoMoveRaw:         false,
		DryRun:            false,
		UseEXIF:           true,
		UseGPS:            false,                  // GPS clustering désactivé par défaut (opt-in)
		GPSRadius:         defaultGPSRadiusMeters, // 2000m = 2km
		SeparateOrphanRaw: true,                   // Activé par défaut (v2.6.0+)
	}
}
