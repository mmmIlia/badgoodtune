SHELL := /bin/sh
.DEFAULT_GOAL := help

GO_PACKAGES := ./cmd/... ./db/... ./internal/...
KUBECTL ?= kubectl
MINIKUBE ?= minikube
HELM ?= helm
K8S_NAMESPACE ?= badgoodtune
K8S_DIR := deploy/kubernetes
BACKEND_IMAGE ?= badgoodtune-backend:local
FRONTEND_IMAGE ?= badgoodtune-frontend:local
K8S_ROLLOUT_TIMEOUT ?= 180s
K8S_JOB_TIMEOUT ?= 120s
OBSERVABILITY_NAMESPACE ?= observability
VICTORIA_LOGS_RELEASE ?= victoria-logs
VICTORIA_LOGS_CHART := oci://ghcr.io/victoriametrics/helm-charts/victoria-logs-single
VICTORIA_LOGS_CHART_VERSION := 0.13.10
VICTORIA_LOGS_VALUES := deploy/observability/victoria-logs-values.yaml
OBSERVABILITY_TIMEOUT ?= 300s
ALERTING_DIR := deploy/observability/alerting
ALERTING_TIMEOUT ?= 180s
LOGS_QUERY ?= _time:15m
LOGS_LIMIT ?= 20
VICTORIA_METRICS_RELEASE ?= victoria-metrics
VICTORIA_METRICS_CHART := oci://ghcr.io/victoriametrics/helm-charts/victoria-metrics-single
VICTORIA_METRICS_CHART_VERSION := 0.49.0
VICTORIA_METRICS_VALUES := deploy/observability/victoria-metrics-values.yaml
POSTGRES_EXPORTER_DIR := deploy/observability/postgres-exporter
METRICS_QUERY ?= up{namespace="badgoodtune"}
export LOGS_QUERY LOGS_LIMIT
export METRICS_QUERY KUBECTL K8S_NAMESPACE

.PHONY: web-check-container k8s-backend-reload k8s-frontend-reload metrics-deploy metrics-status metrics-open metrics-query metrics-targets metrics-check metrics-generate metrics-validate metrics-debug metrics-delete

.PHONY: help check generate generate-check fmt fmt-check vet test integration-db test-integration test-integration-running web-install web-generate web-check build db-up db-down compose-up compose-down compose-smoke security-check minikube-start minikube-status minikube-stop k8s-images k8s-load k8s-validate k8s-deploy k8s-up k8s-down k8s-status k8s-scale-up k8s-scale-down k8s-open k8s-delete logs-validate logs-deploy logs-status logs-debug logs-open logs-query logs-delete logs-generate alerts-validate alerts-deploy alerts-status alerts-debug alerts-open alertmanager-open alerts-check alerts-trigger alerts-delete stack-up stack-down

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; printf "Usage: make <target>\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-24s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

check: ## Run code generation, formatting, static analysis, tests, frontend checks, and manifest validation
	$(MAKE) generate-check
	$(MAKE) fmt-check
	$(MAKE) vet
	$(MAKE) test
	$(MAKE) web-check
	$(MAKE) k8s-validate
	$(MAKE) logs-validate
	$(MAKE) alerts-validate
	$(MAKE) metrics-validate
	$(MAKE) reliability-validate
	$(MAKE) reliability-test

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

web-check-container: ## Check the frontend using the pinned Node runtime in Docker
	docker build --target builder -t $(FRONTEND_IMAGE)-check -f web/Dockerfile .
	docker run --rm --network=none $(FRONTEND_IMAGE)-check sh -ec 'node --version; npm run format:check; npm run lint; npm run test; npm run build'

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
	$(KUBECTL) rollout restart deployment/badgoodtune-backend deployment/badgoodtune-frontend -n $(K8S_NAMESPACE)
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
	$(KUBECTL) rollout status deployment/badgoodtune-backend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(KUBECTL) rollout status deployment/badgoodtune-frontend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)

k8s-open: ## Open the frontend through its Minikube Service
	$(MINIKUBE) service badgoodtune-frontend -n $(K8S_NAMESPACE)

k8s-delete: ## Delete the application Namespace and its persistent data
	$(KUBECTL) delete namespace $(K8S_NAMESPACE) --ignore-not-found

logs-validate: ## Render and validate the VictoriaLogs Helm release
	@set -eu; directory=$$(mktemp -d); manifest="$$directory/rendered.yaml"; \
		trap 'rm -f "$$manifest"; rmdir "$$directory"' EXIT HUP INT TERM; \
		$(HELM) template $(VICTORIA_LOGS_RELEASE) $(VICTORIA_LOGS_CHART) \
			--version $(VICTORIA_LOGS_CHART_VERSION) \
			--namespace $(OBSERVABILITY_NAMESPACE) \
			--values $(VICTORIA_LOGS_VALUES) >"$$manifest"; \
		test -s "$$manifest"; \
		grep -q '^kind:' "$$manifest"; \
		kubeconform -strict -summary "$$manifest"

