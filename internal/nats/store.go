package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"gopresence/internal/models"
)

// KVStore defines the interface for NATS KV operations
type KVStore interface {
	Get(ctx context.Context, userID string) (models.Presence, error)
	Set(ctx context.Context, userID string, presence models.Presence, ttl time.Duration) error
	Delete(ctx context.Context, userID string) error
	GetMultiple(ctx context.Context, userIDs []string) (map[string]models.Presence, error)
	Watch(ctx context.Context, callback func(WatchEvent)) error
	Close() error
}

// WatchEventType represents the type of watch event
type WatchEventType string

const (
	WatchEventPut    WatchEventType = "PUT"
	WatchEventDelete WatchEventType = "DELETE"
)

// WatchEvent represents a change event in the KV store
type WatchEvent struct {
	Key      string
	Type     WatchEventType
	Presence *models.Presence
}

// KVConfig holds configuration for the KV store
type KVConfig struct {
	ServerURL    string
	BucketName   string
	Embedded     bool
	DataDir      string
	NodeType     string // "center" or "leaf"
	CenterURL    string // URL of center node (for leaf nodes)
	LeafPort     int    // Port for leaf connections (for center nodes)
	ClusterPort  int    // Port for cluster connections (for center nodes)
	StartTimeout string // Startup wait duration, e.g., "30s"
}

// kvStore implements KVStore using NATS KV
type kvStore struct {
	config KVConfig
	server *server.Server
	conn   *nats.Conn
	js     jetstream.JetStream
	kv     jetstream.KeyValue
}

// NewKVStore creates a new NATS KV store
func NewKVStore(config KVConfig) (KVStore, error) {
	store := &kvStore{
		config: config,
	}

	// Start embedded server if configured
	if config.Embedded {
		if err := store.startEmbeddedServer(); err != nil {
			return nil, fmt.Errorf("failed to start embedded server: %w", err)
		}
	}

	// Connect to NATS
	serverURL := config.ServerURL
	if serverURL == "" {
		if config.NodeType == "leaf" && config.CenterURL != "" {
			// Leaf nodes should connect to center node for KV operations
			serverURL = config.CenterURL
		} else {
			serverURL = nats.DefaultURL
		}
	}

	conn, err := nats.Connect(serverURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
	)
	if err != nil {
		store.cleanup()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	store.conn = conn

	// Default to center node if NodeType is not specified (for backward compatibility)
	nodeType := config.NodeType
	if nodeType == "" {
		nodeType = "center"
	}

	// Create JetStream context and KV only for center nodes or when connecting to center
	if nodeType == "center" || (nodeType == "leaf" && config.CenterURL != "") {
		js, err := jetstream.New(conn)
		if err != nil {
			store.cleanup()
			return nil, fmt.Errorf("failed to create JetStream context: %w", err)
		}
		store.js = js

		// Create or get KV bucket
		bucketName := config.BucketName
		if bucketName == "" {
			bucketName = "presence"
		}

		// Only center nodes can create KV buckets
		if nodeType == "center" {
			kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
				Bucket: bucketName,
				TTL:    time.Hour, // Default TTL
			})
			if err != nil {
				// Try to get existing bucket
				kv, err = js.KeyValue(context.Background(), bucketName)
				if err != nil {
					store.cleanup()
					return nil, fmt.Errorf("failed to create/get KV bucket: %w", err)
				}
			}
			store.kv = kv
		} else {
			// Leaf nodes access existing KV bucket
			kv, err := js.KeyValue(context.Background(), bucketName)
			if err != nil {
				store.cleanup()
				return nil, fmt.Errorf("failed to access KV bucket: %w", err)
			}
			store.kv = kv
		}
	} else {
		store.cleanup()
		return nil, fmt.Errorf("leaf nodes must specify center URL for KV operations")
	}

	return store, nil
}

