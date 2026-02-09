package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"idatariver-finapi/internal/notify"
	"idatariver-finapi/internal/repo"
	"idatariver-finapi/internal/service"
)

type HealthCheckJob struct {
	JobRunRepo  *repo.JobRunRepo
	TenantRepo  *repo.TenantRepo
	NotifyRepo  *repo.NotifyRepo
	HealthRepo  *repo.HealthRepo
	API         *service.APIService
}

func (j *HealthCheckJob) RunAllTenants(ctx context.Context) error {
	runID, _ := j.JobRunRepo.Start(ctx, "health_check_all")
	start := time.Now()

	tenants, err := j.TenantRepo.ListActive(ctx)
	if err != nil {
		_ = j.JobRunRepo.Finish(ctx, runID, "failed", err.Error())
		return err
	}

	for _, t := range tenants {
		_ = j.runTenant(ctx, t.ID, t.Code)
	}
	_ = j.JobRunRepo.Finish(ctx, runID, "success", fmt.Sprintf("tenants=%d elapsed=%s", len(tenants), time.Since(start)))
	return nil
}

func (j *HealthCheckJob) runTenant(ctx context.Context, tenantID int64, tenantCode string) error {
	rep, err := j.API.DataHealth(ctx, tenantID, []string{}, 20, true)
	if err != nil {
		return err
	}

	statusLevel := "ok"
	if rep.Summary.Stale > 0 || rep.Summary.Missing > 0 {
		statusLevel = "problem"
	}
	severity := "ok"
	if rep.Summary.Missing > 0 {
		severity = "critical"
	} else if rep.Summary.Stale > 0 {
		severity = "warning"
	}
	statusKey := buildStatusKey(rep)

	id, err := j.HealthRepo.InsertWithStatus(ctx, tenantID, repo.HealthCheckInsert{
		CheckTS:  time.Now().UTC(),
		Total:   rep.Summary.Total,
		Fresh:   rep.Summary.Fresh,
		Stale:   rep.Summary.Stale,
		Missing: rep.Summary.Missing,
		Worst:   rep.Worst,
		Items:   rep.Items,
	}, statusLevel, statusKey, severity)
	if err != nil {
		return err
	}

	// Snooze check (only suppress problem notifications)
	snoozed, reason, until, _ := j.HealthRepo.IsSnoozed(ctx, tenantID)
	if snoozed && statusLevel == "problem" {
		_ = j.HealthRepo.MarkNotified(ctx, tenantID, id, []string{}, fmt.Sprintf("snoozed until %s (%s)", until.UTC().Format(time.RFC3339), reason))
		return nil
	}

	cfg, err := j.NotifyRepo.GetTenantNotifyConfig(ctx, tenantID)
	if err != nil {
		// if config missing, just skip notify
		return nil
	}
	n := notify.New(notify.Config{
		Enabled:        cfg.Enabled,
		Webhooks:       cfg.WebhookURLs,
		TelegramToken:  cfg.TelegramToken,
		TelegramChatID: cfg.TelegramChatID,
	})

	last, _ := j.HealthRepo.GetLast(ctx, tenantID)
	cooldown := time.Duration(cfg.CooldownMin) * time.Minute

	shouldNotify, isRecovery := decideNotify(last, statusLevel, statusKey, cooldown)
	if !shouldNotify || len(n.Targets()) == 0 {
		return nil
	}

	var msg string
	if isRecovery {
		msg = fmt.Sprintf("[DataHealth RECOVERED] tenant=%s id=%d total=%d fresh=%d stale=%d missing=%d", tenantCode, id, rep.Summary.Total, rep.Summary.Fresh, rep.Summary.Stale, rep.Summary.Missing)
	} else {
		msg = buildHealthMessage(tenantCode, severity, rep, id)
	}

	nerr := n.Notify(ctx, msg, rep)
	if nerr != nil {
		_ = j.HealthRepo.MarkNotified(ctx, tenantID, id, n.Targets(), nerr.Error())
	} else {
		_ = j.HealthRepo.MarkNotified(ctx, tenantID, id, n.Targets(), "")
	}
	return nil
}

func decideNotify(last *repo.LastHealthCheck, statusLevel, statusKey string, cooldown time.Duration) (bool, bool) {
	if last == nil {
		return statusLevel == "problem", false
	}
	if last.StatusLevel != statusLevel {
		return true, (last.StatusLevel == "problem" && statusLevel == "ok")
	}
	if statusLevel == "problem" {
		if last.StatusKey != statusKey {
			if time.Since(last.CheckTS) >= cooldown {
				return true, false
			}
		}
	}
	return false, false
}

func buildStatusKey(rep service.DataHealthReport) string {
	w := []string{}
	for i := 0; i < len(rep.Worst) && i < 5; i++ {
		w = append(w, rep.Worst[i].Symbol)
	}
	return fmt.Sprintf("stale=%d|missing=%d|worst=%s", rep.Summary.Stale, rep.Summary.Missing, strings.Join(w, ","))
}

func buildHealthMessage(tenantCode string, severity string, rep service.DataHealthReport, id int64) string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "[DataHealth %s] tenant=%s id=%d\n", strings.ToUpper(severity), tenantCode, id)
	fmt.Fprintf(b, "total=%d fresh=%d stale=%d missing=%d\n", rep.Summary.Total, rep.Summary.Fresh, rep.Summary.Stale, rep.Summary.Missing)
	if len(rep.Worst) > 0 {
		b.WriteString("worst:\n")
		for i := 0; i < len(rep.Worst) && i < 8; i++ {
			w := rep.Worst[i]
			lag := "?"
			if w.LagHours != nil { lag = fmt.Sprintf("%.1fh", *w.LagHours) }
			fh := "?"
			if w.FreshnessHours != nil { fh = fmt.Sprintf("%.1fh", *w.FreshnessHours) }
			fmt.Fprintf(b, "- %s (%s) status=%s freshness=%s sla=%.0fh lag=%s last=%s\n", w.Symbol, w.Type, w.Status, fh, w.SLAHours, lag, w.LastCloseTS)
		}
	}
	return b.String()
}
