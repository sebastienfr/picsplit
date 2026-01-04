package handler

import (
	"testing"
	"time"
)

// TestConfig_Validate_MoveDuplicates tests the validation of MoveDuplicates flag
func TestConfig_Validate_MoveDuplicates(t *testing.T) {
	t.Run("move-duplicates requires detect-duplicates", func(t *testing.T) {
		cfg := &Config{
			BasePath:         t.TempDir(),
			Delta:            30 * time.Minute,
			MoveDuplicates:   true,
			DetectDuplicates: false,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() should fail when MoveDuplicates is true but DetectDuplicates is false")
		}
		if err.Error() != "--move-duplicates requires --detect-duplicates" {
			t.Errorf("Validate() error = %v, want '--move-duplicates requires --detect-duplicates'", err)
		}
	})

	t.Run("skip-duplicates and move-duplicates are mutually exclusive", func(t *testing.T) {
		cfg := &Config{
			BasePath:         t.TempDir(),
			Delta:            30 * time.Minute,
			DetectDuplicates: true,
			SkipDuplicates:   true,
			MoveDuplicates:   true,
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Validate() should fail when both SkipDuplicates and MoveDuplicates are true")
		}
		if err.Error() != "--skip-duplicates and --move-duplicates are mutually exclusive" {
			t.Errorf("Validate() error = %v, want '--skip-duplicates and --move-duplicates are mutually exclusive'", err)
		}
	})

	t.Run("move-duplicates with detect-duplicates is valid", func(t *testing.T) {
		cfg := &Config{
			BasePath:         t.TempDir(),
			Delta:            30 * time.Minute,
			DetectDuplicates: true,
			MoveDuplicates:   true,
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})
}
