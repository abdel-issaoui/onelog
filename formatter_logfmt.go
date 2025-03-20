package onelog

import (
	"bytes"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"
)

// LogfmtFormatter formats log entries in logfmt format.
type LogfmtFormatter struct {
	// Options contains the formatter options.
	Options FormatterOptions
	// DisableQuoting disables quoting of values.
	DisableQuoting bool
	// DisableSorting disables sorting of fields.
	DisableSorting bool
	// timeCache caches formatted time strings
	timeCache *sync.Map
}

// NewLogfmtFormatter creates a new LogfmtFormatter with default options.
func NewLogfmtFormatter() *LogfmtFormatter {
	return &LogfmtFormatter{
		Options:        DefaultFormatterOptions(),
		DisableQuoting: false,
		DisableSorting: false,
		timeCache:      &sync.Map{},
	}
}

// getCachedTimeString gets a cached time string or formats a new one
func (f *LogfmtFormatter) getCachedTimeString(t time.Time, format string) string {
	// Use time truncated to milliseconds as cache key for better hit rate
	cacheKey := t.Truncate(time.Millisecond)
	if val, ok := f.timeCache.Load(cacheKey); ok {
		cachedVal := val.(string)
		if cachedVal != "" {
			return cachedVal
		}
	}
	
	// Format the time and cache it
	formatted := t.Format(format)
	f.timeCache.Store(cacheKey, formatted)
	return formatted
}

// Format formats a log entry as logfmt.
func (f *LogfmtFormatter) Format(w io.Writer, e *Entry) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(256) // Pre-allocate a reasonable size
	defer bufferPool.Put(buf)
	
	// Write the timestamp
	if !f.Options.NoTimestamp {
		buf.WriteString(f.Options.TimeKey)
		buf.WriteByte('=')
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		
		// Use cached time string when possible
		timeStr := f.getCachedTimeString(e.time, f.Options.TimeFormat)
		buf.WriteString(timeStr)
		
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	}
	
	// Write the level
	if !f.Options.NoLevel {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(f.Options.LevelKey)
		buf.WriteByte('=')
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		buf.WriteString(e.level.String())
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	}
	
	// Write the message
	if e.message != "" {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(f.Options.MessageKey)
		buf.WriteByte('=')
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		writeEscapedLogfmtString(buf, e.message)
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	}
	
	// Write the caller info
	if e.callerInfo != nil {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(f.Options.CallerKey)
		buf.WriteByte('=')
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		writeEscapedLogfmtString(buf, e.callerInfo.File)
		buf.WriteByte(':')
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(e.callerInfo.Line), 10))
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	}
	
	// Get the fields
	fields := e.fields
	if !f.DisableSorting && len(fields) > 1 {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Key < fields[j].Key
		})
	}
	
	// Write the fields
	for _, field := range fields {
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		
		// Write the field key
		writeEscapedLogfmtString(buf, f.Options.FieldNameConverter(field.Key))
		buf.WriteByte('=')
		
		// Format the field value
		f.formatFieldValue(buf, field)
	}
	
	// Add a newline if not disabled
	if !f.Options.DisableNewline {
		buf.WriteByte('\n')
	}
	
	// Write the buffer to the writer
	_, err := w.Write(buf.Bytes())
	return err
}

// formatFieldValue formats a field value for logfmt.
func (f *LogfmtFormatter) formatFieldValue(buf *bytes.Buffer, field Field) {
	// If the field is sensitive, use the redacted value
	if field.IsSensitive {
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		buf.WriteString(f.Options.RedactedValue)
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		return
	}
	
	switch field.Type {
	case BoolType:
		if field.Integer == 1 {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case IntType, Int64Type:
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), field.Integer, 10))
	case UintType, Uint64Type:
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), uint64(field.Integer), 10))
	case Float32Type, Float64Type:
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), field.Float, 'f', -1, 64))
	case StringType:
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		if f.Options.TruncateStrings > 0 && len(field.String) > f.Options.TruncateStrings {
			writeEscapedLogfmtString(buf, field.String[:f.Options.TruncateStrings])
			buf.WriteString("...")
		} else {
			writeEscapedLogfmtString(buf, field.String)
		}
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	case TimeType:
		t, ok := field.Interface.(time.Time)
		if !ok {
			buf.WriteString("null")
		} else {
			if !f.DisableQuoting {
				buf.WriteByte('"')
			}
			buf.WriteString(t.Format(f.Options.TimeFormat))
			if !f.DisableQuoting {
				buf.WriteByte('"')
			}
		}
	case DurationType:
		d, ok := field.Interface.(time.Duration)
		if !ok {
			buf.WriteString("null")
		} else {
			if !f.DisableQuoting {
				buf.WriteByte('"')
			}
			buf.WriteString(d.String())
			if !f.DisableQuoting {
				buf.WriteByte('"')
			}
		}
	case ErrorType:
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		writeEscapedLogfmtString(buf, field.String)
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	case ObjectType, ArrayType, BinaryType:
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		writeEscapedLogfmtString(buf, stringifyValue(field.Interface))
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	default:
		buf.WriteString("null")
	}
}

// writeEscapedLogfmtString writes an escaped string to the buffer optimized for logfmt.
func writeEscapedLogfmtString(buf *bytes.Buffer, s string) {
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' || c == '"' || c == ' ' || c == '=' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteByte('\\')
			buf.WriteByte(c)
			start = i + 1
		} else if c == '\n' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString("\\n")
			start = i + 1
		} else if c == '\r' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString("\\r")
			start = i + 1
		} else if c == '\t' {
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString("\\t")
			start = i + 1
		}
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
}