# Presence Service Design Document

## 1. Overview

The Presence Service is a distributed Go-based microservice that manages user presence status across multiple clusters with a hub-and-spoke architecture. It provides real-time presence information with low latency through intelligent caching and NATS-based persistence.

## 2. System Architecture

### 2.1 High-Level Architecture

```
                    ┌─────────────────┐
                    │   Center Node   │
                    │  (NATS Server)  │
                    │   KV Store      │
                    │   Memory Cache  │
                    └─────────┬───────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
    ┌───────▼───────┐ ┌───────▼───────┐ ┌───────▼───────┐
    │   Leaf Node   │ │   Leaf Node   │ │   Leaf Node   │
    │  Cluster A    │ │  Cluster B    │ │  Cluster C    │
    │ Memory Cache  │ │ Memory Cache  │ │ Memory Cache  │
    └───────────────┘ └───────────────┘ └───────────────┘
```

### 2.2 Component Architecture

```
┌─────────────────────────────────────────┐
│              API Gateway                │
│           (JWT Auth)                    │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│         Presence Service                │
│  ┌─────────────┐  ┌─────────────────┐   │
│  │   Handler   │  │  Memory Cache   │   │
│  │   Layer     │  │   (LRU/TTL)    │   │
│  └─────────────┘  └─────────────────┘   │
│  ┌─────────────┐  ┌─────────────────┐   │
│  │  Business   │  │   NATS Client   │   │
│  │   Logic     │  │    (JetStream)  │   │
│  └─────────────┘  └─────────────────┘   │
└─────────────┬───────────────────────────┘
              │
┌─────────────▼───────────────────────────┐
│         Embedded NATS                   │
│  ┌─────────────┐  ┌─────────────────┐   │
│  │ KV Storage  │  │   JetStream     │   │
│  │ (Presence)  │  │   (Events)      │   │
│  └─────────────┘  └─────────────────┘   │
└─────────────────────────────────────────┘
```

## 3. Data Models

### 3.1 Presence Status

```go
type PresenceStatus string

const (
    StatusOnline  PresenceStatus = "online"
    StatusAway    PresenceStatus = "away"
    StatusBusy    PresenceStatus = "busy"
    StatusOffline PresenceStatus = "offline"
)
```

### 3.2 Presence Data Structure

```go
type Presence struct {
    UserID       string         `json:"user_id"`
    Status       PresenceStatus `json:"status"`
    Message      string         `json:"message,omitempty"`
    LastSeen     time.Time      `json:"last_seen"`
    UpdatedAt    time.Time      `json:"updated_at"`
    NodeID       string         `json:"node_id"`
    TTL          time.Duration  `json:"ttl,omitempty"`
}

type PresenceResponse struct {
    Success bool                `json:"success"`
    Data    map[string]Presence `json:"data,omitempty"`
    Error   string              `json:"error,omitempty"`
}
```

## 4. API Specification

### 4.1 Base Path
All endpoints start with `/api/v2/presence`

### 4.2 Endpoints

#### 4.2.1 Get Single User Presence
```
GET /api/v2/presence/{user_id}
Authorization: Bearer <JWT_TOKEN>

Response:
{
    "success": true,
    "data": {
        "user123": {
            "user_id": "user123",
            "status": "online",
            "message": "Working on project",
            "last_seen": "2024-01-15T10:30:00Z",
            "updated_at": "2024-01-15T10:30:00Z",
            "node_id": "center-node-1"
        }
    }
}
```

#### 4.2.2 Get Multiple Users Presence
```
GET /api/v2/presence?users=user1,user2,user3
Authorization: Bearer <JWT_TOKEN>

Response:
{
    "success": true,
    "data": {
        "user1": {...},
        "user2": {...},
        "user3": {...}
    }
}
```

#### 4.2.3 Set User Presence
```
PUT /api/v2/presence/{user_id}
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
    "status": "busy",
    "message": "In a meeting",
    "ttl": 3600
}

Response:
{
    "success": true,
    "data": {
        "user123": {
            "user_id": "user123",
            "status": "busy",
            "message": "In a meeting",
            "last_seen": "2024-01-15T10:30:00Z",
            "updated_at": "2024-01-15T10:30:00Z",
            "node_id": "center-node-1"
        }
    }
}
```

#### 4.2.4 Batch Get Presence
```
POST /api/v2/presence/batch
Authorization: Bearer <JWT_TOKEN>
Content-Type: application/json

{
    "user_ids": ["user1", "user2", "user3", "user4"]
}

Response:
{
    "success": true,
    "data": {
        "user1": {...},
        "user2": {...},
        "user3": {...},
        "user4": {...}
    }
}
```

## 5. NATS Architecture

### 5.1 KV Store Structure

```
Bucket: "presence"
Key Pattern: "user:{user_id}"
Value: JSON-encoded Presence struct
TTL: Configurable per user (default: 1 hour)
```

### 5.2 JetStream Subjects

- `presence.update.{user_id}` - Presence updates
- `presence.expire.{user_id}` - Presence expiration events
- `presence.sync.{node_id}` - Node synchronization

