package nats

import (
	"context"
	"testing"

	js "github.com/nats-io/nats.go/jetstream"
)

type fakeKV struct{ watchErr error }

func (f *fakeKV) WatchAll(ctx context.Context, opts ...js.WatchOpt) (js.KeyWatcher, error) {
	return nil, f.watchErr
}

// We can’t easily swap kv inside kvStore from tests without exposing fields, so for now
// we cover constructor and Get() error branches via controlled inputs.

func TestNewKVStore_LeafWithoutCenterURL(t *testing.T) {
	_, err := NewKVStore(KVConfig{Embedded: false, NodeType: "leaf", CenterURL: "", ServerURL: "nats://127.0.0.1:1"})
	if err == nil {
		t.Fatalf("expected error when leaf without center URL")
	}
}

func TestNewKVStore_InvalidCenterURL(t *testing.T) {
	// Embedded leaf with invalid center URL triggers startEmbeddedServer error path
	_, err := NewKVStore(KVConfig{Embedded: true, NodeType: "leaf", CenterURL: "::://bad-url", BucketName: "b1"})
	if err == nil {
		t.Fatalf("expected error for invalid center URL")
	}
}

func TestNewKVStore_ConnectFailure(t *testing.T) {
	// Unreachable server URL should fail connect
	_, err := NewKVStore(KVConfig{Embedded: false, ServerURL: "nats://127.0.0.1:42213", BucketName: "b1", NodeType: "center"})
	if err == nil {
		t.Fatalf("expected connect error")
	}
}

func TestGet_UnmarshalOrValidateFailure(t *testing.T) {
	store, err := NewKVStore(KVConfig{Embedded: true, BucketName: "badjson", NodeType: "center"})
	if err != nil {
		t.Fatalf("kv create: %v", err)
	}
	defer store.Close()
	ks := store.(*kvStore)
	ctx := context.Background()
	// Manually put bad JSON via kv API
	ks.kv.Put(ctx, ks.presenceKey("u1"), []byte("{bad json"))
	if _, err := store.Get(ctx, "u1"); err == nil {
		t.Fatalf("expected unmarshal error path -> wrapped error")
	}
	// Put JSON that fails Validate (missing required fields). Use empty object
	ks.kv.Put(ctx, ks.presenceKey("u2"), []byte("{}"))
	if _, err := store.Get(ctx, "u2"); err == nil {
		t.Fatalf("expected validate-based not found")
	}
}

// We can cover Watch() error by calling Watch when kvStore has been closed (kv nil)
func TestWatch_ErrorWhenClosed(t *testing.T) {
	store, err := NewKVStore(KVConfig{Embedded: true, BucketName: "watcherr", NodeType: "center"})
	if err != nil {
		t.Fatalf("kv create: %v", err)
	}
	// Close the store so Watch should fail gracefully
	_ = store.Close()
	if err := store.Watch(context.Background(), func(WatchEvent) {}); err == nil {
		t.Fatalf("expected error when watcher cannot be created")
	}
}
