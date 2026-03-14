package db

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/nickheyer/discordiance/internal/models"
)

func Open(dbPath string) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// SQLite performance pragmas.
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	slog.Info("db: opened", "path", dbPath)
	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// Core entities
		&models.Product{},
		&models.ProductContext{},
		&models.Agent{},
		&models.Platform{},
		&models.Pipeline{},
		&models.Insight{},
		&models.Reporter{},
		&models.Report{},
		&models.ReportEntry{},

		// Platform-specific config tables
		&models.DiscordPlatformConfig{},
		&models.RedditPlatformConfig{},
		&models.TwitterPlatformConfig{},
		&models.LinkedInPlatformConfig{},
		&models.GitHubPlatformConfig{},

		// Reporter-specific config tables
		&models.WebhookReporterConfig{},
		&models.EmailReporterConfig{},
		&models.DiscordReporterConfig{},
		&models.GitHubIssueReporterConfig{},

		// Pipeline join tables
		&models.PipelineAgent{},
		&models.PipelinePlatform{},
		&models.PipelineReporter{},

		// Reporter filter tables
		&models.ReporterFilterState{},
		&models.ReporterFilterPlatform{},
	)
}
