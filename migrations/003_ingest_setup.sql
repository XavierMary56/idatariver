-- 003_ingest_setup.sql
-- Idempotent setup for ingest mappings + ts_point upsert support.

-- 1) Ensure pgcrypto exists (optional but useful elsewhere)
create extension if not exists pgcrypto;

-- 2) Ensure data_source table has required sources
insert into data_source(name, enabled)
values
  ('stooq', true),
  ('fred',  true)
on conflict (name) do update
set enabled = excluded.enabled;

-- 3) Ensure instrument_mapping has source symbols for sources
-- Stooq example codes:
--   SP500  -> spx
--   NASDAQ -> ndx
with s as (select id from data_source where name='stooq')
insert into instrument_mapping(instrument_id, source_id, source_symbol, enabled)
select i.id, s.id,
  case
    when i.symbol='SP500'  then 'spx'
    when i.symbol='NASDAQ' then 'ndx'
    else null
  end as source_symbol,
  true as enabled
from instrument i
cross join s
where i.symbol in ('SP500','NASDAQ')
on conflict (instrument_id, source_id) do update
set source_symbol = excluded.source_symbol,
    enabled       = true;

-- FRED series_id examples:
--   US10Y -> DGS10
--   US2Y  -> DGS2
-- USD_CNY series_id varies; left NULL by default.
with s as (select id from data_source where name='fred')
insert into instrument_mapping(instrument_id, source_id, source_symbol, enabled)
select i.id, s.id,
  case
    when i.symbol='US10Y'   then 'DGS10'
    when i.symbol='US2Y'    then 'DGS2'
    when i.symbol='USD_CNY' then null
    else null
  end as source_symbol,
  true as enabled
from instrument i
cross join s
where i.symbol in ('US10Y','US2Y','USD_CNY')
on conflict (instrument_id, source_id) do update
set source_symbol = excluded.source_symbol,
    enabled       = true;

-- 4) Ensure ts_point has a unique constraint on (instrument_id, ts)
do $$
begin
  if not exists (
    select 1
    from pg_constraint c
    join pg_class t on t.oid = c.conrelid
    where t.relname = 'ts_point'
      and c.contype = 'u'
      and c.conname = 'ts_point_unique'
  ) then
    alter table ts_point add constraint ts_point_unique unique(instrument_id, ts);
  end if;
end$$;
