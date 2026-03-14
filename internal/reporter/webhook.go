package reporter

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// WebhookReporter delivers classified insights via HTTP POST.
type WebhookReporter struct {
	client *http.Client
}

// NewWebhookReporter creates a webhook reporter handler.
func NewWebhookReporter() *WebhookReporter {
	return &WebhookReporter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (w *WebhookReporter) Type() v1.ReporterType {
	return v1.ReporterType_REPORTER_TYPE_WEBHOOK
}

func (w *WebhookReporter) Deliver(ctx context.Context, reporter *models.Reporter, insights []models.Insight) error {
	if reporter.WebhookConfig == nil {
		return fmt.Errorf("webhook config is nil for reporter %s", reporter.ID)
	}

	cfg := reporter.WebhookConfig

	payload := webhookPayload{
		ReporterID: reporter.ID,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Insights:   make([]webhookInsight, len(insights)),
	}

	for i, ins := range insights {
		payload.Insights[i] = webhookInsight{
			ID:         ins.ID,
			PipelineID: ins.PipelineID,
			PlatformID: ins.PlatformID,
			State:      v1.InsightState_name[ins.State],
			Medium:     v1.InsightMedium_name[ins.Medium],
			Content:    ins.Content,
			Author:     ins.Author,
			SourceURL:  ins.SourceURL,
			Reason:     ins.ClassificationReason,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Discordiance/1.0")

	// HMAC-SHA256 signature if secret is configured.
	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Discordiance-Signature", "sha256="+sig)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}

	slog.Info("webhook: delivered",
		"reporter_id", reporter.ID,
		"url", cfg.URL,
		"insights", len(insights),
		"status", resp.StatusCode)

	return nil
}

type webhookPayload struct {
	ReporterID string           `json:"reporter_id"`
	Timestamp  string           `json:"timestamp"`
	Insights   []webhookInsight `json:"insights"`
}

type webhookInsight struct {
	ID         string `json:"id"`
	PipelineID string `json:"pipeline_id"`
	PlatformID string `json:"platform_id"`
	State      string `json:"state"`
	Medium     string `json:"medium"`
	Content    string `json:"content"`
	Author     string `json:"author"`
	SourceURL  string `json:"source_url"`
	Reason     string `json:"reason"`
}
