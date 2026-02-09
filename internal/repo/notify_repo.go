package repo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotifyRepo struct{ db *pgxpool.Pool }

func NewNotifyRepo(db *pgxpool.Pool) *NotifyRepo { return &NotifyRepo{db: db} }

type TenantNotifyConfig struct {
	Enabled        bool
	WebhookURLs    []string
	TelegramToken  string
	TelegramChatID string
	CooldownMin    int
}

func (r *NotifyRepo) GetTenantNotifyConfig(ctx context.Context, tenantID int64) (TenantNotifyConfig, error) {
	const q = `
select enabled, webhook_urls, telegram_bot_token, telegram_chat_id, cooldown_min
from tenant_notify_config
where tenant_id=$1;`
	var c TenantNotifyConfig
	err := r.db.QueryRow(ctx, q, tenantID).Scan(&c.Enabled, &c.WebhookURLs, &c.TelegramToken, &c.TelegramChatID, &c.CooldownMin)
	if err != nil {
		return TenantNotifyConfig{}, err
	}
	return c, nil
}

func (r *NotifyRepo) UpsertTenantNotifyConfig(ctx context.Context, tenantID int64, c TenantNotifyConfig) error {
	const q = `
insert into tenant_notify_config(tenant_id, enabled, webhook_urls, telegram_bot_token, telegram_chat_id, cooldown_min, updated_at)
values ($1,$2,$3,$4,$5,$6, now())
on conflict (tenant_id) do update
set enabled=excluded.enabled,
    webhook_urls=excluded.webhook_urls,
    telegram_bot_token=excluded.telegram_bot_token,
    telegram_chat_id=excluded.telegram_chat_id,
    cooldown_min=excluded.cooldown_min,
    updated_at=now();`
	_, err := r.db.Exec(ctx, q, tenantID, c.Enabled, c.WebhookURLs, c.TelegramToken, c.TelegramChatID, c.CooldownMin)
	return err
}
