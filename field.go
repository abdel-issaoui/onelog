package onelog

import (
	"fmt"
	"time"
)

// FieldType represents the type of a field.
type FieldType uint8

// Field types.
const (
	// UnknownType is the default field type.
	UnknownType FieldType = iota
	// BoolType is a boolean field type.
	BoolType
	// IntType is an int field type.
	IntType
	// Int64Type is an int64 field type.
	Int64Type
	// UintType is a uint field type.
	UintType
	// Uint64Type is a uint64 field type.
	Uint64Type
	// Float32Type is a float32 field type.
	Float32Type
	// Float64Type is a float64 field type.
	Float64Type
	// StringType is a string field type.
	StringType
	// TimeType is a time.Time field type.
	TimeType
	// DurationType is a time.Duration field type.
	DurationType
	// ErrorType is an error field type.
	ErrorType
	// ObjectType is a structured object field type.
	ObjectType
	// ArrayType is an array field type.
	ArrayType
	// BinaryType is a []byte field type.
	BinaryType
)

// Field represents a structured log field.
type Field struct {
	Key         string
	Type        FieldType
	Integer     int64
	Float       float64
	String      string
	Interface   interface{}
	IsSensitive bool // Renamed from Sensitive to avoid collision with method
}

// Bool creates a Field with a boolean value.
func Bool(key string, val bool) Field {
	var i int64
	if val {
		i = 1
	}
	return Field{
		Key:     key,
		Type:    BoolType,
		Integer: i,
	}
}

// Int creates a Field with an int value.
func Int(key string, val int) Field {
	return Field{
		Key:     key,
		Type:    IntType,
		Integer: int64(val),
	}
}

// Int64 creates a Field with an int64 value.
func Int64(key string, val int64) Field {
	return Field{
		Key:     key,
		Type:    Int64Type,
		Integer: val,
	}
}

// Uint creates a Field with a uint value.
func Uint(key string, val uint) Field {
	return Field{
		Key:     key,
		Type:    UintType,
		Integer: int64(val),
	}
}

// Uint64 creates a Field with a uint64 value.
func Uint64(key string, val uint64) Field {
	return Field{
		Key:     key,
		Type:    Uint64Type,
		Integer: int64(val),
	}
}

// Float32 creates a Field with a float32 value.
func Float32(key string, val float32) Field {
	return Field{
		Key:     key,
		Type:    Float32Type,
		Float:   float64(val),
	}
}

// Float64 creates a Field with a float64 value.
func Float64(key string, val float64) Field {
	return Field{
		Key:     key,
		Type:    Float64Type,
		Float:   val,
	}
}

// Str creates a Field with a string value.
func Str(key string, val string) Field {
	return Field{
		Key:    key,
		Type:   StringType,
		String: val,
	}
}

// Time creates a Field with a time.Time value.
func Time(key string, val time.Time) Field {
	return Field{
		Key:       key,
		Type:      TimeType,
		Interface: val,
	}
}

// Duration creates a Field with a time.Duration value.
func Duration(key string, val time.Duration) Field {
	return Field{
		Key:       key,
		Type:      DurationType,
		Interface: val,
	}
}

// Err creates a Field with an error value.
func Err(err error) Field {
	if err == nil {
		return Field{
			Key:    "error",
			Type:   ErrorType,
			String: "",
		}
	}
	return Field{
		Key:       "error",
		Type:      ErrorType,
		Interface: err,
		String:    err.Error(),
	}
}

// NamedErr creates a Field with a named error value.
func NamedErr(key string, err error) Field {
	if err == nil {
		return Field{
			Key:    key,
			Type:   ErrorType,
			String: "",
		}
	}
	return Field{
		Key:       key,
		Type:      ErrorType,
		Interface: err,
		String:    err.Error(),
	}
}

// Any creates a Field with an interface{} value.
func Any(key string, val interface{}) Field {
	return Field{
		Key:       key,
		Type:      ObjectType,
		Interface: val,
	}
}

// Binary creates a Field with a []byte value.
func Binary(key string, val []byte) Field {
	return Field{
		Key:       key,
		Type:      BinaryType,
		Interface: val,
	}
}

// Array creates a Field with an array value.
func Array(key string, val interface{}) Field {
	return Field{
		Key:       key,
		Type:      ArrayType,
		Interface: val,
	}
}

// Sensitive marks a field as sensitive, which will be redacted in logs.
func (f Field) Sensitive() Field {
	newField := f
	newField.IsSensitive = true
	return newField
}

// GoString implements fmt.GoStringer.
func (f Field) GoString() string {
	return fmt.Sprintf("onelog.Field{Key: %q, Type: %v, Integer: %d, Float: %f, String: %q, Interface: %v}",
		f.Key, f.Type, f.Integer, f.Float, f.String, f.Interface)
}