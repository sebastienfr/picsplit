package handler

import (
	"os"
	"testing"
	"time"
)

func TestClusterByLocation(t *testing.T) {
	tests := []struct {
		name             string
		files            []FileMetadata
		radiusMeters     float64
		expectedClusters int
		expectedNoGPS    int
	}{
		{
			name: "Two clusters within radius",
			files: []FileMetadata{
				{
					FileInfo: &fakeFileInfo{name: "paris1.jpg"},
					DateTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522}, // Paris
				},
				{
					FileInfo: &fakeFileInfo{name: "paris2.jpg"},
					DateTime: time.Date(2024, 6, 15, 11, 0, 0, 0, time.UTC),
					GPS:      &GPSCoord{Lat: 48.8570, Lon: 2.3525}, // Paris (500m away)
				},
				{
					FileInfo: &fakeFileInfo{name: "london1.jpg"},
					DateTime: time.Date(2024, 6, 16, 10, 0, 0, 0, time.UTC),
					GPS:      &GPSCoord{Lat: 51.5074, Lon: -0.1278}, // London
				},
				{
					FileInfo: &fakeFileInfo{name: "london2.jpg"},
					DateTime: time.Date(2024, 6, 16, 11, 0, 0, 0, time.UTC),
					GPS:      &GPSCoord{Lat: 51.5080, Lon: -0.1280}, // London (600m away)
				},
			},
			radiusMeters:     2000, // 2km radius
			expectedClusters: 2,    // Paris cluster + London cluster
			expectedNoGPS:    0,
		},
		{
			name: "Single cluster, all points close",
			files: []FileMetadata{
				{
					FileInfo: &fakeFileInfo{name: "photo1.jpg"},
					GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522},
				},
				{
					FileInfo: &fakeFileInfo{name: "photo2.jpg"},
					GPS:      &GPSCoord{Lat: 48.8570, Lon: 2.3525},
				},
				{
					FileInfo: &fakeFileInfo{name: "photo3.jpg"},
					GPS:      &GPSCoord{Lat: 48.8568, Lon: 2.3520},
				},
			},
			radiusMeters:     2000,
			expectedClusters: 1,
			expectedNoGPS:    0,
		},
		{
			name: "Files without GPS",
			files: []FileMetadata{
				{
					FileInfo: &fakeFileInfo{name: "photo1.jpg"},
					GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522},
				},
				{
					FileInfo: &fakeFileInfo{name: "photo2.jpg"},
					GPS:      nil, // No GPS
				},
				{
					FileInfo: &fakeFileInfo{name: "photo3.jpg"},
					GPS:      nil, // No GPS
				},
			},
			radiusMeters:     2000,
			expectedClusters: 1,
			expectedNoGPS:    2,
		},
		{
			name: "All files without GPS",
			files: []FileMetadata{
				{
					FileInfo: &fakeFileInfo{name: "photo1.jpg"},
					GPS:      nil,
				},
				{
					FileInfo: &fakeFileInfo{name: "photo2.jpg"},
					GPS:      nil,
				},
			},
			radiusMeters:     2000,
			expectedClusters: 0,
			expectedNoGPS:    2,
		},
		{
			name:             "Empty file list",
			files:            []FileMetadata{},
			radiusMeters:     2000,
			expectedClusters: 0,
			expectedNoGPS:    0,
		},
		{
			name: "Multiple distant clusters",
			files: []FileMetadata{
				{FileInfo: &fakeFileInfo{name: "paris.jpg"}, GPS: &GPSCoord{Lat: 48.8566, Lon: 2.3522}},
				{FileInfo: &fakeFileInfo{name: "london.jpg"}, GPS: &GPSCoord{Lat: 51.5074, Lon: -0.1278}},
				{FileInfo: &fakeFileInfo{name: "berlin.jpg"}, GPS: &GPSCoord{Lat: 52.5200, Lon: 13.4050}},
				{FileInfo: &fakeFileInfo{name: "madrid.jpg"}, GPS: &GPSCoord{Lat: 40.4168, Lon: -3.7038}},
			},
			radiusMeters:     2000, // 2km - each city is far apart
			expectedClusters: 4,    // Each city is its own cluster
			expectedNoGPS:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters, noGPS := ClusterByLocation(tt.files, tt.radiusMeters)

			if len(clusters) != tt.expectedClusters {
				t.Errorf("ClusterByLocation() clusters = %d, want %d", len(clusters), tt.expectedClusters)
			}

			if len(noGPS) != tt.expectedNoGPS {
				t.Errorf("ClusterByLocation() noGPS = %d, want %d", len(noGPS), tt.expectedNoGPS)
			}

			// Verify that all clusters have a centroid
			for i, cluster := range clusters {
				if cluster.Centroid.Lat == 0 && cluster.Centroid.Lon == 0 && len(cluster.Files) > 0 {
					t.Errorf("Cluster %d has zero centroid but contains files", i)
				}
			}
		})
	}
}

