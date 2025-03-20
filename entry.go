package onelog

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"
)

// Entry represents a log entry with fields.
type Entry struct {
	logger     *Logger
	level      Level
	time       time.Time
	message    string
	fields     []Field
	fieldPool  *fieldPool
	ctx        context.Context
	callerInfo *CallerInfo
}

// CallerInfo contains information about the caller of the log function.
type CallerInfo struct {
	File     string
	Line     int
	Function string
}

// newEntry creates a new Entry.
func (l *Logger) newEntry() *Entry {
	e := entryPool.Get().(*Entry)
	e.logger = l
	e.time = time.Now()
	e.fields = e.fields[:0] // Reset fields slice
	e.fieldPool = l.fieldPool
	e.ctx = nil
	e.callerInfo = nil
	return e
}

// Enabled returns whether the given level is enabled.
func (e *Entry) Enabled() bool {
	return e.level >= e.logger.level.Level()
}

// WithField adds a field to the entry.
func (e *Entry) WithField(field Field) *Entry {
	e.fields = append(e.fields, field)
	return e
}

// WithFields adds multiple fields to the entry.
func (e *Entry) WithFields(fields []Field) *Entry {
	e.fields = append(e.fields, fields...)
	return e
}

// WithContext adds a context to the entry.
func (e *Entry) WithContext(ctx context.Context) *Entry {
	e.ctx = ctx
	return e
}

// Context returns the entry's context or context.Background() if nil.
func (e *Entry) Context() context.Context {
	if e.ctx == nil {
		return context.Background()
	}
	return e.ctx
}

// Str adds a string field to the entry.
func (e *Entry) Str(key, val string) *Entry {
	e.fields = append(e.fields, Str(key, val))
	return e
}

// Bool adds a boolean field to the entry.
func (e *Entry) Bool(key string, val bool) *Entry {
	e.fields = append(e.fields, Bool(key, val))
	return e
}

// Int adds an int field to the entry.
func (e *Entry) Int(key string, val int) *Entry {
	e.fields = append(e.fields, Int(key, val))
	return e
}

// Int64 adds an int64 field to the entry.
func (e *Entry) Int64(key string, val int64) *Entry {
	e.fields = append(e.fields, Int64(key, val))
	return e
}

// Uint adds a uint field to the entry.
func (e *Entry) Uint(key string, val uint) *Entry {
	e.fields = append(e.fields, Uint(key, val))
	return e
}

// Uint64 adds a uint64 field to the entry.
func (e *Entry) Uint64(key string, val uint64) *Entry {
	e.fields = append(e.fields, Uint64(key, val))
	return e
}

// Float32 adds a float32 field to the entry.
func (e *Entry) Float32(key string, val float32) *Entry {
	e.fields = append(e.fields, Float32(key, val))
	return e
}

// Float64 adds a float64 field to the entry.
func (e *Entry) Float64(key string, val float64) *Entry {
	e.fields = append(e.fields, Float64(key, val))
	return e
}

// Time adds a time.Time field to the entry.
func (e *Entry) Time(key string, val time.Time) *Entry {
	e.fields = append(e.fields, Time(key, val))
	return e
}

// Duration adds a time.Duration field to the entry.
func (e *Entry) Duration(key string, val time.Duration) *Entry {
	e.fields = append(e.fields, Duration(key, val))
	return e
}

// Err adds an error field to the entry.
func (e *Entry) Err(err error) *Entry {
	e.fields = append(e.fields, Err(err))
	return e
}

// NamedErr adds a named error field to the entry.
func (e *Entry) NamedErr(key string, err error) *Entry {
	e.fields = append(e.fields, NamedErr(key, err))
	return e
}

// Any adds an interface{} field to the entry.
func (e *Entry) Any(key string, val interface{}) *Entry {
	e.fields = append(e.fields, Any(key, val))
	return e
}

// Binary adds a []byte field to the entry.
func (e *Entry) Binary(key string, val []byte) *Entry {
	e.fields = append(e.fields, Binary(key, val))
	return e
}

// Array adds an array field to the entry.
func (e *Entry) Array(key string, val interface{}) *Entry {
	e.fields = append(e.fields, Array(key, val))
	return e
}

// Trace logs a message at the trace level.
func (e *Entry) Trace(msg string) {
	if !e.logger.level.Enabled(TraceLevel) {
		e.release()
		return
	}
	e.level = TraceLevel
	e.message = msg
	e.write()
}

