package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sebastienfr/picsplit/handler"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	// Version is the version of the software
	Version string
	// BuildStmp is the build date
	BuildStmp string
	// GitHash is the git build hash
	GitHash string

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

	// verbose -v : print the detailed logs to the output
	verbose = false

	// useEXIF -use-exif : use EXIF metadata for dates (photos and videos)
	useEXIF = true

	// useGPS -gps : enable GPS location clustering
	useGPS = false

	// gpsRadius -gps-radius : GPS clustering radius in meters
	gpsRadius = 2000.0

	header, _ = base64.StdEncoding.DecodeString("ICAgICAgIC5fXyAgICAgICAgICAgICAgICAgICAgICAuX18gIC5fXyAgX18KX19f" +
		"X19fIHxfX3wgX19fXyAgIF9fX19fX19fX19fXyB8ICB8IHxfX3wvICB8XwpcX19fXyBcfCAgfC8gX19fXCAvICBfX18vXF9fX18gXHwgIHw" +
		"gfCAgXCAgIF9fXAp8ICB8Xz4gPiAgXCAgXF9fXyBcX19fIFwgfCAgfF8+ID4gIHxffCAgfHwgIHwKfCAgIF9fL3xfX3xcX19fICA+X19fXy" +
		"AgPnwgICBfXy98X19fXy9fX3x8X198CnxfX3wgICAgICAgICAgIFwvICAgICBcLyB8X198")
)

const (
	// Default configuration values
	defaultPath  = "."
	defaultDelta = 30 * time.Minute

	// Application metadata
	appName        = "picsplit"
	appUsage       = "photo shooting event splitter and merger"
	authorName     = "Sébastien FRIESS"
	copyrightOwner = "sebastienfr"

	// Command names
	cmdMerge = "merge"

	// Flag names
	flagForce   = "force"
	flagDryRun  = "dryrun"
	flagVerbose = "verbose"
)

// InitLog initializes the logrus logger
func InitLog(verbose bool) {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors:    true,
		DisableTimestamp: true,
	})

	logrus.SetOutput(os.Stdout)

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.WarnLevel)
	}
}

func main() {
	// customize version flag
	cli.VersionFlag = &cli.BoolFlag{
		Name:    "print-version",
		Aliases: []string{"V"},
	}

	// new app
	timeStmp, err := strconv.Atoi(BuildStmp)
	if err != nil {
		timeStmp = 0
	}

	app := &cli.App{
		Name:  appName,
		Usage: appUsage,
		Version: Version + ", build on " + time.Unix(int64(timeStmp), 0).String() +
			", git hash " + GitHash,
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
					&cli.BoolFlag{
						Name:    flagVerbose,
						Aliases: []string{"v"},
						Usage:   "Print detailed logs",
					},
				},
				Action: func(c *cli.Context) error {
					// Init logger
					InitLog(c.Bool(flagVerbose))

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

					// Debug info
					logrus.Debugf("Merge configuration:")
					logrus.Debugf("  Sources: %v", sourceFolders)
					logrus.Debugf("  Target: %s", targetFolder)
					logrus.Debugf("  Force: %t", c.Bool(flagForce))
					logrus.Debugf("  DryRun: %t", c.Bool(flagDryRun))

					// Execute merge
					cfg := &handler.MergeConfig{
						SourceFolders: sourceFolders,
						TargetFolder:  targetFolder,
						Force:         c.Bool(flagForce),
						DryRun:        c.Bool(flagDryRun),
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
		&cli.BoolFlag{
			Name:        "verbose",
			Aliases:     []string{"v"},
			Destination: &verbose,
			Usage:       "Print debug information",
		},
		&cli.BoolFlag{
			Name:        "use-exif",
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
	}

	// main action
	// sub action are also possible
	app.Action = func(c *cli.Context) error {
		// init log options from command line params
		InitLog(verbose)

		// print header
		fmt.Println(string(header))

		if c.NArg() == 1 {
			path = c.Args().Get(0)
		} else if c.NArg() > 1 {
			return fmt.Errorf("wrong count of argument %d, a unique path is required", c.NArg())
		}

		logrus.Debug("* ----------------------------------------------------- *")
		logrus.Debugf("| path                 : %s", path)
		logrus.Debugf("| delta duration (min) : %0.f", durationDelta.Minutes())
		logrus.Debugf("| dry run              : %t", dryRun)
		logrus.Debugf("| no move movies       : %t", noMoveMovie)
		logrus.Debugf("| no move raw          : %t", noMoveRaw)
		logrus.Debugf("| verbose              : %t", verbose)
		logrus.Debugf("| use EXIF             : %t", useEXIF)
		logrus.Debugf("| use GPS clustering   : %t", useGPS)
		logrus.Debugf("| GPS radius (meters)  : %.0f", gpsRadius)
		logrus.Debug("* ----------------------------------------------------- *")

		// check path exists
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			return fmt.Errorf("provided path %s is not a directory", path)
		}

		cfg := &handler.Config{
			BasePath:    path,
			Delta:       durationDelta,
			NoMoveMovie: noMoveMovie,
			NoMoveRaw:   noMoveRaw,
			DryRun:      dryRun,
			UseEXIF:     useEXIF,
			UseGPS:      useGPS,
			GPSRadius:   gpsRadius,
		}
		return handler.Split(cfg)
	}

	// run the app
	err = app.Run(os.Args)
	if err != nil {
		logrus.Fatalf("runtime error %q\n", err)
	}

	os.Exit(0)
}
