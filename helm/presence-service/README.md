# presence-service Helm Chart

This chart deploys the Presence Service (hub-and-spoke, NATS + cache) to Kubernetes.

## Health Probes

The service exposes dedicated health endpoints:
- Liveness: `/health/liveness` (process is up)
- Readiness: `/health/readiness` (dependencies, e.g., NATS KV, are ready)

Default probe settings (values.yaml):
```
deployment:
  livenessProbe:
    httpGet:
      path: /health/liveness
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 15
    timeoutSeconds: 3
    failureThreshold: 3

  readinessProbe:
    httpGet:
      path: /health/readiness
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 15
    timeoutSeconds: 3
    failureThreshold: 3
```
Adjust as needed per environment.

## CORS Configuration

CORS is configurable via environment variables that the chart passes to the container through `.Values.env`:
```
env:
  CORS_ENABLED: true
  CORS_ALLOWED_ORIGINS: "*"                     # For production, set to your exact frontend origin
  CORS_ALLOWED_METHODS: "GET,POST,PUT,DELETE,OPTIONS"
  CORS_ALLOWED_HEADERS: "Authorization,Content-Type"
  CORS_ALLOW_CREDENTIALS: false                 # Do not combine '*' with credentials=true
  CORS_MAX_AGE: 600
```

Example override for production:
```
helm upgrade --install presence \
  ./helm/presence-service \
  --set env.CORS_ALLOWED_ORIGINS="https://app.example.com" \
  --set env.CORS_ALLOW_CREDENTIALS=true
```

## Ports and Service

The chart exposes:
- HTTP service on port 8080
- For center nodes only: NATS ports 4222 (client), 7422 (leaf), 6222 (cluster)

These map to `k8sService.ports` in `values.yaml`. Hyphenated map keys are handled correctly in templates.

## Secrets

Provide the JWT secret via the chart's Secret template (see `templates/secret.yaml`). By default, the deployment references secret key `jwt-secret`.

## Notes

- If you add a ConfigMap template later, you can re-introduce a config checksum annotation in the Deployment to trigger rolling updates on config changes.
- For Prometheus scraping, enable `serviceMonitor.enabled` and set labels as needed.
