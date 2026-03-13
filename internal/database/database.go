package database

import (
	"fmt"
	"log/slog"

	"github.com/nickheyer/discordiance/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(dsn string) (*gorm.DB, error) {
	slog.Info("database: opening", "dsn", dsn)

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		slog.Error("database: failed to open", "dsn", dsn, "error", err)
		return nil, fmt.Errorf("opening database: %w", err)
	}

	slog.Info("database: connected", "dsn", dsn)
	return db, nil
}

func Migrate(db *gorm.DB) error {
	slog.Info("database: running migrations")

	tables := []interface{}{
		&models.Product{},
		&models.PlatformConfig{},
		&models.AgentConfig{},
		&models.ReporterConfig{},
		&models.Message{},
		&models.Issue{},
		&models.Report{},
		&models.BackfillCursor{},
	}

	slog.Info("database: migrating tables", "count", len(tables))

	err := db.AutoMigrate(tables...)
	if err != nil {
		slog.Error("database: migration failed", "error", err)
		return fmt.Errorf("running migrations: %w", err)
	}

	slog.Info("database: migrations complete", "tables", len(tables))
	return nil
}
