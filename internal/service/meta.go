package service

import (
	"context"
	"errors"
)

type SymbolMeta struct {
	Symbol         string   `json:"symbol"`
	Type           string   `json:"type"`
	Name           string   `json:"name"`
	Currency       string   `json:"currency"`
	Active         bool     `json:"active"`
	Sources        []string `json:"sources"`
	LastCloseTS    string   `json:"last_close_ts"`
	FreshnessHours *float64 `json:"freshness_hours"`
	SLAHours       float64  `json:"sla_hours"`
	Status         string   `json:"status"`
}

func (s *APIService) ListSymbols(ctx context.Context, tenantID int64, onlyActive bool, types []string) ([]SymbolMeta, error) {
	if s.Instruments == nil {
		return nil, errors.New("instrument repo not configured")
	}
	rows, err := s.Instruments.ListSymbols(ctx, onlyActive, types, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]SymbolMeta, 0, len(rows))
	for _, row := range rows {
		out = append(out, SymbolMeta{
			Symbol:         row.Symbol,
			Type:           row.Type,
			Name:           row.Name,
			Currency:       row.Currency,
			Active:         row.Active,
			Sources:        row.Sources,
			LastCloseTS:    row.LastCloseTS,
			FreshnessHours: row.FreshnessHours,
			SLAHours:       row.SLAHours,
			Status:         row.Status,
		})
	}
	return out, nil
}
