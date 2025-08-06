package cache

import (
	"testing"
	"time"

	"gopresence/internal/models"
)

func TestMemoryCache_Basic(t *testing.T) {
	cache := NewMemoryCache(10, time.Minute)
	
	// Test cache is initially empty
	if cache.Size() != 0 {
		t.Errorf("Expected empty cache, got size %d", cache.Size())
	}

	// Test Get on empty cache
	_, found := cache.Get("user1")
	if found {
		t.Error("Expected not found on empty cache")
	}
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache := NewMemoryCache(10, time.Minute)
	now := time.Now()
	
	presence := models.Presence{
		UserID:    "user1",
		Status:    models.StatusOnline,
		Message:   "Working",
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "node1",
	}

	// Set presence
	cache.Set("user1", presence, time.Minute)
	
	// Verify size
	if cache.Size() != 1 {
		t.Errorf("Expected cache size 1, got %d", cache.Size())
	}

	// Get presence
	retrieved, found := cache.Get("user1")
	if !found {
		t.Error("Expected to find cached presence")
	}
	
	if retrieved.UserID != presence.UserID {
		t.Errorf("Expected UserID %s, got %s", presence.UserID, retrieved.UserID)
	}
	if retrieved.Status != presence.Status {
		t.Errorf("Expected Status %s, got %s", presence.Status, retrieved.Status)
	}
}

func TestMemoryCache_TTLExpiration(t *testing.T) {
	cache := NewMemoryCache(10, 50*time.Millisecond)
	now := time.Now()
	
	presence := models.Presence{
		UserID:    "user1",
		Status:    models.StatusOnline,
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "node1",
	}

	// Set with short TTL
	cache.Set("user1", presence, 50*time.Millisecond)
	
	// Should be available immediately
	_, found := cache.Get("user1")
	if !found {
		t.Error("Expected to find presence immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)
	
	// Should be expired now
	_, found = cache.Get("user1")
	if found {
		t.Error("Expected presence to be expired")
	}
}

func TestMemoryCache_Eviction(t *testing.T) {
	// Create a small cache to test eviction behavior
	cache := NewMemoryCache(2, time.Minute) // Small cache for testing
	now := time.Now()
	
	// Fill cache with multiple items to trigger eviction
	// Ristretto uses TinyLFU admission policy, so behavior differs from simple LRU
	for i := 1; i <= 10; i++ {
		presence := models.Presence{
			UserID:    "user" + string(rune('0'+i)),
			Status:    models.StatusOnline,
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "node1",
		}
		cache.Set(presence.UserID, presence, time.Minute)
	}

	// Due to Ristretto's admission policy and cost-based eviction,
	// we can't predict exactly which items will be evicted.
	// Instead, test that the cache is managing its size appropriately
	
	// Count how many items are actually in cache
	foundCount := 0
	for i := 1; i <= 10; i++ {
		if _, found := cache.Get("user" + string(rune('0'+i))); found {
			foundCount++
		}
	}
	
	// The cache should have admitted some items but not all
	// (exact behavior depends on Ristretto's admission policy)
	if foundCount == 0 {
		t.Error("Expected at least some items to be in cache")
	}
	if foundCount == 10 {
		t.Error("Expected cache to evict some items due to size constraints")
	}
	
	t.Logf("Cache admitted %d out of 10 items", foundCount)
}

func TestMemoryCache_Delete(t *testing.T) {
	cache := NewMemoryCache(10, time.Minute)
	now := time.Now()
	
	presence := models.Presence{
		UserID:    "user1",
		Status:    models.StatusOnline,
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "node1",
	}

	cache.Set("user1", presence, time.Minute)
	
	// Verify it's there
	_, found := cache.Get("user1")
	if !found {
		t.Error("Expected to find presence before deletion")
	}

	// Delete it
	cache.Delete("user1")
	
	// Verify it's gone
	_, found = cache.Get("user1")
	if found {
		t.Error("Expected presence to be deleted")
	}
	
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after deletion, got %d", cache.Size())
	}
}

func TestMemoryCache_GetMultiple(t *testing.T) {
	cache := NewMemoryCache(10, time.Minute)
	now := time.Now()
	
	// Add multiple users
	users := []string{"user1", "user2", "user3"}
	for _, userID := range users {
		presence := models.Presence{
			UserID:    userID,
			Status:    models.StatusOnline,
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "node1",
		}
		cache.Set(userID, presence, time.Minute)
	}

	// Get multiple
	result := cache.GetMultiple([]string{"user1", "user2", "user4"}) // user4 doesn't exist
	
	if len(result) != 2 {
		t.Errorf("Expected 2 results, got %d", len(result))
	}
	
	if _, exists := result["user1"]; !exists {
		t.Error("Expected user1 in results")
	}
	if _, exists := result["user2"]; !exists {
		t.Error("Expected user2 in results")
	}
	if _, exists := result["user4"]; exists {
		t.Error("Did not expect user4 in results")
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache := NewMemoryCache(10, time.Minute)
	now := time.Now()
	
	// Add some items
	for i := 1; i <= 3; i++ {
		presence := models.Presence{
			UserID:    "user" + string(rune('0'+i)),
			Status:    models.StatusOnline,
			LastSeen:  now,
			UpdatedAt: now,
			NodeID:    "node1",
		}
		cache.Set(presence.UserID, presence, time.Minute)
	}

	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Clear cache
	cache.Clear()
	
	if cache.Size() != 0 {
		t.Errorf("Expected cache size 0 after clear, got %d", cache.Size())
	}

	// Verify items are gone
	_, found := cache.Get("user1")
	if found {
		t.Error("Expected cache to be empty after clear")
	}
}

func TestMemoryCache_ConcurrentAccess(t *testing.T) {
	cache := NewMemoryCache(100, time.Minute)
	now := time.Now()
	
	// Test concurrent writes and reads
	done := make(chan bool, 20)
	
	// 10 concurrent writers
	for i := 0; i < 10; i++ {
		go func(id int) {
			presence := models.Presence{
				UserID:    "user" + string(rune('0'+id)),
				Status:    models.StatusOnline,
				LastSeen:  now,
				UpdatedAt: now,
				NodeID:    "node1",
			}
			cache.Set(presence.UserID, presence, time.Minute)
			done <- true
		}(i)
	}
	
	// 10 concurrent readers
	for i := 0; i < 10; i++ {
		go func(id int) {
			cache.Get("user" + string(rune('0'+id)))
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}
	
	// Cache should have up to 10 items (depending on timing)
	size := cache.Size()
	if size < 0 || size > 10 {
		t.Errorf("Expected cache size between 0-10, got %d", size)
	}
}