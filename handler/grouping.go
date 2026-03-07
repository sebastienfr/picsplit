package handler

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// fileGroup represents a group of files detected as an event
type fileGroup struct {
	folderName string
	firstFile  FileMetadata
	files      []FileMetadata
}

// collectMediaFilesWithMetadata retrieves all media files with their EXIF/video metadata
func collectMediaFilesWithMetadata(cfg *Config, ctx *executionContext) ([]FileMetadata, error) {
	entries, err := os.ReadDir(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	mediaFiles := make([]FileMetadata, 0, len(entries))
	var exifFailCount int

	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			slog.Warn("failed to get file info", "file", entry.Name(), "error", err)
			continue
		}

		// Use context to check if file is a media file
		if !ctx.isPhoto(info.Name()) && !ctx.isMovie(info.Name()) {
			slog.Debug("skipping file with unknown extension", "file", info.Name())
			continue
		}

		filePath := filepath.Join(cfg.BasePath, info.Name())

		// Extract metadata (EXIF/video)
		var metadata *FileMetadata
		if cfg.UseEXIF {
			metadata, err = ExtractMetadata(ctx, filePath, info)
			if err != nil || metadata.Source == DateSourceModTime {
				slog.Debug("failed to extract metadata, using ModTime", "file", info.Name())
				exifFailCount++
			}
		} else {
			// Mode without EXIF: use ModTime directly
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

	// Selective fallback: files without EXIF use ModTime, others keep their extracted metadata
	// GPS coordinates are preserved even when DateTime falls back to ModTime
	if cfg.UseEXIF && exifFailCount > 0 {
		slog.Warn("files used ModTime fallback",
			"count", exifFailCount,
			"reason", "EXIF metadata unavailable or corrupted")
		// Each file keeps its individually extracted metadata (DateTime and GPS)
		// No global reset - this allows GPS clustering to work with mixed file sets
	}

	return mediaFiles, nil
}

// sortFilesByDateTime sorts files by ascending date/time (EXIF or ModTime)
func sortFilesByDateTime(files []FileMetadata) {
	sort.Slice(files, func(i, j int) bool {
		// If DateTime are equal, sort by name (deterministic)
		if files[i].DateTime.Equal(files[j].DateTime) {
			return files[i].FileInfo.Name() < files[j].FileInfo.Name()
		}
		return files[i].DateTime.Before(files[j].DateTime)
	})
}

// groupFilesByGaps groups files by time gaps
// A new group starts when gap > delta
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
			// Acceptable gap, continue the group
			currentGroup.files = append(currentGroup.files, files[i])
		} else {
			// Gap too large, finalize current group
			slog.Debug("gap exceeds delta, creating new group",
				"prev_file", files[i-1].FileInfo.Name(),
				"prev_time", files[i-1].DateTime,
				"curr_file", files[i].FileInfo.Name(),
				"curr_time", files[i].DateTime,
				"gap", gap,
				"delta", delta)
			currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
			groups = append(groups, currentGroup)

			// Start new group
			currentGroup = fileGroup{
				files:     []FileMetadata{files[i]},
				firstFile: files[i],
			}
		}
	}

	// Add last group
	currentGroup.folderName = currentGroup.firstFile.DateTime.Format(dateFormatPattern)
	groups = append(groups, currentGroup)

	return groups
}
