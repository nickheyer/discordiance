package engine

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	"github.com/nickheyer/discordiance/internal/reporter"
	"gorm.io/gorm"
)

// Pipeline processes messages for a single product: Platform → Agent → Reporter.
type Pipeline struct {
	db               *gorm.DB
	product          models.Product
	platformConfigID uint
	platform         platform.Platform
	agent            *agent.Agent
	reporters        []reporter.Reporter
	msgChan          chan models.Message
	batchSize        int
	batchTimeout     time.Duration
	logger           *slog.Logger
}

// PipelineConfig holds the configuration for creating a pipeline.
type PipelineConfig struct {
	DB               *gorm.DB
	Product          models.Product
	PlatformConfigID uint
	Platform         platform.Platform
	Agent            *agent.Agent
	Reporters        []reporter.Reporter
	BatchSize        int
	BatchTimeout     time.Duration
}

func NewPipeline(cfg PipelineConfig) *Pipeline {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 10
	}
	if cfg.BatchTimeout <= 0 {
		cfg.BatchTimeout = 30 * time.Second
	}

	logger := slog.With("product", cfg.Product.Name, "product_id", cfg.Product.ID)
	logger.Info("pipeline created", "batch_size", cfg.BatchSize, "batch_timeout", cfg.BatchTimeout,
		"platform_config_id", cfg.PlatformConfigID, "reporters", len(cfg.Reporters))

	return &Pipeline{
		db:               cfg.DB,
		product:          cfg.Product,
		platformConfigID: cfg.PlatformConfigID,
		platform:         cfg.Platform,
		agent:            cfg.Agent,
		reporters:        cfg.Reporters,
		msgChan:          make(chan models.Message, 100),
		batchSize:        cfg.BatchSize,
		batchTimeout:     cfg.BatchTimeout,
		logger:           logger,
	}
}

// Start begins listening on the platform and processing messages.
func (p *Pipeline) Start(ctx context.Context) error {
	p.logger.Info("pipeline starting")

	// Start the platform, routing messages into our channel
	if err := p.platform.Start(ctx, func(msg models.Message) {
		msg.ProductID = p.product.ID
		p.logger.Info("message received from platform", "external_id", msg.ExternalID,
			"channel_id", msg.ChannelID, "author", msg.AuthorName)
		p.msgChan <- msg
	}); err != nil {
		p.logger.Error("failed to start platform", "error", err)
		return fmt.Errorf("pipeline: starting platform: %w", err)
	}

	// Start the processing loop in a goroutine
	go p.processLoop(ctx)

	p.logger.Info("pipeline started")
	return nil
}

// Stop shuts down the pipeline.
func (p *Pipeline) Stop(ctx context.Context) error {
	p.logger.Info("pipeline stopping")

	if err := p.platform.Stop(ctx); err != nil {
		p.logger.Error("stopping platform", "error", err)
	} else {
		p.logger.Info("platform stopped")
	}

	for _, r := range p.reporters {
		if err := r.Close(ctx); err != nil {
			p.logger.Error("closing reporter", "type", r.Type(), "error", err)
		} else {
			p.logger.Info("reporter closed", "type", r.Type())
		}
	}
	close(p.msgChan)
	p.logger.Info("pipeline stopped")
	return nil
}

