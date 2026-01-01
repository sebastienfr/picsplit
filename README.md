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
2. ✅ Run picsplit on the copy (use `--dryrun` first to preview)
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

- 🔍 **Safe Preview Mode**  
  Dry-run mode lets you preview all changes before applying them

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

# 2. Preview changes (dry run)
picsplit --dryrun -v ~/Photos/DCIM_backup

# 3. Organize photos by time
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
# Preview merge first
picsplit merge "2024-1220-0900" "2024-1220-0915" "2024-1220-birthday" --dryrun

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

picsplit évolue continuellement avec de nouvelles fonctionnalités basées sur les retours utilisateurs. Voici les prochaines versions planifiées :

### ✅ v2.7.0 - Logging & Observability (Released - January 2026)

**Objectif** : Améliorer le feedback utilisateur et l'observabilité pendant l'exécution.

**Fonctionnalités livrées** :

- ✅ **Logs structurés** ([#7](https://github.com/sebastienfr/picsplit/issues/7))  
  Migration vers `log/slog` (stdlib Go) pour des logs typés et performants

- ✅ **Niveaux de log configurables** ([#8](https://github.com/sebastienfr/picsplit/issues/8))  
  `--log-level debug|info|warn|error` + formats Text/JSON (`--log-format`)

- ✅ **Barre de progression temps réel** ([#9](https://github.com/sebastienfr/picsplit/issues/9))  
  Affichage du % d'avancement avec détection automatique TTY

- ✅ **Summary enrichi avec métriques** ([#10](https://github.com/sebastienfr/picsplit/issues/10))  
  Métriques détaillées (durée, throughput, stats par type, erreurs/warnings)

- ✅ **Erreurs typées avec contexte** ([#11](https://github.com/sebastienfr/picsplit/issues/11))  
  Messages d'erreur structurés avec suggestions de correction automatiques

**Exemple de nouveau summary** :
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

---

### 🚀 v2.8.0 - Robustness & Advanced Features (Q1 2026)

**Objectif** : Renforcer la robustesse avec gestion d'erreurs avancée et nouveaux modes.

**Nouvelles fonctionnalités** :

- **Mode continue-on-error** ([#12](https://github.com/sebastienfr/picsplit/issues/12))  
  Traiter tous les fichiers possibles sans s'arrêter au premier échec (comportement par défaut depuis v2.7.0)

- **⚠️ Mode validation rapide** ([#13](https://github.com/sebastienfr/picsplit/issues/13))  
  `--mode validate|dryrun|run` pour pré-vérification ultra-rapide (5s vs 2m)  
  **Breaking change** : Retire `--dryrun` (remplacé par `--mode dryrun`)

- **Détection de doublons** ([#14](https://github.com/sebastienfr/picsplit/issues/14))  
  `--detect-duplicates` pour identifier fichiers identiques (hash SHA256)

- **Nettoyage automatique** ([#15](https://github.com/sebastienfr/picsplit/issues/15))  
  `--cleanup-empty-dirs` pour supprimer dossiers vides après traitement

**Exemple de workflow v2.8.0** :
```bash
# 1. Validation rapide (5s)
picsplit /photos --mode validate

# 2. Dry-run complet (30s)
picsplit /photos --mode dryrun

# 3. Exécution réelle avec détection doublons
picsplit /photos --detect-duplicates --skip-duplicates --cleanup-empty-dirs
```

---

### 💡 Suggérer une fonctionnalité

Vous avez une idée pour améliorer picsplit ? [Ouvrez une issue](https://github.com/sebastienfr/picsplit/issues/new) pour proposer votre suggestion !

**Historique complet** : Consultez le [CHANGELOG](./CHANGELOG.md) pour voir toutes les évolutions depuis la v1.0.

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
# Dry run mode (no files moved)
picsplit --dryrun ./photos
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

---

#### Merge Folders

Combine multiple time-based folders into one.

```bash
# Interactive merge (prompts for conflicts)
picsplit merge folder1 folder2 merged-folder

# Force overwrite all conflicts
picsplit merge folder1 folder2 merged --force

# Preview merge operations  
picsplit merge folder1 folder2 merged --dryrun
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

### CLI Reference

#### Main Command

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--help` | `-h` | - | Show help message |
| `--print-version` | `-V` | - | Display version information |
| `--use-exif` | `-ue` | `true` | Use EXIF metadata for dates |
| `--delta` | `-d` | `30m` | Time gap between sessions (e.g., `1h`, `45m`) |
| `--gps` | `-g` | `false` | Enable GPS location clustering |
| `--gps-radius` | `-gr` | `2000` | GPS clustering radius in meters |
| `--dryrun` | `-dr` | `false` | Preview changes without moving files |
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
| `--force` | `false` | Auto-overwrite conflicts |
| `--dryrun` | `false` | Preview merge operations |
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
2. Use `--dryrun` to preview before applying
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