// Get retrieves a presence from the KV store
func (s *kvStore) Get(ctx context.Context, userID string) (models.Presence, error) {
	key := s.presenceKey(userID)

	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		// Check for various "not found" error types
		if errors.Is(err, jetstream.ErrKeyNotFound) ||
			strings.Contains(err.Error(), "not found") ||
			strings.Contains(err.Error(), "no message found") {
			return models.Presence{}, fmt.Errorf("presence not found for user %s", userID)
		}
		return models.Presence{}, fmt.Errorf("failed to get presence: %w", err)
	}

	// Check if the entry is nil or has no data
	if entry == nil || len(entry.Value()) == 0 {
		return models.Presence{}, fmt.Errorf("presence not found for user %s", userID)
	}

	var presence models.Presence
	if err := json.Unmarshal(entry.Value(), &presence); err != nil {
		return models.Presence{}, fmt.Errorf("failed to unmarshal presence: %w", err)
	}

	// Additional validation - check if this is actually a valid presence
	if err := presence.Validate(); err != nil {
		return models.Presence{}, fmt.Errorf("presence not found for user %s", userID)
	}

	return presence, nil
}

// Set stores a presence in the KV store
func (s *kvStore) Set(ctx context.Context, userID string, presence models.Presence, ttl time.Duration) error {
	key := s.presenceKey(userID)

	data, err := json.Marshal(presence)
	if err != nil {
		return fmt.Errorf("failed to marshal presence: %w", err)
	}

	// Note: NATS KV doesn't support per-key TTL easily, so we rely on bucket-level TTL
	// Individual key TTL would require additional application-level logic
	_, err = s.kv.Put(ctx, key, data)
	if err != nil {
		return fmt.Errorf("failed to put presence: %w", err)
	}

	return nil
}

// Delete removes a presence from the KV store
func (s *kvStore) Delete(ctx context.Context, userID string) error {
	key := s.presenceKey(userID)

	err := s.kv.Delete(ctx, key)
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("failed to delete presence: %w", err)
	}

	return nil
}

// GetMultiple retrieves multiple presences from the KV store
func (s *kvStore) GetMultiple(ctx context.Context, userIDs []string) (map[string]models.Presence, error) {
	result := make(map[string]models.Presence)

	for _, userID := range userIDs {
		presence, err := s.Get(ctx, userID)
		if err == nil {
			result[userID] = presence
		}
		// Ignore not found errors, just skip those users
	}

	return result, nil
}

