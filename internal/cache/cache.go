package cache

import (
	"time"
)

// NewMemoryCache creates a new Ristretto-based memory cache
// This function maintains backward compatibility with the old LRU cache interface
func NewMemoryCache(maxSize int, defaultTTL time.Duration) MemoryCache {
	cache, err := NewMemoryCacheLegacy(maxSize, defaultTTL)
	if err != nil {
		// This should rarely happen, but if it does, we need a fallback
		panic("Failed to create Ristretto cache: " + err.Error())
	}
	return cache
}
