package onelog

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogWriter is an interface for log writers.
type LogWriter interface {
	io.Writer
	// Close closes the writer.
	Close() error
}

// ConsoleWriter writes logs to the console.
type ConsoleWriter struct {
	out io.Writer
}

// NewConsoleWriter creates a new ConsoleWriter.
func NewConsoleWriter() *ConsoleWriter {
	return &ConsoleWriter{
		out: os.Stdout,
	}
}

// Write implements io.Writer.
func (w *ConsoleWriter) Write(p []byte) (n int, err error) {
	return w.out.Write(p)
}

// Close implements LogWriter.
func (w *ConsoleWriter) Close() error {
	return nil
}

// SetOutput sets the output writer.
func (w *ConsoleWriter) SetOutput(out io.Writer) {
	w.out = out
}

// FileWriter writes logs to a file.
type FileWriter struct {
	filename  string
	file      *os.File
	mu        sync.Mutex
	maxSize   int64
	maxAge    time.Duration
	maxBackups int
	compress  bool
	size      int64
}

// FileInfo represents information about a log file.
type FileInfo struct {
	name       string
	time       time.Time
	compressed bool
}

// FileWriterOption is a function that configures a FileWriter.
type FileWriterOption func(*FileWriter)

// WithMaxSize sets the maximum size of the log file before it gets rotated.
func WithMaxSize(maxSize int64) FileWriterOption {
	return func(w *FileWriter) {
		w.maxSize = maxSize
	}
}

// WithMaxAge sets the maximum age of a log file before it gets deleted.
func WithMaxAge(maxAge time.Duration) FileWriterOption {
	return func(w *FileWriter) {
		w.maxAge = maxAge
	}
}

// WithMaxBackups sets the maximum number of old log files to retain.
func WithMaxBackups(maxBackups int) FileWriterOption {
	return func(w *FileWriter) {
		w.maxBackups = maxBackups
	}
}

// WithCompress enables compression of rotated log files.
func WithCompress(compress bool) FileWriterOption {
	return func(w *FileWriter) {
		w.compress = compress
	}
}

// NewFileWriter creates a new FileWriter.
func NewFileWriter(filename string, options ...FileWriterOption) (*FileWriter, error) {
	w := &FileWriter{
		filename:   filename,
		maxSize:    100 * 1024 * 1024, // 100 MB
		maxAge:     7 * 24 * time.Hour, // 7 days
		maxBackups: 5,
		compress:   true,
	}
	
	for _, option := range options {
		option(w)
	}
	
	if err := w.openFile(); err != nil {
		return nil, err
	}
	
	return w, nil
}

// openFile opens the log file.
func (w *FileWriter) openFile() error {
	// Create the directory if it doesn't exist
	dir := filepath.Dir(w.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Open the file for appending
	f, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	
	// Get the file size
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	
	w.file = f
	w.size = info.Size()
	
	return nil
}

// Write implements io.Writer.
func (w *FileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.file == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}
	
	// Check if the file needs to be rotated
	if w.maxSize > 0 && w.size+int64(len(p)) > w.maxSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}
	
	n, err = w.file.Write(p)
	w.size += int64(n)
	
	return n, err
}

// Close implements LogWriter.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if w.file == nil {
		return nil
	}
	
	err := w.file.Close()
	w.file = nil
	
	return err
}

// rotate rotates the log file.
func (w *FileWriter) rotate() error {
	// Close the current file
	if err := w.file.Close(); err != nil {
		return err
	}
	
	// Get the current time
	now := time.Now()
	
	// Rotate the file
	rotatedName := fmt.Sprintf("%s.%s", w.filename, now.Format("2006-01-02-15-04-05"))
	if err := os.Rename(w.filename, rotatedName); err != nil {
		return err
	}
	
	// Compress the rotated file if enabled
	if w.compress {
		go func(name string) {
			if err := compressFile(name); err != nil {
				// Handle compression error
			}
		}(rotatedName)
	}
	
	// Open a new file
	if err := w.openFile(); err != nil {
		return err
	}
	
	// Clean up old log files
	go w.cleanup(now)
	
	return nil
}

// cleanup deletes old log files.
func (w *FileWriter) cleanup(now time.Time) {
	pattern := fmt.Sprintf("%s.*", w.filename)
	files, err := filepath.Glob(pattern)
	if err != nil {
		return
	}
	
	var logs []FileInfo
	
	// Collect information about log files
	for _, file := range files {
		// Skip compressed files when collecting for age-based cleanup
		compressed := filepath.Ext(file) == ".gz"
		
		// Get the file modification time
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		
		logs = append(logs, FileInfo{
			name:       file,
			time:       info.ModTime(),
			compressed: compressed,
		})
	}
	
	// Delete old log files based on age
	if w.maxAge > 0 {
		cutoff := now.Add(-w.maxAge)
		for _, log := range logs {
			if log.time.Before(cutoff) {
				os.Remove(log.name)
			}
		}
	}
	
	// Delete old log files based on count
	if w.maxBackups > 0 && len(logs) > w.maxBackups {
		// Sort the logs by time (oldest first)
		sortLogsByTime(logs)
		
		// Delete the oldest logs
		for i := 0; i < len(logs)-w.maxBackups; i++ {
			os.Remove(logs[i].name)
		}
	}
}

// sortLogsByTime sorts logs by time (oldest first).
func sortLogsByTime(logs []FileInfo) {
	for i := 0; i < len(logs); i++ {
		for j := i + 1; j < len(logs); j++ {
			if logs[i].time.After(logs[j].time) {
				logs[i], logs[j] = logs[j], logs[i]
			}
		}
	}
}

// compressFile compresses a file.
func compressFile(name string) error {
	// Open the file for reading
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	
	// Create the compressed file
	compressedName := name + ".gz"
	cf, err := os.OpenFile(compressedName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer cf.Close()
	
	// Create a gzip writer
	gw := gzip.NewWriter(cf)
	defer gw.Close()
	
	// Copy the file to the gzip writer
	if _, err := io.Copy(gw, f); err != nil {
		return err
	}
	
	// Close the gzip writer
	if err := gw.Close(); err != nil {
		return err
	}
	
	// Remove the original file
	return os.Remove(name)
}

// MultiWriter writes logs to multiple writers.
type MultiWriter struct {
	writers []LogWriter
	mu      sync.Mutex
}

// NewMultiWriter creates a new MultiWriter.
func NewMultiWriter(writers ...LogWriter) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write implements io.Writer.
func (w *MultiWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	for _, writer := range w.writers {
		_, err := writer.Write(p)
		if err != nil {
			return 0, err
		}
	}
	
	return len(p), nil
}

// Close implements LogWriter.
func (w *MultiWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	var firstErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	
	return firstErr
}

// AddWriter adds a writer to the MultiWriter.
func (w *MultiWriter) AddWriter(writer LogWriter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.writers = append(w.writers, writer)
}

// RemoveWriter removes a writer from the MultiWriter.
func (w *MultiWriter) RemoveWriter(writer LogWriter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	for i, wr := range w.writers {
		if wr == writer {
			w.writers = append(w.writers[:i], w.writers[i+1:]...)
			break
		}
	}
}