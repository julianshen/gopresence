package cache

import (
	"encoding/json"
	"time"

	"github.com/dgraph-io/ristretto"

	"gopresence/internal/models"
)

// MemoryCache defines the interface for in-memory caching
type MemoryCache interface {
	Get(userID string) (models.Presence, bool)
	Set(userID string, presence models.Presence, ttl time.Duration) // Keep TTL for backward compatibility
	Delete(userID string)
	GetMultiple(userIDs []string) map[string]models.Presence
	Size() int
	Clear()
	Metrics() CacheMetrics
}

// CacheMetrics provides cache performance metrics
type CacheMetrics struct {
	Hits        uint64
	Misses      uint64
	KeysAdded   uint64
	KeysEvicted uint64
	CostAdded   uint64
	CostEvicted uint64
}

// RistrettoConfig holds Ristretto cache configuration
type RistrettoConfig struct {
	MaxCost     int64 // Maximum cost of cache (bytes)
	NumCounters int64 // Number of counters for TinyLFU admission policy
	BufferItems int64 // Buffer size for async operations
	Metrics     bool  // Enable metrics collection
}

// ristrettoCache implements MemoryCache using Ristretto
type ristrettoCache struct {
	cache  *ristretto.Cache
	config RistrettoConfig
}

// NewRistrettoCache creates a new Ristretto-based memory cache
func NewRistrettoCache(config RistrettoConfig) (MemoryCache, error) {
	ristrettoConfig := &ristretto.Config{
		MaxCost:     config.MaxCost,
		NumCounters: config.NumCounters,
		BufferItems: config.BufferItems,
		Metrics:     config.Metrics,
	}

	cache, err := ristretto.NewCache(ristrettoConfig)
	if err != nil {
		return nil, err
	}

	return &ristrettoCache{
		cache:  cache,
		config: config,
	}, nil
}

// Get retrieves a presence from the cache
func (c *ristrettoCache) Get(userID string) (models.Presence, bool) {
	value, found := c.cache.Get(userID)
	if !found {
		return models.Presence{}, false
	}

	presence, ok := value.(models.Presence)
	if !ok {
		// Handle corrupted cache entry
		c.cache.Del(userID)
		return models.Presence{}, false
	}

	// Check if presence is expired (application-level TTL)
	if presence.TTL > 0 && time.Since(presence.UpdatedAt) > presence.TTL {
		c.cache.Del(userID)
		return models.Presence{}, false
	}

	return presence, true
}

// Set stores a presence in the cache (TTL parameter kept for backward compatibility)
func (c *ristrettoCache) Set(userID string, presence models.Presence, ttl time.Duration) {
	// Estimate cost based on JSON serialization size
	cost := c.estimateCost(presence)

	// Store TTL in the presence object for application-level expiry
	if ttl > 0 {
		presence.TTL = ttl
	}

	// Ristretto handles admission and eviction automatically
	// Note: Ristretto operations are asynchronous, but Set usually succeeds immediately
	c.cache.Set(userID, presence, cost)

	// Wait briefly for the set operation to complete in Ristretto's buffers
	// This is needed for tests that expect immediate consistency
	c.cache.Wait()
}

// Delete removes a presence from the cache
func (c *ristrettoCache) Delete(userID string) {
	c.cache.Del(userID)
	// Wait for delete operation to complete
	c.cache.Wait()
}

// GetMultiple retrieves multiple presences from the cache
func (c *ristrettoCache) GetMultiple(userIDs []string) map[string]models.Presence {
	result := make(map[string]models.Presence)

	for _, userID := range userIDs {
		if presence, found := c.Get(userID); found {
			result[userID] = presence
		}
	}

	return result
}

// Size returns the approximate number of items in the cache
// Note: Ristretto is eventually consistent, so this might not be exact
func (c *ristrettoCache) Size() int {
	// Ristretto doesn't provide a direct size method, but we can use metrics
	if c.config.Metrics {
		metrics := c.cache.Metrics
		return int(metrics.KeysAdded() - metrics.KeysEvicted())
	}
	// If metrics are disabled, we can't determine the exact size
	// Return 0 as a safe fallback
	return 0
}

// Clear removes all items from the cache
func (c *ristrettoCache) Clear() {
	c.cache.Clear()
}

// Metrics returns cache performance metrics
func (c *ristrettoCache) Metrics() CacheMetrics {
	if !c.config.Metrics {
		return CacheMetrics{}
	}

	metrics := c.cache.Metrics
	return CacheMetrics{
		Hits:        metrics.Hits(),
		Misses:      metrics.Misses(),
		KeysAdded:   metrics.KeysAdded(),
		KeysEvicted: metrics.KeysEvicted(),
		CostAdded:   metrics.CostAdded(),
		CostEvicted: metrics.CostEvicted(),
	}
}

// estimateCost estimates the memory cost of a presence object
func (c *ristrettoCache) estimateCost(presence models.Presence) int64 {
	// Quick estimation: JSON serialization size + overhead
	data, err := json.Marshal(presence)
	if err != nil {
		// Fallback to a reasonable estimate
		return 200
	}

	// Add some overhead for Go object structure
	return int64(len(data) + 100)
}

// Legacy constructor for backward compatibility
func NewMemoryCacheLegacy(maxSize int, defaultTTL time.Duration) (MemoryCache, error) {
	// Convert old parameters to Ristretto config
	// Estimate max cost based on max size and average presence size
	avgPresenceSize := int64(200) // bytes
	maxCost := int64(maxSize) * avgPresenceSize

	// Set reasonable defaults for Ristretto
	config := RistrettoConfig{
		MaxCost:     maxCost,
		NumCounters: int64(maxSize * 10), // 10x for good admission policy
		BufferItems: 64,                  // Default buffer size
		Metrics:     true,                // Enable metrics
	}

	return NewRistrettoCache(config)
}
