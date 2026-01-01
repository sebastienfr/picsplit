package handler

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ErrorType représente la catégorie d'erreur
type ErrorType string

const (
	ErrTypeIO         ErrorType = "IO"
	ErrTypePermission ErrorType = "Permission"
	ErrTypeEXIF       ErrorType = "EXIF"
	ErrTypeValidation ErrorType = "Validation"
	ErrTypeVideoMeta  ErrorType = "VideoMeta"
	ErrTypeGPS        ErrorType = "GPS"
)

// PicsplitError est l'erreur structurée de picsplit
type PicsplitError struct {
	Type    ErrorType         // Catégorie de l'erreur
	Op      string            // Opération en cours ("move_file", "extract_exif")
	Path    string            // Fichier/dossier concerné
	Err     error             // Erreur originale
	Details map[string]string // Contexte supplémentaire
}

// Error implémente l'interface error
func (e *PicsplitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %s - %v", e.Type, e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Type, e.Op, e.Path)
}

// Unwrap permet d'extraire l'erreur originale
func (e *PicsplitError) Unwrap() error {
	return e.Err
}

// Suggestion génère une action corrective selon le type d'erreur
func (e *PicsplitError) Suggestion() string {
	switch e.Type {
	case ErrTypePermission:
		if e.Op == "read_file" {
			return fmt.Sprintf("chmod +r %s", e.Path)
		}
		if e.Op == "create_folder" {
			return fmt.Sprintf("chmod +w %s", filepath.Dir(e.Path))
		}
		return fmt.Sprintf("Check permissions on %s", e.Path)

	case ErrTypeValidation:
		if ext := e.Details["extension"]; ext != "" {
			return fmt.Sprintf("picsplit <path> --add-extension %s:raw", ext)
		}
		return "Check file format and configuration"

	case ErrTypeIO:
		if e.Err != nil {
			errMsg := e.Err.Error()
			if strings.Contains(errMsg, "disk full") || strings.Contains(errMsg, "no space") {
				return "Free up disk space and retry"
			}
			if strings.Contains(errMsg, "no such file") {
				return "Check that source path exists"
			}
		}
		return "Check filesystem and disk space"

	case ErrTypeEXIF:
		if strings.Contains(e.Error(), "No associated JPEG") {
			return "File will use modification time as fallback (automatic)"
		}
		if strings.Contains(e.Error(), "corrupted") {
			return "File will use modification time as fallback (automatic)"
		}
		return "File will use modification time as fallback"

	case ErrTypeVideoMeta:
		return "File will use modification time as fallback (automatic)"

	default:
		return "See error message for details"
	}
}

// IsCritical détermine si l'erreur est bloquante
func (e *PicsplitError) IsCritical() bool {
	switch e.Type {
	case ErrTypePermission, ErrTypeIO, ErrTypeValidation:
		return true
	case ErrTypeEXIF, ErrTypeVideoMeta, ErrTypeGPS:
		return false // Fallback automatique possible
	default:
		return true
	}
}
