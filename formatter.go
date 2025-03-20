package onelog

import (
	"bytes"
	"io"
	"sync"
)

// Formatter defines the interface for formatting log entries.
type Formatter interface {
	// Format formats a log entry.
	Format(w io.Writer, e *Entry) error
}

// FormatterOptions contains options for formatters.
type FormatterOptions struct {
	// NoTimestamp disables the timestamp in the log entry.
	NoTimestamp bool
	// NoLevel disables the level in the log entry.
	NoLevel bool
	// NoColors disables colors in the log entry.
	NoColors bool
	// TimeFormat sets the format for timestamps.
	TimeFormat string
	// DisableQuote disables quoting of strings.
	DisableQuote bool
	// DisableEscape disables escaping of strings.
	DisableEscape bool
	// PrettyPrint enables pretty-printing for JSON output.
	PrettyPrint bool
	// TruncateStrings truncates strings longer than the given length.
	TruncateStrings int
	// RedactedValue is the value to use for redacted fields.
	RedactedValue string
	// DisableNewline disables the trailing newline in the log entry.
	DisableNewline bool
	// FieldNameConverter converts field names.
	FieldNameConverter func(string) string
	// OmitEmpty omits fields with empty values.
	OmitEmpty bool
	// TimeKey is the key for the timestamp.
	TimeKey string
	// LevelKey is the key for the level.
	LevelKey string
	// MessageKey is the key for the message.
	MessageKey string
	// CallerKey is the key for the caller info.
	CallerKey string
}

var defaultFormatterOptionsInstance *FormatterOptions
var defaultFormatterOptionsOnce sync.Once

// DefaultFormatterOptions returns the default formatter options.
func DefaultFormatterOptions() FormatterOptions {
	defaultFormatterOptionsOnce.Do(func() {
		defaultFormatterOptionsInstance = &FormatterOptions{
			NoTimestamp:      false,
			NoLevel:          false,
			NoColors:         false,
			TimeFormat:       "2006-01-02T15:04:05.000Z07:00",
			DisableQuote:     false,
			DisableEscape:    false,
			PrettyPrint:      false,
			TruncateStrings:  0,
			RedactedValue:    "[REDACTED]",
			DisableNewline:   false,
			FieldNameConverter: func(s string) string {
				return s
			},
			OmitEmpty:  false,
			TimeKey:    "time",
			LevelKey:   "level",
			MessageKey: "message",
			CallerKey:  "caller",
		}
	})

	// Return a copy to prevent modifications
	if defaultFormatterOptionsInstance != nil {
		opts := *defaultFormatterOptionsInstance
		return opts
	}
	
	// Fallback if somehow the initialization failed
	return FormatterOptions{
		NoTimestamp:      false,
		NoLevel:          false,
		NoColors:         false,
		TimeFormat:       "2006-01-02T15:04:05.000Z07:00",
		DisableQuote:     false,
		DisableEscape:    false,
		PrettyPrint:      false,
		TruncateStrings:  0,
		RedactedValue:    "[REDACTED]",
		DisableNewline:   false,
		FieldNameConverter: func(s string) string {
			return s
		},
		OmitEmpty:  false,
		TimeKey:    "time",
		LevelKey:   "level",
		MessageKey: "message",
		CallerKey:  "caller",
	}
}

// FormatField formats a field according to its type.
func FormatField(buf *bytes.Buffer, f Field, opts FormatterOptions) error {
	// If the field is sensitive, use the redacted value.
	if f.IsSensitive {
		_, err := buf.WriteString(opts.RedactedValue)
		return err
	}

	switch f.Type {
	case BoolType:
		if f.Integer == 1 {
			_, err := buf.WriteString("true")
			return err
		}
		_, err := buf.WriteString("false")
		return err
	case IntType, Int64Type:
		return writeInt64(buf, f.Integer)
	case UintType, Uint64Type:
		return writeUint64(buf, uint64(f.Integer))
	case Float32Type, Float64Type:
		return writeFloat64(buf, f.Float)
	case StringType:
		if opts.TruncateStrings > 0 && len(f.String) > opts.TruncateStrings {
			if !opts.DisableQuote {
				if err := writeQuote(buf); err != nil {
					return err
				}
			}
			if !opts.DisableEscape {
				if err := writeEscapedStringOptimized(buf, f.String[:opts.TruncateStrings]); err != nil {
					return err
				}
				if _, err := buf.WriteString("..."); err != nil {
					return err
				}
			} else {
				if _, err := buf.WriteString(f.String[:opts.TruncateStrings]); err != nil {
					return err
				}
				if _, err := buf.WriteString("..."); err != nil {
					return err
				}
			}
			if !opts.DisableQuote {
				if err := writeQuote(buf); err != nil {
					return err
				}
			}
			return nil
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		if !opts.DisableEscape {
			if err := writeEscapedStringOptimized(buf, f.String); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString(f.String); err != nil {
				return err
			}
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		return nil
	case ErrorType:
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		if !opts.DisableEscape {
			if err := writeEscapedStringOptimized(buf, f.String); err != nil {
				return err
			}
		} else {
			if _, err := buf.WriteString(f.String); err != nil {
				return err
			}
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		return nil
	case TimeType:
		t, ok := f.Interface.(interface{ Format(string) string })
		if !ok {
			_, err := buf.WriteString("null")
			return err
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(t.Format(opts.TimeFormat)); err != nil {
			return err
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		return nil
	case DurationType:
		d, ok := f.Interface.(interface{ String() string })
		if !ok {
			_, err := buf.WriteString("null")
			return err
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(d.String()); err != nil {
			return err
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		return nil
	case ObjectType, ArrayType, BinaryType:
		// For complex types, stringify and quote
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		if _, err := buf.WriteString(stringifyValue(f.Interface)); err != nil {
			return err
		}
		if !opts.DisableQuote {
			if err := writeQuote(buf); err != nil {
				return err
			}
		}
		return nil
	default:
		_, err := buf.WriteString("null")
		return err
	}
}

// writeQuote writes a double quote to the buffer.
func writeQuote(buf *bytes.Buffer) error {
	return buf.WriteByte('"')
}