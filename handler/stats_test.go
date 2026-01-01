package handler

import (
	"errors"
	"testing"
	"time"
)

func TestProcessingStats_Duration(t *testing.T) {
	tests := []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		wantMin   time.Duration
		wantMax   time.Duration
	}{
		{
			name:      "completed processing",
			startTime: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			endTime:   time.Date(2024, 1, 1, 10, 5, 30, 0, time.UTC),
			wantMin:   5*time.Minute + 30*time.Second,
			wantMax:   5*time.Minute + 30*time.Second,
		},
		{
			name:      "ongoing processing (no end time)",
			startTime: time.Now().Add(-2 * time.Second),
			endTime:   time.Time{},
			wantMin:   1 * time.Second,
			wantMax:   3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &ProcessingStats{
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
			}
			duration := stats.Duration()
			if duration < tt.wantMin || duration > tt.wantMax {
				t.Errorf("Duration() = %v, want between %v and %v", duration, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestProcessingStats_Throughput(t *testing.T) {
	tests := []struct {
		name       string
		totalBytes int64
		duration   time.Duration
		want       float64
		tolerance  float64
	}{
		{
			name:       "100 MB in 10 seconds",
			totalBytes: 100 * 1024 * 1024,
			duration:   10 * time.Second,
			want:       10.0, // 100MB / 10s = 10 MB/s
			tolerance:  0.1,
		},
		{
			name:       "1 GB in 1 minute",
			totalBytes: 1024 * 1024 * 1024,
			duration:   60 * time.Second,
			want:       17.066667, // ~17 MB/s
			tolerance:  0.1,
		},
		{
			name:       "zero duration",
			totalBytes: 100 * 1024 * 1024,
			duration:   0,
			want:       0,
			tolerance:  0,
		},
		{
			name:       "zero bytes",
			totalBytes: 0,
			duration:   10 * time.Second,
			want:       0,
			tolerance:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			stats := &ProcessingStats{
				StartTime:  now,
				EndTime:    now.Add(tt.duration),
				TotalBytes: tt.totalBytes,
			}
			throughput := stats.Throughput()
			if !floatEquals(throughput, tt.want, tt.tolerance) {
				t.Errorf("Throughput() = %.6f, want %.6f (±%.6f)", throughput, tt.want, tt.tolerance)
			}
		})
	}
}

func TestProcessingStats_SuccessRate(t *testing.T) {
	tests := []struct {
		name           string
		totalFiles     int
		processedFiles int
		want           float64
	}{
		{
			name:           "100% success",
			totalFiles:     100,
			processedFiles: 100,
			want:           100.0,
		},
		{
			name:           "50% success",
			totalFiles:     100,
			processedFiles: 50,
			want:           50.0,
		},
		{
			name:           "0% success",
			totalFiles:     100,
			processedFiles: 0,
			want:           0.0,
		},
		{
			name:           "zero total files",
			totalFiles:     0,
			processedFiles: 0,
			want:           0.0,
		},
		{
			name:           "partial success",
			totalFiles:     1245,
			processedFiles: 1242,
			want:           99.75903614457831,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &ProcessingStats{
				TotalFiles:     tt.totalFiles,
				ProcessedFiles: tt.processedFiles,
			}
			successRate := stats.SuccessRate()
			if !floatEquals(successRate, tt.want, 0.0001) {
				t.Errorf("SuccessRate() = %.10f, want %.10f", successRate, tt.want)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{"bytes", 500, "500 bytes"},
		{"KB boundary", 1024, "1.0 KB"},
		{"KB", 5 * 1024, "5.0 KB"},
		{"MB boundary", 1024 * 1024, "1.0 MB"},
		{"MB", 50 * 1024 * 1024, "50.0 MB"},
		{"GB boundary", 1024 * 1024 * 1024, "1.0 GB"},
		{"GB", 24*1024*1024*1024 + 512*1024*1024, "24.5 GB"},
		{"large GB", 100 * 1024 * 1024 * 1024, "100.0 GB"},
		{"zero", 0, "0 bytes"},
		{"decimal MB", 158*1024*1024 + 51200, "158.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.want)
			}
		})
	}
}

func TestProcessingStats_Integration(t *testing.T) {
	// Simulate a real processing scenario
	stats := &ProcessingStats{
		StartTime:            time.Now(),
		TotalFiles:           1245,
		PhotoCount:           980,
		VideoCount:           165,
		RawCount:             100,
		GroupsCreated:        12,
		PairedRaw:            85,
		OrphanRaw:            15,
		TotalBytes:           24*1024*1024*1024 + 512*1024*1024, // 24.5 GB
		ModTimeFallbackCount: 18,
		ProcessedFiles:       1242,
	}

	// Add some errors (3 critical, 15 warnings)
	stats.Errors = []*PicsplitError{
		{Type: ErrTypePermission, Op: "read_file", Path: "/photos/IMG_001.jpg", Err: errors.New("permission denied")},
		{Type: ErrTypeValidation, Op: "validate_extension", Path: "/photos/IMG_015.orf", Details: map[string]string{"extension": "orf"}},
		{Type: ErrTypeIO, Op: "move_file", Path: "/photos/VID_002.mp4", Err: errors.New("disk full")},
	}

	// Add warnings (non-critical errors)
	for i := 0; i < 15; i++ {
		stats.Errors = append(stats.Errors, &PicsplitError{
			Type: ErrTypeEXIF,
			Op:   "extract_metadata",
			Path: "/photos/IMG_RAW_" + string(rune(i)) + ".nef",
			Err:  errors.New("No associated JPEG found"),
		})
	}

	// Simulate processing time
	time.Sleep(10 * time.Millisecond)
	stats.EndTime = time.Now()

	// Verify calculations
	duration := stats.Duration()
	if duration < 10*time.Millisecond {
		t.Errorf("Duration() = %v, want >= 10ms", duration)
	}

	throughput := stats.Throughput()
	if throughput <= 0 {
		t.Errorf("Throughput() = %v, want > 0", throughput)
	}

	successRate := stats.SuccessRate()
	expectedRate := float64(1242) / float64(1245) * 100
	if !floatEquals(successRate, expectedRate, 0.01) {
		t.Errorf("SuccessRate() = %.2f%%, want %.2f%%", successRate, expectedRate)
	}

	// Verify error counts
	criticalCount := 0
	warningCount := 0
	for _, err := range stats.Errors {
		if err.IsCritical() {
			criticalCount++
		} else {
			warningCount++
		}
	}

	if criticalCount != 3 {
		t.Errorf("Critical errors = %d, want 3", criticalCount)
	}
	if warningCount != 15 {
		t.Errorf("Warnings = %d, want 15", warningCount)
	}
}

func TestProcessingStats_PrintSummary(t *testing.T) {
	// This test just ensures PrintSummary doesn't panic
	// Actual output verification would require capturing stdout/logs
	stats := &ProcessingStats{
		StartTime:      time.Now().Add(-5 * time.Second),
		EndTime:        time.Now(),
		TotalFiles:     100,
		ProcessedFiles: 98,
		PhotoCount:     80,
		VideoCount:     15,
		RawCount:       5,
		GroupsCreated:  10,
		TotalBytes:     1024 * 1024 * 1024, // 1 GB
		Errors: []*PicsplitError{
			{Type: ErrTypeEXIF, Op: "extract_metadata", Path: "/test.nef", Err: errors.New("no JPEG")},
		},
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrintSummary() panicked: %v", r)
		}
	}()

	stats.PrintSummary(false)
	stats.PrintSummary(true) // dry run mode
}

// Helper function to compare floats with tolerance
func floatEquals(a, b, tolerance float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}
