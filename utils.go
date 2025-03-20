package onelog

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	hex = "0123456789abcdef"
)

// itoa appends the string form of the integer i to dst and returns
// the extended buffer.
func itoa(dst *bytes.Buffer, i int64, base int) error {
	s := strconv.FormatInt(i, base)
	_, err := dst.WriteString(s)
	return err
}

// uitoa appends the string form of the unsigned integer i to dst and returns
// the extended buffer.
func uitoa(dst *bytes.Buffer, i uint64, base int) error {
	s := strconv.FormatUint(i, base)
	_, err := dst.WriteString(s)
	return err
}

// ftoa appends the string form of the float f to dst and returns the
// extended buffer.
func ftoa(dst *bytes.Buffer, f float64) error {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	_, err := dst.WriteString(s)
	return err
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
func appendEscapedString(dst *bytes.Buffer, s string) error {
	for i := 0; i < len(s); i++ {
		c := s[i]
		
		if c < utf8.RuneSelf && c >= ' ' && c != '\\' && c != '"' {
			err := dst.WriteByte(c)
			if err != nil {
				return err
			}
			continue
		}
		
		switch c {
		case '\\', '"':
			if err := dst.WriteByte('\\'); err != nil {
				return err
			}
			if err := dst.WriteByte(c); err != nil {
				return err
			}
		case '\n':
			if err := dst.WriteByte('\\'); err != nil {
				return err
			}
			if err := dst.WriteByte('n'); err != nil {
				return err
			}
		case '\r':
			if err := dst.WriteByte('\\'); err != nil {
				return err
			}
			if err := dst.WriteByte('r'); err != nil {
				return err
			}
		case '\t':
			if err := dst.WriteByte('\\'); err != nil {
				return err
			}
			if err := dst.WriteByte('t'); err != nil {
				return err
			}
		default:
			if err := dst.WriteByte('\\'); err != nil {
				return err
			}
			if err := dst.WriteByte('u'); err != nil {
				return err
			}
			if err := dst.WriteByte('0'); err != nil {
				return err
			}
			if err := dst.WriteByte('0'); err != nil {
				return err
			}
			if err := dst.WriteByte(hex[c>>4]); err != nil {
				return err
			}
			if err := dst.WriteByte(hex[c&0xF]); err != nil {
				return err
			}
		}
	}
	
	return nil
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
		buf = make([]byte, 32*1024)
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
	
	if !needsEscaping(s) {
		return s
	}
	
	buf := new(bytes.Buffer)
	
	appendEscapedString(buf, s)
	return buf.String()
}

// needsEscaping returns true if the string contains
// characters that need to be escaped.
func needsEscaping(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < utf8.RuneSelf && c < ' ' || c == '\\' || c == '"' {
			return true
		}
	}
	return false
}

// SensitiveKeys contains keys that should be redacted in logs.
var SensitiveKeys = []string{
	"password", "passwd", "secret", "token", "auth",
	"credential", "credentials", "api_key", "apikey",
	"access_token", "accesstoken", "refresh_token",
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

// toLower converts a string to lowercase without allocations.
func toLower(s string) string {
	hasUpper := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return s
	}
	
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