# idatariver-finapi

Multi-tenant market-data API skeleton (Go + Postgres) with:
- Per-tenant API keys
- Per-tenant SLA overrides
- Data health checks (fresh/stale/missing)
- Worker cron that writes health checks + notifies via webhook/Telegram
- Admin APIs: SLA (JSON + CSV), snooze, notify config

See:
- docs/DEPLOY.md
- docs/API.md
- docs/TESTS.md
