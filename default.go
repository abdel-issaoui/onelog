package onelog

import (
	"context"
	"io"
	"os"
	"time"
)

// defaultLogger is the default logger.
var defaultLogger = New(DefaultConfig())

// DefaultLogger returns the default logger.
func DefaultLogger() *Logger {
	return defaultLogger
}

// SetDefaultLogger sets the default logger.
func SetDefaultLogger(logger *Logger) {
	defaultLogger = logger
}

// SetLevel sets the level of the default logger.
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

// GetLevel returns the level of the default logger.
func GetLevel() Level {
	return defaultLogger.GetLevel()
}

// SetFormatter sets the formatter of the default logger.
func SetFormatter(formatter Formatter) {
	defaultLogger.formatter = formatter
}

// SetWriter sets the writer of the default logger.
func SetWriter(writer io.Writer) {
	defaultLogger.writer = writer
}

// SetErrorHandler sets the error handler of the default logger.
func SetErrorHandler(handler func(error)) {
	defaultLogger.errorHandler = handler
}

// SetCaller enables or disables caller information in the default logger.
func SetCaller(enabled bool) {
	defaultLogger.enableCaller = enabled
}

// SetCallerSkip sets the number of stack frames to skip in the default logger.
func SetCallerSkip(skip int) {
	defaultLogger.callerSkip = skip
}

// SetAsync enables or disables asynchronous logging in the default logger.
func SetAsync(enabled bool) {
	defaultLogger.EnableAsync = enabled
	if enabled && defaultLogger.asyncBuffer == nil {
		defaultLogger.asyncBuffer = newAsyncBuffer(8192, defaultLogger.writer)
	}
}

// SetSampler sets the sampler of the default logger.
func SetSampler(sampler Sampler) {
	defaultLogger.sampler = sampler
}

// With returns a new entry with the given fields from the default logger.
func With(fields ...Field) *Entry {
	return defaultLogger.With(fields...)
}

// WithContext returns a new entry with the given context from the default logger.
func WithContext(ctx context.Context) *Entry {
	return defaultLogger.WithContext(ctx)
}

// WriterLevel returns an io.Writer that writes to the default logger at the given level.
func WriterLevel(level Level) io.Writer {
	return defaultLogger.Writer(level)
}

// Close closes the default logger, flushing any buffered log entries.
func Close() error {
	return defaultLogger.Close()
}

// NewDevelopmentLogger returns a logger configured for development.
func NewDevelopmentLogger() *Logger {
	return New(NewConfig(
		WithLevel(DebugLevel),
		WithFormatter(NewTextFormatter()),
		WithWriter(os.Stdout),
		WithCaller(true),
		WithErrorHandler(func(err error) {
			os.Stderr.WriteString("onelog: " + err.Error() + "\n")
		}),
	))
}

// NewProductionLogger returns a logger configured for production.
func NewProductionLogger() *Logger {
	return New(NewConfig(
		WithLevel(InfoLevel),
		WithFormatter(NewJSONFormatter()),
		WithWriter(os.Stdout),
		WithAsync(true),
		WithAsyncBufferSize(32768),
		WithBackpressureMode(DropMode),
		WithSampling(true),
		WithSampler(NewRateSampler(100)),
		WithRedactSensitiveFields(true),
		WithErrorHandler(func(err error) {
			os.Stderr.WriteString("onelog: " + err.Error() + "\n")
		}),
	))
}

// NewHighPerformanceLogger returns a logger optimized for high performance.
func NewHighPerformanceLogger() *Logger {
	return New(NewConfig(
		WithLevel(InfoLevel),
		WithFormatter(&JSONFormatter{
			Options: FormatterOptions{
				DisableQuote:  false,
				DisableEscape: false,
				PrettyPrint:   false,
				TimeFormat:    time.RFC3339Nano,
			},
		}),
		WithWriter(os.Stdout),
		WithAsync(true),
		WithAsyncBufferSize(65536),
		WithBackpressureMode(DropMode),
		WithSampling(true),
		WithSampler(NewAdaptiveSampler(1, 1000, 1*time.Second, 10000, 0.9)),
		WithCaller(false),
		WithDynamicBufferResizing(true),
		WithBufferResizeThreshold(75),
		WithFlushInterval(100 * time.Millisecond),
		WithRedactSensitiveFields(true),
		WithErrorHandler(func(err error) {
			os.Stderr.WriteString("onelog: " + err.Error() + "\n")
		}),
	))
}

