# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a distributed Go microservice for managing user presence status across multiple clusters using a hub-and-spoke architecture. The service uses embedded NATS for messaging and persistence, with memory caching for performance.

## Development Commands

Since this project is in the design phase, no build/test commands exist yet. When implementation begins:
- `go mod init gopresence` - Initialize Go module  
- `go build` - Build the service
- `go test ./...` - Run all tests
- `go run main.go` - Run the service locally

## Architecture Overview

**Core Components:**
- **Center Node**: Primary NATS server with authoritative KV store
- **Leaf Nodes**: Regional service instances with local caching
- **API Layer**: RESTful endpoints at `/api/v2/presence`
- **Memory Cache**: Ristretto high-performance cache for low latency reads
- **NATS Integration**: Embedded NATS with JetStream and KV store

**Data Flow:**
1. API requests hit leaf nodes for low latency
2. Cache-first approach for reads
3. Updates propagate through NATS to center node
4. KV store provides persistence and cross-node sync

## Key Design Decisions

**Embedded NATS**: Self-contained deployment with no external dependencies. Hub-and-spoke (center node + leaf nodes across clusters). Server could be run as a center node or leaf node.
**JWT Authentication**: Token-based auth for presence updates (reads are public)
**Hub-and-Spoke**: Center node for coordination, leaf nodes for regional access
**Ristretto Cache**: High-performance, concurrent cache with automatic admission and eviction
**Cache Synchronization**: NATS watchers invalidate cache on KV changes

## API Endpoints

- `GET /api/v2/presence/{user_id}` - Get single user presence
- `GET /api/v2/presence?users=user1,user2` - Get multiple users
- `PUT /api/v2/presence/{user_id}` - Set user presence status
- `POST /api/v2/presence/batch` - Batch presence queries

## Data Models

**Presence Status**: online, away, busy, offline
**Presence Data**: UserID, Status, Message, LastSeen, UpdatedAt, NodeID, TTL

## Test-Driven Development Requirements

This project mandates strict TDD practices:
- Write failing tests before implementation
- Maintain 90%+ test coverage
- Use Red-Green-Refactor cycle
- Tests serve as living documentation
- All components designed with interfaces for testability

## Implementation Order (TDD)

1. **Core Data Models**: Presence struct, status validation, JSON serialization
2. **NATS KV Integration**: Connection, CRUD operations, TTL handling
3. **Ristretto Cache**: High-performance caching with admission control and metrics
4. **API Handlers**: HTTP endpoints, request validation, error handling
5. **Authentication**: JWT middleware, permission checks
6. **Integration**: End-to-end flows, performance testing

## Configuration Structure

```yaml
service:
  node_type: center|leaf
  port: 8080
nats:
  embedded: true
  data_dir: ./nats-data
cache:
  max_cost: 1000000      # Maximum memory cost (bytes)
  num_counters: 100000   # Number of counters for admission
  buffer_items: 64       # Buffer size for async operations
auth:
  jwt_secret: ${JWT_SECRET}
```

## Performance Targets

- 100K concurrent users per leaf node
- 10K requests/second per leaf node  
- <10ms latency for cached reads
- 99.9% availability

## Security Considerations

- JWT validation for presence updates
- Input sanitization and rate limiting
- HTTPS and NATS TLS encryption
- No authentication required for reads