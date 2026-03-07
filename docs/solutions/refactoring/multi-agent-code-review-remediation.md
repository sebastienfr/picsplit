---
title: "Refactoring splitter.go into focused files and encapsulating geocoding state"
date: 2026-03-07
category: refactoring
tags:
  - go
  - architecture
  - code-splitting
  - global-state
  - security-hardening
  - performance
  - code-review
components:
  - handler/splitter.go
  - handler/grouping.go
  - handler/organized.go
  - handler/processing.go
  - handler/geo.go
  - handler/config.go
issues:
  - number: 23
    title: "splitter.go should be split into focused files"
  - number: 24
    title: "Geocoding uses global mutable state - should be encapsulated in a struct"
severity: P1
status: completed
findings_total: 16
findings_breakdown:
  security: 4
  architecture: 4
  performance: 4
  quality: 4
review_method: "two-pass multi-agent (security, performance, architecture, patterns, simplicity)"
---

# Refactoring splitter.go and Encapsulating Geocoding State

## Problem Statement

`handler/splitter.go` in picsplit (a Go CLI for photo/video organization) had grown to ~1115 lines with multiple responsibilities: file collection, sorting, grouping, processing, organized folder detection, and orphan RAW handling. Separately, `handler/geo.go` used global mutable state (package-level variables protected by mutexes) for geocoding caching and rate limiting. These issues were tracked as GitHub issues [#23](https://github.com/sebastienfr/picsplit/issues/23) and [#24](https://github.com/sebastienfr/picsplit/issues/24).

## Root Cause

Organic growth without periodic refactoring. `splitter.go` accumulated responsibilities across versions as features were added (GPS clustering, organized folder detection, orphan RAW handling, duplicate detection). Geocoding state was implemented as global variables because it was simpler initially, but this made testing harder, prevented multiple instances, and violated encapsulation principles.

## Solution

Sixteen fixes were applied across eight files, discovered through a two-pass multi-agent code review (5 agents: security-sentinel, performance-oracle, architecture-strategist, pattern-recognition-specialist, code-simplicity-reviewer).

### File Decomposition (issue #23)

| File | Responsibility | Lines |
|------|---------------|-------|
| `splitter.go` | Orchestration (`Split`, `splitInternal`, `finalizeProcessing`) | ~390 |
| `grouping.go` | `fileGroup` type, `collectMediaFilesWithMetadata`, `sortFilesByDateTime`, `groupFilesByGaps` | ~141 |
| `organized.go` | `isOrganizedFolder`, `refreshOrphanRAW`, `processOrganizedFolder` + helpers | ~225 |
| `processing.go` | `handleDuplicate`, `processGroup`, `processMediaFile`, `findOrCreateFolder`, `moveFile`, `isRawPaired` | ~224 |

### GeocodingService Encapsulation (issue #24)

Replaced global variables (`geocodeCache`, `lastNominatimCall`, mutexes) with a `GeocodingService` struct instantiated via `NewGeocodingService()` constructor. Reuses a single `http.Client` on the struct.

```go
type GeocodingService struct {
    cache             map[string]string
    cacheMutex        sync.RWMutex
    lastNominatimCall time.Time
    nominatimMutex    sync.Mutex
    client            *http.Client
}

func NewGeocodingService() *GeocodingService {
    return &GeocodingService{
        cache:  make(map[string]string),
        client: &http.Client{Timeout: geocodingTimeout},
    }
}
```

### Security Hardening (geo.go)

1. **Bounded HTTP responses** -- Added `io.LimitReader(resp.Body, maxGeocodingBodyBytes)` on both Nominatim and BigDataCloud responses (64KB cap)
2. **User-Agent on all endpoints** -- Added `req.Header.Set("User-Agent", geocodingUserAgent())` to `tryBigDataCloud` (was already on `tryNominatim`)
3. **HTTP client reuse** -- Replaced per-call `&http.Client{}` in `tryBigDataCloud` with `gs.client`
4. **Complete input sanitization** -- Enhanced `sanitizeFolderName`: filters null bytes + control characters via `strings.Map` + `unicode.IsControl`, enforces `maxFolderNameLength` with UTF-8-safe rune-boundary truncation

```go
var folderNameReplacer = strings.NewReplacer(/* ... */)

func sanitizeFolderName(name string) string {
    name = folderNameReplacer.Replace(name)
    // Remove null bytes and control characters
    name = strings.Map(func(r rune) rune {
        if r < 0x20 || r == 0x7F || unicode.IsControl(r) {
            return -1
        }
        return r
    }, name)
    name = strings.TrimSpace(name)
    // Enforce maximum length (truncate at rune boundary)
    if len(name) > maxFolderNameLength {
        runes := []rune(name)
        for len(string(runes)) > maxFolderNameLength {
            runes = runes[:len(runes)-1]
        }
        name = string(runes)
    }
    return name
}
```

### Architecture and Quality Fixes

| Fix | Detail |
|-----|--------|
| `permDirectory` relocated | Moved from `merger.go` to `config.go` (shared constant used by 3 files) |
| Duplicate defer cleanup extracted | ~17-line block in `splitInternal` + `refreshOrphanRAW` -> `finalizeProcessing(cfg, stats)` |
| Magic numbers replaced | `18`/`19`/`0.5` -> `dateFormatLength`, `organizedFolderThreshold` constants |
| Helpers extracted | `isDateFormattedName()`, `isSpecialFolder()` in `organized.go` |
| Algorithm deduplicated | `GroupLocationByTime` now calls `groupFilesByGaps` instead of reimplementing |
| `stats.AddError()` consistent | Replaced direct `stats.Errors = append(...)` |
| Dry-run logging harmonized | Consistent `[DRY RUN]` prefix pattern in `organized.go` |
| `findOrCreateFolder` simplified | `os.Stat` + `os.Mkdir` + `os.Stat` (~30 lines) -> `os.MkdirAll` (~10 lines) |
| Symlink protection | Added `entry.Type()&os.ModeSymlink != 0` check in `collectMediaFilesWithMetadata` |
| Slice pre-allocation | `make([]FileMetadata, 0, len(entries))` in `grouping.go` |
| No-GPS loop collapsed | Deduplicated conditional loop over `noGPSGroups` in `splitter.go` |
| `strings.NewReplacer` hoisted | Package-level `folderNameReplacer` instead of per-call allocation |

## Related Documentation

- [Issue #23 -- splitter.go should be split into focused files](https://github.com/sebastienfr/picsplit/issues/23)
- [Issue #24 -- Geocoding uses global mutable state](https://github.com/sebastienfr/picsplit/issues/24)
- [Issue #35 -- Config struct fields could be grouped into sub-structs](https://github.com/sebastienfr/picsplit/issues/35) (deferred)
- [docs/solutions/code-review/picsplit-comprehensive-quality-review.md](../code-review/picsplit-comprehensive-quality-review.md) -- Original v2.10.0 code review

## Prevention Strategies

### 1. Monolithic files

Add a CI check that warns when any `.go` file exceeds 400 lines. When adding a new feature, ask: "Does this belong to an existing responsibility, or is it a new one?" If new, create a new file from the start.

### 2. Global mutable state

Treat any mutable `var` at package level as a code smell. When you need shared state (caches, rate limiters, HTTP clients), put it in a struct. Consider enabling `gochecknoglobals` linter.

### 3. Unbounded HTTP response bodies

Create a single internal HTTP helper that wraps `http.Get`, applies `io.LimitReader`, checks status codes, and decodes JSON. Never call `json.NewDecoder(resp.Body)` directly without `LimitReader`.

### 4. Incomplete input sanitization

Establish a clear boundary: any string from outside the process (API responses, EXIF metadata, filenames) must pass through a sanitization function before being used in a filesystem path.

### 5. Code duplication

When you notice yourself copying a block, stop and extract immediately. For algorithm variants, prefer a single function with parameters over two near-identical implementations. Consider enabling `dupl` linter with threshold ~50 lines.

### 6. Inconsistent API usage

When introducing a method like `AddError()`, immediately search for and replace all existing direct usages. Establish canonical helpers (`moveFile()`, `AddError()`) and make them the only way to perform those operations.

## Code Review Checklist (for future Go refactorings)

**File structure:**
- [ ] No file exceeds 400 lines
- [ ] Each file owns one concept
- [ ] Constants are defined in the file that owns the concept

**State and dependencies:**
- [ ] No package-level mutable `var`
- [ ] Shared state lives in a struct, not globals
- [ ] Functions receive dependencies via parameters

**Security and robustness:**
- [ ] All HTTP response bodies bounded with `io.LimitReader`
- [ ] All external strings sanitized before use in paths
- [ ] Symlinks explicitly handled in file-walking code
- [ ] No `os.Stat` + `os.Mkdir` pattern (use `os.MkdirAll`)

**Consistency:**
- [ ] One canonical way to move files, record errors, create directories
- [ ] No magic numbers or strings

**Tests:**
- [ ] Table-driven tests for new/refactored code
- [ ] Edge cases covered (empty input, nil, symlinks, permission errors)
- [ ] Tests use `t.TempDir()`

## Testing Recommendations

- **Symlink protection test:** Create a symlink in `t.TempDir()` pointing outside the directory. Run file collection. Assert the symlink is not followed.
- **Sanitization test:** Pass strings with null bytes, control characters, path traversal, and long unicode through `sanitizeFolderName()`. Assert clean output.
- **Bounded HTTP test:** Use `httptest.NewServer` returning a large body. Call the geocoding function. Assert it reads at most `maxGeocodingBodyBytes`.
- **Makefile hygiene target:** Add `make check-hygiene` to catch direct `os.Rename`, unbounded `ReadAll`, and files over 400 lines.
