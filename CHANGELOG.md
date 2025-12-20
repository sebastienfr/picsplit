# Changelog

## [2.2.0] - 2024-12-20

### ✨ Added
- **EXIF metadata support**: Dates are now extracted from EXIF DateTimeOriginal for photos
- **Video metadata support**: Extraction from MP4/MOV creation_time metadata
- **RAW+JPEG pairing**: RAW files automatically share EXIF data from associated JPEG files
- **Strict fallback mode**: If any file in a batch lacks valid EXIF metadata, all files fall back to ModTime
- New CLI flag: `--use-exif` (default: true) to enable/disable EXIF metadata extraction
- GPS coordinate extraction from EXIF (preparation for v2.3.0 location clustering)

### 🔧 Technical Changes
- New file `handler/exif.go` with EXIF/video metadata extraction functions
- New `FileMetadata` struct replaces raw `os.FileInfo` usage internally
- Added `DateSource` enum to track origin of date (ModTime, EXIF, VideoMeta)
- Date validation: dates must be between 1990 and now+1 day
- Naive timezone handling (no conversion, treats EXIF dates as-is)
- Updated internal functions to use `FileMetadata` instead of `os.FileInfo`

### 📦 Dependencies
- Added `github.com/rwcarlsen/goexif` for EXIF parsing
- Added `github.com/abema/go-mp4` for video metadata extraction

### 📝 Algorithm Details
**Date Priority Order**:
1. **Photos**: EXIF DateTimeOriginal (if valid and available)
2. **RAW files**: Uses EXIF from associated JPEG file (e.g., PHOTO_01.NEF → PHOTO_01.JPG)
3. **Videos**: MP4/MOV creation_time metadata
4. **Fallback**: File modification time (ModTime)

**Strict Mode Behavior**:
- If ANY file in the batch has invalid or missing EXIF metadata, ALL files revert to ModTime
- Ensures consistent dating across entire event groups
- Invalid dates include: before 1990, more than 1 day in the future, or parsing errors

---

## [2.1.0] - 2024-12-19

### ✨ Improved
- **Gap-based event detection algorithm**: Photos and videos are now grouped by temporal gaps instead of rounded timestamps
- Folder names use the exact timestamp of the first file in each group (no more rounding)
- Better handling of continuous photo sessions that span across hour boundaries
- More natural event detection based on actual shooting patterns

### 🔧 Technical Changes
- New functions: `collectMediaFiles()`, `sortFilesByModTime()`, `groupFilesByGaps()`, `processGroup()`
- Removed obsolete functions: `listDirectories()`, `findOrCreateDatedFolder()`, `processFiles()`
- Refactored `Split()` to use gap-based grouping algorithm
- Updated test suite with comprehensive tests for new grouping logic

### 📝 Algorithm Explanation
**Previous (v2.0.0)**: Files were rounded to the nearest `delta` interval
- Example: Files at 09:45, 10:05, 10:25 with delta=1h would create folders "2024 - 0216 - 1000" and "2024 - 0216 - 1000"

**New (v2.1.0)**: Files are grouped by gaps between consecutive timestamps
- Files are sorted chronologically
- A new group starts when the gap between consecutive files exceeds `delta`
- Folder is named after the first file's exact timestamp
- Example: Same files create a single folder "2024 - 0216 - 0945"

**Migration Note**: Running v2.1.0 on v2.0.0-organized folders will create new folders. Best practice is to reorganize from original source files.

---

## [2.0.0] - 2024-12-19

### 🚀 Breaking Changes
- Migrated from `urfave/cli` v1 to v2
- Updated Go version requirement to 1.25
- Internal API changes in `handler.Split()` signature (now uses `Config` struct)

### ✨ Added
- Support for modern image formats: HEIC, HEIF, WebP, AVIF
- Support for additional RAW formats: DNG (Adobe), ARW (Sony), ORF (Olympus), RAF (Fujifilm)
- Comprehensive test suite with 80%+ code coverage using Go 1.15+ `t.TempDir()` API
- New `handler.Config` struct for better configuration management
- Improved error handling with error wrapping (`fmt.Errorf` with `%w`)
- Better dry run mode with clear `[DRY RUN]` prefix in logs

### 🔄 Changed
- Updated `github.com/sirupsen/logrus` from v1.3.0 to v1.9.3
- Updated `github.com/urfave/cli` from v1.20.0 to v2.27.5
- Replaced deprecated `file.Readdir()` with modern `os.ReadDir()`
- Refactored code to use named constants instead of magic strings
- Changed file move logs from `Warn` to `Info` level
- Split monolithic `processFiles()` into smaller, testable functions (`processPicture`, `processMovie`)
- Improved Makefile with coverage targets and better organization

### 🗑️ Removed
- Removed `etc/mktest.sh` (replaced by Go test fixtures using `t.TempDir()`)

### 🐛 Fixed
- Improved error messages with context
- Better validation of configuration parameters

---

## v0.0.1 [unreleased]
- 17/04/15 doc(all): project start, refactoring from an existing shell script to Golang (SFR)
