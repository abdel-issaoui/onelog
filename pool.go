package onelog

import (
	"sync"
	"sync/atomic"
)

// fieldPool is a pool of field slices.
type fieldPool struct {
	pools     []*sync.Pool
	sizes     []int
	gets      int64
	puts      int64
	misses    int64
	allocations int64
}

// newFieldPool creates a new field pool with the given capacity.
func newFieldPool(maxCapacity int) *fieldPool {
	// Create pools with increasing sizes
	sizes := []int{8, 16, 32, 64, 128, 256, 512, 1024}
	pools := make([]*sync.Pool, len(sizes))

	for i, size := range sizes {
		if size > maxCapacity {
			break
		}

		size := size // Capture for closure
		pools[i] = &sync.Pool{
			New: func() interface{} {
				return make([]Field, 0, size)
			},
		}
	}

	return &fieldPool{
		pools: pools,
		sizes: sizes,
	}
}

// Get gets a field slice with the given capacity.
func (p *fieldPool) Get(capacity int) []Field {
	atomic.AddInt64(&p.gets, 1)

	// Find the appropriate pool
	for i, size := range p.sizes {
		if capacity <= size {
			if i < len(p.pools) && p.pools[i] != nil {
				slice := p.pools[i].Get().([]Field)
				return slice[:0] // Return with length 0
			}
		}
	}

	// No suitable pool found, allocate a new slice
	atomic.AddInt64(&p.misses, 1)
	atomic.AddInt64(&p.allocations, 1)
	return make([]Field, 0, capacity)
}

// Put returns a field slice to the pool.
func (p *fieldPool) Put(slice []Field) {
	atomic.AddInt64(&p.puts, 1)

	// Find the appropriate pool
	cap := cap(slice)
	for i, size := range p.sizes {
		if cap <= size {
			if i < len(p.pools) && p.pools[i] != nil {
				// Clear the slice for security
				for j := range slice {
					slice[j] = Field{}
				}
				p.pools[i].Put(slice[:0]) // Return with length 0
				return
			}
		}
	}
}

// GetMetrics returns the pool metrics.
func (p *fieldPool) GetMetrics() map[string]int64 {
	return map[string]int64{
		"gets":        atomic.LoadInt64(&p.gets),
		"puts":        atomic.LoadInt64(&p.puts),
		"misses":      atomic.LoadInt64(&p.misses),
		"allocations": atomic.LoadInt64(&p.allocations),
	}
}

// tieredBufferPool is a pool of byte buffers with different sizes.
type tieredBufferPool struct {
	pools     []*sync.Pool
	sizes     []int
	gets      int64
	puts      int64
	misses    int64
	allocations int64
}

// newTieredBufferPool creates a new tiered buffer pool.
func newTieredBufferPool() *tieredBufferPool {
	// Create pools with increasing sizes
	sizes := []int{256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	pools := make([]*sync.Pool, len(sizes))

	for i, size := range sizes {
		size := size // Capture for closure
		pools[i] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, size)
			},
		}
	}

	return &tieredBufferPool{
		pools: pools,
		sizes: sizes,
	}
}

// Get gets a byte buffer with the given capacity.
func (p *tieredBufferPool) Get(capacity int) []byte {
	atomic.AddInt64(&p.gets, 1)

	// Find the appropriate pool
	for i, size := range p.sizes {
		if capacity <= size {
			if i < len(p.pools) && p.pools[i] != nil {
				buf := p.pools[i].Get().([]byte)
				return buf[:0] // Return with length 0
			}
		}
	}

	// No suitable pool found, allocate a new buffer
	atomic.AddInt64(&p.misses, 1)
	atomic.AddInt64(&p.allocations, 1)
	return make([]byte, 0, capacity)
}

// Put returns a byte buffer to the pool.
func (p *tieredBufferPool) Put(buf []byte) {
	atomic.AddInt64(&p.puts, 1)

	// Find the appropriate pool
	cap := cap(buf)
	for i, size := range p.sizes {
		if cap <= size {
			if i < len(p.pools) && p.pools[i] != nil {
				// Clear the buffer for security
				for j := range buf {
					buf[j] = 0
				}
				p.pools[i].Put(buf[:0]) // Return with length 0
				return
			}
		}
	}
}

// GetMetrics returns the pool metrics.
func (p *tieredBufferPool) GetMetrics() map[string]int64 {
	return map[string]int64{
		"gets":        atomic.LoadInt64(&p.gets),
		"puts":        atomic.LoadInt64(&p.puts),
		"misses":      atomic.LoadInt64(&p.misses),
		"allocations": atomic.LoadInt64(&p.allocations),
	}
}

// Global buffer pool
var globalBufferPool = newTieredBufferPool()

// GetBuffer gets a byte buffer from the global pool.
func GetBuffer(capacity int) []byte {
	return globalBufferPool.Get(capacity)
}

// PutBuffer returns a byte buffer to the global pool.
func PutBuffer(buf []byte) {
	globalBufferPool.Put(buf)
}

// GetBufferPoolMetrics returns the global buffer pool metrics.
func GetBufferPoolMetrics() map[string]int64 {
	return globalBufferPool.GetMetrics()
}