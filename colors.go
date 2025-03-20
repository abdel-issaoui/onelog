package onelog

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

const (
	// Reset resets the foreground and background colors.
	reset = "\033[0m"
	// Bold makes the foreground bold.
	bold = "\033[1m"
	// Underline underlines the foreground.
	underline = "\033[4m"
	// Blink makes the foreground blink.
	blink = "\033[5m"
	// Reverse reverses the foreground and background colors.
	reverse = "\033[7m"
	// Hidden hides the foreground.
	hidden = "\033[8m"

	// Black is the black foreground color.
	black = "\033[30m"
	// Red is the red foreground color.
	red = "\033[31m"
	// Green is the green foreground color.
	green = "\033[32m"
	// Yellow is the yellow foreground color.
	yellow = "\033[33m"
	// Blue is the blue foreground color.
	blue = "\033[34m"
	// Magenta is the magenta foreground color.
	magenta = "\033[35m"
	// Cyan is the cyan foreground color.
	cyan = "\033[36m"
	// White is the white foreground color.
	white = "\033[37m"

	// BrightBlack is the bright black foreground color.
	brightBlack = "\033[90m"
	// BrightRed is the bright red foreground color.
	brightRed = "\033[91m"
	// BrightGreen is the bright green foreground color.
	brightGreen = "\033[92m"
	// BrightYellow is the bright yellow foreground color.
	brightYellow = "\033[93m"
	// BrightBlue is the bright blue foreground color.
	brightBlue = "\033[94m"
	// BrightMagenta is the bright magenta foreground color.
	brightMagenta = "\033[95m"
	// BrightCyan is the bright cyan foreground color.
	brightCyan = "\033[96m"
	// BrightWhite is the bright white foreground color.
	brightWhite = "\033[97m"

	// BgBlack is the black background color.
	bgBlack = "\033[40m"
	// BgRed is the red background color.
	bgRed = "\033[41m"
	// BgGreen is the green background color.
	bgGreen = "\033[42m"
	// BgYellow is the yellow background color.
	bgYellow = "\033[43m"
	// BgBlue is the blue background color.
	bgBlue = "\033[44m"
	// BgMagenta is the magenta background color.
	bgMagenta = "\033[45m"
	// BgCyan is the cyan background color.
	bgCyan = "\033[46m"
	// BgWhite is the white background color.
	bgWhite = "\033[47m"

	// BgBrightBlack is the bright black background color.
	bgBrightBlack = "\033[100m"
	// BgBrightRed is the bright red background color.
	bgBrightRed = "\033[101m"
	// BgBrightGreen is the bright green background color.
	bgBrightGreen = "\033[102m"
	// BgBrightYellow is the bright yellow background color.
	bgBrightYellow = "\033[103m"
	// BgBrightBlue is the bright blue background color.
	bgBrightBlue = "\033[104m"
	// BgBrightMagenta is the bright magenta background color.
	bgBrightMagenta = "\033[105m"
	// BgBrightCyan is the bright cyan background color.
	bgBrightCyan = "\033[106m"
	// BgBrightWhite is the bright white background color.
	bgBrightWhite = "\033[107m"
)

// Colors for log levels.
var (
	// Default colors
	traceColor = cyan
	debugColor = blue
	infoColor  = green
	warnColor  = yellow
	errorColor = red
	fatalColor = brightRed

	// Special colors
	resetColor   = reset
	keyColor     = cyan
	stringColor  = green
	numberColor  = magenta
	boolColor    = yellow
	timeColor    = blue
	errorStrColor = red
	defaultColor = white

	// Whether colors are enabled
	colorsEnabled = false
)

// init initializes the colors.
func init() {
	// Check if colors should be enabled
	colorsEnabled = checkColorsEnabled()
}

