-- 001_init.sql
-- Core multi-tenant schema + market data + health monitoring.

create table if not exists schema_migrations (
  version text primary key,
  applied_at timestamptz not null default now()
);

-- Tenants
create table if not exists tenant (
  id bigserial primary key,
  code text unique not null,
  name text not null,
  active boolean not null default true,
  created_at timestamptz not null default now()
);

insert into tenant(code, name) values ('default','Default') on conflict do nothing;

-- API keys (tenant scoped)
create table if not exists api_key (
  id bigserial primary key,
  key_hash text unique not null,
  tenant_id bigint not null references tenant(id) on delete cascade,
  is_admin boolean not null default false,
  active boolean not null default true,
  label text not null default '',
  created_at timestamptz not null default now()
);

create index if not exists idx_api_key_tenant on api_key(tenant_id);

-- Per-tenant notification config
create table if not exists tenant_notify_config (
  tenant_id bigint primary key references tenant(id) on delete cascade,
  enabled boolean not null default true,
  webhook_urls text[] not null default '{}'::text[],
  telegram_bot_token text not null default '',
  telegram_chat_id text not null default '',
  cooldown_min int not null default 180,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- Instruments / symbols
create table if not exists instrument (
  id bigserial primary key,
  symbol text unique not null,
  type text not null,
  name text not null default '',
  currency text not null default '',
  active boolean not null default true,
  created_at timestamptz not null default now()
);

create index if not exists idx_instrument_type on instrument(type);

-- Data sources and mappings
create table if not exists data_source (
  id bigserial primary key,
  name text unique not null,
  enabled boolean not null default true,
  created_at timestamptz not null default now()
);

create table if not exists instrument_mapping (
  instrument_id bigint not null references instrument(id) on delete cascade,
  source_id bigint not null references data_source(id) on delete cascade,
  source_symbol text not null,
  enabled boolean not null default true,
  primary key (instrument_id, source_id)
);
-- Time series points
create table if not exists ts_point (
  instrument_id bigint not null references instrument(id) on delete cascade,
  ts timestamptz not null,
  close double precision,
  primary key (instrument_id, ts)
);

create index if not exists idx_ts_point_instr_ts on ts_point(instrument_id, ts desc);

-- Daily metrics
create table if not exists daily_metrics (
  instrument_id bigint not null references instrument(id) on delete cascade,
  d date not null,
  ret_1d double precision,
  ret_5d double precision,
  ret_20d double precision,
  ma_20 double precision,
  ma_60 double precision,
  vol_20d double precision,
  max_drawdown_252d double precision,
  primary key (instrument_id, d)
);

create index if not exists idx_daily_metrics_instr_d on daily_metrics(instrument_id, d desc);

-- Per-tenant SLA overrides
create table if not exists instrument_sla (
  tenant_id bigint not null references tenant(id) on delete cascade,
  instrument_id bigint not null references instrument(id) on delete cascade,
  sla_hours numeric not null check (sla_hours > 0),
  updated_at timestamptz not null default now(),
  primary key (tenant_id, instrument_id)
);

-- Data health checks
create table if not exists data_health_check (
  id bigserial primary key,
  tenant_id bigint not null references tenant(id) on delete cascade,
  check_ts timestamptz not null default now(),
  total int not null,
  fresh int not null,
  stale int not null,
  missing int not null,
  status_key text not null default '',
  status_level text not null default 'unknown',
  severity text not null default 'unknown',
  worst jsonb not null default '[]'::jsonb,
  items jsonb not null default '[]'::jsonb,
  notified boolean not null default false,
  notify_targets text[] not null default '{}'::text[],
  notify_error text not null default ''
);

create index if not exists idx_data_health_check_tenant_ts on data_health_check(tenant_id, check_ts desc);
create index if not exists idx_data_health_check_level_ts on data_health_check(tenant_id, status_level, check_ts desc);
create index if not exists idx_data_health_check_sev_ts on data_health_check(tenant_id, severity, check_ts desc);

-- Snooze
create table if not exists data_health_snooze (
  id bigserial primary key,
  tenant_id bigint not null references tenant(id) on delete cascade,
  enabled boolean not null default true,
  reason text not null default '',
  until_ts timestamptz not null,
  created_at timestamptz not null default now()
);

create index if not exists idx_data_health_snooze_tenant_until on data_health_snooze(tenant_id, until_ts desc);
-- Job run logs
create table if not exists job_run (
  id bigserial primary key,
  name text not null,
  started_at timestamptz not null default now(),
  finished_at timestamptz,
  status text not null default 'running',
  message text not null default ''
);

create index if not exists idx_job_run_name_ts on job_run(name, started_at desc);
