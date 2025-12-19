package handler

import (
	"errors"
	"os"
	"time"
)

// Config holds all configuration for the split operation
type Config struct {
	BasePath    string
	Delta       time.Duration
	NoMoveMovie bool
	NoMoveRaw   bool
	DryRun      bool
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.BasePath == "" {
		return errors.New("base path cannot be empty")
	}

	if c.Delta <= 0 {
		return ErrInvalidDelta
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
		BasePath:    basePath,
		Delta:       30 * time.Minute,
		NoMoveMovie: false,
		NoMoveRaw:   false,
		DryRun:      false,
	}
}