logs-deploy: ## Install or upgrade VictoriaLogs and Vector, then wait for readiness
	$(HELM) upgrade --install $(VICTORIA_LOGS_RELEASE) $(VICTORIA_LOGS_CHART) \
		--version $(VICTORIA_LOGS_CHART_VERSION) \
		--namespace $(OBSERVABILITY_NAMESPACE) \
		--create-namespace \
		--values $(VICTORIA_LOGS_VALUES) \
		--wait --timeout $(OBSERVABILITY_TIMEOUT)
	$(KUBECTL) rollout status statefulset/victoria-logs -n $(OBSERVABILITY_NAMESPACE) --timeout=$(OBSERVABILITY_TIMEOUT)
	$(KUBECTL) rollout status daemonset/victoria-logs-vector -n $(OBSERVABILITY_NAMESPACE) --timeout=$(OBSERVABILITY_TIMEOUT)

logs-status: ## Show VictoriaLogs, Vector, Service, and storage state
	$(KUBECTL) get pods,services,statefulsets,daemonsets,persistentvolumeclaims -n $(OBSERVABILITY_NAMESPACE)
	@$(KUBECTL) rollout status statefulset/victoria-logs -n $(OBSERVABILITY_NAMESPACE) --timeout=10s
	@if ! $(KUBECTL) rollout status daemonset/victoria-logs-vector -n $(OBSERVABILITY_NAMESPACE) --timeout=10s; then \
		$(KUBECTL) logs daemonset/victoria-logs-vector -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix; \
		exit 1; \
	fi

logs-debug: ## Show recent Vector and VictoriaLogs container logs
	$(KUBECTL) logs daemonset/victoria-logs-vector -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix
	$(KUBECTL) logs statefulset/victoria-logs -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix

logs-open: ## Forward VictoriaLogs vmui to http://127.0.0.1:9428/select/vmui/
	$(KUBECTL) port-forward service/victoria-logs 9428:9428 -n $(OBSERVABILITY_NAMESPACE)

logs-query: ## Run LOGS_QUERY against VictoriaLogs through a temporary port-forward
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-logs 19428:9428 \
		curl --fail --silent --show-error --max-time 15 --get \
			--data-urlencode "query=$$LOGS_QUERY" --data-urlencode "limit=$$LOGS_LIMIT" \
			http://127.0.0.1:19428/select/logsql/query

logs-generate: ## Run the CRUD smoke test through Kubernetes to generate application logs
	@./scripts/port-forward.sh $(K8S_NAMESPACE) service/badgoodtune-frontend 18080:80 \
		env BASE_URL=http://127.0.0.1:18080 ./scripts/smoke.sh

logs-delete: ## Delete VictoriaLogs and Vector with their stored logs
	$(HELM) uninstall $(VICTORIA_LOGS_RELEASE) -n $(OBSERVABILITY_NAMESPACE) --ignore-not-found
	$(KUBECTL) delete persistentvolumeclaims -n $(OBSERVABILITY_NAMESPACE) -l app.kubernetes.io/instance=$(VICTORIA_LOGS_RELEASE)

alerts-validate: ## Validate vmalert and Alertmanager manifests against strict schemas
	kubeconform -strict -summary $(ALERTING_DIR)

alerts-deploy: ## Deploy vmalert and Alertmanager, then wait for readiness
	@set -eu; \
		alertmanager_config_version=$$($(KUBECTL) get configmap/alertmanager-config -n $(OBSERVABILITY_NAMESPACE) -o jsonpath='{.metadata.resourceVersion}' 2>/dev/null || true); \
		vmalert_rules_version=$$($(KUBECTL) get configmap/vmalert-rules -n $(OBSERVABILITY_NAMESPACE) -o jsonpath='{.metadata.resourceVersion}' 2>/dev/null || true); \
		$(KUBECTL) get service/victoria-logs -n $(OBSERVABILITY_NAMESPACE) >/dev/null; \
		$(KUBECTL) apply -f $(ALERTING_DIR)/10-configuration; \
		new_alertmanager_config_version=$$($(KUBECTL) get configmap/alertmanager-config -n $(OBSERVABILITY_NAMESPACE) -o jsonpath='{.metadata.resourceVersion}'); \
		new_vmalert_rules_version=$$($(KUBECTL) get configmap/vmalert-rules -n $(OBSERVABILITY_NAMESPACE) -o jsonpath='{.metadata.resourceVersion}'); \
		$(KUBECTL) apply -f $(ALERTING_DIR)/20-services; \
		$(KUBECTL) apply -f $(ALERTING_DIR)/30-workloads; \
		if [ -n "$$alertmanager_config_version" ] && [ "$$alertmanager_config_version" != "$$new_alertmanager_config_version" ]; then \
			$(KUBECTL) rollout restart deployment/alertmanager -n $(OBSERVABILITY_NAMESPACE); \
		fi; \
		if [ -n "$$vmalert_rules_version" ] && [ "$$vmalert_rules_version" != "$$new_vmalert_rules_version" ]; then \
			$(KUBECTL) rollout restart deployment/vmalert -n $(OBSERVABILITY_NAMESPACE); \
		fi; \
		$(KUBECTL) rollout status deployment/alertmanager -n $(OBSERVABILITY_NAMESPACE) --timeout=$(ALERTING_TIMEOUT); \
		$(KUBECTL) rollout status deployment/vmalert -n $(OBSERVABILITY_NAMESPACE) --timeout=$(ALERTING_TIMEOUT)

