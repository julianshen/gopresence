# Makefile for Presence Service

# Variables
DOCKER_REGISTRY ?= your-registry.com
IMAGE_NAME ?= presence-service
VERSION ?= v2.0.0
NAMESPACE ?= presence-system
JWT_SECRET ?= change-this-in-production-please

# Build targets
.PHONY: build test docker-build docker-push helm-install-center helm-install-leaf clean

# Build the Go binary
build:
	@mkdir -p build
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o build/presence-service ./cmd/presence-service

# Run service benchmarks (in-memory KV fake)
bench-service:
	go test -bench=. -benchmem ./internal/service -run ^$

# Run tests
test:
	go test ./... -v

# Run tests with coverage
test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

# Enforce minimum total coverage of 75%
coverage-check:
	go test ./... -coverprofile=coverage.out
	@cov=$(go tool cover -func=coverage.out | tail -n 1 | awk '{print $NF}' | sed 's/%//'); \
	if [ $(printf '%.0f' $cov) -lt 75 ]; then \
		echo "Coverage $cov% is below required 75%"; \
		exit 1; \
	else \
		echo "Coverage $cov% meets requirement (>=75%)"; \
	fi


# Run tests, enforce >=75% coverage, and generate HTML report
test-coverage-enforced:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -n 1 | awk '{print $NF}' | sed 's/%//' | awk '{if ($1 < 75) exit 1}'
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage OK (>=75%). Report at coverage.html"

# Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME):$(VERSION) .
	docker tag $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest

# Build multi-arch Docker image
docker-buildx:
	docker buildx build --platform linux/amd64,linux/arm64 \
		-t $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		-t $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest \
		--push .

# Push Docker image to registry
docker-push: docker-build
	docker tag $(IMAGE_NAME):$(VERSION) $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker tag $(IMAGE_NAME):latest $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):$(VERSION)
	docker push $(DOCKER_REGISTRY)/$(IMAGE_NAME):latest

# Start local development with docker-compose
dev-up:
	docker-compose up --build -d

# Stop local development
dev-down:
	docker-compose down -v

# View logs from all containers
dev-logs:
	docker-compose logs -f

# Kubernetes deployment targets

# Create namespace
k8s-namespace:
	kubectl create namespace $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

# Create JWT secret
k8s-secret:
	kubectl create secret generic presence-jwt-secret \
		--from-literal=jwt-secret="$(JWT_SECRET)" \
		-n $(NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -

# Install center node
helm-install-center: k8s-namespace k8s-secret
	helm upgrade --install presence-center ./helm/presence-service \
		-f ./helm/presence-service/values-center.yaml \
		--set image.repository=$(DOCKER_REGISTRY)/$(IMAGE_NAME) \
		--set image.tag=$(VERSION) \
		--set auth.jwtSecret="$(JWT_SECRET)" \
		-n $(NAMESPACE)

# Install leaf node (US East)
helm-install-leaf-us: k8s-namespace k8s-secret
	helm upgrade --install presence-leaf-us-east ./helm/presence-service \
		-f ./helm/presence-service/values-leaf.yaml \
		--set image.repository=$(DOCKER_REGISTRY)/$(IMAGE_NAME) \
		--set image.tag=$(VERSION) \
		--set service.nodeId="leaf-node-us-east-1" \
		--set nats.centerUrl="nats://presence-center:4222" \
		--set auth.jwtSecret="$(JWT_SECRET)" \
		-n $(NAMESPACE)

# Install leaf node (EU West)
helm-install-leaf-eu: k8s-namespace k8s-secret
	helm upgrade --install presence-leaf-eu-west ./helm/presence-service \
		-f ./helm/presence-service/values-leaf.yaml \
		--set image.repository=$(DOCKER_REGISTRY)/$(IMAGE_NAME) \
		--set image.tag=$(VERSION) \
		--set service.nodeId="leaf-node-eu-west-1" \
		--set nats.centerUrl="nats://presence-center:4222" \
		--set auth.jwtSecret="$(JWT_SECRET)" \
		-n $(NAMESPACE)

# Install complete deployment (center + leaf nodes)
helm-install-all: helm-install-center
	@echo "Waiting for center node to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/presence-center -n $(NAMESPACE)
	$(MAKE) helm-install-leaf-us
	$(MAKE) helm-install-leaf-eu

# Uninstall all deployments
helm-uninstall:
	helm uninstall presence-leaf-eu-west -n $(NAMESPACE) || true
	helm uninstall presence-leaf-us-east -n $(NAMESPACE) || true
	helm uninstall presence-center -n $(NAMESPACE) || true

# Get deployment status
k8s-status:
	kubectl get pods,svc,pvc -n $(NAMESPACE)

# Get logs
k8s-logs:
	kubectl logs -l app.kubernetes.io/name=presence-service -n $(NAMESPACE) --tail=100 -f

# Port forward for testing
k8s-port-forward:
	kubectl port-forward svc/presence-center 8080:8080 -n $(NAMESPACE)

# Run health check
health-check:
	curl -f http://localhost:8080/health || exit 1

# Lint Helm charts
helm-lint:
	helm lint ./helm/presence-service

# Template Helm charts
helm-template:
	helm template presence-center ./helm/presence-service \
		-f ./helm/presence-service/values-center.yaml \
		--set image.repository=$(DOCKER_REGISTRY)/$(IMAGE_NAME) \
		--set image.tag=$(VERSION)

# Clean up
clean:
	rm -f presence-service
	rm -f coverage.out coverage.html
	docker rmi $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest 2>/dev/null || true
	docker-compose down -v --rmi all 2>/dev/null || true

# Help
help:
	@echo "Available targets:"
	@echo "  build                 - Build Go binary"
	@echo "  test                  - Run tests"
	@echo "  test-coverage         - Run tests with coverage"
	@echo "  coverage-check        - Run coverage and enforce >=85%"
	@echo "  test-coverage-enforced- Run tests with coverage and enforce >=85%"
	@echo "  docker-build          - Build Docker image"
	@echo "  docker-buildx         - Build multi-arch Docker image"
	@echo "  docker-push           - Push Docker image to registry"
	@echo "  dev-up                - Start local development environment"
	@echo "  dev-down              - Stop local development environment"
	@echo "  dev-logs              - View development logs"
	@echo "  helm-install-center   - Install center node on Kubernetes"
	@echo "  helm-install-leaf-us  - Install US leaf node on Kubernetes"
	@echo "  helm-install-leaf-eu  - Install EU leaf node on Kubernetes"
	@echo "  helm-install-all      - Install complete deployment"
	@echo "  helm-uninstall        - Uninstall all deployments"
	@echo "  k8s-status            - Get deployment status"
	@echo "  k8s-logs              - Get Kubernetes logs"
	@echo "  k8s-port-forward      - Port forward for testing"
	@echo "  health-check          - Run health check"
	@echo "  helm-lint             - Lint Helm charts"
	@echo "  helm-template         - Template Helm charts"
	@echo "  clean                 - Clean up build artifacts"
	@echo ""
	@echo "Variables:"
	@echo "  DOCKER_REGISTRY       - Docker registry (default: your-registry.com)"
	@echo "  IMAGE_NAME            - Image name (default: presence-service)"
	@echo "  VERSION               - Version tag (default: v2.0.0)"
	@echo "  NAMESPACE             - Kubernetes namespace (default: presence-system)"
	@echo "  JWT_SECRET            - JWT secret for authentication"