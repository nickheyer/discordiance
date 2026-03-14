package engine

import (
	"context"
	"log/slog"

	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	"github.com/nickheyer/discordiance/internal/reporter"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// runner manages the lifecycle of a single pipeline's execution.
type runner struct {
	pipelineID       string
	db               *gorm.DB
	agentClient      *agent.Client
	platformRegistry *platform.Registry
	reporterRegistry *reporter.Registry
	internalReporter *reporter.InternalReporter
	cancel           context.CancelFunc
}

func newRunner(pipelineID string, db *gorm.DB, agentClient *agent.Client, platformReg *platform.Registry, reporterReg *reporter.Registry, internalRep *reporter.InternalReporter) *runner {
	return &runner{
		pipelineID:       pipelineID,
		db:               db,
		agentClient:      agentClient,
		platformRegistry: platformReg,
		reporterRegistry: reporterReg,
		internalReporter: internalRep,
	}
}

func (r *runner) start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(ctx)

	// Load pipeline with all associations.
	var pipeline models.Pipeline
	if err := r.db.Preload("Agents").Preload("Platforms").Preload("Reporters").
		First(&pipeline, "id = ?", r.pipelineID).Error; err != nil {
		slog.Error("runner: failed to load pipeline", "pipeline_id", r.pipelineID, "error", err)
		return
	}

	// Load product with contexts.
	var product models.Product
	if err := r.db.Preload("Contexts").First(&product, "id = ?", pipeline.ProductID).Error; err != nil {
		slog.Error("runner: failed to load product", "pipeline_id", r.pipelineID, "error", err)
		return
	}

	// Load full agent models.
	var agents []models.Agent
	for _, pa := range pipeline.Agents {
		var a models.Agent
		if err := r.db.First(&a, "id = ?", pa.AgentID).Error; err == nil {
			agents = append(agents, a)
		}
	}

	// Create platform adapters with full typed configs.
	var adapters []platform.Adapter
	for _, pp := range pipeline.Platforms {
		p, err := loadPlatformWithConfig(r.db, pp.PlatformID)
		if err != nil {
			continue
		}
		adapter, err := r.platformRegistry.Create(p)
		if err != nil {
			slog.Warn("runner: no adapter for platform",
				"pipeline_id", r.pipelineID,
				"platform_id", p.ID,
				"error", err)
			continue
		}
		adapters = append(adapters, adapter)
	}

	// Load full reporter models with filters and typed configs.
	var reporters []models.Reporter
	for _, pr := range pipeline.Reporters {
		rep, err := loadReporterWithConfig(r.db, pr.ReporterID)
		if err == nil {
			reporters = append(reporters, *rep)
		}
	}

	slog.Info("runner: starting pipeline",
		"pipeline_id", r.pipelineID,
		"adapters", len(adapters),
		"agents", len(agents),
		"reporters", len(reporters))

	// Launch the three stages.
	if len(adapters) > 0 {
		go ingestion(ctx, r.db, r.pipelineID, adapters)
	}

	go classification(ctx, r.db, r.agentClient, &pipeline, agents, &product)
	go dispatch(ctx, r.db, r.pipelineID, reporters, r.internalReporter, r.reporterRegistry)
}

func (r *runner) stop() {
	if r.cancel != nil {
		r.cancel()
	}
	slog.Info("runner: stopped pipeline", "pipeline_id", r.pipelineID)
}

// loadPlatformWithConfig loads a platform and its type-specific config table.
func loadPlatformWithConfig(db *gorm.DB, id string) (*models.Platform, error) {
	var p models.Platform
	if err := db.First(&p, "id = ?", id).Error; err != nil {
		return nil, err
	}

	switch v1.PlatformType(p.Type) {
	case v1.PlatformType_PLATFORM_TYPE_DISCORD:
		var cfg models.DiscordPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.DiscordConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_REDDIT:
		var cfg models.RedditPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.RedditConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_TWITTER:
		var cfg models.TwitterPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.TwitterConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_LINKEDIN:
		var cfg models.LinkedInPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.LinkedInConfig = &cfg
		}
	case v1.PlatformType_PLATFORM_TYPE_GITHUB:
		var cfg models.GitHubPlatformConfig
		if err := db.First(&cfg, "platform_id = ?", id).Error; err == nil {
			p.GitHubConfig = &cfg
		}
	}

	return &p, nil
}

func loadReporterWithConfig(db *gorm.DB, id string) (*models.Reporter, error) {
	var r models.Reporter
	if err := db.Preload("FilterStates").Preload("FilterPlatforms").
		First(&r, "id = ?", id).Error; err != nil {
		return nil, err
	}

	switch v1.ReporterType(r.Type) {
	case v1.ReporterType_REPORTER_TYPE_WEBHOOK:
		var cfg models.WebhookReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.WebhookConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_EMAIL:
		var cfg models.EmailReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.EmailConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_DISCORD:
		var cfg models.DiscordReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.DiscordConfig = &cfg
		}
	case v1.ReporterType_REPORTER_TYPE_GITHUB_ISSUE:
		var cfg models.GitHubIssueReporterConfig
		if err := db.First(&cfg, "reporter_id = ?", id).Error; err == nil {
			r.GitHubIssueConfig = &cfg
		}
	}

	return &r, nil
}
