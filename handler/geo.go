package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	earthRadiusMeters = 6371000.0 // Mean Earth radius in meters
)

// CalculateDistance calculates the distance in meters between two GPS coordinates
// using the Haversine formula
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := degreesToRadians(lat1)
	lon1Rad := degreesToRadians(lon1)
	lat2Rad := degreesToRadians(lat2)
	lon2Rad := degreesToRadians(lon2)

	// Differences
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in meters
	distance := earthRadiusMeters * c

	return distance
}

// CalculateCentroid calculates the centroid (geometric center) of a set of GPS coordinates
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

// FormatLocationName formats GPS coordinates as folder name
// Format: "48.8566N-2.3522E" or "34.0522S-118.2437W"
func FormatLocationName(coord GPSCoord) string {
	// Determine directions
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

	// Format with 4 decimal places
	return fmt.Sprintf("%.4f%s-%.4f%s", absLat, latDir, absLon, lonDir)
}

// degreesToRadians converts degrees to radians
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

const (
	geocodingTimeout   = 3 * time.Second
	nominatimRateLimit = 1 * time.Second // Nominatim requires max 1 req/s
	geocodingUserAgent = "picsplit/2.9.0 (https://github.com/sebastienfr/picsplit)"
	nominatimURL       = "https://nominatim.openstreetmap.org/reverse"
	bigDataCloudURL    = "https://api.bigdatacloud.net/data/reverse-geocode-client"
)

var (
	geocodeCache      = make(map[string]string)
	geocodeCacheMutex sync.RWMutex
	lastNominatimCall time.Time
	nominatimMutex    sync.Mutex
)

// LocationInfo contains reverse geocoded location information
type LocationInfo struct {
	City    string
	Country string
}

// ReverseGeocode tries to get a human-readable location name from GPS coordinates
// Returns an enriched folder name with format: "coordinates - country - city"
// Falls back to just coordinates if geocoding fails (offline, timeout, error) or if useGeocoding is false
func ReverseGeocode(lat, lon float64, useGeocoding bool) string {
	coords := FormatLocationName(GPSCoord{Lat: lat, Lon: lon})

	// If geocoding disabled, return coordinates only
	if !useGeocoding {
		return coords
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%.4f,%.4f", lat, lon)
	if cached, found := getCachedLocation(cacheKey); found {
		return cached
	}

	// Try to get location info (non-blocking, with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), geocodingTimeout)
	defer cancel()

	var locationInfo *LocationInfo

	// Try Nominatim first
	if info := tryNominatim(ctx, lat, lon); info != nil {
		locationInfo = info
	} else if info := tryBigDataCloud(ctx, lat, lon); info != nil {
		// Fallback to BigDataCloud
		locationInfo = info
	}

	// Build folder name
	folderName := coords
	if locationInfo != nil && locationInfo.Country != "" {
		if locationInfo.City != "" {
			folderName = fmt.Sprintf("%s - %s - %s", coords, locationInfo.Country, locationInfo.City)
		} else {
			folderName = fmt.Sprintf("%s - %s", coords, locationInfo.Country)
		}
		slog.Debug("reverse geocoded location",
			"coords", coords,
			"country", locationInfo.Country,
			"city", locationInfo.City)
	}

	// Sanitize and cache
	folderName = sanitizeFolderName(folderName)
	setCachedLocation(cacheKey, folderName)

	return folderName
}

// tryNominatim attempts reverse geocoding using OpenStreetMap Nominatim
func tryNominatim(ctx context.Context, lat, lon float64) *LocationInfo {
	// Respect rate limit (1 req/s)
	nominatimMutex.Lock()
	timeSinceLastCall := time.Since(lastNominatimCall)
	if timeSinceLastCall < nominatimRateLimit {
		time.Sleep(nominatimRateLimit - timeSinceLastCall)
	}
	lastNominatimCall = time.Now()
	nominatimMutex.Unlock()

	url := fmt.Sprintf("%s?lat=%.4f&lon=%.4f&format=json&zoom=10&addressdetails=1",
		nominatimURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("failed to create nominatim request", "error", err)
		return nil
	}
	req.Header.Set("User-Agent", geocodingUserAgent)

	client := &http.Client{Timeout: geocodingTimeout}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("nominatim request failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("nominatim returned non-OK status", "status", resp.StatusCode)
		return nil
	}

	var result struct {
		Address struct {
			City    string `json:"city"`
			Town    string `json:"town"`
			Village string `json:"village"`
			County  string `json:"county"`
			State   string `json:"state"`
			Country string `json:"country"`
		} `json:"address"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Debug("failed to decode nominatim response", "error", err)
		return nil
	}

	// Choose best available city name
	city := result.Address.City
	if city == "" {
		city = result.Address.Town
	}
	if city == "" {
		city = result.Address.Village
	}

	if result.Address.Country == "" {
		return nil
	}

	return &LocationInfo{
		City:    city,
		Country: result.Address.Country,
	}
}

// tryBigDataCloud attempts reverse geocoding using BigDataCloud API
func tryBigDataCloud(ctx context.Context, lat, lon float64) *LocationInfo {
	url := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&localityLanguage=en",
		bigDataCloudURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("failed to create bigdatacloud request", "error", err)
		return nil
	}

	client := &http.Client{Timeout: geocodingTimeout}
	resp, err := client.Do(req)
	if err != nil {
		slog.Debug("bigdatacloud request failed", "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("bigdatacloud returned non-OK status", "status", resp.StatusCode)
		return nil
	}

	var result struct {
		City        string `json:"city"`
		Locality    string `json:"locality"`
		CountryName string `json:"countryName"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Debug("failed to decode bigdatacloud response", "error", err)
		return nil
	}

	city := result.City
	if city == "" {
		city = result.Locality
	}

	if result.CountryName == "" {
		return nil
	}

	return &LocationInfo{
		City:    city,
		Country: result.CountryName,
	}
}

// sanitizeFolderName removes or replaces characters that are invalid in folder names
func sanitizeFolderName(name string) string {
	// Replace invalid characters: / \ : * ? " < > |
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "-",
	)
	return strings.TrimSpace(replacer.Replace(name))
}

// getCachedLocation retrieves a cached geocoding result
func getCachedLocation(key string) (string, bool) {
	geocodeCacheMutex.RLock()
	defer geocodeCacheMutex.RUnlock()
	name, exists := geocodeCache[key]
	return name, exists
}

// setCachedLocation stores a geocoding result in cache
func setCachedLocation(key, value string) {
	geocodeCacheMutex.Lock()
	defer geocodeCacheMutex.Unlock()
	geocodeCache[key] = value
}
