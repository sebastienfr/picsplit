# picsplit — Claude Code Instructions

## Project Overview

Go CLI tool that organizes photos and videos into dated folders based on EXIF metadata, GPS coordinates, and file timestamps. Two main commands: `split` (default) and `merge`.

- **Language:** Go 1.25
- **Module:** `github.com/sebastienfr/picsplit`
- **CLI framework:** `urfave/cli/v2`
- **Author:** Sebastien FRIESS

## Architecture

```
picsplit.go          # CLI entry point, flag parsing, app configuration
handler/             # Core logic package
  config.go          # Config/MergeConfig structs, validation, defaults
  splitter.go        # Main Split orchestration, file grouping, processing
  clustering.go      # DBSCAN-like GPS clustering, time sub-grouping
  geo.go             # Haversine distance, centroid, reverse geocoding (Nominatim/BigDataCloud)
  exif.go            # EXIF date/GPS extraction, MP4 metadata parsing
  merger.go          # Merge command, conflict resolution
  extensions.go      # Extension maps, executionContext (photo/video/raw detection)
  duplicates.go      # SHA256 duplicate detection with size pre-filtering
  cleanup.go         # Empty directory cleanup (multi-pass bottom-up)
  errors.go          # PicsplitError structured error type
  stats.go           # ProcessingStats, FormatBytes, summary printing
  progress.go        # Progress bar (schollz/progressbar)
  validator.go       # Fast validation mode (no EXIF extraction)
```

## Build & Test (use Makefile)

```bash
make build           # Build binary to ./bin/picsplit
make test            # Run tests with race detector
make test-coverage   # Tests + coverage report (target: 80%)
make lint            # go vet
make lint-ci         # golangci-lint (comprehensive)
make format          # go fmt
make clean           # Clean artifacts
make help            # Show all targets
```

Always use `make` targets — do not run go commands directly unless necessary.

## Key Concepts

- **ExecutionMode:** `validate` (fast scan), `dryrun` (simulate), `run` (execute)
- **executionContext:** Holds merged extension maps (defaults + custom). Use `isPhoto()`, `isMovie()`, `isRaw()`, `isMediaFile()` for file type checks
- **FileMetadata:** Contains FileInfo, DateTime, GPS coords, DateSource
- **GPS clustering:** DBSCAN-like algorithm groups files by location, then by time within each location
- **MinGroupSize:** Groups below threshold leave files at root instead of creating folders
- **Orphan RAW:** RAW files without matching JPEG/HEIC are separated to `orphan/` subfolder

## Conventions

- **Language:** Code and comments in English
- **Error handling:** Use `PicsplitError` for structured errors with Type, Op, Path, Suggestion. Use `fmt.Errorf("...: %w", err)` for wrapping
- **Constants:** Define constants near their usage. No magic strings — use named constants
- **Logging:** `slog` for structured messages. `fmt.Println` only for interactive prompts and visual separators
- **Tests:** Table-driven tests with `t.TempDir()` for isolation
- **Naming:** Go idioms — PascalCase exported, camelCase unexported
- **Commit messages:** Conventional commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`)

## Dependencies

- `urfave/cli/v2` — CLI framework
- `rwcarlsen/goexif` — EXIF metadata extraction
- `abema/go-mp4` — MP4/MOV video metadata
- `schollz/progressbar/v3` — Terminal progress bars
- `mattn/go-isatty` — TTY detection

## CI/CD

- **GitHub Actions:** test (Go 1.24/1.25, ubuntu/macos/windows), lint (staticcheck), build, CodeQL
- **Release:** GoReleaser on tag push (`v*.*.*`), builds for linux/darwin/windows (amd64/arm64)
- **Main branch:** `master` (for PRs), development on `main`

## Known Issues (deferred from v2.10.0 code review)

- [#23](https://github.com/sebastienfr/picsplit/issues/23): splitter.go should be split into focused files
- [#24](https://github.com/sebastienfr/picsplit/issues/24): Geocoding uses global mutable state — should be encapsulated in a struct
- [#35](https://github.com/sebastienfr/picsplit/issues/35): Config struct fields could be grouped into sub-structs
- [#40](https://github.com/sebastienfr/picsplit/issues/40): Migrate to Go 1.26
