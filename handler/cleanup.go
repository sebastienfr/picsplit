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

// Liste des dossiers système à protéger
var protectedDirs = []string{
	".git",
	".svn",
	".hg",
	"node_modules",
}

// Liste des fichiers système à ignorer (ne comptent pas comme "contenu")
var ignoredFiles = []string{
	".DS_Store",   // macOS
	"Thumbs.db",   // Windows
	"desktop.ini", // Windows
	"._.DS_Store", // macOS AppleDouble
}

// CleanupResult contient les résultats du nettoyage
type CleanupResult struct {
	RemovedDirs []string
	FailedDirs  map[string]error
}

// CleanupEmptyDirs supprime récursivement les dossiers vides
// en utilisant un parcours bottom-up (post-order traversal).
//
// Paramètres:
//   - rootPath: Le chemin racine à partir duquel chercher les dossiers vides
//   - mode: Le mode d'exécution (ModeValidate, ModeDryRun, ModeRun)
//   - force: Si true, supprime sans confirmation. Si false, demande confirmation en mode Run
//   - customIgnoredFiles: Liste de fichiers supplémentaires à ignorer (en plus des fichiers système par défaut)
//
// Retourne:
//   - CleanupResult contenant la liste des dossiers supprimés et les erreurs
//   - error si une erreur fatale survient
func CleanupEmptyDirs(rootPath string, mode ExecutionMode, force bool, customIgnoredFiles []string) (*CleanupResult, error) {
	result := &CleanupResult{
		RemovedDirs: []string{},
		FailedDirs:  make(map[string]error),
	}

	// Mode validate ne fait pas de cleanup
	if mode == ModeValidate {
		slog.Debug("skipping cleanup in validate mode")
		return result, nil
	}

	// Combiner les fichiers ignorés par défaut avec ceux de l'utilisateur
	allIgnoredFiles := append([]string{}, ignoredFiles...)
	allIgnoredFiles = append(allIgnoredFiles, customIgnoredFiles...)

	if len(customIgnoredFiles) > 0 {
		slog.Debug("using custom ignored files for cleanup", "files", customIgnoredFiles)
	}

	// Faire plusieurs passages pour supprimer les dossiers imbriqués vides
	// Chaque passage peut rendre des parents vides, donc on continue jusqu'à ce qu'il n'y ait plus de changement
	maxPasses := 100 // Protection contre les boucles infinies
	for pass := 0; pass < maxPasses; pass++ {
		emptyDirs := []string{}

		// Collecter les dossiers vides
		err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("failed to access path during cleanup", "path", path, "error", err)
				return nil // Continue le walk
			}

			// Skip fichiers
			if !d.IsDir() {
				return nil
			}

			// Skip root path
			if path == rootPath {
				return nil
			}

			// Skip dossiers protégés
			if isProtectedDir(path) {
				slog.Debug("skipping protected directory", "path", path)
				return fs.SkipDir
			}

			// Vérifier si vide (en tenant compte des fichiers ignorés)
			empty, err := isDirEmptyWithIgnored(path, allIgnoredFiles)
			if err != nil {
				slog.Warn("failed to check if directory is empty", "path", path, "error", err)
				result.FailedDirs[path] = err
				return nil // Continue le walk
			}

			if empty {
				emptyDirs = append(emptyDirs, path)
			}

			return nil
		})

		if err != nil {
			return result, fmt.Errorf("failed to walk directory tree: %w", err)
		}

		// Si aucun dossier vide trouvé, on a fini
		if len(emptyDirs) == 0 {
			break
		}

		// En mode Run sans force, demander confirmation au premier passage
		if mode == ModeRun && !force && pass == 0 {
			if !askConfirmation(emptyDirs) {
				slog.Info("cleanup cancelled by user")
				return result, nil
			}
		}

		// Parcourir les dossiers vides en ordre inverse (bottom-up)
		// pour supprimer les sous-dossiers avant les parents
		removedInPass := 0
		for i := len(emptyDirs) - 1; i >= 0; i-- {
			dir := emptyDirs[i]

			// Re-vérifier si vide (peut avoir changé pendant ce passage)
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
				// Supprimer d'abord les fichiers ignorés dans le dossier
				if err := removeIgnoredFiles(dir, allIgnoredFiles); err != nil {
					slog.Warn("failed to remove ignored files", "path", dir, "error", err)
				}

				// Puis supprimer le dossier vide
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

		// Si aucun dossier n'a été supprimé dans ce passage, on a fini
		if removedInPass == 0 {
			break
		}

		// En mode dry-run, on fait un seul passage (on ne supprime pas vraiment)
		if mode == ModeDryRun {
			break
		}
	}

	return result, nil
}

// isDirEmpty vérifie si un dossier est vide
// Ignore les fichiers système par défaut (.DS_Store, Thumbs.db, etc.)
func isDirEmpty(path string) (bool, error) {
	return isDirEmptyWithIgnored(path, ignoredFiles)
}

// isDirEmptyWithIgnored vérifie si un dossier est vide en ignorant certains fichiers
func isDirEmptyWithIgnored(path string, ignoredFilesList []string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("failed to read directory: %w", err)
	}

	// Compter seulement les fichiers/dossiers non-ignorés
	realCount := 0
	for _, entry := range entries {
		// Ignorer les fichiers spécifiés
		if !entry.IsDir() && isIgnoredFile(entry.Name(), ignoredFilesList) {
			continue
		}
		realCount++
	}

	return realCount == 0, nil
}

// isIgnoredFile vérifie si un fichier doit être ignoré
func isIgnoredFile(name string, ignoredFilesList []string) bool {
	for _, ignored := range ignoredFilesList {
		if name == ignored {
			return true
		}
	}
	return false
}

// removeIgnoredFiles supprime tous les fichiers ignorés d'un dossier
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

// isProtectedDir vérifie si le chemin contient un dossier protégé
func isProtectedDir(path string) bool {
	for _, protected := range protectedDirs {
		if strings.Contains(path, string(filepath.Separator)+protected) ||
			strings.HasSuffix(path, string(filepath.Separator)+protected) {
			return true
		}
	}
	return false
}

// askConfirmation demande confirmation à l'utilisateur pour supprimer les dossiers vides
// Retourne true si l'utilisateur confirme, false sinon
func askConfirmation(emptyDirs []string) bool {
	if len(emptyDirs) == 0 {
		return false
	}

	fmt.Println()
	slog.Warn("found empty directories",
		"count", len(emptyDirs),
		"action", "will be removed if confirmed")

	// Afficher les dossiers (max 10)
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
