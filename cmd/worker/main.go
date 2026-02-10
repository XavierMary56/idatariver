package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robfig/cron/v3"

	"idatariver-finapi/internal/jobs"
	"idatariver-finapi/internal/repo"
	"idatariver-finapi/internal/service"
	"idatariver-finapi/internal/util"
)

func main() {
	ctx := context.Background()
	dsn := util.Getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable")
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	jobRunRepo := repo.NewJobRunRepo(pool)
	tenantRepo := repo.NewTenantRepo(pool)
	notifyRepo := repo.NewNotifyRepo(pool)
	healthRepo := repo.NewHealthRepo(pool)
	apiSvc := service.NewAPIService(pool)

	healthJob := &jobs.HealthCheckJob{
		JobRunRepo: jobRunRepo,
		TenantRepo: tenantRepo,
		NotifyRepo: notifyRepo,
		HealthRepo: healthRepo,
		API:        apiSvc,
	}

	spec := util.Getenv("HEALTH_CRON", "0 5 * * * *") // every hour at :05
	c := cron.New(cron.WithSeconds())
	_, err = c.AddFunc(spec, func() {
		log.Println("[cron] health_check_all start")
		_ = healthJob.RunAllTenants(context.Background())
		log.Println("[cron] health_check_all done")
	})
	if err != nil {
		panic(err)
	}

	c.Start()
	log.Printf("worker started (cron=%s)\n", spec)
	select {
	case <-time.After(365 * 24 * time.Hour):
	}
}
