package onelog

import (
	"hash/fnv"
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
	// Increment the counter and check if it's a multiple of N.
	return atomic.AddInt64(&s.counter, 1)%int64(s.N) == 0
}

// KeySampler samples logs based on a key field.
type KeySampler struct {
	// N is the sample rate (1 in N).
	N int
	// Key is the field key to use for sampling.
	Key string
}

// NewKeySampler creates a new KeySampler with the given rate and key.
func NewKeySampler(n int, key string) *KeySampler {
	if n <= 0 {
		n = 1
	}
	return &KeySampler{
		N:   n,
		Key: key,
	}
}

// Sample implements the Sampler interface.
func (s *KeySampler) Sample(e *Entry) bool {
	// Find the key field.
	for _, field := range e.fields {
		if field.Key == s.Key {
			// Hash the field value.
			h := fnv.New32a()
			switch field.Type {
			case StringType:
				h.Write([]byte(field.String))
			case IntType, Int64Type:
				var buf [8]byte
				writeInt64Bytes(buf[:], field.Integer)
				h.Write(buf[:])
			case UintType, Uint64Type:
				var buf [8]byte
				writeUint64Bytes(buf[:], uint64(field.Integer))
				h.Write(buf[:])
			case ErrorType:
				h.Write([]byte(field.String))
			default:
				// Can't hash this, so sample it.
				return true
			}

			// Check if the hash is a multiple of N.
			return h.Sum32()%uint32(s.N) == 0
		}
	}

	// Key not found, so sample it.
	return true
}

// writeInt64Bytes writes an int64 to a byte slice.
func writeInt64Bytes(b []byte, v int64) {
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (i * 8))
	}
}

// writeUint64Bytes writes a uint64 to a byte slice.
func writeUint64Bytes(b []byte, v uint64) {
	for i := 0; i < 8; i++ {
		b[i] = byte(v >> (i * 8))
	}
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
		// Adjust the sampling rate based on the volume.
		volume := atomic.SwapInt64(&s.volume, 0)
		s.lastReset = now

		if volume > int64(s.Threshold) {
			// Increase the sampling rate.
			s.currentRate = min(s.currentRate*2, s.MaxRate)
		} else {
			// Decrease the sampling rate.
			newRate := int(float64(s.currentRate) * s.DecayFactor)
			s.currentRate = max(newRate, s.BaseRate)
		}
	}

	// Increment the counter and check if it's a multiple of the current rate.
	return atomic.AddInt64(&s.counter, 1)%int64(s.currentRate) == 0
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
		volume := atomic.SwapInt64(&s.volume, 0)
		s.lastReset = now

		s.inSpike = volume > int64(s.Threshold)
	}

	// Use the appropriate sampling rate.
	rate := s.NormalRate
	if s.inSpike {
		rate = s.SpikeRate
	}

	// Increment the counter and check if it's a multiple of the rate.
	return atomic.AddInt64(&s.counter, 1)%int64(rate) == 0
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