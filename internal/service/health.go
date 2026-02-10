package service

import (
	"context"
	"errors"
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
	if limitWorst <= 0 {
		limitWorst = 10
	}
	if limitWorst > 100 {
		limitWorst = 100
	}
	if s.HealthData == nil {
		return DataHealthReport{}, errors.New("health data repo not configured")
	}

	rows, err := s.HealthData.ListDataHealth(ctx, types, tenantID)
	if err != nil {
		return DataHealthReport{}, err
	}

	all := make([]DataHealthItem, 0, len(rows))
	sum := DataHealthSummary{}
	for _, row := range rows {
		it := DataHealthItem{
			Symbol:         row.Symbol,
			Type:           row.Type,
			Status:         row.Status,
			SLAHours:       row.SLAHours,
			FreshnessHours: row.FreshnessHours,
			LagHours:       row.LagHours,
			LastCloseTS:    row.LastCloseTS,
			Sources:        row.Sources,
		}
		all = append(all, it)
		sum.Total++
		switch it.Status {
		case "fresh":
			sum.Fresh++
		case "stale":
			sum.Stale++
		case "missing":
			sum.Missing++
		}
	}

	worst := []DataHealthItem{}
	for _, it := range all {
		if it.Status == "stale" && it.LagHours != nil {
			worst = append(worst, it)
		}
	}
	sort.SliceStable(worst, func(i, j int) bool { return *worst[i].LagHours > *worst[j].LagHours })
	if len(worst) > limitWorst {
		worst = worst[:limitWorst]
	}

	rep := DataHealthReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Summary: sum, Worst: worst}
	if includeItems {
		rep.Items = all
	}
	return rep, nil
}
