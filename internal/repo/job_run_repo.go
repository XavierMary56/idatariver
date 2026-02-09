package repo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRunRepo struct{ db *pgxpool.Pool }

func NewJobRunRepo(db *pgxpool.Pool) *JobRunRepo { return &JobRunRepo{db: db} }

func (r *JobRunRepo) Start(ctx context.Context, name string) (int64, error) {
	const q = `insert into job_run(name, started_at, status) values ($1,$2,'running') returning id;`
	var id int64
	err := r.db.QueryRow(ctx, q, name, time.Now().UTC()).Scan(&id)
	return id, err
}

func (r *JobRunRepo) Finish(ctx context.Context, id int64, status string, message string) error {
	const q = `update job_run set finished_at=$2, status=$3, message=$4 where id=$1;`
	_, err := r.db.Exec(ctx, q, id, time.Now().UTC(), status, message)
	return err
}
