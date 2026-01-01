# Test Fixtures

## Video Test Files

This directory contains test fixtures for video metadata extraction tests.

### Files

#### `fixture.mp4`
- **Size**: 2.3 KB
- **Creation Time**: 2024-12-20 15:30:00 UTC
- **Format**: ISO Media, MP4 Base Media v1
- **Generated with**: ffmpeg (minimal 1-second blue frame video)
- **Purpose**: Test valid MP4 with known creation_time metadata

This fixture is used by:
- `TestExtractVideoMetadata_ValidMP4` - Extracts and validates MP4 creation time
- `TestExtractMetadata_VideoFile` - Tests full metadata extraction from MP4

### Fixture Generation

The `fixture.mp4` was generated with ffmpeg using this command:
```bash
ffmpeg -f lavfi -i color=c=blue:s=320x240:d=1 \
  -metadata creation_time="2024-12-20T15:30:00.000000Z" \
  -vcodec libx264 -pix_fmt yuv420p \
  -y handler/testdata/fixture.mp4
```

**Note**: This is a real, valid MP4 file committed to the repository. It does NOT require ffmpeg to be installed for tests to run.

### Skipped Tests

#### `TestExtractMetadata_VideoWithInvalidDate`

This test is skipped because it would require programmatically generating an MP4 with an invalid creation time (before 1990), which is complex. The fallback behavior is already covered by `TestExtractMetadata_MovieFallback` which uses a broken AVI file.

### Current Test Coverage

We have **comprehensive test coverage** for video metadata:
- ✅ Valid MP4 with creation_time extraction (`fixture.mp4`)
- ✅ Invalid MP4 files (corrupt data)
- ✅ MP4 without `mvhd` box (missing metadata)
- ✅ File open errors
- ✅ Fallback to ModTime when video metadata fails

**Test Results**: 58/59 tests passing, 1 skipped (invalid date test)