alerts-status: ## Show vmalert and Alertmanager resources and readiness
	$(KUBECTL) get pods,services,deployments,configmaps -n $(OBSERVABILITY_NAMESPACE) -l app.kubernetes.io/part-of=badgoodtune-observability
	$(KUBECTL) rollout status deployment/alertmanager -n $(OBSERVABILITY_NAMESPACE) --timeout=10s
	$(KUBECTL) rollout status deployment/vmalert -n $(OBSERVABILITY_NAMESPACE) --timeout=10s

alerts-debug: ## Show recent vmalert and Alertmanager logs
	$(KUBECTL) logs deployment/vmalert -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix
	$(KUBECTL) logs deployment/alertmanager -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix

alerts-open: ## Forward vmalert UI to http://127.0.0.1:8880/
	$(KUBECTL) port-forward service/vmalert 8880:8880 -n $(OBSERVABILITY_NAMESPACE)

alertmanager-open: ## Forward Alertmanager UI to http://127.0.0.1:9093/
	$(KUBECTL) port-forward service/alertmanager 9093:9093 -n $(OBSERVABILITY_NAMESPACE)

alerts-check: ## Wait until the BadGoodTune warning-log alert is firing
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/vmalert 18880:8880 ./scripts/check-alerts.sh

alerts-trigger: logs-generate ## Generate warning logs and verify that the alert fires
	$(MAKE) alerts-check

alerts-delete: ## Delete vmalert and Alertmanager resources
	$(KUBECTL) delete -R -f $(ALERTING_DIR) --ignore-not-found

stack-up: ## Deploy the application and the complete observability stack
	$(MAKE) k8s-up
	$(MAKE) logs-deploy
	$(MAKE) alerts-deploy
	$(MAKE) reliability-deploy

stack-down: ## Delete the application and observability stack, then stop Minikube
	$(MAKE) metrics-delete
	$(MAKE) k8s-delete
	$(MAKE) alerts-delete
	$(MAKE) logs-delete
	$(KUBECTL) delete namespace $(OBSERVABILITY_NAMESPACE) --ignore-not-found
	$(MAKE) minikube-stop

k8s-backend-reload: ## Rebuild, load, and restart only the backend using the local image tag
	docker build -t $(BACKEND_IMAGE) .
	$(MINIKUBE) image load --daemon --overwrite $(BACKEND_IMAGE)
	$(KUBECTL) apply -f $(K8S_DIR)/10-configuration
	$(KUBECTL) apply -f $(K8S_DIR)/50-application/backend-deployment.yaml
	$(KUBECTL) rollout restart deployment/badgoodtune-backend -n $(K8S_NAMESPACE)
	$(KUBECTL) rollout status deployment/badgoodtune-backend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)

k8s-frontend-reload: ## Rebuild, load, and restart only the frontend using the local image tag
	docker build -t $(FRONTEND_IMAGE) -f web/Dockerfile .
	$(MINIKUBE) image load --daemon --overwrite $(FRONTEND_IMAGE)
	$(KUBECTL) apply -f $(K8S_DIR)/50-application/frontend-deployment.yaml
	$(KUBECTL) rollout restart deployment/badgoodtune-frontend -n $(K8S_NAMESPACE)
	$(KUBECTL) rollout status deployment/badgoodtune-frontend -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)

metrics-validate: ## Render and validate VictoriaMetrics and PostgreSQL exporter manifests
	@set -eu; directory=$$(mktemp -d); manifest="$$directory/rendered.yaml"; \
		trap 'rm -f "$$manifest"; rmdir "$$directory"' EXIT HUP INT TERM; \
		$(HELM) template $(VICTORIA_METRICS_RELEASE) $(VICTORIA_METRICS_CHART) \
			--version $(VICTORIA_METRICS_CHART_VERSION) --namespace $(OBSERVABILITY_NAMESPACE) \
			--values $(VICTORIA_METRICS_VALUES) >"$$manifest"; \
		test -s "$$manifest"; \
		grep -q '^kind:' "$$manifest"; \
		kubeconform -strict -summary "$$manifest"; \
		kubeconform -strict -summary $(POSTGRES_EXPORTER_DIR)

