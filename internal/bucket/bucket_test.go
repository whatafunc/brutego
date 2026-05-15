package bucket_test

import (
	"sync"
	"testing"

	"github.com/whatafunc/brutego/internal/bucket"
)

// ---------------------------------------------------------------------------
// Bucket tests
// ---------------------------------------------------------------------------

func TestBucket_AllowsUpToLimit(t *testing.T) {
	t.Parallel()

	const limit = 5
	b := bucket.New(limit)

	for i := 0; i < limit; i++ {
		if !b.Allow() {
			t.Fatalf("expected Allow()=true on call %d/%d", i+1, limit)
		}
	}
}

func TestBucket_DeniesOverLimit(t *testing.T) {
	t.Parallel()

	const limit = 3
	b := bucket.New(limit)

	for i := 0; i < limit; i++ {
		b.Allow()
	}

	if b.Allow() {
		t.Fatal("expected Allow()=false after limit exceeded")
	}
}

func TestBucket_Reset_ClearsCounter(t *testing.T) {
	t.Parallel()

	const limit = 2
	b := bucket.New(limit)

	b.Allow()
	b.Allow()

	if b.Allow() {
		t.Fatal("expected denial before reset")
	}

	b.Reset()

	if !b.Allow() {
		t.Fatal("expected Allow()=true after reset")
	}
}

func TestBucket_Count_ReflectsAllowCalls(t *testing.T) {
	t.Parallel()

	b := bucket.New(10)
	b.Allow()
	b.Allow()

	if got := b.Count(); got != 2 {
		t.Fatalf("expected count=2, got %d", got)
	}
}

func TestBucket_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const (
		limit      = 100
		goroutines = 50
		callsEach  = 10
	)

	b := bucket.New(limit)

	var wg sync.WaitGroup
	allowed := make(chan bool, goroutines*callsEach)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsEach; j++ {
				allowed <- b.Allow()
			}
		}()
	}

	wg.Wait()
	close(allowed)

	count := 0
	for a := range allowed {
		if a {
			count++
		}
	}

	if count != limit {
		t.Fatalf("expected exactly %d allowed requests, got %d", limit, count)
	}
}

// ---------------------------------------------------------------------------
// Store tests
// ---------------------------------------------------------------------------

func TestStore_IndependentBucketsPerKey(t *testing.T) {
	t.Parallel()

	const limit = 2
	s := bucket.NewStore(limit)

	s.Allow("alice")
	s.Allow("alice")

	// alice is now at limit — bob should still be allowed
	if !s.Allow("bob") {
		t.Fatal("bob should not be affected by alice's bucket")
	}

	if s.Allow("alice") {
		t.Fatal("alice should be denied after reaching limit")
	}
}

func TestStore_Reset_ClearsSpecificKey(t *testing.T) {
	t.Parallel()

	const limit = 1
	s := bucket.NewStore(limit)

	s.Allow("alice")

	if s.Allow("alice") {
		t.Fatal("alice should be denied before reset")
	}

	s.Reset("alice")

	if !s.Allow("alice") {
		t.Fatal("alice should be allowed after reset")
	}
}

func TestStore_Reset_UnknownKey_NoOp(t *testing.T) {
	t.Parallel()

	s := bucket.NewStore(10)
	// should not panic
	s.Reset("nonexistent")
}

func TestStore_ConcurrentDifferentKeys(t *testing.T) {
	t.Parallel()

	const (
		limit = 5
		keys  = 20
	)

	s := bucket.NewStore(limit)

	var wg sync.WaitGroup
	for i := 0; i < keys; i++ {
		key := string(rune('a' + i))
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			allowed := 0
			for j := 0; j < limit+5; j++ {
				if s.Allow(k) {
					allowed++
				}
			}
			if allowed != limit {
				t.Errorf("key %q: expected %d allowed, got %d", k, limit, allowed)
			}
		}(key)
	}

	wg.Wait()
}