// RunBackfill discovers channels and walks through history using saved cursors.
func (p *Pipeline) RunBackfill(ctx context.Context) {
	p.logger.Info("starting backfill")

	channels, err := p.platform.DiscoverChannels(ctx)
	if err != nil {
		p.logger.Error("backfill: discovering channels", "error", err)
		return
	}

	p.logger.Info("backfill: channels discovered", "count", len(channels), "channels", channels)

	for _, channelID := range channels {
		cursor := p.loadCursor(channelID)
		p.logger.Info("backfill: starting channel", "channel_id", channelID, "cursor", cursor)

		count := 0
		err := p.platform.Backfill(ctx, channelID, cursor, func(msg models.Message) {
			msg.ProductID = p.product.ID
			p.msgChan <- msg
			count++
			if count%100 == 0 {
				p.saveCursor(channelID, msg.ExternalID)
				p.logger.Info("backfill: progress", "channel_id", channelID, "messages", count, "cursor", msg.ExternalID)
			}
			cursor = msg.ExternalID
		})

		// Save final cursor regardless of error
		if cursor != "" && cursor != "0" {
			p.saveCursor(channelID, cursor)
		}

		if err != nil {
			p.logger.Error("backfill: channel error, continuing", "channel_id", channelID, "messages_so_far", count, "error", err)
			continue
		}

		p.logger.Info("backfill: channel complete", "channel_id", channelID, "messages", count)
	}

	p.logger.Info("backfill complete")
}

func (p *Pipeline) loadCursor(channelID string) string {
	var cursor models.BackfillCursor
	err := p.db.Where("platform_config_id = ? AND channel_id = ?", p.platformConfigID, channelID).
		First(&cursor).Error
	if err != nil {
		p.logger.Debug("no existing cursor found", "channel_id", channelID)
		return ""
	}
	p.logger.Info("loaded cursor", "channel_id", channelID, "last_message_id", cursor.LastMessageID)
	return cursor.LastMessageID
}

func (p *Pipeline) saveCursor(channelID string, messageID string) {
	cursor := models.BackfillCursor{
		PlatformConfigID: p.platformConfigID,
		ChannelID:        channelID,
		LastMessageID:    messageID,
	}
	result := p.db.
		Where("platform_config_id = ? AND channel_id = ?", p.platformConfigID, channelID).
		Assign(models.BackfillCursor{LastMessageID: messageID}).
		FirstOrCreate(&cursor)
	if result.Error != nil {
		p.logger.Error("backfill: saving cursor", "channel_id", channelID, "error", result.Error)
	} else {
		p.logger.Debug("cursor saved", "channel_id", channelID, "message_id", messageID)
	}
}

func (p *Pipeline) processLoop(ctx context.Context) {
	p.logger.Info("process loop started", "batch_size", p.batchSize, "batch_timeout", p.batchTimeout)

	batch := make([]models.Message, 0, p.batchSize)
	timer := time.NewTimer(p.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("process loop context cancelled", "remaining_batch", len(batch))
			if len(batch) > 0 {
				p.processBatch(context.Background(), batch)
			}
			p.logger.Info("process loop exited")
			return

		case msg, ok := <-p.msgChan:
			if !ok {
				p.logger.Info("message channel closed, process loop exiting")
				return
			}
			// Persist the message
			if err := p.persistMessage(&msg); err != nil {
				p.logger.Info("skipping message", "external_id", msg.ExternalID, "reason", err)
				continue
			}
			batch = append(batch, msg)
			p.logger.Info("message added to batch", "external_id", msg.ExternalID, "batch_size", len(batch), "batch_max", p.batchSize)

			if len(batch) >= p.batchSize {
				p.logger.Info("batch full, processing", "size", len(batch))
				p.processBatch(ctx, batch)
				batch = make([]models.Message, 0, p.batchSize)
				timer.Reset(p.batchTimeout)
			}

		case <-timer.C:
			if len(batch) > 0 {
				p.logger.Info("batch timeout, processing", "size", len(batch))
				p.processBatch(ctx, batch)
				batch = make([]models.Message, 0, p.batchSize)
			}
			timer.Reset(p.batchTimeout)
		}
	}
}

