package service

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SymbolMeta struct {
	Symbol         string    `json:"symbol"`
	Type           string    `json:"type"`
	Name           string    `json:"name"`
	Currency       string    `json:"currency"`
	Active         bool      `json:"active"`
	Sources        []string  `json:"sources"`
	LastCloseTS    string    `json:"last_close_ts"`
	FreshnessHours *float64  `json:"freshness_hours"`
	SLAHours       float64   `json:"sla_hours"`
	Status         string    `json:"status"`
}

func (s *APIService) ListSymbols(ctx context.Context, tenantID int64, onlyActive bool, types []string) ([]SymbolMeta, error) {
	const q = `
with base as (
  select i.id, i.symbol, i.type, coalesce(i.name,'') as name, coalesce(i.currency,'') as currency, i.active
  from instrument i
  where ($1::bool = false or i.active = true)
    and (cardinality($2::text[]) = 0 or i.type = any($2::text[]))
),
src as (
  select im.instrument_id, array_agg(distinct ds.name order by ds.name) as sources
  from instrument_mapping im
  join data_source ds on ds.id = im.source_id
  where im.enabled = true and ds.enabled = true
  group by im.instrument_id
),
lastp as (
  select instrument_id, max(ts) as last_ts
  from ts_point
  group by instrument_id
),
custom as (
  select instrument_id, sla_hours
  from instrument_sla
  where tenant_id = $3
),
enriched as (
  select
    b.*,
    coalesce(src.sources, '{}'::text[]) as sources,
    lastp.last_ts,
    case when lastp.last_ts is null then null
         else extract(epoch from (now() at time zone 'UTC' - lastp.last_ts)) / 3600.0 end as freshness_hours,
    coalesce(custom.sla_hours::float8,
      case when b.type='index' then 36.0
           when b.type='rate' then 72.0
           when b.type='fx' then 72.0
           else 72.0 end
    ) as sla_hours
  from base b
  left join src on src.instrument_id=b.id
  left join lastp on lastp.instrument_id=b.id
  left join custom on custom.instrument_id=b.id
)
select
  e.symbol, e.type, e.name, e.currency, e.active,
  e.sources,
  case when e.last_ts is null then '' else to_char(e.last_ts at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end as last_close_ts,
  e.freshness_hours,
  e.sla_hours,
  case when e.last_ts is null then 'missing'
       when e.freshness_hours <= e.sla_hours then 'fresh'
       else 'stale' end as status
from enriched e
order by e.type asc, e.symbol asc;`

	rows, err := s.DB.Query(ctx, q, onlyActive, types, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SymbolMeta{}
	for rows.Next() {
		var r SymbolMeta
		if err := rows.Scan(&r.Symbol, &r.Type, &r.Name, &r.Currency, &r.Active, &r.Sources, &r.LastCloseTS, &r.FreshnessHours, &r.SLAHours, &r.Status); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// compile-time check to ensure pgxpool is used
var _ = pgxpool.Config{}
var _ = time.RFC3339
