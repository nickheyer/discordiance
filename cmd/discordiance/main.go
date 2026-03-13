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
	"github.com/nickheyer/discordiance/internal/database"
	"github.com/nickheyer/discordiance/internal/engine"
	"github.com/nickheyer/discordiance/internal/rpc"

	// Register platform implementations
	_ "github.com/nickheyer/discordiance/internal/platform/discord"

	// Register reporter implementations
	_ "github.com/nickheyer/discordiance/internal/reporter/github"
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
	slog.Info("opening database", "path", cfg.Database.Path)
	db, err := database.Open(cfg.Database.Path)
	if err != nil {
		slog.Error("opening database", "error", err)
		os.Exit(1)
	}
	slog.Info("database opened")

	slog.Info("running database migrations")
	if err := database.Migrate(db); err != nil {
		slog.Error("running migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations complete")

	// Create engine and set up pipelines
	slog.Info("creating engine")
	eng := engine.New(db)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slog.Info("setting up engine")
	if err := eng.Setup(ctx); err != nil {
		slog.Error("setting up engine", "error", err)
		os.Exit(1)
	}
	slog.Info("engine setup complete")

	// Start RPC server
	slog.Info("setting up RPC server")
	rpcServer := rpc.NewServer(db, eng)
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
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	slog.Info("stopping engine")
	eng.Stop(shutdownCtx)
	slog.Info("engine stopped")

	slog.Info("shutting down HTTP server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	} else {
		slog.Info("HTTP server shut down")
	}

	slog.Info("shutdown complete")
}
