# Deployment Guide

## Hub-and-Spoke Architecture

The presence service supports a hub-and-spoke deployment model with:
- **Center Node**: Authoritative data storage with JetStream and KV store
- **Leaf Nodes**: Regional service instances that connect to the center for data persistence

## Center Node Deployment

Configure as a center node using environment variables:

```bash
export NODE_TYPE=center
export NODE_ID=center-node-1
export SERVICE_PORT=8080
export NATS_LEAF_PORT=7422      # Accept leaf connections
export NATS_CLUSTER_PORT=6222   # For center clustering (optional)
export JWT_SECRET=your-secret-key
```

Run the center node:
```bash
./presence-service
```

## Leaf Node Deployment

Configure as a leaf node:

```bash
export NODE_TYPE=leaf
export NODE_ID=leaf-node-us-east-1
export SERVICE_PORT=8080
export NATS_CENTER_URL=nats://center-node:4222  # Connect to center
export JWT_SECRET=your-secret-key               # Same as center
```

Run the leaf node:
```bash
./presence-service
```

## Configuration Files

Use the provided configuration examples:
- `config.center-node.env` - Center node configuration
- `config.leaf-node.env` - Leaf node configuration

## Network Requirements

- **Center Node**: Must be accessible by leaf nodes on port 4222 (NATS) and 7422 (leaf connections)
- **Leaf Nodes**: Must be able to connect to center node
- **Clients**: Can connect to any node (center or leaf) for API access

## Data Flow

1. **Reads**: Served from Ristretto cache first (high-performance, TinyLFU admission), then center KV store if cache miss
2. **Writes**: Stored in center KV store and cached locally with automatic cost-based eviction
3. **Synchronization**: NATS ensures data consistency across all nodes
4. **Cache Management**: Ristretto automatically handles admission, eviction, and provides detailed metrics

## Scaling

- Deploy multiple leaf nodes in different regions for low latency
- Center node handles all persistence and coordination
- Each leaf node can serve 100K+ concurrent users independently
- Load balance client requests across leaf nodes for maximum throughput