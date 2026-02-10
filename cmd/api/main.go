package main

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"idatariver-finapi/internal/httpapi"
	"idatariver-finapi/internal/repo"
	"idatariver-finapi/internal/service"
	"idatariver-finapi/internal/util"
)

func main() {
	dsn := util.Getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/idatariver?sslmode=disable")
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	authRepo := repo.NewAuthRepo(pool)
	healthRepo := repo.NewHealthRepo(pool)
	notifyRepo := repo.NewNotifyRepo(pool)

	apiSvc := service.NewAPIService(pool)
	adminSvc := &service.AdminService{API: apiSvc, HealthRepo: healthRepo, NotifyRepo: notifyRepo}

	defaultWorst := util.GetenvInt("HEALTH_WORST_DEFAULT", 10)
	r := httpapi.NewRouter(apiSvc, adminSvc, authRepo, defaultWorst)

	addr := util.Getenv("API_ADDR", ":8080")
	_ = os.Setenv("GIN_MODE", util.Getenv("GIN_MODE", "release"))
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
