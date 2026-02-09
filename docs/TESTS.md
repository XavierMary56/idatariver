# Interface Test Examples (curl)

Assuming docker compose is running:
- API: `http://localhost:8080`
- Demo keys: `devkey` (user), `adminkey` (admin)

## 1) Health

```bash
curl 'http://localhost:8080/healthz'
```

## 2) List fields

```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/meta/fields'
```

## 3) List symbols + status

```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/meta/symbols?types=index,rate,fx'
```

## 4) Latest horizontal compare (strict fields)

```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/market/compare/latest?symbols=SP500,NASDAQ,US10Y&fields=symbol,close,ret_1d,vol_20d&sort_by=ret_1d&order=desc'
```

## 5) Data health report

```bash
curl -H 'X-API-Key: devkey' \
  'http://localhost:8080/api/v1/health/data?include_items=false&worst=10'
```

## 6) Admin: set SLA for a symbol

```bash
curl -X POST -H 'X-Admin-Key: adminkey' -H 'Content-Type: application/json' \
  -d '{"symbol":"SP500","sla_hours":12}' \
  'http://localhost:8080/api/v1/admin/sla'
```

## 7) Admin: list SLA

```bash
curl -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/sla'
```

## 8) Admin: export SLA CSV

```bash
curl -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/sla.csv'
```

## 9) Admin: import SLA CSV

```bash
cat <<'CSV' > /tmp/sla.csv
symbol,sla_hours
SP500,10
US10Y,24
CSV

curl -X POST -H 'X-Admin-Key: adminkey' --data-binary '@/tmp/sla.csv' \
  'http://localhost:8080/api/v1/admin/sla.csv'
```

## 10) Admin: snooze notifications (2 hours)

```bash
curl -X POST -H 'X-Admin-Key: adminkey' -H 'Content-Type: application/json' \
  -d '{"minutes":120,"reason":"deploy"}' \
  'http://localhost:8080/api/v1/admin/health/snooze'
```

## 11) Admin: get snooze status

```bash
curl -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/health/snooze'
```

## 12) Admin: cancel snooze

```bash
curl -X DELETE -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/health/snooze'
```

## 13) Admin: notification config

Get:
```bash
curl -H 'X-Admin-Key: adminkey' \
  'http://localhost:8080/api/v1/admin/notify'
```

Update (enable webhook):
```bash
curl -X PUT -H 'X-Admin-Key: adminkey' -H 'Content-Type: application/json' \
  -d '{"enabled":true,"webhook_urls":["https://example.com/webhook"],"telegram_bot_token":"","telegram_chat_id":"","cooldown_min":180}' \
  'http://localhost:8080/api/v1/admin/notify'
```
