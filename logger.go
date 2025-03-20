package onelog

import (
	"bytes"
	"context"
	"io"
	"os"
	"runtime"
	"sync"
)

// Logger is the main struct that provides logging functionality.
type Logger struct {
	level        *AtomicLevel
	formatter    Formatter
	writer       io.Writer
	errorHandler func(error)
	fieldPool    *fieldPool
	EnableAsync  bool
	asyncBuffer  *asyncBuffer
	sampler      Sampler
	enableCaller bool
	callerSkip   int
	hooks        []Hook
}

// Hook is a function that is called for each log entry.
type Hook func(*Entry) error

var (
	// Buffer pool for temporary buffers
	bufferPool = &sync.Pool{
		New: func() interface{} {
			return &bytes.Buffer{}
		},
	}

	// Entry pool for reusing entries
	entryPool = &sync.Pool{
		New: func() interface{} {
			return &Entry{
				fields: make([]Field, 0, 8),
			}
		},
	}
	
	// Global exit function for testing
	exit = os.Exit
)

// New creates a new Logger with the given configuration.
func New(config *Config) *Logger {
	logger := &Logger{
		level:        NewAtomicLevel(config.Level),
		formatter:    config.Formatter,
		writer:       config.Writer,
		errorHandler: config.ErrorHandler,
		fieldPool:    newFieldPool(1024),
		EnableAsync:  config.EnableAsync,
		sampler:      config.Sampler,
		enableCaller: config.EnableCaller,
		callerSkip:   config.CallerSkip,
		hooks:        config.Hooks,
	}

	// Set default values if not provided
	if logger.formatter == nil {
		logger.formatter = &TextFormatter{}
	}
	if logger.writer == nil {
		logger.writer = os.Stdout
	}
	if logger.EnableAsync {
		bufferSize := config.AsyncBufferSize
		if bufferSize <= 0 {
			bufferSize = 8192 // Default buffer size
		}
		logger.asyncBuffer = newAsyncBuffer(bufferSize, logger.writer)
	}

	return logger
}

// WithLevel returns a new Logger with the given level.
func (l *Logger) WithLevel(level Level) *Logger {
	newLogger := *l
	newLogger.level = NewAtomicLevel(level)
	return &newLogger
}

// WithFormatter returns a new Logger with the given formatter.
func (l *Logger) WithFormatter(formatter Formatter) *Logger {
	newLogger := *l
	newLogger.formatter = formatter
	return &newLogger
}

// WithWriter returns a new Logger with the given writer.
func (l *Logger) WithWriter(writer io.Writer) *Logger {
	newLogger := *l
	newLogger.writer = writer
	if newLogger.EnableAsync {
		newLogger.asyncBuffer = newAsyncBuffer(newLogger.asyncBuffer.size, writer)
	}
	return &newLogger
}

// WithErrorHandler returns a new Logger with the given error handler.
func (l *Logger) WithErrorHandler(handler func(error)) *Logger {
	newLogger := *l
	newLogger.errorHandler = handler
	return &newLogger
}

// WithAsync returns a new Logger with async logging enabled or disabled.
func (l *Logger) WithAsync(enabled bool) *Logger {
	newLogger := *l
	newLogger.EnableAsync = enabled
	if enabled && newLogger.asyncBuffer == nil {
		newLogger.asyncBuffer = newAsyncBuffer(8192, newLogger.writer)
	}
	return &newLogger
}

// WithSampler returns a new Logger with the given sampler.
func (l *Logger) WithSampler(sampler Sampler) *Logger {
	newLogger := *l
	newLogger.sampler = sampler
	return &newLogger
}

// WithCaller returns a new Logger with caller information enabled or disabled.
func (l *Logger) WithCaller(enabled bool) *Logger {
	newLogger := *l
	newLogger.enableCaller = enabled
	return &newLogger
}

// WithHook returns a new Logger with the given hook added.
func (l *Logger) WithHook(hook Hook) *Logger {
	newLogger := *l
	newLogger.hooks = append(newLogger.hooks, hook)
	return &newLogger
}

// With returns a new Entry with the given fields.
func (l *Logger) With(fields ...Field) *Entry {
	e := l.newEntry()
	e.WithFields(fields)
	return e
}

