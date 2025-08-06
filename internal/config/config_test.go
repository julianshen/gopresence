package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_ConfigFromEnvAndValidation(t *testing.T) {
	// Save and restore env
	saved := os.Environ()
	defer func() {
		// restore
		for _, kv := range saved {
			parts := []rune(kv)
			eq := -1
			for i, r := range parts {
				if r == '=' {
					eq = i
					break
				}
			}
			if eq >= 0 {
				os.Setenv(string(parts[:eq]), string(parts[eq+1:]))
			}
		}
	}()
	os.Clearenv()

	// Minimal required
	os.Setenv("JWT_SECRET", "secret")
	os.Setenv("CACHE_MAX_COST", "2048")
	os.Setenv("CACHE_NUM_COUNTERS", "1000")
	os.Setenv("CACHE_BUFFER_ITEMS", "64")
	os.Setenv("CACHE_METRICS", "true")
	os.Setenv("NATS_EMBEDDED", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Service.Name == "" || cfg.Service.Version == "" {
		t.Fatalf("defaults not applied for service fields")
	}
	if cfg.Auth.JWTSecret != "secret" {
		t.Fatalf("expected JWT secret from env")
	}

	// Duration getters (set after Load, so update the struct then reparse)
	os.Setenv("CACHE_TTL", "10s")
	os.Setenv("NATS_KV_TTL", "30s")
	os.Setenv("JWT_TTL", "1h")

	cfg.Cache.TTL = "10s"
	cfg.NATS.KVTTL = "30s"
	cfg.Auth.JWTTTL = "1h"

	d, err := cfg.Cache.GetCacheTTL()
	if err != nil || d != 10*time.Second {
		t.Fatalf("cache ttl parse failed: %v %v", d, err)
	}
	kd, err := cfg.NATS.GetKVTTL()
	if err != nil || kd != 30*time.Second {
		t.Fatalf("kv ttl parse failed: %v %v", kd, err)
	}
	jd, err := cfg.Auth.GetJWTTTL()
	if err != nil || jd != time.Hour {
		t.Fatalf("jwt ttl parse failed: %v %v", jd, err)
	}
}

func TestLoad_MissingJWTSecret(t *testing.T) {
	saved := os.Environ()
	defer func() {
		for _, kv := range saved {
			parts := []rune(kv)
			eq := -1
			for i, r := range parts {
				if r == '=' {
					eq = i
					break
				}
			}
			if eq >= 0 {
				os.Setenv(string(parts[:eq]), string(parts[eq+1:]))
			}
		}
	}()
	os.Clearenv()

	if _, err := Load(); err == nil {
		t.Fatalf("expected error when JWT_SECRET missing")
	}
}
