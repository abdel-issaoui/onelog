package onelog

import (
	"sync"
	"sync/atomic"
)

// fieldPool is a pool of field slices.
type fieldPool struct {
	pools       []*sync.Pool
	sizes       []int
	gets        int64
	puts        int64
	misses      int64
	allocations int64
}

// newFieldPool creates a new field pool with the given capacity.
func newFieldPool(maxCapacity int) *fieldPool {
	// Create pools with increasing sizes, optimized for common use patterns
	sizes := []int{8, 16, 32, 64, 128, 256, 512}
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

	// Fast path for small capacities (most common case)
	if capacity <= p.sizes[0] && p.pools[0] != nil {
		slice := p.pools[0].Get().([]Field)
		return slice[:0]
	}

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

	if cap(slice) == 0 {
		return
	}

	// Fast path for small slices (most common case)
	if cap(slice) <= p.sizes[0] && p.pools[0] != nil {
		// Clear the slice for security
		for i := range slice {
			slice[i] = Field{}
		}
		p.pools[0].Put(slice[:0])
		return
	}

	// Find the appropriate pool
	cap := cap(slice)
	for i, size := range p.sizes {
		if cap <= size {
			if i < len(p.pools) && p.pools[i] != nil {
				// Clear the slice for security
				for i := range slice {
					slice[i] = Field{}
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