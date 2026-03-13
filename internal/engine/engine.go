package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	"github.com/nickheyer/discordiance/internal/reporter"
	"gorm.io/gorm"
)

// PipelineStatus represents the runtime state of a single pipeline.
type PipelineStatus struct {
	PipelineID   uint
	PipelineName string
	ProductName  string
	Running      bool
	Healthy      bool
}

// Engine is the top-level orchestrator that manages one Pipeline per pipeline record.
type Engine struct {
	mu        sync.Mutex
	db        *gorm.DB
	pipelines map[uint]*Pipeline // keyed by pipeline DB ID, only contains running pipelines
	ctx       context.Context
}

func New(db *gorm.DB) *Engine {
	slog.Info("engine created")
	return &Engine{
		db:        db,
		pipelines: make(map[uint]*Pipeline),
	}
}

// Setup stores the app context. Does NOT start any pipelines.
func (e *Engine) Setup(ctx context.Context) error {
	slog.Info("engine setup starting")
	e.ctx = ctx
	slog.Info("engine setup complete")
	return nil
}

// buildPipeline creates a Pipeline from a fully-loaded Pipeline model.
func (e *Engine) buildPipeline(ctx context.Context, pl models.Pipeline) (*Pipeline, error) {
	slog.Info("creating platform", "pipeline", pl.Name, "type", pl.Platform.Type, "platform_id", pl.Platform.ID)

	plat, err := platform.Create(pl.Platform.Type)
	if err != nil {
		return nil, fmt.Errorf("creating platform %s: %w", pl.Platform.Type, err)
	}
	if err := plat.Connect(ctx, pl.Platform); err != nil {
		return nil, fmt.Errorf("connecting platform %s: %w", pl.Platform.Type, err)
	}
	slog.Info("platform connected", "pipeline", pl.Name, "type", pl.Platform.Type)

	ag := agent.New(pl.Agent)
	slog.Info("agent created", "pipeline", pl.Name, "model", pl.Agent.Model)

	var reporters []reporterEntry
	for _, rc := range pl.Reporters {
		slog.Info("creating reporter", "pipeline", pl.Name, "type", rc.Type)
		r, err := reporter.Create(rc.Type)
		if err != nil {
			slog.Warn("skipping reporter", "type", rc.Type, "error", err)
			continue
		}
		if err := r.Init(ctx, rc); err != nil {
			slog.Warn("skipping reporter, init failed", "type", rc.Type, "error", err)
			continue
		}
		reporters = append(reporters, reporterEntry{reporter: r, config: rc})
		slog.Info("reporter initialized", "pipeline", pl.Name, "type", rc.Type)
	}

	batchSize := pl.Agent.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	batchTimeout := pl.Agent.BatchTimeout
	if batchTimeout <= 0 {
		batchTimeout = 30
	}

	slog.Info("pipeline build complete", "pipeline", pl.Name,
		"batch_size", batchSize, "batch_timeout_s", batchTimeout, "reporters", len(reporters))

	return NewPipeline(PipelineConfig{
		DB:         e.db,
		PipelineID: pl.ID,
		Product:    pl.Product,
		PlatformID: pl.Platform.ID,
		Platform:   plat,
		Agent:      ag,
		Reporters:  reporters,
		BatchSize:  batchSize,
		BatchTimeout: time.Duration(batchTimeout) * time.Second,
	}), nil
}

func (e *Engine) loadPipeline(pipelineID uint) (models.Pipeline, error) {
	var pl models.Pipeline
	if err := e.db.
		Preload("Product").
		Preload("Platform", "enabled = ?", true).
		Preload("Agent").
		Preload("Reporters", "enabled = ?", true).
		First(&pl, pipelineID).Error; err != nil {
		return pl, fmt.Errorf("engine: loading pipeline %d: %w", pipelineID, err)
	}
	return pl, nil
}

