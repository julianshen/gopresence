# Kubernetes Deployment Guide

This guide covers deploying the presence service on Kubernetes using Docker and Helm charts.

## Prerequisites

- Kubernetes cluster (1.20+)
- Helm 3.x
- Docker (for building images)
- kubectl configured for your cluster

## Building the Docker Image

### 1. Build the Image

```bash
# Build the Docker image
docker build -t presence-service:v2.0.0 .

# Tag for your registry (replace with your registry)
docker tag presence-service:v2.0.0 your-registry.com/presence-service:v2.0.0

# Push to registry
docker push your-registry.com/presence-service:v2.0.0
```

### 2. Multi-Architecture Build (Optional)

```bash
# Create and use a new builder
docker buildx create --use

# Build for multiple architectures
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-registry.com/presence-service:v2.0.0 \
  --push .
```

## Deployment Architecture

### Hub-and-Spoke Model

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│   Leaf Node     │       │  Center Node    │       │   Leaf Node     │
│   (US-East)     │◄─────►│   (Primary)     │◄─────►│   (EU-West)     │
│                 │       │                 │       │                 │
│ - Ristretto     │       │ - Ristretto     │       │ - Ristretto     │
│ - NATS Leaf     │       │ - NATS Server   │       │ - NATS Leaf     │
│ - HTTP API      │       │ - JetStream/KV  │       │ - HTTP API      │
└─────────────────┘       └─────────────────┘       └─────────────────┘
```

## Helm Deployment

### 1. Deploy Center Node

First, create a namespace:

```bash
kubectl create namespace presence-system
```

Deploy the center node:

```bash
# Create JWT secret
kubectl create secret generic presence-jwt-secret \
  --from-literal=jwt-secret="your-super-secret-jwt-key-change-in-production" \
  -n presence-system

# Deploy center node
helm install presence-center ./helm/presence-service \
  -f ./helm/presence-service/values-center.yaml \
  --set image.repository=your-registry.com/presence-service \
  --set image.tag=v2.0.0 \
  --set auth.jwtSecret="your-super-secret-jwt-key-change-in-production" \
  -n presence-system
```

Wait for center node to be ready:

```bash
kubectl wait --for=condition=available --timeout=300s deployment/presence-center -n presence-system
```

### 2. Deploy Leaf Nodes

Get the center node service name:

```bash
kubectl get svc -n presence-system
```

Deploy leaf nodes (update centerUrl as needed):

```bash
# Deploy US-East leaf node
helm install presence-leaf-us-east ./helm/presence-service \
  -f ./helm/presence-service/values-leaf.yaml \
  --set image.repository=your-registry.com/presence-service \
  --set image.tag=v2.0.0 \
  --set service.nodeId="leaf-node-us-east-1" \
  --set nats.centerUrl="nats://presence-center:4222" \
  --set auth.jwtSecret="your-super-secret-jwt-key-change-in-production" \
  -n presence-system

# Deploy EU-West leaf node  
helm install presence-leaf-eu-west ./helm/presence-service \
  -f ./helm/presence-service/values-leaf.yaml \
  --set image.repository=your-registry.com/presence-service \
  --set image.tag=v2.0.0 \
  --set service.nodeId="leaf-node-eu-west-1" \
  --set nats.centerUrl="nats://presence-center:4222" \
  --set auth.jwtSecret="your-super-secret-jwt-key-change-in-production" \
  -n presence-system
```

### 3. Verify Deployment

```bash
# Check all pods
kubectl get pods -n presence-system

# Check services
kubectl get svc -n presence-system

# Check logs
kubectl logs -l app.kubernetes.io/name=presence-service -n presence-system

# Test health endpoints
kubectl port-forward svc/presence-center 8080:8080 -n presence-system &
curl http://localhost:8080/health
```

## Configuration Options

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `NODE_TYPE` | Node type: `center` or `leaf` | `center` |
| `NODE_ID` | Unique node identifier | `node-1` |
| `NATS_CENTER_URL` | Center NATS URL (leaf nodes) | - |
| `CACHE_MAX_COST` | Ristretto max memory cost | `1000000` |
| `JWT_SECRET` | JWT signing secret | **Required** |

### Helm Values

Key configuration options in `values.yaml`:

```yaml
# Service configuration
service:
  nodeType: center  # or leaf
  nodeId: "center-node-1"

# Cache configuration
cache:
  maxCost: 2000000
  numCounters: 200000
  metrics: true

# Resources
deployment:
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
```

## Scaling and High Availability

### Horizontal Pod Autoscaling

Enable HPA in your values file:

```yaml
hpa:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
```

### Pod Disruption Budget

Ensure availability during updates:

```yaml
pdb:
  enabled: true
  minAvailable: 1
```

### Multi-Zone Deployment

Deploy across availability zones:

```yaml
deployment:
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values: ["presence-service"]
        topologyKey: topology.kubernetes.io/zone
```

## Monitoring

### Prometheus Integration

Enable ServiceMonitor for Prometheus:

```yaml
serviceMonitor:
  enabled: true
  labels:
    prometheus: kube-prometheus
  interval: 30s
  path: /metrics
```

### Grafana Dashboard

Key metrics to monitor:
- Cache hit/miss ratio
- Response latency
- NATS connection status
- Memory usage
- CPU utilization

## Troubleshooting

### Common Issues

1. **JWT Secret Missing**
   ```bash
   kubectl logs deployment/presence-center -n presence-system
   # Error: JWT_SECRET environment variable is required
   ```
   Solution: Ensure JWT secret is created and referenced correctly.

2. **Leaf Node Can't Connect to Center**
   ```bash
   kubectl logs deployment/presence-leaf-us-east -n presence-system
   # Error: failed to access KV bucket: context deadline exceeded
   ```
   Solution: Check NATS_CENTER_URL configuration and network policies.

3. **NATS Server Won't Start**
   ```bash
   # Error: server failed to start within 15s
   ```
   Solution: Check PVC mounting and data directory permissions.

### Debugging Commands

```bash
# Check pod status
kubectl describe pod <pod-name> -n presence-system

# Check logs
kubectl logs <pod-name> -n presence-system --follow

# Port forward for testing
kubectl port-forward svc/presence-center 8080:8080 -n presence-system

# Execute into pod
kubectl exec -it <pod-name> -n presence-system -- sh

# Check NATS connectivity
kubectl exec -it <center-pod> -n presence-system -- nats server info
```

## Security Considerations

### Network Policies

Create network policies to restrict traffic:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: presence-service-netpol
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: presence-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 8080
```

### RBAC

Create appropriate service accounts and roles:

```bash
kubectl create serviceaccount presence-service -n presence-system
kubectl create clusterrole presence-service --verb=get,list,watch --resource=configmaps,secrets
kubectl create clusterrolebinding presence-service --clusterrole=presence-service --serviceaccount=presence-system:presence-service
```

### Secret Management

Use external secret management:

```yaml
# Example with External Secrets Operator
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
spec:
  provider:
    vault:
      server: "https://vault.example.com"
      path: "secret"
      auth:
        kubernetes:
          mountPath: "kubernetes"
          role: "presence-service"
```

## Production Checklist

- [ ] JWT secret properly configured and secured
- [ ] Resource limits and requests set appropriately
- [ ] Persistent storage configured for center nodes
- [ ] Network policies defined
- [ ] Monitoring and alerting configured
- [ ] Backup strategy for NATS data
- [ ] Load balancer configured for external access
- [ ] SSL/TLS certificates configured
- [ ] Health checks and readiness probes configured
- [ ] Log aggregation configured