// NewQuietLogger returns a logger that only outputs error and fatal logs.
func NewQuietLogger() *Logger {
	return New(NewConfig(
		WithLevel(ErrorLevel),
		WithFormatter(NewTextFormatter()),
		WithWriter(os.Stderr),
	))
}

// NewVerboseLogger returns a logger that outputs all logs including trace level.
func NewVerboseLogger() *Logger {
	return New(NewConfig(
		WithLevel(TraceLevel),
		WithFormatter(NewTextFormatter()),
		WithWriter(os.Stdout),
		WithCaller(true),
	))
}

// NewFileLogger returns a logger that writes to a file.
func NewFileLogger(filename string) (*Logger, error) {
	fileWriter, err := NewFileWriter(filename)
	if err != nil {
		return nil, err
	}
	
	return New(NewConfig(
		WithLevel(InfoLevel),
		WithFormatter(NewJSONFormatter()),
		WithWriter(fileWriter),
		WithAsync(true),
	)), nil
}

// NewRotatingFileLogger returns a logger that writes to a rotating file.
func NewRotatingFileLogger(filename string, maxSize int64, maxAge time.Duration, maxBackups int) (*Logger, error) {
	fileWriter, err := NewFileWriter(
		filename,
		WithMaxSize(maxSize),
		WithMaxAge(maxAge),
		WithMaxBackups(maxBackups),
		WithCompress(true),
	)
	if err != nil {
		return nil, err
	}
	
	return New(NewConfig(
		WithLevel(InfoLevel),
		WithFormatter(NewJSONFormatter()),
		WithWriter(fileWriter),
		WithAsync(true),
	)), nil
}

// NewConsoleAndFileLogger returns a logger that writes to both console and file.
func NewConsoleAndFileLogger(filename string) (*Logger, error) {
	fileWriter, err := NewFileWriter(filename)
	if err != nil {
		return nil, err
	}
	
	consoleWriter := NewConsoleWriter()
	multiWriter := NewMultiWriter(consoleWriter, fileWriter)
	
	return New(NewConfig(
		WithLevel(InfoLevel),
		WithFormatter(NewJSONFormatter()),
		WithWriter(multiWriter),
		WithAsync(true),
	)), nil
}

// Package-level logging functions

// Trace logs a message at the trace level with the default logger.
func Trace(msg string, fields ...Field) {
	defaultLogger.Trace(msg, fields...)
}

// Debug logs a message at the debug level with the default logger.
func Debug(msg string, fields ...Field) {
	defaultLogger.Debug(msg, fields...)
}

// Info logs a message at the info level with the default logger.
func Info(msg string, fields ...Field) {
	defaultLogger.Info(msg, fields...)
}

// Warn logs a message at the warn level with the default logger.
func Warn(msg string, fields ...Field) {
	defaultLogger.Warn(msg, fields...)
}

// Error logs a message at the error level with the default logger.
func Error(msg string, fields ...Field) {
	defaultLogger.Error(msg, fields...)
}

// Fatal logs a message at the fatal level with the default logger and calls os.Exit(1).
func Fatal(msg string, fields ...Field) {
	defaultLogger.Fatal(msg, fields...)
}

// Tracef logs a formatted message at the trace level with the default logger.
func Tracef(format string, args ...interface{}) {
	defaultLogger.Tracef(format, args...)
}

// Debugf logs a formatted message at the debug level with the default logger.
func Debugf(format string, args ...interface{}) {
	defaultLogger.Debugf(format, args...)
}

// Infof logs a formatted message at the info level with the default logger.
func Infof(format string, args ...interface{}) {
	defaultLogger.Infof(format, args...)
}

// Warnf logs a formatted message at the warn level with the default logger.
func Warnf(format string, args ...interface{}) {
	defaultLogger.Warnf(format, args...)
}

// Errorf logs a formatted message at the error level with the default logger.
func Errorf(format string, args ...interface{}) {
	defaultLogger.Errorf(format, args...)
}

// Fatalf logs a formatted message at the fatal level with the default logger and calls os.Exit(1).
func Fatalf(format string, args ...interface{}) {
	defaultLogger.Fatalf(format, args...)
}