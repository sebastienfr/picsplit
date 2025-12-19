# Changelog

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
