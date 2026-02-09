-- 002_seed.sql
-- Demo seed data (safe to re-run).

insert into data_source(name) values ('stooq') on conflict do nothing;
insert into data_source(name) values ('fred') on conflict do nothing;

insert into instrument(symbol, type, name, currency) values
 ('SP500','index','S&P 500 Index','USD'),
 ('NASDAQ','index','NASDAQ 100','USD'),
 ('US10Y','rate','US 10Y Treasury Yield','USD'),
 ('US2Y','rate','US 2Y Treasury Yield','USD'),
 ('USD_CNY','fx','USD/CNY','CNY')
on conflict (symbol) do nothing;

-- Example mappings
insert into instrument_mapping(instrument_id, source_id, source_symbol, enabled)
select i.id, ds.id, i.symbol, true
from instrument i
join data_source ds on ds.name='stooq'
where i.symbol in ('SP500','NASDAQ')
on conflict do nothing;

insert into instrument_mapping(instrument_id, source_id, source_symbol, enabled)
select i.id, ds.id, i.symbol, true
from instrument i
join data_source ds on ds.name='fred'
where i.symbol in ('US10Y','US2Y','USD_CNY')
on conflict do nothing;

-- Demo tenant notify config row (disabled by default)
insert into tenant_notify_config(tenant_id, enabled, cooldown_min)
select t.id, false, 180 from tenant t where t.code='default'
on conflict (tenant_id) do nothing;

-- Demo API keys
-- Store SHA256(key) hex in key_hash. Example keys below:
--   devkey
--   adminkey
create extension if not exists pgcrypto;

insert into api_key(key_hash, tenant_id, is_admin, active, label)
values
(encode(digest('devkey','sha256'),'hex'),  (select id from tenant where code='default'), false, true, 'demo user key (devkey)'),
(encode(digest('adminkey','sha256'),'hex'),(select id from tenant where code='default'), true,  true, 'demo admin key (adminkey)')
on conflict do nothing;
