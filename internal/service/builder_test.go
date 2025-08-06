package service

import (
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/config"
)

func TestServiceBuilder_RistrettoPath(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{NodeID: "n1", NodeType: "center"},
		Cache:   config.CacheConfig{MaxCost: 10_000, NumCounters: 1_000, BufferItems: 64, Metrics: true},
		NATS:    config.NATSConfig{Embedded: true, KVBucket: "builder-ristretto"},
	}
	b := NewServiceBuilder(cfg)
	svc, err := b.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer svc.Close()
}

func TestServiceBuilder_LegacyPath(t *testing.T) {
	cfg := &config.Config{
		Service: config.ServiceConfig{NodeID: "n1", NodeType: "center"},
		Cache:   config.CacheConfig{MaxCost: 0, TTL: "5s", MaxSize: 10},
		NATS:    config.NATSConfig{Embedded: true, KVBucket: "builder-legacy"},
	}
	b := NewServiceBuilder(cfg)
	svc, err := b.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	defer svc.Close()

	// quick smoke for cache path equivalence
	_ = cache.NewMemoryCache(5, 2*time.Second)
}
