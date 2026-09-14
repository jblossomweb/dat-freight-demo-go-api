# Freight Load Board API

Go + MongoDB REST API backing the DAT Freight Load Board demo SPA. Implements the AG Grid
server-side row model contract (pagination, sort, filter, quicksearch) for
freight loads, plus a single-load lookup endpoint.

# Quick Start

## Requirements

- Docker (with Compose v2)
- Go 1.27+ (only needed for local editor tooling; the app itself runs in Docker)

## Run

Get everything setup, and start the 2 dev containers:

```bash
cp .env.example .env
make setup
make docker-up
```

## Verify

Verify the dev database and API server is up:

```bash
curl -i http://localhost:8080/health
```

Expect `200 OK` with `{"status":"ok","mongo":"ok"}` once both containers are healthy.

## Seed Data

In a second terminal, seed the local dev database:

```bash
MONGO_URI=mongodb://localhost:27017 make seed
```

# Developer Workflow

> Whenever you change Go code, re-run `make docker-up` — the binary is compiled
> into the image, and this target rebuilds the image automatically.

## Development commands

The `Makefile` provides shortcuts for common local workflows. Run `make help`
to see all available targets.

```bash
make help        # list all available targets
make setup       # download dependencies, tools, and enable Git hooks
make deps        # download Go module dependencies
make tools       # install development tools
make tidy        # add missing and remove unused dependencies
make build       # build all Go packages
make test        # run Go tests
make vet         # run go vet
make staticcheck # run Staticcheck
make docs        # regenerate Swagger documentation
make docs-check  # verify generated Swagger documentation is current
make setup-hooks # enable the repository's Git hooks
make run         # run the API locally
make seed        # seed MongoDB from the local host
make docker-up   # build and start the Docker Compose stack
make docker-down # stop and remove the Docker Compose stack
make docker-logs # follow API container logs
make test-integration       # run MongoDB integration tests
make mongo-integration-up   # start isolated integration MongoDB
make mongo-integration-down # remove isolated integration MongoDB
```

### Tests

To run unit tests at any time:

```bash
make test
```

This will also be triggered by githooks and in CI.

The normal
`make test` command does not require MongoDB.

### Integration Tests

The integration suite has a separate command:

```bash
make test-integration
```

It uses a unique temporary database, only spins it up on demand, and cleans it up after finishing.

Local runs default to the isolated integration MongoDB at `mongodb://localhost:27018`.

Set `MONGO_TEST_URI` explicitly only when using a different dedicated test instance.

Make does not automatically load `.env`, so pass `MONGO_TEST_URI` explicitly
when using a different dedicated test instance.

The integration tests' MongoDB is completely separate from the application's MongoDB.

#### Local Development

In a local dev environment, the test database runs on host port `27018` with its own Docker volume.

The `test-integration` target starts it, waits for it to become healthy, and
removes it automatically after the test completes.

#### CI

In CI, `CI=true` selects the internal `test-integration-ci` path, which uses
the GitHub Actions MongoDB service without starting the local Compose service:

```bash
CI=true MONGO_TEST_URI=mongodb://localhost:27017 make test-integration
```

`MONGO_TEST_URI` must be set explicitly in CI. It intentionally has no CI fallback

## Coming from Node.js?

Go spreads responsibilities across a few focused files instead of centralizing
them in `package.json`.

| Node.js concept                    | Go equivalent in this project   |
| ---------------------------------- | ------------------------------- |
| `package.json` dependencies        | `go.mod`                        |
| `package-lock.json` or `yarn.lock` | `go.sum`                        |
| `npm run` scripts                  | `Makefile` targets              |
| `npm install`                      | `make deps` (`go mod download`) |
| `node_modules`                     | Go's module cache               |
| `main` or `bin` entry point        | Executables under `cmd/`        |
| `npm test`                         | `make test`                     |
| Custom documentation script        | `make docs`                     |
| `docker compose up` script         | `make docker-up`                |

The `Makefile` is a task runner, not a replacement for Go's toolchain. It
provides stable, memorable project commands while `go.mod`, `go.sum`, the
`go` command, and Docker continue to handle their respective concerns.

Use `make deps` when refreshing downloaded modules. Use `make tidy` when
changing imports or dependencies; it updates `go.mod` and `go.sum` to match
the packages used by the source code. `make setup` also installs the pinned
Staticcheck version used by the local hooks and CI.

Run `make setup` once after cloning to download dependencies, install tools,
and enable the repository's Git hooks. The pre-commit hook runs the local
validation commands, including Staticcheck and the generated documentation
check. The pre-push hook runs `make build` as a final compilation check.
GitHub Actions runs the type checks, linter, Staticcheck, tests, and generated
documentation check as separate jobs for pushes and pull requests, so CI
remains the source of truth when local hooks are not enabled.

## API documentation

