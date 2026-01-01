package handler

import (
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/schollz/progressbar/v3"
)

// createProgressBar creates a progress bar if conditions are met
// Returns nil if progress bar should not be displayed
func createProgressBar(total int, description string, logLevel string, logFormat string) *progressbar.ProgressBar {
	// Don't show progress bar if:
	// - stdout is not a terminal (e.g., piped to file)
	// - log level is debug (detailed logs take priority)
	// - log format is json (structured output)
	showProgress := isatty.IsTerminal(os.Stdout.Fd()) &&
		strings.ToLower(logLevel) != "debug" &&
		strings.ToLower(logFormat) != "json"

	if !showProgress {
		return nil
	}

	return progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetRenderBlankState(false),
	)
}
