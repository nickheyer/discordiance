package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nickheyer/discordiance/internal/app"
)

func main() {
	configPath := flag.String("config", "", "path to configuration directory")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("starting discordiance", "pid", os.Getpid())

	application, err := app.New(*configPath)
	if err != nil {
		slog.Error("initialization failed", "error", err)
		os.Exit(1)
	}

	application.Start()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigChan
	slog.Info("received shutdown signal", "signal", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	application.Stop(ctx)
}
