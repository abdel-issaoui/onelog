package onelog

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// CLFFormatter formats log entries in Common Log Format (CLF).
// CLF format: %h %l %u %t "%r" %>s %b
type CLFFormatter struct {
	// Options contains the formatter options.
	Options FormatterOptions
	// ExtendedFormat enables extended format (include Referer and User-Agent).
	ExtendedFormat bool
	// timeCache caches formatted time strings
	timeCache *sync.Map
}

// NewCLFFormatter creates a new CLFFormatter with default options.
func NewCLFFormatter() *CLFFormatter {
	return &CLFFormatter{
		Options:        DefaultFormatterOptions(),
		ExtendedFormat: false,
		timeCache:      &sync.Map{},
	}
}

// getCachedTimeString gets a cached time string or formats a new one
func (f *CLFFormatter) getCachedTimeString(t time.Time) string {
	// Use time truncated to seconds as cache key for better hit rate
	cacheKey := t.Truncate(time.Second)
	if val, ok := f.timeCache.Load(cacheKey); ok {
		cachedVal := val.(string)
		if cachedVal != "" {
			return cachedVal
		}
	}
	
	// Format the time and cache it
	formatted := t.Format("02/Jan/2006:15:04:05 -0700")
	f.timeCache.Store(cacheKey, formatted)
	return formatted
}

// Format formats a log entry as CLF.
func (f *CLFFormatter) Format(w io.Writer, e *Entry) error {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Grow(256) // Pre-allocate a reasonable size
	defer bufferPool.Put(buf)
	
	// Extract required fields
	var remoteHost, remoteLogname, remoteUser, requestLine, method, path, protocol string
	var statusCode int
	var responseSize int64
	var referer, userAgent string
	
	// Find required fields using faster iteration
	for i := range e.fields {
		field := &e.fields[i]
		switch field.Key {
		case "remote_host", "client_ip", "remote_addr", "ip":
			remoteHost = field.String
		case "remote_logname", "logname":
			remoteLogname = field.String
		case "remote_user", "user":
			remoteUser = field.String
		case "request_line", "request":
			requestLine = field.String
		case "method":
			method = field.String
		case "path", "uri", "url":
			path = field.String
		case "protocol":
			protocol = field.String
		case "status", "status_code":
			statusCode = int(field.Integer)
		case "response_size", "bytes", "size":
			responseSize = field.Integer
		case "referer":
			referer = field.String
		case "user_agent":
			userAgent = field.String
		}
	}
	
	// Try to build the request line if not provided
	if requestLine == "" && method != "" && path != "" {
		if protocol == "" {
			protocol = "HTTP/1.1"
		}
		// Avoid string concatenation, write directly to buffer
		buf.Grow(len(method) + len(path) + len(protocol) + 2)
		buf.WriteString(method)
		buf.WriteByte(' ')
		buf.WriteString(path)
		buf.WriteByte(' ')
		buf.WriteString(protocol)
		requestLine = buf.String()
		buf.Reset()
	}
	
	// Format: %h %l %u %t "%r" %>s %b
	
	// %h - Remote host
	if remoteHost == "" {
		remoteHost = "-"
	}
	buf.WriteString(remoteHost)
	buf.WriteByte(' ')
	
	// %l - Remote logname (from identd, if supplied)
	if remoteLogname == "" {
		remoteLogname = "-"
	}
	buf.WriteString(remoteLogname)
	buf.WriteByte(' ')
	
	// %u - Remote user (from auth)
	if remoteUser == "" {
		remoteUser = "-"
	}
	buf.WriteString(remoteUser)
	buf.WriteByte(' ')
	
	// %t - Time the request was received
	buf.WriteByte('[')
	// Use cached time string
	timeStr := f.getCachedTimeString(e.time)
	buf.WriteString(timeStr)
	buf.WriteByte(']')
	buf.WriteByte(' ')
	
	// %r - Request line
	buf.WriteByte('"')
	if requestLine == "" {
		requestLine = "-"
	}
	buf.WriteString(requestLine)
	buf.WriteByte('"')
	buf.WriteByte(' ')
	
	// %>s - Status code
	if statusCode <= 0 {
		statusCode = 200 // Default to 200 if not provided
	}
	buf.Write(strconv.AppendInt(buf.AvailableBuffer(), int64(statusCode), 10))
	buf.WriteByte(' ')
	
	// %b - Size of the response in bytes
	if responseSize <= 0 {
		buf.WriteByte('-') // Use "-" for zero bytes
	} else {
		buf.Write(strconv.AppendInt(buf.AvailableBuffer(), responseSize, 10))
	}
	
	// Extended format fields (Combined Log Format)
	if f.ExtendedFormat {
		// Referer
		buf.WriteByte(' ')
		buf.WriteByte('"')
		if referer == "" {
			referer = "-"
		}
		buf.WriteString(referer)
		buf.WriteByte('"')
		
		// User-Agent
		buf.WriteByte(' ')
		buf.WriteByte('"')
		if userAgent == "" {
			userAgent = "-"
		}
		buf.WriteString(userAgent)
		buf.WriteByte('"')
	}
	
	// Add a newline if not disabled
	if !f.Options.DisableNewline {
		buf.WriteByte('\n')
	}
	
	// Write the buffer to the writer
	_, err := w.Write(buf.Bytes())
	return err
}

