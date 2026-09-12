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

**Phase 6 (done):** Internal cleanups and refactors — centralized the Mongo
query timeout (previously a duplicated magic number) into a single constant,
introduced a repository/service/routes layering so HTTP handlers never call
the Mongo driver directly, and extracted `GET /health` out of `main.go` into
its own `internal/health` package. Regrouped packages by role: HTTP-facing
code under `internal/api/*`, shared infrastructure (`internal/db`) at the top
level. `main.go` is now a pure composition root with no inline HTTP handlers.
No behavior change.

**Phase 7 (done):** More human-friendly aliases on `GET /loads`, plus file
naming cleanups — added `q` (alias for `quickSearch`), `sortBy`/`sortDir`
(alias for a single-column `sortModel` entry; `sortDir` accepts
`asc`/`ascending`/`desc`/`descending` case-insensitively and defaults to
ascending for any other value, matching `sortModel`'s own lenient handling),
and `offset`/`limit` (aliases for `startRow`/`endRow`). All follow the
existing alias precedence rule: the real AG Grid param wins if present.
Renamed `list.go`/`get.go` to `list_handler.go`/`get_handler.go` and
`mongo.go` to `connect.go` for clarity. No other behavior change.

**Phase 8 (done):** Fixed a missing index on the `id` field — it's the
primary lookup key for `GET /load/{id}`, the `id` alias, and quicksearch's
`$or`, but had no index besides Mongo's default `_id`, causing a full
collection scan on every such query. `cmd/seed` now creates a unique index
on `{id: 1}` after inserting. Verified via `explain()` that lookups on `id`
now use `IXSCAN` instead of `COLLSCAN`. Existing deployments need to re-run
the seed command to pick up the index.

## Stack

- Go 1.27+, standard `net/http` (no router/framework libraries)
- MongoDB via `go.mongodb.org/mongo-driver/v2`
- Docker Compose for local orchestration (`compose.yml`)

## Structure

- `cmd/api/main.go` — server entrypoint, env config, Mongo connection at startup
- `cmd/seed/main.go` — one-off command to load `internal/db/seeds/loads.json` into the `loads` collection
- `internal/api/health` — `GET /health` route + handler (live Mongo ping per request)
- `internal/api/loads` — `routes.go` (`RegisterRoutes`, owns the package's route
  paths), `GET /loads` and `GET /load` handlers (HTTP transport only, no direct
  Mongo calls), query param parsing, Mongo filter/sort building, plus
  `service.go` (business logic) and `repository.go` (the only place that talks
  to `mongo-driver`)
- `internal/db` — MongoDB client setup (`db.Connect`), shared by both
  `cmd/api` and `cmd/seed`
- `internal/db/seeds` — seed data files
- `internal/` — reserved for future packages, grouped by role (`api/` for
  HTTP-facing packages; a `shared/` grouping can be introduced later if a
  second cross-cutting package joins `db`)

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
