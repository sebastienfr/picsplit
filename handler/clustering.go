package handler

import (
	"log/slog"
	"time"
)

const (
	noLocationFolderName = "NoLocation"
)

// LocationCluster represents a cluster of files grouped by location
type LocationCluster struct {
	Files    []FileMetadata
	Centroid GPSCoord
}

// ClusterByLocation groups files by geographic proximity (DBSCAN-like)
// Files without GPS are returned separately
func ClusterByLocation(files []FileMetadata, radiusMeters float64) ([]LocationCluster, []FileMetadata) {
	var filesWithGPS []FileMetadata
	var filesWithoutGPS []FileMetadata

	// Separate files with/without GPS
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

		// Create a new cluster
		cluster := LocationCluster{
			Files: []FileMetadata{filesWithGPS[i]},
		}
		visited[i] = true

		// Find all files within radius
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

		// Calculate cluster centroid
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

// GroupLocationByTime groups files from a location cluster by time gaps
func GroupLocationByTime(cluster LocationCluster, delta time.Duration) [][]FileMetadata {
	if len(cluster.Files) == 0 {
		return nil
	}

	// Sort by date/time
	sortFilesByDateTime(cluster.Files)

	// Group by time gaps (same algorithm as groupFilesByGaps)
	groups := [][]FileMetadata{}
	currentGroup := []FileMetadata{cluster.Files[0]}

	for i := 1; i < len(cluster.Files); i++ {
		gap := cluster.Files[i].DateTime.Sub(cluster.Files[i-1].DateTime)

		if gap > delta {
			// Gap too large, create new group
			groups = append(groups, currentGroup)
			currentGroup = []FileMetadata{cluster.Files[i]}
		} else {
			// Same group
			currentGroup = append(currentGroup, cluster.Files[i])
		}
	}

	// Add last group
	groups = append(groups, currentGroup)

	slog.Debug("location split into time-based groups",
		"location", FormatLocationName(cluster.Centroid),
		"groups", len(groups),
		"delta", delta)

	return groups
}

// GetNoLocationFolderName returns the folder name for files without GPS
func GetNoLocationFolderName() string {
	return noLocationFolderName
}
