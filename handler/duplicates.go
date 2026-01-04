package handler

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// DuplicateDetector détecte les fichiers dupliqués via hash SHA256
type DuplicateDetector struct {
	hashes     map[string]string  // hash → first file path
	duplicates map[string]string  // duplicate path → original path
	sizeGroups map[int64][]string // size → file paths (pré-filtrage)
	enabled    bool
}

// NewDuplicateDetector crée un nouveau détecteur de doublons
func NewDuplicateDetector(enabled bool) *DuplicateDetector {
	return &DuplicateDetector{
		hashes:     make(map[string]string),
		duplicates: make(map[string]string),
		sizeGroups: make(map[int64][]string),
		enabled:    enabled,
	}
}

// AddFile ajoute un fichier au pré-filtrage par taille
// Cette étape est optionnelle mais améliore les performances
func (d *DuplicateDetector) AddFile(filePath string, size int64) {
	if !d.enabled {
		return
	}
	d.sizeGroups[size] = append(d.sizeGroups[size], filePath)
}

// Check vérifie si le fichier est un doublon
// Retourne (isDuplicate, originalPath, error)
func (d *DuplicateDetector) Check(filePath string, size int64) (bool, string, error) {
	if !d.enabled {
		return false, "", nil
	}

	// Optimisation : si un seul fichier de cette taille, pas de doublon possible
	if len(d.sizeGroups[size]) == 1 {
		slog.Debug("unique file size, skipping hash", "file", filePath, "size", size)
		return false, "", nil
	}

	// Calculer le hash
	hash, err := sha256File(filePath)
	if err != nil {
		return false, "", fmt.Errorf("failed to hash file: %w", err)
	}

	// Vérifier si hash déjà vu
	if original, found := d.hashes[hash]; found {
		// Doublon détecté !
		d.duplicates[filePath] = original
		slog.Debug("duplicate detected", "file", filePath, "original", original, "hash", hash[:16])
		return true, original, nil
	}

	// Premier fichier avec ce hash
	d.hashes[hash] = filePath
	return false, "", nil
}

// GetDuplicates retourne la map des doublons détectés
// map[duplicate_path]original_path
func (d *DuplicateDetector) GetDuplicates() map[string]string {
	return d.duplicates
}

// GetStats retourne les statistiques du détecteur
func (d *DuplicateDetector) GetStats() (totalFiles int, uniqueSizes int, potentialDuplicates int, confirmedDuplicates int) {
	totalFiles = 0
	uniqueSizes = 0
	potentialDuplicates = 0

	for _, files := range d.sizeGroups {
		totalFiles += len(files)
		if len(files) == 1 {
			uniqueSizes++
		} else {
			potentialDuplicates += len(files)
		}
	}

	confirmedDuplicates = len(d.duplicates)
	return
}

// sha256File calcule le hash SHA256 d'un fichier
func sha256File(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
