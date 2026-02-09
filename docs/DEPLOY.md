# Deployment Guide (idatariver-finapi)

This project contains:
- **API server** (`cmd/api`) on port **8080**
- **Worker** (`cmd/worker`) for scheduled health checks
- **Migrator** (`cmd/migrate`) that applies SQL in `/migrations`
- **Postgres schema + seed data** in `/migrations`

## Quick start (Docker Compose)

```bash
docker compose up --build
```

What happens:
- Postgres starts
- `migrate` applies migrations + seed data
- `api` starts on `http://localhost:8080`
- `worker` starts and runs health checks every hour (default: `:05`)

## Environment variables

All services accept:

- `DATABASE_URL` (required for non-compose usage)
  - Example: `postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable`

API:
- `API_ADDR` (default `:8080`)
- `GIN_MODE` (default `release`)

Worker:
- `HEALTH_CRON` (default `0 5 * * * *`)
  - cron with seconds, e.g. every 10 minutes: `0 */10 * * * *`

## Demo API keys

Seed data creates a default tenant and two demo keys:

- User key (header `X-API-Key`): `devkey`
- Admin key (header `X-Admin-Key`): `adminkey`

> Keys are stored as SHA256 hex in DB.

## Local run (without Docker)

1) Start Postgres and create DB `idatariver`
2) Apply migrations:

```bash
export DATABASE_URL='postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable'

go run ./cmd/migrate
```

3) Run API:

```bash
go run ./cmd/api
```

4) Run worker:

```bash
go run ./cmd/worker
```

## Adding real data

The schema includes `ts_point` and `daily_metrics`. This repo focuses on the **API/tenant/SLA/health/notify** skeleton.
To make the API return real values, insert your ingested prices/yields/fx rates into:
- `ts_point(instrument_id, ts, close)`
- `daily_metrics(instrument_id, d, ...metrics...)

You can start with a simple manual insert to validate end-to-end.
