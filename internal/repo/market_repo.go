package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MarketRepo struct{ db *pgxpool.Pool }

func NewMarketRepo(db *pgxpool.Pool) *MarketRepo { return &MarketRepo{db: db} }

type LatestMetricsRow struct {
	Date    time.Time
	Ret1D   *float64
	Ret5D   *float64
	Ret20D  *float64
	MA20    *float64
	MA60    *float64
	Vol20D  *float64
	MDD252D *float64
}

func (r *MarketRepo) LatestClose(ctx context.Context, instrumentID int64) (time.Time, float64, bool, error) {
	const q = `select ts, close from ts_point where instrument_id=$1 and close is not null order by ts desc limit 1;`
	var ts time.Time
	var close float64
	if err := r.db.QueryRow(ctx, q, instrumentID).Scan(&ts, &close); err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, 0, false, nil
		}
		return time.Time{}, 0, false, err
	}
	return ts, close, true, nil
}

func (r *MarketRepo) LatestMetrics(ctx context.Context, instrumentID int64) (*LatestMetricsRow, error) {
	const q = `
select d, ret_1d, ret_5d, ret_20d, ma_20, ma_60, vol_20d, max_drawdown_252d
from daily_metrics
where instrument_id=$1
order by d desc
limit 1;`
	var row LatestMetricsRow
	if err := r.db.QueryRow(ctx, q, instrumentID).Scan(&row.Date, &row.Ret1D, &row.Ret5D, &row.Ret20D, &row.MA20, &row.MA60, &row.Vol20D, &row.MDD252D); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}
