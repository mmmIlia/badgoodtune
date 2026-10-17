SHELL := /bin/sh
.DEFAULT_GOAL := help

GO_PACKAGES := ./cmd/... ./db/... ./internal/...
KUBECTL ?= kubectl
MINIKUBE ?= minikube
K8S_NAMESPACE ?= badgoodtune
K8S_DIR := deploy/kubernetes
BACKEND_IMAGE ?= badgoodtune-backend:hw2
FRONTEND_IMAGE ?= badgoodtune-frontend:hw2
K8S_ROLLOUT_TIMEOUT ?= 180s
K8S_JOB_TIMEOUT ?= 120s

.PHONY: help check generate generate-check fmt fmt-check vet test integration-db test-integration test-integration-running web-install web-generate web-check build db-up db-down compose-up compose-down compose-smoke security-check minikube-start minikube-status minikube-stop k8s-images k8s-load k8s-validate k8s-deploy k8s-up k8s-down k8s-status k8s-scale-up k8s-scale-down k8s-open k8s-delete

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Run code generation, formatting, static analysis, tests, frontend checks, and manifest validation
	$(MAKE) generate-check
	$(MAKE) fmt-check
	$(MAKE) vet
	$(MAKE) test
	$(MAKE) web-check
	$(MAKE) k8s-validate

generate: ## Regenerate Go and TypeScript code from OpenAPI and SQL
	cd api && go tool oapi-codegen --config oapi-codegen.yaml openapi.yaml
	go tool sqlc generate
	gofmt -w internal/transport/http/openapi internal/repository/postgres/sqlc
	cd web && npm run generate

generate-check: generate ## Verify that generated code is up to date
	git diff --exit-code -- internal/transport/http/openapi internal/repository/postgres/sqlc
	git diff --exit-code -- web/src/api/generated

fmt: ## Format Go and frontend sources
	gofmt -w $$(find cmd db internal -name '*.go' -type f)
	cd web && npm run format

fmt-check: ## Verify source formatting
	test -z "$$(gofmt -l $$(find cmd db internal -name '*.go' -type f))"
	cd web && npm run format:check

vet: ## Run Go static analysis
	go vet $(GO_PACKAGES)

test: ## Run Go unit and architecture tests
	go test $(GO_PACKAGES)

integration-db: db-up
	docker compose exec -T postgres sh -ec 'if [ "$$(psql -U "$$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '\''badgoodtune_test'\''")" != "1" ]; then createdb -U "$$POSTGRES_USER" -O "$$POSTGRES_USER" badgoodtune_test; fi'

test-integration: integration-db ## Run PostgreSQL integration tests
	$(MAKE) test-integration-running

test-integration-running:
	TEST_DATABASE_URL="postgres://badgoodtune:badgoodtune@localhost:$${POSTGRES_PORT:-54329}/badgoodtune_test?sslmode=disable" go test -tags=integration -count=1 ./internal/repository/postgres

web-install: ## Install locked frontend dependencies
	cd web && npm ci

web-generate: ## Regenerate the TypeScript API client
	cd web && npm run generate

web-check: ## Run frontend formatting, lint, typecheck, tests, and build
	cd web && npm run check

build: ## Build backend binaries into bin/
	mkdir -p bin
	go build -trimpath -o bin/server ./cmd/server
	go build -trimpath -o bin/migrate ./cmd/migrate

db-up: ## Start the Compose PostgreSQL dependency
	docker compose up -d --wait postgres

db-down: ## Stop Compose services without deleting database data
	docker compose down

compose-up: ## Build and start the full application with Compose
	docker compose up --build -d --wait --remove-orphans

compose-down: ## Stop the Compose application without deleting database data
	docker compose down --remove-orphans

compose-smoke: compose-up ## Start Compose and run the end-to-end CRUD smoke test
	./scripts/smoke.sh

security-check: ## Audit reachable Go and high-severity npm vulnerabilities
	go tool govulncheck $(GO_PACKAGES)
	cd web && npm audit --audit-level=high

minikube-start: ## Start the local single-node Kubernetes cluster
	$(MINIKUBE) start --driver=docker --container-runtime=docker --cpus=4 --memory=4096mb

minikube-status: ## Show Minikube status
	$(MINIKUBE) status

minikube-stop: ## Stop Minikube without deleting the cluster
	$(MINIKUBE) stop

k8s-images: ## Build backend and frontend images for Kubernetes
	docker build -t $(BACKEND_IMAGE) .
	docker build -t $(FRONTEND_IMAGE) -f web/Dockerfile .

k8s-load: ## Load locally built images into Minikube
	$(MINIKUBE) image load --daemon --overwrite $(BACKEND_IMAGE)
	$(MINIKUBE) image load --daemon --overwrite $(FRONTEND_IMAGE)

k8s-validate: ## Validate Kubernetes manifests against strict schemas
	kubeconform -strict -summary $(K8S_DIR)

k8s-deploy: ## Apply manifests in dependency order and wait for readiness
	$(KUBECTL) apply -f $(K8S_DIR)/00-namespace
	$(KUBECTL) apply -f $(K8S_DIR)/10-configuration
	$(KUBECTL) apply -f $(K8S_DIR)/20-database
	$(KUBECTL) rollout status statefulset/postgres -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(KUBECTL) delete job/badgoodtune-migration -n $(K8S_NAMESPACE) --ignore-not-found
	$(KUBECTL) apply -f $(K8S_DIR)/30-migration
	$(KUBECTL) wait --for=condition=complete job/badgoodtune-migration -n $(K8S_NAMESPACE) --timeout=$(K8S_JOB_TIMEOUT)
	$(KUBECTL) apply -f $(K8S_DIR)/40-services
	$(KUBECTL) apply -f $(K8S_DIR)/50-application
	$(KUBECTL) rollout status deployment/badgoodtune-backend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(KUBECTL) rollout status deployment/badgoodtune-frontend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)

k8s-up: ## Start Minikube, build and load images, and deploy the application
	$(MAKE) minikube-start
	$(MAKE) k8s-images
	$(MAKE) k8s-load
	$(MAKE) k8s-deploy

k8s-down: ## Delete the application and its data, then stop Minikube
	$(MAKE) k8s-delete
	$(MAKE) minikube-stop

k8s-status: ## Show application workloads, services, migration, and storage
	$(KUBECTL) get pods,services,deployments,statefulsets,jobs,persistentvolumeclaims -n $(K8S_NAMESPACE)

k8s-scale-up: ## Scale backend to 3 and frontend to 2 replicas
	$(KUBECTL) scale deployment/badgoodtune-backend --replicas=3 -n $(K8S_NAMESPACE)
	$(KUBECTL) scale deployment/badgoodtune-frontend --replicas=2 -n $(K8S_NAMESPACE)
	$(KUBECTL) rollout status deployment/badgoodtune-backend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(KUBECTL) rollout status deployment/badgoodtune-frontend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)

k8s-scale-down: ## Scale backend and frontend back to 1 replica
	$(KUBECTL) scale deployment/badgoodtune-backend --replicas=1 -n $(K8S_NAMESPACE)
	$(KUBECTL) scale deployment/badgoodtune-frontend --replicas=1 -n $(K8S_NAMESPACE)

k8s-open: ## Open the frontend through its Minikube Service
	$(MINIKUBE) service badgoodtune-frontend -n $(K8S_NAMESPACE)

k8s-delete: ## Delete the application Namespace and its persistent data
	$(KUBECTL) delete namespace $(K8S_NAMESPACE) --ignore-not-found