metrics-deploy: ## Bootstrap the PostgreSQL monitoring role and install exporters and VictoriaMetrics
	$(KUBECTL) rollout status statefulset/postgres -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	@set -eu; \
		previous=$$($(KUBECTL) get secret/postgres-exporter-secret configmap/postgres-exporter-config -n $(K8S_NAMESPACE) -o json 2>/dev/null | jq -c '[.items[].data]' || true); \
		$(KUBECTL) apply -f $(POSTGRES_EXPORTER_DIR)/10-configuration; \
		current=$$($(KUBECTL) get secret/postgres-exporter-secret configmap/postgres-exporter-config -n $(K8S_NAMESPACE) -o json | jq -c '[.items[].data]'); \
		$(KUBECTL) delete job/postgres-exporter-bootstrap -n $(K8S_NAMESPACE) --ignore-not-found; \
		$(KUBECTL) apply -f $(POSTGRES_EXPORTER_DIR)/20-bootstrap; \
		if ! $(KUBECTL) wait --for=condition=complete job/postgres-exporter-bootstrap -n $(K8S_NAMESPACE) --timeout=$(K8S_JOB_TIMEOUT); then \
			$(KUBECTL) logs job/postgres-exporter-bootstrap -n $(K8S_NAMESPACE); exit 1; \
		fi; \
		$(KUBECTL) apply -f $(POSTGRES_EXPORTER_DIR)/30-workloads; \
		if [ -n "$$previous" ] && [ "$$previous" != "$$current" ]; then \
			$(KUBECTL) rollout restart deployment/postgres-exporter -n $(K8S_NAMESPACE); \
		fi; \
		$(KUBECTL) rollout status deployment/postgres-exporter -n $(K8S_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(HELM) upgrade --install $(VICTORIA_METRICS_RELEASE) $(VICTORIA_METRICS_CHART) \
		--version $(VICTORIA_METRICS_CHART_VERSION) --namespace $(OBSERVABILITY_NAMESPACE) \
		--create-namespace --values $(VICTORIA_METRICS_VALUES) --wait --timeout $(OBSERVABILITY_TIMEOUT)

metrics-status: ## Show VictoriaMetrics, PostgreSQL exporter, and persistent storage
	$(KUBECTL) get pods,services,statefulsets,persistentvolumeclaims -n $(OBSERVABILITY_NAMESPACE) -l app.kubernetes.io/instance=$(VICTORIA_METRICS_RELEASE)
	$(KUBECTL) get pods,deployments,jobs -n $(K8S_NAMESPACE) -l app.kubernetes.io/component=postgres-exporter

metrics-open: ## Forward VictoriaMetrics API, /targets, and /vmui/ to http://127.0.0.1:8428/
	$(KUBECTL) port-forward --address=127.0.0.1 service/victoria-metrics 8428:8428 -n $(OBSERVABILITY_NAMESPACE)

metrics-query: ## Query stored metrics using METRICS_QUERY
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics 18428:8428 \
		curl --fail --silent --show-error --max-time 15 --get --data-urlencode "query=$$METRICS_QUERY" http://127.0.0.1:18428/api/v1/query

metrics-check: ## Verify discovery, runtime and business metrics, and PostgreSQL health in the TSDB
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics 18428:8428 ./scripts/check-metrics.sh

metrics-targets: ## Show discovered scrape targets and their current health
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics 18428:8428 \
		sh -ec 'curl --fail --silent --show-error --max-time 15 http://127.0.0.1:18428/api/v1/targets | jq --exit-status ".data.activeTargets | map({job: .labels.job, pod: .labels.pod, health: .health, endpoint: .scrapeUrl, error: .lastError})"'

metrics-generate: ## Run controlled CRUD traffic and verify exact business metric changes in the TSDB
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics 18428:8428 ./scripts/generate-metrics.sh

metrics-debug: ## Show recent VictoriaMetrics and PostgreSQL exporter logs
	$(KUBECTL) logs statefulset/victoria-metrics -n $(OBSERVABILITY_NAMESPACE) --tail=100 --prefix
	$(KUBECTL) logs deployment/postgres-exporter -n $(K8S_NAMESPACE) --tail=100 --prefix

metrics-delete: ## Uninstall metrics workloads; retain TSDB PVC and the PostgreSQL monitoring role
	$(HELM) uninstall $(VICTORIA_METRICS_RELEASE) -n $(OBSERVABILITY_NAMESPACE) --ignore-not-found
	$(KUBECTL) delete -R -f $(POSTGRES_EXPORTER_DIR) --ignore-not-found

include make/reliability.mk
