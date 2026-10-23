RELIABILITY_DIR := deploy/observability/reliability
RELIABILITY_TEST_IMAGE := victoriametrics/vmalert-tool:v1.154.0

.PHONY: reliability-validate reliability-test reliability-deploy reliability-status reliability-check reliability-open reliability-graphs reliability-monthly reliability-traffic reliability-stop-traffic reliability-demo-prepare reliability-demo-latency reliability-demo-errors reliability-demo-recover reliability-demo-finish

reliability-validate: ## Render and strictly validate the reliability stack and experiment Jobs
	@set -eu; directory=$$(mktemp -d); manifest="$$directory/rendered.yaml"; \
		trap 'rm -f "$$manifest"; rmdir "$$directory"' EXIT; \
		$(KUBECTL) kustomize $(RELIABILITY_DIR) >"$$manifest"; \
		test -s "$$manifest"; \
		kubeconform -strict -summary "$$manifest"; \
		kubeconform -strict -summary deploy/experiments/reliability/*.yaml

reliability-test: ## Test MetricsQL rules on synthetic successes, failures, resets, and missing data
	docker run --rm -v "$(CURDIR)/$(RELIABILITY_DIR):/rules:ro" -w /rules $(RELIABILITY_TEST_IMAGE) unittest -files=/rules/rules.test.yaml

reliability-deploy: reliability-validate reliability-test ## Install the prober and recording rules, then update VictoriaMetrics scraping
	$(KUBECTL) apply -k $(RELIABILITY_DIR)
	$(KUBECTL) rollout status deployment/blackbox-exporter -n $(OBSERVABILITY_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(KUBECTL) rollout status deployment/reliability-evaluator -n $(OBSERVABILITY_NAMESPACE) --timeout=$(K8S_ROLLOUT_TIMEOUT)
	$(MAKE) metrics-deploy

reliability-status: ## Show prober, evaluator, and temporary experiment workloads
	$(KUBECTL) get pods,services,deployments -n $(OBSERVABILITY_NAMESPACE) -l app.kubernetes.io/part-of=badgoodtune-reliability
	$(KUBECTL) get jobs,pods -n $(K8S_NAMESPACE) -l app.kubernetes.io/part-of=badgoodtune-reliability-experiment

reliability-check: ## Wait for fresh probe data and all four five-minute SLI series
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics :8428 ./scripts/check-reliability.sh

reliability-open: ## Forward VictoriaMetrics and print ready-to-use reliability graph links
	@$(MAKE) --no-print-directory reliability-graphs
	$(MAKE) metrics-open

reliability-graphs: ## Print vmui links for SLI, SLO, SLA, coverage, and error budgets
	@./scripts/reliability-graphs.sh

reliability-monthly: ## Calculate provisional calendar-month SLA from actual recorded minute samples
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics :8428 ./scripts/reliability-monthly.sh

reliability-traffic: ## Start bounded, read-only catalog traffic in Minikube
	./scripts/reliability-experiment.sh traffic

reliability-stop-traffic: ## Stop the read-only traffic generator
	./scripts/reliability-experiment.sh stop-traffic

reliability-demo-prepare: ## Prepare a local experiment with a two-second backend SQL timeout
	./scripts/reliability-experiment.sh prepare

reliability-demo-latency: ## Briefly lock catalog reads without changing rows to demonstrate slow successful responses
	./scripts/reliability-experiment.sh latency
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics :8428 ./scripts/verify-reliability-phase.sh latency

reliability-demo-errors: ## Hold a bounded database lock to demonstrate real query timeouts and HTTP errors
	./scripts/reliability-experiment.sh errors
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics :8428 ./scripts/verify-reliability-phase.sh errors

reliability-demo-recover: ## Immediately release experiment locks without restarting the backend
	./scripts/reliability-experiment.sh recover
	@./scripts/port-forward.sh $(OBSERVABILITY_NAMESPACE) service/victoria-metrics :8428 ./scripts/verify-reliability-phase.sh recovery

reliability-demo-finish: ## Stop experiments and traffic, and restore the normal backend SQL timeout
	./scripts/reliability-experiment.sh finish
