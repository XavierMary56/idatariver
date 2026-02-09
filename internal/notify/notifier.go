package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Enabled        bool
	Webhooks       []string
	TelegramToken  string
	TelegramChatID string
}

type Notifier struct {
	cfg  Config
	http *http.Client
}

func New(cfg Config) *Notifier {
	return &Notifier{cfg: cfg, http: &http.Client{Timeout: 10 * time.Second}}
}

func (n *Notifier) Targets() []string {
	out := []string{}
	for _, w := range n.cfg.Webhooks {
		w = strings.TrimSpace(w)
		if w != "" {
			out = append(out, "webhook:"+w)
		}
	}
	if strings.TrimSpace(n.cfg.TelegramToken) != "" && strings.TrimSpace(n.cfg.TelegramChatID) != "" {
		out = append(out, "telegram:"+n.cfg.TelegramChatID)
	}
	return out
}

func (n *Notifier) Notify(ctx context.Context, text string, payload any) error {
	if !n.cfg.Enabled {
		return nil
	}
	var errs []string

	for _, wh := range n.cfg.Webhooks {
		wh = strings.TrimSpace(wh)
		if wh == "" {
			continue
		}
		if err := n.postWebhook(ctx, wh, text, payload); err != nil {
			errs = append(errs, "webhook:"+err.Error())
		}
	}

	if strings.TrimSpace(n.cfg.TelegramToken) != "" && strings.TrimSpace(n.cfg.TelegramChatID) != "" {
		if err := n.sendTelegram(ctx, text); err != nil {
			errs = append(errs, "telegram:"+err.Error())
		}
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func (n *Notifier) postWebhook(ctx context.Context, wh string, text string, payload any) error {
	body := map[string]any{"text": text, "payload": payload}
	b, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}

func (n *Notifier) sendTelegram(ctx context.Context, text string) error {
	endpoint := "https://api.telegram.org/bot" + n.cfg.TelegramToken + "/sendMessage"
	form := url.Values{}
	form.Set("chat_id", n.cfg.TelegramChatID)
	form.Set("text", text)
	form.Set("disable_web_page_preview", "true")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := n.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return errors.New(resp.Status)
	}
	return nil
}
