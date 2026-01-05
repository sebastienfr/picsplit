package handler

import (
	"context"
	"math"
	"sync"
	"testing"
)

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name      string
		lat1      float64
		lon1      float64
		lat2      float64
		lon2      float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "Paris to London",
			lat1:      48.8566, // Paris
			lon1:      2.3522,
			lat2:      51.5074, // London
			lon2:      -0.1278,
			expected:  343558, // ~344 km
			tolerance: 1000,   // ±1km tolerance
		},
		{
			name:      "New York to Los Angeles",
			lat1:      40.7128, // New York
			lon1:      -74.0060,
			lat2:      34.0522, // Los Angeles
			lon2:      -118.2437,
			expected:  3936000, // ~3936 km
			tolerance: 10000,   // ±10km tolerance (variation due to Earth's shape)
		},
		{
			name:      "Same location (Paris to Paris)",
			lat1:      48.8566,
			lon1:      2.3522,
			lat2:      48.8566,
			lon2:      2.3522,
			expected:  0,
			tolerance: 1,
		},
		{
			name:      "Short distance (1km)",
			lat1:      48.8566,
			lon1:      2.3522,
			lat2:      48.8656, // ~1km north
			lon2:      2.3522,
			expected:  1000,
			tolerance: 50,
		},
		{
			name:      "Equator crossing",
			lat1:      5.0,
			lon1:      0.0,
			lat2:      -5.0,
			lon2:      0.0,
			expected:  1111949, // ~1112 km
			tolerance: 1000,
		},
		{
			name:      "Negative coordinates (Southern hemisphere)",
			lat1:      -33.8688, // Sydney
			lon1:      151.2093,
			lat2:      -37.8136, // Melbourne
			lon2:      144.9631,
			expected:  713594, // ~714 km
			tolerance: 1000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)

			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("CalculateDistance() = %.2f meters, want %.2f±%.2f meters",
					result, tt.expected, tt.tolerance)
			}
		})
	}
}

func TestCalculateCentroid(t *testing.T) {
	tests := []struct {
		name      string
		coords    []GPSCoord
		expected  GPSCoord
		tolerance float64
	}{
		{
			name: "Three points triangle",
			coords: []GPSCoord{
				{Lat: 48.8566, Lon: 2.3522},  // Paris
				{Lat: 51.5074, Lon: -0.1278}, // London
				{Lat: 52.5200, Lon: 13.4050}, // Berlin
			},
			expected: GPSCoord{
				Lat: (48.8566 + 51.5074 + 52.5200) / 3,
				Lon: (2.3522 - 0.1278 + 13.4050) / 3,
			},
			tolerance: 0.0001,
		},
		{
			name: "Two points (midpoint)",
			coords: []GPSCoord{
				{Lat: 48.8566, Lon: 2.3522},
				{Lat: 51.5074, Lon: -0.1278},
			},
			expected: GPSCoord{
				Lat: (48.8566 + 51.5074) / 2,
				Lon: (2.3522 - 0.1278) / 2,
			},
			tolerance: 0.0001,
		},
		{
			name: "Single point",
			coords: []GPSCoord{
				{Lat: 48.8566, Lon: 2.3522},
			},
			expected:  GPSCoord{Lat: 48.8566, Lon: 2.3522},
			tolerance: 0.0001,
		},
		{
			name:      "Empty coordinates",
			coords:    []GPSCoord{},
			expected:  GPSCoord{Lat: 0, Lon: 0},
			tolerance: 0.0001,
		},
		{
			name: "Negative coordinates",
			coords: []GPSCoord{
				{Lat: -33.8688, Lon: 151.2093}, // Sydney
				{Lat: -37.8136, Lon: 144.9631}, // Melbourne
			},
			expected: GPSCoord{
				Lat: (-33.8688 - 37.8136) / 2,
				Lon: (151.2093 + 144.9631) / 2,
			},
			tolerance: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCentroid(tt.coords)

			if math.Abs(result.Lat-tt.expected.Lat) > tt.tolerance ||
				math.Abs(result.Lon-tt.expected.Lon) > tt.tolerance {
				t.Errorf("CalculateCentroid() = (%.4f, %.4f), want (%.4f, %.4f)",
					result.Lat, result.Lon, tt.expected.Lat, tt.expected.Lon)
			}
		})
	}
}

