package handler

import (
	"log/slog"
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
	isTTY := isatty.IsTerminal(os.Stdout.Fd())
	isDebug := strings.ToLower(logLevel) == "debug"
	isJSON := strings.ToLower(logFormat) == "json"

	showProgress := isTTY && !isDebug && !isJSON

	// Log why progress bar might be disabled (only in debug mode)
	if !showProgress && isDebug {
		slog.Debug("progress bar disabled",
			"is_tty", isTTY,
			"is_debug", isDebug,
			"is_json", isJSON)
	}

	if !showProgress {
		return nil
	}

	// Create progress bar with visible output
	bar := progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionSetWidth(40),
		progressbar.OptionSetPredictTime(true),
		progressbar.OptionThrottle(50*time.Millisecond), // Update faster (50ms instead of 100ms)
		progressbar.OptionShowElapsedTimeOnFinish(),
		// Don't clear on finish so user can see the final state
		progressbar.OptionOnCompletion(func() {
			// Add a newline after completion for clean output
			println()
		}),
	)

	return bar
}
