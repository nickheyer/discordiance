package engine

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/reporter"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

const (
	dispatchPollInterval = 5 * time.Second
	dispatchBatchLimit   = 100
)

// dispatch polls for newly classified insights and routes them to reporters.
func dispatch(ctx context.Context, db *gorm.DB, pipelineID string, reporters []models.Reporter, internalReporter *reporter.InternalReporter, registry *reporter.Registry) {
	// Track the last dispatched time to avoid re-processing.
	var lastDispatch time.Time

	ticker := time.NewTicker(dispatchPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			lastDispatch = dispatchBatch(ctx, db, pipelineID, reporters, internalReporter, registry, lastDispatch)
		}
	}
}

func dispatchBatch(ctx context.Context, db *gorm.DB, pipelineID string, reporters []models.Reporter, internalReporter *reporter.InternalReporter, registry *reporter.Registry, since time.Time) time.Time {
	var insights []models.Insight
	query := db.Where("pipeline_id = ? AND state != ? AND classified_at IS NOT NULL",
		pipelineID, int32(v1.InsightState_INSIGHT_STATE_RAW))

	if !since.IsZero() {
		query = query.Where("classified_at > ?", since)
	}

	query.Order("classified_at ASC").Limit(dispatchBatchLimit).Find(&insights)

	if len(insights) == 0 {
		return since
	}

	// Track the latest classified_at as our new watermark.
	latest := since
	for _, ins := range insights {
		if ins.ClassifiedAt != nil && ins.ClassifiedAt.After(latest) {
			latest = *ins.ClassifiedAt
		}
	}

	// Always deliver to internal soft reporter.
	if err := internalReporter.Deliver(ctx, nil, insights); err != nil {
		slog.Error("dispatch: internal reporter error",
			"pipeline_id", pipelineID,
			"error", err)
	}

	// Deliver to external reporters based on their filters.
	for i := range reporters {
		r := &reporters[i]
		filtered := filterInsights(insights, r)
		if len(filtered) == 0 {
			continue
		}

		handler, err := registry.Get(v1.ReporterType(r.Type))
		if err != nil {
			slog.Error("dispatch: no handler for reporter",
				"reporter_id", r.ID,
				"type", r.Type,
				"error", err)
			continue
		}

		if err := handler.Deliver(ctx, r, filtered); err != nil {
			slog.Error("dispatch: reporter delivery error",
				"reporter_id", r.ID,
				"pipeline_id", pipelineID,
				"error", err)
		}
	}

	return latest
}

// filterInsights applies a reporter's filter rules to a set of insights.
func filterInsights(insights []models.Insight, r *models.Reporter) []models.Insight {
	// Build lookup sets from filter tables.
	stateFilter := make(map[int32]bool, len(r.FilterStates))
	for _, s := range r.FilterStates {
		stateFilter[s.State] = true
	}
	platformFilter := make(map[string]bool, len(r.FilterPlatforms))
	for _, p := range r.FilterPlatforms {
		platformFilter[p.PlatformID] = true
	}

	var result []models.Insight
	for _, ins := range insights {
		// State filter: empty = all non-RAW (already filtered in query).
		if len(stateFilter) > 0 && !stateFilter[ins.State] {
			continue
		}
		// Platform filter: empty = all platforms.
		if len(platformFilter) > 0 && !platformFilter[ins.PlatformID] {
			continue
		}
		result = append(result, ins)
	}
	return result
}
