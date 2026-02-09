package service

import (
	"context"
	"sort"
	"time"
)

type DataHealthItem struct {
	Symbol         string   `json:"symbol"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	SLAHours       float64  `json:"sla_hours"`
	FreshnessHours *float64 `json:"freshness_hours"`
	LagHours       *float64 `json:"lag_hours"`
	LastCloseTS    string   `json:"last_close_ts"`
	Sources        []string `json:"sources"`
}

type DataHealthSummary struct {
	Total   int `json:"total"`
	Fresh   int `json:"fresh"`
	Stale   int `json:"stale"`
	Missing int `json:"missing"`
}

type DataHealthReport struct {
	GeneratedAt string            `json:"generated_at"`
	Summary     DataHealthSummary `json:"summary"`
	Worst       []DataHealthItem  `json:"worst"`
	Items       []DataHealthItem  `json:"items,omitempty"`
}

func (s *APIService) DataHealth(ctx context.Context, tenantID int64, types []string, limitWorst int, includeItems bool) (DataHealthReport, error) {
	if limitWorst <= 0 { limitWorst = 10 }
	if limitWorst > 100 { limitWorst = 100 }

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

	rows, err := s.DB.Query(ctx, q, types, tenantID)
	if err != nil { return DataHealthReport{}, err }
	defer rows.Close()

	all := []DataHealthItem{}
	sum := DataHealthSummary{}
	for rows.Next() {
		var it DataHealthItem
		if err := rows.Scan(&it.Symbol, &it.Type, &it.Status, &it.SLAHours, &it.FreshnessHours, &it.LagHours, &it.LastCloseTS, &it.Sources); err != nil {
			return DataHealthReport{}, err
		}
		all = append(all, it)
		sum.Total++
		switch it.Status {
		case "fresh": sum.Fresh++
		case "stale": sum.Stale++
		case "missing": sum.Missing++
		}
	}
	if err := rows.Err(); err != nil { return DataHealthReport{}, err }

	worst := []DataHealthItem{}
	for _, it := range all {
		if it.Status == "stale" && it.LagHours != nil {
			worst = append(worst, it)
		}
	}
	sort.SliceStable(worst, func(i,j int) bool { return *worst[i].LagHours > *worst[j].LagHours })
	if len(worst) > limitWorst { worst = worst[:limitWorst] }

	rep := DataHealthReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Summary: sum, Worst: worst}
	if includeItems { rep.Items = all }
	return rep, nil
}
