// Package onelog provides a high-performance, structured logging package
// optimized for speed, low memory allocation, and high throughput.
package onelog

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// Level represents the severity level of a log message.
type Level uint32

// Log levels.
const (
	// TraceLevel is the lowest level, used for detailed debugging information.
	TraceLevel Level = iota
	// DebugLevel is used for debugging information.
	DebugLevel
	// InfoLevel is used for general operational information.
	InfoLevel
	// WarnLevel is used for warnings that might require attention.
	WarnLevel
	// ErrorLevel is used for errors that should be addressed.
	ErrorLevel
	// FatalLevel is used for critical errors that require immediate attention.
	// Logging at this level typically calls os.Exit(1).
	FatalLevel
	// Disabled turns off all logging.
	Disabled
)

var levelNames = [...]string{
	TraceLevel: "TRACE",
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
	Disabled:   "DISABLED",
}

var levelMap = map[string]Level{
	"TRACE":    TraceLevel,
	"DEBUG":    DebugLevel,
	"INFO":     InfoLevel,
	"WARN":     WarnLevel,
	"ERROR":    ErrorLevel,
	"FATAL":    FatalLevel,
	"DISABLED": Disabled,
}

// String returns the string representation of the log level.
func (l Level) String() string {
	if l >= Level(len(levelNames)) {
		return fmt.Sprintf("Level(%d)", l)
	}
	return levelNames[l]
}

// ParseLevel parses a level string into a Level. It's case-insensitive.
// Returns an error if the level string is invalid.
func ParseLevel(levelStr string) (Level, error) {
	upperLevelStr := strings.ToUpper(levelStr)
	level, ok := levelMap[upperLevelStr]
	if !ok {
		return InfoLevel, fmt.Errorf("unknown level: %s", levelStr)
	}
	return level, nil
}

// MustParseLevel parses a level string into a Level. It's case-insensitive.
// Panics if the level string is invalid.
func MustParseLevel(levelStr string) Level {
	level, err := ParseLevel(levelStr)
	if err != nil {
		panic(err)
	}
	return level
}

// AtomicLevel is an atomic wrapper around a Level value.
type AtomicLevel struct {
	level uint32
}

// NewAtomicLevel creates a new AtomicLevel with the given level.
func NewAtomicLevel(level Level) *AtomicLevel {
	return &AtomicLevel{
		level: uint32(level),
	}
}

// Level returns the wrapped Level value.
func (a *AtomicLevel) Level() Level {
	return Level(atomic.LoadUint32(&a.level))
}

// SetLevel sets the Level value.
func (a *AtomicLevel) SetLevel(level Level) {
	atomic.StoreUint32(&a.level, uint32(level))
}

// Enabled returns true if the given level is enabled.
func (a *AtomicLevel) Enabled(level Level) bool {
	return level >= Level(atomic.LoadUint32(&a.level))
}

// String returns the string representation of the AtomicLevel.
func (a *AtomicLevel) String() string {
	return a.Level().String()
}