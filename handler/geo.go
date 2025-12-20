package handler

import (
	"fmt"
	"math"
)

const (
	earthRadiusMeters = 6371000.0 // Rayon moyen de la Terre en mètres
)

// CalculateDistance calcule la distance en mètres entre deux coordonnées GPS
// en utilisant la formule de Haversine
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Conversion des degrés en radians
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)

	// Différences
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Formule de Haversine
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance en mètres
	distance := earthRadiusMeters * c

	return distance
}

// CalculateCentroid calcule le centroid (center géométrique) d'un ensemble de coordonnées GPS
func CalculateCentroid(coords []GPSCoord) GPSCoord {
	if len(coords) == 0 {
		return GPSCoord{Lat: 0, Lon: 0}
	}

	var sumLat, sumLon float64
	for _, coord := range coords {
		sumLat += coord.Lat
		sumLon += coord.Lon
	}

	return GPSCoord{
		Lat: sumLat / float64(len(coords)),
		Lon: sumLon / float64(len(coords)),
	}
}

// FormatLocationName formate des coordonnées GPS en nom de dossier
// Format: "48.8566N-2.3522E" ou "34.0522S-118.2437W"
func FormatLocationName(coord GPSCoord) string {
	// Déterminer les directions
	latDir := "N"
	if coord.Lat < 0 {
		latDir = "S"
	}

	lonDir := "E"
	if coord.Lon < 0 {
		lonDir = "W"
	}

	// Utiliser valeurs absolues
	absLat := math.Abs(coord.Lat)
	absLon := math.Abs(coord.Lon)

	// Formater avec 4 décimales
	return fmt.Sprintf("%.4f%s-%.4f%s", absLat, latDir, absLon, lonDir)
}

// degreesToRadians convertit des degrés en radians
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}
