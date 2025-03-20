package onelog

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	hex = "0123456789abcdef"
)

// Base64 encoding helpers
const base64EncodeTable = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// Cached UTF-8 encoding checks to avoid repeated computation
var utf8NeedsEscapeCache sync.Map

// jsonEscapeTable is a lookup table for characters that need escaping in JSON
var jsonEscapeTable = [utf8.RuneSelf]bool{
	'"':  true,
	'\\': true,
	'\n': true,
	'\r': true,
	'\t': true,
	'\b': true,
	'\f': true,
}

// writeInt64 writes an int64 to the buffer using strconv.AppendInt.
func writeInt64(buf *bytes.Buffer, i int64) error {
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), i, 10))
	return nil
}

// writeUint64 writes a uint64 to the buffer using strconv.AppendUint.
func writeUint64(buf *bytes.Buffer, i uint64) error {
	buf.Write(strconv.AppendUint(buf.AvailableBuffer(), i, 10))
	return nil
}

// writeFloat64 writes a float64 to the buffer using strconv.AppendFloat.
func writeFloat64(buf *bytes.Buffer, f float64) error {
	buf.Write(strconv.AppendFloat(buf.AvailableBuffer(), f, 'f', -1, 64))
	return nil
}

// appendQuote appends a quoted string to the buffer.
func appendQuote(dst *bytes.Buffer, s string) error {
	err := dst.WriteByte('"')
	if err != nil {
		return err
	}
	
	err = appendEscapedString(dst, s)
	if err != nil {
		return err
	}
	
	return dst.WriteByte('"')
}

// appendEscapedString appends an escaped string to the buffer.
// This is an optimized version that minimizes buffer writes.
func appendEscapedString(dst *bytes.Buffer, s string) error {
	if s == "" {
		return nil
	}
	
	// Check if string needs escaping (using cache to avoid repeated checks)
	needsEscape, ok := utf8NeedsEscapeCache.Load(s)
	if !ok {
		needsEscape = stringNeedsEscaping(s)
		// Only cache short strings to avoid memory issues
		if len(s) <= 64 {
			utf8NeedsEscapeCache.Store(s, needsEscape)
		}
	}
	
	// Fast path for strings that don't need escaping
	if !needsEscape.(bool) {
		_, err := dst.WriteString(s)
		return err
	}

	// For strings that need escaping, use optimized implementation
	start := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		
		if c < utf8.RuneSelf && (c < ' ' || c == '\\' || c == '"') {
			if start < i {
				dst.WriteString(s[start:i])
			}
			
			dst.WriteByte('\\')
			switch c {
			case '\\', '"':
				dst.WriteByte(c)
			case '\n':
				dst.WriteByte('n')
			case '\r':
				dst.WriteByte('r')
			case '\t':
				dst.WriteByte('t')
			case '\b':
				dst.WriteByte('b')
			case '\f':
				dst.WriteByte('f')
			default:
				// For other control characters, use \uXXXX format
				dst.WriteString(`u00`)
				dst.WriteByte(hex[c>>4])
				dst.WriteByte(hex[c&0xF])
			}
			
			start = i + 1
		}
	}
	
	if start < len(s) {
		dst.WriteString(s[start:])
	}
	
	return nil
}

// writeEscapedStringOptimized writes an escaped string to the buffer optimized for JSON.
// writeEscapedStringOptimized writes an escaped string to the buffer optimized for JSON.
// Returns an error if any buffer operations fail.
func writeEscapedStringOptimized(buf *bytes.Buffer, s string) error {
    start := 0
    for i := 0; i < len(s); i++ {
        c := s[i]
        if c < utf8.RuneSelf && jsonEscapeTable[c] {
            if start < i {
                if _, err := buf.WriteString(s[start:i]); err != nil {
                    return err
                }
            }
            if err := buf.WriteByte('\\'); err != nil {
                return err
            }
            switch c {
            case '"':
                if err := buf.WriteByte('"'); err != nil {
                    return err
                }
            case '\\':
                if err := buf.WriteByte('\\'); err != nil {
                    return err
                }
            case '\n':
                if err := buf.WriteByte('n'); err != nil {
                    return err
                }
            case '\r':
                if err := buf.WriteByte('r'); err != nil {
                    return err
                }
            case '\t':
                if err := buf.WriteByte('t'); err != nil {
                    return err
                }
            case '\b':
                if err := buf.WriteByte('b'); err != nil {
                    return err
                }
            case '\f':
                if err := buf.WriteByte('f'); err != nil {
                    return err
                }
            default:
                // This should never happen, as we only call this function
                // for chars in jsonEscapeTable
                if err := buf.WriteByte(c); err != nil {
                    return err
                }
            }
            start = i + 1
        }
    }
    if start < len(s) {
        if _, err := buf.WriteString(s[start:]); err != nil {
            return err
        }
    }
    return nil
}

