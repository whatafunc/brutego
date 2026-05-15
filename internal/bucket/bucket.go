// Package bucket implements a leaky-bucket rate limiter.
//
// Each Bucket tracks the number of requests in the current minute window.
// When Allow() is called:
//   - if the window has expired, the counter resets to 1 and the call is allowed.
//   - if the counter is below the limit, it increments and the call is allowed.
//   - otherwise the call is denied.
//
// This is the core logic unit covered by unit tests.
package bucket

import (
	"sync"
	"time"
)

// Bucket is a single rate-limit counter for one key (login / password / IP).
// It is safe for concurrent use.
type Bucket struct {
	mu        sync.Mutex
	limit     int       // max requests per window
	count     int       // requests in current window
	windowEnd time.Time // when the current window expires
}

// New creates a Bucket with the given requests-per-minute limit.
func New(limit int) *Bucket {
	return &Bucket{
		limit:     limit,
		windowEnd: time.Now().Add(time.Minute),
	}
}

// Allow reports whether the next request should be allowed.
// It advances the window automatically when it expires.
func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.After(b.windowEnd) {
		// New window — reset counter.
		b.count = 0
		b.windowEnd = now.Add(time.Minute)
	}

	if b.count >= b.limit {
		return false
	}

	b.count++
	return true
}

// Reset clears the counter and starts a fresh window immediately.
func (b *Bucket) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.count = 0
	b.windowEnd = time.Now().Add(time.Minute)
}

// Count returns the current request count in the active window (useful for tests).
func (b *Bucket) Count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.count
}

// -----------------------------------------------------------------------------
// BucketStore — a thread-safe map of key → *Bucket with idle-entry cleanup.
// -----------------------------------------------------------------------------

// Store manages a collection of buckets, one per unique key.
// Inactive buckets (window expired) are pruned on every N-th access to
// prevent unbounded memory growth.
type Store struct {
	mu      sync.Mutex
	buckets map[string]*Bucket
	limit   int // shared limit for all buckets in this store
	pruneAt int // prune when len(buckets) reaches this threshold
}

// NewStore creates a Store where every bucket enforces the given RPM limit.
func NewStore(limit int) *Store {
	return &Store{
		buckets: make(map[string]*Bucket),
		limit:   limit,
		pruneAt: 10_000,
	}
}

// Allow returns true if the key is within its rate limit for this window.
func (s *Store) Allow(key string) bool {
	s.mu.Lock()
	b, ok := s.buckets[key]
	if !ok {
		b = New(s.limit)
		s.buckets[key] = b
		if len(s.buckets) >= s.pruneAt {
			s.pruneExpired()
		}
	}
	s.mu.Unlock()

	return b.Allow()
}

// Reset clears the bucket for the given key.
func (s *Store) Reset(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if b, ok := s.buckets[key]; ok {
		b.Reset()
	}
}

// pruneExpired removes buckets whose window has already elapsed.
// Must be called with s.mu held.
func (s *Store) pruneExpired() {
	now := time.Now()
	for key, b := range s.buckets {
		b.mu.Lock()
		expired := now.After(b.windowEnd)
		b.mu.Unlock()
		if expired {
			delete(s.buckets, key)
		}
	}
}
