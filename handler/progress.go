package handler

import (
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

// createProgressBar creates a progress bar if conditions are met
// Returns nil if progress bar should not be displayed
func createProgressBar(total int, description string, logLevel string, logFormat string) *progressbar.ProgressBar {
	isDebug := strings.ToLower(logLevel) == "debug"
	isJSON := strings.ToLower(logFormat) == "json"

	// Don't show progress bar in debug or json mode
	if isDebug || isJSON {
		return nil
	}

	// Always show progress bar in run mode (even if not TTY)
	// Use stderr to avoid interfering with piped stdout
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(50*time.Millisecond),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionOnCompletion(func() {
			println()
		}),
	)

	return bar
}
