package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nickheyer/discordiance/internal/agent"
	"github.com/nickheyer/discordiance/internal/config"
	"github.com/nickheyer/discordiance/internal/models"
	"github.com/nickheyer/discordiance/internal/platform"
	"github.com/nickheyer/discordiance/internal/reporter"
	"gorm.io/gorm"
)

// ProductStatus represents the runtime state of a single product pipeline.
type ProductStatus struct {
	ProductID   uint
	ProductName string
	Running     bool
	Healthy     bool
}

// Engine is the top-level orchestrator that manages one Pipeline per product.
type Engine struct {
	mu        sync.Mutex
	db        *gorm.DB
	pipelines map[uint]*Pipeline // keyed by product ID, only contains running pipelines
	ctx       context.Context
}

func New(db *gorm.DB) *Engine {
	slog.Info("engine created")
	return &Engine{
		db:        db,
		pipelines: make(map[uint]*Pipeline),
	}
}

// Setup seeds the database from config and stores the app context. Does NOT start any pipelines.
func (e *Engine) Setup(ctx context.Context, cfg *config.Config) error {
	slog.Info("engine setup starting")
	e.ctx = ctx
	if err := e.seed(ctx, cfg); err != nil {
		return err
	}
	slog.Info("engine setup complete")
	return nil
}

// seed writes config products to the database only if the DB has zero products (first boot).
func (e *Engine) seed(ctx context.Context, cfg *config.Config) error {
	var count int64
	if err := e.db.Model(&models.Product{}).Count(&count).Error; err != nil {
		return fmt.Errorf("engine: counting products: %w", err)
	}
	if count > 0 {
		slog.Info("database already seeded, skipping config import", "products", count)
		return nil
	}

	slog.Info("seeding database from config", "products", len(cfg.Products))
	for _, pc := range cfg.Products {
		product := models.Product{Name: pc.Name, Enabled: true}
		if err := e.db.Create(&product).Error; err != nil {
			return fmt.Errorf("engine: creating product %s: %w", pc.Name, err)
		}

		for _, ps := range pc.Platforms {
			platCfg := models.PlatformConfig{
				ProductID: product.ID,
				Type:      ps.Type,
				Enabled:   true,
				Settings:  models.JSONMap(ps.Settings),
			}
			if err := e.db.Create(&platCfg).Error; err != nil {
				return fmt.Errorf("engine: creating platform config for %s: %w", pc.Name, err)
			}
			slog.Info("seeded platform config", "product", pc.Name, "type", ps.Type)
		}

		agentCfg := models.AgentConfig{
			ProductID:    product.ID,
			BaseURL:      pc.Agent.BaseURL,
			APIKey:       pc.Agent.APIKey,
			OrgID:        pc.Agent.OrgID,
			Model:        pc.Agent.Model,
			SystemPrompt: pc.Agent.SystemPrompt,
			BatchSize:    pc.Agent.BatchSize,
			BatchTimeout: pc.Agent.BatchTimeout,
		}
		if agentCfg.BatchSize <= 0 {
			agentCfg.BatchSize = 10
		}
		if agentCfg.BatchTimeout <= 0 {
			agentCfg.BatchTimeout = 30
		}
		if err := e.db.Create(&agentCfg).Error; err != nil {
			return fmt.Errorf("engine: creating agent config for %s: %w", pc.Name, err)
		}
		slog.Info("seeded agent config", "product", pc.Name, "model", agentCfg.Model)

		for _, rs := range pc.Reporters {
			repCfg := models.ReporterConfig{
				ProductID: product.ID,
				Type:      rs.Type,
				Enabled:   true,
				Settings:  models.JSONMap(rs.Settings),
			}
			if err := e.db.Create(&repCfg).Error; err != nil {
				return fmt.Errorf("engine: creating reporter config for %s: %w", pc.Name, err)
			}
			slog.Info("seeded reporter config", "product", pc.Name, "type", rs.Type)
		}

		slog.Info("seeded product", "product", pc.Name, "id", product.ID)
	}
	return nil
}

// buildPipeline creates a Pipeline from a fully-loaded Product model.
func (e *Engine) buildPipeline(ctx context.Context, product models.Product) (*Pipeline, error) {
	if len(product.PlatformConfigs) == 0 {
		return nil, fmt.Errorf("no platform configs for product %s", product.Name)
	}

	platCfg := product.PlatformConfigs[0]
	slog.Info("creating platform", "product", product.Name, "type", platCfg.Type, "platform_config_id", platCfg.ID)

	plat, err := platform.Create(platCfg.Type)
	if err != nil {
		return nil, fmt.Errorf("creating platform %s: %w", platCfg.Type, err)
	}
	if err := plat.Connect(ctx, platCfg.Settings); err != nil {
		return nil, fmt.Errorf("connecting platform %s: %w", platCfg.Type, err)
	}
	slog.Info("platform connected", "product", product.Name, "type", platCfg.Type)

	if product.AgentConfig == nil {
		return nil, fmt.Errorf("no agent config for product %s", product.Name)
	}
	ag := agent.New(*product.AgentConfig)
	slog.Info("agent created", "product", product.Name, "model", product.AgentConfig.Model)

	var reporters []reporter.Reporter
	for _, rc := range product.ReporterConfigs {
		slog.Info("creating reporter", "product", product.Name, "type", rc.Type)
		r, err := reporter.Create(rc.Type)
		if err != nil {
			slog.Warn("skipping reporter", "type", rc.Type, "error", err)
			continue
		}
		if err := r.Init(ctx, rc.Settings); err != nil {
			slog.Warn("skipping reporter, init failed", "type", rc.Type, "error", err)
			continue
		}
		reporters = append(reporters, r)
		slog.Info("reporter initialized", "product", product.Name, "type", rc.Type)
	}

	batchSize := product.AgentConfig.BatchSize
	if batchSize <= 0 {
		batchSize = 10
	}
	batchTimeout := product.AgentConfig.BatchTimeout
	if batchTimeout <= 0 {
		batchTimeout = 30
	}

	slog.Info("pipeline build complete", "product", product.Name,
		"batch_size", batchSize, "batch_timeout_s", batchTimeout, "reporters", len(reporters))

	return NewPipeline(PipelineConfig{
		DB:               e.db,
		Product:          product,
		PlatformConfigID: platCfg.ID,
		Platform:         plat,
		Agent:            ag,
		Reporters:        reporters,
		BatchSize:        batchSize,
		BatchTimeout:     time.Duration(batchTimeout) * time.Second,
	}), nil
}

