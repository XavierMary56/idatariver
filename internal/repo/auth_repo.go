package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AuthRepo struct{ db *pgxpool.Pool }

func NewAuthRepo(db *pgxpool.Pool) *AuthRepo { return &AuthRepo{db: db} }

type KeyRecord struct {
	TenantID int64
	IsAdmin  bool
	Active   bool
	Label    string
}

func (r *AuthRepo) LookupKeyHash(ctx context.Context, keyHash string) (*KeyRecord, error) {
	const q = `
select tenant_id, is_admin, active, label
from api_key
where key_hash=$1
limit 1;`
	var rec KeyRecord
	err := r.db.QueryRow(ctx, q, keyHash).Scan(&rec.TenantID, &rec.IsAdmin, &rec.Active, &rec.Label)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}
