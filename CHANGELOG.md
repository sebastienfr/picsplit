# Changelog

## [2.4.0] - 2024-12-20

### ✨ Added
- **Merge command**: Combine multiple time-based folders into one
- Interactive conflict resolution with 5 options:
  - `[r]` Rename: Generate unique filename to avoid conflict
  - `[s]` Skip: Keep target version, don't move source file
  - `[o]` Overwrite: Replace target with source file
  - `[a]` Apply to all: Apply chosen action to remaining conflicts
  - `[q]` Quit: Abort merge operation immediately
- `--force` flag: Automatically overwrite all conflicts without prompting
- `--dryrun` flag: Preview merge operations without moving files
- Media folder validation: Only folders containing media files can be merged
- Automatic source folder cleanup after successful merge
- Structure preservation: `mov/` and `raw/` subdirectories are maintained

### 🔧 Technical Changes
- New file `handler/merger.go` with merge logic (~430 lines)
- New file `handler/merger_test.go` with comprehensive test suite (~680 lines, 15+ test cases)
- New `MergeConfig` struct with `SourceFolders`, `TargetFolder`, `Force`, `DryRun` fields
- New `FileConflict` struct for conflict tracking
- Validation functions:
  - `isMediaFile()`: Checks if file extension is a supported media type
  - `isMediaFolder()`: Validates folder contains only media files + mov/raw subdirs
  - `validateMergeFolders()`: Pre-merge validation of sources and target
- Conflict handling functions:
  - `detectConflict()`: Checks for file conflicts at target path
  - `askUserConflictResolution()`: Interactive stdin prompts for conflict resolution
  - `generateUniqueName()`: Creates unique filenames (photo.jpg → photo_1.jpg → photo_2.jpg)
- File operations:
  - `collectFilesRecursive()`: Recursively collects all files from a directory
  - `Merge()`: Main orchestration function with stats tracking

### 📝 Usage
```bash
# Basic merge with interactive conflict resolution
picsplit merge "2025 - 0616 - 0945" "2025 - 0616 - 1430" "2025 - 0616 - merged"

# Force mode: auto-overwrite all conflicts
picsplit merge folder1 folder2 folder3 target --force

# Dry run: preview merge operations
picsplit merge source1 source2 target --dryrun -v
```

### ⚙️ Validation Rules
**Media folder requirements**:
- Must contain ONLY media files (photos: jpg/jpeg/heic/webp/avif, videos: mov/mp4/avi, raw: nef/cr2/dng/arw/orf/raf)
- May contain ONLY `mov/` and `raw/` subdirectories (case-insensitive)
- Other file types or subdirectories will cause validation errors
- Empty folders are allowed (will be merged and cleaned up)

**Restrictions**:
- GPS location folders (e.g., `48.8566N-2.3522E`) cannot be merged
  - These contain nested time-based subfolders
  - Would violate media folder validation (non-media subdirectories)
- Only time-based folders (e.g., `2025 - 0616 - 0945`) are mergeable

### 🧪 Test Coverage
- `handler/merger.go`: 77.6% coverage (15+ test cases)
  - `isMediaFile()`: 100% coverage
  - `isMediaFolder()`: 85.7% coverage (7 test scenarios)
  - `validateMergeFolders()`: 81.2% coverage
  - `Merge()`: 77.6% coverage (main function)
  - Untested: `askUserConflictResolution()` (requires interactive stdin)
- Test scenarios:
  - Basic 2-folder merge
  - Multiple folder merge (4 sources)
  - Empty source folders
  - Target folder already exists
  - mov/raw/ structure preservation
  - No conflicts
  - Conflicts with force flag
  - Dry-run mode
  - Dry-run with conflicts
  - Media folder validation (valid/invalid cases)
- Overall project coverage: 72.2%

### 📚 Documentation
- README.md updated with merge command section
- Usage examples with common scenarios
- Important notes about GPS folder restrictions
- Roadmap updated to mark v2.4.0 as completed

