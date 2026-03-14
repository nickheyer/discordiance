package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nickheyer/discordiance/internal/config"
	"github.com/nickheyer/discordiance/internal/db"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/platform"
	discordadapter "github.com/nickheyer/discordiance/internal/platform/discord"
	"github.com/nickheyer/discordiance/internal/reporter"
	"github.com/nickheyer/discordiance/internal/rpc"
	v1 "github.com/nickheyer/discordiance/pkg/proto/discordiance/v1"
)

func main() {
	configPath := flag.String("config", "", "path to configuration directory")
	flag.Parse()

	// Set up structured logging
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting discordiance", "pid", os.Getpid())

	// Load config
	slog.Info("loading configuration", "config_path", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("loading config", "error", err)
		os.Exit(1)
	}
	slog.Info("configuration loaded", "port", cfg.Server.Port, "db_path", cfg.Database.Path)

	// Open database
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("opening database", "error", err)
		os.Exit(1)
	}

	// Initialize registries
	platformReg := platform.NewRegistry()
	platformReg.Register(v1.PlatformType_PLATFORM_TYPE_DISCORD, discordadapter.Factory)

	reporterReg := reporter.NewRegistry()
	reporterReg.Register(reporter.NewWebhookReporter())

	// Start engine
	eng := engine.New(database, platformReg, reporterReg)
	eng.Start()

	// Start RPC server
	slog.Info("setting up RPC server")
	rpcServer := rpc.NewServer(database, eng)
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      rpcServer.Handler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr, "url", "http://localhost:"+cfg.Server.Port,
			"read_timeout", cfg.Server.ReadTimeout, "write_timeout", cfg.Server.WriteTimeout, "idle_timeout", cfg.Server.IdleTimeout)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
		slog.Info("server stopped listening")
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan

	slog.Info("received shutdown signal", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	slog.Info("stopping engine")
	eng.Stop()

	slog.Info("shutting down HTTP server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server shut down")
	}

	slog.Info("shutdown complete")
}
