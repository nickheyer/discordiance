package reporter

import (
	"context"

	"github.com/nickheyer/discordiance/internal/models"
)

// Reporter defines the interface for any insight destination (GitHub, Jira, etc.).
type Reporter interface {
	// Type returns the reporter identifier (e.g. "github").
	Type() string

	// Init initializes the reporter with the given config.
	Init(ctx context.Context, cfg models.Reporter) error

	// FileReport creates a new issue/ticket in the external system.
	FileReport(ctx context.Context, insight models.Insight) (*models.Report, error)

	// UpdateReport updates an existing issue/ticket.
	UpdateReport(ctx context.Context, externalID string, insight models.Insight) error

	// FindDuplicate checks if a similar insight already exists.
	FindDuplicate(ctx context.Context, insight models.Insight) (string, error)

	// Close gracefully shuts down the reporter.
	Close(ctx context.Context) error

	// Healthy reports whether the reporter is operational.
	Healthy(ctx context.Context) bool
}
