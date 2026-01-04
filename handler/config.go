package handler

import (
	"errors"
	"os"
	"time"
)

const (
	defaultGPSRadiusMeters = 2000.0 // Rayon par défaut pour le clustering GPS : 2km
)

// ExecutionMode defines the mode of execution
type ExecutionMode string

const (
	ModeValidate ExecutionMode = "validate" // Fast validation (scan + count)
	ModeDryRun   ExecutionMode = "dryrun"   // Full simulation (scan + EXIF + simulate)
	ModeRun      ExecutionMode = "run"      // Real execution (default)
)

// Config holds all configuration for the split operation
type Config struct {
	BasePath    string
	Delta       time.Duration
	NoMoveMovie bool
	NoMoveRaw   bool
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

	// Logging configuration (v2.7.0+)
	LogLevel  string // Log level: debug, info, warn, error
	LogFormat string // Log format: text, json

	// Error handling & execution mode (v2.8.0+)
	ContinueOnError  bool          // Continue processing even if errors occur (collect all errors instead of stopping)
	Mode             ExecutionMode // Execution mode: validate (fast check), dryrun (simulate), run (execute - default)
	CleanupEmptyDirs bool          // Remove empty directories after processing
	CleanupIgnore    []string      // Additional files to ignore when checking if directory is empty (beyond .DS_Store, Thumbs.db, etc.)
	Force            bool          // Skip confirmation prompts (cleanup, merge conflicts, etc.)
	DetectDuplicates bool          // Detect duplicate files via SHA256 hash (v2.8.0+)
	SkipDuplicates   bool          // Skip duplicate files automatically (requires DetectDuplicates) (v2.8.0+)
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

	if c.SkipDuplicates && !c.DetectDuplicates {
		return errors.New("--skip-duplicates requires --detect-duplicates")
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
		UseEXIF:           true,
		UseGPS:            false,                  // GPS clustering désactivé par défaut (opt-in)
		GPSRadius:         defaultGPSRadiusMeters, // 2000m = 2km
		SeparateOrphanRaw: true,                   // Activé par défaut (v2.6.0+)
		ContinueOnError:   false,                  // Stop au premier échec par défaut (v2.8.0+)
		Mode:              ModeRun,                // Execution réelle par défaut (v2.8.0+)
		CleanupEmptyDirs:  false,                  // Désactivé par défaut (v2.8.0+)
		Force:             false,                  // Demander confirmation par défaut (v2.8.0+)
		DetectDuplicates:  false,                  // Détection désactivée par défaut (v2.8.0+)
		SkipDuplicates:    false,                  // Skip désactivé par défaut (v2.8.0+)
	}
}