func (p *Pipeline) persistMessage(msg *models.Message) error {
	// Deduplicate by external ID
	if msg.ExternalID != "" {
		var count int64
		p.db.Model(&models.Message{}).
			Where("external_id = ? AND platform_type = ?", msg.ExternalID, msg.PlatformType).
			Count(&count)
		if count > 0 {
			return fmt.Errorf("duplicate message: %s", msg.ExternalID)
		}
	}
	if err := p.db.Create(msg).Error; err != nil {
		p.logger.Error("failed to persist message", "external_id", msg.ExternalID, "error", err)
		return err
	}
	p.logger.Info("message persisted", "id", msg.ID, "external_id", msg.ExternalID, "author", msg.AuthorName)
	return nil
}

func (p *Pipeline) processBatch(ctx context.Context, batch []models.Message) {
	p.logger.Info("processing batch", "size", len(batch))
	start := time.Now()

	result, err := p.agent.Analyze(ctx, batch)
	elapsed := time.Since(start)
	if err != nil {
		p.logger.Error("agent analysis failed", "error", err, "duration", elapsed, "batch_size", len(batch))
		return
	}

	p.logger.Info("agent analysis complete", "issues_detected", len(result.Issues), "duration", elapsed, "batch_size", len(batch))

	// Collect message IDs for source tracking
	msgIDs := make([]string, len(batch))
	for i, msg := range batch {
		msgIDs[i] = fmt.Sprintf("%d", msg.ID)
	}
	sourceMsgIDs := strings.Join(msgIDs, ",")

	for _, detected := range result.Issues {
		p.logger.Info("processing detected issue", "title", detected.Title, "severity", detected.Severity, "category", detected.Category)

		issue := models.Issue{
			ProductID:    p.product.ID,
			Title:        detected.Title,
			Description:  detected.Description,
			Severity:     detected.Severity,
			Category:     detected.Category,
			Fingerprint:  fingerprint(detected.Title, detected.Category),
			Status:       "open",
			SourceMsgIDs: sourceMsgIDs,
		}

		// Check for duplicate by fingerprint
		var existing models.Issue
		if err := p.db.Where("fingerprint = ? AND product_id = ? AND status = ?",
			issue.Fingerprint, p.product.ID, "open").First(&existing).Error; err == nil {
			p.logger.Info("duplicate issue found, skipping", "title", issue.Title, "existing_id", existing.ID, "fingerprint", issue.Fingerprint)
			continue
		}

		if err := p.db.Create(&issue).Error; err != nil {
			p.logger.Error("creating issue", "title", issue.Title, "error", err)
			continue
		}

		p.logger.Info("issue created", "id", issue.ID, "title", issue.Title, "severity", issue.Severity)

		// Mark messages as processed
		p.db.Model(&models.Message{}).Where("id IN ?", msgIDs).Update("processed", true)
		p.logger.Info("messages marked as processed", "count", len(msgIDs))

		// File to all reporters
		for _, r := range p.reporters {
			p.logger.Info("checking reporter for duplicate", "reporter", r.Type(), "issue_title", issue.Title)
			// Check for duplicate in external system
			dupID, err := r.FindDuplicate(ctx, issue)
			if err != nil {
				p.logger.Error("checking duplicate", "reporter", r.Type(), "error", err)
			}
			if dupID != "" {
				p.logger.Info("external duplicate found, skipping reporter", "reporter", r.Type(), "external_id", dupID)
				continue
			}

			p.logger.Info("filing report", "reporter", r.Type(), "issue_id", issue.ID, "issue_title", issue.Title)
			report, err := r.FileReport(ctx, issue)
			if err != nil {
				p.logger.Error("filing report failed", "reporter", r.Type(), "error", err)
				continue
			}

			if err := p.db.Create(report).Error; err != nil {
				p.logger.Error("saving report", "error", err)
			} else {
				p.logger.Info("report saved", "reporter", r.Type(), "external_url", report.ExternalURL)
			}
		}
	}

	p.logger.Info("batch processing complete", "batch_size", len(batch), "issues_created", len(result.Issues), "total_duration", time.Since(start))
}

func fingerprint(title, category string) string {
	data := strings.ToLower(title) + "|" + strings.ToLower(category)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8])
}
