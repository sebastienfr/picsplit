package main

import (
	"testing"
)

func TestSetupLogger(t *testing.T) {
	t.Run("verbose mode", func(t *testing.T) {
		// Test that setupLogger doesn't panic with verbose=true
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("setupLogger(true) panicked: %v", r)
			}
		}()
		setupLogger(true)
	})

	t.Run("non-verbose mode", func(t *testing.T) {
		// Test that setupLogger doesn't panic with verbose=false
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("setupLogger(false) panicked: %v", r)
			}
		}()
		setupLogger(false)
	})
}
