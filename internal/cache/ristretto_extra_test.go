package cache

import (
	"testing"
)

func TestCache_CorruptedEntryAndSizeNoMetrics(t *testing.T) {
	c, _ := NewRistrettoCache(RistrettoConfig{MaxCost: 10000, NumCounters: 1000, BufferItems: 64, Metrics: true})
	// Insert a non-Presence via underlying type info by casting
	rc := c.(*ristrettoCache)
	rc.cache.Set("bad", 12345, 1)
	rc.cache.Wait()
	if _, ok := c.Get("bad"); ok {
		t.Fatalf("expected corrupted entry to be treated as miss")
	}

	// Size without metrics
	c2, _ := NewRistrettoCache(RistrettoConfig{MaxCost: 10000, NumCounters: 1000, BufferItems: 64, Metrics: false})
	if s := c2.Size(); s != 0 {
		t.Fatalf("expected size 0 without metrics, got %d", s)
	}
}
