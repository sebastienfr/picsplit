---
title: "Code Review v2.10.0 - 18 fixes across security, performance, architecture and quality"
date: 2026-03-07
category: code-review
tags:
  - go
  - cli
  - security
  - performance
  - refactoring
  - bug-fix
  - exif
  - gps
  - deduplication
modules:
  - handler/merger.go
  - handler/splitter.go
  - handler/exif.go
  - handler/duplicates.go
  - handler/cleanup.go
  - handler/geo.go
  - handler/stats.go
  - handler/config.go
  - handler/extensions.go
  - handler/validator.go
symptoms:
  - path traversal in merge command
  - symlinks followed during file operations
  - double EXIF file open degrading performance
  - redundant os.Stat calls
  - full-file SHA256 on every duplicate check
  - multi-pass directory cleanup walks
  - nil map panic in ProcessingStats
  - hardcoded version in user-agent string
  - custom extensions ignored by raw pairing
  - inconsistent logging (fmt vs slog)
severity: high
status: released
version: v2.10.0
issues_fixed: 19,20,21,22,25,26,27,28,29,30,31,32,33,34,36,37,38,39
issues_deferred: 23,24,35
---

# Comprehensive Code Review - picsplit v2.10.0

## Problem Statement

The picsplit codebase (Go CLI for organizing photos/videos) had accumulated technical debt across security, performance, and code quality. A systematic code review using 5 specialized agents identified 21 findings. 18 were fixed and released as v2.10.0.

## Investigation Approach

Five review agents ran in parallel:
1. **Security Sentinel** - path traversal, symlinks, input validation
2. **Performance Oracle** - I/O patterns, algorithmic complexity
3. **Architecture Strategist** - code structure, duplication, coupling
4. **Code Simplicity Reviewer** - YAGNI, unnecessary abstractions
5. **Pattern Recognition Specialist** - naming conventions, consistency

