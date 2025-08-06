package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"gopresence/internal/cache"
	"gopresence/internal/models"
	"gopresence/internal/nats"
)

type fakeStore2 struct{ nats.KVStore }

func (f *fakeStore2) GetMultiple(ctx context.Context, ids []string) (map[string]models.Presence, error) {
	return nil, errors.New("boom")
}

func (f *fakeStore2) Get(ctx context.Context, userID string) (models.Presence, error) { return models.Presence{}, nil }
func (f *fakeStore2) Set(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error { return nil }
func (f *fakeStore2) Delete(ctx context.Context, userID string) error { return nil }
func (f *fakeStore2) Watch(ctx context.Context, cb func(nats.WatchEvent)) error { return nil }
func (f *fakeStore2) Close() error { return nil }

func TestGetMultiplePresences_StoreError(t *testing.T) {
	mc := cache.NewMemoryCache(10, time.Minute)
	s := NewPresenceService(mc, &fakeStore2{}, "n1")
	if _, err := s.GetMultiplePresences(context.Background(), []string{"a", "b"}); err == nil {
		t.Fatalf("expected store error")
	}
}

// A concrete fake implementing KVStore with GetMultiple returning u1
// This avoids embedded composite literal pitfalls in tests.
type fakeStoreMulti struct{}

func (f *fakeStoreMulti) Get(ctx context.Context, userID string) (models.Presence, error) {
	return models.Presence{UserID: userID, UpdatedAt: time.Now().UTC(), TTL: time.Minute}, nil
}
func (f *fakeStoreMulti) Set(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error { return nil }
func (f *fakeStoreMulti) Delete(ctx context.Context, userID string) error { return nil }
func (f *fakeStoreMulti) GetMultiple(ctx context.Context, ids []string) (map[string]models.Presence, error) {
	m := map[string]models.Presence{}
	for _, id := range ids {
		if id == "u1" {
			m[id] = models.Presence{UserID: id, UpdatedAt: time.Now().UTC(), TTL: time.Minute}
		}
	}
	return m, nil
}
func (f *fakeStoreMulti) Watch(ctx context.Context, cb func(nats.WatchEvent)) error { return nil }
func (f *fakeStoreMulti) Close() error { return nil }

func TestGetMultiplePresences_DeletesExpiredFromCache(t *testing.T) {
	mc := cache.NewMemoryCache(10, time.Minute)
	now := time.Now().UTC().Add(-2 * time.Minute)
	mc.Set("u1", models.Presence{UserID: "u1", UpdatedAt: now, TTL: time.Minute}, time.Minute)

	s := NewPresenceService(mc, &fakeStoreMulti{}, "n1")
	res, _ := s.GetMultiplePresences(context.Background(), []string{"u1"})
	if _, ok := res["u1"]; !ok {
		t.Fatalf("expected u1 present in result after refresh")
	}
}