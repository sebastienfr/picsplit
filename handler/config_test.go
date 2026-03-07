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

func TestExecutionMode_IsValid(t *testing.T) {
	tests := []struct {
		mode ExecutionMode
		want bool
	}{
		{ModeValidate, true},
		{ModeDryRun, true},
		{ModeRun, true},
		{ExecutionMode("invalid"), false},
		{ExecutionMode(""), false},
	}

	for _, tt := range tests {
		if got := tt.mode.IsValid(); got != tt.want {
			t.Errorf("ExecutionMode(%q).IsValid() = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestParseExecutionMode(t *testing.T) {
	tests := []struct {
		input   string
		want    ExecutionMode
		wantErr bool
	}{
		{"validate", ModeValidate, false},
		{"dryrun", ModeDryRun, false},
		{"run", ModeRun, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		got, err := ParseExecutionMode(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("ParseExecutionMode(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseExecutionMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