// checkColorsEnabled checks if colors should be enabled.
func checkColorsEnabled() bool {
	// If on Windows, check if the terminal supports ANSI colors
	if runtime.GOOS == "windows" {
		// Check for ANSICON, TERM, CI environment variables
		_, hasAnsicon := os.LookupEnv("ANSICON")
		term, hasTerm := os.LookupEnv("TERM")
		_, hasCI := os.LookupEnv("CI")

		// Enable colors if any of these conditions are met
		return hasAnsicon || hasTerm && term != "dumb" || hasCI
	}

	// On other platforms, check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Enable colors if stdout is a terminal
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// SetColorsEnabled sets whether colors are enabled.
func SetColorsEnabled(enabled bool) {
	colorsEnabled = enabled
}

// EnableColors enables colors.
func EnableColors() {
	colorsEnabled = true
}

// DisableColors disables colors.
func DisableColors() {
	colorsEnabled = false
}

// getColorForLevel returns the ANSI color for the given log level.
func getColorForLevel(level Level) string {
	if !colorsEnabled {
		return ""
	}

	switch level {
	case TraceLevel:
		return traceColor
	case DebugLevel:
		return debugColor
	case InfoLevel:
		return infoColor
	case WarnLevel:
		return warnColor
	case ErrorLevel:
		return errorColor
	case FatalLevel:
		return fatalColor
	default:
		return defaultColor
	}
}

// Color is a type for ANSI colors.
type Color string

// SetLevelColor sets the color for the given log level.
func SetLevelColor(level Level, color Color) {
	switch level {
	case TraceLevel:
		traceColor = string(color)
	case DebugLevel:
		debugColor = string(color)
	case InfoLevel:
		infoColor = string(color)
	case WarnLevel:
		warnColor = string(color)
	case ErrorLevel:
		errorColor = string(color)
	case FatalLevel:
		fatalColor = string(color)
	}
}

// SetKeyColor sets the color for field keys.
func SetKeyColor(color Color) {
	keyColor = string(color)
}

// SetStringColor sets the color for string values.
func SetStringColor(color Color) {
	stringColor = string(color)
}

// SetNumberColor sets the color for number values.
func SetNumberColor(color Color) {
	numberColor = string(color)
}

// SetBoolColor sets the color for boolean values.
func SetBoolColor(color Color) {
	boolColor = string(color)
}

// SetTimeColor sets the color for time values.
func SetTimeColor(color Color) {
	timeColor = string(color)
}

// SetErrorColor sets the color for error values.
func SetErrorColor(color Color) {
	errorStrColor = string(color)
}

// SetDefaultColor sets the color for other values.
func SetDefaultColor(color Color) {
	defaultColor = string(color)
}

// RGB creates a custom RGB color.
func RGB(r, g, b int) Color {
	return Color(fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b))
}

// BgRGB creates a custom RGB background color.
func BgRGB(r, g, b int) Color {
	return Color(fmt.Sprintf("\033[48;2;%d;%d;%dm", r, g, b))
}

// Xterm256 creates a color using the xterm 256 color palette.
func Xterm256(code int) Color {
	return Color(fmt.Sprintf("\033[38;5;%dm", code))
}

// BgXterm256 creates a background color using the xterm 256 color palette.
func BgXterm256(code int) Color {
	return Color(fmt.Sprintf("\033[48;5;%dm", code))
}

// Combine combines multiple colors.
func Combine(colors ...Color) Color {
	var combined strings.Builder
	for _, color := range colors {
		combined.WriteString(string(color))
	}
	return Color(combined.String())
}

// Colors available for use.
var (
	Reset          = Color(reset)
	Bold           = Color(bold)
	Underline      = Color(underline)
	Blink          = Color(blink)
	Reverse        = Color(reverse)
	Hidden         = Color(hidden)
	Black          = Color(black)
	Red            = Color(red)
	Green          = Color(green)
	Yellow         = Color(yellow)
	Blue           = Color(blue)
	Magenta        = Color(magenta)
	Cyan           = Color(cyan)
	White          = Color(white)
	BrightBlack    = Color(brightBlack)
	BrightRed      = Color(brightRed)
	BrightGreen    = Color(brightGreen)
	BrightYellow   = Color(brightYellow)
	BrightBlue     = Color(brightBlue)
	BrightMagenta  = Color(brightMagenta)
	BrightCyan     = Color(brightCyan)
	BrightWhite    = Color(brightWhite)
	BgBlack        = Color(bgBlack)
	BgRed          = Color(bgRed)
	BgGreen        = Color(bgGreen)
	BgYellow       = Color(bgYellow)
	BgBlue         = Color(bgBlue)
	BgMagenta      = Color(bgMagenta)
	BgCyan         = Color(bgCyan)
	BgWhite        = Color(bgWhite)
	BgBrightBlack  = Color(bgBrightBlack)
	BgBrightRed    = Color(bgBrightRed)
	BgBrightGreen  = Color(bgBrightGreen)
	BgBrightYellow = Color(bgBrightYellow)
	BgBrightBlue   = Color(bgBrightBlue)
	BgBrightMagenta = Color(bgBrightMagenta)
	BgBrightCyan   = Color(bgBrightCyan)
	BgBrightWhite  = Color(bgBrightWhite)
)