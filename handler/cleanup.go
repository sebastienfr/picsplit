package handler

import (
	"bufio"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// List of system directories to protect
var protectedDirs = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
}

// List of system files to ignore (don't count as "content")
var ignoredFiles = []string{
	".DS_Store",   // macOS
	"Thumbs.db",   // Windows
	"desktop.ini", // Windows
	"._.DS_Store", // macOS AppleDouble
}

// CleanupResult contains the cleanup results
type CleanupResult struct {
	RemovedDirs []string
	FailedDirs  map[string]error
}

// CleanupEmptyDirs recursively removes empty directories
// using a bottom-up traversal (post-order traversal).
//
// Parameters:
//   - rootPath: The root path from which to search for empty directories
//   - mode: The execution mode (ModeValidate, ModeDryRun, ModeRun)
//   - force: If true, removes without confirmation. If false, asks for confirmation in Run mode
//   - customIgnoredFiles: List of additional files to ignore (in addition to default system files)
//
// Returns:
//   - CleanupResult containing the list of removed directories and errors
//   - error if a fatal error occurs
func CleanupEmptyDirs(rootPath string, mode ExecutionMode, force bool, customIgnoredFiles []string) (*CleanupResult, error) {
	result := &CleanupResult{
		RemovedDirs: []string{},
		FailedDirs:  make(map[string]error),
	}

	// Validate mode does not perform cleanup
	if mode == ModeValidate {
		slog.Debug("skipping cleanup in validate mode")
		return result, nil
	}

	// Combine default ignored files with user's custom list
	allIgnoredFiles := append([]string{}, ignoredFiles...)
	allIgnoredFiles = append(allIgnoredFiles, customIgnoredFiles...)

	if len(customIgnoredFiles) > 0 {
		slog.Debug("using custom ignored files for cleanup", "files", customIgnoredFiles)
	}

	// Make multiple passes to remove nested empty directories
	// Each pass can make parents empty, so continue until no more changes
	maxPasses := 100 // Protection against infinite loops
	for pass := 0; pass < maxPasses; pass++ {
		emptyDirs := []string{}

		// Collect empty directories
		err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("failed to access path during cleanup", "path", path, "error", err)
				return nil // Continue walk
			}

			// Skip files
			if !d.IsDir() {
				return nil
			}

			// Skip root path
			if path == rootPath {
				return nil
			}

			// Skip protected directories
			if isProtectedDir(path) {
				slog.Debug("skipping protected directory", "path", path)
				return fs.SkipDir
			}

			// Check if empty (considering ignored files)
			empty, err := isDirEmptyWithIgnored(path, allIgnoredFiles)
			if err != nil {
				slog.Warn("failed to check if directory is empty", "path", path, "error", err)
				result.FailedDirs[path] = err
				return nil // Continue walk
			}

			if empty {
				emptyDirs = append(emptyDirs, path)
			}

			return nil
		})

		if err != nil {
			return result, fmt.Errorf("failed to walk directory tree: %w", err)
		}

		// If no empty directory found, we're done
		if len(emptyDirs) == 0 {
			break
		}

		// In Run mode without force, ask confirmation on first pass
		if mode == ModeRun && !force && pass == 0 {
			if !askConfirmation(emptyDirs) {
				slog.Info("cleanup cancelled by user")
				return result, nil
			}
		}

		// Iterate empty directories in reverse order (bottom-up)
		// to remove subdirectories before parents
		removedInPass := 0
		for i := len(emptyDirs) - 1; i >= 0; i-- {
			dir := emptyDirs[i]

			// Re-check if empty (may have changed during this pass)
			empty, err := isDirEmptyWithIgnored(dir, allIgnoredFiles)
			if err != nil {
				slog.Warn("failed to re-check if directory is empty", "path", dir, "error", err)
				result.FailedDirs[dir] = err
				continue
			}

			if !empty {
				slog.Debug("directory no longer empty, skipping", "path", dir)
				continue
			}

			if mode == ModeDryRun {
				slog.Info("would remove empty directory", "path", dir)
				result.RemovedDirs = append(result.RemovedDirs, dir)
				removedInPass++
			} else {
				// First remove ignored files in directory
				if err := removeIgnoredFiles(dir, allIgnoredFiles); err != nil {
					slog.Warn("failed to remove ignored files", "path", dir, "error", err)
				}

				// Then remove empty directory
				if err := os.Remove(dir); err != nil {
					slog.Warn("failed to remove empty directory", "path", dir, "error", err)
					result.FailedDirs[dir] = err
				} else {
					slog.Info("removed empty directory", "path", dir)
					result.RemovedDirs = append(result.RemovedDirs, dir)
					removedInPass++
				}
			}
		}

		// If no directory was removed in this pass, we're done
		if removedInPass == 0 {
			break
		}

		// In dry-run mode, only one pass needed (not actually deleting)
		if mode == ModeDryRun {
			break
		}
	}

	return result, nil
}

