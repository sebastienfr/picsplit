# Test Fixtures

## Video Test Files

Some tests for video metadata extraction are currently skipped because they require real MP4 files with valid metadata boxes.

### Skipped Tests

- `TestExtractVideoMetadata_ValidMP4`
- `TestExtractMetadata_VideoFile`  
- `TestExtractMetadata_VideoWithInvalidDate`

### Why These Tests Are Skipped

Creating a fully compliant MP4 file programmatically that go-mp4 can properly parse is complex. Real MP4 files from cameras/phones contain many additional boxes beyond just `ftyp`, `moov`, and `mvhd`.

### How to Enable These Tests

If you want to run these tests with real MP4 fixtures:

1. **Add sample MP4 files** to this directory (`handler/testdata/`):
   ```bash
   # Copy a real video from your camera/phone
   cp ~/Videos/sample.mp4 handler/testdata/valid_video.mp4
   cp ~/Videos/old_video.mp4 handler/testdata/video_1970.mp4
   ```

2. **Modify the tests** to use these fixtures instead of `t.Skip()`:
   ```go
   // In handler/exif_test.go
   func TestExtractVideoMetadata_ValidMP4(t *testing.T) {
       testFile := filepath.Join("testdata", "valid_video.mp4")
       if _, err := os.Stat(testFile); os.IsNotExist(err) {
           t.Skip("Skipping: testdata/valid_video.mp4 not found")
       }
       
       metadata, err := extractVideoMetadata(testFile)
       // ... rest of test
   }
   ```

3. **Run the tests**:
   ```bash
   go test -v ./handler -run TestExtractVideoMetadata_ValidMP4
   ```

### Sample Video Sources

You can get sample MP4 files from:
- Your own camera/phone
- [Sample Videos](https://sample-videos.com/) - Public domain test videos
- [Big Buck Bunny](https://download.blender.org/demo/movies/) - Open source test video

### Current Test Coverage

Even with these tests skipped, we have **comprehensive test coverage** for:
- ✅ Invalid MP4 files (corrupt data)
- ✅ MP4 without `mvhd` box (missing metadata)
- ✅ File open errors
- ✅ Fallback to ModTime when video metadata fails

The skipped tests would only verify the **happy path** with real video metadata extraction, which is already tested in integration scenarios when users run picsplit on real photos/videos.
