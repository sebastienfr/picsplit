package main

import (
	"encoding/base64"
	"fmt"
	"github.com/sebastienfr/picsplit/handler"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"time"
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

	// delta -delta : change the default (1h) delta time between 2 events to be split
	durationDelta time.Duration = 1 * time.Hour

	// movie -nomvmov : do not move the movie files in a separate subfolder called mov
	noMoveMovie bool = false

	// raw -nomvraw : move the raw files in a separate subfolder called raw
	noMoveRaw bool = false

	// dryRun -dryrun : print the modification to be done without really moving the files
	dryRun = false

	// verbose -v : print the detailed logs to the output
	verbose = false

	header, _ = base64.StdEncoding.DecodeString("ICAgICAgIC5fXyAgICAgICAgICAgICAgICAgICAgICAuX18gIC5fXyAgX18KX19f" +
		"X19fIHxfX3wgX19fXyAgIF9fX19fX19fX19fXyB8ICB8IHxfX3wvICB8XwpcX19fXyBcfCAgfC8gX19fXCAvICBfX18vXF9fX18gXHwgIHw" +
		"gfCAgXCAgIF9fXAp8ICB8Xz4gPiAgXCAgXF9fXyBcX19fIFwgfCAgfF8+ID4gIHxffCAgfHwgIHwKfCAgIF9fL3xfX3xcX19fICA+X19fXy" +
		"AgPnwgICBfXy98X19fXy9fX3x8X198CnxfX3wgICAgICAgICAgIFwvICAgICBcLyB8X198")
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
	cli.VersionFlag = cli.BoolFlag{Name: "print-version, V"}

	// new app
	app := cli.NewApp()
	app.Name = "picsplit"
	app.Usage = "picture event spliter"

	timeStmp, err := strconv.Atoi(BuildStmp)
	if err != nil {
		timeStmp = 0
	}

	app.Version = Version + ", build on " + time.Unix(int64(timeStmp), 0).String() + ", git hash " + GitHash
	app.Authors = []cli.Author{{Name: "sfr"}}
	app.Copyright = "Sfeir " + strconv.Itoa(time.Now().Year())

	// command line flags
	app.Flags = []cli.Flag{
		cli.DurationFlag{
			Destination: &durationDelta,
			Value:       durationDelta,
			Name:        "delta, d",
			Usage:       "The duration between two files, default 1h",
		},
		cli.BoolFlag{
			Destination: &noMoveMovie,
			Name:        "nomvmov, nmm",
			Usage:       "Move movies in a separate mov folder, default true",
		},
		cli.BoolFlag{
			Destination: &noMoveRaw,
			Name:        "nomvraw, nmr",
			Usage:       "Move raw files in a separate raw folder, default true",
		},
		cli.BoolFlag{
			Destination: &dryRun,
			Name:        "dryrun, dr",
			Usage:       "Only print actions to do, do not move physically the files",
		},
		cli.BoolFlag{
			Destination: &verbose,
			Name:        "verbose, v",
			Usage:       "Print debug information",
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
			path = c.Args()[0]
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
		logrus.Debug("* ----------------------------------------------------- *")

		// check path exists
		fi, err := os.Stat(path)
		if err != nil {
			return err
		}

		if !fi.IsDir() {
			return fmt.Errorf("provided path %s is not a directory", path)
		}

		return handler.Split(path, durationDelta, noMoveMovie, noMoveRaw, dryRun)
	}

	// run the app
	err = app.Run(os.Args)
	if err != nil {
		logrus.Fatalf("runtime error %q\n", err)
	}

	os.Exit(0)
}
