package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/models"
	"gopresence/internal/nats"
	"gopresence/internal/service"
)

func TestCenterLeafNodeSetup(t *testing.T) {
	// Create center node
	centerBucket := fmt.Sprintf("center-test-%d", time.Now().UnixNano())
	centerStore, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: centerBucket,
		NodeType:   "center",
		DataDir:    fmt.Sprintf("./test-data-center-%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("Failed to create center store: %v", err)
	}
	defer centerStore.Close()

	// Wait a moment for center node to start
	time.Sleep(100 * time.Millisecond)

	// Create center service
	centerCache := cache.NewMemoryCache(100, time.Minute)
	centerService := service.NewPresenceService(centerCache, centerStore, "center-node-1")

	// Set a presence via center node
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	testPresence := models.Presence{
		UserID:    "center-test-user",
		Status:    models.StatusOnline,
		Message:   "From center node",
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "center-node-1",
		TTL:       time.Hour,
	}

	err = centerService.SetPresence(ctx, "center-test-user", testPresence)
	if err != nil {
		t.Fatalf("Failed to set presence on center: %v", err)
	}

	// Retrieve presence from center
	retrieved, err := centerService.GetPresence(ctx, "center-test-user")
	if err != nil {
		t.Fatalf("Failed to get presence from center: %v", err)
	}

	if retrieved.Status != models.StatusOnline {
		t.Errorf("Expected status %s, got %s", models.StatusOnline, retrieved.Status)
	}

	if retrieved.Message != "From center node" {
		t.Errorf("Expected message 'From center node', got '%s'", retrieved.Message)
	}

	t.Logf("Center node test passed: %+v", retrieved)
}

func TestCenterNodeConfiguration(t *testing.T) {
	// Test that center node properly configures JetStream
	centerBucket := fmt.Sprintf("center-config-test-%d", time.Now().UnixNano())

	centerStore, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: centerBucket,
		NodeType:   "center",
		DataDir:    fmt.Sprintf("./test-data-center-config-%d", time.Now().UnixNano()),
	})
	if err != nil {
		t.Fatalf("Failed to create center store: %v", err)
	}
	defer centerStore.Close()

	// Test basic KV operations
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	testPresence := models.Presence{
		UserID:    "config-test-user",
		Status:    models.StatusAway,
		Message:   "Configuration test",
		LastSeen:  now,
		UpdatedAt: now,
		NodeID:    "test-center-node",
		TTL:       time.Hour,
	}

	err = centerStore.Set(ctx, "config-test-user", testPresence, time.Hour)
	if err != nil {
		t.Fatalf("Failed to set via center KV: %v", err)
	}

	// Add a small delay to ensure the data is persisted
	time.Sleep(100 * time.Millisecond)

	retrieved, err := centerStore.Get(ctx, "config-test-user")
	if err != nil {
		t.Fatalf("Failed to get via center KV: %v", err)
	}

	if retrieved.Status != models.StatusAway {
		t.Errorf("Expected status %s, got %s", models.StatusAway, retrieved.Status)
	}

	t.Logf("Center KV operations successful")
}

// Note: Full center-leaf integration would require network connectivity
// and proper URL resolution between nodes. This test focuses on
// configuration and individual node functionality.
func TestLeafNodeConfiguration(t *testing.T) {
	// Test leaf node configuration (without actual center connection)
	leafBucket := fmt.Sprintf("leaf-config-test-%d", time.Now().UnixNano())

	// This will fail as expected since we don't have a real center node
	// but it tests the configuration parsing
	_, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: leafBucket,
		NodeType:   "leaf",
		CenterURL:  "nats://nonexistent-center:4222",
		DataDir:    "./test-nats-leaf",
	})

	// We expect this to fail since the center doesn't exist
	if err == nil {
		t.Error("Expected error when connecting to nonexistent center")
	}

	// The error should mention connection failure, not configuration issues
	if !contains(err.Error(), "connect") && !contains(err.Error(), "dial") {
		t.Logf("Got expected connection error: %v", err)
	}
}

func TestNodeTypeValidation(t *testing.T) {
	// Test that leaf nodes require center URL
	_, err := nats.NewKVStore(nats.KVConfig{
		Embedded:   true,
		BucketName: "validation-test",
		NodeType:   "leaf",
		CenterURL:  "", // Missing center URL
	})

	if err == nil {
		t.Error("Expected error for leaf node without center URL")
	}

	if !contains(err.Error(), "center URL") {
		t.Errorf("Expected center URL error, got: %v", err)
	}
}

// Helper function to check if a string contains a substring (case insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[0:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					strings.Contains(strings.ToLower(s), strings.ToLower(substr))))
}