func TestGroupLocationByTime(t *testing.T) {
	tests := []struct {
		name           string
		cluster        LocationCluster
		delta          time.Duration
		expectedGroups int
	}{
		{
			name: "Two groups with large gap",
			cluster: LocationCluster{
				Centroid: GPSCoord{Lat: 48.8566, Lon: 2.3522},
				Files: []FileMetadata{
					{
						FileInfo: &fakeFileInfo{name: "photo1.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
						GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522},
					},
					{
						FileInfo: &fakeFileInfo{name: "photo2.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 15, 0, 0, time.UTC),
						GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522},
					},
					{
						FileInfo: &fakeFileInfo{name: "photo3.jpg"},
						DateTime: time.Date(2024, 6, 15, 14, 0, 0, 0, time.UTC), // 3h45 gap
						GPS:      &GPSCoord{Lat: 48.8566, Lon: 2.3522},
					},
				},
			},
			delta:          1 * time.Hour,
			expectedGroups: 2, // Group 1: 10:00-10:15, Group 2: 14:00
		},
		{
			name: "Single group continuous",
			cluster: LocationCluster{
				Centroid: GPSCoord{Lat: 48.8566, Lon: 2.3522},
				Files: []FileMetadata{
					{
						FileInfo: &fakeFileInfo{name: "photo1.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					},
					{
						FileInfo: &fakeFileInfo{name: "photo2.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 20, 0, 0, time.UTC),
					},
					{
						FileInfo: &fakeFileInfo{name: "photo3.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 45, 0, 0, time.UTC),
					},
				},
			},
			delta:          1 * time.Hour,
			expectedGroups: 1,
		},
		{
			name: "Empty cluster",
			cluster: LocationCluster{
				Centroid: GPSCoord{Lat: 48.8566, Lon: 2.3522},
				Files:    []FileMetadata{},
			},
			delta:          1 * time.Hour,
			expectedGroups: 0,
		},
		{
			name: "Single file",
			cluster: LocationCluster{
				Centroid: GPSCoord{Lat: 48.8566, Lon: 2.3522},
				Files: []FileMetadata{
					{
						FileInfo: &fakeFileInfo{name: "photo1.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					},
				},
			},
			delta:          1 * time.Hour,
			expectedGroups: 1,
		},
		{
			name: "All files separated by gaps",
			cluster: LocationCluster{
				Centroid: GPSCoord{Lat: 48.8566, Lon: 2.3522},
				Files: []FileMetadata{
					{
						FileInfo: &fakeFileInfo{name: "photo1.jpg"},
						DateTime: time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC),
					},
					{
						FileInfo: &fakeFileInfo{name: "photo2.jpg"},
						DateTime: time.Date(2024, 6, 15, 13, 0, 0, 0, time.UTC), // 3h gap
					},
					{
						FileInfo: &fakeFileInfo{name: "photo3.jpg"},
						DateTime: time.Date(2024, 6, 15, 16, 0, 0, 0, time.UTC), // 3h gap
					},
				},
			},
			delta:          1 * time.Hour,
			expectedGroups: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := GroupLocationByTime(tt.cluster, tt.delta)

			if len(groups) != tt.expectedGroups {
				t.Errorf("GroupLocationByTime() groups = %d, want %d", len(groups), tt.expectedGroups)
			}

			// Verify that groups are not empty (except if expectedGroups == 0)
			for i, group := range groups {
				if len(group) == 0 {
					t.Errorf("Group %d is empty", i)
				}
			}
		})
	}
}

func TestGetNoLocationFolderName(t *testing.T) {
	result := GetNoLocationFolderName()
	expected := "NoLocation"

	if result != expected {
		t.Errorf("GetNoLocationFolderName() = %v, want %v", result, expected)
	}
}

// fakeFileInfo implements os.FileInfo for tests
type fakeFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return f.size }
func (f *fakeFileInfo) Mode() os.FileMode  { return f.mode }
func (f *fakeFileInfo) ModTime() time.Time { return f.modTime }
func (f *fakeFileInfo) IsDir() bool        { return f.isDir }
func (f *fakeFileInfo) Sys() interface{}   { return nil }