// WithContext returns a new Entry with the given context.
func (l *Logger) WithContext(ctx context.Context) *Entry {
	e := l.newEntry()
	e.WithContext(ctx)
	return e
}

// Trace logs a message at the trace level.
func (l *Logger) Trace(msg string, fields ...Field) {
	if !l.level.Enabled(TraceLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Trace(msg)
}

// Debug logs a message at the debug level.
func (l *Logger) Debug(msg string, fields ...Field) {
	if !l.level.Enabled(DebugLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Debug(msg)
}

// Info logs a message at the info level.
func (l *Logger) Info(msg string, fields ...Field) {
	if !l.level.Enabled(InfoLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Info(msg)
}

// Warn logs a message at the warn level.
func (l *Logger) Warn(msg string, fields ...Field) {
	if !l.level.Enabled(WarnLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Warn(msg)
}

// Error logs a message at the error level.
func (l *Logger) Error(msg string, fields ...Field) {
	if !l.level.Enabled(ErrorLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Error(msg)
}

// Fatal logs a message at the fatal level and calls os.Exit(1).
func (l *Logger) Fatal(msg string, fields ...Field) {
	if !l.level.Enabled(FatalLevel) {
		return
	}
	e := l.newEntry()
	e.WithFields(fields)
	e.Fatal(msg)
}

// Tracef logs a formatted message at the trace level.
func (l *Logger) Tracef(format string, args ...interface{}) {
	if !l.level.Enabled(TraceLevel) {
		return
	}
	e := l.newEntry()
	e.Tracef(format, args...)
}

// Debugf logs a formatted message at the debug level.
func (l *Logger) Debugf(format string, args ...interface{}) {
	if !l.level.Enabled(DebugLevel) {
		return
	}
	e := l.newEntry()
	e.Debugf(format, args...)
}

// Infof logs a formatted message at the info level.
func (l *Logger) Infof(format string, args ...interface{}) {
	if !l.level.Enabled(InfoLevel) {
		return
	}
	e := l.newEntry()
	e.Infof(format, args...)
}

// Warnf logs a formatted message at the warn level.
func (l *Logger) Warnf(format string, args ...interface{}) {
	if !l.level.Enabled(WarnLevel) {
		return
	}
	e := l.newEntry()
	e.Warnf(format, args...)
}

// Errorf logs a formatted message at the error level.
func (l *Logger) Errorf(format string, args ...interface{}) {
	if !l.level.Enabled(ErrorLevel) {
		return
	}
	e := l.newEntry()
	e.Errorf(format, args...)
}

// Fatalf logs a formatted message at the fatal level and calls os.Exit(1).
func (l *Logger) Fatalf(format string, args ...interface{}) {
	if !l.level.Enabled(FatalLevel) {
		return
	}
	e := l.newEntry()
	e.Fatalf(format, args...)
}

// Writer returns an io.Writer that writes to the logger at the given level.
func (l *Logger) Writer(level Level) io.Writer {
	return l.newEntry().Writer(level)
}

// Close closes the logger, flushing any buffered log entries.
func (l *Logger) Close() error {
	if l.EnableAsync && l.asyncBuffer != nil {
		return l.asyncBuffer.close()
	}
	return nil
}

// SetLevel sets the logger's level.
func (l *Logger) SetLevel(level Level) {
	l.level.SetLevel(level)
}

// GetLevel returns the logger's level.
func (l *Logger) GetLevel() Level {
	return l.level.Level()
}

// writeAsync writes the given bytes to the async buffer.
func (l *Logger) writeAsync(p []byte) {
	if l.asyncBuffer == nil {
		// Fallback to synchronous write if async buffer is not initialized
		if _, err := l.writer.Write(p); err != nil && l.errorHandler != nil {
			l.errorHandler(err)
		}
		return
	}

	if err := l.asyncBuffer.write(p); err != nil && l.errorHandler != nil {
		l.errorHandler(err)
	}
}

// getCaller returns the file and line number of the caller.
func getCaller(skip int) *CallerInfo {
	pc, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return &CallerInfo{
			File:     "unknown",
			Line:     0,
			Function: "unknown",
		}
	}

	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return &CallerInfo{
		File:     file,
		Line:     line,
		Function: funcName,
	}
}