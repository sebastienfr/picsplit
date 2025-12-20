package handler

import (
	"reflect"
	"testing"
)

func TestValidateExtension(t *testing.T) {
	tests := []struct {
		name    string
		ext     string
		wantErr bool
	}{
		// Valid cases
		{"valid short", "jpg", false},
		{"valid medium", "jpeg", false},
		{"valid 8 chars", "12345678", false},
		{"valid with dot", ".png", false},
		{"valid uppercase", "PNG", false},
		{"valid mixed case", "NeF", false},
		{"valid numbers", "mp4", false},

		// Invalid cases
		{"empty", "", true},
		{"too long 9 chars", "123456789", true},
		{"too long word", "verylongext", true},
		{"with space", "jp g", true},
		{"with dash", "jp-g", true},
		{"with underscore", "jp_g", true},
		{"with dot in middle", "jp.g", true},
		{"special char", "jpg!", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExtension(tt.ext)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExtension(%q) error = %v, wantErr %v", tt.ext, err, tt.wantErr)
			}
		})
	}
}

func TestBuildExtensionMap(t *testing.T) {
	tests := []struct {
		name     string
		defaults map[string]bool
		custom   []string
		expected map[string]bool
		wantErr  bool
	}{
		{
			name:     "no custom extensions",
			defaults: map[string]bool{".jpg": true, ".png": true},
			custom:   nil,
			expected: map[string]bool{".jpg": true, ".png": true},
			wantErr:  false,
		},
		{
			name:     "add custom extensions",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"png", "gif"},
			expected: map[string]bool{".jpg": true, ".png": true, ".gif": true},
			wantErr:  false,
		},
		{
			name:     "custom with leading dot",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{".png", "gif"},
			expected: map[string]bool{".jpg": true, ".png": true, ".gif": true},
			wantErr:  false,
		},
		{
			name:     "uppercase custom normalized",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"PNG", "GIF"},
			expected: map[string]bool{".jpg": true, ".png": true, ".gif": true},
			wantErr:  false,
		},
		{
			name:     "whitespace handled",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{" png ", "  gif  "},
			expected: map[string]bool{".jpg": true, ".png": true, ".gif": true},
			wantErr:  false,
		},
		{
			name:     "empty strings ignored",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"", "  ", "png"},
			expected: map[string]bool{".jpg": true, ".png": true},
			wantErr:  false,
		},
		{
			name:     "duplicate ignored",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"jpg"},
			expected: map[string]bool{".jpg": true},
			wantErr:  false,
		},
		{
			name:     "invalid too long",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"verylongext"},
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid special char",
			defaults: map[string]bool{".jpg": true},
			custom:   []string{"jp-g"},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildExtensionMap(tt.defaults, tt.custom)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildExtensionMap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("buildExtensionMap() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExecutionContext_IsMovie(t *testing.T) {
	ctx := &executionContext{
		movieExtensions: map[string]bool{".mov": true, ".mp4": true, ".mkv": true},
		rawExtensions:   map[string]bool{".nef": true},
		photoExtensions: map[string]bool{".jpg": true},
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"mov file", "video.mov", true},
		{"MOV uppercase", "video.MOV", true},
		{"mp4 file", "clip.mp4", true},
		{"custom mkv", "movie.mkv", true},
		{"jpg not movie", "photo.jpg", false},
		{"nef not movie", "raw.nef", false},
		{"unknown ext", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.isMovie(tt.filename); got != tt.want {
				t.Errorf("isMovie(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExecutionContext_IsPhoto(t *testing.T) {
	ctx := &executionContext{
		movieExtensions: map[string]bool{".mov": true},
		rawExtensions:   map[string]bool{".nef": true, ".cr2": true},
		photoExtensions: map[string]bool{".jpg": true, ".png": true},
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"jpg photo", "image.jpg", true},
		{"JPG uppercase", "image.JPG", true},
		{"png custom", "graphic.png", true},
		{"nef raw is photo", "raw.nef", true},
		{"cr2 raw is photo", "raw.cr2", true},
		{"mov not photo", "video.mov", false},
		{"unknown ext", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.isPhoto(tt.filename); got != tt.want {
				t.Errorf("isPhoto(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExecutionContext_IsRaw(t *testing.T) {
	ctx := &executionContext{
		movieExtensions: map[string]bool{".mov": true},
		rawExtensions:   map[string]bool{".nef": true, ".rwx": true},
		photoExtensions: map[string]bool{".jpg": true},
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"nef raw", "image.nef", true},
		{"NEF uppercase", "image.NEF", true},
		{"rwx custom raw", "photo.rwx", true},
		{"jpg not raw", "photo.jpg", false},
		{"mov not raw", "video.mov", false},
		{"unknown ext", "file.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.isRaw(tt.filename); got != tt.want {
				t.Errorf("isRaw(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestExecutionContext_IsMediaFile(t *testing.T) {
	ctx := &executionContext{
		movieExtensions: map[string]bool{".mov": true, ".mkv": true},
		rawExtensions:   map[string]bool{".nef": true},
		photoExtensions: map[string]bool{".jpg": true, ".png": true},
	}

	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"jpg media", "photo.jpg", true},
		{"png custom media", "image.png", true},
		{"nef raw media", "raw.nef", true},
		{"mov video media", "video.mov", true},
		{"mkv custom video media", "clip.mkv", true},
		{"txt not media", "file.txt", false},
		{"doc not media", "readme.doc", false},
		{"no extension", "noext", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.isMediaFile(tt.filename); got != tt.want {
				t.Errorf("isMediaFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestNewExecutionContext(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "no custom extensions",
			cfg: &Config{
				CustomPhotoExts: nil,
				CustomVideoExts: nil,
				CustomRawExts:   nil,
			},
			wantErr: false,
		},
		{
			name: "valid custom extensions",
			cfg: &Config{
				CustomPhotoExts: []string{"png", "gif"},
				CustomVideoExts: []string{"mkv"},
				CustomRawExts:   []string{"rwx"},
			},
			wantErr: false,
		},
		{
			name: "invalid photo extension (too long)",
			cfg: &Config{
				CustomPhotoExts: []string{"verylongext"},
			},
			wantErr: true,
		},
		{
			name: "invalid video extension (special char)",
			cfg: &Config{
				CustomVideoExts: []string{"mk-v"},
			},
			wantErr: true,
		},
		{
			name: "invalid raw extension (space)",
			cfg: &Config{
				CustomRawExts: []string{"r wx"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := newExecutionContext(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("newExecutionContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ctx == nil {
				t.Error("newExecutionContext() returned nil context")
			}
		})
	}
}

func TestNewDefaultExecutionContext(t *testing.T) {
	ctx := newDefaultExecutionContext()

	if ctx == nil {
		t.Fatal("newDefaultExecutionContext() returned nil")
	}

	// Verify it has default extensions
	if !ctx.isMovie("video.mov") {
		t.Error("default context should recognize .mov")
	}
	if !ctx.isPhoto("image.jpg") {
		t.Error("default context should recognize .jpg")
	}
	if !ctx.isRaw("photo.nef") {
		t.Error("default context should recognize .nef")
	}

	// Verify it doesn't have custom extensions
	if ctx.isPhoto("image.png") {
		t.Error("default context should NOT recognize .png (not in defaults)")
	}
}