### 5.3 NATS Configuration

```yaml
nats:
  center_node:
    port: 4222
    cluster_port: 6222
    jetstream:
      enabled: true
      max_memory: 1GB
      max_file: 10GB
  leaf_nodes:
    - remotes:
        - url: "nats://center-node:7422"
    - port: 4223
```

## 6. Memory Cache Design

### 6.1 Cache Strategy

- **Type**: Ristretto high-performance cache
- **Cost**: Memory-based cost accounting (configurable max cost)
- **Admission**: TinyLFU admission policy for optimal hit ratios
- **Eviction**: Automatic based on cost and access patterns
- **Concurrency**: Lock-free operations for maximum throughput

### 6.2 Cache Invalidation

1. **KV Watcher**: Monitor NATS KV changes
2. **TTL Expiry**: Remove expired entries
3. **Manual Invalidation**: On presence updates
4. **Periodic Cleanup**: Every 30 seconds

### 6.3 Cache Implementation

```go
type MemoryCache interface {
    Get(userID string) (Presence, bool)
    Set(userID string, presence Presence, cost int64)
    Delete(userID string)
    GetMultiple(userIDs []string) map[string]Presence
    Clear()
    Metrics() CacheMetrics
}

type CacheMetrics struct {
    Hits       uint64
    Misses     uint64
    KeysAdded  uint64
    KeysEvicted uint64
    CostAdded  uint64
    CostEvicted uint64
}
```

## 7. Authentication & Authorization

### 7.1 JWT Structure

```json
{
    "sub": "user_id",
    "iat": 1642291200,
    "exp": 1642377600,
    "iss": "presence-service"
}
```

User with validated JWT token can write his own status. Read permission does not require authentication.

## 8. Deployment Architecture

### 8.1 Center Node

- **Role**: Primary NATS server with KV store
- **Responsibilities**:
  - Authoritative data storage
  - Cluster coordination
  - Leaf node management
  - Data persistence

### 8.2 Leaf Nodes

- **Role**: Regional presence service instances
- **Responsibilities**:
  - Local API handling
  - Memory caching
  - NATS leaf connection
  - Local user presence aggregation

### 8.3 Service Discovery

```yaml
center_node:
  host: center.presence.internal
  nats_port: 4222
  leaf_port: 7422
  
leaf_nodes:
  - region: us-east-1
    host: leaf-us-east.presence.internal
  - region: eu-west-1
    host: leaf-eu-west.presence.internal
```

## 9. Configuration

### 9.1 Service Configuration

```yaml
service:
  name: presence-service
  version: v2
  port: 8080
  node_type: center  # or leaf
  node_id: center-node-1

nats:
  embedded: true
  data_dir: ./nats-data
  jetstream:
    max_memory: 1GB
    max_file: 10GB
  kv:
    bucket: presence
    ttl: 3600s

cache:
  type: ristretto
  max_cost: 1000000     # Maximum memory cost (bytes)
  num_counters: 100000  # Number of counters for TinyLFU
  buffer_items: 64      # Buffer size for async operations
  metrics: true         # Enable cache metrics

auth:
  jwt_secret: ${JWT_SECRET}
  jwt_issuer: presence-service
  jwt_ttl: 24h

logging:
  level: info
  format: json
```

## 10. Performance Considerations

### 10.1 Scalability Targets

- **Concurrent Users**: 100K per leaf node
- **Requests/Second**: 10K per leaf node
- **Latency**: < 10ms for cached reads
- **Availability**: 99.9%

### 10.2 Optimization Strategies

1. **Memory Cache**: Hot data in memory
2. **Connection Pooling**: NATS connection reuse
3. **Batch Operations**: Multiple user queries
4. **Compression**: NATS message compression
5. **Circuit Breaker**: Fault tolerance

## 11. Monitoring & Health Checks

### 11.1 Health Endpoints

- `GET /health` - Service health
- `GET /health/nats` - NATS connectivity
- `GET /health/cache` - Cache statistics
- `GET /metrics` - Prometheus metrics

### 11.2 Key Metrics

- Presence updates/second
- Cache hit ratio
- NATS message latency
- Active connections
- Memory usage

## 12. Error Handling

### 12.1 Error Codes

- `400` - Bad Request (invalid status, malformed JSON)
- `401` - Unauthorized (invalid/expired JWT)
- `403` - Forbidden (insufficient permissions)
- `404` - Not Found (user not found)
- `429` - Too Many Requests (rate limiting)
- `500` - Internal Server Error
- `503` - Service Unavailable (NATS down)

### 12.2 Fallback Strategy

1. **Cache First**: Always check memory cache
2. **KV Fallback**: Query NATS KV if cache miss
3. **Default Status**: Return "offline" if all fails
4. **Circuit Breaker**: Prevent cascade failures

## 13. Security Considerations

### 13.1 Data Protection

- JWT token validation
- Rate limiting per user
- Input sanitization
- HTTPS enforcement
- NATS TLS encryption

### 13.2 Privacy

- Presence data encryption at rest
- Audit logging for presence changes
- Data retention policies
- GDPR compliance considerations