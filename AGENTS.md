# AGENTS.md

Guidance for AI coding agents working in this repo.

## Project

REST API backing the DAT freight demo SPA (AG Grid frontend). Goal: replace
client-side pagination/sorting/filtering/quicksearch with server-side equivalents.

**Phase 1 (done):** Docker infrastructure only — Go API container + MongoDB
container, `GET /health` returns 200. No AG Grid contract endpoints yet.

**Phase 2 (done):** Seed data — `cmd/seed` loads `internal/db/seeds/loads.json`
into the `loads` collection.

**Phase 3 (done):** `GET /loads` implements the AG Grid server-side row
model contract — pagination (`startRow`/`endRow`), single-column sort
(`sortModel`), multi-column filtering (`filterModel`: text/number/date/set), and
quicksearch (`quickSearch`). Response: `{ rows, lastRow }`.

**Phase 4 (done):** Human-friendly additions on top of the AG Grid contract —
alias query params (any load field as a shorthand query param; single value is
exact match, repeated values are "one of"; numeric fields auto-coerced) and
response metadata (`requestURL`, `query` echo, `meta`: totalRows,
filteredRows, pageSize, numPages, currentPage, hasNextPage, nextPageURL,
responseTimeMs, timestamp). `rows`/`lastRow` remain the untouched AG Grid
contract; everything else is additive for debugging/readability.

**Phase 5 (done):** `GET /load/{id}` and `GET /load?id={id}` fetch a single load
by its `id` field. Response: `{ requestURL, meta: {responseTimeMs, timestamp},
result }`. Responds 404 (`LOAD_NOT_FOUND`) if no match, 400 (`LOAD_ID_REQUIRED`)
if no id given.

## Stack

- Go 1.27+, standard `net/http` (no router/framework libraries)
- MongoDB via `go.mongodb.org/mongo-driver/v2`
- Docker Compose for local orchestration (`compose.yml`)

## Structure

- `cmd/api/main.go` — server entrypoint, env config, Mongo connection at startup
- `cmd/seed/main.go` — one-off command to load `internal/db/seeds/loads.json` into the `loads` collection
- `internal/loads` — `GET /loads` and `GET /load` handlers, query param parsing, Mongo filter/sort building
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
