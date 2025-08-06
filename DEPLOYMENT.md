# Deployment Guide (Updated)

This document summarizes key updates to testing and performance tooling relevant for deployments.

## Test & Coverage Policy

- Minimum total test coverage is enforced at 75% via `make coverage-check`.
- CI skips certain NATS deep/integration tests that can be flaky in constrained environments. These tests remain in the repository for local or dedicated integration runs.

### Commands

```bash
# Run full test suite
make test

# Run with coverage and report
make test-coverage

# Enforce minimum coverage >= 75%
make coverage-check
```

## Benchmarks

Service-level benchmarks (using an in-memory KV fake to avoid I/O and variance) are provided.

```bash
# Run service benchmarks
make bench-service
```

Use benchmark results to validate performance regressions during CI/CD or pre-release checks.

## Notes

- For production-like performance validation of NATS operations (KV/JetStream), prefer running integration tests in a controlled environment with adequate timeouts and resources.
- If enabling skipped integration tests, consider gating with build tags and adjusting timeouts.
