package handler

import (
	"log/slog"
	"time"
)

const (
	noLocationFolderName = "NoLocation"
)

// LocationCluster représente un cluster de fichiers groupés par localisation
type LocationCluster struct {
	Files    []FileMetadata
	Centroid GPSCoord
}

// ClusterByLocation groupe les fichiers par proximité géographique (DBSCAN-like)
// Les fichiers sans GPS sont retournés séparément
func ClusterByLocation(files []FileMetadata, radiusMeters float64) ([]LocationCluster, []FileMetadata) {
	var filesWithGPS []FileMetadata
	var filesWithoutGPS []FileMetadata

	// Séparer les fichiers avec/sans GPS
	for _, file := range files {
		if file.GPS != nil {
			filesWithGPS = append(filesWithGPS, file)
		} else {
			filesWithoutGPS = append(filesWithoutGPS, file)
		}
	}

	if len(filesWithGPS) == 0 {
		slog.Debug("no files with GPS coordinates found")
		return nil, filesWithoutGPS
	}

	// DBSCAN-like clustering
	clusters := []LocationCluster{}
	visited := make(map[int]bool)

	for i := range filesWithGPS {
		if visited[i] {
			continue
		}

		// Créer un nouveau cluster
		cluster := LocationCluster{
			Files: []FileMetadata{filesWithGPS[i]},
		}
		visited[i] = true

		// Trouver tous les fichiers dans le rayon
		queue := []int{i}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			for j := range filesWithGPS {
				if visited[j] {
					continue
				}

				distance := CalculateDistance(
					filesWithGPS[current].GPS.Lat,
					filesWithGPS[current].GPS.Lon,
					filesWithGPS[j].GPS.Lat,
					filesWithGPS[j].GPS.Lon,
				)

				if distance <= radiusMeters {
					cluster.Files = append(cluster.Files, filesWithGPS[j])
					visited[j] = true
					queue = append(queue, j)
				}
			}
		}

		// Calculer le centroid du cluster
		coords := make([]GPSCoord, len(cluster.Files))
		for i, file := range cluster.Files {
			coords[i] = *file.GPS
		}
		cluster.Centroid = CalculateCentroid(coords)

		clusters = append(clusters, cluster)
	}

	slog.Debug("location clusters created",
		"clusters", len(clusters),
		"files_with_gps", len(filesWithGPS))

	return clusters, filesWithoutGPS
}

// GroupLocationByTime groupe les fichiers d'un cluster de localisation par gaps temporels
func GroupLocationByTime(cluster LocationCluster, delta time.Duration) [][]FileMetadata {
	if len(cluster.Files) == 0 {
		return nil
	}

	// Trier par date/heure
	sortFilesByDateTime(cluster.Files)

	// Grouper par gaps temporels (même algorithme que groupFilesByGaps)
	groups := [][]FileMetadata{}
	currentGroup := []FileMetadata{cluster.Files[0]}

	for i := 1; i < len(cluster.Files); i++ {
		gap := cluster.Files[i].DateTime.Sub(cluster.Files[i-1].DateTime)

		if gap > delta {
			// Gap trop grand, créer un nouveau groupe
			groups = append(groups, currentGroup)
			currentGroup = []FileMetadata{cluster.Files[i]}
		} else {
			// Même groupe
			currentGroup = append(currentGroup, cluster.Files[i])
		}
	}

	// Ajouter le dernier groupe
	groups = append(groups, currentGroup)

	slog.Debug("location split into time-based groups",
		"location", FormatLocationName(cluster.Centroid),
		"groups", len(groups),
		"delta", delta)

	return groups
}

// GetNoLocationFolderName retourne le nom du dossier pour les fichiers sans GPS
func GetNoLocationFolderName() string {
	return noLocationFolderName
}
