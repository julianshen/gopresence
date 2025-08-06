package nats

import (
	"context"
	"testing"
)

// This test exercises center node creating a bucket vs accessing existing
func TestNewKVStore_CreateThenGetBucket(t *testing.T){
	// First create center with embedded server; should create bucket
	s1, err := NewKVStore(KVConfig{Embedded:true, BucketName:"bucket-create", NodeType:"center"})
	if err != nil { t.Fatalf("create center: %v", err) }
	defer s1.Close()
	// Second store to same embedded server URL (reuse url) should get existing bucket
	ks1 := s1.(*kvStore)
	url := ks1.config.ServerURL
	s2, err := NewKVStore(KVConfig{Embedded:false, ServerURL:url, BucketName:"bucket-create", NodeType:"center"})
	if err != nil { t.Fatalf("get existing bucket: %v", err) }
	defer s2.Close()
}

// This test ensures Delete handles non-existent key gracefully
func TestDelete_NotFoundIsIgnored(t *testing.T){
	s, err := NewKVStore(KVConfig{Embedded:true, BucketName:"del-bkt", NodeType:"center"})
	if err != nil { t.Fatalf("kv: %v", err) }
	defer s.Close()
	if err := s.Delete(context.Background(), "nope"); err != nil { t.Fatalf("expected no error on deleting missing key, got %v", err) }
}
