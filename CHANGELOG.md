# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Planned
- Interactive console GUI (TUI)
- Duplicate detection and handling
- Photo similarity clustering

---

## [2.5.2] - 2024-12-31

### Added
- Automatic version detection using Git tags
- `make version` command to display detected version
- Documentation for version management system in README

### Changed
- Simplified build system (removed hardcoded version variables)
- Build flags now inject version via ldflags automatically
- Improved consistency between local and CI builds
- Enhanced README with new Building section

### Technical
- Refactored `getBuildInfo()` to use ldflags-injected version
- Added `fetch-depth: 0` in all CI workflow jobs for tag availability
- Updated GoReleaser config with automatic version injection via `{{.Version}}`
- Makefile now uses `git describe` for automatic version detection

---

## [2.5.1] - 2024-12-31

### Changed
- Modernized build configuration for Go 1.25
- Removed obsolete ldflags from GoReleaser (version info now via runtime/debug)
- Cleaned up Makefile (removed hardcoded VERSION variable)

### Technical
- Version info now handled by `runtime/debug.ReadBuildInfo`
- Simplified build flags to only use `-s -w` for binary size optimization
- Fixed typo in golangci-lint configuration comment

---

## [2.5.0] - 2024-12-20

### Added
- **Custom file extension support** via CLI flags
  - `--photo-ext` / `-pext`: Add custom photo extensions
  - `--video-ext` / `-vext`: Add custom video extensions
  - `--raw-ext` / `-rext`: Add custom RAW extensions
- Runtime extension configuration without recompiling
- Support for comma-separated extension lists (e.g., `--photo-ext png,bmp`)
- Extension validation (max 8 characters, alphanumeric only)
- Case-insensitive extension matching

### Improved
- **Smarter NoLocation handling in GPS mode**
  - `NoLocation/` folder only created when location clusters exist AND some files lack GPS
  - When ALL files lack GPS: time-based folders created at root (no unnecessary NoLocation segregation)
  - More intuitive folder structure for photos without GPS data

### Technical
- New `handler/extensions.go` (~180 lines) with extension management
- New `handler/extensions_test.go` (~320 lines, 18 test cases, 100% coverage)
- New `executionContext` struct for runtime extension maps
- Updated `Config` and `MergeConfig` with custom extension fields
- Refactored handlers to use `executionContext` for extension checks
- CLI flag implementation with `parseExtensions()` function

### Notes
- Custom extensions are **additive** (extend defaults, never remove them)
- ⚠️ Flag order matters: flags must come BEFORE paths (urfave/cli limitation)
- Extensions validated on startup with clear error messages

**Closes**: Issue #3

---

## [2.4.0] - 2024-12-20

### Added
- **Merge command** to combine time-based folders
- Interactive conflict resolution with 5 options:
  - Rename: Generate unique filename
  - Skip: Keep target version
  - Overwrite: Replace with source file
  - Apply to all: Use same action for remaining conflicts
  - Quit: Abort merge operation
- `--force` flag for automatic conflict overwrite (no prompts)
- `--dryrun` flag for merge preview
- Media folder validation (only folders with media files can be merged)
- Automatic source folder cleanup after successful merge
- Structure preservation for `mov/` and `raw/` subdirectories

### Technical
- New `handler/merger.go` (~430 lines) with merge orchestration
- New `handler/merger_test.go` (~680 lines, 15+ test cases)
- New `MergeConfig` struct
- Validation functions: `isMediaFile()`, `isMediaFolder()`, `validateMergeFolders()`
- Conflict handling: `detectConflict()`, `askUserConflictResolution()`, `generateUniqueName()`
- Test coverage: 77.6%

### Limitations
- GPS location folders (e.g., `48.8566N-2.3522E/`) cannot be merged (contains nested time folders)
- Only time-based folders (e.g., `2025 - 0616 - 0945/`) are mergeable

**Closes**: Issue #4

---

## [2.3.0] - 2024-12-20

### Added
- **GPS location clustering** mode
  - Groups photos by geographic location first, then by time within each location
  - DBSCAN-like spatial clustering algorithm
  - Configurable clustering radius (default: 2km)
- New CLI flags:
  - `--gps` / `-g`: Enable GPS location clustering (opt-in, default: false)
  - `--gps-radius` / `-gr`: Set clustering radius in meters (default: 2000)
- `NoLocation/` folder for files without GPS coordinates
- Haversine distance calculation for accurate geographic distances
- Folder naming: `<Lat>N/S-<Lon>E/W` (e.g., `48.8566N-2.3522E`)

### Technical
- New `handler/geo.go` with geographic calculations:
  - Haversine distance formula
  - Centroid calculation
  - Coordinate formatting
- New `handler/clustering.go` with spatial clustering logic
- New `GPSCoord` and `LocationCluster` structs
- Updated `Config` with `UseGPS` and `GPSRadius` fields
- Test coverage: 100% for GPS modules (30 test cases)

### Algorithm
- **Spatial clustering**: Files within radius belong to same location cluster
- **Temporal grouping**: Within each location, files grouped by time gaps
- **NoLocation handling**: Files without GPS placed in separate folder (only when location clusters exist)

