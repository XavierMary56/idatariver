package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TenantRepo struct{ db *pgxpool.Pool }

func NewTenantRepo(db *pgxpool.Pool) *TenantRepo { return &TenantRepo{db: db} }

type Tenant struct {
	ID   int64
	Code string
	Name string
}

func (r *TenantRepo) ListActive(ctx context.Context) ([]Tenant, error) {
	const q = `select id, code, name from tenant where active=true order by id asc;`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tenant{}
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Code, &t.Name); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
