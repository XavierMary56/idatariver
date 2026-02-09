package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"idatariver-finapi/internal/repo"
)

type AdminService struct {
	API       *APIService
	HealthRepo *repo.HealthRepo
	NotifyRepo *repo.NotifyRepo
}

// ---------- SLA ----------

type SetSLARequest struct {
	Symbol   string  `json:"symbol"`
	SLAHours float64 `json:"sla_hours"`
}

type SetSLABatchItem struct {
	Symbol   string  `json:"symbol"`
	SLAHours float64 `json:"sla_hours"`
}

type SetSLABatchRequest struct {
	Items []SetSLABatchItem `json:"items"`
}

type SetSLABatchResult struct {
	Updated []string          `json:"updated"`
	Failed  map[string]string `json:"failed"`
}

type SLARow struct {
	Symbol    string  `json:"symbol"`
	SLAHours  float64 `json:"sla_hours"`
	UpdatedAt string  `json:"updated_at"`
}

func (s *AdminService) SetInstrumentSLA(ctx context.Context, tenantID int64, symbol string, slaHours float64) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" { return errors.New("missing symbol") }
	if slaHours <= 0 { return errors.New("sla_hours must be > 0") }

	id, err := s.API.InstrumentID(ctx, symbol)
	if err != nil { return errors.New("unknown symbol") }

	const q = `
insert into instrument_sla(tenant_id, instrument_id, sla_hours, updated_at)
values ($1,$2,$3, now())
on conflict (tenant_id, instrument_id) do update
set sla_hours=excluded.sla_hours, updated_at=now();`
	_, err = s.API.DB.Exec(ctx, q, tenantID, id, slaHours)
	return err
}

func (s *AdminService) DeleteInstrumentSLA(ctx context.Context, tenantID int64, symbol string) error {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" { return errors.New("missing symbol") }
	id, err := s.API.InstrumentID(ctx, symbol)
	if err != nil { return errors.New("unknown symbol") }
	_, err = s.API.DB.Exec(ctx, `delete from instrument_sla where tenant_id=$1 and instrument_id=$2`, tenantID, id)
	return err
}

