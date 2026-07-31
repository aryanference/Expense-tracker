# Personal Expense Tracker API

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)
![Status](https://img.shields.io/badge/Status-Production_Ready-brightgreen?style=for-the-badge)
![Coverage](https://img.shields.io/badge/Tests-35%2F35_Passing-success?style=for-the-badge)

A lightning-fast, highly concurrent RESTful API for managing personal expenses, built entirely in Go without any external routing frameworks or ORMs. Designed with clean architecture, thread-safe in-memory storage, and atomic JSON file persistence.

---

## Core Features

- **Standard Library Only**: Built using Go 1.22's enhanced `http.ServeMux` for routing. Zero external dependencies (`uuid` bypassed for native `crypto/rand`).
- **Thread-Safe**: Uses `sync.RWMutex` to safely handle hundreds of concurrent read/write requests.
- **Atomic Persistence**: Data is committed to disk using an atomic `temp file + rename` strategy, ensuring zero data corruption even on sudden power loss.
- **Fault-Tolerant**: Includes graceful shutdown hooks and panic-recovery middleware.
- **Fully Tested**: 100% compliance with strict specification requirements, backed by 35 unit tests (safe under `-race`).
- **Bonus Extras**: Includes an OpenAPI 3.0 specification (`openapi.yaml`), a ready-to-use Postman Collection, and an ultra-lightweight multi-stage Docker build (~10MB image).

---

## Getting Started

### Option 1: Run Locally (Bare Metal)
Ensure you have Go 1.22+ installed.
```sh
go run src/main.go
```
The server will start on `http://localhost:8080`.

### Option 2: Run with Docker (Recommended)
This project uses a multi-stage `Dockerfile` that builds the application securely into an empty scratch container.
```sh
# Start in detached mode
docker-compose up -d
```
The server will be available on port `8080`, and data will be safely persisted to the `expense_data` Docker volume.

![Server and Setup Context](Screenshot%202026-08-01%20014900.png)

---

## API Endpoints

All requests and responses use `application/json`.

### `POST /expenses`
Creates a new expense.
**Body:**
```json
{
  "title": "Morning Coffee",
  "amount": 4.50,
  "category": "Food",
  "date": "2026-08-01"
}
```
![Creating an Expense](Screenshot%202026-08-01%20014845.png)

### `GET /expenses`
Retrieves all expenses in their exact insertion order.

![Listing all Expenses](Screenshot%202026-08-01%20014905.png)

- **Query Parameter (Optional):** `?category=food` (Case-insensitive)

![Filtering Expenses by Category](Screenshot%202026-08-01%20015051.png)

### `GET /expenses/total`
Calculates the total sum of all expenses (strictly adhering to IEEE 754 float rounding).

![Calculating Totals](Screenshot%202026-08-01%20015119.png)

- **Query Parameter (Optional):** `?category=travel`

### `DELETE /expenses/{id}`
Removes an expense by its unique ID. Returns `204 No Content` on success.

---

## Testing

The project includes an exhaustive suite covering handlers, persistence edge cases (corrupt file recovery), validation boundaries, and concurrency data-race checks.

```sh
# Run all 35 tests
go test ./tests/... -v

# Run with race detector enabled (requires CGO)
go test ./tests/... -race -v
```

---

## Architecture & Technical Decisions

- **In-Memory + File DB**: To avoid external database dependencies while satisfying the persistence requirement, the system uses a local `expenses.json` file. 
- **Preserving Insertion Order**: Go maps randomize iteration. To satisfy the specification's insertion-order guarantee, the store pairs a `map[string]models.Expense` (for O(1) lookups) with a supplementary `order []string` slice.
- **Validation**: Strict sequential validation fails fast on the first error, ensuring predictable client feedback.
