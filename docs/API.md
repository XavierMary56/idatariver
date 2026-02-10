# API Documentation

Base URL (docker): `http://localhost:8080`

Authentication:
- User endpoints: header `X-API-Key: <key>`
- Admin endpoints: header `X-Admin-Key: <key>`

Error responses:
- JSON body: `{ "error": "message" }`
- Status: `4xx` for validation/auth, `5xx` for server errors

Seed demo keys:
- `devkey` (user)
- `adminkey` (admin)

---

## GET /healthz

Returns server liveness.

---

## GET /api/v1/meta/fields

Returns available field keys and aliases.

Example:
```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/meta/fields'
```

---

## GET /api/v1/meta/symbols

Returns symbols + sources + freshness + SLA + status (tenant-aware).

Query params:
- `types` (optional, csv): `index,rate,fx`
- `active` (optional): `true` (default) or `false`

Example:
```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/meta/symbols?types=index,rate'
```

---

## GET /api/v1/market/compare/latest

Returns **latest** metrics horizontally for multiple symbols.

Query params:
- `symbols` (required, csv): up to 20
- `fields` (optional, csv): strict output fields
- `include_close` (optional): `true` (default) / `false`
- `sort_by` (optional): `symbol|close|ret_1d|ret_5d|ret_20d|ma_20|ma_60|vol_20d|mdd_252d`
- `order` (optional): `asc|desc` (default `desc`)

Example (strict fields + sort):
```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/market/compare/latest?symbols=SP500,NASDAQ,US10Y&fields=symbol,close,ret_20d,vol_20d&sort_by=ret_20d&order=desc'
```

---

## GET /api/v1/health/data

Returns data health summary and worst stale symbols for current tenant.

Query params:
- `types` (optional, csv)
- `worst` (optional int): default 10
- `include_items` (optional bool): default false

Example:
```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/health/data?include_items=false&worst=10'
```

---

# Admin API

## GET /api/v1/admin/sla
List SLA overrides for the current tenant.

```bash
curl -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/sla'
```

## POST /api/v1/admin/sla
Upsert a single SLA override.

```bash
curl -X POST -H 'X-Admin-Key: adminkey' -H 'Content-Type: application/json' \
  -d '{"symbol":"US10Y","sla_hours":96}' \
  'http://localhost:8080/api/v1/admin/sla'
```

## DELETE /api/v1/admin/sla?symbol=US10Y
Delete an SLA override.

## POST /api/v1/admin/sla/batch
Batch upsert (partial success; returns updated + failed).

```bash
curl -X POST -H 'X-Admin-Key: adminkey' -H 'Content-Type: application/json' \
  -d '{"items":[{"symbol":"SP500","sla_hours":24},{"symbol":"BAD","sla_hours":10}]}' \
  'http://localhost:8080/api/v1/admin/sla/batch'
```

## GET /api/v1/admin/sla.csv
Export tenant SLA as CSV.

## POST /api/v1/admin/sla.csv
Import tenant SLA from CSV (strict validation; returns line-numbered errors).

```bash
curl -X POST -H 'X-Admin-Key: adminkey' --data-binary @instrument_sla.csv \
  'http://localhost:8080/api/v1/admin/sla.csv'
```

---

## Snooze (tenant-aware)

### GET /api/v1/admin/health/snooze
Returns current snooze status.

### POST /api/v1/admin/health/snooze
Body: `{ "minutes": 120, "reason": "deploy" }`

### DELETE /api/v1/admin/health/snooze
Cancels active snooze.

---

## Notification config (tenant-aware)

### GET /api/v1/admin/notify

### PUT /api/v1/admin/notify

Body example:
```json
{
  "enabled": true,
  "webhook_urls": ["https://example.com/webhook"],
  "telegram_bot_token": "123:ABC",
  "telegram_chat_id": "-1001234567890",
  "cooldown_min": 180
}
```
