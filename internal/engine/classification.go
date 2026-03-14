package engine

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

const (
	classifyPollInterval = 5 * time.Second
	classifyBatchLimit   = 50
)

// classification polls for RAW insights, batches them, sends to agents, and updates state.
func classification(ctx context.Context, db *gorm.DB, client *agent.Client, pipeline *models.Pipeline, agents []models.Agent, product *models.Product) {
	ticker := time.NewTicker(classifyPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			classifyBatch(ctx, db, client, pipeline, agents, product)
		}
	}
}

func classifyBatch(ctx context.Context, db *gorm.DB, client *agent.Client, pipeline *models.Pipeline, agents []models.Agent, product *models.Product) {
	if len(agents) == 0 {
		return
	}

	var insights []models.Insight
	query := db.Where("pipeline_id = ? AND state = ?", pipeline.ID, int32(v1.InsightState_INSIGHT_STATE_RAW)).
		Order("ingested_at ASC")

	batchMode := v1.BatchMode(pipeline.BatchMode)

	switch batchMode {
	case v1.BatchMode_BATCH_MODE_SINGLE:
		query.Limit(1).Find(&insights)
	case v1.BatchMode_BATCH_MODE_FIXED_SIZE:
		limit := int(pipeline.BatchSize)
		if limit <= 0 {
			limit = 10
		}
		query.Limit(limit).Find(&insights)
	default:
		// For conversation/author/time_window modes, fetch a batch and let the agent handle grouping.
		query.Limit(classifyBatchLimit).Find(&insights)
	}

	if len(insights) == 0 {
		return
	}

	// Round-robin across agents.
	agentIdx := 0
	for _, a := range agents {
		_ = a
		agentIdx++
	}
	selectedAgent := agents[0]

	results, err := client.Classify(ctx, &selectedAgent, product, insights)
	if err != nil {
		slog.Error("classification: agent error",
			"pipeline_id", pipeline.ID,
			"agent_id", selectedAgent.ID,
			"error", err)
		return
	}

	now := time.Now()
	for _, r := range results {
		if err := db.Model(&models.Insight{}).
			Where("id = ?", r.InsightID).
			Updates(map[string]interface{}{
				"state":                 int32(r.State),
				"classification_reason": r.Reason,
				"classifying_agent_id":  selectedAgent.ID,
				"classified_at":         now,
			}).Error; err != nil {
			slog.Error("classification: failed to update insight",
				"insight_id", r.InsightID,
				"error", err)
		}
	}

	slog.Info("classification: batch complete",
		"pipeline_id", pipeline.ID,
		"classified", len(results),
		"agent_id", selectedAgent.ID)
}
