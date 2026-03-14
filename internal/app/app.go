package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/nickheyer/discordiance/internal/config"
	"github.com/nickheyer/discordiance/internal/db"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/platform"
	discordadapter "github.com/nickheyer/discordiance/internal/platform/discord"
	"github.com/nickheyer/discordiance/internal/reporter"
	"github.com/nickheyer/discordiance/internal/rpc"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

// App is the top-level dependency container. It owns the full lifecycle
// of all application components: database, engine, and HTTP server.
type App struct {
	cfg    *config.Config
	db     *gorm.DB
	engine *engine.Engine
	http   *http.Server
}

// New constructs every dependency and returns a ready-to-start App.
func New(configPath string) (*App, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	platformReg := buildPlatformRegistry()
	reporterReg := buildReporterRegistry()

	eng := engine.New(database, platformReg, reporterReg)

	rpcServer := rpc.NewServer(database, eng)
	httpServer := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      rpcServer.Handler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	return &App{
		cfg:    cfg,
		db:     database,
		engine: eng,
		http:   httpServer,
	}, nil
}

// Start begins the engine and HTTP server. It returns immediately;
// the HTTP server runs in a background goroutine.
func (a *App) Start() {
	a.engine.Start()

	go func() {
		slog.Info("server listening",
			"addr", a.http.Addr,
			"url", "http://localhost:"+a.cfg.Server.Port)
		if err := a.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()
}

// Stop gracefully shuts down the engine and HTTP server.
func (a *App) Stop(ctx context.Context) {
	a.engine.Stop()

	if err := a.http.Shutdown(ctx); err != nil {
		slog.Error("http server shutdown error", "error", err)
	}

	slog.Info("shutdown complete")
}

func buildPlatformRegistry() *platform.Registry {
	r := platform.NewRegistry()
	r.Register(v1.PlatformType_PLATFORM_TYPE_DISCORD, discordadapter.Factory)
	return r
}

func buildReporterRegistry() *reporter.Registry {
	r := reporter.NewRegistry()
	r.Register(reporter.NewWebhookReporter())
	return r
}
