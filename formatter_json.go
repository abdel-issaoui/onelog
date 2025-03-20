package onelog

import (
	"bytes"
	"io"
	"strconv"
	"sync"
	"time"
)

// JSONFormatter formats log entries as JSON.
type JSONFormatter struct {
	// Options contains the formatter options.
	Options FormatterOptions
	// DisableHTMLEscape disables HTML escaping.
	DisableHTMLEscape bool
	// timeCache caches formatted time strings
	timeCache *sync.Map
}

// NewJSONFormatter creates a new JSONFormatter with default options.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		Options:   DefaultFormatterOptions(),
		timeCache: &sync.Map{},
	}
}

// getCachedTimeString gets a cached time string or formats a new one
func (f *JSONFormatter) getCachedTimeString(t time.Time, format string) string {
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

// Format formats a log entry as JSON.
func (f *JSONFormatter) Format(w io.Writer, e *Entry) error {
	buf := GetBuffer(512) // Pre-allocate a reasonable size
	defer PutBuffer(buf)

	// Start the JSON object
	buf.WriteByte('{')

	// Track if we need to add a comma
	needComma := false

	// Write the timestamp
	if !f.Options.NoTimestamp {
		buf.WriteString("\"")
		buf.WriteString(f.Options.TimeKey)
		buf.WriteString("\":\"")
		
		// Use cached time string when possible
		timeStr := f.getCachedTimeString(e.time, f.Options.TimeFormat)
		buf.WriteString(timeStr)
		
		buf.WriteString("\"")
		needComma = true
	}

	// Write the level
	if !f.Options.NoLevel && e.level < Disabled {
		if needComma {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.LevelKey)
		buf.WriteString("\":\"")
		buf.WriteString(e.level.String())
		buf.WriteString("\"")
		needComma = true
	}

	// Write the message
	if e.message != "" {
		if needComma {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.MessageKey)
		buf.WriteString("\":\"")
		writeEscapedStringOptimized(buf, e.message)
		buf.WriteString("\"")
		needComma = true
	}

	// Write the caller info
	if e.callerInfo != nil {
		if needComma {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.CallerKey)
		buf.WriteString("\":{\"file\":\"")
		writeEscapedStringOptimized(buf, e.callerInfo.File)
		buf.WriteString("\",\"line\":")
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(e.callerInfo.Line), 10))
		buf.WriteString(",\"function\":\"")
		writeEscapedStringOptimized(buf, e.callerInfo.Function)
		buf.WriteString("\"}")
		needComma = true
	}

	// Write the fields
	for _, field := range e.fields {
		if needComma {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		writeEscapedStringOptimized(buf, f.Options.FieldNameConverter(field.Key))
		buf.WriteString("\":")

		// Format the field value
		formatJSONFieldValue(buf, field, f.Options)
		
		needComma = true
	}

	// End the JSON object
	buf.WriteByte('}')

	// Add a newline if not disabled
	if !f.Options.DisableNewline {
		buf.WriteByte('\n')
	}

	// Write the buffer to the writer
	_, err := w.Write(buf.Bytes())
	return err
}

// formatJSONFieldValue formats a field value as JSON.
func formatJSONFieldValue(buf *bytes.Buffer, field Field, opts FormatterOptions) {
	// If the field is sensitive, use the redacted value
	if field.IsSensitive {
		buf.WriteByte('"')
		buf.WriteString(opts.RedactedValue)
		buf.WriteByte('"')
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
		buf.WriteByte('"')
		if opts.TruncateStrings > 0 && len(field.String) > opts.TruncateStrings {
			writeEscapedStringOptimized(buf, field.String[:opts.TruncateStrings])
			buf.WriteString("...")
		} else {
			writeEscapedStringOptimized(buf, field.String)
		}
		buf.WriteByte('"')
	case TimeType:
		t, ok := field.Interface.(time.Time)
		if !ok {
			buf.WriteString("null")
		} else {
			buf.WriteByte('"')
			buf.WriteString(t.Format(opts.TimeFormat))
			buf.WriteByte('"')
		}
	case DurationType:
		d, ok := field.Interface.(time.Duration)
		if !ok {
			buf.WriteString("null")
		} else {
			buf.WriteByte('"')
			buf.WriteString(d.String())
			buf.WriteByte('"')
		}
	case ErrorType:
		buf.WriteByte('"')
		writeEscapedStringOptimized(buf, field.String)
		buf.WriteByte('"')
	case ObjectType:
		// For simplicity, quote the stringified value
		buf.WriteByte('"')
		writeEscapedStringOptimized(buf, stringifyValue(field.Interface))
		buf.WriteByte('"')
	case ArrayType:
		// For simplicity, quote the stringified value
		buf.WriteByte('"')
		writeEscapedStringOptimized(buf, stringifyValue(field.Interface))
		buf.WriteByte('"')
	case BinaryType:
		data, ok := field.Interface.([]byte)
		if !ok || data == nil {
			buf.WriteString("null")
		} else {
			buf.WriteByte('"')
			// Use base64 encoding for binary data
			encodedLen := base64EncodedLen(len(data))
			if buf.Available() < encodedLen {
				buf.Grow(encodedLen)
			}
			encodeBase64(buf, data)
			buf.WriteByte('"')
		}
	default:
		buf.WriteString("null")
	}
}