Interactive Swagger UI (generated from Go code annotations via
[swaggo/swag](https://github.com/swaggo/swag)):

```
http://localhost:8080/docs/index.html
```

The raw OpenAPI spec is served at `http://localhost:8080/docs/doc.json`.

Swagger is intentionally unauthenticated for this demo. A private production
API should authenticate or otherwise restrict access to its documentation.

The spec is generated into `internal/api/docs/` and committed to the repo (it
is imported by `cmd/api/main.go` to register itself with the Swagger UI
handler). Regenerate it after changing any handler annotations or
request/response types:

```bash
make docs
```

> **The `mongo` container in `compose.yml` is for local development only.** It
> runs with no authentication configured, so it must never be exposed to a
> public network or used as-is in production/staging. For any non-local
> environment, point `MONGO_URI` at an authenticated, network-isolated MongoDB
> (e.g. MongoDB Atlas) instead — see [Environment variables](#environment-variables).

## Structure

```
cmd/api/main.go        - HTTP server entrypoint (composition root)
cmd/seed/main.go       - loads seed data into MongoDB
Makefile               - common build, test, docs, and Docker commands
internal/api/docs      - generated Swagger/OpenAPI spec (swag init output, committed)
internal/api/health    - GET /health route + handler
internal/api/loads     - routes, GET /loads, GET /loads/stats, and GET /load handlers, service, repository
internal/db            - MongoDB client setup (shared by cmd/api and cmd/seed)
internal/db/seeds      - seed data (loads.json)
Dockerfile             - multi-stage build for the api service
compose.yml            - api + mongo services
```

## Environment variables

| Var                  | Default  | Description                                    |
| -------------------- | -------- | ---------------------------------------------- |
| PORT                 | 8080     | HTTP listen port                               |
| MONGO_URI            | required | MongoDB connection string                      |
| MONGO_DB_NAME        | freight  | Target database name                           |
| LOAD_STATS_CACHE_TTL | 5m       | Stats cache duration; `0` disables caching     |
| CORS_ALLOWED_ORIGINS | none     | Comma-separated exact browser origins to allow |

`MONGO_URI` is read directly from the environment by both `cmd/api` and
`cmd/seed` (and passed through by `compose.yml` from `.env`), so pointing at a
remote database — staging, QA, or production (e.g. MongoDB Atlas) — from a
local run is just a matter of setting it, no code changes required:

```bash
MONGO_URI="mongodb+srv://user:pass@your-cluster.mongodb.net" MONGO_DB_NAME=freight go run ./cmd/api
```

Or copy `.env.example` to `.env` and set `MONGO_URI` there before
`docker compose up` to point the whole stack's `api` container at a remote
database instead of the local `mongo` container (in which case you likely
don't need the `mongo` service running at all).

### Local frontend CORS

The local Compose stack allows the Vite development and preview origins by
default:

```text
http://localhost:5173,http://localhost:4173
```

Override `CORS_ALLOWED_ORIGINS` in `.env` when the frontend uses a different
origin. Origins must include the scheme and port, must not contain a path, and
must be separated by commas. Wildcard origins are not accepted. Restart the API
with `make docker-up` after changing this value. When the variable is unset
outside Compose, cross-origin browser access is disabled.

## Seeding data

With the `mongo` container running, load `internal/db/seeds/loads.json` into the
`loads` collection (drops and replaces any existing documents, and ensures a
unique index on `id`):

```bash
docker compose up -d mongo
MONGO_URI=mongodb://localhost:27017 go run ./cmd/seed
```

## GET /loads

Implements the AG Grid server-side row model contract.

| Param          | Example                                                                                           | Notes                                                                                                                                                                                                                                                                                                                          |
| -------------- | ------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `startRow`     | `0`                                                                                               | default `0`; alias: `offset` (ignored if `startRow` is set)                                                                                                                                                                                                                                                                    |
| `endRow`       | `25`                                                                                              | default `startRow + 25`, capped at 500 rows/page; must not be less than `startRow`. If equal to `startRow`, treated leniently as a 1-row page (`endRow` bumped to `startRow + 1`). Alias: `limit` (computes `endRow` as `startRow + limit`; ignored if `endRow` is set)                                                        |
| `quickSearch`  | `chicago`                                                                                         | matches every field (`id`, `companyName`, `origin`, `destination`, `weight`, `equipmentType`, `date`, `price`, `distance`, `status`), mirroring the client's AG Grid quick filter. Alias: `q` (ignored if `quickSearch` is set)                                                                                                |
| `sortModel`    | `[{"colId":"price","sort":"desc"}]`                                                               | single-column sort; JSON-encoded, URL-escaped. Alias: `sortBy`/`sortDir` (e.g. `sortBy=price&sortDir=desc`; `sortDir` defaults to `asc`; ignored if `sortModel` is set)                                                                                                                                                        |
| `filterModel`  | `{"status":{"filterType":"text","type":"equals","filter":"Available"}}`                           | per-field AG Grid simple filter model (`text`, `number`, `date`, `set`); JSON-encoded, URL-escaped                                                                                                                                                                                                                             |
| any load field | `status=Available&status=In+Transit`, `companyName=Swift+Transport`, `price=1082`, `id=LD-000590` | shorthand alias for any load field (`id`, `companyName`, `origin`, `destination`, `weight`, `equipmentType`, `date`, `price`, `distance`, `status`); a single value is an exact match, repeated values are "one of"; numeric fields are coerced automatically; ignored per-field if `filterModel` already specifies that field |

Response:

```json
{
  "requestURL": "/loads?quickSearch=chicago&startRow=0&endRow=25",
  "query": {
    "quickSearch": "",
    "sortModel": [],
    "filterModel": {},
    "startRow": 0,
    "endRow": 25
  },
  "meta": {
    "numResults": 4175,
    "numTotal": 100000,
    "pageSize": 25,
    "numPages": 167,
    "currentPage": 1,
    "hasNextPage": true,
    "nextPageURL": "/loads?quickSearch=chicago&startRow=25&endRow=50",
    "responseTimeMs": 48,
    "timestamp": "2026-09-12T03:24:44.397635387Z"
  },
  "rows": [{ "id": "LD-000001", "companyName": "J.B. Hunt", "...": "..." }],
  "lastRow": 4175
}
```

`meta.numTotal` is the full collection size, ignoring filters/quicksearch.
`meta.numResults` (same value as `lastRow`) is the count matching the
current filters/quicksearch. `meta.pageSize` is derived from `endRow - startRow`.
`meta.numPages` is `ceil(filteredRows / pageSize)`. `meta.currentPage` is
1-indexed, derived from `startRow / pageSize`. `meta.nextPageURL` is the path
(no host) plus query string for the next page, preserving all current params;
`null` when `hasNextPage` is `false`.

`requestURL` is the incoming path+query exactly as received (raw, not
re-encoded) — useful for spotting URL-encoding mistakes at a glance.
`meta.hasNextPage` is whether more filtered rows exist beyond the current page.
`meta.responseTimeMs` is server-side processing time for the request.
`meta.timestamp` is the server time (UTC, RFC3339) the response was generated.
`query` echoes back the effective params actually applied, including defaults
used when a param was omitted from the request.

`lastRow` is the total count of rows matching the current filters/quicksearch
(not the full collection size). Errors are returned as:

```json
{ "error": { "code": "LOAD_QUERY_FAILED", "message": "..." } }
```

## GET /loads/stats

Returns aggregate counts for loads matching `quickSearch` or its `q` alias.
Search matching uses the same case-insensitive, all-field semantics as
`GET /loads`; `quickSearch` wins when both aliases are present. Other filtering,
sorting, and pagination parameters do not apply.

```json
{
  "requestURL": "/loads/stats?q=chicago",
  "query": {
    "quickSearch": "chicago"
  },
  "meta": {
    "numTotal": 100000,
    "numResults": 1248,
    "responseTimeMs": 68,
    "timestamp": "2026-09-13T07:40:43.281552398Z"
  },
  "stats": {
    "totals": {
      "equipmentType": [
        { "label": "Flatbed", "value": 33334 },
        { "label": "Reefer", "value": 33333 },
        { "label": "Van", "value": 33333 }
      ],
      "status": [
        { "label": "Available", "value": 33334 },
        { "label": "In Transit", "value": 33333 },
        { "label": "Delivered", "value": 33333 }
      ]
    }
  }
}
```

The category labels and ordering are stable; a category with no matching loads
has a value of zero. `meta.numTotal` is always the full collection count, while
`meta.numResults` and the category values reflect the effective quick search.
Unexpected category values remain part of `meta.numResults` but are not added
to the category arrays. `query.quickSearch` is always present, including as an
empty string when no search was supplied.

Successful aggregate results are cached by effective search string in each API
process for `LOAD_STATS_CACHE_TTL` (default `5m`). The least recently used entry
is evicted after 50 distinct searches. Request timing and timestamp metadata
are generated fresh for every response. Set the TTL to `0` to disable caching.

## GET /load/{id} and GET /load?id={id}

Fetch a single load by its `id` field. Either URL form works identically.

```json
{
  "requestURL": "/load/LD-000001",
  "meta": {
    "responseTimeMs": 18,
    "timestamp": "2026-09-12T04:48:50.934689669Z"
  },
  "result": {
    "id": "LD-000001",
    "companyName": "J.B. Hunt",
    "origin": "Richmond, VA",
    "destination": "Las Vegas, NV",
    "weight": 32000,
    "equipmentType": "Van",
    "date": "2024-07-05",
    "price": 5660,
    "distance": 2384,
    "status": "Available"
  }
}
```

If no load matches the given id, responds `404`:

```json
{
  "error": {
    "code": "LOAD_NOT_FOUND",
    "message": "No load was found with the given id."
  }
}
```

If no id is provided at all, responds `400` with code `LOAD_ID_REQUIRED`.
