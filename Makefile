.DEFAULT_GOAL := help

GO := go
SWAG_VERSION ?= v1.16.6
SWAG := $(GO) run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
STATICCHECK_VERSION ?= v0.8.1
STATICCHECK := $(shell $(GO) env GOPATH)/bin/staticcheck
DOCS_DIR := internal/api/docs
MONGO_URI ?= mongodb://localhost:27017
MONGO_DB_NAME ?= freight

.PHONY: help setup deps tools tidy build test mongo-integration-up mongo-integration-down test-integration test-integration-ci coverage vet staticcheck docs docs-check setup-hooks run seed docker-up docker-down docker-logs

help:
	@printf "Available targets:\n"
	@printf "  make setup        Download dependencies, tools, and enable Git hooks\n"
	@printf "  make deps          Download Go module dependencies\n"
	@printf "  make tools         Install development tools\n"
	@printf "  make tidy          Add missing and remove unused dependencies\n"
	@printf "  make build         Build all Go packages\n"
	@printf "  make test          Run Go tests\n"
	@printf "  make mongo-integration-up   Start isolated integration MongoDB\n"
	@printf "  make mongo-integration-down Stop isolated integration MongoDB\n"
	@printf "  make test-integration       Run MongoDB integration tests\n"
	@printf "  make coverage      Generate and open an HTML coverage report\n"
	@printf "  make vet           Run go vet\n"
	@printf "  make staticcheck   Run Staticcheck\n"
	@printf "  make docs          Regenerate Swagger documentation\n"
	@printf "  make docs-check    Verify generated Swagger documentation is current\n"
	@printf "  make setup-hooks   Enable the repository's Git hooks\n"
	@printf "  make run           Run the API locally\n"
	@printf "  make seed          Seed MongoDB from the local host\n"
	@printf "  make docker-up     Build and start the Docker Compose stack\n"
	@printf "  make docker-down   Stop and remove the Docker Compose stack\n"
	@printf "  make docker-logs   Follow API container logs\n"

setup: deps tools setup-hooks

deps:
	$(GO) mod download

tools:
	$(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)

tidy:
	$(GO) mod tidy

build:
	$(GO) build ./...

test:
	$(GO) test ./...

mongo-integration-up:
	docker compose --profile integration up -d --wait mongo-integration

mongo-integration-down:
	docker compose --profile integration rm -sf mongo-integration
	@volumes=$$(docker volume ls -q --filter label=com.docker.compose.volume=mongo-integration-data); \
	if [ -n "$$volumes" ]; then docker volume rm $$volumes; fi

test-integration:
	@if [ "$(CI)" = "true" ]; then \
		$(MAKE) test-integration-ci; \
	else \
		mongo_test_uri="$(MONGO_TEST_URI)"; \
		if [ -z "$$mongo_test_uri" ]; then mongo_test_uri="mongodb://localhost:27018"; fi; \
		trap '$(MAKE) mongo-integration-down' EXIT INT TERM; \
		$(MAKE) mongo-integration-up; \
		MONGO_INTEGRATION=1 MONGO_TEST_URI="$$mongo_test_uri" $(GO) test -v ./internal/api/loads -run TestRepositoryIntegration -count=1; \
	fi

test-integration-ci:
	@test -n "$(MONGO_TEST_URI)" || { echo "MONGO_TEST_URI must be set for integration tests"; exit 1; }; \
	MONGO_INTEGRATION=1 MONGO_TEST_URI="$(MONGO_TEST_URI)" $(GO) test -v ./internal/api/loads -run TestRepositoryIntegration -count=1;

coverage:
	$(GO) test -coverprofile=coverage.out $$(go list ./... | grep -v '/internal/api/docs$$')
	$(GO) tool cover -func=coverage.out
	$(GO) tool cover -html=coverage.out

vet:
	$(GO) vet ./...

staticcheck:
	$(STATICCHECK) ./...

docs:
	$(SWAG) init -g cmd/api/main.go -o $(DOCS_DIR) --parseInternal

docs-check:
	@tmp_dir=$$(mktemp -d); \
	trap 'rm -rf "$$tmp_dir"' EXIT; \
	$(SWAG) init -g cmd/api/main.go -o "$$tmp_dir" --parseInternal --packageName docs; \
	diff -q $(DOCS_DIR)/docs.go "$$tmp_dir/docs.go"; \
	diff -q $(DOCS_DIR)/swagger.json "$$tmp_dir/swagger.json"; \
	diff -q $(DOCS_DIR)/swagger.yaml "$$tmp_dir/swagger.yaml"

setup-hooks:
	git config core.hooksPath .githooks

run:
	$(GO) run ./cmd/api

seed:
	MONGO_URI="$(MONGO_URI)" MONGO_DB_NAME="$(MONGO_DB_NAME)" $(GO) run ./cmd/seed

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-logs:
	docker compose logs -f api
