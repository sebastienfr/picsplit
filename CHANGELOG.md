# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

---

## [2.8.0] - 2026-01-04

### Added
- **Move duplicates to dedicated folder** ([#16](https://github.com/sebastienfr/picsplit/issues/16)): New `--move-duplicates` flag to move duplicate files to `duplicates/` folder
  - Requires `--detect-duplicates` flag (validation enforced)
  - Mutually exclusive with `--skip-duplicates` 
  - Creates `duplicates/` folder automatically at basePath root
  - Moves detected duplicates to isolated folder for easy cleanup
  - Eliminates magic strings with `duplicatesFolderName` constant
  - Works in all modes (validate, dryrun, run)
  - Compatible with `--continue-on-error` flag
  - Flags: `--move-duplicates` / `--md`
  - New tests: `TestSplit_MoveDuplicates` with 3 comprehensive subtests
  - New tests: `TestConfig_Validate_MoveDuplicates` for flag validation
- **Empty directory cleanup** ([#14](https://github.com/sebastienfr/picsplit/issues/14)): New `--cleanup-empty-dirs` flag to remove empty directories after processing
  - Multi-pass bottom-up traversal to remove nested empty directories
  - Smart file ignoring: directories with only `.DS_Store`, `Thumbs.db`, etc. are considered empty
  - Custom ignored files via `--cleanup-ignore` (comma-separated list, e.g., `.picasa.ini,.nomedia`)
  - Ignored files are automatically deleted before directory removal
  - Interactive confirmation before deletion (unless `--force` is used)
  - Accepts multiple confirmation inputs: `y`, `yes`, `o`, `oui` (case insensitive)
  - Protects system directories (`.git`, `.svn`, `node_modules`, etc.)
  - Skip in validate mode, simulate in dryrun mode, execute in run mode
  - **Limitation**: In dryrun mode, shows only currently empty directories (not future state after moves)
  - Displays cleanup statistics in processing summary
  - New file: `handler/cleanup.go` with `CleanupEmptyDirs()` function
- **Force flag**: New `--force` / `-f` flag to skip confirmation prompts (cleanup, merge conflicts, etc.)
- **Duplicate detection** ([#15](https://github.com/sebastienfr/picsplit/issues/15)): New `--detect-duplicates` flag to detect duplicate files via SHA256 hash
  - Two modes: detection-only (warning) or auto-skip with `--skip-duplicates`
  - Size-based pre-filtering optimization (10x faster: only hash files with matching sizes)
  - Displays duplicate statistics in processing summary
  - Shows duplicate pairs (duplicate → original) in logs
  - Validation: `--skip-duplicates` requires `--detect-duplicates`
  - New file: `handler/duplicates.go` with `DuplicateDetector` and SHA256 hashing
  - Flags: `--detect-duplicates` / `--dd` and `--skip-duplicates` / `--sd`
- **Fast validation mode** ([#13](https://github.com/sebastienfr/picsplit/issues/13)): New `--mode validate` for ultra-fast pre-checks (5s vs 2min)
  - Validates file extensions, permissions, and disk space without EXIF extraction
  - Returns detailed `ValidationReport` with file counts, critical errors, and warnings
  - New file: `handler/validator.go` with `Validate()` function
  - Works for both `split` and `merge` commands
- **Execution modes system**: `--mode validate|dryrun|run` for split and merge operations
  - `validate`: Fast validation without EXIF extraction (~5s for 1000+ files)
  - `dryrun`: Full simulation with EXIF extraction but no file moves
  - `run`: Real execution (default)
- **Merge validation**: New `validateMerge()` function for merge pre-checks
  - Detects conflicts between source folders
  - Estimates disk space requirements
  - Reports `MergeValidationReport` with detailed statistics
- **Makefile improvements**:
  - New `clean-all` target: Deep clean including build cache (for debugging)
  - Modified `clean`: Faster clean (keeps build cache, 4x faster rebuilds: 1.3s vs 5.7s)
  - New `uninstall` target: Remove installed binary from GOPATH/bin
- **Build versioning**: Enhanced version display with build time + commit time
  - Shows actual build timestamp (not just commit time)
  - Format: `2.6.0-dev, built on 2026-01-04 18:41:54 +0000 UTC, git hash 8195207-dirty (commit: 2026-01-01 22:47:04)`

### Changed
- **BREAKING**: Removed `--dryrun` flag → use `--mode dryrun` instead
- **Version flag**: Changed from `--print-version` / `-V` to `--version` / `-v` (more conventional)
- **Test coverage**: Maintained at 79.0% in handler package (excellent coverage on new features)
  - New file: `handler/validator_test.go` (15 test scenarios)
  - New file: `handler/cleanup_test.go` (20 test scenarios for cleanup - comprehensive coverage)
  - New file: `handler/duplicates_test.go` (11 test scenarios - 100% coverage on duplicates.go)
  - Enhanced `handler/merger_test.go` (+9 tests for `validateMerge()`)
  - Enhanced `handler/splitter_test.go` (+15 tests for modes and orphan RAW)
  - Total: ~70 new tests added
  - Cleanup module: 84.8% coverage (CleanupEmptyDirs), 100% for helpers
  - Duplicates module: 100% coverage (all functions)

### Fixed
- **Bug**: `isOrganizedFolder()` incorrectly checked `len(name) == 19` instead of `18`
  - Impact: Date-formatted subfolders were never detected as organized
  - Files affected: `handler/splitter.go:87`
- **Bug**: `Split()` missing mode handling - validate mode executed full split anyway
  - Impact: `--mode validate` moved files instead of just validating
  - Solution: Added `switch cfg.Mode` and extracted `splitInternal()`
  - Files affected: `handler/splitter.go:404`
- **Bug**: `refreshOrphanRAW()` incorrect statistics when processing from within date folder
  - Symptoms: `total=0`, `success_rate=0.0%`, `skipped=-XX` (negative)
  - Impact: Misleading summary in orphan RAW refresh mode
  - Solution: Fix stats initialization in both code paths
  - Files affected: `handler/splitter.go:286-291, 326-330`

### Documentation
- Updated README: Added `--move-duplicates` documentation with examples
- Updated README: Corrected all `merge` command examples (flags before arguments)
- Updated README: Added execution modes documentation with examples
- Updated README: Roadmap section (v2.8.0 completed)
- New CHANGELOG entry for v2.8.0 release

### Improvements
- **Code quality**: Translated all French comments to English (100% English codebase)
  - Improved code maintainability and international collaboration
  - ~40 comments translated across 13 files
  - No French words remaining in production code

### Technical
- New config field: `Config.MoveDuplicates` (default: `false`)
- New constant: `duplicatesFolderName = "duplicates"`
- Enhanced `processGroup()` in `handler/splitter.go` with duplicate move logic
- Enhanced `Config.Validate()` with MoveDuplicates validation rules
- New file: `handler/config_test.go` with validation tests
- Enhanced `handler/splitter_test.go` with move-duplicates scenarios
- Test coverage maintained at 79.8%

---

## [2.7.0] - 2026-01-04

### Added
- **Structured logging** ([#7](https://github.com/sebastienfr/picsplit/issues/7)): Migration to `log/slog` (Go stdlib)
  - Typed, structured logs with key-value pairs
  - Better performance than previous logging library
  - Consistent logging format across the application

- **Configurable log levels** ([#8](https://github.com/sebastienfr/picsplit/issues/8)): New logging configuration
  - `--log-level debug|info|warn|error` flag
  - `--log-format text|json` flag
  - JSON format for log parsing and monitoring tools

- **Real-time progress bar** ([#9](https://github.com/sebastienfr/picsplit/issues/9)): Visual feedback during processing
  - Shows completion percentage in real-time
  - Automatic TTY detection (only in terminals)
  - Disabled in CI/CD environments and with JSON logging

- **Enhanced summary with metrics** ([#10](https://github.com/sebastienfr/picsplit/issues/10)): Detailed processing statistics
  - Duration and throughput (MB/s)
  - File breakdown by type (photos, videos, RAW)
  - Disk usage statistics
  - Separate critical errors and warnings sections

- **Typed errors with context** ([#11](https://github.com/sebastienfr/picsplit/issues/11)): Better error handling
  - Structured error types (Permission, IO, Validation, EXIF, etc.)
  - Automatic suggestions for common issues
  - Context-rich error messages with file paths and operations

- **Continue-on-error mode** ([#12](https://github.com/sebastienfr/picsplit/issues/12)): Process all files despite errors
  - `--continue-on-error` / `--coe` flag
  - Collects all errors instead of stopping at first failure
  - Shows all errors in final summary
  - Useful for large libraries with mixed quality sources

- **Fast validation mode** ([#13](https://github.com/sebastienfr/picsplit/issues/13)): Ultra-fast pre-checks (5s vs 2min)
  - `--mode validate|dryrun|run` execution modes
  - Validates extensions, permissions, disk space without EXIF extraction
  - New file: `handler/validator.go`
  - Breaking change: Removed `--dryrun` flag (use `--mode dryrun`)

- **Empty directory cleanup** ([#14](https://github.com/sebastienfr/picsplit/issues/14)): Automatic cleanup after processing
  - `--cleanup-empty-dirs` / `--ced` flag
  - Multi-pass bottom-up removal of nested empty directories
  - Smart file ignoring (`.DS_Store`, `Thumbs.db`, etc.)
  - Custom ignored files via `--cleanup-ignore`
  - Interactive confirmation (unless `--force`)
  - Protected directories (`.git`, `.svn`, `node_modules`)
  - New file: `handler/cleanup.go`

- **Duplicate detection** ([#15](https://github.com/sebastienfr/picsplit/issues/15)): Identify duplicate files via SHA256 hash
  - `--detect-duplicates` / `--dd` flag for detection
  - `--skip-duplicates` / `--sd` flag to skip duplicates
  - Size-based pre-filtering optimization (10x faster)
  - Shows duplicate statistics in summary
  - New file: `handler/duplicates.go`

- **Force flag**: `--force` / `-f` to skip confirmation prompts
- **Enhanced version display**: Shows build time + commit time

### Changed
- **BREAKING**: Removed `--dryrun` flag → use `--mode dryrun` instead
- **Version flag**: Changed from `--print-version` / `-V` to `--version` / `-v`
- Test coverage maintained at 79.0% in handler package
  - 70+ new tests added across all new features

### Fixed
- **Bug**: `isOrganizedFolder()` incorrectly checked `len(name) == 19` instead of `18`
- **Bug**: `Split()` missing mode handling - validate mode executed full split
- **Bug**: `refreshOrphanRAW()` incorrect statistics when processing from within date folder

### Documentation
- Updated README with all new features and examples
- Updated CLI reference with new flags
- Added comprehensive CHANGELOG entries

---

## [2.6.0] - 2024-12-31

### Added
- **Orphan RAW separation**: New `orphan/` folder for unpaired RAW files (RAW without corresponding JPEG/HEIC)
  - Enabled by default (`--separate-orphan` / `-so` flag)
  - Helps photographers identify RAW files without JPEG after culling workflow
  - Supports JPEG (`.jpg`, `.jpeg`) and HEIC (`.heic`) pairing for iPhone ProRAW
  - Smart pairing detection: checks both source directory and destination folder (handles JPEG processed before RAW)
- CLI flag: `--separate-orphan` / `-so` (default: `true`)
- New test suite: `TestIsRawPaired` and `TestSplit_OrphanRawSeparation` (10 test cases)
  - Covers both processing orders (JPEG before RAW, RAW before JPEG)

### Changed
- Improved `findAssociatedJPEG()` to support HEIC format (iPhone ProRAW+HEIC workflow)
- Updated folder organization: `raw/` for paired RAW, `orphan/` for unpaired RAW

### Technical
- New config field: `Config.SeparateOrphanRaw` (default: `true`)
- New constant: `orphanFolderName = "orphan"`
- New function: `isRawPaired(rawPath, basePath) bool` in `handler/splitter.go`
- Updated `processPicture()` to route RAW files to `raw/` or `orphan/` based on pairing status
- Enhanced `findAssociatedJPEG()` with HEIC support (`.heic`, `.HEIC` extensions)

### Documentation
- New use case: "Photographer workflow: Culling RAW files" in README
- Updated "Smart Organization" feature description with orphan folder mention
- CLI Reference updated with `--separate-orphan` flag

### Example
```bash
# Default behavior (orphan separation enabled)
picsplit /photos

# Disable orphan separation (old behavior)
picsplit /photos --separate-orphan=false

# Result structure:
# photos/
# └── 2024 - 1220 - 1400/
#     ├── PHOTO_01.JPG
#     ├── raw/              # Paired RAW
#     │   └── PHOTO_01.NEF
#     └── orphan/           # Unpaired RAW
#         └── PHOTO_02.NEF
```

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

[Unreleased]: https://github.com/sebastienfr/picsplit/compare/v2.8.0...HEAD
[2.8.0]: https://github.com/sebastienfr/picsplit/compare/v2.7.0...v2.8.0
[2.7.0]: https://github.com/sebastienfr/picsplit/compare/v2.6.0...v2.7.0
[2.6.0]: https://github.com/sebastienfr/picsplit/compare/v2.5.2...v2.6.0
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
