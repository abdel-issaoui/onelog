package onelog

import (
	"bytes"
	"fmt"
	"io"
	"sort"
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
}

// NewLogfmtFormatter creates a new LogfmtFormatter with default options.
func NewLogfmtFormatter() *LogfmtFormatter {
	return &LogfmtFormatter{
		Options:        DefaultFormatterOptions(),
		DisableQuoting: false,
		DisableSorting: false,
	}
}

// Format formats a log entry as logfmt.
func (f *LogfmtFormatter) Format(w io.Writer, e *Entry) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	
	// Write the timestamp
	if !f.Options.NoTimestamp {
		buf.WriteString(f.Options.TimeKey)
		buf.WriteByte('=')
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
		buf.WriteString(e.time.Format(f.Options.TimeFormat))
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
		writeInt64(buf, int64(e.callerInfo.Line))
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	}
	
	// Get the fields
	fields := e.fields
	if !f.DisableSorting {
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

// formatFieldValue formats a field value.
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
		writeInt64(buf, field.Integer)
	case UintType, Uint64Type:
		writeUint64(buf, uint64(field.Integer))
	case Float32Type, Float64Type:
		writeFloat64(buf, field.Float)
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
		writeEscapedLogfmtString(buf, string([]byte(fmt.Sprintf("%v", field.Interface))))
		if !f.DisableQuoting {
			buf.WriteByte('"')
		}
	default:
		buf.WriteString("null")
	}
}

// writeEscapedLogfmtString writes an escaped string to the buffer.
func writeEscapedLogfmtString(buf *bytes.Buffer, s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\', '"', ' ', '=':
			buf.WriteByte('\\')
			buf.WriteByte(c)
		case '\n':
			buf.WriteByte('\\')
			buf.WriteByte('n')
		case '\r':
			buf.WriteByte('\\')
			buf.WriteByte('r')
		case '\t':
			buf.WriteByte('\\')
			buf.WriteByte('t')
		default:
			buf.WriteByte(c)
		}
	}
}