// LogRequest creates log fields from an HTTP request.
func LogRequest(r *http.Request, statusCode int, responseSize int64) []Field {
	// Pre-allocate fields array with capacity for typical case
	fields := make([]Field, 0, 10)
	
	// Remote host
	remoteHost := r.RemoteAddr
	if remoteHost != "" {
		fields = append(fields, Str("remote_host", remoteHost))
	}
	
	// Remote user
	if r.URL != nil && r.URL.User != nil {
		username := r.URL.User.Username()
		if username != "" {
			fields = append(fields, Str("remote_user", username))
		}
	}
	
	// Method
	fields = append(fields, Str("method", r.Method))
	
	// Path
	if r.URL != nil {
		fields = append(fields, Str("path", r.URL.Path))
	}
	
	// Protocol
	fields = append(fields, Str("protocol", r.Proto))
	
	// Status code
	fields = append(fields, Int("status_code", statusCode))
	
	// Response size
	fields = append(fields, Int64("response_size", responseSize))
	
	// Referer
	referer := r.Referer()
	if referer != "" {
		fields = append(fields, Str("referer", referer))
	}
	
	// User-Agent
	userAgent := r.UserAgent()
	if userAgent != "" {
		fields = append(fields, Str("user_agent", userAgent))
	}
	
	// Build request line
	if r.URL != nil {
		requestLine := fmt.Sprintf("%s %s %s", r.Method, r.URL.Path, r.Proto)
		fields = append(fields, Str("request_line", requestLine))
	}
	
	return fields
}

// LogResponseWriter is a wrapper around http.ResponseWriter that captures the
// status code and response size.
type LogResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

// NewLogResponseWriter creates a new LogResponseWriter.
func NewLogResponseWriter(w http.ResponseWriter) *LogResponseWriter {
	return &LogResponseWriter{
		ResponseWriter: w,
		statusCode:     200, // Default to 200 OK
	}
}

// WriteHeader captures the status code.
func (w *LogResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the response size.
func (w *LogResponseWriter) Write(b []byte) (int, error) {
	size, err := w.ResponseWriter.Write(b)
	w.responseSize += int64(size)
	return size, err
}

// Status returns the status code.
func (w *LogResponseWriter) Status() int {
	return w.statusCode
}

// Size returns the response size.
func (w *LogResponseWriter) Size() int64 {
	return w.responseSize
}

// HTTPMiddleware returns a middleware function that logs requests.
func HTTPMiddleware(logger *Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// Wrap the response writer
			lw := NewLogResponseWriter(w)
			
			// Call the next handler
			next.ServeHTTP(lw, r)
			
			// Log the request
			duration := time.Since(start)
			fields := LogRequest(r, lw.Status(), lw.Size())
			fields = append(fields, Duration("duration", duration))
			
			// Log at the appropriate level based on status code
			if lw.Status() >= 500 {
				logger.Error("HTTP Request", fields...)
			} else if lw.Status() >= 400 {
				logger.Warn("HTTP Request", fields...)
			} else {
				logger.Info("HTTP Request", fields...)
			}
		})
	}
}