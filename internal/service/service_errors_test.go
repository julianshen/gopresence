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

type fakeStore struct {
	get   func(ctx context.Context, userID string) (models.Presence, error)
	set   func(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error
	close func() error
}

func (f *fakeStore) Get(ctx context.Context, userID string) (models.Presence, error) {
	return f.get(ctx, userID)
}
func (f *fakeStore) Set(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error {
	return f.set(ctx, userID, p, ttl)
}
func (f *fakeStore) Delete(ctx context.Context, userID string) error { return nil }
func (f *fakeStore) GetMultiple(ctx context.Context, ids []string) (map[string]models.Presence, error) {
	return map[string]models.Presence{}, nil
}
func (f *fakeStore) Watch(ctx context.Context, cb func(nats.WatchEvent)) error { return nil }
func (f *fakeStore) Close() error {
	if f.close != nil {
		return f.close()
	}
	return nil
}

func TestGetPresence_CacheExpiredFallsBack(t *testing.T) {
	mc := cache.NewMemoryCache(10, time.Minute)
	now := time.Now().UTC().Add(-2 * time.Minute)
	mc.Set("u1", models.Presence{UserID: "u1", UpdatedAt: now, TTL: time.Minute}, time.Minute)
	fs := &fakeStore{get: func(ctx context.Context, userID string) (models.Presence, error) {
		return models.Presence{UserID: "u1", Status: models.StatusOnline, UpdatedAt: time.Now().UTC()}, nil
	}}
	s := NewPresenceService(mc, fs, "n1")
	p, err := s.GetPresence(context.Background(), "u1")
	if err != nil || p.UserID != "u1" {
		t.Fatalf("expected fetched from store: %v %v", p, err)
	}
}

func TestSetPresence_ValidateAndStoreErrors(t *testing.T) {
	mc := cache.NewMemoryCache(10, time.Minute)
	// invalid presence (empty user id inside presence after service sets id) won't fail Validate, so use invalid status
	bad := models.Presence{UserID: "u1", Status: "", TTL: time.Minute}
	fs := &fakeStore{set: func(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error { return nil }, get: func(ctx context.Context, userID string) (models.Presence, error) { return models.Presence{}, nil }}
	s := NewPresenceService(mc, fs, "n1")
	if err := s.SetPresence(context.Background(), "u1", bad); err == nil {
		t.Fatalf("expected validation error")
	}
	// now valid but store error
	good := models.Presence{UserID: "u1", Status: models.StatusOnline, TTL: time.Minute}
	fs.set = func(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error {
		return errors.New("store down")
	}
	if err := s.SetPresence(context.Background(), "u1", good); err == nil {
		t.Fatalf("expected store error")
	}
}

func TestClose_StoreCloseError(t *testing.T) {
	mc := cache.NewMemoryCache(10, time.Minute)
	fs := &fakeStore{get: func(ctx context.Context, userID string) (models.Presence, error) {
		return models.Presence{}, errors.New("x")
	}, set: func(ctx context.Context, userID string, p models.Presence, ttl time.Duration) error { return nil }, close: func() error { return errors.New("boom") }}
	s := NewPresenceService(mc, fs, "n1")
	if err := s.Close(); err == nil {
		t.Fatalf("expected close error")
	}
}
