package handler

import (
	"testing"
)

func TestCreateProgressBar_DebugMode(t *testing.T) {
	// Should return nil in debug mode
	bar := createProgressBar(100, "Test", "debug", "text")
	if bar != nil {
		t.Error("createProgressBar should return nil in debug mode")
	}
}

func TestCreateProgressBar_JSONMode(t *testing.T) {
	// Should return nil in json mode
	bar := createProgressBar(100, "Test", "info", "json")
	if bar != nil {
		t.Error("createProgressBar should return nil in json mode")
	}
}

func TestCreateProgressBar_NormalMode(t *testing.T) {
	// Should create progress bar in normal mode
	bar := createProgressBar(100, "Test", "info", "text")
	if bar == nil {
		t.Error("createProgressBar should create progress bar in normal mode")
	}
}

func TestCreateProgressBar_DebugAndJSON(t *testing.T) {
	// Should return nil when both debug and json
	bar := createProgressBar(100, "Test", "debug", "json")
	if bar != nil {
		t.Error("createProgressBar should return nil in debug+json mode")
	}
}

func TestCreateProgressBar_CaseInsensitive(t *testing.T) {
	// Test case insensitive level/format
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
		expectNil bool
	}{
		{"DEBUG uppercase", "DEBUG", "text", true},
		{"Debug mixed", "Debug", "text", true},
		{"JSON uppercase", "info", "JSON", true},
		{"Json mixed", "info", "Json", true},
		{"Info normal", "info", "text", false},
		{"INFO uppercase", "INFO", "text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := createProgressBar(100, "Test", tt.logLevel, tt.logFormat)
			if tt.expectNil && bar != nil {
				t.Errorf("Expected nil bar for %s/%s", tt.logLevel, tt.logFormat)
			}
			if !tt.expectNil && bar == nil {
				t.Errorf("Expected non-nil bar for %s/%s", tt.logLevel, tt.logFormat)
			}
		})
	}
}
