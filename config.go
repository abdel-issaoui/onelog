// Package onelog provides a high-performance, structured logging package
// optimized for speed, low memory allocation, and high throughput.
package onelog

import (
	"io"
	"os"
	"time"
)

// Config contains the configuration for a logger.
type Config struct {
	// Level is the minimum log level.
	Level Level
	// Formatter is the log formatter.
	Formatter Formatter
	// Writer is the log writer.
	Writer io.Writer
	// ErrorHandler is called when an error occurs.
	ErrorHandler func(error)
	// EnableCaller enables caller information.
	EnableCaller bool
	// CallerSkip is the number of stack frames to skip when getting caller info.
	CallerSkip int
	// EnableAsync enables asynchronous logging.
	EnableAsync bool
	// AsyncBufferSize is the size of the async buffer.
	AsyncBufferSize int
	// BackpressureMode is the mode for handling backpressure.
	BackpressureMode BackpressureMode
	// EnableSampling enables log sampling.
	EnableSampling bool
	// Sampler is the log sampler.
	Sampler Sampler
	// Hooks are functions called for each log entry.
	Hooks []Hook
	// RedactSensitiveFields enables redaction of sensitive fields.
	RedactSensitiveFields bool
	// AdditionalSensitiveKeys are additional keys to redact.
	AdditionalSensitiveKeys []string
	// EnableDynamicBufferResizing enables dynamic buffer resizing.
	EnableDynamicBufferResizing bool
	// BufferResizeThreshold is the buffer utilization threshold for resizing.
	BufferResizeThreshold int
	// FlushInterval is the interval for flushing the async buffer.
	FlushInterval time.Duration
}

// Option is a function that configures a Config.
type Option func(*Config)

// WithLevel sets the log level.
func WithLevel(level Level) Option {
	return func(c *Config) {
		c.Level = level
	}
}

// WithFormatter sets the log formatter.
func WithFormatter(formatter Formatter) Option {
	return func(c *Config) {
		c.Formatter = formatter
	}
}

// WithWriter sets the log writer.
func WithWriter(writer io.Writer) Option {
	return func(c *Config) {
		c.Writer = writer
	}
}

// WithErrorHandler sets the error handler.
func WithErrorHandler(handler func(error)) Option {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

// WithCaller enables caller information.
func WithCaller(enabled bool) Option {
	return func(c *Config) {
		c.EnableCaller = enabled
	}
}

// WithCallerSkip sets the number of stack frames to skip.
func WithCallerSkip(skip int) Option {
	return func(c *Config) {
		c.CallerSkip = skip
	}
}

// WithAsync enables asynchronous logging.
func WithAsync(enabled bool) Option {
	return func(c *Config) {
		c.EnableAsync = enabled
	}
}

// WithAsyncBufferSize sets the async buffer size.
func WithAsyncBufferSize(size int) Option {
	return func(c *Config) {
		c.AsyncBufferSize = size
	}
}

// WithBackpressureMode sets the backpressure mode.
func WithBackpressureMode(mode BackpressureMode) Option {
	return func(c *Config) {
		c.BackpressureMode = mode
	}
}

// WithSampling enables log sampling.
func WithSampling(enabled bool) Option {
	return func(c *Config) {
		c.EnableSampling = enabled
	}
}

// WithSampler sets the log sampler.
func WithSampler(sampler Sampler) Option {
	return func(c *Config) {
		c.Sampler = sampler
	}
}

// WithHooks sets the log hooks.
func WithHooks(hooks ...Hook) Option {
	return func(c *Config) {
		c.Hooks = hooks
	}
}

// WithRedactSensitiveFields enables redaction of sensitive fields.
func WithRedactSensitiveFields(enabled bool) Option {
	return func(c *Config) {
		c.RedactSensitiveFields = enabled
	}
}

// WithAdditionalSensitiveKeys sets additional keys to redact.
func WithAdditionalSensitiveKeys(keys ...string) Option {
	return func(c *Config) {
		c.AdditionalSensitiveKeys = keys
	}
}

// WithDynamicBufferResizing enables dynamic buffer resizing.
func WithDynamicBufferResizing(enabled bool) Option {
	return func(c *Config) {
		c.EnableDynamicBufferResizing = enabled
	}
}

// WithBufferResizeThreshold sets the buffer utilization threshold for resizing.
func WithBufferResizeThreshold(threshold int) Option {
	return func(c *Config) {
		c.BufferResizeThreshold = threshold
	}
}

// WithFlushInterval sets the interval for flushing the async buffer.
func WithFlushInterval(interval time.Duration) Option {
	return func(c *Config) {
		c.FlushInterval = interval
	}
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:                    InfoLevel,
		Formatter:                NewTextFormatter(),
		Writer:                   os.Stdout,
		ErrorHandler:             nil,
		EnableCaller:             false,
		CallerSkip:               0,
		EnableAsync:              false,
		AsyncBufferSize:          8192,
		BackpressureMode:         DropMode,
		EnableSampling:           false,
		Sampler:                  nil,
		Hooks:                    nil,
		RedactSensitiveFields:    true,
		AdditionalSensitiveKeys:  nil,
		EnableDynamicBufferResizing: true,
		BufferResizeThreshold:    75,
		FlushInterval:            100 * time.Millisecond,
	}
}

// NewConfig creates a new configuration with the given options.
func NewConfig(options ...Option) *Config {
	config := DefaultConfig()
	
	for _, option := range options {
		option(config)
	}
	
	return config
}

// ApplyOptions applies the given options to the configuration.
func (c *Config) ApplyOptions(options ...Option) {
	for _, option := range options {
		option(c)
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Formatter == nil {
		return ErrInvalidFormatter
	}
	
	if c.Writer == nil {
		return ErrInvalidWriter
	}
	
	return nil
}

// Clone creates a copy of the configuration.
func (c *Config) Clone() *Config {
	clone := *c
	
	// Deep copy slices
	if c.Hooks != nil {
		clone.Hooks = make([]Hook, len(c.Hooks))
		copy(clone.Hooks, c.Hooks)
	}
	
	if c.AdditionalSensitiveKeys != nil {
		clone.AdditionalSensitiveKeys = make([]string, len(c.AdditionalSensitiveKeys))
		copy(clone.AdditionalSensitiveKeys, c.AdditionalSensitiveKeys)
	}
	
	return &clone
}