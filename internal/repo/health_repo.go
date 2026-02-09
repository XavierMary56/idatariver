package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepo struct{ db *pgxpool.Pool }

func NewHealthRepo(db *pgxpool.Pool) *HealthRepo { return &HealthRepo{db: db} }

type HealthCheckInsert struct {
	CheckTS time.Time
	Total   int
	Fresh   int
	Stale   int
	Missing int
	Worst any
	Items any
}

type LastHealthCheck struct {
	ID          int64
	CheckTS     time.Time
	StatusLevel string
	StatusKey   string
	Severity    string
}

func (r *HealthRepo) GetLast(ctx context.Context, tenantID int64) (*LastHealthCheck, error) {
	const q = `
select id, check_ts, status_level, status_key, severity
from data_health_check
where tenant_id=$1
order by check_ts desc
limit 1;`
	var o LastHealthCheck
	err := r.db.QueryRow(ctx, q, tenantID).Scan(&o.ID, &o.CheckTS, &o.StatusLevel, &o.StatusKey, &o.Severity)
	if err != nil {
		return nil, nil
	}
	return &o, nil
}

func (r *HealthRepo) InsertWithStatus(ctx context.Context, tenantID int64, in HealthCheckInsert, statusLevel, statusKey, severity string) (int64, error) {
	worstB, err := json.Marshal(in.Worst)
	if err != nil {
		return 0, err
	}
	itemsB, err := json.Marshal(in.Items)
	if err != nil {
		return 0, err
	}

	const q = `
insert into data_health_check(
  tenant_id, check_ts,total,fresh,stale,missing,worst,items,status_level,status_key,severity
) values ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9,$10,$11)
returning id;`
	var id int64
	err = r.db.QueryRow(ctx, q,
		tenantID, in.CheckTS.UTC(), in.Total, in.Fresh, in.Stale, in.Missing, worstB, itemsB,
		statusLevel, statusKey, severity,
	).Scan(&id)
	return id, err
}

func (r *HealthRepo) MarkNotified(ctx context.Context, tenantID, id int64, targets []string, notifyErr string) error {
	const q = `
update data_health_check
set notified = true,
    notify_targets = $3::text[],
    notify_error = $4
where tenant_id=$1 and id = $2;`
	_, err := r.db.Exec(ctx, q, tenantID, id, targets, notifyErr)
	return err
}

func (r *HealthRepo) IsSnoozed(ctx context.Context, tenantID int64) (bool, string, time.Time, error) {
	const q = `
select reason, until_ts
from data_health_snooze
where tenant_id=$1 and enabled=true and until_ts > now()
order by until_ts desc
limit 1;`
	var reason string
	var until time.Time
	err := r.db.QueryRow(ctx, q, tenantID).Scan(&reason, &until)
	if err != nil {
		return false, "", time.Time{}, nil
	}
	return true, reason, until.UTC(), nil
}

func (r *HealthRepo) Snooze(ctx context.Context, tenantID int64, minutes int, reason string) (time.Time, error) {
	if minutes <= 0 {
		return time.Time{}, errors.New("minutes must be > 0")
	}
	if minutes > 7*24*60 {
		return time.Time{}, errors.New("minutes too large (max 7 days)")
	}
	until := time.Now().UTC().Add(time.Duration(minutes) * time.Minute)
	const q = `
insert into data_health_snooze(tenant_id, enabled, reason, until_ts)
values ($1, true, $2, $3)
returning until_ts;`
	var out time.Time
	if err := r.db.QueryRow(ctx, q, tenantID, reason, until).Scan(&out); err != nil {
		return time.Time{}, err
	}
	return out.UTC(), nil
}

func (r *HealthRepo) CancelSnooze(ctx context.Context, tenantID int64) (int64, error) {
	const q = `
update data_health_snooze
set enabled=false
where tenant_id=$1 and enabled=true and until_ts > now();`
	tag, err := r.db.Exec(ctx, q, tenantID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
