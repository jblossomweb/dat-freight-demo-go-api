# AGENTS.md

Guidance for AI coding agents working in this repo.

## Project

REST API backing the DAT freight demo SPA (AG Grid frontend). Goal: replace
client-side pagination/sorting/filtering/quicksearch with server-side equivalents.

**Phase 1 (done):** Docker infrastructure only — Go API container + MongoDB
container, `GET /health` returns 200. No AG Grid contract endpoints yet.

**Phase 2 (current):** Seed data — `cmd/seed` loads `internal/db/seeds/loads.json`
into the `loads` collection. Still no AG Grid contract endpoints.

**Phase 3 (planned):** AG Grid server-side row model contract (pagination, sort,
filter, quicksearch) built on top of the `loads` collection.

## Stack

- Go 1.27+, standard `net/http` (no router/framework libraries)
- MongoDB via `go.mongodb.org/mongo-driver/v2`
- Docker Compose for local orchestration (`compose.yml`)

## Structure

- `cmd/api/main.go` — server entrypoint, env config, Mongo connection at startup
- `cmd/seed/main.go` — one-off command to load `internal/db/seeds/loads.json` into the `loads` collection
- `internal/db` — MongoDB client setup (`db.Connect`)
- `internal/db/seeds` — seed data files
- `internal/` — reserved for future packages (handlers, repositories, models)

## Conventions

- Module name is `dat-freight-demo-api` — matches the repo root folder name.
- Prefer the standard library over third-party dependencies unless there's a
  concrete need (e.g. the Mongo driver).
- Server must fail fast (non-zero exit) if MongoDB is unreachable at startup.
- `/health` performs a live Mongo ping on each request (not just cached startup state).

## Running & verifying

```bash
docker compose up --build
curl -i http://localhost:8080/health
```

Seed data (with `mongo` running):

```bash
MONGO_URI=mongodb://localhost:27017 go run ./cmd/seed
```

## Out of scope for phase 1

AG Grid server-side row model contract (pagination, sort, filter, quicksearch
endpoints), collections/models/queries, auth.
