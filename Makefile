.DEFAULT_GOAL := help

GO := go
SWAG := $(GO) run github.com/swaggo/swag/cmd/swag@latest
STATICCHECK_VERSION ?= v0.8.1
STATICCHECK := $(shell $(GO) env GOPATH)/bin/staticcheck
DOCS_DIR := internal/api/docs
MONGO_URI ?= mongodb://localhost:27017
MONGO_DB_NAME ?= freight

.PHONY: help setup deps tools tidy build test coverage vet staticcheck docs docs-check setup-hooks run seed docker-up docker-down docker-logs

help:
	@printf "Available targets:\n"
	@printf "  make setup        Download dependencies, tools, and enable Git hooks\n"
	@printf "  make deps          Download Go module dependencies\n"
	@printf "  make tools         Install development tools\n"
	@printf "  make tidy          Add missing and remove unused dependencies\n"
	@printf "  make build         Build all Go packages\n"
	@printf "  make test          Run Go tests\n"
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

coverage:
	$(GO) test -coverprofile=coverage.out ./...
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
