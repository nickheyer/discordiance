package reporter

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// InternalReporter is the built-in soft reporter that writes reports to the DB.
// It is always active on every pipeline and cannot be removed.
type InternalReporter struct {
	db *gorm.DB
}

// NewInternalReporter creates the built-in soft reporter.
func NewInternalReporter(db *gorm.DB) *InternalReporter {
	return &InternalReporter{db: db}
}

func (r *InternalReporter) Type() v1.ReporterType {
	return v1.ReporterType_REPORTER_TYPE_INTERNAL
}

func (r *InternalReporter) Deliver(_ context.Context, _ *models.Reporter, insights []models.Insight) error {
	if len(insights) == 0 {
		return nil
	}

	pipelineID := insights[0].PipelineID

	// Look up product ID from pipeline.
	var pipeline models.Pipeline
	if err := r.db.Select("product_id").First(&pipeline, "id = ?", pipelineID).Error; err != nil {
		return err
	}

	var hot, burn, cold int32
	entries := make([]models.ReportEntry, 0, len(insights))

	for _, ins := range insights {
		switch v1.InsightState(ins.State) {
		case v1.InsightState_INSIGHT_STATE_HOT:
			hot++
		case v1.InsightState_INSIGHT_STATE_BURN:
			burn++
		case v1.InsightState_INSIGHT_STATE_COLD:
			cold++
		}

		preview := ins.Content
		if len(preview) > 200 {
			preview = preview[:200]
		}

		entries = append(entries, models.ReportEntry{
			ID:                   uuid.NewString(),
			InsightID:            ins.ID,
			State:                ins.State,
			ContentPreview:       preview,
			ClassificationReason: ins.ClassificationReason,
			PlatformID:           ins.PlatformID,
			SourceURL:            ins.SourceURL,
		})
	}

	report := models.Report{
		ID:            uuid.NewString(),
		PipelineID:    pipelineID,
		ProductID:     pipeline.ProductID,
		TotalInsights: int32(len(insights)),
		HotCount:      hot,
		BurnCount:     burn,
		ColdCount:     cold,
		Entries:       entries,
	}

	if err := r.db.Create(&report).Error; err != nil {
		slog.Error("internal reporter: failed to write report", "error", err, "pipeline_id", pipelineID)
		return err
	}

	slog.Info("internal reporter: report created",
		"report_id", report.ID,
		"pipeline_id", pipelineID,
		"total", len(insights),
		"hot", hot, "burn", burn, "cold", cold)

	return nil
}
