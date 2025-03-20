package onelog

import (
	"hash"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// Sampler is an interface for log samplers.
type Sampler interface {
	// Sample returns true if the log entry should be sampled.
	Sample(e *Entry) bool
}

// RateSampler samples logs at a fixed rate.
type RateSampler struct {
	// N is the sample rate (1 in N).
	N int
	// Counter is the current counter value.
	counter int64
}

// NewRateSampler creates a new RateSampler with the given rate.
func NewRateSampler(n int) *RateSampler {
	if n <= 0 {
		n = 1
	}
	return &RateSampler{
		N: n,
	}
}

// Sample implements the Sampler interface.
func (s *RateSampler) Sample(_ *Entry) bool {
	// Use faster remainder check for powers of 2
	if (s.N & (s.N - 1)) == 0 {
		// N is a power of 2, use bitwise AND
		mask := int64(s.N - 1)
		return (atomic.AddInt64(&s.counter, 1) & mask) == 0
	}
	
	// For non-power-of-2 values, use modulo
	return atomic.AddInt64(&s.counter, 1)%int64(s.N) == 0
}

// KeySampler samples logs based on a key field.
type KeySampler struct {
	// N is the sample rate (1 in N).
	N int
	// Key is the field key to use for sampling.
	Key string
	// hashPool contains pre-allocated hash functions
	hashPool sync.Pool
}

// NewKeySampler creates a new KeySampler with the given rate and key.
func NewKeySampler(n int, key string) *KeySampler {
	if n <= 0 {
		n = 1
	}
	
	return &KeySampler{
		N:   n,
		Key: key,
		hashPool: sync.Pool{
			New: func() interface{} {
				return fnv.New32a()
			},
		},
	}
}

// Sample implements the Sampler interface.
func (s *KeySampler) Sample(e *Entry) bool {
	// Find the key field.
	for i := range e.fields {
		field := &e.fields[i]
		if field.Key == s.Key {
			// Get a hash function from the pool
			h := s.hashPool.Get().(hash.Hash32)
			h.Reset()
			
			// Hash the field value.
			switch field.Type {
			case StringType:
				h.Write([]byte(field.String))
			case IntType, Int64Type:
				var buf [8]byte
				for i := 0; i < 8; i++ {
					buf[i] = byte(field.Integer >> (i * 8))
				}
				h.Write(buf[:])
			case UintType, Uint64Type:
				var buf [8]byte
				v := uint64(field.Integer)
				for i := 0; i < 8; i++ {
					buf[i] = byte(v >> (i * 8))
				}
				h.Write(buf[:])
			case ErrorType:
				h.Write([]byte(field.String))
			default:
				// Can't hash this, so sample it.
				s.hashPool.Put(h)
				return true
			}

			// Check if the hash is a multiple of N.
			result := h.Sum32()%uint32(s.N) == 0
			
			// Return the hash function to the pool
			s.hashPool.Put(h)
			
			return result
		}
	}

	// Key not found, so sample it.
	return true
}

// AdaptiveSampler samples logs based on log volume.
type AdaptiveSampler struct {
	// BaseRate is the base sampling rate.
	BaseRate int
	// MaxRate is the maximum sampling rate.
	MaxRate int
	// WindowSize is the time window for volume measurement.
	WindowSize time.Duration
	// Threshold is the log volume threshold for increasing the sampling rate.
	Threshold int
	// DecayFactor is the decay factor for the sampling rate.
	DecayFactor float64

	// currentRate is the current sampling rate.
	currentRate int
	// counter is the current counter value.
	counter int64
	// volume is the log volume in the current window.
	volume int64
	// lastReset is the last time the volume was reset.
	lastReset time.Time
	// rateLock protects rate changes
	rateLock sync.RWMutex
}

// NewAdaptiveSampler creates a new AdaptiveSampler with the given parameters.
func NewAdaptiveSampler(baseRate, maxRate int, windowSize time.Duration, threshold int, decayFactor float64) *AdaptiveSampler {
	if baseRate <= 0 {
		baseRate = 1
	}
	if maxRate <= 0 {
		maxRate = 1000
	}
	if windowSize <= 0 {
		windowSize = 1 * time.Second
	}
	if threshold <= 0 {
		threshold = 1000
	}
	if decayFactor <= 0 || decayFactor >= 1 {
		decayFactor = 0.9
	}
	return &AdaptiveSampler{
		BaseRate:    baseRate,
		MaxRate:     maxRate,
		WindowSize:  windowSize,
		Threshold:   threshold,
		DecayFactor: decayFactor,
		currentRate: baseRate,
		lastReset:   time.Now(),
	}
}

// Sample implements the Sampler interface.
func (s *AdaptiveSampler) Sample(_ *Entry) bool {
	// Increment the volume.
	atomic.AddInt64(&s.volume, 1)

	// Check if we need to reset the volume.
	now := time.Now()
	if now.Sub(s.lastReset) > s.WindowSize {
		// Need to make rate adjustments
		s.adjustSamplingRate(now)
	}

	// Use a read lock for checking the current rate (faster for concurrent access)
	s.rateLock.RLock()
	currentRate := s.currentRate
	s.rateLock.RUnlock()
	
	// Check if currentRate is a power of 2 for faster sampling decision
	if (currentRate & (currentRate - 1)) == 0 {
		// Power of 2 optimization
		mask := int64(currentRate - 1)
		return (atomic.AddInt64(&s.counter, 1) & mask) == 0
	}
	
	// For non-power-of-2 values, use modulo
	return atomic.AddInt64(&s.counter, 1)%int64(currentRate) == 0
}

// adjustSamplingRate adjusts the sampling rate based on current volume
func (s *AdaptiveSampler) adjustSamplingRate(now time.Time) {
	// Use a write lock for rate adjustments
	s.rateLock.Lock()
	defer s.rateLock.Unlock()
	
	// Get the volume and reset
	volume := atomic.SwapInt64(&s.volume, 0)
	s.lastReset = now

	if volume > int64(s.Threshold) {
		// Increase the sampling rate - double it but cap at MaxRate
		s.currentRate = min(s.currentRate*2, s.MaxRate)
	} else {
		// Decrease the sampling rate gradually
		newRate := int(float64(s.currentRate) * s.DecayFactor)
		s.currentRate = max(newRate, s.BaseRate)
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SpikeSampler samples logs normally but reduces sampling during traffic spikes.
type SpikeSampler struct {
	// NormalRate is the normal sampling rate.
	NormalRate int
	// SpikeRate is the sampling rate during spikes.
	SpikeRate int
	// WindowSize is the time window for spike detection.
	WindowSize time.Duration
	// Threshold is the threshold for spike detection.
	Threshold int

	// counter is the current counter value.
	counter int64
	// volume is the log volume in the current window.
	volume int64
	// lastReset is the last time the volume was reset.
	lastReset time.Time
	// inSpike indicates whether we're currently in a spike.
	inSpike bool
	// lock protects inSpike
	lock sync.RWMutex
}

// NewSpikeSampler creates a new SpikeSampler with the given parameters.
func NewSpikeSampler(normalRate, spikeRate int, windowSize time.Duration, threshold int) *SpikeSampler {
	if normalRate <= 0 {
		normalRate = 1
	}
	if spikeRate <= 0 {
		spikeRate = 100
	}
	if windowSize <= 0 {
		windowSize = 1 * time.Second
	}
	if threshold <= 0 {
		threshold = 1000
	}
	return &SpikeSampler{
		NormalRate: normalRate,
		SpikeRate:  spikeRate,
		WindowSize: windowSize,
		Threshold:  threshold,
		lastReset:  time.Now(),
	}
}

// Sample implements the Sampler interface.
func (s *SpikeSampler) Sample(_ *Entry) bool {
	// Increment the volume.
	atomic.AddInt64(&s.volume, 1)

	// Check if we need to reset the volume.
	now := time.Now()
	if now.Sub(s.lastReset) > s.WindowSize {
		// Check for spikes.
		s.detectSpike(now)
	}

	// Use the appropriate sampling rate.
	rate := s.NormalRate
	
	// Check spike status with read lock (faster for concurrent access)
	s.lock.RLock()
	if s.inSpike {
		rate = s.SpikeRate
	}
	s.lock.RUnlock()

	// Check if rate is a power of 2 for faster sampling
	if (rate & (rate - 1)) == 0 {
		// Power of 2 optimization
		mask := int64(rate - 1)
		return (atomic.AddInt64(&s.counter, 1) & mask) == 0
	}
	
	// For non-power-of-2 rates, use modulo
	return atomic.AddInt64(&s.counter, 1)%int64(rate) == 0
}

// detectSpike checks for traffic spikes and updates state
func (s *SpikeSampler) detectSpike(now time.Time) {
	// Use a write lock when updating spike status
	s.lock.Lock()
	defer s.lock.Unlock()
	
	// Get the volume and reset
	volume := atomic.SwapInt64(&s.volume, 0)
	s.lastReset = now

	// Update spike status
	s.inSpike = volume > int64(s.Threshold)
}

// MultiSampler combines multiple samplers.
type MultiSampler struct {
	// Samplers is the list of samplers.
	Samplers []Sampler
	// Mode is the sampling mode.
	Mode MultiSamplerMode
}

// MultiSamplerMode is the mode for the MultiSampler.
type MultiSamplerMode int

const (
	// AndMode samples only if all samplers sample.
	AndMode MultiSamplerMode = iota
	// OrMode samples if any sampler samples.
	OrMode
)

// NewMultiSampler creates a new MultiSampler with the given samplers and mode.
func NewMultiSampler(mode MultiSamplerMode, samplers ...Sampler) *MultiSampler {
	return &MultiSampler{
		Samplers: samplers,
		Mode:     mode,
	}
}

// Sample implements the Sampler interface.
func (s *MultiSampler) Sample(e *Entry) bool {
	if len(s.Samplers) == 0 {
		return true
	}

	if s.Mode == AndMode {
		// Sample only if all samplers sample.
		for _, sampler := range s.Samplers {
			if !sampler.Sample(e) {
				return false
			}
		}
		return true
	}

	// Sample if any sampler samples.
	for _, sampler := range s.Samplers {
		if sampler.Sample(e) {
			return true
		}
	}
	return false
}