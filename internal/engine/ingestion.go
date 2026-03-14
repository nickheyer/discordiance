package engine

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// ingestion pulls content from platform adapters and writes RAW insights to the DB.
func ingestion(ctx context.Context, db *gorm.DB, pipelineID string, adapters []platform.Adapter) {
	contentCh := make(chan platform.RawContent, 100)

	// Start all adapters, each pushing to the shared channel.
	for _, adapter := range adapters {
		a := adapter
		go func() {
			if err := a.Start(ctx, contentCh); err != nil {
				slog.Error("ingestion: adapter error",
					"pipeline_id", pipelineID,
					"type", a.Type(),
					"error", err)
			}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-contentCh:
			// Dedup by source_id within this pipeline.
			if raw.SourceID != "" {
				var count int64
				db.Model(&models.Insight{}).
					Where("pipeline_id = ? AND source_id = ?", pipelineID, raw.SourceID).
					Count(&count)
				if count > 0 {
					continue
				}
			}

			now := time.Now()
			insight := models.Insight{
				ID:              uuid.NewString(),
				PipelineID:      pipelineID,
				PlatformID:      raw.PlatformID,
				State:           int32(v1.InsightState_INSIGHT_STATE_RAW),
				Medium:          int32(raw.Medium),
				Content:         raw.Content,
				Author:          raw.Author,
				SourceURL:       raw.SourceURL,
				SourceID:        raw.SourceID,
				ConversationID:  raw.ConversationID,
				SourceTimestamp: raw.Timestamp,
				IngestedAt:      now,
			}

			if err := db.Create(&insight).Error; err != nil {
				slog.Error("ingestion: failed to store insight",
					"pipeline_id", pipelineID,
					"error", err)
			}
		}
	}
}