// stringNeedsEscaping checks if a string contains characters that need escaping.
func stringNeedsEscaping(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < utf8.RuneSelf && (c < ' ' || c == '\\' || c == '"') {
			return true
		}
	}
	return false
}

// truncateString truncates a string to the given length.
func truncateString(s string, length int) string {
	if length <= 0 || len(s) <= length {
		return s
	}
	return s[:length]
}

// copyBuffer copies from src to dst until either EOF is reached
// on src or an error occurs. Similar to io.Copy but allows reusing
// a buffer across calls to reduce allocations.
func copyBuffer(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	if buf == nil {
		// Default buffer size aligned with typical log message size
		buf = make([]byte, 4096)
	}
	
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	
	return written, err
}

// SafeString returns a string that is safe to use in logs.
// It truncates the string if it's too long and replaces
// control characters with their escape sequences.
func SafeString(s string, maxLength int) string {
	if maxLength > 0 && len(s) > maxLength {
		s = s[:maxLength]
	}
	
	if !stringNeedsEscaping(s) {
		return s
	}
	
	buf := GetBuffer(len(s) * 2)
	defer PutBuffer(buf)
	
	appendEscapedString(buf, s)
	return buf.String()
}

// Base64 encoding helpers
func base64EncodedLen(n int) int {
	return (n + 2) / 3 * 4
}

// encodeBase64 encodes src into dst using base64 encoding
func encodeBase64(dst *bytes.Buffer, src []byte) {
	for len(src) >= 3 {
		// Process 3 bytes at a time
		dst.WriteByte(base64EncodeTable[(src[0]>>2)&0x3F])
		dst.WriteByte(base64EncodeTable[((src[0]&0x3)<<4)|((src[1]>>4)&0xF)])
		dst.WriteByte(base64EncodeTable[((src[1]&0xF)<<2)|((src[2]>>6)&0x3)])
		dst.WriteByte(base64EncodeTable[src[2]&0x3F])
		
		src = src[3:]
	}
	
	// Handle remaining bytes (if any)
	if len(src) > 0 {
		// Encode first byte (always present in remainder)
		dst.WriteByte(base64EncodeTable[(src[0]>>2)&0x3F])
		
		if len(src) == 1 {
			// One byte remainder
			dst.WriteByte(base64EncodeTable[(src[0]&0x3)<<4])
			dst.WriteByte('=')
			dst.WriteByte('=')
		} else {
			// Two bytes remainder
			dst.WriteByte(base64EncodeTable[((src[0]&0x3)<<4)|((src[1]>>4)&0xF)])
			dst.WriteByte(base64EncodeTable[(src[1]&0xF)<<2])
			dst.WriteByte('=')
		}
	}
}

// SensitiveKeys contains keys that should be redacted in logs.
var SensitiveKeys = []string{
	"password", "passwd", "secret", "token", "auth",
	"credential", "credentials", "api_key", "apikey",
	"access_token", "accesstoken", "refresh_token",
	"private_key", "privatekey", "authorization", "key",
}

// IsSensitiveKey returns true if the key is sensitive.
func IsSensitiveKey(key string) bool {
	lowerKey := strings.ToLower(key)
	for _, sensitiveKey := range SensitiveKeys {
		if strings.Contains(lowerKey, sensitiveKey) {
			return true
		}
	}
	return false
}

// fastLowerCase converts ASCII string to lowercase without allocations
// for short keys (optimization for key matching)
func fastLowerCase(s string) string {
	// Fast path: if string is all lowercase already, return it as is
	allLower := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			allLower = false
			break
		}
	}
	if allLower {
		return s
	}
	
	// For short strings, use a byte buffer to avoid allocations
	if len(s) <= 32 {
		buf := make([]byte, len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if c >= 'A' && c <= 'Z' {
				buf[i] = c + ('a' - 'A')
			} else {
				buf[i] = c
			}
		}
		return string(buf)
	}
	
	// For longer strings, use strings.Builder
	var b strings.Builder
	b.Grow(len(s))
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b.WriteByte(c)
	}
	
	return b.String()
}

// stringifyValue converts a value to its string representation
func stringifyValue(val interface{}) string {
	if val == nil {
		return "null"
	}
	return fmt.Sprintf("%v", val)
}

// Global buffer functions
// These act as façades to the actual buffer pool implementation

// GetBuffer gets a byte buffer with the given capacity.
func GetBuffer(capacity int) *bytes.Buffer {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	if buf.Cap() < capacity {
		buf.Grow(capacity - buf.Cap())
	}
	return buf
}

// PutBuffer returns a byte buffer to the pool.
func PutBuffer(buf *bytes.Buffer) {
	if buf != nil {
		buf.Reset()
		bufferPool.Put(buf)
	}
}