// Debug logs a message at the debug level.
func (e *Entry) Debug(msg string) {
	if !e.logger.level.Enabled(DebugLevel) {
		e.release()
		return
	}
	e.level = DebugLevel
	e.message = msg
	e.write()
}

// Info logs a message at the info level.
func (e *Entry) Info(msg string) {
	if !e.logger.level.Enabled(InfoLevel) {
		e.release()
		return
	}
	e.level = InfoLevel
	e.message = msg
	e.write()
}

// Warn logs a message at the warn level.
func (e *Entry) Warn(msg string) {
	if !e.logger.level.Enabled(WarnLevel) {
		e.release()
		return
	}
	e.level = WarnLevel
	e.message = msg
	e.write()
}

// Error logs a message at the error level.
func (e *Entry) Error(msg string) {
	if !e.logger.level.Enabled(ErrorLevel) {
		e.release()
		return
	}
	e.level = ErrorLevel
	e.message = msg
	e.write()
}

// Fatal logs a message at the fatal level and calls os.Exit(1).
func (e *Entry) Fatal(msg string) {
	if !e.logger.level.Enabled(FatalLevel) {
		e.release()
		return
	}
	e.level = FatalLevel
	e.message = msg
	e.write()
	exit(1)
}

// Tracef logs a formatted message at the trace level.
func (e *Entry) Tracef(format string, args ...interface{}) {
	if !e.logger.level.Enabled(TraceLevel) {
		e.release()
		return
	}
	e.level = TraceLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
}

// Debugf logs a formatted message at the debug level.
func (e *Entry) Debugf(format string, args ...interface{}) {
	if !e.logger.level.Enabled(DebugLevel) {
		e.release()
		return
	}
	e.level = DebugLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
}

// Infof logs a formatted message at the info level.
func (e *Entry) Infof(format string, args ...interface{}) {
	if !e.logger.level.Enabled(InfoLevel) {
		e.release()
		return
	}
	e.level = InfoLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
}

// Warnf logs a formatted message at the warn level.
func (e *Entry) Warnf(format string, args ...interface{}) {
	if !e.logger.level.Enabled(WarnLevel) {
		e.release()
		return
	}
	e.level = WarnLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
}

// Errorf logs a formatted message at the error level.
func (e *Entry) Errorf(format string, args ...interface{}) {
	if !e.logger.level.Enabled(ErrorLevel) {
		e.release()
		return
	}
	e.level = ErrorLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
}

// Fatalf logs a formatted message at the fatal level and calls os.Exit(1).
func (e *Entry) Fatalf(format string, args ...interface{}) {
	if !e.logger.level.Enabled(FatalLevel) {
		e.release()
		return
	}
	e.level = FatalLevel
	e.message = fmt.Sprintf(format, args...)
	e.write()
	exit(1)
}

// write writes the entry to the logger's writer.
func (e *Entry) write() {
	// If sampling is enabled, check if the entry should be sampled.
	if e.logger.sampler != nil && !e.logger.sampler.Sample(e) {
		e.release()
		return
	}

	// If caller info is enabled, get the caller info.
	if e.logger.enableCaller {
		e.callerInfo = getCaller(2)
	}

	// Format and write the entry.
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)

	if err := e.logger.formatter.Format(buf, e); err != nil {
		// Handle formatting error
		if e.logger.errorHandler != nil {
			e.logger.errorHandler(err)
		}
		e.release()
		return
	}

	// Write the entry to the writer.
	if e.logger.EnableAsync {
		e.logger.writeAsync(buf.Bytes())
	} else {
		if _, err := e.logger.writer.Write(buf.Bytes()); err != nil {
			if e.logger.errorHandler != nil {
				e.logger.errorHandler(err)
			}
		}
	}

	e.release()
}

// release returns the entry to the pool.
func (e *Entry) release() {
	entryPool.Put(e)
}

// Writer returns an io.Writer that writes to the entry at the given level.
func (e *Entry) Writer(level Level) io.Writer {
	return &entryWriter{
		entry: e,
		level: level,
	}
}

// entryWriter is an io.Writer that writes to an entry.
type entryWriter struct {
	entry *Entry
	level Level
}

// Write implements io.Writer.
func (w *entryWriter) Write(p []byte) (int, error) {
	w.entry.level = w.level
	w.entry.message = string(p)
	w.entry.write()
	return len(p), nil
}