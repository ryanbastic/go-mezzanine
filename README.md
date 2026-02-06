# go-mezzanine

A Go implementation of [Uber's Schemaless](https://www.uber.com/blog/schemaless-part-one-mysql-datastore/) datastore — an immutable, versioned, cell-based storage system backed by PostgreSQL.

Mezzanine stores data as JSON cells addressed by three coordinates: **row key** (UUID), **column name** (string), and **ref key** (version number). It supports hash-based sharding, secondary indexes, an event-driven trigger framework, and a circuit breaker for resilience.

## Architecture

```
                         HTTP Clients
                              │
                    ┌─────────▼──────────┐
                    │   Chi Router /v1   │
                    │  (middleware stack) │
                    └─────────┬──────────┘
                              │
                     ┌────────▼────────┐
                     │  Shard Router   │
                     │  FNV-32a hash   │
                     └────────┬────────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
      ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
      │ cells_0000 │    │ cells_0001 │···│ cells_0063 │
      └─────┬──────┘    └─────┬──────┘   └─────┬──────┘
            └─────────────────┼─────────────────┘
                              │
                       ┌──────▼──────┐
                       │ PostgreSQL  │
                       └─────────────┘

      Trigger Watchers (per shard × column) ──▶ Handlers
```

| Package | Purpose |
|---|---|
| `cmd/mezzanine` | Entry point and server bootstrap |
| `internal/cell` | Core data model (`Cell`, `CellRef`, `WriteCellRequest`) |
| `internal/shard` | Deterministic shard routing via FNV-32a |
| `internal/storage` | PostgreSQL persistence and migrations |
| `internal/api` | HTTP handlers and middleware |
| `internal/index` | Secondary index support |
| `internal/trigger` | Event-driven trigger framework with checkpointing |
| `internal/circuitbreaker` | Circuit breaker resilience pattern |
| `internal/config` | Environment-based configuration |

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 18+ (or Docker)

### Run with Docker Compose

```bash
# Start PostgreSQL
docker compose up -d

# Build and run
go run ./cmd/mezzanine
```

The server starts on port `8080` by default. Migrations run automatically on startup, creating per-shard tables and indexes.

### Configuration

All settings are configured via environment variables:

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/mezzanine?sslmode=disable` | PostgreSQL connection string |
| `PORT` | `8080` | HTTP server port |
| `NUM_SHARDS` | `64` | Number of data shards |
| `LOG_LEVEL` | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `TRIGGER_POLL_INTERVAL` | `100ms` | How often triggers poll for new cells |
| `TRIGGER_BATCH_SIZE` | `100` | Max cells processed per trigger poll |
| `CB_MAX_FAILURES` | `5` | Circuit breaker failure threshold |
| `CB_RESET_TIMEOUT` | `30s` | Circuit breaker recovery timeout |

## API Reference

All endpoints are under the `/v1` prefix.

### Health Check

```
GET /v1/health
```

```bash
curl http://localhost:8080/v1/health
```

### Write a Cell

```
POST /v1/cells
```

Creates an immutable cell. Each write produces a new version — existing cells are never modified.

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `row_key` | UUID | yes | Row identifier |
| `column_name` | string | yes | Column identifier |
| `ref_key` | int64 | yes | Version number |
| `body` | object | yes | Arbitrary JSON payload |

**Example:**

```bash
curl -X POST http://localhost:8080/v1/cells \
  -H "Content-Type: application/json" \
  -d '{
    "row_key": "550e8400-e29b-41d4-a716-446655440000",
    "column_name": "profile",
    "ref_key": 1,
    "body": {
      "name": "Alice",
      "email": "alice@example.com",
      "city": "San Francisco"
    }
  }'
```

**Response** `201 Created`:

```json
{
  "added_id": 1,
  "row_key": "550e8400-e29b-41d4-a716-446655440000",
  "column_name": "profile",
  "ref_key": 1,
  "body": {
    "name": "Alice",
    "email": "alice@example.com",
    "city": "San Francisco"
  },
  "created_at": "2026-02-06T12:00:00Z"
}
```

Write a second version of the same cell:

```bash
curl -X POST http://localhost:8080/v1/cells \
  -H "Content-Type: application/json" \
  -d '{
    "row_key": "550e8400-e29b-41d4-a716-446655440000",
    "column_name": "profile",
    "ref_key": 2,
    "body": {
      "name": "Alice",
      "email": "alice@newdomain.com",
      "city": "New York"
    }
  }'
