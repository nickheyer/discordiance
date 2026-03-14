package engine

import (
	"context"
	"log/slog"
	"sync"

	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	"github.com/nickheyer/discordiance/internal/reporter"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// Engine manages pipeline runners.
type Engine struct {
	db               *gorm.DB
	agentClient      *agent.Client
	platformRegistry *platform.Registry
	reporterRegistry *reporter.Registry
	internalReporter *reporter.InternalReporter

	mu      sync.Mutex
	runners map[string]*runner
	ctx     context.Context
	cancel  context.CancelFunc
}

// New creates a new engine.
func New(db *gorm.DB, platformReg *platform.Registry, reporterReg *reporter.Registry) *Engine {
	ir := reporter.NewInternalReporter(db)
	reporterReg.Register(ir)

	return &Engine{
		db:               db,
		agentClient:      agent.NewClient(),
		platformRegistry: platformReg,
		reporterRegistry: reporterReg,
		internalReporter: ir,
		runners:          make(map[string]*runner),
	}
}

// Start begins the engine and resumes all pipelines that were in RUNNING state.
func (e *Engine) Start() {
	e.ctx, e.cancel = context.WithCancel(context.Background())

	var pipelines []models.Pipeline
	e.db.Where("status = ?", int32(v1.PipelineStatus_PIPELINE_STATUS_RUNNING)).Find(&pipelines)

	for _, p := range pipelines {
		e.startRunner(p.ID)
	}

	slog.Info("engine: started", "resumed_pipelines", len(pipelines))
}

// Stop shuts down all running pipeline runners.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}

	for id, r := range e.runners {
		r.stop()
		delete(e.runners, id)
	}

	slog.Info("engine: stopped")
}

// StartPipeline starts a specific pipeline's runner.
func (e *Engine) StartPipeline(pipelineID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.runners[pipelineID]; exists {
		return
	}

	e.startRunner(pipelineID)
}

// StopPipeline stops a specific pipeline's runner.
func (e *Engine) StopPipeline(pipelineID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if r, exists := e.runners[pipelineID]; exists {
		r.stop()
		delete(e.runners, pipelineID)
	}
}

func (e *Engine) startRunner(pipelineID string) {
	r := newRunner(pipelineID, e.db, e.agentClient, e.platformRegistry, e.reporterRegistry, e.internalReporter)
	e.runners[pipelineID] = r
	r.start(e.ctx)
}
