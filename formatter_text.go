package onelog

import (
	"bytes"
	"io"
	"sort"
	"strconv"
	"sync"
	"time"
)

// TextFormatter formats log entries as human-readable text.
type TextFormatter struct {
	// Options contains the formatter options.
	Options FormatterOptions
	// FieldSeparator is the separator between fields.
	FieldSeparator string
	// EnableColors enables colored output.
	EnableColors bool
	// DisableSorting disables sorting of fields.
	DisableSorting bool
	// EnableFieldNames enables field names in the output.
	EnableFieldNames bool
	// ForceQuote forces quoting of all values.
	ForceQuote bool
	// timeCache caches formatted time strings
	timeCache *sync.Map
}

// NewTextFormatter creates a new TextFormatter with default options.
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		Options:          DefaultFormatterOptions(),
		FieldSeparator:   " ",
		EnableColors:     false,
		DisableSorting:   false,
		EnableFieldNames: true,
		ForceQuote:       false,
		timeCache:        &sync.Map{},
	}
}

// getCachedTimeString gets a cached time string or formats a new one
func (f *TextFormatter) getCachedTimeString(t time.Time, format string) string {
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

// Format formats a log entry as text.
func (f *TextFormatter) Format(w io.Writer, e *Entry) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(256) // Pre-allocate a reasonable size
	defer bufferPool.Put(buf)

	// Write the timestamp
	if !f.Options.NoTimestamp {
		// Use cached time string when possible
		timeStr := f.getCachedTimeString(e.time, f.Options.TimeFormat)
		buf.WriteString(timeStr)
		buf.WriteString(f.FieldSeparator)
	}

	// Write the level
	if !f.Options.NoLevel {
		if f.EnableColors {
			levelColor := getColorForLevel(e.level)
			buf.WriteString(levelColor)
			buf.WriteString(e.level.String())
			buf.WriteString(resetColor)
		} else {
			buf.WriteString(e.level.String())
		}
		buf.WriteString(f.FieldSeparator)
	}

	// Write the message
	buf.WriteString(e.message)

	// Get the fields
	fields := e.fields
	if !f.DisableSorting && len(fields) > 1 {
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Key < fields[j].Key
		})
	}

	// Write the fields
	if len(fields) > 0 {
		buf.WriteString(f.FieldSeparator)
	}

	for i, field := range fields {
		if i > 0 {
			buf.WriteString(f.FieldSeparator)
		}

		// Write the field name if enabled
		if f.EnableFieldNames {
			if f.EnableColors {
				buf.WriteString(keyColor)
			}
			buf.WriteString(f.Options.FieldNameConverter(field.Key))
			buf.WriteString("=")
			if f.EnableColors {
				buf.WriteString(resetColor)
			}
		}

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

// formatFieldValue formats a field value.
func (f *TextFormatter) formatFieldValue(buf *bytes.Buffer, field Field) {
	// If the field is sensitive, use the redacted value
	if field.IsSensitive {
		if f.ForceQuote {
			buf.WriteString("\"")
		}
		buf.WriteString(f.Options.RedactedValue)
		if f.ForceQuote {
			buf.WriteString("\"")
		}
		return
	}

	switch field.Type {
	case BoolType:
		if f.EnableColors {
			buf.WriteString(boolColor)
		}
		if field.Integer == 1 {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case IntType, Int64Type:
		if f.EnableColors {
			buf.WriteString(numberColor)
		}
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), field.Integer, 10))
	case UintType, Uint64Type:
		if f.EnableColors {
			buf.WriteString(numberColor)
		}
		buf.Write(strconv.AppendUint(buf.AvailableBuffer(), uint64(field.Integer), 10))
	case Float32Type, Float64Type:
		if f.EnableColors {
			buf.WriteString(numberColor)
		}
		buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), field.Float, 'f', -1, 64))
	case StringType:
		if f.EnableColors {
			buf.WriteString(stringColor)
		}
		if f.ForceQuote {
			buf.WriteString("\"")
		}
		if f.Options.TruncateStrings > 0 && len(field.String) > f.Options.TruncateStrings {
			buf.WriteString(field.String[:f.Options.TruncateStrings])
			buf.WriteString("...")
		} else {
			buf.WriteString(field.String)
		}
		if f.ForceQuote {
			buf.WriteString("\"")
		}
	case TimeType:
		if f.EnableColors {
			buf.WriteString(timeColor)
		}
		t, ok := field.Interface.(time.Time)
		if !ok {
			buf.WriteString("null")
		} else {
			if f.ForceQuote {
				buf.WriteString("\"")
			}
			buf.WriteString(t.Format(f.Options.TimeFormat))
			if f.ForceQuote {
				buf.WriteString("\"")
			}
		}
	case DurationType:
		if f.EnableColors {
			buf.WriteString(timeColor)
		}
		d, ok := field.Interface.(time.Duration)
		if !ok {
			buf.WriteString("null")
		} else {
			if f.ForceQuote {
				buf.WriteString("\"")
			}
			buf.WriteString(d.String())
			if f.ForceQuote {
				buf.WriteString("\"")
			}
		}
	case ErrorType:
		if f.EnableColors {
			buf.WriteString(errorStrColor)
		}
		if f.ForceQuote {
			buf.WriteString("\"")
		}
		buf.WriteString(field.String)
		if f.ForceQuote {
			buf.WriteString("\"")
		}
	case ObjectType, ArrayType, BinaryType:
		if f.EnableColors {
			buf.WriteString(defaultColor)
		}
		if f.ForceQuote {
			buf.WriteString("\"")
		}
		// Use stringifyValue helper for consistent formatting
		buf.WriteString(stringifyValue(field.Interface))
		if f.ForceQuote {
			buf.WriteString("\"")
		}
	default:
		buf.WriteString("null")
	}

	if f.EnableColors {
		buf.WriteString(resetColor)
	}
}