```

### Get a Cell (exact version)

```
GET /v1/cells/{row_key}/{column_name}/{ref_key}
```

Retrieves a specific version of a cell.

```bash
curl http://localhost:8080/v1/cells/550e8400-e29b-41d4-a716-446655440000/profile/1
```

**Response** `200 OK`:

```json
{
  "added_id": 1,
  "row_key": "550e8400-e29b-41d4-a716-446655440000",
  "column_name": "profile",
  "ref_key": 1,
  "body": {
    "name": "Alice",
    "email": "alice@example.com",
    "city": "San Francisco"
  },
  "created_at": "2026-02-06T12:00:00Z"
}
```

Returns `404` if the cell does not exist.

### Get Latest Cell

```
GET /v1/cells/{row_key}/{column_name}
```

Retrieves the latest version (highest `ref_key`) of a cell.

```bash
curl http://localhost:8080/v1/cells/550e8400-e29b-41d4-a716-446655440000/profile
```

### Get Row

```
GET /v1/cells/{row_key}
```

Retrieves all columns for a given row key (latest version of each).

```bash
curl http://localhost:8080/v1/cells/550e8400-e29b-41d4-a716-446655440000
```

**Response** `200 OK`:

```json
[
  {
    "added_id": 2,
    "row_key": "550e8400-e29b-41d4-a716-446655440000",
    "column_name": "profile",
    "ref_key": 2,
    "body": {"name": "Alice", "email": "alice@newdomain.com", "city": "New York"},
    "created_at": "2026-02-06T12:01:00Z"
  },
  {
    "added_id": 3,
    "row_key": "550e8400-e29b-41d4-a716-446655440000",
    "column_name": "settings",
    "ref_key": 1,
    "body": {"theme": "dark", "notifications": true},
    "created_at": "2026-02-06T12:02:00Z"
  }
]
```

### Query a Secondary Index

```
GET /v1/index/{index_name}/{shard_key}
```

Queries a secondary index by its shard key.

```bash
curl http://localhost:8080/v1/index/user_by_email/550e8400-e29b-41d4-a716-446655440000
```

**Response** `200 OK`:

```json
[
  {
    "added_id": 1,
    "shard_key": "550e8400-e29b-41d4-a716-446655440000",
    "row_key": "661f9511-f30c-52e5-b827-557766551111",
    "body": {"email": "alice@example.com"},
    "created_at": "2026-02-06T12:00:00Z"
  }
]
```

### Error Responses

All errors return a JSON body:

```json
{"error": "cell not found"}
```

| Status | Meaning |
|---|---|
| `400` | Invalid request (missing fields, bad UUID, etc.) |
| `404` | Cell or index entry not found |
| `500` | Internal server error |

Every response includes an `X-Request-ID` header (auto-generated UUID) for tracing.

## Data Model

Mezzanine uses **three-dimensional cell addressing**:

```
              column_name
              ┌──────────┬──────────┬──────────┐
              │ profile  │ settings │ billing  │
   row_key    ├──────────┼──────────┼──────────┤
   (UUID)     │ ref 1    │ ref 1    │ ref 1    │
              │ ref 2    │          │ ref 2    │
              │ ref 3    │          │          │
              └──────────┴──────────┴──────────┘
                           ref_key (version)
```

- **Row key** — UUID identifying a logical entity (e.g., a user)
- **Column name** — A named attribute group (e.g., "profile", "settings")
- **Ref key** — An integer version number, allowing immutable history

Cells are immutable: writing with a higher `ref_key` does not overwrite previous versions.

## Sharding

Row keys are deterministically mapped to shards using FNV-32a hashing:

```
shard_id = fnv32a(row_key) % num_shards
```

Each shard has its own PostgreSQL table (`cells_0000` through `cells_0063`), providing natural partitioning. All versions of a given row key live on the same shard.

## Triggers

Triggers react to cell writes asynchronously. The framework polls each shard for new cells (tracked via `added_id`) and invokes registered handler functions.

Handlers must be **idempotent** — they may be called more than once for the same cell if a failure occurs before the checkpoint advances.

```go
triggerRegistry.Register("profile", func(ctx context.Context, c cell.Cell) error {
    log.Printf("new profile write: row=%s ref=%d", c.RowKey, c.RefKey)
    return nil
})
```

## Secondary Indexes

Secondary indexes denormalize cell data into separate per-shard tables for efficient lookup by fields other than the row key. Indexes are populated via triggers and defined with:

- **Name** — Table name prefix (e.g., `user_by_email`)
- **Source column** — The column name that triggers index updates
- **Shard key field** — JSON field used for index sharding
- **Fields** — JSON fields to copy into the index

## License

See [LICENSE](LICENSE) for details.
