package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"
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

	// Use absolute values
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
	geocodingTimeout      = 3 * time.Second
	nominatimRateLimit    = 1 * time.Second // Nominatim requires max 1 req/s
	nominatimURL          = "https://nominatim.openstreetmap.org/reverse"
	bigDataCloudURL       = "https://api.bigdatacloud.net/data/reverse-geocode-client"
	maxGeocodingBodyBytes = 64 * 1024 // 64 KB max response body
	maxFolderNameLength   = 200       // Max sanitized folder name length
)

func geocodingUserAgent() string {
	return "picsplit/" + Version + " (https://github.com/sebastienfr/picsplit)"
}

// LocationInfo contains reverse geocoded location information
type LocationInfo struct {
	City    string
	Country string
}

type GeocodingService struct {
	cache             map[string]string
	cacheMutex        sync.RWMutex
	lastNominatimCall time.Time
	nominatimMutex    sync.Mutex
	client            *http.Client
}

func NewGeocodingService() *GeocodingService {
	return &GeocodingService{
		cache:  make(map[string]string),
		client: &http.Client{Timeout: geocodingTimeout},
	}
}

func (gs *GeocodingService) ReverseGeocode(lat, lon float64, useGeocoding bool) string {
	coords := FormatLocationName(GPSCoord{Lat: lat, Lon: lon})

	// If geocoding disabled, return coordinates only
	if !useGeocoding {
		return coords
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%.4f,%.4f", lat, lon)
	if cached, found := gs.getCachedLocation(cacheKey); found {
		return cached
	}

	// Try to get location info (non-blocking, with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), geocodingTimeout)
	defer cancel()

	var locationInfo *LocationInfo

	// Try Nominatim first
	if info := gs.tryNominatim(ctx, lat, lon); info != nil {
		locationInfo = info
	} else if info := gs.tryBigDataCloud(ctx, lat, lon); info != nil {
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
	gs.setCachedLocation(cacheKey, folderName)

	return folderName
}

func (gs *GeocodingService) tryNominatim(ctx context.Context, lat, lon float64) *LocationInfo {
	// Respect rate limit (1 req/s)
	gs.nominatimMutex.Lock()
	timeSinceLastCall := time.Since(gs.lastNominatimCall)
	if timeSinceLastCall < nominatimRateLimit {
		time.Sleep(nominatimRateLimit - timeSinceLastCall)
	}
	gs.lastNominatimCall = time.Now()
	gs.nominatimMutex.Unlock()

	url := fmt.Sprintf("%s?lat=%.4f&lon=%.4f&format=json&zoom=10&addressdetails=1",
		nominatimURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("failed to create nominatim request", "error", err)
		return nil
	}
	req.Header.Set("User-Agent", geocodingUserAgent())

	resp, err := gs.client.Do(req)
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

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxGeocodingBodyBytes)).Decode(&result); err != nil {
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

func (gs *GeocodingService) tryBigDataCloud(ctx context.Context, lat, lon float64) *LocationInfo {
	url := fmt.Sprintf("%s?latitude=%.4f&longitude=%.4f&localityLanguage=en",
		bigDataCloudURL, lat, lon)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("failed to create bigdatacloud request", "error", err)
		return nil
	}
	req.Header.Set("User-Agent", geocodingUserAgent())

	resp, err := gs.client.Do(req)
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

	if err := json.NewDecoder(io.LimitReader(resp.Body, maxGeocodingBodyBytes)).Decode(&result); err != nil {
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

// folderNameReplacer replaces characters that are invalid in folder names: / \ : * ? " < > |
var folderNameReplacer = strings.NewReplacer(
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

// sanitizeFolderName removes or replaces characters that are invalid in folder names
func sanitizeFolderName(name string) string {
	name = folderNameReplacer.Replace(name)

	// Remove null bytes and control characters
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7F || unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)

	name = strings.TrimSpace(name)

	// Enforce maximum length (truncate at rune boundary to preserve UTF-8 validity)
	if len(name) > maxFolderNameLength {
		runes := []rune(name)
		for len(string(runes)) > maxFolderNameLength {
			runes = runes[:len(runes)-1]
		}
		name = string(runes)
	}

	return name
}

func (gs *GeocodingService) getCachedLocation(key string) (string, bool) {
	gs.cacheMutex.RLock()
	defer gs.cacheMutex.RUnlock()
	name, exists := gs.cache[key]
	return name, exists
}

func (gs *GeocodingService) setCachedLocation(key, value string) {
	gs.cacheMutex.Lock()
	defer gs.cacheMutex.Unlock()
	gs.cache[key] = value
}
