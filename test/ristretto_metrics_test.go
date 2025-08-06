package test

import (
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/models"
)

func TestRistrettoMetrics(t *testing.T) {
	// Create a cache with metrics enabled
	config := cache.RistrettoConfig{
		MaxCost:     1000000, // 1MB
		NumCounters: 10000,
		BufferItems: 64,
		Metrics:     true,
	}

	ristrettoCache, err := cache.NewRistrettoCache(config)
	if err != nil {
		t.Fatalf("Failed to create Ristretto cache: %v", err)
	}

	// Test metrics functionality
	now := time.Now().UTC().Truncate(time.Second)

	// Add some items
	for i := 0; i < 5; i++ {
		presence := models.Presence{
			UserID:    "metrics-user-" + string(rune('0'+i)),
			Status:    models.StatusOnline,
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "test-node",
		}
		ristrettoCache.Set(presence.UserID, presence, time.Minute)
	}

	// Get some items (should generate hits)
	for i := 0; i < 3; i++ {
		userID := "metrics-user-" + string(rune('0'+i))
		_, found := ristrettoCache.Get(userID)
		if !found {
			t.Logf("Item %s not found (might be due to admission policy)", userID)
		}
	}

	// Try to get non-existent items (should generate misses)
	for i := 10; i < 15; i++ {
		userID := "metrics-user-" + string(rune('0'+i))
		_, found := ristrettoCache.Get(userID)
		if found {
			t.Errorf("Unexpected hit for non-existent user %s", userID)
		}
	}

	// Get metrics
	metrics := ristrettoCache.Metrics()

	t.Logf("Cache Metrics:")
	t.Logf("  Hits: %d", metrics.Hits)
	t.Logf("  Misses: %d", metrics.Misses)
	t.Logf("  Keys Added: %d", metrics.KeysAdded)
	t.Logf("  Keys Evicted: %d", metrics.KeysEvicted)
	t.Logf("  Cost Added: %d bytes", metrics.CostAdded)
	t.Logf("  Cost Evicted: %d bytes", metrics.CostEvicted)

	// We should have some misses from the non-existent items
	if metrics.Misses == 0 {
		t.Error("Expected some cache misses")
	}

	// We should have added some keys
	if metrics.KeysAdded == 0 {
		t.Error("Expected some keys to be added")
	}
}

func TestRistrettoCacheConfiguration(t *testing.T) {
	// Test that we can create caches with different configurations
	configs := []cache.RistrettoConfig{
		{
			MaxCost:     100000, // Small cache
			NumCounters: 1000,
			BufferItems: 16,
			Metrics:     false,
		},
		{
			MaxCost:     10000000, // Large cache
			NumCounters: 100000,
			BufferItems: 128,
			Metrics:     true,
		},
	}

	for i, config := range configs {
		cache, err := cache.NewRistrettoCache(config)
		if err != nil {
			t.Fatalf("Failed to create cache %d: %v", i, err)
		}

		// Test basic operations
		presence := models.Presence{
			UserID:    "config-test-user",
			Status:    models.StatusAway,
			LastSeen:  time.Now(),
			UpdatedAt: time.Now(),
			NodeID:    "test-node",
		}

		cache.Set("config-test-user", presence, time.Minute)

		if retrieved, found := cache.Get("config-test-user"); found {
			if retrieved.UserID != "config-test-user" {
				t.Errorf("Config %d: Expected UserID 'config-test-user', got '%s'", i, retrieved.UserID)
			}
		} else {
			t.Logf("Config %d: Item not found (admission policy may have rejected it)", i)
		}

		t.Logf("Cache %d configured successfully with MaxCost: %d", i, config.MaxCost)
	}
}
