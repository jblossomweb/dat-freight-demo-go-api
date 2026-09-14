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

**Phase 9 (done):** Added interactive API documentation via
[swaggo/swag](https://github.com/swaggo/swag) annotations. Exported response
types so swag can generate accurate schemas, committed the generated spec under
`internal/api/docs/`, and served Swagger UI at `GET /swagger/index.html` via
`github.com/swaggo/http-swagger/v2`. No behavior change to existing endpoints.

**Phase 10 (done):** Added a Makefile for common development workflows,
including dependency setup, Swagger generation/freshness checks, Staticcheck,
tests, vet, builds, seeding, and Docker commands. Added native Git hooks under
`.githooks/` and a reusable `workflow_call` validation workflow under
`.github/workflows/validate.yml`; validation runs type checks, vet, Staticcheck,
tests, and generated-doc checks as separate jobs. README documents the setup
and Node.js-to-Go workflow parallels.

**Phase 11 (done):** Added a `make coverage` target for HTML Go coverage
reports and ignored the generated `coverage.out` profile. Added dependency
injection interfaces at the repository, service, and health Mongo ping
boundaries so unit tests can use handwritten fakes without MongoDB. Added
unit coverage for query parsing, Mongo filter construction, HTTP handlers,
service behavior, and health responses. Repository integration tests remain
deferred to a separate follow-up commit.

**Phase 12 (done):** Made `MONGO_URI` required for the API so startup fails
fast when it is unset or empty, while retaining defaults for `PORT` and
`MONGO_DB_NAME`. Removed Compose's Mongo URI fallback and documented the
required environment variable in README. Added unit tests for configured,
missing, and empty Mongo URI values.

**Phase 13 (done):** Refactored command configuration and composition for
readability: split the API composition root into configuration, server setup,
and lifecycle files; extracted shared environment helpers into `internal/env`;
and added a dedicated seed configuration. Expanded unit coverage for shared
environment behavior and route registration, including method handling. No
repository integration tests were added; those remain a separate follow-up.

**Phase 14 (done):** Pinned the Swagger generator to the latest compatible
stable version (`swaggo/swag` v1.16.6) so upstream generator changes cannot
unexpectedly make `docs-check` fail. Updated the coverage target to exclude
generated Swagger output from the coverage scan while retaining the
overall summary and HTML report. Moved the Swagger UI route from `/swagger/`
to `/docs/`, with the UI at `GET /docs/index.html` and the raw spec at
`GET /docs/doc.json`.

**Phase 15 (done):** Added a lightweight, opt-in MongoDB repository integration
suite covering counts, filtered/sorted/paginated reads, successful lookups, and
not-found translation. Added `make test-integration` and an independent Mongo 7
service job to the reusable validation workflow. Unit tests and Git hooks remain
Mongo-free; integration coverage runs separately in CI and on demand locally.

**Phase 16 (done):** Hardened local and container security by running the API
as a non-root distroless user, binding local Mongo ports to loopback only, and
preventing Make from echoing Mongo connection strings. Improved README guidance
for development seeding, isolated local/CI integration tests, required test
URIs, and intentionally unauthenticated demo Swagger documentation.

**Phase 17 (done):** Renamed the Go module and project identity from
`dat-freight-demo-api` to `dat-freight-demo-go-api`, updating internal imports,
Swagger metadata, and generated documentation.

**Phase 18 (done):** Added a production-style EC2 deployment flow using a
GitHub Actions workflow that validates the code, builds a container image, pushes
it to GHCR, and uses AWS Systems Manager to run the deploy script on the EC2
instance without requiring SSH access. Provisioning included: creating an EC2
instance with the SSM agent enabled and the `AmazonSSMManagedInstanceCore`
role attached; setting up a GitHub Actions IAM user with programmatic access and
a least-privilege SSM policy; creating GitHub repository secrets for AWS access,
AWS region, EC2 instance ID, GHCR pull credentials, and MongoDB connection
settings; and moving the database to MongoDB Atlas with a cluster, database user,
private/public access configuration, network access rules, and a runtime
`MONGO_URI` stored in GitHub secrets. The app container is configured to listen
on port 8080 and is launched with `PORT`, `MONGO_URI`, and `MONGO_DB_NAME`
variables, allowing the same image to serve both local Docker runs and the EC2
deployment while keeping the repository deployment workflow and runtime
configuration separated.

**Phase 19 (done):** Added `GET /loads/stats` for the frontend statistics page.
The endpoint runs a single MongoDB aggregation across the full loads collection
and returns the total row count plus fixed, ordered counts for the `Flatbed`,
`Reefer`, and `Van` equipment types and the `Available`, `In Transit`, and
`Delivered` statuses. Successful aggregate results are cached in memory using
the configurable `LOAD_STATS_CACHE_TTL` duration (`5m` by default; `0` disables
caching), while request timing and timestamps remain fresh. Added route,
handler, service/cache, configuration, and repository integration coverage,
documented the endpoint in README, and regenerated the committed Swagger spec.

**Phase 20 (done):** Added configurable CORS support at the API server boundary
so browser-based frontends can call every endpoint consistently. The
standard-library middleware uses an exact origin allowlist from the
comma-separated `CORS_ALLOWED_ORIGINS` environment variable, handles `OPTIONS`
preflight requests for `GET`, returns `Vary: Origin`, and rejects disallowed
preflights without enabling wildcard origins. CORS remains disabled when the
variable is unset; local Compose and `.env.example` explicitly allow the Vite
development and preview origins on ports 5173 and 4173. Added configuration
validation, middleware tests, and README guidance for local and environment-
specific origin policy.

**Phase 21 (done):** Extended `GET /loads/stats` with the same `quickSearch` and
`q` all-field search supported by `GET /loads`. The endpoint now echoes the
effective quick search, preserves the full collection count as `meta.numTotal`,
reports the matching count as `meta.numResults`, and calculates its fixed,
ordered equipment and status totals from the filtered dataset. Reworked the
stats cache into a fixed 50-entry TTL/LRU keyed by effective search term, added
filtered aggregation and cache coverage, and updated README and Swagger docs.

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

- Module name is `dat-freight-demo-go-api` — matches the intended repo name.
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
