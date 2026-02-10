package service

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"idatariver-finapi/internal/repo"
)

func NewAPIService(db *pgxpool.Pool) *APIService {
	return &APIService{
		DB:          db,
		Instruments: repo.NewInstrumentRepo(db),
		Market:      repo.NewMarketRepo(db),
		HealthData:  repo.NewHealthDataRepo(db),
	}
}