Findings were categorized by severity (P1-critical, P2-important, P3-nice-to-have), GitHub issues created (#19-#39), then fixed incrementally with tests passing after each batch.

## Key Solutions

### 1. SHA256 Partial Hash Bug - Silent Failure on Moved Files

**The most interesting finding.** Agent #27 introduced a 64KB partial hash optimization for duplicate detection. But it caused a regression that only surfaced in integration tests.

**Root cause:** The optimization deferred full SHA256 computation to when a collision was detected. By that time, the first file (the "candidate") had already been moved to its destination by `processMediaFile()`. The call `sha256File(candidate)` silently failed because the file no longer existed at its original path.

**Fix:** Pre-compute the full hash at first encounter, while the file is still in place.

```go
// BEFORE (broken): deferred full hash - file may be moved before re-read
if !partialFound {
    d.partialHashes[partialHash] = []string{filePath}
    return false, "", nil  // full hash never computed!
}

// AFTER (fixed): pre-compute full hash immediately at first encounter
if !partialFound {
    d.partialHashes[partialHash] = []string{filePath}
    fullHash, fErr := sha256File(filePath)
    if fErr == nil {
        d.hashes[fullHash] = filePath
        d.fullHashed[filePath] = true
    }
    return false, "", nil
}
```

**Key insight:** Any optimization that defers reading a file in a pipeline that relocates files must account for the file's lifecycle. Compute all needed data from a file before yielding control to code that may move or delete it.

### 2. Path Traversal in Merge Command (#22)

**Root cause:** The merge command used user-supplied paths directly with `os.RemoveAll` without validating that resolved paths stayed within expected boundaries.

**Solution:** Three defensive functions:
- `resolvePath()` - resolves symlinks to canonical paths (with fallback for non-existent children via parent resolution)
- `isSubPath()` - verifies containment after resolution
- `isSafeToRemoveAll()` - guard before any `os.RemoveAll`, rejects root, home, and shallow paths

**macOS nuance:** `/tmp` is a symlink to `/private/tmp`. When the child directory doesn't exist yet, `filepath.EvalSymlinks` fails. Fix: resolve the parent directory and append the child component.

```go
func resolvePath(p string) (string, error) {
    absPath, err := filepath.Abs(p)
    if err != nil {
        return "", fmt.Errorf("failed to resolve absolute path: %w", err)
    }
    resolved, err := filepath.EvalSymlinks(absPath)
    if err != nil {
        // Child doesn't exist yet - resolve parent, append base
        parentResolved, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
        if parentErr != nil {
            return absPath, nil
        }
        return filepath.Join(parentResolved, filepath.Base(absPath)), nil
    }
    return resolved, nil
}
```

### 3. Single EXIF Decode Pass (#19)

**Root cause:** Each photo was opened and decoded via `exif.Decode()` twice - once for DateTime, once for GPS coordinates.

**Solution:** New `extractEXIFAll()` opens the file once, calls `exif.Decode` once, extracts both DateTime and GPS from the same `*exif.Exif` object. Halves I/O for photos with GPS data.

### 4. Unified processMediaFile (#30)

**Root cause:** Four near-identical functions existed: `processPicture`, `processMovie`, `processPictureAtRoot`, `processMovieAtRoot`. Fixes applied to one copy were missed in others.

**Solution:** Single `processMediaFile()` determines subfolder logic based on file type internally. Eliminated ~60% of duplicated code.

### 5. Dynamic Version in User-Agent (#37)

**Root cause:** `const geocodingUserAgent = "picsplit/2.9.0 ..."` drifted out of sync on every release.

**Solution:** Replaced const with function reading `handler.Version` (set from main via ldflags at startup).

## All Changes Summary

| Category | Issues | Count |
|----------|--------|-------|
| Security | #22 (path traversal), #28 (symlinks) | 2 |
| Performance | #19 (EXIF), #25 (cleanup), #26 (os.Stat), #27 (SHA256) | 4 |
| Bug fixes | #21 (extensions), #36 (nil map), #37 (version) | 3 |
| Refactoring | #20, #29, #30, #31, #32, #33, #34 | 7 |
| Tests | #38, #39 | 2 |
| **Total fixed** | | **18** |
| Deferred | #23 (split splitter.go), #24 (geocoding struct), #35 (config sub-structs) | 3 |

## Prevention Strategies

### 1. File-State Assumptions
- **Gather-then-act pattern:** Collect all metadata (hash, EXIF, size) in a single pass before any file mutation
- **Make moves the terminal operation:** No code should reference the source path after a move
- Pass `os.FileInfo` / open file handles through the pipeline rather than re-reading

### 2. Copy-Paste Duplication
- Treat two copies of structurally identical logic (>30 lines) as the threshold for extraction
- Use `golangci-lint` with the `dupl` plugin in CI
- When merging functions, test with both code paths to confirm behavior parity

### 3. Hardcoded Values
- All string literals representing domain concepts belong in named constants
- Version strings must be injected at build time via `-ldflags`
- CI grep check: `grep -rn '"photos"' --include='*.go' | grep -v constants.go` should return zero

### 4. Missing Input Validation
- Canonicalize all path inputs (`filepath.EvalSymlinks` + `filepath.Abs`) at entry points
- Check source/destination overlap before merge operations
- Declare symlink policy explicitly: resolve or reject, never silently follow

### 5. Performance Anti-Patterns
- Open each file at most once per pipeline pass
- Cache `os.FileInfo` / stat results - pass them, don't re-query
- Use partial hashing for large files with full hash only on collision
- Add `Benchmark*` functions for hot paths

### 6. Test Naming Drift
- Convention: `TestFoo` tests function `Foo`, `TestType_Method` tests `Type.Method`
- When renaming a function, `grep -rn 'OldName' --include='*.go'` including test files

## Best Practices Checklist

```
[ ] No file path is used after the file has been moved or renamed
[ ] Each file is opened at most once per pipeline pass
[ ] os.FileInfo results are passed, not re-queried
[ ] No two functions share more than 30 lines of structurally identical logic
[ ] All domain string/numeric literals are named constants
[ ] Version string is injected at build time, not hardcoded
[ ] All path inputs are canonicalized and validated at entry points
[ ] Source/destination overlap is checked before merge operations
[ ] Symlinks are either resolved or explicitly rejected
[ ] Large files use partial hashing
[ ] Every Test* function name corresponds to an existing function/method
```

## Cross-References

- [CHANGELOG v2.10.0](../../../CHANGELOG.md)
- [GitHub compare v2.9.0...v2.10.0](https://github.com/sebastienfr/picsplit/compare/v2.9.0...v2.10.0)
- [CLAUDE.md - Project Conventions](../../../CLAUDE.md)
- Deferred issues: [#23](https://github.com/sebastienfr/picsplit/issues/23), [#24](https://github.com/sebastienfr/picsplit/issues/24), [#35](https://github.com/sebastienfr/picsplit/issues/35)
- Follow-up: [#40 - Migrate to Go 1.26](https://github.com/sebastienfr/picsplit/issues/40)

## Files Modified

18 files changed, +959 insertions, -503 deletions:

| File | Changes |
|------|---------|
| handler/splitter.go | Unified processMediaFile, handleDuplicate extraction, AtRoot removal |
| handler/merger.go | Path traversal protection, symlink skip, overlap detection |
| handler/duplicates.go | Partial hash optimization with pre-computed full hash |
| handler/cleanup.go | Single-pass bottom-up walk, symlink skip |
| handler/exif.go | extractEXIFAll single decode, os.Stat removal, ctx in findAssociatedJPEG |
| handler/config.go | Version var, ExecutionMode.IsValid(), ParseExecutionMode() |
| handler/geo.go | Dynamic geocodingUserAgent(), English comments |
| handler/extensions.go | photoSearchExtensions() method |
| handler/stats.go | NewProcessingStats(), slog standardization |
| handler/validator.go | slog standardization |
| handler/errors.go | English comments |
| picsplit.go | handler.Version injection, ParseExecutionMode() usage |
| handler/*_test.go | New tests for all changes, renamed/fixed existing tests |
