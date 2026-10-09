SHELL := /bin/sh
GO_PACKAGES := ./cmd/... ./db/... ./internal/...

.PHONY: generate generate-check fmt fmt-check vet test integration-db test-integration test-integration-running web-install web-generate web-check build db-up db-down compose-up compose-down compose-smoke security-check

generate:
	cd api && go tool oapi-codegen --config oapi-codegen.yaml openapi.yaml
	go tool sqlc generate
	gofmt -w internal/transport/http/openapi internal/repository/postgres/sqlc
	cd web && npm run generate

generate-check: generate
	git diff --exit-code -- internal/transport/http/openapi internal/repository/postgres/sqlc
	git diff --exit-code -- web/src/api/generated

fmt:
	gofmt -w $$(find cmd db internal -name '*.go' -type f)
	cd web && npm run format

fmt-check:
	test -z "$$(gofmt -l $$(find cmd db internal -name '*.go' -type f))"
	cd web && npm run format:check

vet:
	go vet $(GO_PACKAGES)

test:
	go test $(GO_PACKAGES)

integration-db: db-up
	docker compose exec -T postgres sh -ec 'if [ "$$(psql -U "$$POSTGRES_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '\''badgoodtune_test'\''")" != "1" ]; then createdb -U "$$POSTGRES_USER" -O "$$POSTGRES_USER" badgoodtune_test; fi'

test-integration: integration-db
	$(MAKE) test-integration-running

test-integration-running:
	TEST_DATABASE_URL="postgres://badgoodtune:badgoodtune@localhost:$${POSTGRES_PORT:-54329}/badgoodtune_test?sslmode=disable" go test -tags=integration -count=1 ./internal/repository/postgres

web-install:
	cd web && npm ci

web-generate:
	cd web && npm run generate

web-check:
	cd web && npm run check

build:
	mkdir -p bin
	go build -trimpath -o bin/server ./cmd/server
	go build -trimpath -o bin/migrate ./cmd/migrate

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

compose-up:
	docker compose up --build -d --wait

compose-down:
	docker compose down

compose-smoke: compose-up
	./scripts/smoke.sh

security-check:
	go tool govulncheck $(GO_PACKAGES)
	cd web && npm audit --audit-level=high
