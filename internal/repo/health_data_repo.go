package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthDataRepo struct{ db *pgxpool.Pool }

func NewHealthDataRepo(db *pgxpool.Pool) *HealthDataRepo { return &HealthDataRepo{db: db} }

type DataHealthRow struct {
	Symbol         string
	Type           string
	Status         string
	SLAHours       float64
	FreshnessHours *float64
	LagHours       *float64
	LastCloseTS    string
	Sources        []string
}

func (r *HealthDataRepo) ListDataHealth(ctx context.Context, types []string, tenantID int64) ([]DataHealthRow, error) {
	const q = `
with base as (
  select i.id, i.symbol, i.type
  from instrument i
  where i.active=true
    and (cardinality($1::text[]) = 0 or i.type = any($1::text[]))
),
src as (
  select im.instrument_id, array_agg(distinct ds.name order by ds.name) as sources
  from instrument_mapping im
  join data_source ds on ds.id = im.source_id
  where im.enabled=true and ds.enabled=true
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
  where tenant_id = $2
),
enriched as (
  select
    b.symbol,
    b.type,
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
),
final as (
  select
    e.symbol,
    e.type,
    e.sources,
    case when e.last_ts is null then '' else to_char(e.last_ts at time zone 'UTC','YYYY-MM-DD"T"HH24:MI:SS"Z"') end as last_close_ts,
    e.freshness_hours,
    e.sla_hours,
    case when e.last_ts is null then 'missing'
         when e.freshness_hours <= e.sla_hours then 'fresh'
         else 'stale' end as status,
    case when e.last_ts is null then null else (e.freshness_hours - e.sla_hours) end as lag_hours
  from enriched e
)
select symbol, type, status, sla_hours, freshness_hours, lag_hours, last_close_ts, sources
from final
order by type asc, symbol asc;`

	rows, err := r.db.Query(ctx, q, types, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DataHealthRow{}
	for rows.Next() {
		var r DataHealthRow
		if err := rows.Scan(&r.Symbol, &r.Type, &r.Status, &r.SLAHours, &r.FreshnessHours, &r.LagHours, &r.LastCloseTS, &r.Sources); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