// Watch watches for changes in the KV store
func (s *kvStore) Watch(ctx context.Context, callback func(WatchEvent)) error {
	watcher, err := s.kv.WatchAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	go func() {
		defer watcher.Stop()

		for {
			select {
			case entry := <-watcher.Updates():
				if entry == nil {
					return
				}

				event := WatchEvent{
					Key: entry.Key(),
				}

				if entry.Operation() == jetstream.KeyValuePut {
					event.Type = WatchEventPut
					var presence models.Presence
					if err := json.Unmarshal(entry.Value(), &presence); err == nil {
						event.Presence = &presence
					}
				} else if entry.Operation() == jetstream.KeyValueDelete {
					event.Type = WatchEventDelete
				}

				callback(event)

			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close closes the KV store and cleans up resources
func (s *kvStore) Close() error {
	return s.cleanup()
}

// presenceKey generates a KV key for a user presence
func (s *kvStore) presenceKey(userID string) string {
	return fmt.Sprintf("user.%s", userID)
}

// startEmbeddedServer starts an embedded NATS server
func (s *kvStore) startEmbeddedServer() error {
	// Default to center node if NodeType is not specified
	nodeType := s.config.NodeType
	if nodeType == "" {
		nodeType = "center"
	}

	opts := &server.Options{
		Host:       "0.0.0.0",
		Port:       -1,                   // Random port for client connections
		JetStream:  nodeType == "center", // Only center nodes have JetStream
		ServerName: fmt.Sprintf("%s-%d", nodeType, time.Now().UnixNano()),
	}

	if s.config.DataDir != "" {
		opts.StoreDir = s.config.DataDir
		// Ensure directory exists and is writable
		if err := ensureDirectory(s.config.DataDir); err != nil {
			return fmt.Errorf("failed to ensure data directory: %w", err)
		}
	}

	// Configure based on node type
	if nodeType == "center" {
		// Center node configuration
		opts.JetStreamMaxMemory = 64 * 1024 * 1024  // 64MB
		opts.JetStreamMaxStore = 1024 * 1024 * 1024 // 1GB
		
		// Enable debug logging for JetStream issues
		opts.Debug = false  // Set to true for more verbose logging if needed
		opts.Trace = false  // Set to true for even more verbose logging if needed

		// Setup leaf node connections
		if s.config.LeafPort > 0 {
			opts.LeafNode.Host = "0.0.0.0"
			opts.LeafNode.Port = s.config.LeafPort
		}

		// Setup cluster if configured
		if s.config.ClusterPort > 0 {
			opts.Cluster.Host = "0.0.0.0"
			opts.Cluster.Port = s.config.ClusterPort
			opts.Cluster.Name = "presence-cluster"
		}

	} else if nodeType == "leaf" {
		// Leaf node configuration
		if s.config.CenterURL != "" {
			centerURL, err := url.Parse(s.config.CenterURL)
			if err != nil {
				return fmt.Errorf("invalid center URL: %w", err)
			}
			opts.LeafNode.Remotes = []*server.RemoteLeafOpts{
				{
					URLs: []*url.URL{centerURL},
				},
			}
		}
		// Leaf nodes don't have JetStream
		opts.JetStream = false
	}

	// Log important startup params - use simplified opts for actual server
	fmt.Printf("NATS embedded start: nodeType=%s dataDir=%s host=%s jetstream=%t\n", nodeType, s.config.DataDir, "0.0.0.0", nodeType == "center")
	
	// Create server with simplified options - basic embedded server
	simpleOpts := &server.Options{
		Host:      "0.0.0.0",
		Port:      -1,
		JetStream: nodeType == "center",
		ServerName: fmt.Sprintf("%s-%d", nodeType, time.Now().UnixNano()),
	}
	
	if s.config.DataDir != "" && nodeType == "center" {
		simpleOpts.StoreDir = s.config.DataDir
		simpleOpts.JetStreamMaxMemory = 32 * 1024 * 1024  // Reduce to 32MB
		simpleOpts.JetStreamMaxStore = 256 * 1024 * 1024  // Reduce to 256MB
	}

	ns, err := server.NewServer(simpleOpts)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server in background
	go ns.Start()

	// Determine timeout - increase default for center nodes with JetStream
	var timeout time.Duration
	if s.config.StartTimeout != "" {
		if d, err := time.ParseDuration(s.config.StartTimeout); err == nil {
			timeout = d
		}
	}
	if timeout == 0 {
		if nodeType == "center" {
			timeout = 30 * time.Second  // Increased from 15s to 30s for JetStream initialization
		} else {
			timeout = 15 * time.Second  // Increased from 10s to 15s
		}
	}

	// Wait for server to be ready with progress logging
	fmt.Printf("NATS server starting, waiting up to %v for readiness...\n", timeout)
	
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	// Poll for readiness with status updates
	for {
		if ns.ReadyForConnections(100 * time.Millisecond) {
			break
		}
		
		elapsed := time.Since(startTime)
		if elapsed >= timeout {
			ns.Shutdown()
			return fmt.Errorf("server failed to start within %v (node type: %s)", timeout, nodeType)
		}
		
		select {
		case <-ticker.C:
			fmt.Printf("NATS server still starting... elapsed: %v, JetStream: %t\n", elapsed.Truncate(time.Second), simpleOpts.JetStream)
		default:
		}
		
		time.Sleep(100 * time.Millisecond)
	}

	s.server = ns

	// Update config with server URL
	s.config.ServerURL = ns.ClientURL()
	fmt.Printf("NATS embedded started: url=%s\n", s.config.ServerURL)

	return nil
}

// cleanup closes connections and shuts down embedded server
func (s *kvStore) cleanup() error {
	if s.conn != nil {
		s.conn.Close()
	}

	if s.server != nil {
		s.server.Shutdown()
		s.server.WaitForShutdown()
	}

	return nil
}

// ensureDirectory creates the directory if it doesn't exist and verifies it's writable
func ensureDirectory(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Test if directory is writable by creating a temporary file
	testFile := dir + "/.write-test"
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("directory not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)
	
	return nil
}
