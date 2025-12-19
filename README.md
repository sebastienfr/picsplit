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

**Gap-based event detection (v2.1.0+):**
Files are sorted chronologically and grouped by temporal gaps. When the time gap 
between two consecutive files exceeds the configured `delta` (default: 30 minutes), 
a new folder is created. Each folder is named after the exact timestamp of its 
first file (pattern: YYYY - MMDD - hhmm).

The file modification time is used as the grouping parameter.

Supported extension are the following :

- Image : JPG, JPEG, HEIC, HEIF, WebP, AVIF
- Raw : NEF, NRW, CR2, CRW, RW2, DNG, ARW, ORF, RAF
- Movie : MOV, AVI, MP4

## Technology stack

1. [Go 1.25](https://golang.org) is the language
2. [Urfave/cli v2](https://github.com/urfave/cli) the CLI library
3. [Logrus](https://github.com/sirupsen/logrus) the logger

## CLI Parameters

* `-nomvmov` : do not move movies in a separate `mov` folder
* `-nomvraw` : do not move raw files in a separate `raw` folder
* `-delta` : change the default (30min) delta time between 2 files to be split
* `-dryrun` : print the modification to be done without really moving the files
* `-v` : verbose
* `-h` : help

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

    picsplit -v -dryrun ./data
    picsplit -v ./data
    picsplit -v -nomvmov -nomvraw ./data

## Roadmap

### Version 2.1.0 (Current - December 2024)

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

- [ ] merge folder command (case split too much)
- [ ] add an option to read dating data from EXIF instead of file dates
- [ ] add a console GUI

