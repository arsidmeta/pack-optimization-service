# Pack Optimization Service

A Go HTTP API that calculates the optimal pack combination to fulfil a customer order.

## Rules

1. Only whole packs can be sent. Packs cannot be broken open.
2. Send the least number of **items** to fulfil the order (minimise overshoot).
3. Within rule 2, send as few **packs** as possible.

## Quick Start

### Run locally (requires Go 1.21+)

```bash
go run .
```

### Run with Docker

```bash
docker build -t pack-optimization-service .
docker run -p 8080:8080 pack-optimization-service
```

### Run with Docker Compose (recommended — persists pack size changes)

```bash
docker compose up --build
```

Open **http://localhost:8080** in your browser.

---

## Makefile Commands

| Command            | Description                          |
|--------------------|--------------------------------------|
| `make run`         | Run the server locally               |
| `make build`       | Compile the binary                   |
| `make test`        | Run all unit tests                   |
| `make docker-build`| Build the Docker image               |
| `make docker-run`  | Run the container on port 8080       |
| `make docker-up`   | Start with Docker Compose            |
| `make docker-down` | Stop Docker Compose                  |

---

## API Reference

### `POST /api/calculate`

Calculate the optimal packs for an order.

**Request**
```json
{ "items": 12001 }
```

**Response**
```json
{
  "ordered_items": 12001,
  "total_items": 12250,
  "packs": [
    { "size": 5000, "quantity": 2 },
    { "size": 2000, "quantity": 1 },
    { "size": 250,  "quantity": 1 }
  ]
}
```

---

### `GET /api/packs`

List the current pack sizes.

**Response**
```json
[
  { "size": 5000 },
  { "size": 2000 },
  { "size": 1000 },
  { "size": 500 },
  { "size": 250 }
]
```

---

### `POST /api/packs`

Add a new pack size. Changes are persisted to `packs.json` and survive restarts.

**Request**
```json
{ "size": 300 }
```

---

### `DELETE /api/packs?size=300`

Remove a pack size.

---

## Running Tests

```bash
make test
# or
go test ./... -v
```

Includes an edge-case test with pack sizes `[23, 31, 53]` and order `500,000`:

| Pack | Quantity |
|------|----------|
| 53   | 9429     |
| 31   | 7        |
| 23   | 2        |

Total shipped: **500,000 items** in **9,438 packs**.

---

## Pack Size Persistence

Pack sizes are stored in `packs.json` in the working directory. When running via
Docker Compose, this file is mounted as a volume so changes persist across container
restarts. On first run, the default sizes `[250, 500, 1000, 2000, 5000]` are written
to the file automatically.