func (e *Engine) loadProduct(productID uint) (models.Product, error) {
	var product models.Product
	if err := e.db.
		Preload("PlatformConfigs", "enabled = ?", true).
		Preload("AgentConfig").
		Preload("ReporterConfigs", "enabled = ?", true).
		First(&product, productID).Error; err != nil {
		return product, fmt.Errorf("engine: loading product %d: %w", productID, err)
	}
	return product, nil
}

// StartProduct builds and starts the pipeline for a product.
func (e *Engine) StartProduct(ctx context.Context, productID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.pipelines[productID]; ok {
		return fmt.Errorf("engine: pipeline for product %d is already running", productID)
	}

	product, err := e.loadProduct(productID)
	if err != nil {
		return err
	}

	if !product.Enabled {
		return fmt.Errorf("engine: product %s is disabled", product.Name)
	}

	slog.Info("starting product pipeline", "product", product.Name, "product_id", productID)

	pipeline, err := e.buildPipeline(e.ctx, product)
	if err != nil {
		return fmt.Errorf("engine: building pipeline for %s: %w", product.Name, err)
	}

	if err := pipeline.Start(e.ctx); err != nil {
		return fmt.Errorf("engine: starting pipeline for %s: %w", product.Name, err)
	}

	e.pipelines[product.ID] = pipeline
	slog.Info("pipeline started", "product", product.Name, "product_id", product.ID)
	return nil
}

// StopProduct stops the pipeline for a product.
func (e *Engine) StopProduct(ctx context.Context, productID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	p, ok := e.pipelines[productID]
	if !ok {
		return fmt.Errorf("engine: no running pipeline for product %d", productID)
	}

	slog.Info("stopping pipeline", "product_id", productID)
	if err := p.Stop(ctx); err != nil {
		slog.Error("stopping pipeline", "product_id", productID, "error", err)
	}
	delete(e.pipelines, productID)
	slog.Info("pipeline stopped", "product_id", productID)
	return nil
}

// RestartProduct stops the existing pipeline for a product, reloads from DB, and starts a new one.
func (e *Engine) RestartProduct(ctx context.Context, productID uint) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	slog.Info("restarting product pipeline", "product_id", productID)

	if p, ok := e.pipelines[productID]; ok {
		slog.Info("stopping existing pipeline", "product_id", productID)
		if err := p.Stop(ctx); err != nil {
			slog.Error("stopping pipeline for restart", "product_id", productID, "error", err)
		}
		delete(e.pipelines, productID)
	}

	product, err := e.loadProduct(productID)
	if err != nil {
		return err
	}

	if !product.Enabled {
		slog.Info("product disabled, not restarting pipeline", "product", product.Name)
		return nil
	}

	pipeline, err := e.buildPipeline(e.ctx, product)
	if err != nil {
		return fmt.Errorf("engine: building pipeline for %s: %w", product.Name, err)
	}

	if err := pipeline.Start(e.ctx); err != nil {
		return fmt.Errorf("engine: starting pipeline for %s: %w", product.Name, err)
	}

	e.pipelines[product.ID] = pipeline
	slog.Info("pipeline restarted", "product", product.Name, "product_id", product.ID)
	return nil
}

// BackfillProduct triggers a backfill on a running pipeline.
func (e *Engine) BackfillProduct(ctx context.Context, productID uint) error {
	e.mu.Lock()
	p, ok := e.pipelines[productID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("engine: no running pipeline for product %d", productID)
	}

	slog.Info("triggering manual backfill", "product_id", productID)
	go p.RunBackfill(e.ctx)
	return nil
}

// Status returns the state of all products, indicating which are running.
func (e *Engine) Status() []ProductStatus {
	e.mu.Lock()
	defer e.mu.Unlock()

	var products []models.Product
	e.db.Where("enabled = ?", true).Find(&products)

	statuses := make([]ProductStatus, 0, len(products))
	for _, prod := range products {
		st := ProductStatus{
			ProductID:   prod.ID,
			ProductName: prod.Name,
		}
		if p, ok := e.pipelines[prod.ID]; ok {
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
			slog.Error("stopping pipeline", "product_id", id, "error", err)
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