---

## [2.2.0] - 2024-12-20

### Added
- **EXIF metadata support** for photos
  - Dates extracted from EXIF `DateTimeOriginal` field
  - More accurate than file modification times
- **Video metadata support** for MP4/MOV files
  - Extraction from `creation_time` metadata
- **RAW+JPEG pairing**
  - RAW files automatically share EXIF data from associated JPEG
  - Example: `PHOTO_01.NEF` uses EXIF from `PHOTO_01.JPG`
- GPS coordinate extraction from EXIF (preparation for v2.3.0)
- CLI flag: `--use-exif` (default: true) to enable/disable EXIF extraction
- Date validation: dates must be between 1990 and now+1 day

### Changed
- **Strict fallback mode**: If ANY file lacks valid EXIF, ALL files use ModTime
- Internal refactoring to use `FileMetadata` struct instead of raw `os.FileInfo`

### Technical
- New `handler/exif.go` with EXIF/video metadata extraction
- New `FileMetadata` struct with date source tracking
- New dependencies:
  - `github.com/rwcarlsen/goexif` for EXIF parsing
  - `github.com/abema/go-mp4` for video metadata

### Metadata Priority
1. Photos: EXIF DateTimeOriginal
2. RAW files: EXIF from paired JPEG
3. Videos: MP4/MOV creation_time
4. Fallback: File modification time

---

## [2.1.0] - 2024-12-19

### Improved
- **Gap-based event detection algorithm**
  - Files grouped by temporal gaps instead of rounded timestamps
  - Folder names use exact timestamp of first file (no more rounding)
  - Better handling of continuous sessions across hour boundaries
  - More natural event detection based on shooting patterns

### Changed
- Refactored `Split()` function with new grouping algorithm
- New functions: `collectMediaFiles()`, `sortFilesByModTime()`, `groupFilesByGaps()`, `processGroup()`
- Removed obsolete functions: `listDirectories()`, `findOrCreateDatedFolder()`, `processFiles()`

### Migration Note
Running v2.1.0 on v2.0.0-organized folders will create new folders.  
**Best practice**: Reorganize from original source files.

### Example
**Before (v2.0.0)**: Files rounded to nearest delta interval  
**After (v2.1.0)**: Files grouped when gap exceeds delta

---

## [2.0.0] - 2024-12-19

### Breaking Changes
- Migrated from `urfave/cli` v1 to v2
- Go version requirement: 1.25+
- `handler.Split()` signature changed (now uses `Config` struct)

### Added
- Support for modern image formats:
  - HEIC, HEIF (Apple)
  - WebP (Google)
  - AVIF (AV1)
- Support for additional RAW formats:
  - DNG (Adobe)
  - ARW (Sony)
  - ORF (Olympus)
  - RAF (Fujifilm)
- Comprehensive test suite (80%+ code coverage)
- New `handler.Config` struct for better configuration management
- Improved error handling with error wrapping (`fmt.Errorf` with `%w`)
- Better dry-run mode with `[DRY RUN]` prefix in logs

### Changed
- Updated dependencies:
  - `github.com/sirupsen/logrus`: v1.3.0 → v1.9.3
  - `github.com/urfave/cli`: v1.20.0 → v2.27.5
- Replaced deprecated `file.Readdir()` with modern `os.ReadDir()`
- Refactored code to use named constants instead of magic strings
- Changed file move logs from `Warn` to `Info` level
- Split monolithic functions into smaller, testable functions
- Improved Makefile with coverage targets

### Removed
- `etc/mktest.sh` (replaced by Go test fixtures using `t.TempDir()`)

### Technical
- CI/CD pipeline with GitHub Actions
- Automated releases with GoReleaser (multi-platform binaries)
- Security scanning with CodeQL
- Test coverage reporting

---

## [1.0.0] - 2019-01-09

### Added
- Initial public release
- Configurable delta for time gap detection
- Move movies to separate `mov/` folder
- Move RAW files to separate `raw/` folder
- Dry-run mode for safe previewing
- Basic DCIM folder organization

---

## [0.0.1] - 2015-04-17

### Added
- Project start
- Migration from shell script to Go
- Basic file organization functionality

---

[Unreleased]: https://github.com/sebastienfr/picsplit/compare/v2.5.2...HEAD
[2.5.2]: https://github.com/sebastienfr/picsplit/compare/v2.5.1...v2.5.2
[2.5.1]: https://github.com/sebastienfr/picsplit/compare/v2.5.0...v2.5.1
[2.5.0]: https://github.com/sebastienfr/picsplit/compare/v2.4.0...v2.5.0
[2.4.0]: https://github.com/sebastienfr/picsplit/compare/v2.3.0...v2.4.0
[2.3.0]: https://github.com/sebastienfr/picsplit/compare/v2.2.0...v2.3.0
[2.2.0]: https://github.com/sebastienfr/picsplit/compare/v2.1.0...v2.2.0
[2.1.0]: https://github.com/sebastienfr/picsplit/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/sebastienfr/picsplit/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/sebastienfr/picsplit/releases/tag/v1.0.0
[0.0.1]: https://github.com/sebastienfr/picsplit/commit/first
