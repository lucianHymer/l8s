package color

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset     = "\033[0m"
	Red       = "\033[31m"
	Green     = "\033[32m"
	Yellow    = "\033[33m"
	Blue      = "\033[34m"
	Magenta   = "\033[35m"
	Cyan      = "\033[36m"
	White     = "\033[37m"
	BoldStyle = "\033[1m"
)

// isColorEnabled checks if color output should be enabled
func isColorEnabled() bool {
	// Check if NO_COLOR env var is set
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	
	// Check if we're in a terminal
	if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) == 0 {
		return false
	}
	
	// Check TERM env var
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}
	
	return true
}

// colorize wraps text with color codes if color is enabled
func colorize(color, text string) string {
	if !isColorEnabled() {
		return text
	}
	return color + text + Reset
}

// Success formats success messages with green color
func Success(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(colorize(Green, message))
}

// Error formats error messages with red color
func Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, colorize(Red, message))
}

// Warning formats warning messages with yellow color
func Warning(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(colorize(Yellow, message))
}

// Info formats info messages with cyan color
func Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	fmt.Println(colorize(Cyan, message))
}

// Bold formats text in bold
func Bold(format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	return colorize(BoldStyle, message)
}

// Printf prints formatted text with optional color
func Printf(format string, args ...interface{}) {
	// Replace color markers in format string
	if isColorEnabled() {
		format = strings.ReplaceAll(format, "{green}", Green)
		format = strings.ReplaceAll(format, "{red}", Red)
		format = strings.ReplaceAll(format, "{yellow}", Yellow)
		format = strings.ReplaceAll(format, "{cyan}", Cyan)
		format = strings.ReplaceAll(format, "{bold}", BoldStyle)
		format = strings.ReplaceAll(format, "{reset}", Reset)
	} else {
		// Remove color markers if colors are disabled
		format = strings.ReplaceAll(format, "{green}", "")
		format = strings.ReplaceAll(format, "{red}", "")
		format = strings.ReplaceAll(format, "{yellow}", "")
		format = strings.ReplaceAll(format, "{cyan}", "")
		format = strings.ReplaceAll(format, "{bold}", "")
		format = strings.ReplaceAll(format, "{reset}", "")
	}
	
	fmt.Printf(format, args...)
}