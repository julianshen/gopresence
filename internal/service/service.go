package service

import (
	"context"
	"fmt"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/config"
	"gopresence/internal/models"
	"gopresence/internal/nats"
)

// PresenceService implements the core business logic for presence management
type PresenceService struct {
	cache cache.MemoryCache
	store nats.KVStore
	nodeID string
}

// Ready checks whether dependencies are available (e.g., KV store)
func (s *PresenceService) Ready(ctx context.Context) error {
	// Use a lightweight call to validate store connectivity
	_, err := s.store.GetMultiple(ctx, []string{})
	return err
}

// NewPresenceService creates a new presence service
func NewPresenceService(cache cache.MemoryCache, store nats.KVStore, nodeID string) *PresenceService {
	return &PresenceService{
		cache:  cache,
		store:  store,
		nodeID: nodeID,
	}
}

// GetPresence retrieves a user's presence, checking cache first
func (s *PresenceService) GetPresence(ctx context.Context, userID string) (models.Presence, error) {
	// Try cache first
	if presence, found := s.cache.Get(userID); found {
		// Check if expired
		if !presence.IsExpired() {
			return presence, nil
		}
		// Remove expired entry from cache
		s.cache.Delete(userID)
	}

	// Fall back to KV store
	presence, err := s.store.Get(ctx, userID)
	if err != nil {
		return models.Presence{}, &PresenceNotFoundError{UserID: userID}
	}

	// Cache the result
	s.cache.Set(userID, presence, presence.TTL)

	return presence, nil
}

// SetPresence sets a user's presence in both cache and store
func (s *PresenceService) SetPresence(ctx context.Context, userID string, presence models.Presence) error {
	// Set node ID and timestamps
	presence.NodeID = s.nodeID
	presence.UpdatedAt = time.Now().UTC()
	presence.LastSeen = presence.UpdatedAt

	// Validate presence
	if err := presence.Validate(); err != nil {
		return fmt.Errorf("invalid presence: %w", err)
	}

	// Store in KV store first
	if err := s.store.Set(ctx, userID, presence, presence.TTL); err != nil {
		return fmt.Errorf("failed to store presence: %w", err)
	}

	// Update cache
	s.cache.Set(userID, presence, presence.TTL)

	return nil
}

// GetMultiplePresences retrieves multiple users' presences
func (s *PresenceService) GetMultiplePresences(ctx context.Context, userIDs []string) (map[string]models.Presence, error) {
	result := make(map[string]models.Presence)
	var missingUsers []string

	// Check cache first
	for _, userID := range userIDs {
		if presence, found := s.cache.Get(userID); found && !presence.IsExpired() {
			result[userID] = presence
		} else {
			if found && presence.IsExpired() {
				s.cache.Delete(userID)
			}
			missingUsers = append(missingUsers, userID)
		}
	}

	// Fetch missing users from store
	if len(missingUsers) > 0 {
		storeResults, err := s.store.GetMultiple(ctx, missingUsers)
		if err != nil {
			return nil, fmt.Errorf("failed to get presences from store: %w", err)
		}

		// Add store results to final result and cache them
		for userID, presence := range storeResults {
			result[userID] = presence
			s.cache.Set(userID, presence, presence.TTL)
		}
	}

	return result, nil
}

// Close closes the service and its dependencies
func (s *PresenceService) Close() error {
	if err := s.store.Close(); err != nil {
		return fmt.Errorf("failed to close store: %w", err)
	}
	s.cache.Clear()
	return nil
}

// PresenceNotFoundError represents an error when a presence is not found
type PresenceNotFoundError struct {
	UserID string
}

func (e *PresenceNotFoundError) Error() string {
	return fmt.Sprintf("presence not found for user %s", e.UserID)
}

// ServiceBuilder helps build a complete presence service
type ServiceBuilder struct {
	config *config.Config
}

// NewServiceBuilder creates a new service builder
func NewServiceBuilder(config *config.Config) *ServiceBuilder {
	return &ServiceBuilder{config: config}
}

// Build builds and configures all service components
func (b *ServiceBuilder) Build() (*PresenceService, error) {
	// Create cache - use new Ristretto config if available, fallback to legacy
	var memCache cache.MemoryCache
	if b.config.Cache.MaxCost > 0 {
		// Use Ristretto-specific configuration
		ristrettoConfig := cache.RistrettoConfig{
			MaxCost:     b.config.Cache.MaxCost,
			NumCounters: b.config.Cache.NumCounters,
			BufferItems: b.config.Cache.BufferItems,
			Metrics:     b.config.Cache.Metrics,
		}
		var err error
		memCache, err = cache.NewRistrettoCache(ristrettoConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ristretto cache: %w", err)
		}
	} else {
		// Fallback to legacy configuration
		cacheTTL, err := b.config.Cache.GetCacheTTL()
		if err != nil {
			return nil, fmt.Errorf("invalid cache TTL: %w", err)
		}
		memCache = cache.NewMemoryCache(b.config.Cache.MaxSize, cacheTTL)
	}

	// Create NATS KV store
	natsConfig := nats.KVConfig{
		ServerURL:   b.config.NATS.ServerURL,
		BucketName:  b.config.NATS.KVBucket,
		Embedded:    b.config.NATS.Embedded,
		DataDir:     b.config.NATS.DataDir,
		NodeType:    b.config.Service.NodeType,
		CenterURL:   b.config.NATS.CenterURL,
		LeafPort:    b.config.NATS.LeafPort,
		ClusterPort: b.config.NATS.ClusterPort,
	}

	store, err := nats.NewKVStore(natsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS KV store: %w", err)
	}

	// Create presence service
	service := NewPresenceService(memCache, store, b.config.Service.NodeID)

	return service, nil
}