func TestFormatLocationName(t *testing.T) {
	tests := []struct {
		name     string
		coord    GPSCoord
		expected string
	}{
		{
			name:     "Paris (North, East)",
			coord:    GPSCoord{Lat: 48.8566, Lon: 2.3522},
			expected: "48.8566N-2.3522E",
		},
		{
			name:     "London (North, West)",
			coord:    GPSCoord{Lat: 51.5074, Lon: -0.1278},
			expected: "51.5074N-0.1278W",
		},
		{
			name:     "Sydney (South, East)",
			coord:    GPSCoord{Lat: -33.8688, Lon: 151.2093},
			expected: "33.8688S-151.2093E",
		},
		{
			name:     "Buenos Aires (South, West)",
			coord:    GPSCoord{Lat: -34.6037, Lon: -58.3816},
			expected: "34.6037S-58.3816W",
		},
		{
			name:     "Equator and Prime Meridian",
			coord:    GPSCoord{Lat: 0.0, Lon: 0.0},
			expected: "0.0000N-0.0000E",
		},
		{
			name:     "High precision coordinates",
			coord:    GPSCoord{Lat: 48.858844, Lon: 2.294351},
			expected: "48.8588N-2.2944E",
		},
		{
			name:     "Negative latitude (South)",
			coord:    GPSCoord{Lat: -0.0001, Lon: 100.0},
			expected: "0.0001S-100.0000E",
		},
		{
			name:     "Negative longitude (West)",
			coord:    GPSCoord{Lat: 40.0, Lon: -0.0001},
			expected: "40.0000N-0.0001W",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatLocationName(tt.coord)

			if result != tt.expected {
				t.Errorf("FormatLocationName() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDegreesToRadians(t *testing.T) {
	tests := []struct {
		name      string
		degrees   float64
		expected  float64
		tolerance float64
	}{
		{
			name:      "0 degrees",
			degrees:   0,
			expected:  0,
			tolerance: 0.0001,
		},
		{
			name:      "90 degrees",
			degrees:   90,
			expected:  math.Pi / 2,
			tolerance: 0.0001,
		},
		{
			name:      "180 degrees",
			degrees:   180,
			expected:  math.Pi,
			tolerance: 0.0001,
		},
		{
			name:      "360 degrees",
			degrees:   360,
			expected:  2 * math.Pi,
			tolerance: 0.0001,
		},
		{
			name:      "Negative degrees",
			degrees:   -90,
			expected:  -math.Pi / 2,
			tolerance: 0.0001,
		},
		{
			name:      "45 degrees",
			degrees:   45,
			expected:  math.Pi / 4,
			tolerance: 0.0001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := degreesToRadians(tt.degrees)

			if math.Abs(result-tt.expected) > tt.tolerance {
				t.Errorf("degreesToRadians(%v) = %v, want %v", tt.degrees, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFolderName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid name with spaces",
			input:    "Paris - France - Ile de France",
			expected: "Paris - France - Ile de France",
		},
		{
			name:     "name with forward slash",
			input:    "Paris/France",
			expected: "Paris-France",
		},
		{
			name:     "name with backslash",
			input:    "Paris\\France",
			expected: "Paris-France",
		},
		{
			name:     "name with colon",
			input:    "Paris:France",
			expected: "Paris-France",
		},
		{
			name:     "name with asterisk",
			input:    "Paris*France",
			expected: "ParisFrance",
		},
		{
			name:     "name with question mark",
			input:    "Paris?France",
			expected: "ParisFrance",
		},
		{
			name:     "name with quotes",
			input:    "Paris\"France\"",
			expected: "ParisFrance",
		},
		{
			name:     "name with angle brackets",
			input:    "Paris<France>",
			expected: "ParisFrance",
		},
		{
			name:     "name with pipe",
			input:    "Paris|France",
			expected: "Paris-France",
		},
		{
			name:     "multiple invalid characters",
			input:    "Paris / France \\ Test : 2024 * ?",
			expected: "Paris - France - Test - 2024",
		},
		{
			name:     "leading and trailing spaces",
			input:    "  Paris - France  ",
			expected: "Paris - France",
		},
		{
			name:     "already sanitized",
			input:    "48.8566N-2.3522E - France - Paris",
			expected: "48.8566N-2.3522E - France - Paris",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFolderName(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFolderName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestReverseGeocode_Cache(t *testing.T) {
	// Reset cache before test
	geocodeCacheMutex.Lock()
	geocodeCache = make(map[string]string)
	geocodeCacheMutex.Unlock()

	// Paris coordinates
	lat, lon := 48.8566, 2.3522

	// First call - will attempt geocoding (may succeed or fallback to coords)
	result1 := ReverseGeocode(lat, lon, true)

	// Second call - should use cache (same result)
	result2 := ReverseGeocode(lat, lon, true)

	if result1 != result2 {
		t.Errorf("ReverseGeocode cache not working: first=%q, second=%q", result1, result2)
	}

	// Result should at least contain coordinates
	expectedCoords := "48.8566N-2.3522E"
	if result1[:len(expectedCoords)] != expectedCoords {
		t.Errorf("ReverseGeocode result should start with coordinates: got %q", result1)
	}
}

func TestReverseGeocode_Fallback(t *testing.T) {
	// Reset cache
	geocodeCacheMutex.Lock()
	geocodeCache = make(map[string]string)
	geocodeCacheMutex.Unlock()

	// Use invalid coordinates that will fail geocoding
	lat, lon := 0.0, 0.0

	result := ReverseGeocode(lat, lon, true)

	// Should at least return coordinates format
	expectedCoords := "0.0000N-0.0000E"
	if !containsString(result, expectedCoords) {
		t.Errorf("ReverseGeocode fallback should contain coordinates: got %q", result)
	}
}

func TestReverseGeocode_Disabled(t *testing.T) {
	// When geocoding is disabled, should return only coordinates
	lat, lon := 48.8566, 2.3522

	result := ReverseGeocode(lat, lon, false)

	expected := "48.8566N-2.3522E"
	if result != expected {
		t.Errorf("ReverseGeocode with geocoding disabled should return only coordinates: got %q, want %q", result, expected)
	}
}

func TestGetCachedLocation(t *testing.T) {
	// Reset cache
	geocodeCacheMutex.Lock()
	geocodeCache = make(map[string]string)
	geocodeCacheMutex.Unlock()

	key := "48.8566,2.3522"
	value := "48.8566N-2.3522E - France - Paris"

	// Not in cache yet
	if _, found := getCachedLocation(key); found {
		t.Error("getCachedLocation should return false for non-existent key")
	}

	// Add to cache
	setCachedLocation(key, value)

	// Should now be in cache
	cached, found := getCachedLocation(key)
	if !found {
		t.Error("getCachedLocation should return true for cached key")
	}
	if cached != value {
		t.Errorf("getCachedLocation returned %q, want %q", cached, value)
	}
}

// Helper function for string contains
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || s == substr)
}

func TestTryNominatim_InvalidCoordinates(t *testing.T) {
	ctx := context.Background()

	// Test with invalid coordinates (should return nil or handle gracefully)
	result := tryNominatim(ctx, 999.0, 999.0)

	// Should handle invalid coordinates gracefully (return nil or valid response)
	// We don't assert the result as it depends on the API behavior
	_ = result
}

func TestTryBigDataCloud_InvalidCoordinates(t *testing.T) {
	ctx := context.Background()

	// Test with invalid coordinates
	result := tryBigDataCloud(ctx, 999.0, 999.0)

	// Should handle invalid coordinates gracefully
	_ = result
}

func TestTryNominatim_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return nil when context is canceled
	result := tryNominatim(ctx, 48.8566, 2.3522)

	if result != nil {
		t.Error("tryNominatim should return nil with canceled context")
	}
}

func TestTryBigDataCloud_CanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Should return nil when context is canceled
	result := tryBigDataCloud(ctx, 48.8566, 2.3522)

	if result != nil {
		t.Error("tryBigDataCloud should return nil with canceled context")
	}
}

func TestReverseGeocode_ConcurrentAccess(t *testing.T) {
	// Reset cache
	geocodeCacheMutex.Lock()
	geocodeCache = make(map[string]string)
	geocodeCacheMutex.Unlock()

	// Test concurrent access to cache (should not panic)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			lat := 48.8566 + float64(idx)*0.001
			lon := 2.3522 + float64(idx)*0.001
			_ = ReverseGeocode(lat, lon, false) // Use false to avoid API calls
		}(i)
	}
	wg.Wait()
}
