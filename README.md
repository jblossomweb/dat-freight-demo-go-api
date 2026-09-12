# dat-freight-demo-api

Go + MongoDB REST API backing the DAT freight demo SPA. Implements the AG Grid
server-side row model contract (pagination, sort, filter, quicksearch) for
freight loads, plus a single-load lookup endpoint.

## Requirements

- Docker (with Compose v2)
- Go 1.27+ (only needed for local editor tooling; the app itself runs in Docker)

## Run

```bash
cp .env.example .env
docker compose up --build
```

> Whenever you change Go code, re-run `docker compose up --build` — the binary
> is compiled into the image, so `docker compose up` alone won't pick up edits.

Verify:

```bash
curl -i http://localhost:8080/health
```

Expect `200 OK` with `{"status":"ok","mongo":"ok"}` once both containers are healthy.

## Structure

```
cmd/api/main.go        - HTTP server entrypoint (composition root)
cmd/seed/main.go       - loads seed data into MongoDB
internal/api/health    - GET /health route + handler
internal/api/loads     - routes.go, GET /loads and GET /load handlers, service.go, repository.go
internal/db            - MongoDB client setup (shared by cmd/api and cmd/seed)
internal/db/seeds      - seed data (loads.json)
Dockerfile             - multi-stage build for the api service
compose.yml            - api + mongo services
```

## Environment variables

| Var           | Default               | Description               |
| ------------- | --------------------- | ------------------------- |
| PORT          | 8080                  | HTTP listen port          |
| MONGO_URI     | mongodb://mongo:27017 | MongoDB connection string |
| MONGO_DB_NAME | freight               | Target database name      |

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