func (s *AdminService) ListInstrumentSLA(ctx context.Context, tenantID int64) ([]SLARow, error) {
	const q = `
select i.symbol, isl.sla_hours, isl.updated_at
from instrument_sla isl
join instrument i on i.id=isl.instrument_id
where isl.tenant_id=$1
order by i.symbol asc;`
	rows, err := s.API.DB.Query(ctx, q, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := []SLARow{}
	for rows.Next() {
		var sym string
		var sla float64
		var upd time.Time
		if err := rows.Scan(&sym, &sla, &upd); err != nil { return nil, err }
		out = append(out, SLARow{Symbol: sym, SLAHours: sla, UpdatedAt: upd.UTC().Format(time.RFC3339)})
	}
	return out, rows.Err()
}

func (s *AdminService) SetInstrumentSLABatch(ctx context.Context, tenantID int64, items []SetSLABatchItem) (SetSLABatchResult, error) {
	res := SetSLABatchResult{Updated: []string{}, Failed: map[string]string{}}
	if len(items) == 0 { return res, errors.New("missing items") }
	if len(items) > 200 { return res, errors.New("too many items (max 200)") }

	tx, err := s.API.DB.Begin(ctx)
	if err != nil { return res, err }
	defer tx.Rollback(ctx)

	const upsert = `
insert into instrument_sla(tenant_id, instrument_id, sla_hours, updated_at)
values ($1,$2,$3, now())
on conflict (tenant_id, instrument_id) do update
set sla_hours=excluded.sla_hours, updated_at=now();`

	for _, it := range items {
		sym := strings.ToUpper(strings.TrimSpace(it.Symbol))
		if sym == "" { res.Failed[it.Symbol] = "missing symbol"; continue }
		if it.SLAHours <= 0 { res.Failed[sym] = "sla_hours must be > 0"; continue }

		var id int64
		if err := tx.QueryRow(ctx, `select id from instrument where symbol=$1 and active=true limit 1`, sym).Scan(&id); err != nil {
			res.Failed[sym] = "unknown symbol"; continue
		}
		if _, err := tx.Exec(ctx, upsert, tenantID, id, it.SLAHours); err != nil {
			res.Failed[sym] = err.Error(); continue
		}
		res.Updated = append(res.Updated, sym)
	}

	if err := tx.Commit(ctx); err != nil { return res, err }
	return res, nil
}

func (s *AdminService) ExportSLAAsCSV(ctx context.Context, tenantID int64) (string, error) {
	rows, err := s.ListInstrumentSLA(ctx, tenantID)
	if err != nil { return "", err }
	var b strings.Builder
	b.WriteString("symbol,sla_hours,updated_at\n")
	for _, r := range rows {
		b.WriteString(r.Symbol)
		b.WriteString(",")
		b.WriteString(strconv.FormatFloat(r.SLAHours, 'f', -1, 64))
		b.WriteString(",")
		b.WriteString(r.UpdatedAt)
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (s *AdminService) ImportSLAFromCSV(ctx context.Context, tenantID int64, csvText string) (SetSLABatchResult, error) {
	lines := strings.Split(csvText, "\n")
	items := []SetSLABatchItem{}
	errorsByLine := map[string]string{}

	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		if lineNo == 1 && strings.HasPrefix(strings.ToLower(line), "symbol") { continue }

		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			errorsByLine[fmt.Sprintf("line %d", lineNo)] = "expected format: symbol,sla_hours"; continue
		}
		sym := strings.ToUpper(strings.TrimSpace(parts[0]))
		slaStr := strings.TrimSpace(parts[1])
		if sym == "" { errorsByLine[fmt.Sprintf("line %d", lineNo)] = "missing symbol"; continue }
		if slaStr == "" { errorsByLine[fmt.Sprintf("line %d (%s)", lineNo, sym)] = "missing sla_hours"; continue }
		sla, err := strconv.ParseFloat(slaStr, 64)
		if err != nil { errorsByLine[fmt.Sprintf("line %d (%s)", lineNo, sym)] = "invalid sla_hours"; continue }
		if sla <= 0 { errorsByLine[fmt.Sprintf("line %d (%s)", lineNo, sym)] = "sla_hours must be > 0"; continue }

		items = append(items, SetSLABatchItem{Symbol: sym, SLAHours: sla})
	}

	if len(items) == 0 {
		return SetSLABatchResult{Updated: []string{}, Failed: errorsByLine}, errors.New("no valid rows in csv")
	}

	res, err := s.SetInstrumentSLABatch(ctx, tenantID, items)
	if err != nil { return res, err }
	for k, v := range errorsByLine { res.Failed[k] = v }
	return res, nil
}

// ---------- Snooze ----------

type SnoozeRequest struct {
	Minutes int    `json:"minutes"`
	Reason  string `json:"reason"`
}

type SnoozeResponse struct {
	Snoozed   bool   `json:"snoozed"`
	Reason    string `json:"reason,omitempty"`
	UntilTS   string `json:"until_ts,omitempty"`
	Cancelled int64  `json:"cancelled,omitempty"`
}

func (s *AdminService) Snooze(ctx context.Context, tenantID int64, minutes int, reason string) (SnoozeResponse, error) {
	until, err := s.HealthRepo.Snooze(ctx, tenantID, minutes, strings.TrimSpace(reason))
	if err != nil { return SnoozeResponse{}, err }
	return SnoozeResponse{Snoozed: true, Reason: strings.TrimSpace(reason), UntilTS: until.UTC().Format(time.RFC3339)}, nil
}

func (s *AdminService) CancelSnooze(ctx context.Context, tenantID int64) (SnoozeResponse, error) {
	n, err := s.HealthRepo.CancelSnooze(ctx, tenantID)
	if err != nil { return SnoozeResponse{}, err }
	return SnoozeResponse{Snoozed: false, Cancelled: n}, nil
}

func (s *AdminService) GetSnoozeStatus(ctx context.Context, tenantID int64) (SnoozeResponse, error) {
	ok, reason, until, err := s.HealthRepo.IsSnoozed(ctx, tenantID)
	if err != nil { return SnoozeResponse{}, err }
	if !ok { return SnoozeResponse{Snoozed: false}, nil }
	return SnoozeResponse{Snoozed: true, Reason: reason, UntilTS: until.UTC().Format(time.RFC3339)}, nil
}

// ---------- Notify Config ----------

type NotifyConfigDTO struct {
	Enabled        bool     `json:"enabled"`
	WebhookURLs    []string `json:"webhook_urls"`
	TelegramToken  string   `json:"telegram_bot_token"`
	TelegramChatID string   `json:"telegram_chat_id"`
	CooldownMin    int      `json:"cooldown_min"`
}

func (s *AdminService) GetNotifyConfig(ctx context.Context, tenantID int64) (NotifyConfigDTO, error) {
	c, err := s.NotifyRepo.GetTenantNotifyConfig(ctx, tenantID)
	if err != nil { return NotifyConfigDTO{}, err }
	return NotifyConfigDTO{Enabled: c.Enabled, WebhookURLs: c.WebhookURLs, TelegramToken: c.TelegramToken, TelegramChatID: c.TelegramChatID, CooldownMin: c.CooldownMin}, nil
}

func (s *AdminService) UpdateNotifyConfig(ctx context.Context, tenantID int64, dto NotifyConfigDTO) error {
	if dto.CooldownMin <= 0 { dto.CooldownMin = 180 }
	c := repo.TenantNotifyConfig{Enabled: dto.Enabled, WebhookURLs: dto.WebhookURLs, TelegramToken: strings.TrimSpace(dto.TelegramToken), TelegramChatID: strings.TrimSpace(dto.TelegramChatID), CooldownMin: dto.CooldownMin}
	return s.NotifyRepo.UpsertTenantNotifyConfig(ctx, tenantID, c)
}
