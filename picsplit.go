package main

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/sebastienfr/picsplit/handler"
	"github.com/urfave/cli/v2"
)

var (
	// version will be set via ldflags during build (-X main.version=X.Y.Z)
	// Default: "dev" for development builds without ldflags
	version = "dev"

	// path -path : the path to the folder containing the files to be processed
	path = "."

	// delta -delta : change the default (30min) delta time between 2 events to be split
	durationDelta = 30 * time.Minute

	// movie -nomvmov : do not move the movie files in a separate subfolder called mov
	noMoveMovie = false

	// raw -nomvraw : do not move the raw files in a separate subfolder called raw
	noMoveRaw = false

	// dryRun -dryrun : print the modification to be done without really moving the files
	dryRun = false

	// useEXIF -use-exif : use EXIF metadata for dates (photos and videos)
	useEXIF = true

	// useGPS -gps : enable GPS location clustering
	useGPS = false

	// gpsRadius -gps-radius : GPS clustering radius in meters
	gpsRadius = 2000.0

	// customPhotoExts -pext : additional photo extensions (v2.5.0+)
	customPhotoExts string

	// customVideoExts -vext : additional video extensions (v2.5.0+)
	customVideoExts string

	// customRawExts -rext : additional RAW extensions (v2.5.0+)
	customRawExts string

	// separateOrphanRaw -separate-orphan : separate unpaired RAW files to orphan/ folder (v2.6.0+)
	separateOrphanRaw = true

	// continueOnError -continue-on-error : continue processing even if errors occur (v2.8.0+)
	continueOnError = false

	header, _ = base64.StdEncoding.DecodeString("ICAgICAgIC5fXyAgICAgICAgICAgICAgICAgICAgICAuX18gIC5fXyAgX18KX19f" +
		"X19fIHxfX3wgX19fXCAgIF9fX19fX19fX19fXyB8ICB8IHxfX3wvICB8XwpcX19fXyBcfCAgfC8gX19fXCAvICBfX18vXF9fX18gXHwgIHw" +
		"gfCAgXCAgIF9fXAp8ICB8Xz4gPiAgXCAgXF9fXyBcX19fIFwgfCAgfF8+ID4gIHxffCAgfHwgIHwKfCAgIF9fL3xfX3xcX19fICA+X19fXy" +
		"AgPnwgICBfXy98X19fXy9fX3x8X198CnxfX3wgICAgICAgICAgIFwvICAgICBcLyB8X198")
)

const (
	// Default configuration values
	defaultPath      = "."
	defaultDelta     = 30 * time.Minute
	defaultLogLevel  = "info"
	defaultLogFormat = "text"

	// Application metadata
	appName        = "picsplit"
	appUsage       = "photo shooting event splitter and merger"
	authorName     = "Sébastien FRIESS"
	copyrightOwner = "sebastienfr"

	// Command names
	cmdMerge = "merge"

	// Flag names
	flagForce     = "force"
	flagDryRun    = "dryrun"
	flagLogLevel  = "log-level"
	flagLogFormat = "log-format"
)

// parseExtensions parses comma-separated extension string into slice
// Returns error if any extension is invalid
func parseExtensions(extString string) ([]string, error) {
	if extString == "" {
		return nil, nil
	}

	parts := strings.Split(extString, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Validate extension
		if err := handler.ValidateExtension(part); err != nil {
			return nil, err
		}

		result = append(result, part)
	}

	return result, nil
}

// setupLogger initializes the slog logger with the specified level and format
func setupLogger(logLevel, logFormat string) {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo // Default to info if invalid
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	switch strings.ToLower(logFormat) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts) // Default to text if invalid
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)
}

