package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/models"
	"gopresence/internal/nats"
)

// benchmarkStore is a fast in-memory fake implementing KVStore to avoid I/O
type benchmarkStore struct{}

func (b *benchmarkStore) Get(ctx context.Context, userID string) (models.Presence, error) {
	return models.Presence{UserID: userID, Status: models.StatusOnline, UpdatedAt: time.Now().UTC(), TTL: time.Hour}, nil
}
func (b *benchmarkStore) Set(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error { return nil }
func (b *benchmarkStore) Delete(ctx context.Context, userID string) error { return nil }
func (b *benchmarkStore) GetMultiple(ctx context.Context, ids []string) (map[string]models.Presence, error) {
	m := make(map[string]models.Presence, len(ids))
	for _, id := range ids { m[id] = models.Presence{UserID:id, Status: models.StatusOnline, UpdatedAt: time.Now().UTC(), TTL: time.Hour} }
	return m, nil
}
func (b *benchmarkStore) Watch(ctx context.Context, cb func(nats.WatchEvent)) error { return nil }
func (b *benchmarkStore) Close() error { return nil }

func benchmarkService(b *testing.B) *PresenceService {
	mc := cache.NewMemoryCache(10000, time.Minute)
	bs := &benchmarkStore{}
	return NewPresenceService(mc, bs, "bench-node")
}

func BenchmarkService_SetPresence(b *testing.B) {
	svc := benchmarkService(b)
	ctx := context.Background()
	p := models.Presence{UserID: "user", Status: models.StatusOnline, Message: "bench", TTL: time.Minute}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		uid := "user" + strconv.Itoa(i%1000) // reuse a pool of users to exercise cache
		_ = svc.SetPresence(ctx, uid, p)
	}
}

func BenchmarkService_GetPresence_CacheHit(b *testing.B) {
	svc := benchmarkService(b)
	ctx := context.Background()
	// warm cache
	p := models.Presence{UserID: "u0", Status: models.StatusOnline, TTL: time.Hour, UpdatedAt: time.Now().UTC()}
	_ = svc.SetPresence(ctx, "u0", p)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = svc.GetPresence(ctx, "u0")
	}
}

func BenchmarkService_GetPresence_Mixed(b *testing.B) {
	svc := benchmarkService(b)
	ctx := context.Background()
	// pre-populate some
	p := models.Presence{UserID: "user", Status: models.StatusOnline, TTL: time.Hour}
	for i := 0; i < 500; i++ { _ = svc.SetPresence(ctx, "u"+strconv.Itoa(i), p) }
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		uid := "u" + strconv.Itoa(i%1000)
		_, _ = svc.GetPresence(ctx, uid)
	}
}
