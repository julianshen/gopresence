package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds the application configuration
type Config struct {
	Service ServiceConfig `yaml:"service"`
	NATS    NATSConfig    `yaml:"nats"`
	Cache   CacheConfig   `yaml:"cache"`
	Auth    AuthConfig    `yaml:"auth"`
	Logging LoggingConfig `yaml:"logging"`
}

// ServiceConfig holds service-level configuration
type ServiceConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	Port     int    `yaml:"port"`
	NodeType string `yaml:"node_type"` // "center" or "leaf"
	NodeID   string `yaml:"node_id"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	Embedded           bool   `yaml:"embedded"`
	ServerURL          string `yaml:"server_url"`
	DataDir            string `yaml:"data_dir"`
	JetStreamMaxMemory int64  `yaml:"jetstream_max_memory"`
	JetStreamMaxStore  int64  `yaml:"jetstream_max_store"`
	KVBucket           string `yaml:"kv_bucket"`
	KVTTL              string `yaml:"kv_ttl"`
	CenterURL          string `yaml:"center_url"`       // URL of center node (for leaf nodes)
	LeafPort           int    `yaml:"leaf_port"`        // Port for leaf connections (for center nodes)
	ClusterPort        int    `yaml:"cluster_port"`     // Port for cluster connections
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Type         string `yaml:"type"`
	MaxSize      int    `yaml:"max_size"`       // Legacy: converted to MaxCost for Ristretto
	TTL          string `yaml:"ttl"`            // Legacy: used for backward compatibility
	MaxCost      int64  `yaml:"max_cost"`       // Ristretto: Maximum memory cost in bytes
	NumCounters  int64  `yaml:"num_counters"`   // Ristretto: Number of counters for TinyLFU
	BufferItems  int64  `yaml:"buffer_items"`   // Ristretto: Buffer size for async operations  
	Metrics      bool   `yaml:"metrics"`        // Ristretto: Enable cache metrics
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
	JWTIssuer string `yaml:"jwt_issuer"`
	JWTTTL    string `yaml:"jwt_ttl"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Service: ServiceConfig{
			Name:     getEnvOrDefault("SERVICE_NAME", "presence-service"),
			Version:  getEnvOrDefault("SERVICE_VERSION", "v2"),
			Port:     getEnvIntOrDefault("SERVICE_PORT", 8080),
			NodeType: getEnvOrDefault("NODE_TYPE", "center"),
			NodeID:   getEnvOrDefault("NODE_ID", "node-1"),
		},
		NATS: NATSConfig{
			Embedded:           getEnvBoolOrDefault("NATS_EMBEDDED", true),
			ServerURL:          getEnvOrDefault("NATS_SERVER_URL", ""),
			DataDir:            getEnvOrDefault("NATS_DATA_DIR", "./nats-data"),
			JetStreamMaxMemory: getEnvInt64OrDefault("NATS_JETSTREAM_MAX_MEMORY", 64*1024*1024), // 64MB
			JetStreamMaxStore:  getEnvInt64OrDefault("NATS_JETSTREAM_MAX_STORE", 1024*1024*1024), // 1GB
			KVBucket:           getEnvOrDefault("NATS_KV_BUCKET", "presence"),
			KVTTL:              getEnvOrDefault("NATS_KV_TTL", "3600s"),
			CenterURL:          getEnvOrDefault("NATS_CENTER_URL", ""),
			LeafPort:           getEnvIntOrDefault("NATS_LEAF_PORT", 7422),
			ClusterPort:        getEnvIntOrDefault("NATS_CLUSTER_PORT", 6222),
		},
		Cache: CacheConfig{
			Type:        getEnvOrDefault("CACHE_TYPE", "ristretto"),
			MaxSize:     getEnvIntOrDefault("CACHE_MAX_SIZE", 10000),      // Legacy support
			TTL:         getEnvOrDefault("CACHE_TTL", "300s"),             // Legacy support
			MaxCost:     getEnvInt64OrDefault("CACHE_MAX_COST", 1000000),  // 1MB default
			NumCounters: getEnvInt64OrDefault("CACHE_NUM_COUNTERS", 100000),
			BufferItems: getEnvInt64OrDefault("CACHE_BUFFER_ITEMS", 64),
			Metrics:     getEnvBoolOrDefault("CACHE_METRICS", true),
		},
		Auth: AuthConfig{
			JWTSecret: getEnvOrDefault("JWT_SECRET", ""),
			JWTIssuer: getEnvOrDefault("JWT_ISSUER", "presence-service"),
			JWTTTL:    getEnvOrDefault("JWT_TTL", "24h"),
		},
		Logging: LoggingConfig{
			Level:  getEnvOrDefault("LOG_LEVEL", "info"),
			Format: getEnvOrDefault("LOG_FORMAT", "json"),
		},
	}

	// Validate required fields
	if config.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return config, nil
}

// GetCacheTTL returns cache TTL as duration
func (c *CacheConfig) GetCacheTTL() (time.Duration, error) {
	return time.ParseDuration(c.TTL)
}

// Note: GetCleanupInterval removed as Ristretto handles cleanup automatically

// GetKVTTL returns KV TTL as duration
func (c *NATSConfig) GetKVTTL() (time.Duration, error) {
	return time.ParseDuration(c.KVTTL)
}

// GetJWTTTL returns JWT TTL as duration
func (c *AuthConfig) GetJWTTTL() (time.Duration, error) {
	return time.ParseDuration(c.JWTTTL)
}

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64OrDefault(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}