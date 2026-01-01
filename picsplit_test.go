package main

import (
	"testing"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		logFormat string
	}{
		{"debug level with text format", "debug", "text"},
		{"info level with text format", "info", "text"},
		{"warn level with text format", "warn", "text"},
		{"error level with text format", "error", "text"},
		{"debug level with json format", "debug", "json"},
		{"info level with json format", "info", "json"},
		{"invalid level defaults to info", "invalid", "text"},
		{"invalid format defaults to text", "info", "invalid"},
		{"case insensitive level", "DEBUG", "text"},
		{"case insensitive format", "info", "JSON"},
		{"warning alias for warn", "warning", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that setupLogger doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("setupLogger(%q, %q) panicked: %v", tt.logLevel, tt.logFormat, r)
				}
			}()
			setupLogger(tt.logLevel, tt.logFormat)
		})
	}
}
