package onelog

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

// JSONFormatter formats log entries as JSON.
type JSONFormatter struct {
	// Options contains the formatter options.
	Options FormatterOptions
	// DisableHTMLEscape disables HTML escaping.
	DisableHTMLEscape bool
}

// NewJSONFormatter creates a new JSONFormatter with default options.
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		Options: DefaultFormatterOptions(),
	}
}

// Format formats a log entry as JSON.
func (f *JSONFormatter) Format(w io.Writer, e *Entry) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)

	// Start the JSON object
	buf.WriteByte('{')

	// Write the timestamp
	if !f.Options.NoTimestamp {
		buf.WriteString("\"")
		buf.WriteString(f.Options.TimeKey)
		buf.WriteString("\":\"")
		buf.WriteString(e.time.Format(f.Options.TimeFormat))
		buf.WriteString("\"")
	}

	// Write the level
	if !f.Options.NoLevel {
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.LevelKey)
		buf.WriteString("\":\"")
		buf.WriteString(e.level.String())
		buf.WriteString("\"")
	}

	// Write the message
	if e.message != "" {
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.MessageKey)
		buf.WriteString("\":\"")
		writeEscapedString(buf, e.message)
		buf.WriteString("\"")
	}

	// Write the caller info
	if e.callerInfo != nil {
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		buf.WriteString(f.Options.CallerKey)
		buf.WriteString("\":{\"file\":\"")
		writeEscapedString(buf, e.callerInfo.File)
		buf.WriteString("\",\"line\":")
		writeInt64(buf, int64(e.callerInfo.Line))
		buf.WriteString(",\"function\":\"")
		writeEscapedString(buf, e.callerInfo.Function)
		buf.WriteString("\"}")
	}

	// Write the fields
	for _, field := range e.fields {
		if buf.Len() > 1 {
			buf.WriteByte(',')
		}
		buf.WriteString("\"")
		writeEscapedString(buf, f.Options.FieldNameConverter(field.Key))
		buf.WriteString("\":")

		// Format the field value
		switch field.Type {
		case BoolType:
			if field.Integer == 1 {
				buf.WriteString("true")
			} else {
				buf.WriteString("false")
			}
		case IntType, Int64Type:
			writeInt64(buf, field.Integer)
		case UintType, Uint64Type:
			writeUint64(buf, uint64(field.Integer))
		case Float32Type, Float64Type:
			writeFloat64(buf, field.Float)
		case StringType:
			buf.WriteByte('"')
			if field.IsSensitive {
				buf.WriteString(f.Options.RedactedValue)
			} else {
				writeEscapedString(buf, field.String)
			}
			buf.WriteByte('"')
		case TimeType:
			t, ok := field.Interface.(time.Time)
			if !ok {
				buf.WriteString("null")
			} else {
				buf.WriteByte('"')
				buf.WriteString(t.Format(f.Options.TimeFormat))
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
			if field.IsSensitive {
				buf.WriteString(f.Options.RedactedValue)
			} else {
				writeEscapedString(buf, field.String)
			}
			buf.WriteByte('"')
		case ObjectType:
			// For simplicity, we'll just use a string representation
			if field.IsSensitive {
				buf.WriteByte('"')
				buf.WriteString(f.Options.RedactedValue)
				buf.WriteByte('"')
			} else {
				buf.WriteString("{\"value\":\"")
				writeEscapedString(buf, fmt.Sprintf("%v", field.Interface))
				buf.WriteString("\"}")
			}
		case ArrayType:
			// For simplicity, we'll just use a string representation
			if field.IsSensitive {
				buf.WriteByte('"')
				buf.WriteString(f.Options.RedactedValue)
				buf.WriteByte('"')
			} else {
				buf.WriteString("[\"")
				writeEscapedString(buf, fmt.Sprintf("%v", field.Interface))
				buf.WriteString("\"]")
			}
		case BinaryType:
			// For simplicity, we'll just use a string representation
			if field.IsSensitive {
				buf.WriteByte('"')
				buf.WriteString(f.Options.RedactedValue)
				buf.WriteByte('"')
			} else {
				data, ok := field.Interface.([]byte)
				if !ok {
					buf.WriteString("null")
				} else {
					buf.WriteString("\"")
					writeEscapedString(buf, string(data))
					buf.WriteString("\"")
				}
			}
		default:
			buf.WriteString("null")
		}
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