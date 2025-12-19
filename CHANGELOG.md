# Changelog

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
