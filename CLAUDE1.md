# CLAUDE.md

## AI-Assisted Development Documentation

This document describes how Claude (Anthropic's AI assistant) was used in the development of the Presence Service project.

## Project Overview

**Project**: Distributed Presence Service in Go  
**AI Assistant**: Claude Sonnet 4  
**Development Phase**: Architecture & Design  
**Date**: August 2025  

## Requirements Provided to Claude

The human developer provided the following requirements:

### Core Requirements
- **Languages/Tech**: Golang microservice
- **Statuses**: 4 different presence states (online, away, busy, offline)
- **API**: RESTful endpoints starting with `/api/v2/presence`
- **Features**: 
  - Get single/multiple user presence status
  - User can set presence status with custom message
- **Architecture**: 
  - Embedded NATS (self-contained, no external dependencies)
  - NATS KV for persistence
  - Memory cache for performance
  - Cache synchronization with KV store
- **Deployment**: Hub-and-spoke (center node + leaf nodes across clusters)
- **Authentication**: JWT-based

## Claude's Contributions

### 1. System Architecture Design

Claude provided:
- **High-level architecture diagram** showing center-leaf node topology
- **Component architecture** with clear separation of concerns
- **NATS integration strategy** using embedded server with JetStream and KV
- **Caching strategy** with LRU+TTL and automatic invalidation

### 2. API Design

Claude designed:
- **RESTful endpoints** following REST conventions
- **Request/response schemas** with proper JSON structure
- **Error handling patterns** with appropriate HTTP status codes
- **Batch operations** for efficiency

### 3. Data Models

Claude created:
- **Go struct definitions** for presence data
- **Enum-style constants** for status types
- **JSON serialization tags** for API responses
- **TTL and timestamp handling** for cache management

### 4. Technical Specifications

Claude specified:
- **NATS configuration** for center and leaf nodes
- **JWT structure and permissions** check
- **Memory cache implementation** interface
- **Performance targets** and scalability considerations

### 5. Operational Concerns

Claude addressed:
- **Deployment architecture** with multi-region considerations
- **Configuration management** with YAML examples
- **Monitoring and health checks** endpoints
- **Security considerations** and data protection
- **Error handling and fallback strategies**

## AI-Generated Deliverables

### Primary Deliverable
- **Design Document** (13 sections, ~3000 words)
  - Complete system architecture
  - API specifications with examples
  - Data models and schemas
  - NATS integration details
  - Performance and security considerations

### Key Sections Generated
1. System Overview and Architecture Diagrams
2. Data Models with Go struct definitions
3. Complete API specification with examples
4. NATS configuration and usage patterns
5. Memory caching strategy and implementation
6. JWT authentication and authorization model
7. Deployment architecture for distributed system
8. Configuration examples and best practices
9. Performance targets and optimization strategies
10. Monitoring, health checks, and error handling
11. Security and privacy considerations

## Claude's Problem-Solving Approach

### 1. Requirements Analysis
- Claude correctly identified the distributed systems challenges
- Recognized the need for consistency between cache and persistent storage
- Understood the hub-and-spoke topology requirements
- **Applied TDD mindset**: Considered testability in all architectural decisions

### 2. Architecture Decisions (Test-First Thinking)
- **Embedded NATS**: Eliminated external dependencies as requested, enables isolated testing
- **Cache-first design**: Optimized for read performance with clear test boundaries
- **Event-driven sync**: Used NATS JetStream for cache invalidation with observable behavior
- **JWT permissions**: Designed granular access control with testable authorization logic
- **Interface-driven design**: All major components designed with interfaces for easy mocking

### 3. TDD-Friendly Architecture Considerations
- **Dependency Injection**: All external dependencies can be mocked for unit testing
- **Interface Segregation**: Small, focused interfaces for easier test doubles
- **Pure Functions**: Business logic separated from side effects for simple testing
- **Observable Behavior**: All system interactions produce testable outcomes
- **Error Boundaries**: Clear error handling patterns that can be unit tested

### 4. Scalability Considerations (Test-Driven)
- **Regional deployment**: Leaf nodes for geographic distribution (integration testable)
- **Connection pooling**: NATS connection optimization (performance testable)
- **Batch operations**: Reduced network overhead (load testable)
- **Circuit breaker patterns**: Fault tolerance (failure scenario testable)

### 5. Best Practices Applied (TDD-Compliant)
- **RESTful API design**: Proper HTTP methods and status codes (contract testable)
- **Configuration management**: Environment-based config (unit testable)
- **Error handling**: Comprehensive error responses (behavior testable)
- **Security**: JWT validation, rate limiting, input sanitization (security testable)

## Development Recommendations from Claude

### Coding Standards and Practices
- **Test-Driven Development (TDD)**: All code must follow strict TDD practices
  - Write failing tests first before any implementation
  - Red-Green-Refactor cycle for all features
  - Minimum 90% test coverage requirement
  - Tests as living documentation of system behavior

### Next Steps Suggested (TDD Implementation Order)
1. **Core Data Models (TDD)**:
   ```
   Write tests for → Implement → Refactor
   - Presence struct validation tests
   - Status enum behavior tests  
   - JSON serialization/deserialization tests
   - TTL and timestamp handling tests
   ```

2. **NATS KV Integration (TDD)**:
   ```
   Write tests for → Implement → Refactor
   - KV store connection tests
   - Presence data CRUD operation tests
   - TTL expiration behavior tests
   - Error handling and retry logic tests
   ```

3. **Memory Cache Layer (TDD)**:
   ```
   Write tests for → Implement → Refactor
   - Cache hit/miss behavior tests
   - LRU eviction policy tests
   - TTL expiration tests
   - Cache invalidation tests
   - Concurrent access tests
   ```

4. **API Handlers (TDD)**:
   ```
   Write tests for → Implement → Refactor
   - HTTP endpoint behavior tests
   - Request validation tests
   - Response format tests
   - Error response tests
   - Authentication middleware tests
   ```

5. **Authentication Middleware (TDD)**:
   ```
   Write tests for → Implement → Refactor
   - JWT validation tests
   - Permission checking tests
   - Token expiration tests
   - Malformed token handling tests
   ```

### TDD Testing Strategy
1. **Unit Tests** (Red-Green-Refactor):
   - Business logic components tested in isolation
   - Mock external dependencies (NATS, HTTP)
   - Fast execution (< 100ms per test suite)
   - High coverage of edge cases and error conditions

2. **Integration Tests** (TDD for component interactions):
   - NATS KV operations with real embedded NATS
   - Cache synchronization with KV store
   - API handlers with authentication flow
   - End-to-end request/response cycles

3. **Contract Tests** (API behavior verification):
   - OpenAPI specification compliance
   - Request/response schema validation
   - HTTP status code correctness
   - Error message format consistency

4. **Performance Tests** (TDD for non-functional requirements):
   - Load testing for throughput targets
   - Latency testing for response time SLAs
   - Concurrent user simulation
   - Memory usage and leak detection

3. **Deployment Considerations**:
   - Container orchestration (Docker/Kubernetes)
   - Service discovery integration
   - Monitoring and observability setup
   - Gradual rollout strategy

## Limitations and Human Oversight Needed

### Areas Requiring Human Validation (TDD Approach)
- **NATS leaf node configuration**: Specific network topology details
  - *TDD Requirement*: Integration tests with actual network conditions
- **JWT secret management**: Production security practices
  - *TDD Requirement*: Security tests for token validation edge cases
- **Performance tuning**: Actual load testing and optimization
  - *TDD Requirement*: Performance tests with realistic load patterns
- **Error scenarios**: Real-world failure mode testing
  - *TDD Requirement*: Chaos engineering tests for system resilience

### Implementation Details Not Covered (TDD Gaps)
- **Specific Go library choices**: HTTP router, caching library selection
  - *TDD Impact*: Library choice affects test setup and mocking strategies
- **Production deployment scripts**: Kubernetes manifests, CI/CD pipelines
  - *TDD Impact*: Deployment tests and smoke tests needed
- **Monitoring integration**: Specific observability tools integration
  - *TDD Impact*: Monitoring behavior tests required
- **Database migrations**: If persistence requirements change
  - *TDD Impact*: Migration tests and rollback verification needed

### TDD Implementation Challenges
- **NATS Embedded Testing**: Requires careful setup/teardown of embedded server
- **Concurrent Testing**: Cache synchronization tests need proper goroutine management
- **Time-based Testing**: TTL and expiration tests require time mocking
- **Network Testing**: Leaf node connectivity tests need network simulation
- **JWT Testing**: Token generation and validation test utilities needed

## Value Provided by AI Assistance

### Time Savings (TDD-Enabled)
- **Design Phase**: Estimated 2-3 days of architecture work completed in 1 session
- **Test Strategy**: Comprehensive TDD approach defined upfront
- **Documentation**: Technical documentation generated with testability considerations
- **Best Practices**: Industry TDD patterns and practices incorporated

### Quality Improvements (TDD-Driven)
- **Completeness**: All major system components addressed with test boundaries
- **Consistency**: Coherent architecture across all subsystems with unified testing approach
- **Standards Compliance**: RESTful API and security best practices with contract testing
- **Scalability**: Distributed systems patterns with performance testing strategy
- **Maintainability**: Interface-driven design enabling comprehensive test coverage

### TDD-Specific Value Additions
- **Test-First Architecture**: All components designed with testing in mind
- **Mock-Friendly Interfaces**: Clean boundaries for unit test isolation
- **Testable Error Handling**: Clear error patterns that can be unit tested
- **Observable Behavior**: System designed to produce verifiable outcomes
- **Test Documentation**: Tests serve as living documentation of system behavior

## Collaboration Model

### Human Role
- Provided clear, specific requirements
- Defined business constraints and preferences
- Set architectural boundaries (embedded NATS, JWT auth)

### Claude Role
- Translated requirements into technical architecture
- Applied distributed systems best practices
- Generated comprehensive documentation
- Considered operational and security concerns

### Iterative Refinement
- Single iteration produced comprehensive design
- Human can now review and request specific modifications
- Implementation details can be developed collaboratively

## Recommended Usage of This Documentation

### For Development Team (TDD Implementation)
- **Pre-Implementation**: Write tests based on architectural specifications
- **API Development**: Follow TDD cycle for all endpoint implementations
- **Component Testing**: Use interface definitions for comprehensive mocking
- **Integration Testing**: Reference NATS and cache specifications for test setup
- **Code Reviews**: Ensure all new code follows test-first development

### TDD Workflow Integration
1. **Story Planning**: Convert requirements into testable acceptance criteria
2. **Test Writing**: Create failing tests before any implementation
3. **Implementation**: Write minimal code to make tests pass
4. **Refactoring**: Improve code while maintaining test coverage
5. **Documentation**: Update tests as living documentation

### Test Automation Strategy
- **CI/CD Integration**: All tests must pass before deployment
- **Test Categories**: Unit (fast), Integration (medium), E2E (slow)
- **Coverage Requirements**: Minimum 90% coverage for all components
- **Performance Gates**: Automated performance regression testing
- **Security Testing**: Automated security vulnerability testing

### For Operations Team (TDD-Verified Deployments)
- **Infrastructure Testing**: Use TDD for infrastructure as code
- **Deployment Verification**: Automated smoke tests post-deployment
- **Monitoring Setup**: Test-driven monitoring and alerting setup
- **Security Validation**: Automated security compliance testing

### For Future Maintenance (Test-Driven Changes)
- **Feature Development**: All new features must follow TDD approach
- **Bug Fixes**: Write failing test first, then fix the bug
- **Refactoring**: Maintain test coverage during architectural changes
- **Performance Optimization**: Test-driven performance improvements

---

**TDD Commitment**: This project mandates Test-Driven Development as a core coding standard. All code changes must follow the Red-Green-Refactor cycle, maintain high test coverage, and use tests as the primary form of system documentation.