// isDirEmpty checks if a directory is empty
// Ignores default system files (.DS_Store, Thumbs.db, etc.)
func isDirEmpty(path string) (bool, error) {
	return isDirEmptyWithIgnored(path, ignoredFiles)
}

// isDirEmptyWithIgnored checks if a directory is empty ignoring certain files
func isDirEmptyWithIgnored(path string, ignoredFilesList []string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("failed to read directory: %w", err)
	}

	// Count only non-ignored files/directories
	realCount := 0
	for _, entry := range entries {
		// Ignore specified files
		if !entry.IsDir() && isIgnoredFile(entry.Name(), ignoredFilesList) {
			continue
		}
		realCount++
	}

	return realCount == 0, nil
}

// isIgnoredFile checks if a file should be ignored
func isIgnoredFile(name string, ignoredFilesList []string) bool {
	for _, ignored := range ignoredFilesList {
		if name == ignored {
			return true
		}
	}
	return false
}

// removeIgnoredFiles removes all ignored files from a directory
func removeIgnoredFiles(dirPath string, ignoredFilesList []string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		// Skip directories
		if entry.IsDir() {
			continue
		}

		// Check if file should be removed (is ignored)
		if isIgnoredFile(entry.Name(), ignoredFilesList) {
			filePath := filepath.Join(dirPath, entry.Name())
			if err := os.Remove(filePath); err != nil {
				slog.Debug("failed to remove ignored file", "path", filePath, "error", err)
				// Continue anyway, not critical
			} else {
				slog.Debug("removed ignored file", "path", filePath)
			}
		}
	}

	return nil
}

// isProtectedDir checks if the path contains a protected directory
func isProtectedDir(path string) bool {
	for _, protected := range protectedDirs {
		if strings.Contains(path, string(filepath.Separator)+protected) ||
			strings.HasSuffix(path, string(filepath.Separator)+protected) {
			return true
		}
	}
	return false
}

// askConfirmation asks user confirmation to remove empty directories
// Returns true if user confirms, false otherwise
func askConfirmation(emptyDirs []string) bool {
	if len(emptyDirs) == 0 {
		return false
	}

	fmt.Println()
	slog.Warn("found empty directories",
		"count", len(emptyDirs),
		"action", "will be removed if confirmed")

	// Display directories (max 10)
	displayCount := len(emptyDirs)
	if displayCount > 10 {
		displayCount = 10
	}

	fmt.Println("\nEmpty directories to remove:")
	for i := 0; i < displayCount; i++ {
		fmt.Printf("  - %s\n", emptyDirs[i])
	}
	if len(emptyDirs) > 10 {
		fmt.Printf("  ... and %d more\n", len(emptyDirs)-10)
	}

	fmt.Print("\nDo you want to remove these empty directories? [y/o/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		slog.Warn("failed to read confirmation", "error", err)
		return false
	}

	response = strings.ToLower(strings.TrimSpace(response))
	// Accept: y, yes, o, oui (case insensitive)
	// Reject: n, no, non, or anything else
	return response == "y" || response == "yes" || response == "o" || response == "oui"
}