// getBuildInfo returns version information from build metadata
// Version is injected via ldflags, VCS info comes from runtime/debug (Go 1.18+)
func getBuildInfo() (string, string, string) {
	buildTime := time.Now().Format(time.RFC3339)
	gitHash := "unknown"

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, buildTime, gitHash
	}

	// Extract VCS information (commit hash, build time, dirty flag)
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) > 7 {
				gitHash = setting.Value[:7] // Short hash
			} else {
				gitHash = setting.Value
			}
		case "vcs.time":
			buildTime = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				gitHash += "-dirty"
			}
		}
	}

	return version, buildTime, gitHash
}

func main() {
	// customize version flag
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
	}

	// Get build information
	version, buildTime, gitHash := getBuildInfo()

	// Parse build time
	buildTimeObj, err := time.Parse(time.RFC3339, buildTime)
	if err != nil {
		buildTimeObj = time.Now()
	}

	app := &cli.App{
		Name:  appName,
		Usage: appUsage,
		Version: version + ", build on " + buildTimeObj.Format("2006-01-02 15:04:05 -0700 MST") +
			", git hash " + gitHash,
		Authors: []*cli.Author{
			{Name: authorName},
		},
		Copyright: copyrightOwner + " " + strconv.Itoa(time.Now().Year()),
		Commands: []*cli.Command{
			{
				Name:      cmdMerge,
				Usage:     "Merge multiple time-based folders into one",
				ArgsUsage: "SOURCE1 [SOURCE2 ...] TARGET",
				Description: `Merge multiple time-based folders into a single target folder.
   Files are moved (not copied) to save disk space.
   Source folders are automatically deleted after successful merge.
   
   IMPORTANT: GPS location folders (e.g., "48.8566N-2.3522E") cannot be merged.
   Only time-based folders (e.g., "2025 - 0616 - 0945") are supported.
   
   Conflict handling:
   - By default, asks user how to resolve each conflict (rename/skip/overwrite)
   - Use --force to automatically overwrite all conflicts without asking
   
   Examples:
     picsplit merge "2025 - 0616 - 0945" "2025 - 0616 - 1430" "2025 - 0616 - merged"
     picsplit merge folder1 folder2 folder3 target --force
     picsplit merge folder1 folder2 target --dryrun -v`,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    flagForce,
						Aliases: []string{"f"},
						Usage:   "Overwrite files without asking on conflict",
					},
					&cli.BoolFlag{
						Name:    flagDryRun,
						Aliases: []string{"dr"},
						Usage:   "Simulate merge without moving files",
					},
					&cli.StringFlag{
						Name:    flagLogLevel,
						Aliases: []string{"l"},
						Value:   defaultLogLevel,
						Usage:   "Set log level (debug, info, warn, error)",
					},
					&cli.StringFlag{
						Name:    flagLogFormat,
						Aliases: []string{"lf"},
						Value:   defaultLogFormat,
						Usage:   "Set log format (text, json)",
					},
					&cli.StringFlag{
						Name:    "photo-ext",
						Aliases: []string{"pext"},
						Usage:   "Additional photo extensions (comma-separated, e.g., 'png,gif,bmp'). Max 8 chars, alphanumeric only",
					},
					&cli.StringFlag{
						Name:    "video-ext",
						Aliases: []string{"vext"},
						Usage:   "Additional video extensions (comma-separated, e.g., 'mkv,mpeg,wmv'). Max 8 chars, alphanumeric only",
					},
					&cli.StringFlag{
						Name:    "raw-ext",
						Aliases: []string{"rext"},
						Usage:   "Additional RAW extensions (comma-separated, e.g., 'rwx,srw,3fr'). Max 8 chars, alphanumeric only",
					},
				},
				Action: func(c *cli.Context) error {
					// Init logger
					setupLogger(c.String(flagLogLevel), c.String(flagLogFormat))

					// Print header
					fmt.Println(string(header))

					// Validate arguments
					if c.NArg() < 2 {
						return fmt.Errorf("merge requires at least 2 arguments (SOURCE... TARGET)")
					}

					// Parse arguments
					args := c.Args().Slice()
					targetFolder := args[len(args)-1]
					sourceFolders := args[:len(args)-1]

					// Parse custom extensions from command flags
					photoExts, err := parseExtensions(c.String("photo-ext"))
					if err != nil {
						return fmt.Errorf("invalid photo extensions: %w", err)
					}

					videoExts, err := parseExtensions(c.String("video-ext"))
					if err != nil {
						return fmt.Errorf("invalid video extensions: %w", err)
					}

					rawExts, err := parseExtensions(c.String("raw-ext"))
					if err != nil {
						return fmt.Errorf("invalid RAW extensions: %w", err)
					}

					// Debug info
					slog.Debug("merge configuration",
						"sources", sourceFolders,
						"target", targetFolder,
						"force", c.Bool(flagForce),
						"dryrun", c.Bool(flagDryRun))
					if len(photoExts) > 0 {
						slog.Debug("custom photo extensions", "extensions", strings.Join(photoExts, ", "))
					}
					if len(videoExts) > 0 {
						slog.Debug("custom video extensions", "extensions", strings.Join(videoExts, ", "))
					}
					if len(rawExts) > 0 {
						slog.Debug("custom raw extensions", "extensions", strings.Join(rawExts, ", "))
					}

					// Execute merge
					cfg := &handler.MergeConfig{
						SourceFolders:   sourceFolders,
						TargetFolder:    targetFolder,
						Force:           c.Bool(flagForce),
						DryRun:          c.Bool(flagDryRun),
						CustomPhotoExts: photoExts,
						CustomVideoExts: videoExts,
						CustomRawExts:   rawExts,
					}

					return handler.Merge(cfg)
				},
			},
		},
	}

	// command line flags
	app.Flags = []cli.Flag{
		&cli.DurationFlag{
			Name:        "delta",
			Aliases:     []string{"d"},
			Value:       defaultDelta,
			Destination: &durationDelta,
			Usage:       "The duration between two files to split",
		},
		&cli.BoolFlag{
			Name:        "nomvmov",
			Aliases:     []string{"nmm"},
			Destination: &noMoveMovie,
			Usage:       "Do not move movies in a separate mov folder",
		},
		&cli.BoolFlag{
			Name:        "nomvraw",
			Aliases:     []string{"nmr"},
			Destination: &noMoveRaw,
			Usage:       "Do not move raw files in a separate raw folder",
		},
		&cli.BoolFlag{
			Name:        "dryrun",
			Aliases:     []string{"dr"},
			Destination: &dryRun,
			Usage:       "Only print actions to do, do not move physically the files",
		},
		&cli.StringFlag{
			Name:    flagLogLevel,
			Aliases: []string{"l"},
			Value:   defaultLogLevel,
			Usage:   "Set log level (debug, info, warn, error)",
		},
		&cli.StringFlag{
			Name:    flagLogFormat,
			Aliases: []string{"lf"},
			Value:   defaultLogFormat,
			Usage:   "Set log format (text, json)",
		},
		&cli.BoolFlag{
			Name:        "use-exif",
			Aliases:     []string{"ue"},
			Value:       true,
			Destination: &useEXIF,
			Usage:       "Use EXIF metadata for dates (photos and videos)",
		},
		&cli.BoolFlag{
			Name:        "gps",
			Aliases:     []string{"g"},
			Destination: &useGPS,
			Usage:       "Enable GPS location clustering (group by location then time)",
		},
		&cli.Float64Flag{
			Name:        "gps-radius",
			Aliases:     []string{"gr"},
			Value:       2000.0,
			Destination: &gpsRadius,
			Usage:       "GPS clustering radius in meters (default: 2000m = 2km)",
		},
		&cli.StringFlag{
			Name:        "photo-ext",
			Aliases:     []string{"pext"},
			Destination: &customPhotoExts,
			Usage:       "Additional photo extensions (comma-separated, e.g., 'png,gif,bmp'). Max 8 chars, alphanumeric only",
		},
		&cli.StringFlag{
			Name:        "video-ext",
			Aliases:     []string{"vext"},
			Destination: &customVideoExts,
			Usage:       "Additional video extensions (comma-separated, e.g., 'mkv,mpeg,wmv'). Max 8 chars, alphanumeric only",
		},
		&cli.StringFlag{
			Name:        "raw-ext",
			Aliases:     []string{"rext"},
			Destination: &customRawExts,
			Usage:       "Additional RAW extensions (comma-separated, e.g., 'rwx,srw,3fr'). Max 8 chars, alphanumeric only",
		},
		&cli.BoolFlag{
			Name:        "separate-orphan",
			Aliases:     []string{"so"},
			Value:       true,
			Destination: &separateOrphanRaw,
			Usage:       "Separate unpaired RAW files to orphan/ folder (default: true)",
		},
		&cli.BoolFlag{
			Name:        "continue-on-error",
			Aliases:     []string{"coe"},
			Destination: &continueOnError,
			Usage:       "Continue processing even if errors occur (collect all errors instead of stopping at first failure)",
		},
	}

	// main action
	// sub action are also possible
	app.Action = func(c *cli.Context) error {
		// init log options from command line params
		setupLogger(c.String(flagLogLevel), c.String(flagLogFormat))

		// print header
		fmt.Println(string(header))

		if c.NArg() == 1 {
			path = c.Args().Get(0)
		} else if c.NArg() > 1 {
			return fmt.Errorf("wrong count of argument %d, a unique path is required", c.NArg())
		}

		// Parse custom extensions
		photoExts, err := parseExtensions(customPhotoExts)
		if err != nil {
			return fmt.Errorf("invalid photo extensions: %w", err)
		}

		videoExts, err := parseExtensions(customVideoExts)
		if err != nil {
			return fmt.Errorf("invalid video extensions: %w", err)
		}

		rawExts, err := parseExtensions(customRawExts)
		if err != nil {
			return fmt.Errorf("invalid RAW extensions: %w", err)
		}

		slog.Debug("configuration",
			"path", path,
			"delta_minutes", durationDelta.Minutes(),
			"dry_run", dryRun,
			"no_move_movies", noMoveMovie,
			"no_move_raw", noMoveRaw,
			"use_exif", useEXIF,
			"use_gps", useGPS,
			"gps_radius_meters", gpsRadius,
			"separate_orphan_raw", separateOrphanRaw)
		if len(photoExts) > 0 {
			slog.Debug("custom photo extensions", "extensions", strings.Join(photoExts, ", "))
		}
		if len(videoExts) > 0 {
			slog.Debug("custom video extensions", "extensions", strings.Join(videoExts, ", "))
		}
		if len(rawExts) > 0 {
			slog.Debug("custom raw extensions", "extensions", strings.Join(rawExts, ", "))
		}

		// check path exists
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			return fmt.Errorf("provided path %s is not a directory", path)
		}

		cfg := &handler.Config{
			BasePath:          path,
			Delta:             durationDelta,
			NoMoveMovie:       noMoveMovie,
			NoMoveRaw:         noMoveRaw,
			DryRun:            dryRun,
			UseEXIF:           useEXIF,
			UseGPS:            useGPS,
			GPSRadius:         gpsRadius,
			CustomPhotoExts:   photoExts,
			CustomVideoExts:   videoExts,
			CustomRawExts:     rawExts,
			SeparateOrphanRaw: separateOrphanRaw,
			ContinueOnError:   continueOnError,
			LogLevel:          c.String(flagLogLevel),
			LogFormat:         c.String(flagLogFormat),
		}
		return handler.Split(cfg)
	}

	// run the app
	err = app.Run(os.Args)
	if err != nil {
		slog.Error("runtime error", "error", err)
		os.Exit(1)
	}

	os.Exit(0)
}
