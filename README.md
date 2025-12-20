# Picture splitter (picsplit)

[![CI](https://github.com/sebastienfr/picsplit/actions/workflows/test.yml/badge.svg)](https://github.com/sebastienfr/picsplit/actions/workflows/test.yml)
[![CodeQL](https://github.com/sebastienfr/picsplit/actions/workflows/codeql.yml/badge.svg)](https://github.com/sebastienfr/picsplit/actions/workflows/codeql.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sebastienfr/picsplit)](https://goreportcard.com/report/github.com/sebastienfr/picsplit)
[![GoDoc](https://godoc.org/github.com/sebastienfr/picsplit?status.svg)](https://godoc.org/github.com/sebastienfr/picsplit)
[![Latest Release](https://img.shields.io/github/v/release/sebastienfr/picsplit)](https://github.com/sebastienfr/picsplit/releases)
[![Software License](http://img.shields.io/badge/license-APACHE2-blue.svg)](https://github.com/sebastienfr/picsplit/blob/master/LICENSE)

## License
Apache License version 2.0.

## Description
Picture splitter (`picsplit`) is meant to process digital camera DCIM folder
in order to split contiguous files (pictures and movies) in dedicated subfolders.

**Smart date detection (v2.2.0+):**
Picsplit uses EXIF metadata when available to determine file dates:
- **Photos**: EXIF DateTimeOriginal field
- **RAW files**: Shares EXIF from associated JPEG (e.g., PHOTO_01.NEF → PHOTO_01.JPG)
- **Videos**: MP4/MOV creation_time metadata
- **Fallback**: File modification time (ModTime)

**GPS location clustering (v2.3.0+):**
When GPS mode is enabled (`--gps`), picsplit groups files by geographic location first, 
then by time within each location. Files are clustered using a DBSCAN-like algorithm 
with a configurable radius (default: 2km). Each location cluster gets a folder named 
after its centroid coordinates (e.g., `48.8566N-2.3522E`), containing time-based 
subfolders. Files without GPS coordinates are placed in a special `NoLocation/` folder.

**Gap-based event detection (v2.1.0+):**
Files are sorted chronologically and grouped by temporal gaps. When the time gap 
between two consecutive files exceeds the configured `delta` (default: 30 minutes), 
a new folder is created. Each folder is named after the exact timestamp of its 
first file (pattern: YYYY - MMDD - hhmm).

Supported extension are the following :

- Image : JPG, JPEG, HEIC, HEIF, WebP, AVIF
- Raw : NEF, NRW, CR2, CRW, RW2, DNG, ARW, ORF, RAF
- Movie : MOV, AVI, MP4

## Technology stack

1. [Go 1.25](https://golang.org) is the language
2. [Urfave/cli v2](https://github.com/urfave/cli) the CLI library
3. [Logrus](https://github.com/sirupsen/logrus) the logger

## CLI Parameters

### Core Options
* `--use-exif` : use EXIF metadata for dates (default: true, set to false to use ModTime only)
* `-delta` / `-d` : change the default (30min) delta time between 2 files to be split
* `-dryrun` / `-dr` : print the modification to be done without really moving the files
* `-v` / `--verbose` : enable verbose/debug logging
* `-h` / `--help` : show help

### GPS Location Clustering (v2.3.0+)
* `--gps` / `-g` : enable GPS location clustering (default: false, opt-in feature)
* `--gps-radius` / `-gr` : GPS clustering radius in meters (default: 2000m = 2km)

### File Organization
* `-nomvmov` / `-nmm` : do not move movies in a separate `mov` folder
* `-nomvraw` / `-nmr` : do not move raw files in a separate `raw` folder

## Results

Effects of **picsplit** are the following :

```
data
├── PHOTO_01.JPG
├── PHOTO_02.JPG
├── PHOTO_03.CR2
├── PHOTO_03.JPG
├── PHOTO_04.JPG
├── PHOTO_04.MOV
├── PHOTO_04.NEF
├── PHOTO_04.test
└── TEST
```

to

```
data
├── 2019 - 0216 - 0835
│   ├── PHOTO_01.JPG
│   ├── PHOTO_02.JPG
│   ├── PHOTO_03.JPG
│   ├── PHOTO_04.JPG
│   ├── mov
│   │   └── PHOTO_04.MOV
│   └── raw
│       ├── PHOTO_03.CR2
│       └── PHOTO_04.NEF
├── PHOTO_04.test
└── TEST
```

**Note**: With default delta=30min, photos would be split into multiple folders 
when gaps exceed 30min. Example with delta=1h shown: all photos (08:35, 09:35, 
10:35, 11:35, 11:44) grouped in one folder since gaps ≤1h. Folder named 
"2019 - 0216 - 0835" (timestamp of first file PHOTO_01.JPG).

## Installation

### Pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/sebastienfr/picsplit/releases).

Available platforms:
- macOS (Intel & Apple Silicon)
- Linux (amd64 & arm64)
- Windows (amd64)

### Build from source

```bash
git clone https://github.com/sebastienfr/picsplit.git
cd picsplit
make build
```

The binary will be created in `./bin/picsplit`.

To install system-wide:

```bash
make install
```

## Development

### Build and test

Makefile is used to build `picsplit`:

```bash
make              # Clean, build and test
make build        # Build binary to ./bin/picsplit
make test         # Run tests
make test-coverage # Run tests with coverage
make coverage-html # Generate HTML coverage report
make lint-ci      # Run golangci-lint
```

### Running tests

```bash
make test-coverage
make coverage-html  # Opens HTML report in browser
```

### Linting

```bash
make lint         # Basic go vet
make lint-ci      # Comprehensive linting with golangci-lint
```

### Release (maintainers only)

```bash
make release-snapshot  # Test release build locally
git tag -a v2.0.1 -m "Release v2.0.1"
git push origin v2.0.1 # Triggers automatic release via GitHub Actions
```

## Usage

### Basic usage (time-only mode)
```bash
# Dry run to preview changes
picsplit -v -dryrun ./data

# Organize photos by time (default 30min delta)
picsplit -v ./data

# Don't separate movies and raw files into subfolders
picsplit -v -nomvmov -nomvraw ./data

# Custom time delta (1 hour)
picsplit -v --delta 1h ./data
```

### GPS location clustering (v2.3.0+)
```bash
# Enable GPS clustering with default 2km radius
picsplit --gps ./photos

# Custom 5km radius for larger geographic areas
picsplit --gps --gps-radius 5000 ./photos

# GPS clustering with custom time delta
picsplit --gps --delta 1h ./photos

# Dry run to preview GPS clustering structure
picsplit --gps -v -dryrun ./photos

# Combine with EXIF and other options
picsplit --gps --use-exif --delta 2h ./photos
```

### Example output structures

**Time-only mode** (default):
```
data/
├── 2025 - 0616 - 0945/
│   ├── photo1.jpg
│   ├── photo2.jpg
│   └── mov/
│       └── video1.mov
└── 2025 - 0616 - 1445/
    └── photo3.jpg
```

**GPS clustering mode** (`--gps`):
```
photos/
├── 48.8566N-2.3522E/          # Paris location cluster
│   ├── 2025 - 0616 - 0945/    # Morning photos
│   │   ├── photo1.jpg
│   │   └── photo2.jpg
│   └── 2025 - 0616 - 1430/    # Afternoon photos
│       └── photo3.jpg
├── 51.5074N-0.1278W/          # London location cluster
│   └── 2025 - 0617 - 1000/
│       └── photo4.jpg
└── NoLocation/                 # Files without GPS
    └── 2025 - 0618 - 1200/
        └── scan1.jpg
```

## Roadmap

### Version 2.3.0 (Current - December 2024)

- [X] GPS location clustering (group by location + time)
- [X] DBSCAN-like spatial clustering algorithm
- [X] Configurable clustering radius (default: 2km)
- [X] NoLocation folder for files without GPS coordinates
- [X] Haversine distance calculation for accurate geographic distances
- [X] 100% test coverage for GPS modules

### Version 2.2.0 (December 2024)

- [X] EXIF metadata support for photos (DateTimeOriginal)
- [X] Video metadata support (MP4/MOV creation_time)
- [X] RAW+JPEG pairing (share EXIF from associated JPEG)
- [X] Strict fallback mode (all files use ModTime if any lacks EXIF)
- [X] GPS coordinate extraction (preparation for location clustering)
- [X] Date validation (1990 < date < now+1day)

### Version 2.1.0 (December 2024)

- [X] Gap-based event detection algorithm
- [X] Improved grouping of continuous photo sessions
- [X] Folder naming based on first file's exact timestamp
- [X] Better handling of sessions spanning hour boundaries

### Version 2.0.0 (December 2024)

- [X] Migration to Go 1.25
- [X] Migration to urfave/cli v2
- [X] Support for modern image formats (HEIC, WebP, AVIF)
- [X] Support for additional RAW formats (DNG, ARW, ORF, RAF)
- [X] Comprehensive test suite (82.8% coverage)
- [X] Improved error handling
- [X] GitHub Actions CI/CD pipeline
- [X] Automated releases with GoReleaser (multi-platform binaries)
- [X] Security scanning with CodeQL

### Version 1.0.0

- [X] configurable delta
- [X] move movies
- [X] move raw
- [X] dry run mode

### Next releases

- [ ] Version 2.4.0: Merge folder command (case split too much)
- [ ] Version 3.0.0: Interactive console GUI with TUI
- [ ] Duplicate detection and handling
- [ ] Photo similarity detection (group similar photos together)

