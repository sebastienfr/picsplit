# picsplit - Intelligent Photo & Video Organizer

[![CI](https://github.com/sebastienfr/picsplit/actions/workflows/test.yml/badge.svg)](https://github.com/sebastienfr/picsplit/actions/workflows/test.yml)
[![CodeQL](https://github.com/sebastienfr/picsplit/actions/workflows/codeql.yml/badge.svg)](https://github.com/sebastienfr/picsplit/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sebastienfr/picsplit)](https://goreportcard.com/report/github.com/sebastienfr/picsplit)
[![GoDoc](https://godoc.org/github.com/sebastienfr/picsplit?status.svg)](https://godoc.org/github.com/sebastienfr/picsplit)
[![Latest Release](https://img.shields.io/github/v/release/sebastienfr/picsplit)](https://github.com/sebastienfr/picsplit/releases)
[![License](http://img.shields.io/badge/license-APACHE2-blue.svg)](https://github.com/sebastienfr/picsplit/blob/main/LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/sebastienfr/picsplit)](https://go.dev/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-blue)](https://github.com/sebastienfr/picsplit/releases)

> Automatically organize thousands of photos and videos by shooting date and GPS location using EXIF metadata and intelligent clustering algorithms.

---

## ⚠️ IMPORTANT DISCLAIMER

**Always work on a COPY of your photos!**

picsplit moves files on your filesystem. While extensively tested, you should NEVER run it directly on your only copy of important photos.

**Recommended workflow:**
1. ✅ Create a backup or working copy of your photos
2. ✅ Run picsplit on the copy (use `--mode validate` then `--mode dryrun` first to preview)
3. ✅ Verify the results before deleting originals

**USE AT YOUR OWN RISK.** The authors are not responsible for any data loss.

---

## Table of Contents

- [Key Features](#-key-features)
- [Quick Start](#-quick-start)
- [Use Cases](#-use-cases)
- [Roadmap](#%EF%B8%8F-roadmap)
- [How It Works](#-how-it-works)
- [Usage Guide](#-usage-guide)
  - [Basic Usage](#basic-usage)
  - [Advanced Features](#advanced-features)
  - [CLI Reference](#cli-reference)
- [Installation](#-installation)
- [Building from Source](#-building-for-developers)
- [FAQ](#-faq)
- [Contributing](#-contributing)
- [License](#-license)
- [Links](#-links)

---

## ⭐ Key Features

- 🕐 **Smart Time-Based Grouping**  
  Groups photos by shooting sessions using configurable time gaps (default: 30min)

- 📍 **GPS Location Clustering**  
  Organizes by geographic location first, then by time within each location (DBSCAN algorithm)

- 📷 **EXIF Metadata Support**  
  Uses real shooting dates from EXIF (photos), video metadata (MP4/MOV), and RAW+JPEG pairing

- 🔄 **Intelligent Merge**  
  Combine multiple photo sessions with interactive conflict resolution

- 🎨 **Custom File Extensions**  
  Add support for new file formats at runtime without recompiling

- 🔍 **Safe Preview Modes**  
  Validate mode (fast check) and dry-run mode (full simulation) let you preview changes before applying them

- 🗂️ **Smart Organization**  
  Automatically separates RAW files and videos into dedicated subfolders  
  *NEW in v2.6.0:* Orphan RAW files (without JPEG/HEIC) go to `orphan/` folder for easy cleanup

- 🌍 **Multi-Format Support**  
  JPG, HEIC, WebP, AVIF, NEF, CR2, DNG, ARW, MOV, MP4, and more

---

## 🎬 Quick Start

### Installation

**macOS/Linux:**
```bash
# Download latest release (replace <version>, <os>, <arch> with your values)
curl -LO https://github.com/sebastienfr/picsplit/releases/latest/download/picsplit_<version>_<os>_<arch>.tar.gz
tar -xzf picsplit_*.tar.gz
sudo mv picsplit /usr/local/bin/

# Verify installation
picsplit --version
```

**Or build from source:**
```bash
git clone https://github.com/sebastienfr/picsplit.git
cd picsplit
make build
```

See [Installation](#-installation) for detailed instructions.

---

### Basic Usage

```bash
# 1. ALWAYS work on a copy of your photos!
cp -r ~/Photos/DCIM ~/Photos/DCIM_backup

# 2. Fast validation (5s - check for issues)
picsplit --mode validate ~/Photos/DCIM_backup

# 3. Preview changes (dry run - full simulation)
picsplit --mode dryrun ~/Photos/DCIM_backup

# 4. Organize photos by time
picsplit ~/Photos/DCIM_backup
```

**Result:**
```
DCIM_backup/
├── 2024 - 1220 - 0915/    # Morning session
│   ├── photo1.jpg
│   ├── photo2.heic
│   └── raw/
│       └── photo1.nef
└── 2024 - 1220 - 1445/    # Afternoon session
    └── photo3.jpg
```

---

## 📚 Use Cases

### 🏖️ Organize vacation photos by location

**Scenario**: You returned from a 2-week European trip with 5,000+ photos from multiple cities.

**Challenge**: Photos are mixed (Paris, Rome, Barcelona) and you want to organize by destination.

**Solution:**
```bash
picsplit --gps --gps-radius 5000 ./europe-vacation
```

**Result:**
```
europe-vacation/
├── 48.8566N-2.3522E/          # Paris
│   ├── 2024 - 0615 - 1030/    # Eiffel Tower morning
│   └── 2024 - 0615 - 1530/    # Louvre afternoon
├── 41.9028N-12.4964E/         # Rome
│   └── 2024 - 0617 - 0900/    # Colosseum
└── 41.3851N-2.1734E/          # Barcelona
    └── 2024 - 0620 - 1100/    # Sagrada Familia
```

---

### 📸 Clean up camera DCIM folder

**Scenario**: Your camera's DCIM folder has 3 months of unsorted photos (events, daily life, trips).

**Challenge**: Need to split into separate folders by shooting session.

**Solution:**
```bash
# Use EXIF dates with 1-hour gap detection
picsplit --use-exif --delta 1h ./DCIM
```

**Result:** Each shooting session becomes a separate folder, even if files were imported at different times.

---

### 🔄 Merge duplicate imports

**Scenario**: You imported the same event twice from different cameras, now you have duplicate folders.

**Challenge**: `2024-1220-0900/` and `2024-1220-0915/` are actually the same event.

**Solution:**
```bash
# Fast validation first (check for conflicts)
picsplit merge --mode validate "2024-1220-0900" "2024-1220-0915" "2024-1220-birthday"

# Preview merge (full simulation)
picsplit merge --mode dryrun "2024-1220-0900" "2024-1220-0915" "2024-1220-birthday"

# Execute merge with conflict resolution
picsplit merge "2024-1220-0900" "2024-1220-0915" "2024-1220-birthday"
```

**Result:** One consolidated folder with interactive handling of duplicate filenames.

---

### 📷 Process RAW+JPEG workflow

**Scenario**: You shoot RAW+JPEG and want them organized separately but together.

**Challenge**: Keep RAW files accessible but separate from JPEGs for easier browsing.

**Solution:**
```bash
picsplit --use-exif ./photoshoot
```

**Result:**
```
photoshoot/
└── 2024 - 1220 - 1400/
    ├── photo1.jpg
    ├── photo2.jpg
    └── raw/
        ├── photo1.nef
        └── photo2.nef
```

---

### 🗑️ Photographer workflow: Culling RAW files

**Scenario**: You shoot in RAW+JPEG, cull bad JPEGs during review, but forget to delete corresponding RAW files.

**Challenge**: After organizing, your `raw/` folder contains a mix of "good" RAW (with JPEG) and "bad" RAW (without JPEG).

**Solution:**
```bash
# Orphan separation is enabled by default in v2.6.0+
picsplit ./photoshoot
```

**Result:**
```
photoshoot/
└── 2024 - 1220 - 1400/
    ├── PHOTO_01.JPG       # Kept during culling ✅
    ├── PHOTO_02.JPG       # Kept during culling ✅
    ├── raw/               # Good RAW files (with JPEG)
    │   ├── PHOTO_01.NEF
    │   └── PHOTO_02.NEF
    └── orphan/            # Bad RAW files (without JPEG)
        ├── PHOTO_03.NEF   # Delete this ❌
        └── PHOTO_04.NEF   # Delete this ❌
```

**Benefit**: Easily identify and delete unwanted RAW files without risking good ones.

**Supported pairs**: JPEG (`.jpg`, `.jpeg`) and HEIC (`.heic` - iPhone ProRAW+HEIC workflow)

**Disable feature:**
```bash
# Use old behavior (all RAW in raw/ folder)
picsplit --separate-orphan=false ./photoshoot
```

---

### 🎥 Multi-camera event coverage

**Scenario**: Wedding shot with 3 cameras (Canon, Sony, DJI drone) = mixed file types.

**Challenge**: Files have different timestamps, formats, and some lack GPS.

**Solution:**
```bash
# Add drone video formats + use EXIF from cameras
picsplit --video-ext dng --use-exif --delta 2h ./wedding-footage
```

**Result:** All cameras' footage organized by timeline, grouped into ceremony/reception/etc.

---

## 🗺️ Roadmap

picsplit continuously evolves with new features based on user feedback.

### ✅ v2.9.0 - Smart Folder Creation & GPS Improvements (Released - January 2026)

**Goal**: Make GPS clustering work with real-world iPhone backups and reduce folder clutter.

**Features delivered**:

- ✅ **GPS clustering for mixed file sets** ([#18](https://github.com/sebastienfr/picsplit/issues/18))  
  Fixed strict EXIF fallback bug that deleted GPS from all files  
  GPS coordinates now preserved even when some files lack EXIF  
  Makes `--gps` finally usable with iPhone backups (photos + screenshots + videos)

- ✅ **Minimum group size threshold** ([#17](https://github.com/sebastienfr/picsplit/issues/17))  
  `--min-group-size 5` prevents creating folders for small photo sets (default: 5)  
  Groups below threshold stay at parent root for cleaner structure  
  Works in both time-based and GPS modes

**Impact**:
- 🍎 iPhone backups now organize by location (70% photos with GPS + 30% screenshots without)
- 🌍 GPS clustering activates if **any** files have coordinates (not all-or-nothing)
- 📊 New GPS coverage analysis logs show extraction statistics
- 📂 Cleaner folder structure: Small groups (< 5 files) stay at root
- 🎯 Focus on real events: Only create folders for significant photo sets

**Breaking changes**: 
- Selective EXIF fallback (bugfix)
- Default MinGroupSize = 5 (was 0)
- See CHANGELOG for migration notes

---

### ✅ v2.8.0 - Duplicate Management & Code Quality (Released - January 2026)

**Goal**: Finalize duplicate management with automatic move and improve code quality.

**Features delivered**:

- ✅ **Automatic duplicate move** ([#16](https://github.com/sebastienfr/picsplit/issues/16))  
  `--move-duplicates` moves duplicates to `duplicates/` folder (recommended)

- ✅ **100% English codebase**  
  Complete translation of all French comments to improve maintainability

**All v2.8.0 features are implemented! 🎉**

---

### ✅ v2.7.0 - Logging & Observability (Released - January 2026)

**Goal**: Improve user feedback and observability during execution.

**Features delivered**:

- ✅ **Structured logging** ([#7](https://github.com/sebastienfr/picsplit/issues/7))  
  Migration to `log/slog` (Go stdlib) for typed and performant logs

- ✅ **Configurable log levels** ([#8](https://github.com/sebastienfr/picsplit/issues/8))  
  `--log-level debug|info|warn|error` + Text/JSON formats (`--log-format`)

- ✅ **Real-time progress bar** ([#9](https://github.com/sebastienfr/picsplit/issues/9))  
  Shows completion percentage with automatic TTY detection

- ✅ **Enhanced summary with metrics** ([#10](https://github.com/sebastienfr/picsplit/issues/10))  
  Detailed metrics (duration, throughput, stats by type, errors/warnings)

- ✅ **Typed errors with context** ([#11](https://github.com/sebastienfr/picsplit/issues/11))  
  Structured error messages with automatic correction suggestions

**Example of new summary**:
```
=== Processing Summary ===
Duration: 2m 35s
Files processed: 1,242 / 1,245 (99.8%)
File breakdown:
  - Photos: 980 (78.7%)
  - Videos: 165 (13.3%)
  - RAW: 100 (8.0%)
Groups created: 12
Disk usage: 24.5 GB moved, 158.0 MB/s throughput

❌ Critical errors (3):
  [Permission] read_file: /photos/IMG_001.jpg
    → chmod +r /photos/IMG_001.jpg

⚠ Warnings (15):
  [EXIF] No associated JPEG - using ModTime fallback

⚠ Operation completed with 3 errors
```

**Recommended workflow**:
```bash
# 1. Fast validation (5s) - detect critical issues
picsplit --mode validate /photos

# 2. Complete dry-run (30s) - simulate all moves
picsplit --mode dryrun /photos

# 3. Real execution (default mode)
picsplit /photos
# or explicitly: picsplit --mode run /photos
```

---

### 💡 Suggest a Feature

Have an idea to improve picsplit? [Open an issue](https://github.com/sebastienfr/picsplit/issues/new) to propose your suggestion!

**Full history**: Check the [CHANGELOG](./CHANGELOG.md) to see all changes since v1.0.

---

## 💡 How It Works

### Core Algorithm

```
┌─────────────────────────────────────────────────────────────┐
│ 1. SCAN FILES                                               │
│    • Recursively find all media files                       │
│    • Extract metadata (EXIF, video timestamps, GPS)         │
│    • Validate file types and extensions                     │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 2. GROUP FILES                                              │
│                                                              │
│   GPS Mode (--gps):                                         │
│   ├─ Spatial clustering (DBSCAN, configurable radius)       │
│   │  └─ Groups: Location1, Location2, NoLocation            │
│   └─ Time grouping within each location                     │
│      └─ Gap-based detection (configurable delta)            │
│                                                              │
│   Time-Only Mode (default):                                 │
│   └─ Gap-based detection (configurable delta)               │
│      └─ New group when gap > delta                          │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│ 3. ORGANIZE                                                 │
│    • Create folder structure                                │
│      - GPS mode: Location/Time                              │
│      - Time mode: Time only                                 │
│    • Move files to appropriate folders                      │
│    • Separate RAW/videos into subfolders (optional)         │
└─────────────────────────────────────────────────────────────┘
```

### Metadata Priority

1. **Photos**: EXIF `DateTimeOriginal` field
2. **RAW files**: Paired with associated JPEG (e.g., `.NEF` → `.JPG`)
3. **Videos**: MP4/MOV `creation_time` metadata
4. **Fallback**: File modification time (`ModTime`)

### Supported Formats

| Type | Extensions |
|------|------------|
| **Photos** | JPG, JPEG, HEIC, HEIF, WebP, AVIF |
| **RAW** | NEF, NRW, CR2, CRW, RW2, DNG, ARW, ORF, RAF |
| **Videos** | MOV, AVI, MP4 |

**+ Custom extensions** via `--photo-ext`, `--video-ext`, `--raw-ext` flags.

---

## 📖 Usage Guide

### Basic Usage

#### Organize photos by time
```bash
# Default: 30-minute gap detection
picsplit ./photos

# Custom time gap (1 hour)
picsplit --delta 1h ./photos

# Use file modification time instead of EXIF
picsplit --use-exif=false ./photos
```

#### Preview before applying
```bash
# Fast validation (5 seconds - no EXIF extraction)
picsplit --mode validate ./photos

# Dry run mode (full simulation - with EXIF extraction)
picsplit --mode dryrun ./photos

# Explicit run mode (same as default)
picsplit --mode run ./photos
```

#### Configure logging
```bash
# Debug mode with detailed logs
picsplit --log-level debug ./photos

# JSON output for parsing/monitoring
picsplit --log-format json ./photos > processing.log

# Quiet mode (errors only)
picsplit --log-level error ./photos
```

#### Don't separate RAW/videos
```bash
# Keep all files in same folder (no mov/raw subfolders)
picsplit --nomvmov --nomvraw ./photos
```

---

### Advanced Features

#### GPS Location Clustering

Group photos by where they were taken, then by when.

```bash
# Enable GPS mode with default 2km radius
picsplit --gps ./travel-photos

# Larger radius for countryside/road trips (5km)
picsplit --gps --gps-radius 5000 ./roadtrip

# Combine with custom time gap
picsplit --gps --delta 2h ./photos
```

**Output structure:**
```
photos/
├── 48.8566N-2.3522E/          # Paris
│   └── 2024 - 0615 - 1030/
└── NoLocation/                 # Files without GPS
    └── 2024 - 0616 - 0900/
```

**Note**: `NoLocation/` only appears when some files have GPS and others don't. If all files lack GPS, time-based folders are created at root level.

**GPS with Mixed Files** (v2.9.0+):

picsplit intelligently handles files with and without GPS metadata:
- **Photos with GPS**: Organized by location + time
- **Photos without GPS** (screenshots, edited photos): Grouped in `NoLocation/` by time
- **Videos**: Usually lack GPS, grouped by time with photos from same period

**Example**: iPhone backup (200 photos with GPS, 50 screenshots without GPS)
```bash
picsplit --gps ./iphone-backup
```

Result:
```
iphone-backup/
├── 50.6508N-3.0735E/          # Lille (150 photos)
│   ├── 2020 - 1106 - 2023/
│   └── 2020 - 1107 - 1145/
├── 48.8707N-2.3390E/          # Paris (50 photos)
│   └── 2021 - 0520 - 1600/
└── NoLocation/                 # Screenshots + videos
    └── 2020 - 1108 - 0900/
```

**Key behavior**:
- ✅ GPS clustering activates if **any** files have GPS coordinates
- ✅ Files without GPS are **not skipped** - they're grouped separately
- ✅ Each file uses its own extracted metadata (GPS preserved even if some files lack EXIF)

---

#### Minimum Group Size

Reduce folder clutter by setting a threshold for folder creation.

**Problem**: Small photo sets (1-2 screenshots, random photos) create too many folders.

**Solution**: Groups below threshold stay at parent root instead of creating folders.

```bash
# Default: Groups with < 5 files stay at root
picsplit ./photos

# Custom threshold: Groups with < 10 files stay at root
picsplit --min-group-size 10 ./photos

# No filtering: Create folders for all groups (even single files)
picsplit --min-group-size 0 ./photos
```

**Example**: Default threshold (5 files)

```
Input: 100 photos creating 15 groups (sizes: 20, 15, 8, 6, 5, 4, 3, 2, 2, 1, 1, 1, 1, 1, 1)

Output:
photos/
├── 2024 - 0615 - 1200/  # 20 files ✅
├── 2024 - 0618 - 0900/  # 15 files ✅
├── 2024 - 0620 - 1400/  # 8 files ✅
├── 2024 - 0622 - 1000/  # 6 files ✅
├── 2024 - 0624 - 1600/  # 5 files ✅
├── screenshot1.png       # Small group (4 files) at root
├── screenshot2.png
├── screenshot3.png
├── screenshot4.png
├── random1.jpg           # Small group (3 files) at root
├── random2.jpg
└── random3.jpg
    ... (remaining 7 small groups at root)
```

**GPS Mode Behavior**:

Groups below threshold stay at **location root** (not basePath root).

```bash
picsplit --gps --min-group-size 5 ./travel

# Result:
travel/
├── 48.8566N-2.3522E/          # Paris
│   ├── 2024 - 0615 - 1200/    # Large group (8 photos)
│   ├── photo1.jpg             # Small group (2 photos) at Paris/ root
│   └── photo2.jpg
└── 50.6508N-3.0735E/          # Lille
    ├── 2024 - 0620 - 0900/    # Large group (12 photos)
    ├── quick1.jpg             # Small group (1 photo) at Lille/ root
    └── ...
```

**Benefits**:
- ✅ Cleaner structure: Focus on real events (>= 5 photos)
- ✅ Easy access: Small sets at root (no deep diving)
- ✅ Flexible: Adjust threshold per use case

**Breaking change (v2.9.0)**: Default threshold is 5 (was 0 in v2.8.0). Set `--min-group-size 0` for old behavior.

---

#### Merge Folders

Combine multiple time-based folders into one.

```bash
# Fast validation (check for conflicts without processing)
picsplit merge --mode validate folder1 folder2 merged-folder

# Preview merge operations (full simulation)
picsplit merge --mode dryrun folder1 folder2 merged-folder

# Interactive merge (prompts for conflicts)
picsplit merge folder1 folder2 merged-folder

# Force overwrite all conflicts
picsplit merge --force folder1 folder2 merged
```

**Conflict resolution options:**
- `[r]` Rename: Generate unique filename
- `[s]` Skip: Keep existing file
- `[o]` Overwrite: Replace with new file
- `[a]` Apply to all: Use same action for remaining conflicts
- `[q]` Quit: Abort merge

**Limitations:**
- ❌ Cannot merge GPS location folders (e.g., `48.8566N-2.3522E/`)
- ✅ Only time-based folders (e.g., `2024 - 1220 - 0900/`)

---

#### Custom File Extensions

Add support for additional file formats at runtime.

```bash
# Add custom photo format
picsplit --photo-ext png ./photos

# Multiple custom extensions (comma-separated)
picsplit --photo-ext png,bmp --video-ext mkv,webm ./photos

# Works with merge command
picsplit merge --raw-ext rwx folder1 folder2 merged
```

**Rules:**
- Extensions are **additive** (defaults always included)
- Case-insensitive: `.PNG` = `.png`
- Max 8 characters, alphanumeric only
- ⚠️ **Flags must come BEFORE paths**: `picsplit --photo-ext png ./data` ✅

---

#### Automatic Empty Directory Cleanup

Automatically remove empty directories after organizing files.

```bash
# Basic cleanup (with confirmation prompt)
picsplit --cleanup-empty-dirs ./photos

# Skip confirmation (useful for automation)
picsplit --cleanup-empty-dirs --force ./photos

# Ignore additional custom files beyond defaults (.DS_Store, Thumbs.db, etc.)
picsplit --cleanup-empty-dirs --cleanup-ignore ".picasa.ini,.nomedia" ./photos

# Combine with other options
picsplit --gps --cleanup-empty-dirs --force ./photos
```

**How it works:**

1. **Multi-pass bottom-up removal**: Processes directories from deepest to shallowest
   ```
   Pass 1: Remove /photos/import/january/ (deepest)
   Pass 2: Remove /photos/import/ (now empty after Pass 1)
   ```

2. **Smart file ignoring**: Directories containing ONLY ignored files are considered empty
   - **Default ignored**: `.DS_Store`, `Thumbs.db`, `desktop.ini`, `._.DS_Store`
   - **Custom ignored**: Via `--cleanup-ignore` (comma-separated list)
   - Ignored files are automatically deleted before removing the directory

3. **Interactive confirmation** (unless `--force`):
   ```
   Found empty directories (count=5):
     - /photos/import/january/
     - /photos/import/february/
     ...
   
   Do you want to remove these empty directories? [y/o/N]:
   ```
   - Accepts: `y`, `yes`, `o`, `oui` (case-insensitive)

4. **Protected directories**: Never scanned or removed: `.git`, `.svn`, `.hg`, `node_modules`

**Mode behavior:**
- ✅ `--mode validate`: Cleanup is skipped entirely
- ⚠️ `--mode dryrun`: Shows currently empty directories only (not directories that would become empty after file moves)
- ✅ `--mode run`: Full multi-pass cleanup with actual removal

**Limitation:**

In `dryrun` mode, the cleanup preview only detects directories that are **currently empty**, not directories that **would become empty** after the simulated file moves. This is a known limitation to keep the implementation simple and performant.

**Example:**
```bash
# Before organizing
photos/
├── IMG_001.jpg        # File at root
├── import/            # Already empty directory
│   └── old/           # Nested empty directory
└── import2/           # Directory with file (NOT empty yet)
    └── IMG_002.jpg

# Dryrun mode shows:
# ✅ Would remove: photos/import/old/ (currently empty)
# ✅ Would remove: photos/import/ (currently empty)
# ❌ Does NOT show: photos/import2/ (not empty yet, but will become empty after IMG_002.jpg is moved)

# After organizing (run mode)
photos/
├── 2024 - 1220 - 0900/
│   └── IMG_001.jpg
└── 2024 - 1220 - 1015/
    └── IMG_002.jpg
    # import/, import/old/ AND import2/ all removed
    # - import/ and import/old/ were already empty
    # - import2/ became empty after IMG_002.jpg was moved
```

**Use cases:**
- Clean up empty import folders after organizing photos
- Remove directories left behind after culling/deleting photos
- Maintain a clean directory structure automatically

---

#### Error Handling

Control how picsplit behaves when encountering errors during processing.

```bash
# Default: stop at first error
picsplit ./photos

# Continue processing despite errors (collects all errors, reports at end)
picsplit --continue-on-error ./photos

# Short alias
picsplit --coe ./photos

# Combine with dry run to preview error handling
picsplit --coe --mode dryrun ./photos
```

**Behavior:**

**Default mode** (`--continue-on-error=false`):
- ❌ Stops immediately when encountering a critical error (IO, permission, validation)
- ✅ Returns error code 1
- ⚠️ Non-critical errors (EXIF/metadata issues) use fallback and continue

**Continue-on-error mode** (`--continue-on-error=true`):
- ✅ Collects all errors and continues processing remaining files
- ✅ All errors displayed in summary at the end
- ✅ Returns error code 1 if any critical errors occurred
- ✅ Processes as many files as possible despite failures

**Use cases:**
- Large photo libraries where you want to process everything possible
- Debugging: collect all errors in one run instead of fixing them one by one
- Mixed quality sources where some files may be corrupted

---

#### Duplicate Detection & Management

Detect and manage duplicate files based on binary content (SHA256 hash) with three modes.

```bash
# Mode 1: Detection only (warns about duplicates but processes them anyway)
picsplit --detect-duplicates ./photos

# Mode 2: Skip duplicates (leaves them in source folder)
picsplit --detect-duplicates --skip-duplicates ./photos

# Mode 3: Move duplicates to dedicated folder (RECOMMENDED ⭐)
picsplit --detect-duplicates --move-duplicates ./photos

# Short aliases
picsplit --dd --md ./photos

# Combine with other flags
picsplit --dd --md --cleanup-empty-dirs ./photos
```

**How it works:**

1. **Size-based pre-filtering** (optimization):
   - Groups files by size first
   - Only hashes files that share the same size with others
   - Files with unique sizes are automatically non-duplicates (10x faster)

2. **SHA256 hashing**:
   - Calculates SHA256 hash of file content
   - First file with a hash becomes the "original"
   - Subsequent files with same hash are marked as duplicates

3. **Three modes**:
   - **Detection-only** (`--detect-duplicates`): Warns but processes all files
   - **Skip mode** (`--detect-duplicates --skip-duplicates`): Skips duplicates (remain in source)
   - **Move mode** (`--detect-duplicates --move-duplicates`): Moves duplicates to `duplicates/` folder ⭐

**Output examples:**

Detection-only mode:
```
⚠ Duplicates detected (processed anyway) (count=3):
  - duplicate detected file=IMG_001_copy.jpg original=IMG_001.jpg
  - duplicate detected file=VID_002 (1).mp4 original=VID_002.mp4
  ...
```

Skip mode:
```
ℹ Duplicates skipped (count=3):
  - skipped duplicate file=IMG_001_copy.jpg original=IMG_001.jpg
  - skipped duplicate file=VID_002 (1).mp4 original=VID_002.mp4
  ...
```

Move mode (RECOMMENDED):
```
ℹ Duplicates moved (count=3):
  - moved duplicate file=IMG_001_copy.jpg to=duplicates/ original=IMG_001.jpg
  - moved duplicate file=VID_002 (1).mp4 to=duplicates/ original=VID_002.mp4
  ...
```

**Result structure with move mode:**
```
photos/
├── duplicates/              ✅ Duplicates isolated here
│   ├── IMG_001_copy.jpg
│   └── VID_002 (1).mp4
├── 2024-01-15_Event1/       ✅ Originals organized
│   └── IMG_001.jpg
└── 2024-01-20_Event2/
    └── VID_002.mp4
```

**Performance:**

- Without optimization: ~200 MB/s hashing speed
- With size pre-filtering: 10x faster (only hashes potential duplicates)
- Example: 1000 files (50 size groups) → ~2.5s instead of ~25s

**Use cases:**
- Clean up duplicate imports from multiple cameras
- Detect accidental re-imports of same photo session
- Isolate duplicates for manual review before deletion
- Identify backup copies mixed with originals

**Validation:**
```bash
# Error: --skip-duplicates requires --detect-duplicates
picsplit --skip-duplicates ./photos
# ❌ Error: --skip-duplicates requires --detect-duplicates

# Error: --move-duplicates requires --detect-duplicates
picsplit --move-duplicates ./photos
# ❌ Error: --move-duplicates requires --detect-duplicates

# Error: --skip-duplicates and --move-duplicates are mutually exclusive
picsplit --dd --sd --md ./photos
# ❌ Error: --skip-duplicates and --move-duplicates are mutually exclusive

# Correct usage
picsplit --detect-duplicates --move-duplicates ./photos
# ✅ Works
```

**Recommended workflow:**
```bash
# 1. Preview what would be moved (dry run)
picsplit --dd --md --mode dryrun ./photos

# 2. Execute move
picsplit --dd --md ./photos

# 3. Review duplicates/ folder content

# 4. Delete duplicates if confirmed
rm -rf ./photos/duplicates/
```

---

### CLI Reference

#### Main Command

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--help` | `-h` | - | Show help message |
| `--version` | `-v` | - | Display version information |
| `--mode` | `-m` | `run` | Execution mode: `validate` (fast check), `dryrun` (simulate), `run` (execute) |
| `--use-exif` | `-ue` | `true` | Use EXIF metadata for dates |
| `--delta` | `-d` | `30m` | Time gap between sessions (e.g., `1h`, `45m`) |
| `--gps` | `-g` | `false` | Enable GPS location clustering |
| `--gps-radius` | `-gr` | `2000` | GPS clustering radius in meters |
| `--continue-on-error` | `--coe` | `false` | Continue processing despite errors (collect all errors instead of stopping at first failure) |
| `--cleanup-empty-dirs` | `--ced` | `false` | Automatically remove empty directories after processing |
| `--cleanup-ignore` | `--ci` | - | Additional files to ignore when checking if directory is empty (comma-separated, e.g., `.picasa.ini,.nomedia`) |
| `--detect-duplicates` | `--dd` | `false` | Detect duplicate files via SHA256 hash |
| `--skip-duplicates` | `--sd` | `false` | Skip duplicate files automatically (requires `--detect-duplicates`) |
| `--move-duplicates` | `--md` | `false` | Move duplicates to `duplicates/` folder (requires `--detect-duplicates`, mutually exclusive with `--skip-duplicates`) |
| `--force` | `-f` | `false` | Skip all confirmation prompts (cleanup, merge, etc.) |
| `--log-level` | - | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | - | `text` | Log format: `text` or `json` |
| `--nomvmov` | `-nmm` | `false` | Don't separate videos into `mov/` folder |
| `--nomvraw` | `-nmr` | `false` | Don't separate RAW into `raw/` folder |
| `--separate-orphan` | `-so` | `true` | Separate unpaired RAW files to `orphan/` folder |
| `--photo-ext` | `-pext` | - | Add custom photo extensions (e.g., `png,bmp`) |
| `--video-ext` | `-vext` | - | Add custom video extensions (e.g., `mkv`) |
| `--raw-ext` | `-rext` | - | Add custom RAW extensions (e.g., `rwx`) |

#### Merge Command

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `run` | Execution mode: `validate`, `dryrun`, `run` |
| `--force` | `false` | Auto-overwrite conflicts |
| `--log-level` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `--log-format` | `text` | Log format: `text` or `json` |

**Full help:**
```bash
picsplit --help
picsplit merge --help
```

---

## 🔧 Installation

### Pre-built Binaries

Download the latest release for your platform:

**[📥 Download Latest Release](https://github.com/sebastienfr/picsplit/releases/latest)**

**Available platforms:**
- macOS (Intel & Apple Silicon)
- Linux (amd64 & arm64)
- Windows (amd64)

**Installation (macOS/Linux):**
```bash
# Download and extract (replace <version>, <os>, <arch> with your values)
curl -LO https://github.com/sebastienfr/picsplit/releases/latest/download/picsplit_<version>_<os>_<arch>.tar.gz
tar -xzf picsplit_*.tar.gz

# Install system-wide
sudo mv picsplit /usr/local/bin/

# Verify installation
picsplit --version
```

**Example for macOS Apple Silicon:**
```bash
curl -LO https://github.com/sebastienfr/picsplit/releases/download/v2.5.2/picsplit_2.5.2_darwin_arm64.tar.gz
tar -xzf picsplit_2.5.2_darwin_arm64.tar.gz
sudo mv picsplit /usr/local/bin/
picsplit --version
```

**Installation (Windows):**
1. Download the `.zip` file for your architecture
2. Extract to a folder (e.g., `C:\Program Files\picsplit\`)
3. Add to PATH (optional)

---

### Build from Source

**Requirements:**
- Go 1.25 or later
- Git

**Steps:**
```bash
git clone https://github.com/sebastienfr/picsplit.git
cd picsplit
make build

# Binary created at ./bin/picsplit
./bin/picsplit --version
```

**Install to GOPATH/bin:**
```bash
make install
# Installs picsplit to $GOPATH/bin (usually ~/go/bin)
# Make sure $GOPATH/bin is in your PATH
```

---

## 🏗️ Building (For Developers)

### Version Management

picsplit uses automatic version detection via Git tags:

| Context | Version Displayed |
|---------|-------------------|
| Clean tag (`v2.5.2`) | `2.5.2` |
| Dirty tree (uncommitted changes) | `2.5.2-dev` |
| Between tags | `2.5.2-dev` |
| No tags | `dev` |

### Build Commands

```bash
# Build with automatic version detection
make build

# Check detected version
make version

# Run tests
make test

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make coverage-html

# Lint code
make lint-ci

# Clean build artifacts
make clean
```

### Development Workflow

```bash
# 1. Make changes
# 2. Format code
make format

# 3. Run tests
make test

# 4. Check coverage
make test-coverage

# 5. Lint
make lint-ci

# 6. Build
make build
```

### Release Process (Maintainers)

```bash
# Test release locally
make release-snapshot

# Create tag
git tag -a v2.5.3 -m "Release v2.5.3: Description"

# Push tag (triggers CI/CD)
git push origin v2.5.3
```

GitHub Actions will automatically build and publish the release.

**Technology Stack:**
- [Go 1.25](https://golang.org) - Programming language
- [urfave/cli v2](https://github.com/urfave/cli) - CLI framework
- [log/slog](https://pkg.go.dev/log/slog) - Structured logging (Go stdlib)
- [progressbar/v3](https://github.com/schollz/progressbar) - Real-time progress display
- [goexif](https://github.com/rwcarlsen/goexif) - EXIF parsing
- [go-mp4](https://github.com/abema/go-mp4) - Video metadata extraction

---

## ❓ FAQ

<details>
<summary><b>Does picsplit modify my original files?</b></summary>

No, picsplit only **moves** files between folders. It does not modify file contents, EXIF data, or timestamps. However, always work on a copy first!

</details>

<details>
<summary><b>What happens if picsplit crashes mid-operation?</b></summary>

File moves are atomic operations. If picsplit crashes:
- Files already moved remain in their new location
- Files not yet processed remain in original location
- No partial/corrupted files (either moved or not)

Run picsplit again on the same folder - it will skip already organized files.

</details>

<details>
<summary><b>Can I undo picsplit's organization?</b></summary>

There's no built-in undo. Best practice:
1. Always work on a copy
2. Use `--mode validate` for fast check, then `--mode dryrun` to preview before applying
3. Keep backups of your originals

</details>

<details>
<summary><b>How does picsplit handle duplicate filenames?</b></summary>

**During split:** Files with same name in different source folders are moved to time-appropriate folders (unlikely to conflict).

**During merge:** Interactive conflict resolution lets you rename, skip, or overwrite. Use `--force` to auto-overwrite.

</details>

<details>
<summary><b>Why aren't my RAW files being detected?</b></summary>

Supported RAW formats: NEF, NRW, CR2, CRW, RW2, DNG, ARW, ORF, RAF

For other formats, use `--raw-ext`:
```bash
picsplit --raw-ext rwl,iiq ./photos
```

</details>

<details>
<summary><b>Does picsplit work with network drives / NAS?</b></summary>

Yes, but performance depends on network speed. For large photo libraries:
1. Copy to local disk first
2. Organize locally
3. Copy organized structure back to NAS

</details>

**Have more questions?** [Open an issue](https://github.com/sebastienfr/picsplit/issues/new)

---

## 🤝 Contributing

Contributions are welcome! Please:

1. **Report bugs**: [Open an issue](https://github.com/sebastienfr/picsplit/issues/new)
2. **Request features**: [Open an issue](https://github.com/sebastienfr/picsplit/issues/new)
3. **Submit PRs**: 
   - Fork the repository
   - Create a feature branch
   - Run tests (`make test`)
   - Ensure coverage doesn't decrease
   - Submit PR with clear description

**Development:**
```bash
make build        # Build
make test         # Run tests
make test-coverage # Check coverage
make lint-ci      # Lint code
```

See [Building](#-building-for-developers) section for more details.

---

## 📄 License

Apache License 2.0 - see [LICENSE](./LICENSE) file for details.

---

## 🔗 Links

- **[📝 Changelog](./CHANGELOG.md)** - Full version history
- **[🚀 Releases](https://github.com/sebastienfr/picsplit/releases)** - Download binaries
- **[🐛 Issues](https://github.com/sebastienfr/picsplit/issues)** - Report bugs or request features
- **[📊 CI/CD](https://github.com/sebastienfr/picsplit/actions)** - Build status

---

<div align="center">

**Made with ❤️ for photographers and digital hoarders**

</div>
