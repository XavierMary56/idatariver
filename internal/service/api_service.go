package service

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIService struct {
	DB *pgxpool.Pool
}

func (s *APIService) InstrumentID(ctx context.Context, symbol string) (int64, error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return 0, errors.New("missing symbol")
	}
	const q = `select id from instrument where symbol=$1 and active=true limit 1;`
	var id int64
	if err := s.DB.QueryRow(ctx, q, symbol).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

type LatestCompareItem struct {
	Symbol      string  `json:"symbol"`
	MetricsDate string  `json:"metrics_date,omitempty"`
	Close       *float64 `json:"close,omitempty"`
	CloseTS     string  `json:"close_ts,omitempty"`

	Ret1D   *float64 `json:"ret_1d_pct,omitempty"`
	Ret5D   *float64 `json:"ret_5d_pct,omitempty"`
	Ret20D  *float64 `json:"ret_20d_pct,omitempty"`
	MA20    *float64 `json:"ma_20,omitempty"`
	MA60    *float64 `json:"ma_60,omitempty"`
	Vol20D  *float64 `json:"vol_20d_pct,omitempty"`
	MDD252D *float64 `json:"mdd_252d_pct,omitempty"`
}

func isAllowedSortKey(k string) bool {
	switch k {
	case "symbol", "close", "ret_1d", "ret_5d", "ret_20d", "ma_20", "ma_60", "vol_20d", "mdd_252d":
		return true
	default:
		return false
	}
}

func sortValue(it LatestCompareItem, sortBy string) (float64, bool) {
	switch sortBy {
	case "close":
		if it.Close == nil { return 0, false }
		return *it.Close, true
	case "ret_1d":
		if it.Ret1D == nil { return 0, false }
		return *it.Ret1D, true
	case "ret_5d":
		if it.Ret5D == nil { return 0, false }
		return *it.Ret5D, true
	case "ret_20d":
		if it.Ret20D == nil { return 0, false }
		return *it.Ret20D, true
	case "ma_20":
		if it.MA20 == nil { return 0, false }
		return *it.MA20, true
	case "ma_60":
		if it.MA60 == nil { return 0, false }
		return *it.MA60, true
	case "vol_20d":
		if it.Vol20D == nil { return 0, false }
		return *it.Vol20D, true
	case "mdd_252d":
		if it.MDD252D == nil { return 0, false }
		return *it.MDD252D, true
	default:
		return 0, false
	}
}

// Fields metadata + normalization

type FieldMeta struct {
	Key         string   `json:"key"`
	Aliases     []string `json:"aliases"`
	Type        string   `json:"type"`
	Unit        string   `json:"unit"`
	Description string   `json:"description"`
}

func AvailableFieldMeta() []FieldMeta {
	return []FieldMeta{
		{Key: "symbol", Aliases: []string{"sym"}, Type: "string", Unit: "", Description: "Instrument symbol"},
		{Key: "metrics_date", Aliases: []string{"date", "metric_date"}, Type: "string", Unit: "date", Description: "Date of latest computed metrics (YYYY-MM-DD)"},
		{Key: "close", Aliases: []string{"price", "last"}, Type: "number", Unit: "price", Description: "Latest close/value"},
		{Key: "close_ts", Aliases: []string{"closets", "ts", "timestamp"}, Type: "string", Unit: "rfc3339", Description: "Timestamp of latest close (RFC3339)"},
		{Key: "ret_1d", Aliases: []string{"ret1d", "r1d", "chg1d"}, Type: "number", Unit: "%", Description: "1-day return (%)"},
		{Key: "ret_5d", Aliases: []string{"ret5d", "r5d"}, Type: "number", Unit: "%", Description: "5-day return (%)"},
		{Key: "ret_20d", Aliases: []string{"ret20d", "r20d"}, Type: "number", Unit: "%", Description: "20-day return (%)"},
		{Key: "ma_20", Aliases: []string{"ma20", "sma20"}, Type: "number", Unit: "price", Description: "20-day simple moving average"},
		{Key: "ma_60", Aliases: []string{"ma60", "sma60"}, Type: "number", Unit: "price", Description: "60-day simple moving average"},
		{Key: "vol_20d", Aliases: []string{"vol20d", "vol", "volatility", "sigma"}, Type: "number", Unit: "%", Description: "20-day annualized volatility (%)"},
		{Key: "mdd_252d", Aliases: []string{"mdd252d", "mdd", "max_drawdown", "drawdown"}, Type: "number", Unit: "%", Description: "252-day max drawdown (%) (negative)"},
	}
}

