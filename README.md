# Neobank API

[![Go Version](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go)](https://go.dev/)
[![Database](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql)](https://www.postgresql.org/)
[![Architecture](https://img.shields.io/badge/Style-Clean%20Architecture-111827)](#architecture)
[![SQL Safety](https://img.shields.io/badge/SQL-sqlc%20type--safe-0EA5E9)](https://sqlc.dev/)

Backend banking API that showcases production-minded engineering in Go: strong domain boundaries, transaction-safe transfer logic, compile-time SQL checks, and financial precision without floating-point risk.

## Portfolio Snapshot

This project is intentionally built to demonstrate senior backend competencies that matter in fintech and high-integrity systems.

What this repository highlights:

- System design with clear responsibility boundaries
- Correctness-first transfer orchestration under concurrency
- Practical reliability patterns (locking, optimistic concurrency, retries, idempotency)
- Secure auth and access control at API boundaries
- Maintainable code organization suitable for team scaling

## Engineering Decisions That Matter

### 1) SQL as Source of Truth with Compile-Time Validation

- SQL lives in db/queries and is generated via sqlc.
- Query contracts are validated at build-time, reducing runtime query mismatch bugs.
- Repositories wrap generated query code to keep domain language clean.

Why this choice:

- Better correctness and traceability than dynamic query builders for a data-sensitive domain.

### 2) Money Uses Decimal, Not Float

- Monetary values use DECIMAL(19,4) in PostgreSQL.
- Go values use shopspring/decimal.

Why this choice:

- Avoids precision loss and rounding drift common with float-based arithmetic.

### 3) Transfer Path Optimized for Data Integrity

Transfer execution includes all of the following:

1. Consistent lock order (sorted account IDs) to lower deadlock risk.
2. Pessimistic row locks with FOR UPDATE during balance mutation.
3. Atomic write of transfer + debit transaction + credit transaction.
4. Optimistic locking on account version for balance updates.
5. Retry policy for serialization/deadlock style database conflicts.
6. Idempotency key support to prevent duplicate business effects.

Why this choice:

- Preserves ledger correctness under concurrent requests and retry storms.

### 4) Clean Layering for Long-Term Changeability

The architecture follows:

```text
HTTP Handlers
    -> Services
        -> Repositories
            -> sqlc Generated Queries
                -> PostgreSQL
```

Why this choice:

- Allows business logic to evolve independently of transport or persistence details.

## Tech Stack

| Area | Choice |
| --- | --- |
| Language | Go 1.24 |
| Router | go-chi/chi |
| Database | PostgreSQL 15 |
| Driver | pgx/v5 |
| SQL Generation | sqlc |
| Validation | go-playground/validator |
| Money | shopspring/decimal |
| Auth | JWT (golang-jwt/jwt) |
| Config | viper |

## Feature Coverage

- Authentication: register, login, token refresh, logout, profile read/update
- Accounts: create, list, read, balance view
- Transfers: create transfer, fetch transfer, account-level transfer history
- Transactions: account-level listing (with filters), transaction detail lookup
- Middleware: request ID, logging, panic recovery, timeout, CORS, security headers, rate limiting

## API Surface

Base URL:

- http://localhost:8080

OpenAPI contract:

- api/openapi.yaml (aligned with current router paths, auth requirements, and DTO payloads)

Public routes:

| Method | Path | Purpose |
| --- | --- | --- |
| GET | /health | Health check |
| POST | /api/v1/auth/register | Create user |
| POST | /api/v1/auth/login | Get access/refresh tokens |
| POST | /api/v1/auth/refresh | Rotate token pair |
| POST | /api/v1/auth/logout | Stateless logout |

Logout semantics:

- Logout is stateless and always returns 204.
- The client must discard access and refresh tokens after logout.
- The server does not keep a token revocation store in this version.
- A refresh token issued before logout remains valid until expiry.

Protected routes (Authorization: Bearer ACCESS_TOKEN):

| Method | Path | Purpose |
| --- | --- | --- |
| GET | /api/v1/auth/profile | Current user profile |
| PUT | /api/v1/auth/profile | Update profile |
| POST | /api/v1/accounts/ | Create account |
| GET | /api/v1/accounts/ | List user accounts |
| GET | /api/v1/accounts/{id} | Account details |
| GET | /api/v1/accounts/{id}/balance | Balance snapshot |
| GET | /api/v1/accounts/{id}/transfers | Transfer history |
| GET | /api/v1/accounts/{id}/transactions | Transaction history |
| POST | /api/v1/transfers/ | Execute transfer |
| GET | /api/v1/transfers/{id} | Transfer details |
| GET | /api/v1/transactions/{id} | Transaction details |

Transaction list filters:

- type: deposit | withdrawal | transfer_in | transfer_out
- from: RFC3339 or YYYY-MM-DD
- to: RFC3339 or YYYY-MM-DD
- limit: integer, default 50
- offset: integer, default 0

Transfer history pagination:

- limit: integer, default 20, min 1, max 200
- offset: integer, default 0, min 0

## Run Locally

### Prerequisites

- Go 1.24+
- Docker + Docker Compose
- sqlc
- golang-migrate
- golangci-lint (optional)

### 1) Start Database

```bash
docker-compose up -d
```

### 2) Export Environment Variables

```bash
export DATABASE_URL="postgres://banking:banking@localhost:5432/banking?sslmode=disable"
export JWT_SECRET="replace-with-strong-secret"
```

Optional runtime tuning:

```bash
export SERVER_HOST="0.0.0.0"
export SERVER_PORT="8080"

export DATABASE_MAX_CONNS="20"
export DATABASE_MIN_CONNS="2"
export DATABASE_MAX_CONN_LIFETIME="30m"
export DATABASE_MAX_CONN_IDLE_TIME="10m"
export DATABASE_HEALTH_CHECK_PERIOD="1m"

export JWT_EXPIRY="15m"
export JWT_REFRESH_TTL="168h"

export LOGGING_LEVEL="info"
export LOGGING_FORMAT="json"

export RATE_LIMITER_GENERAL_PER_MINUTE="100"
export RATE_LIMITER_LOGIN_PER_MINUTE="5"
export RATE_LIMITER_TRANSFER_PER_MINUTE="10"
```

### 3) Apply Migrations and Generate SQL Code

```bash
make migrate-up
make sqlc
```

### 4) Run API

```bash
make run
```

Quick check:

```bash
curl http://localhost:8080/health
```

## Build and Quality Commands

```bash
make build
make test
make lint
```

Binary output:

- bin/api

## Example Request Flow

### Register

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "supersecret123",
    "full_name": "Alice Example"
  }'
```

### Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "supersecret123"
  }'
```

### Create Account

```bash
curl -X POST http://localhost:8080/api/v1/accounts/ \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "account_type": "checking",
    "currency": "USD"
  }'
```

### Create Transfer with Idempotency

```bash
curl -X POST http://localhost:8080/api/v1/transfers/ \
  -H "Authorization: Bearer ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: transfer-20260324-001" \
  -d '{
    "from_account_id": "11111111-1111-4111-8111-111111111111",
    "to_account_id": "22222222-2222-4222-8222-222222222222",
    "amount": "125.50",
    "currency": "USD",
    "description": "Invoice payment"
  }'
```

## Data Model Summary

- users: authentication identity and profile data
- accounts: customer accounts with balance, status, and concurrency version
- transactions: immutable account activity records
- transfers: transfer intent and lifecycle between two accounts

## Repository Layout

```text
api/                 OpenAPI scaffold
cmd/api/             App entrypoint
db/migrations/       SQL migrations
db/queries/          sqlc query files
internal/auth/       JWT logic and middleware helpers
internal/config/     Environment configuration
internal/container/  Dependency composition root
internal/db/         Generated sqlc code
internal/domain/     Entities, value objects, domain errors
internal/handler/    HTTP handlers, middleware, DTOs
internal/repository/ Repository contracts and adapters
internal/service/    Use-case orchestration
```

## What This Demonstrates in an Interview

- Ability to model financial invariants and concurrency risks
- Practical separation of concerns without over-abstraction
- Balance of explicit SQL control and code generation productivity
- Defensive API behavior through validation, auth, and structured errors
- Production-minded defaults for reliability and maintainability

## Future Enhancements

- OpenAPI contract linting/validation in CI
- Integration test suite with containerized PostgreSQL
- Metrics/tracing and SLA-oriented observability
- CI pipeline with migration validation and quality gates
- Role-based authorization and richer audit trail support

## License

No license file is currently included.