// StartPipeline builds and starts the pipeline.
func (e *Engine) StartPipeline(ctx context.Context, pipelineID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.pipelines[pipelineID]; ok {
		return fmt.Errorf("engine: pipeline %d is already running", pipelineID)
	}

	pl, err := e.loadPipeline(pipelineID)
	if err != nil {
		return err
	}

	if !pl.Enabled {
		return fmt.Errorf("engine: pipeline %s is disabled", pl.Name)
	}

	slog.Info("starting pipeline", "pipeline", pl.Name, "pipeline_id", pipelineID)

	pipeline, err := e.buildPipeline(e.ctx, pl)
	if err != nil {
		return fmt.Errorf("engine: building pipeline %s: %w", pl.Name, err)
	}

	if err := pipeline.Start(e.ctx); err != nil {
		return fmt.Errorf("engine: starting pipeline %s: %w", pl.Name, err)
	}

	e.pipelines[pipelineID] = pipeline
	slog.Info("pipeline started", "pipeline", pl.Name, "pipeline_id", pipelineID)
	return nil
}

// StopPipeline stops a running pipeline.
func (e *Engine) StopPipeline(ctx context.Context, pipelineID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.pipelines[pipelineID]
	if !ok {
		return fmt.Errorf("engine: no running pipeline %d", pipelineID)
	}

	slog.Info("stopping pipeline", "pipeline_id", pipelineID)
	if err := p.Stop(ctx); err != nil {
		slog.Error("stopping pipeline", "pipeline_id", pipelineID, "error", err)
	}
	delete(e.pipelines, pipelineID)
	slog.Info("pipeline stopped", "pipeline_id", pipelineID)
	return nil
}

// RestartPipeline stops the existing pipeline, reloads from DB, and starts a new one.
func (e *Engine) RestartPipeline(ctx context.Context, pipelineID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	slog.Info("restarting pipeline", "pipeline_id", pipelineID)

	if p, ok := e.pipelines[pipelineID]; ok {
		slog.Info("stopping existing pipeline", "pipeline_id", pipelineID)
		if err := p.Stop(ctx); err != nil {
			slog.Error("stopping pipeline for restart", "pipeline_id", pipelineID, "error", err)
		}
		delete(e.pipelines, pipelineID)
	}

	pl, err := e.loadPipeline(pipelineID)
	if err != nil {
		return err
	}

	if !pl.Enabled {
		slog.Info("pipeline disabled, not restarting", "pipeline", pl.Name)
		return nil
	}

	pipeline, err := e.buildPipeline(e.ctx, pl)
	if err != nil {
		return fmt.Errorf("engine: building pipeline %s: %w", pl.Name, err)
	}

	if err := pipeline.Start(e.ctx); err != nil {
		return fmt.Errorf("engine: starting pipeline %s: %w", pl.Name, err)
	}

	e.pipelines[pipelineID] = pipeline
	slog.Info("pipeline restarted", "pipeline", pl.Name, "pipeline_id", pipelineID)
	return nil
}

// BackfillPipeline triggers a backfill on a running pipeline.
func (e *Engine) BackfillPipeline(ctx context.Context, pipelineID uint) error {
	e.mu.Lock()
	p, ok := e.pipelines[pipelineID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("engine: no running pipeline %d", pipelineID)
	}

	slog.Info("triggering manual backfill", "pipeline_id", pipelineID)
	go p.RunBackfill(e.ctx)
	return nil
}

// Status returns the state of all enabled pipelines, indicating which are running.
func (e *Engine) Status() []PipelineStatus {
	e.mu.Lock()
	defer e.mu.Unlock()

	var dbPipelines []models.Pipeline
	e.db.Where("enabled = ?", true).Preload("Product").Find(&dbPipelines)

	statuses := make([]PipelineStatus, 0, len(dbPipelines))
	for _, pl := range dbPipelines {
		st := PipelineStatus{
			PipelineID:   pl.ID,
			PipelineName: pl.Name,
			ProductName:  pl.Product.Name,
		}
		if p, ok := e.pipelines[pl.ID]; ok {
			st.Running = true
			st.Healthy = p.platform.Healthy(context.Background())
		}
		statuses = append(statuses, st)
	}
	return statuses
}

// Stop shuts down all running pipelines (called on app shutdown).
func (e *Engine) Stop(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	slog.Info("engine stopping all pipelines", "count", len(e.pipelines))
	for id, p := range e.pipelines {
		if err := p.Stop(ctx); err != nil {
			slog.Error("stopping pipeline", "pipeline_id", id, "error", err)
		}
	}
	e.pipelines = make(map[uint]*Pipeline)
	slog.Info("engine stopped")
}

// Healthy returns true if all running pipelines are healthy.
func (e *Engine) Healthy() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, p := range e.pipelines {
		if !p.platform.Healthy(context.Background()) {
			return false
		}
	}
	return true
}