func NormalizeFieldName(s string) string {
	x := strings.TrimSpace(strings.ToLower(s))
	if x == "" { return "" }
	switch x {
	case "symbol", "sym":
		return "symbol"
	case "metrics_date", "metric_date", "date":
		return "metrics_date"
	case "close", "price", "last":
		return "close"
	case "close_ts", "closets", "close_ts_utc", "ts", "timestamp":
		return "close_ts"
	case "ret_1d", "ret1d", "r1d", "chg1d", "change1d":
		return "ret_1d"
	case "ret_5d", "ret5d", "r5d":
		return "ret_5d"
	case "ret_20d", "ret20d", "r20d":
		return "ret_20d"
	case "ma_20", "ma20", "sma20":
		return "ma_20"
	case "ma_60", "ma60", "sma60":
		return "ma_60"
	case "vol_20d", "vol20d", "vol", "sigma", "volatility":
		return "vol_20d"
	case "mdd_252d", "mdd252d", "mdd", "max_drawdown", "drawdown":
		return "mdd_252d"
	default:
		return x
	}
}

// Compare latest with strict fields.
func (s *APIService) CompareLatest(ctx context.Context, symbols []string, fields []string, includeClose bool, sortBy string, order string) ([]LatestCompareItem, error) {
	if len(symbols) == 0 { return nil, errors.New("missing symbols") }
	if len(symbols) > 20 { return nil, errors.New("too many symbols (max 20)") }

	needAll := len(fields) == 0
	want := map[string]struct{}{}
	for _, f := range fields {
		nf := NormalizeFieldName(f)
		if nf != "" { want[nf] = struct{}{} }
	}

	sortBy = NormalizeFieldName(sortBy)
	if sortBy == "" { sortBy = "symbol" }
	if sortBy == "metrics_date" { sortBy = "symbol" }
	order = strings.TrimSpace(strings.ToLower(order))
	if order == "" { order = "desc" }
	if order != "asc" && order != "desc" { return nil, errors.New("invalid order (asc|desc)") }
	if !isAllowedSortKey(sortBy) { return nil, errors.New("invalid sort_by") }

	out := make([]LatestCompareItem, 0, len(symbols))
	for _, sym := range symbols {
		id, err := s.InstrumentID(ctx, sym)
		if err != nil { return nil, errors.New("unknown symbol: "+sym) }
		item := LatestCompareItem{Symbol: strings.ToUpper(strings.TrimSpace(sym))}

		needCloseForSort := (sortBy == "close")
		if includeClose || needCloseForSort {
			const q = `select ts, close from ts_point where instrument_id=$1 and close is not null order by ts desc limit 1;`
			var ts time.Time
			var close float64
			if err := s.DB.QueryRow(ctx, q, id).Scan(&ts, &close); err == nil {
				v := close
				item.Close = &v
				if includeClose {
					item.CloseTS = ts.UTC().Format(time.RFC3339)
				}
			}
		}

		{
			const q = `
select d, ret_1d, ret_5d, ret_20d, ma_20, ma_60, vol_20d, max_drawdown_252d
from daily_metrics
where instrument_id=$1
order by d desc
limit 1;`
			var d time.Time
			var r1, r5, r20, ma20, ma60, vol, mdd *float64
			if err := s.DB.QueryRow(ctx, q, id).Scan(&d, &r1, &r5, &r20, &ma20, &ma60, &vol, &mdd); err == nil {
				item.MetricsDate = d.UTC().Format("2006-01-02")
				item.Ret1D, item.Ret5D, item.Ret20D = r1, r5, r20
				item.MA20, item.MA60 = ma20, ma60
				item.Vol20D, item.MDD252D = vol, mdd
			}
		}

		if !includeClose { item.CloseTS = "" }
		out = append(out, item)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if sortBy == "symbol" {
			if order == "asc" { return out[i].Symbol < out[j].Symbol }
			return out[i].Symbol > out[j].Symbol
		}
		ai, aok := sortValue(out[i], sortBy)
		bj, bok := sortValue(out[j], sortBy)
		if !aok && !bok { return out[i].Symbol < out[j].Symbol }
		if !aok { return false }
		if !bok { return true }
		if order == "asc" {
			if ai == bj { return out[i].Symbol < out[j].Symbol }
			return ai < bj
		}
		if ai == bj { return out[i].Symbol < out[j].Symbol }
		return ai > bj
	})

	// Strict fields output
	for i := range out {
		if !includeClose {
			out[i].Close = nil
			out[i].CloseTS = ""
		} else if !needAll {
			if _, ok := want["close"]; !ok { out[i].Close = nil }
			if _, ok := want["close_ts"]; !ok { out[i].CloseTS = "" }
		}

		if !needAll {
			if _, ok := want["metrics_date"]; !ok { out[i].MetricsDate = "" }
			if _, ok := want["ret_1d"]; !ok { out[i].Ret1D = nil }
			if _, ok := want["ret_5d"]; !ok { out[i].Ret5D = nil }
			if _, ok := want["ret_20d"]; !ok { out[i].Ret20D = nil }
			if _, ok := want["ma_20"]; !ok { out[i].MA20 = nil }
			if _, ok := want["ma_60"]; !ok { out[i].MA60 = nil }
			if _, ok := want["vol_20d"]; !ok { out[i].Vol20D = nil }
			if _, ok := want["mdd_252d"]; !ok { out[i].MDD252D = nil }
		}
	}

	return out, nil
}
