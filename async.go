package onelog

import (
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// BackpressureMode defines how the asyncBuffer handles backpressure.
type BackpressureMode int

const (
	// DropMode drops log entries when the buffer is full.
	DropMode BackpressureMode = iota
	// BlockMode blocks until space is available in the buffer.
	BlockMode
)

// asyncBuffer is a lock-free ring buffer for asynchronous logging.
type asyncBuffer struct {
	// The buffer size (power of 2).
	size int
	// The buffer mask.
	mask int64
	// The read index.
	readIndex int64
	// The write index.
	writeIndex int64
	// The buffer.
	buffer [][]byte
	// The writer.
	writer io.Writer
	// The stop channel.
	stopCh chan struct{}
	// The wait group.
	wg sync.WaitGroup
	// The backpressure mode.
	backpressureMode BackpressureMode
	// The resize lock.
	resizeLock sync.RWMutex
	// The buffer utilization (0-100).
	utilization int64
	// The drop count.
	dropCount int64
	// Whether dynamic resizing is enabled.
	dynamicResize bool
	// The resize threshold.
	resizeThreshold int
	// The flush interval.
	flushInterval time.Duration
}

// newAsyncBuffer creates a new asyncBuffer.
func newAsyncBuffer(size int, writer io.Writer) *asyncBuffer {
	// Ensure the size is a power of 2.
	if size <= 0 || (size&(size-1)) != 0 {
		size = roundUpPowerOfTwo(size)
	}

	b := &asyncBuffer{
		size:             size,
		mask:             int64(size - 1),
		buffer:           make([][]byte, size),
		writer:           writer,
		stopCh:           make(chan struct{}),
		backpressureMode: DropMode,
		dynamicResize:    true,
		resizeThreshold:  75, // 75% utilization
		flushInterval:    100 * time.Millisecond,
	}

	// Start the worker goroutine.
	b.wg.Add(1)
	go b.worker()

	return b
}

// roundUpPowerOfTwo rounds up to the next power of 2.
func roundUpPowerOfTwo(n int) int {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// write writes a log entry to the buffer.
func (b *asyncBuffer) write(p []byte) error {
	b.resizeLock.RLock()
	defer b.resizeLock.RUnlock()

	// Copy the log entry.
	entry := make([]byte, len(p))
	copy(entry, p)

	// Get the current write index.
	writeIndex := atomic.LoadInt64(&b.writeIndex)

	// Loop until we can write or drop.
	for {
		// Calculate the next write index.
		nextWriteIndex := writeIndex + 1
		// Get the read index.
		readIndex := atomic.LoadInt64(&b.readIndex)
		// Calculate the buffer usage.
		usage := nextWriteIndex - readIndex

		// Check if the buffer is full.
		if usage > int64(b.size) {
			// In drop mode, drop the log entry.
			if b.backpressureMode == DropMode {
				atomic.AddInt64(&b.dropCount, 1)
				return ErrBufferFull
			}

			// In block mode, wait a bit and try again.
			time.Sleep(1 * time.Millisecond)
			writeIndex = atomic.LoadInt64(&b.writeIndex)
			continue
		}

		// Try to atomically update the write index.
		if atomic.CompareAndSwapInt64(&b.writeIndex, writeIndex, nextWriteIndex) {
			// We got the slot, write the log entry.
			b.buffer[writeIndex&b.mask] = entry

			// Update utilization metric.
			atomic.StoreInt64(&b.utilization, usage*100/int64(b.size))

			// Maybe resize the buffer.
			if b.dynamicResize && usage*100/int64(b.size) > int64(b.resizeThreshold) {
				go b.maybeResize()
			}

			return nil
		}

		// Someone else got the slot, try again.
		writeIndex = atomic.LoadInt64(&b.writeIndex)
	}
}

// maybeResize resizes the buffer if it's too full.
func (b *asyncBuffer) maybeResize() {
	// Check if we need to resize.
	utilization := atomic.LoadInt64(&b.utilization)
	if utilization <= int64(b.resizeThreshold) {
		return
	}

	// Acquire the resize lock.
	b.resizeLock.Lock()
	defer b.resizeLock.Unlock()

	// Check again now that we have the lock.
	utilization = atomic.LoadInt64(&b.utilization)
	if utilization <= int64(b.resizeThreshold) {
		return
	}

	// Calculate the new size.
	newSize := b.size * 2
	if newSize > 1024*1024 {
		// Max buffer size is 1M entries.
		return
	}

	// Create a new buffer.
	newBuffer := make([][]byte, newSize)
	newMask := int64(newSize - 1)

	// Copy entries from the old buffer to the new buffer.
	readIndex := atomic.LoadInt64(&b.readIndex)
	writeIndex := atomic.LoadInt64(&b.writeIndex)
	for i := readIndex; i < writeIndex; i++ {
		newBuffer[i&newMask] = b.buffer[i&b.mask]
	}

	// Update the buffer, size, and mask.
	b.buffer = newBuffer
	b.size = newSize
	b.mask = newMask
}

// close closes the buffer and waits for all writes to complete.
func (b *asyncBuffer) close() error {
	// Signal the worker to stop.
	close(b.stopCh)
	// Wait for the worker to finish.
	b.wg.Wait()
	return nil
}

// worker processes log entries from the buffer.
func (b *asyncBuffer) worker() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.stopCh:
			// Drain the buffer before exiting.
			b.flush()
			return
		case <-ticker.C:
			// Flush the buffer periodically.
			b.flush()
		}
	}
}

// flush flushes the buffer.
func (b *asyncBuffer) flush() {
	b.resizeLock.RLock()
	defer b.resizeLock.RUnlock()

	// Get the read index.
	readIndex := atomic.LoadInt64(&b.readIndex)
	// Get the write index.
	writeIndex := atomic.LoadInt64(&b.writeIndex)

	// Process all entries from readIndex to writeIndex.
	for i := readIndex; i < writeIndex; i++ {
		// Get the entry.
		entry := b.buffer[i&b.mask]
		if entry == nil {
			continue
		}

		// Write the entry.
		_, err := b.writer.Write(entry)
		if err != nil {
			// TODO: Handle error?
		}

		// Clear the entry.
		b.buffer[i&b.mask] = nil

		// Update the read index.
		atomic.StoreInt64(&b.readIndex, i+1)
	}
}

// SetBackpressureMode sets the backpressure mode.
func (b *asyncBuffer) SetBackpressureMode(mode BackpressureMode) {
	b.backpressureMode = mode
}

// SetDynamicResize sets whether dynamic resizing is enabled.
func (b *asyncBuffer) SetDynamicResize(enabled bool) {
	b.dynamicResize = enabled
}

// SetResizeThreshold sets the resize threshold.
func (b *asyncBuffer) SetResizeThreshold(threshold int) {
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	b.resizeThreshold = threshold
}

// SetFlushInterval sets the flush interval.
func (b *asyncBuffer) SetFlushInterval(interval time.Duration) {
	b.flushInterval = interval
}

// GetUtilization returns the buffer utilization (0-100).
func (b *asyncBuffer) GetUtilization() int {
	return int(atomic.LoadInt64(&b.utilization))
}

// GetDropCount returns the number of dropped log entries.
func (b *asyncBuffer) GetDropCount() int64 {
	return atomic.LoadInt64(&b.dropCount)
}