**Closes** [#4](https://github.com/sebastienfr/picsplit/issues/4)

---

## [2.3.0] - 2024-12-20

### ✨ Added
- **GPS location clustering**: Group photos by geographic location first, then by time within each location
- Spatial-first algorithm: DBSCAN-like clustering with configurable radius (default: 2km)
- Folder structure: `<LocationCoords>/<TimeFolder>/` (e.g., `48.8566N-2.3522E/2025 - 0616 - 0945/`)
- Special `NoLocation/` folder for files without GPS coordinates
- New CLI flags:
  - `--gps` / `-g`: Enable GPS location clustering (default: false, opt-in feature)
  - `--gps-radius` / `-gr`: Set clustering radius in meters (default: 2000m = 2km)
- 100% test coverage for GPS clustering and geographic calculation modules

### 🔧 Technical Changes
- New file `handler/geo.go`: Geographic calculations (Haversine distance, centroid, coordinate formatting)
- New file `handler/clustering.go`: DBSCAN-like spatial clustering and time-based grouping
- New `GPSCoord` struct for latitude/longitude coordinates
- New `LocationCluster` struct for geographic file grouping
- Updated `Config` struct with `UseGPS` and `GPSRadius` fields
- Config validation: GPS radius must be positive when GPS enabled
- Enhanced `Split()` function with dual-mode operation (GPS or time-only)

### 📝 Algorithm Details
**GPS Clustering Mode** (when `--gps` enabled):
1. **Spatial clustering**: Files with GPS coordinates are grouped by proximity (DBSCAN-like)
   - Files within `GPSRadius` meters belong to the same location cluster
   - Each cluster's centroid becomes the folder name (e.g., `48.8566N-2.3522E`)
2. **Temporal grouping**: Within each location cluster, files are grouped by time gaps
   - Same `--delta` parameter as time-only mode (default: 30 minutes)
   - Results in nested structure: `<Location>/<TimeGroup>/`
3. **NoLocation handling**: Files without GPS coordinates are placed in `NoLocation/<TimeGroup>/`

**Time-Only Mode** (default, `--gps` disabled):
- Backward compatible with v2.1.0 and v2.2.0 behavior
- Files grouped purely by temporal gaps (no GPS consideration)

**Geographic Calculations**:
- Haversine formula for accurate distance calculation on Earth's surface
- Earth radius: 6,371,000 meters (mean radius)
- Centroid calculation: Simple arithmetic mean of coordinates (suitable for small clusters)
- Coordinate formatting: `<Lat>N/S-<Lon>E/W` (e.g., `48.8566N-2.3522E`, `51.5074N-0.1278W`)

### ⚙️ Configuration
**Default values**:
- `UseGPS`: `false` (opt-in to maintain backward compatibility)
- `GPSRadius`: `2000.0` meters (2km, typical city exploration range)

**Usage examples**:
```bash
# Enable GPS clustering with default 2km radius
picsplit --gps ./photos

# Custom 5km radius for larger geographic areas
picsplit --gps --gps-radius 5000 ./photos

# Dry run to preview GPS clustering structure
picsplit --gps --dryrun ./photos

# Combine with EXIF and other options
picsplit --gps --use-exif --delta 1h ./photos
```

### 🧪 Test Coverage
- `handler/geo.go`: 100% coverage (19 test cases)
  - Distance calculations (Paris-London, NY-LA, equator crossing, etc.)
  - Centroid calculations (triangle, midpoint, empty, negative coords)
  - Coordinate formatting (all hemispheres, high precision)
  - Radians conversion
- `handler/clustering.go`: 100% coverage (11 test cases)
  - Location clustering (multiple clusters, single cluster, no GPS, distant clusters)
  - Time-based grouping within locations
  - NoLocation folder naming
- Overall project coverage: 73.8%

---

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
