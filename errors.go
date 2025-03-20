package onelog

import (
	"errors"
	"fmt"
)

// Common errors used throughout the package.
var (
	// ErrBufferFull is returned when the async buffer is full.
	ErrBufferFull = errors.New("onelog: buffer full")
	// ErrInvalidLevel is returned when an invalid log level is provided.
	ErrInvalidLevel = errors.New("onelog: invalid log level")
	// ErrInvalidFormatter is returned when an invalid formatter is provided.
	ErrInvalidFormatter = errors.New("onelog: invalid formatter")
	// ErrInvalidWriter is returned when an invalid writer is provided.
	ErrInvalidWriter = errors.New("onelog: invalid writer")
	// ErrWriteFailed is returned when a write operation fails.
	ErrWriteFailed = errors.New("onelog: write failed")
	// ErrLoggerClosed is returned when a closed logger is used.
	ErrLoggerClosed = errors.New("onelog: logger closed")
	// ErrFieldNotFound is returned when a field is not found.
	ErrFieldNotFound = errors.New("onelog: field not found")
)

// WrapError wraps an error with a message.
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// WrapErrorf wraps an error with a formatted message.
func WrapErrorf(err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}

// IsBufferFullError returns whether the error is an ErrBufferFull.
func IsBufferFullError(err error) bool {
	return errors.Is(err, ErrBufferFull)
}

// IsInvalidLevelError returns whether the error is an ErrInvalidLevel.
func IsInvalidLevelError(err error) bool {
	return errors.Is(err, ErrInvalidLevel)
}

// IsInvalidFormatterError returns whether the error is an ErrInvalidFormatter.
func IsInvalidFormatterError(err error) bool {
	return errors.Is(err, ErrInvalidFormatter)
}

// IsInvalidWriterError returns whether the error is an ErrInvalidWriter.
func IsInvalidWriterError(err error) bool {
	return errors.Is(err, ErrInvalidWriter)
}

// IsWriteFailedError returns whether the error is an ErrWriteFailed.
func IsWriteFailedError(err error) bool {
	return errors.Is(err, ErrWriteFailed)
}

// IsLoggerClosedError returns whether the error is an ErrLoggerClosed.
func IsLoggerClosedError(err error) bool {
	return errors.Is(err, ErrLoggerClosed)
}

// IsFieldNotFoundError returns whether the error is an ErrFieldNotFound.
func IsFieldNotFoundError(err error) bool {
	return errors.Is(err, ErrFieldNotFound)
}