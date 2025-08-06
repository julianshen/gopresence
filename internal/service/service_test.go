package service

import (
	"context"
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/models"
	"gopresence/internal/nats"
)

func TestPresenceService_Integration(t *testing.T) {
	// Create test components
	memCache := cache.NewMemoryCache(100, time.Minute)

	store, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: "test-presence-service",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewPresenceService(memCache, store, "test-node")
	ctx := context.Background()

	// Test setting and getting presence
	testPresence := models.Presence{
		UserID:  "user1",
		Status:  models.StatusOnline,
		Message: "Working",
		TTL:     time.Hour,
	}

	// Set presence
	err = service.SetPresence(ctx, "user1", testPresence)
	if err != nil {
		t.Fatalf("Failed to set presence: %v", err)
	}

	// Get presence
	retrieved, err := service.GetPresence(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to get presence: %v", err)
	}

	// Verify presence data
	if retrieved.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", retrieved.UserID)
	}
	if retrieved.Status != models.StatusOnline {
		t.Errorf("Expected Status %s, got %s", models.StatusOnline, retrieved.Status)
	}
	if retrieved.NodeID != "test-node" {
		t.Errorf("Expected NodeID 'test-node', got '%s'", retrieved.NodeID)
	}
}

func TestPresenceService_CacheIntegration(t *testing.T) {
	// Create test components
	memCache := cache.NewMemoryCache(100, time.Minute)

	store, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: "test-presence-cache",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewPresenceService(memCache, store, "test-node")
	ctx := context.Background()

	// Set presence
	testPresence := models.Presence{
		UserID:  "user1",
		Status:  models.StatusBusy,
		Message: "In meeting",
		TTL:     time.Hour,
	}

	err = service.SetPresence(ctx, "user1", testPresence)
	if err != nil {
		t.Fatalf("Failed to set presence: %v", err)
	}

	// First get should come from cache
	retrieved1, err := service.GetPresence(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to get presence: %v", err)
	}

	// Clear cache to test store fallback
	memCache.Clear()

	// Second get should come from store
	retrieved2, err := service.GetPresence(ctx, "user1")
	if err != nil {
		t.Fatalf("Failed to get presence from store: %v", err)
	}

	// Both should be the same
	if retrieved1.Status != retrieved2.Status {
		t.Errorf("Cache and store results differ: %s vs %s", retrieved1.Status, retrieved2.Status)
	}
}

func TestPresenceService_MultiplePresences(t *testing.T) {
	// Create test components
	memCache := cache.NewMemoryCache(100, time.Minute)

	store, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: "test-presence-multi",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewPresenceService(memCache, store, "test-node")
	ctx := context.Background()

	// Set multiple presences
	users := []string{"user1", "user2", "user3"}
	statuses := []models.PresenceStatus{models.StatusOnline, models.StatusAway, models.StatusBusy}

	for i, userID := range users {
		presence := models.Presence{
			UserID: userID,
			Status: statuses[i],
			TTL:    time.Hour,
		}
		err = service.SetPresence(ctx, userID, presence)
		if err != nil {
			t.Fatalf("Failed to set presence for %s: %v", userID, err)
		}
	}

	// Get multiple presences
	results, err := service.GetMultiplePresences(ctx, []string{"user1", "user2", "user4"}) // user4 doesn't exist
	if err != nil {
		t.Fatalf("Failed to get multiple presences: %v", err)
	}

	// Should get user1 and user2, but not user4
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if presence, exists := results["user1"]; !exists {
		t.Error("Expected user1 in results")
	} else if presence.Status != models.StatusOnline {
		t.Errorf("Expected user1 status %s, got %s", models.StatusOnline, presence.Status)
	}

	if presence, exists := results["user2"]; !exists {
		t.Error("Expected user2 in results")
	} else if presence.Status != models.StatusAway {
		t.Errorf("Expected user2 status %s, got %s", models.StatusAway, presence.Status)
	}

	if _, exists := results["user4"]; exists {
		t.Error("Did not expect user4 in results")
	}
}

func TestPresenceService_NotFound(t *testing.T) {
	// Create test components
	memCache := cache.NewMemoryCache(100, time.Minute)

	store, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: "test-presence-notfound",
	})
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	service := NewPresenceService(memCache, store, "test-node")
	ctx := context.Background()

	// Try to get non-existent presence
	_, err = service.GetPresence(ctx, "nonexistent-user")
	if err == nil {
		t.Error("Expected error for non-existent user")
	}

	if _, ok := err.(*PresenceNotFoundError); !ok {
		t.Errorf("Expected PresenceNotFoundError, got %T", err)
	}
}
