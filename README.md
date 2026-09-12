# dat-freight-demo-api

Go + MongoDB REST API backing the DAT freight demo SPA. Phase 1: Docker
infrastructure (server boot, Mongo connectivity, health check). Phase 2: seed
data. Server-side pagination/sorting/filtering/quicksearch for AG Grid come in
a later phase.

## Requirements

- Docker (with Compose v2)
- Go 1.27+ (only needed for local editor tooling; the app itself runs in Docker)

## Run

```bash
cp .env.example .env
docker compose up --build
```

Verify:

```bash
curl -i http://localhost:8080/health
```

Expect `200 OK` with `{"status":"ok","mongo":"ok"}` once both containers are healthy.

## Structure

```
cmd/api/main.go      - HTTP server entrypoint
cmd/seed/main.go     - loads seed data into MongoDB
internal/db          - MongoDB client setup
internal/db/seeds    - seed data (loads.json)
Dockerfile           - multi-stage build for the api service
compose.yml          - api + mongo services
```

## Environment variables

| Var           | Default               | Description               |
| ------------- | --------------------- | ------------------------- |
| PORT          | 8080                  | HTTP listen port          |
| MONGO_URI     | mongodb://mongo:27017 | MongoDB connection string |
| MONGO_DB_NAME | freight               | Target database name      |

## Seeding data

With the `mongo` container running, load `internal/db/seeds/loads.json` into the
`loads` collection (drops and replaces any existing documents):

```bash
docker compose up -d mongo
MONGO_URI=mongodb://localhost:27017 go run ./cmd/seed
```
